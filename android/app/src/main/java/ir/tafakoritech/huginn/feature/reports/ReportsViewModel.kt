package ir.tafakoritech.huginn.feature.reports

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.data.model.SubmittedReport
import ir.tafakoritech.huginn.data.repository.ReportRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * The global Reports tab (گزارش‌ها).
 *
 * This screen shows SUBMITTED contact reports from GET /me/reports — never
 * pending-report placeholders (در انتظار گزارش is a different concept, served
 * by GET /me/pending-reports and gated by the app). An empty backend response
 * is the ONLY condition that renders the "no reports" empty state; a network /
 * backend / parsing failure surfaces an explicit error state instead of being
 * masked as an empty list.
 */
sealed interface ReportsUiState {
    data object Loading : ReportsUiState
    data class Loaded(val reports: List<SubmittedReport>) : ReportsUiState
    data object Empty : ReportsUiState
    data class Error(val message: String? = null) : ReportsUiState
}

@HiltViewModel
class ReportsViewModel @Inject constructor(
    private val reportRepository: ReportRepository,
) : ViewModel() {

    private val _state = MutableStateFlow<ReportsUiState>(ReportsUiState.Loading)
    val state: StateFlow<ReportsUiState> = _state

    init {
        refresh()
    }

    fun refresh() {
        _state.value = ReportsUiState.Loading
        viewModelScope.launch {
            when (val result = reportRepository.myReports()) {
                is AppResult.Success -> {
                    val items = result.data.items
                    _state.value = if (items.isEmpty()) {
                        ReportsUiState.Empty
                    } else {
                        ReportsUiState.Loaded(items)
                    }
                }
                is AppResult.Failure -> _state.value = ReportsUiState.Error(result.error::class.simpleName)
            }
        }
    }
}
