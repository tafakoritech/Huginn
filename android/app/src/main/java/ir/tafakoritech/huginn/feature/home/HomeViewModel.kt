package ir.tafakoritech.huginn.feature.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import ir.tafakoritech.huginn.core.auth.SessionManager
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.getOrNull
import ir.tafakoritech.huginn.data.model.MeStats
import ir.tafakoritech.huginn.data.model.NextActionItem
import ir.tafakoritech.huginn.data.repository.WorkQueueRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

data class HomeUiState(
    val loading: Boolean = true,
    val offline: Boolean = false,
    val userName: String = "",
    val stats: MeStats = MeStats(),
    val nextActions: List<NextActionItem> = emptyList(),
    val totalActionable: Int = 0,
    val urgentCount: Int = 0,
    val overdueCount: Int = 0,
    val hasError: Boolean = false,
)

@HiltViewModel
class HomeViewModel @Inject constructor(
    private val workQueueRepository: WorkQueueRepository,
    sessionManager: SessionManager,
) : ViewModel() {

    private val _state = MutableStateFlow(HomeUiState())
    val state: StateFlow<HomeUiState> = _state

    private val userName = sessionManager.currentUser?.name ?: ""

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _state.value = _state.value.copy(loading = _state.value.nextActions.isEmpty())
            val statsResult = workQueueRepository.me()
            val queueResult = workQueueRepository.nextActions(limit = 3)
            val stats = statsResult.getOrNull() ?: MeStats()
            val offline = statsResult is AppResult.Failure && queueResult is AppResult.Failure
            val queue = queueResult.getOrNull()
            _state.value = HomeUiState(
                loading = false,
                offline = offline,
                userName = userName,
                stats = stats,
                nextActions = queue?.items ?: emptyList(),
                totalActionable = queue?.totalActionable ?: 0,
                urgentCount = queue?.urgentCount ?: 0,
                overdueCount = queue?.overdueCount ?: 0,
                hasError = queueResult is AppResult.Failure && offline,
            )
        }
    }

    /** Snoozes a follow-up-backed action; updates the real source record. */
    fun snooze(action: NextActionItem, untilIso: String) {
        viewModelScope.launch {
            workQueueRepository.snooze(action.id, untilIso)
            refresh()
        }
    }

    /** Completes a follow-up-backed action; maps to a completed follow-up. */
    fun complete(action: NextActionItem) {
        viewModelScope.launch {
            workQueueRepository.complete(action.id)
            refresh()
        }
    }
}
