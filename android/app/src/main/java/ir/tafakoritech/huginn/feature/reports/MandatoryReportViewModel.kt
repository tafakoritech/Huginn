package ir.tafakoritech.huginn.feature.reports

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import ir.tafakoritech.huginn.data.model.PendingReport
import ir.tafakoritech.huginn.data.model.ReportDraft
import ir.tafakoritech.huginn.data.model.ReportOutcome
import ir.tafakoritech.huginn.data.model.ReportSubmitResponse
import ir.tafakoritech.huginn.data.model.ReportValidationError
import ir.tafakoritech.huginn.data.model.ReportValidator
import ir.tafakoritech.huginn.data.repository.ReportRepository
import ir.tafakoritech.huginn.data.repository.ReportSubmissionResult
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

data class MandatoryReportUiState(
    val companyName: String = "",
    val contactName: String = "",
    val channel: String = "",
    val draft: ReportDraft = ReportDraft(),
    val validationErrors: Set<ReportValidationError> = emptySet(),
    val submitting: Boolean = false,
    val result: ReportSubmitResponse? = null,
    val queuedOffline: Boolean = false,
    val errorKey: String? = null,
) {
    val outcome: String? get() = draft.outcome
}

/**
 * Mandatory contact report. This screen is blocking: it cannot be dismissed
 * until the report is submitted (or the backend voids it). The interaction id
 * comes from the backend's pending-reports endpoint (source of truth), which
 * also makes the workflow survive process death.
 */
@HiltViewModel
class MandatoryReportViewModel @Inject constructor(
    private val reportRepository: ReportRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(MandatoryReportUiState())
    val state: StateFlow<MandatoryReportUiState> = _state

    private var interactionId: String? = null

    /**
     * Bind the form to [report]. Transient submission state (a previous success
     * result or an offline queue) is always cleared so the overlay never shows a
     * stale result/queued card when it reopens — the success screen is a
     * one-time event, not persisted UI state.
     *
     * The draft is preserved only while the same report is still being filled
     * out (recomposition re-entry with no completed submission).
     */
    fun init(report: PendingReport) {
        val sameReport = interactionId == report.interactionId
        val hasStaleCompletion = _state.value.result != null || _state.value.queuedOffline
        if (sameReport && !hasStaleCompletion) return
        interactionId = report.interactionId
        _state.value = MandatoryReportUiState(
            companyName = report.companyName ?: "",
            contactName = report.contactName ?: "",
            channel = report.channel,
            // Persistent facts promoted from earlier reports pre-fill the form so
            // stable information (decision maker, correct person) is not re-asked.
            draft = ReportDraft(
                correctPersonReached = report.correctPersonReached,
                decisionMakerIdentified = report.decisionMakerIdentified,
                decisionMakerName = report.decisionMakerName,
                decisionMakerRole = report.decisionMakerRole,
            ),
        )
    }

    /** Consume the one-time submission result when the user continues past it. */
    fun consumeResult() {
        _state.value = _state.value.copy(
            result = null,
            queuedOffline = false,
            submitting = false,
            errorKey = null,
            validationErrors = emptySet(),
        )
    }

    fun onOutcomeSelected(outcome: String) {
        // Keep a fresh draft for the new outcome, but carry forward the
        // persistent facts so the salesperson is never asked to re-enter stable
        // information when they switch outcomes.
        val current = _state.value.draft
        _state.value = _state.value.copy(
            draft = ReportDraft(
                outcome = outcome,
                correctPersonReached = current.correctPersonReached,
                decisionMakerIdentified = current.decisionMakerIdentified,
                decisionMakerName = current.decisionMakerName,
                decisionMakerRole = current.decisionMakerRole,
            ),
            validationErrors = emptySet(),
        )
    }

    fun updateDraft(transform: (ReportDraft) -> Unit) {
        val draft = _state.value.draft
        transform(draft)
        _state.value = _state.value.copy(draft = draft, validationErrors = emptySet())
    }

    /** Replace the draft wholesale (used by the stateless form / tests). */
    fun setDraft(draft: ReportDraft) {
        _state.value = _state.value.copy(draft = draft, validationErrors = emptySet())
    }

    fun submit() {
        if (_state.value.submitting) return
        val interactionId = interactionId ?: return
        val draft = _state.value.draft
        val errors = ReportValidator.validate(draft)
        if (errors.isNotEmpty()) {
            _state.value = _state.value.copy(validationErrors = errors.toSet())
            return
        }
        viewModelScope.launch {
            _state.value = _state.value.copy(submitting = true, errorKey = null)
            when (val result = reportRepository.submitReport(interactionId, draft.toRequest())) {
                is ReportSubmissionResult.Submitted -> {
                    _state.value = _state.value.copy(submitting = false, result = result.response)
                }
                is ReportSubmissionResult.Queued -> {
                    _state.value = _state.value.copy(submitting = false, queuedOffline = true)
                }
                is ReportSubmissionResult.Failed -> {
                    _state.value = _state.value.copy(submitting = false, errorKey = result.error::class.simpleName)
                }
            }
        }
    }

    /** Whether any fields are required for the current outcome. */
    fun requiresSummaryField(outcome: String?): Boolean = outcome in setOf(
        ReportOutcome.NO_ANSWER,
        ReportOutcome.BUSY,
        ReportOutcome.WRONG_NUMBER,
        ReportOutcome.NOT_RELEVANT,
        ReportOutcome.DISQUALIFIED,
        ReportOutcome.ANSWERED,
    )

    fun requiresFollowUpTime(outcome: String?): Boolean = outcome in setOf(
        ReportOutcome.NO_ANSWER,
        ReportOutcome.BUSY,
        ReportOutcome.WRONG_NUMBER,
        ReportOutcome.CALL_BACK_REQUESTED,
        ReportOutcome.ANSWERED,
        ReportOutcome.INTERESTED,
        ReportOutcome.HIGHLY_INTERESTED,
    )
}
