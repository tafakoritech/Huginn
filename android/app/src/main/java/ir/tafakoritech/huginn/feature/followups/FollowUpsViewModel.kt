package ir.tafakoritech.huginn.feature.followups

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.data.model.FollowUp
import ir.tafakoritech.huginn.data.model.FollowUpCreateRequest
import ir.tafakoritech.huginn.data.model.FollowUpUpdateRequest
import ir.tafakoritech.huginn.data.repository.FollowUpRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

/** Global follow-up views; scope must match the backend list scopes exactly. */
enum class FollowUpTab(val scope: String) {
    TODAY("today"),
    OVERDUE("overdue"),
    UPCOMING("upcoming"),
    COMPLETED("completed"),
}

sealed interface FollowUpsUiState {
    val activeTab: FollowUpTab

    data class Loading(override val activeTab: FollowUpTab = FollowUpTab.TODAY) : FollowUpsUiState
    data class Loaded(
        override val activeTab: FollowUpTab,
        val items: List<FollowUp>,
        val total: Int = 0,
    ) : FollowUpsUiState
    data class Empty(override val activeTab: FollowUpTab = FollowUpTab.TODAY) : FollowUpsUiState
    data class Error(
        override val activeTab: FollowUpTab = FollowUpTab.TODAY,
        val message: String? = null,
    ) : FollowUpsUiState
}

@HiltViewModel
class FollowUpsViewModel @Inject constructor(
    private val followUpRepository: FollowUpRepository,
) : ViewModel() {

    private val _state = MutableStateFlow<FollowUpsUiState>(FollowUpsUiState.Loading())
    val state: StateFlow<FollowUpsUiState> = _state

    private var currentTab: FollowUpTab = FollowUpTab.TODAY

    init {
        load(currentTab)
    }

    fun selectTab(tab: FollowUpTab) {
        if (tab == currentTab && _state.value !is FollowUpsUiState.Error) return
        currentTab = tab
        _state.value = FollowUpsUiState.Loading(tab)
        load(tab)
    }

    fun refresh() {
        _state.value = FollowUpsUiState.Loading(currentTab)
        load(currentTab)
    }

    private fun load(tab: FollowUpTab) {
        viewModelScope.launch {
            when (val result = followUpRepository.myFollowUps(tab.scope)) {
                is AppResult.Success -> {
                    val items = result.data.items
                    _state.value = if (items.isEmpty()) {
                        FollowUpsUiState.Empty(tab)
                    } else {
                        FollowUpsUiState.Loaded(activeTab = tab, items = items, total = result.data.total)
                    }
                }
                is AppResult.Failure -> _state.value = FollowUpsUiState.Error(activeTab = tab, message = result.error::class.simpleName)
            }
        }
    }

    fun complete(id: String) {
        viewModelScope.launch {
            followUpRepository.complete(id)
            load(currentTab)
        }
    }

    fun reopen(id: String) {
        viewModelScope.launch {
            followUpRepository.reopen(id)
            load(currentTab)
        }
    }

    fun reschedule(id: String, dueAtIso: String) {
        viewModelScope.launch {
            followUpRepository.reschedule(id, dueAtIso)
            load(currentTab)
        }
    }

    fun update(id: String, request: FollowUpUpdateRequest) {
        viewModelScope.launch {
            followUpRepository.update(id, request)
            load(currentTab)
        }
    }

    fun cancel(id: String) {
        viewModelScope.launch {
            followUpRepository.cancel(id)
            load(currentTab)
        }
    }

    fun create(request: FollowUpCreateRequest) {
        viewModelScope.launch {
            followUpRepository.create(request)
            load(currentTab)
        }
    }
}
