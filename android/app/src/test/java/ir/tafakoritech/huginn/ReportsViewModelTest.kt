package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.database.PendingReportDao
import ir.tafakoritech.huginn.data.mock.MockHuginnApi
import ir.tafakoritech.huginn.data.model.ReportDraft
import ir.tafakoritech.huginn.data.model.ReportOutcome
import ir.tafakoritech.huginn.data.model.SubmittedReport
import ir.tafakoritech.huginn.data.repository.ReportRepository
import ir.tafakoritech.huginn.feature.reports.ReportsUiState
import ir.tafakoritech.huginn.feature.reports.ReportsViewModel
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

/**
 * The Reports tab must render SUBMITTED contact reports (GET /me/reports), with
 * the empty state shown ONLY for a successful empty response and a real error
 * state for failures. A freshly submitted report must appear immediately after
 * the next refresh — no app restart required.
 */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class ReportsViewModelTest {

    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private fun reportRepository(api: MockHuginnApi): ReportRepository = ReportRepository(
        api = api,
        pendingReportDao = FakePendingReportDao(),
        json = json,
        context = RuntimeEnvironment.getApplication(),
    )

    private fun runVmTest(block: suspend TestScope.() -> Unit) = runTest {
        Dispatchers.setMain(StandardTestDispatcher(testScheduler))
        try {
            block()
        } finally {
            Dispatchers.resetMain()
        }
    }

    private fun seedReports(api: MockHuginnApi, reports: List<SubmittedReport>) {
        reports.forEach { api.seedSubmittedReport(it) }
    }

    @Test
    fun `one submitted report renders one report card`() = runVmTest {
        val api = MockHuginnApi()
        seedReports(
            api,
            listOf(
                SubmittedReport(
                    id = "rep-1",
                    interactionId = "int-1",
                    leadId = "lead-1",
                    companyName = "شرکت نمونه ۱",
                    channel = "CALL",
                    outcome = ReportOutcome.INTERESTED,
                    interestLevel = "MEDIUM",
                    summary = "خلاصه تماس",
                    nextAction = "SEND_INFORMATION",
                    submittedBy = "u-agent-1",
                    createdAt = "2026-08-24T10:00:00Z",
                ),
            ),
        )
        val vm = ReportsViewModel(reportRepository(api))
        advanceUntilIdle()
        val state = vm.state.value
        assertTrue("expected Loaded, got $state", state is ReportsUiState.Loaded)
        assertEquals(1, (state as ReportsUiState.Loaded).reports.size)
    }

    @Test
    fun `multiple submitted reports render one card each`() = runVmTest {
        val api = MockHuginnApi()
        seedReports(
            api,
            (1..3).map { i ->
                SubmittedReport(
                    id = "rep-$i",
                    interactionId = "int-$i",
                    leadId = "lead-$i",
                    companyName = "شرکت $i",
                    channel = "CALL",
                    outcome = ReportOutcome.ANSWERED,
                    submittedBy = "u-agent-1",
                    createdAt = "2026-08-2${i}T10:00:00Z",
                )
            },
        )
        val vm = ReportsViewModel(reportRepository(api))
        advanceUntilIdle()
        val state = vm.state.value
        assertTrue("expected Loaded, got $state", state is ReportsUiState.Loaded)
        assertEquals(3, (state as ReportsUiState.Loaded).reports.size)
    }

    @Test
    fun `successful empty response is the no-reports empty state`() = runVmTest {
        val vm = ReportsViewModel(reportRepository(MockHuginnApi()))
        advanceUntilIdle()
        val state = vm.state.value
        assertTrue("expected Empty, got $state", state is ReportsUiState.Empty)
    }

    @Test
    fun `API error is an error state and never the empty state`() = runVmTest {
        val api = MockHuginnApi().apply { failReports = true }
        val vm = ReportsViewModel(reportRepository(api))
        advanceUntilIdle()
        val state = vm.state.value
        assertTrue("expected Error, got $state", state is ReportsUiState.Error)
        assertFalse("a backend failure must never be masked as 'no reports'", state is ReportsUiState.Empty)
    }

    @Test
    fun `successful submission makes the report visible in the list after refresh`() = runVmTest {
        val api = MockHuginnApi()
        val repo = reportRepository(api)
        val vm = ReportsViewModel(repo)
        advanceUntilIdle()
        assertTrue("no reports before submission", vm.state.value is ReportsUiState.Empty)

        // The Android submit flow: interaction created then mandatory report submitted.
        val interactionId = "int-e2e"
        api.seedPendingInteraction(interactionId, leadId = "lead-1")
        val result = repo.submitReport(
            interactionId,
            ReportDraft(
                outcome = ReportOutcome.INTERESTED,
                interestLevel = "HIGH",
                nextAction = "SEND_INFORMATION",
                summary = "مشتری درخواست کاتالوگ کرد",
            ).toRequest(),
        )
        assertTrue("submission must succeed online", result is ir.tafakoritech.huginn.data.repository.ReportSubmissionResult.Submitted)

        // The submitted report is released from pending-reports.
        val pending = api.pendingReports()
        assertFalse("submitted interaction must no longer require a report", pending.requiresReport)

        // The Reports tab refresh shows the newly submitted report immediately.
        vm.refresh()
        advanceUntilIdle()
        val state = vm.state.value
        assertTrue("expected Loaded after submission, got $state", state is ReportsUiState.Loaded)
        val loaded = state as ReportsUiState.Loaded
        assertTrue(
            "newly submitted report must appear in the list",
            loaded.reports.any { it.interactionId == interactionId },
        )
    }
}
