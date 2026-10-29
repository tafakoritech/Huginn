package ir.tafakoritech.huginn.feature.leads

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.getOrNull
import ir.tafakoritech.huginn.data.model.Lead
import ir.tafakoritech.huginn.data.repository.LeadFilters
import ir.tafakoritech.huginn.data.repository.LeadRepository
import kotlinx.coroutines.FlowPreview
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.debounce
import kotlinx.coroutines.flow.filterNotNull
import kotlinx.coroutines.flow.flatMapLatest
import kotlinx.coroutines.launch
import javax.inject.Inject

data class LeadsFiltersUi(
    val scope: String? = null,
    val status: String? = null,
    val minScore: Int? = null,
    val sort: String? = null,
    val order: String? = null,
)

data class LeadsUiState(
    val loading: Boolean = true,
    val refreshing: Boolean = false,
    val loadingMore: Boolean = false,
    val offline: Boolean = false,
    val leads: List<Lead> = emptyList(),
    val total: Int = 0,
    val page: Int = 1,
    val endReached: Boolean = false,
    val query: String = "",
    val filters: LeadsFiltersUi = LeadsFiltersUi(),
    val error: Boolean = false,
)

@OptIn(FlowPreview::class)
@HiltViewModel
class LeadsViewModel @Inject constructor(
    private val leadRepository: LeadRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(LeadsUiState())
    val state: StateFlow<LeadsUiState> = _state

    private val pageSize = 20
    private var loadJob: Job? = null

    private val queryFlow = MutableStateFlow<String?>(null)

    init {
        viewModelScope.launch {
            queryFlow
                .filterNotNull()
                .debounce(400)
                .collect { query ->
                    _state.value = _state.value.copy(query = query)
                    loadFirstPage()
                }
        }
        loadFirstPage()
    }

    fun onQueryChange(query: String) {
        queryFlow.value = query
    }

    fun applyFilters(filters: LeadsFiltersUi) {
        _state.value = _state.value.copy(filters = filters)
        loadFirstPage()
    }

    fun resetFilters() {
        _state.value = _state.value.copy(filters = LeadsFiltersUi())
        loadFirstPage()
    }

    fun refresh() {
        loadFirstPage()
    }

    fun loadMore() {
        val s = _state.value
        if (s.loading || s.loadingMore || s.endReached || s.offline) return
        if (s.leads.size >= s.total) {
            _state.value = s.copy(endReached = true)
            return
        }
        loadPage(s.page + 1, append = true)
    }

    private fun loadFirstPage() = loadPage(1, append = false)

    private fun loadPage(page: Int, append: Boolean) {
        loadJob?.cancel()
        loadJob = viewModelScope.launch {
            val s = _state.value
            _state.value = if (append) s.copy(loadingMore = true) else s.copy(loading = true, refreshing = false, error = false)
            val filters = LeadFilters(
                search = s.query.takeIf { it.isNotBlank() },
                scope = s.filters.scope,
                status = s.filters.status,
                minScore = s.filters.minScore,
                sort = s.filters.sort,
                order = s.filters.order,
            )
            when (val result = leadRepository.listLeads(filters, page, pageSize)) {
                is AppResult.Success -> {
                    val data = result.data
                    _state.value = _state.value.copy(
                        loading = false,
                        loadingMore = false,
                        offline = false,
                        error = false,
                        leads = if (append) _state.value.leads + data.leads else data.leads,
                        total = data.total,
                        page = data.page,
                        endReached = data.leads.isEmpty() || (_state.value.leads.size + data.leads.size) >= data.total,
                    )
                }
                is AppResult.Failure -> {
                    _state.value = _state.value.copy(
                        loading = false,
                        loadingMore = false,
                        offline = true,
                        error = _state.value.leads.isEmpty(),
                    )
                }
            }
        }
    }
}
