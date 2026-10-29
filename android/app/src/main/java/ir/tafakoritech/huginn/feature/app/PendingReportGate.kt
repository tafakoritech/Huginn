package ir.tafakoritech.huginn.feature.app

import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.getOrNull
import ir.tafakoritech.huginn.data.model.PendingReport
import ir.tafakoritech.huginn.data.model.PendingReportsResponse
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow

/**
 * Mandatory-report gate state machine.
 *
 * The backend remains the source of truth: the gate re-queries pending reports
 * on every app resume. A report is only released when the backend stops
 * reporting it as pending — this survives process death. After a successful
 * submission the app clears the gate (the backend has already released the
 * report); any other outstanding report reopens on the next resume check.
 */
class PendingReportGate(
    private val pendingReportSource: suspend () -> AppResult<PendingReportsResponse>,
) {
    private val _pendingReport = MutableStateFlow<PendingReport?>(null)
    val pendingReport: StateFlow<PendingReport?> = _pendingReport

    private val _checking = MutableStateFlow(false)
    val checking: StateFlow<Boolean> = _checking

    suspend fun check(): PendingReport? {
        _checking.value = true
        try {
            val response = pendingReportSource().getOrNull()
            val report = response?.takeIf { it.requiresReport }?.reports?.firstOrNull()
            _pendingReport.value = report
            return report
        } finally {
            _checking.value = false
        }
    }

    /** Force-open a report for an interaction id (e.g. deep link / Reports tab). */
    fun forceOpen(interactionId: String) {
        _pendingReport.value = PendingReport(interactionId = interactionId, leadId = "")
    }

    /** Called after a successful submission; the next check confirms release. */
    fun clear() {
        _pendingReport.value = null
    }

    val hasPendingReport: Boolean
        get() = _pendingReport.value != null
}
