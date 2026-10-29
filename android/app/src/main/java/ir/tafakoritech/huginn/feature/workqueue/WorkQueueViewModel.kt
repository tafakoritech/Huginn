package ir.tafakoritech.huginn.feature.workqueue

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.getOrNull
import ir.tafakoritech.huginn.data.model.NextActionItem
import ir.tafakoritech.huginn.data.model.NextActionPriority
import ir.tafakoritech.huginn.data.model.NextActionSource
import ir.tafakoritech.huginn.data.model.NextActionType
import ir.tafakoritech.huginn.data.repository.WorkQueueRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

enum class WorkQueueTab { ALL, URGENT, TODAY, OVERDUE, REPORTS, CALL, MESSAGE, EMAIL }

data class WorkQueueUiState(
    val loading: Boolean = true,
    val items: List<NextActionItem> = emptyList(),
    val hasError: Boolean = false,
)

@HiltViewModel
class WorkQueueViewModel @Inject constructor(
    private val repository: WorkQueueRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(WorkQueueUiState())
    val state: StateFlow<WorkQueueUiState> = _state

    private val _tab = MutableStateFlow(WorkQueueTab.ALL)
    val tab: StateFlow<WorkQueueTab> = _tab

    init {
        refresh()
    }

    fun selectTab(tab: WorkQueueTab) {
        _tab.value = tab
    }

    fun refresh() {
        viewModelScope.launch {
            _state.value = _state.value.copy(loading = true)
            val result = repository.nextActions(limit = 200)
            _state.value = WorkQueueUiState(
                loading = false,
                items = result.getOrNull()?.items ?: emptyList(),
                hasError = result is AppResult.Failure,
            )
        }
    }

    fun snooze(action: NextActionItem, untilIso: String) {
        viewModelScope.launch {
            repository.snooze(action.id, untilIso)
            refresh()
        }
    }

    fun complete(action: NextActionItem) {
        viewModelScope.launch {
            repository.complete(action.id)
            refresh()
        }
    }
}

/** Pure filter for the work-queue tabs (unit-tested). */
internal fun filterForTab(items: List<NextActionItem>, tab: WorkQueueTab): List<NextActionItem> = when (tab) {
    WorkQueueTab.ALL -> items
    WorkQueueTab.URGENT -> items.filter { it.priority == NextActionPriority.URGENT }
    WorkQueueTab.TODAY -> items.filter { it.dueAt != null && ir.tafakoritech.huginn.core.util.DateTimeUtil.isToday(it.dueAt) }
    WorkQueueTab.OVERDUE -> items.filter { it.overdue }
    WorkQueueTab.REPORTS -> items.filter { it.sourceType == NextActionSource.PENDING_REPORT }
    WorkQueueTab.CALL -> items.filter { it.actionType == NextActionType.CALL }
    WorkQueueTab.MESSAGE -> items.filter { it.actionType == NextActionType.MESSAGE }
    WorkQueueTab.EMAIL -> items.filter { it.actionType == NextActionType.EMAIL }
}
