package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.data.model.PendingReport
import ir.tafakoritech.huginn.data.model.PendingReportsResponse
import ir.tafakoritech.huginn.feature.app.PendingReportGate
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The mandatory-report critical scenario:
 *
 *  call started -> app background -> app resumes -> pending report opens ->
 *  back pressed (report stays) -> process killed -> app reopened -> backend
 *  still reports pending -> report opens again -> user submits -> normal
 *  navigation becomes available.
 */
class PendingReportGateTest {

    private val pending = PendingReport(
        interactionId = "int-1",
        leadId = "lead-1",
        companyName = "عروسک‌سازی پارس",
        channel = "CALL",
    )

    @Test
    fun `critical mandatory report scenario survives process death`() = runTest {
        // Backend says a report is pending.
        var backendSaysPending = true
        val gate = PendingReportGate {
            if (backendSaysPending) {
                AppResult.Success(PendingReportsResponse(
                        requiresReport = true,
                        interactionId = "int-1",
                        leadId = "lead-1",
                        channel = "CALL",
                    ))
            } else {
                AppResult.Success(PendingReportsResponse())
            }
        }

        // 1) Resume after the call -> report opens.
        val report = gate.check()
        assertNotNull(report)
        assertEquals("int-1", report!!.interactionId)
        assertTrue(gate.hasPendingReport)

        // 2) The report cannot be dismissed: it is still present even if the
        //    UI layer ignores it; re-checking keeps it open.
        gate.clear() // simulates an attempted dismissal that is NOT backed by the backend
        val afterAttemptedDismiss = gate.check()
        assertNotNull(afterAttemptedDismiss)
        assertTrue(gate.hasPendingReport)

        // 3) Process death: a brand-new gate instance (new process) still sees
        //    the backend as source of truth.
        val restartedGate = PendingReportGate {
            if (backendSaysPending) {
                AppResult.Success(PendingReportsResponse(
                        requiresReport = true,
                        interactionId = "int-1",
                        leadId = "lead-1",
                        channel = "CALL",
                    ))
            } else {
                AppResult.Success(PendingReportsResponse())
            }
        }
        val reportAfterRestart = restartedGate.check()
        assertNotNull(reportAfterRestart)
        assertEquals("int-1", reportAfterRestart!!.interactionId)

        // 4) User submits the report; the backend confirms and stops reporting pending.
        backendSaysPending = false
        restartedGate.clear() // submission acknowledged
        val afterSubmit = restartedGate.check()
        assertNull(afterSubmit)
        assertFalse(restartedGate.hasPendingReport)
        // Normal navigation is available again.
    }

    @Test
    fun `offline check does not fake completion`() = runTest {
        val gate = PendingReportGate { AppResult.Failure(ir.tafakoritech.huginn.core.util.AppError.Network) }
        assertNull(gate.check())
        assertFalse(gate.hasPendingReport)
    }

    @Test
    fun `checking flag flips during check`() = runTest {
        val gate = PendingReportGate { AppResult.Success(PendingReportsResponse(requiresReport = false)) }
        gate.check()
        assertFalse(gate.checking.value)
    }

    @Test
    fun `forceOpen routes the user to a specific interaction report`() {
        val gate = PendingReportGate { AppResult.Success(PendingReportsResponse()) }
        gate.forceOpen("int-42")
        assertTrue(gate.hasPendingReport)
        assertEquals("int-42", gate.pendingReport.value?.interactionId)
    }
}
