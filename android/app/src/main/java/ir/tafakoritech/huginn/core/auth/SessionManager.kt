package ir.tafakoritech.huginn.core.auth

import ir.tafakoritech.huginn.core.api.ApiCaller
import ir.tafakoritech.huginn.core.api.RefreshApi
import ir.tafakoritech.huginn.core.network.ApiErrorCodes
import ir.tafakoritech.huginn.core.network.ApiException
import ir.tafakoritech.huginn.core.util.SafeLog
import ir.tafakoritech.huginn.data.model.User
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.serialization.json.Json
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Owns the JWT session lifecycle:
 *  - persistent restore from secure storage
 *  - authenticated / anonymous state for the navigation host
 *  - single-flight refresh (one refresh at a time, shared across callers)
 *
 * When a refresh fails with UNAUTHORIZED the session is cleared and the app
 * returns to login. Never creates infinite retry loops.
 */
@Singleton
class SessionManager @Inject constructor(
    private val storage: TokenStore,
    private val refreshApi: RefreshApi,
    private val json: Json,
) {
    private val _state = MutableStateFlow<SessionState>(SessionState.Unknown)
    val state: StateFlow<SessionState> = _state

    private val refreshMutex = Mutex()
    private var inFlight: kotlinx.coroutines.Deferred<Boolean>? = null

    val isAuthenticated: Boolean
        get() = _state.value is SessionState.Authenticated

    val currentUser: User?
        get() = (_state.value as? SessionState.Authenticated)?.user

    /** Current access token for the auth header. Returns null when logged out. */
    fun accessToken(): String? = storage.accessToken

    /** Current refresh token (used for logout revocation). Null when logged out. */
    fun refreshToken(): String? = storage.refreshToken

    /** Restore a previously persisted session without any network call. */
    fun restoreFromStorage() {
        val cachedUser = storage.userJson?.let {
            runCatching { json.decodeFromString(User.serializer(), it) }.getOrNull()
        }
        _state.value = if (storage.hasTokens && cachedUser != null) {
            SessionState.Authenticated(cachedUser)
        } else {
            SessionState.Anonymous
        }
    }

    fun onAuthenticated(user: User) {
        runCatching { storage.userJson = json.encodeToString(User.serializer(), user) }
        _state.value = SessionState.Authenticated(user)
    }

    fun onTokensRefreshed(accessToken: String, refreshToken: String) {
        storage.saveTokens(accessToken, refreshToken)
    }

    /** Single-flight refresh. Returns true on success, false otherwise. */
    suspend fun refreshTokens(): Boolean {
        inFlight?.let { return it.await() }
        return refreshMutex.withLock {
            coroutineScope {
                val existing = inFlight
                if (existing != null) {
                    existing.await()
                } else {
                    val deferred = async { doRefresh() }
                    inFlight = deferred
                    try {
                        deferred.await()
                    } finally {
                        inFlight = null
                    }
                }
            }
        }
    }

    private suspend fun doRefresh(): Boolean {
        val refreshToken = storage.refreshToken
        if (refreshToken.isNullOrBlank()) {
            clearSession()
            return false
        }
        return try {
            val tokens = ApiCaller.call(json) { refreshApi.refresh(refreshToken) }
            if (tokens.accessToken.isNotBlank() && tokens.refreshToken.isNotBlank()) {
                storage.saveTokens(tokens.accessToken, tokens.refreshToken)
                true
            } else {
                clearSession()
                false
            }
        } catch (e: ApiException) {
            SafeLog.d("SessionManager: refresh failed code=${e.code} status=${e.httpStatus}")
            if (e.code == ApiErrorCodes.UNAUTHORIZED || e.httpStatus == 401) {
                clearSession()
            }
            false
        } catch (e: Throwable) {
            SafeLog.d("SessionManager: refresh network failure")
            false
        }
    }

    fun clearSession() {
        storage.clear()
        _state.value = SessionState.Anonymous
    }

    /** Full logout: best-effort server revocation then local session clear. */
    suspend fun logout() {
        storage.clear()
        _state.value = SessionState.Anonymous
    }
}
