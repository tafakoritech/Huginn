package ir.tafakoritech.huginn.feature.app

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import ir.tafakoritech.huginn.core.auth.SessionState
import ir.tafakoritech.huginn.core.datastore.SettingsRepository
import ir.tafakoritech.huginn.core.locale.LocaleManager
import ir.tafakoritech.huginn.core.util.AppError
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.getOrNull
import ir.tafakoritech.huginn.data.model.PendingReport
import ir.tafakoritech.huginn.data.repository.AuthRepository
import ir.tafakoritech.huginn.data.repository.WorkQueueRepository
import ir.tafakoritech.huginn.work.WorkScheduler
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * Root application gate: owns the session state, the mandatory-report gate and
 * lifecycle resume checks.
 *
 * Mandatory reports are re-checked from the backend whenever the app resumes —
 * after a call or messaging round-trip, and after process death.
 */
@HiltViewModel
class AppGateViewModel @Inject constructor(
    private val authRepository: AuthRepository,
    workQueueRepository: WorkQueueRepository,
    private val settingsRepository: SettingsRepository,
    private val localeManager: LocaleManager,
    private val workScheduler: WorkScheduler,
) : ViewModel() {

    private val _sessionState = MutableStateFlow<SessionState>(SessionState.Unknown)
    val sessionState: StateFlow<SessionState> = _sessionState

    val gate = PendingReportGate(workQueueRepository::pendingReports)
    val pendingReport: StateFlow<PendingReport?> = gate.pendingReport
    val checkingPendingReport: StateFlow<Boolean> = gate.checking

    private val _loginBusy = MutableStateFlow(false)
    val loginBusy: StateFlow<Boolean> = _loginBusy

    private val _loginError = MutableStateFlow<AppError?>(null)
    val loginError: StateFlow<AppError?> = _loginError

    private val _deepLinkRoute = MutableStateFlow<String?>(null)
    val deepLinkRoute: StateFlow<String?> = _deepLinkRoute

    init {
        // Session restore and locale application run independently so that a
        // failure in either can never leave the app stuck on the splash
        // (SessionState.Unknown).
        viewModelScope.launch {
            authRepository.sessionState.collect { _sessionState.value = it }
        }
        viewModelScope.launch {
            try {
                authRepository.restoreSession()
            } catch (t: Throwable) {
                ir.tafakoritech.huginn.core.util.SafeLog.w("Session restore failed", t)
            } finally {
                // Guarantee we leave the splash regardless of the outcome.
                if (_sessionState.value is SessionState.Unknown) {
                    _sessionState.value = SessionState.Anonymous
                }
            }
        }
        viewModelScope.launch {
            runCatching {
                // Saved user preference -> device language -> English fallback.
                val lang = settingsRepository.language.first()
                    ?: ir.tafakoritech.huginn.core.locale.LanguageResolver.fromLocale()
                localeManager.apply(lang)
            }.onFailure {
                ir.tafakoritech.huginn.core.util.SafeLog.w("Locale apply failed", it)
            }
        }
    }

    /** Watchdog: never let the splash (Unknown) block the app indefinitely. */
    fun ensureSessionResolved() {
        if (_sessionState.value is SessionState.Unknown) {
            _sessionState.value = SessionState.Anonymous
        }
    }

    /** Must be called on every ON_RESUME while authenticated. */
    fun onAppResumed() {
        if (_sessionState.value is SessionState.Authenticated) {
            checkPendingReports()
        }
    }

    fun checkPendingReports() {
        viewModelScope.launch {
            gate.check()
        }
    }

    fun login(username: String, password: String) {
        if (_loginBusy.value) return
        viewModelScope.launch {
            _loginBusy.value = true
            _loginError.value = null
            when (val result = authRepository.login(username, password)) {
                is AppResult.Success -> {
                    workScheduler.scheduleFollowUpReminders()
                    checkPendingReports()
                }
                is AppResult.Failure -> {
                    _loginError.value = result.error
                }
            }
            _loginBusy.value = false
        }
    }

    fun logout() {
        viewModelScope.launch {
            authRepository.logout()
            gate.clear()
        }
    }

    /**
     * Called after the mandatory report is submitted (or voided upstream).
     *
     * Only the gate is cleared here. We deliberately do NOT re-query pending
     * reports in the same call: the just-submitted report is already released
     * by the backend, and an immediate re-check (which may race an in-flight
     * snapshot taken before the submit) can re-open the overlay right after the
     * user continued, showing the stale result screen. Any other outstanding
     * report is surfaced by the next ON_RESUME re-check, which is the gate's
     * source-of-truth mechanism.
     */
    fun onMandatoryReportDone() {
        gate.clear()
    }

    /** Opens the blocking report for a specific interaction (e.g. from Reports tab). */
    fun showReport(interactionId: String) {
        gate.forceOpen(interactionId)
        checkPendingReports()
    }

    /** Validated deep link from a huginn:// intent. */
    fun handleDeepLink(uri: android.net.Uri) {
        val route = ir.tafakoritech.huginn.core.external.DeepLinkValidator.parse(uri.toString()) ?: return
        if (route.startsWith("mandatory-report")) {
            // Backend is the source of truth for pending reports.
            checkPendingReports()
        } else {
            _deepLinkRoute.value = route
        }
    }

    fun consumeDeepLink() {
        _deepLinkRoute.value = null
    }
}
