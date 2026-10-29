package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.database.PendingReportDao
import ir.tafakoritech.huginn.core.database.PendingReportLocalEntity
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.data.mock.MockHuginnApi
import ir.tafakoritech.huginn.data.model.PendingReport
import ir.tafakoritech.huginn.data.model.PendingReportsResponse
import ir.tafakoritech.huginn.data.model.ReportDraft
import ir.tafakoritech.huginn.data.model.ReportOutcome
import ir.tafakoritech.huginn.data.repository.ReportRepository
import ir.tafakoritech.huginn.feature.app.PendingReportGate
import ir.tafakoritech.huginn.feature.reports.MandatoryReportViewModel
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
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

/** Test-only Room DAO mirror for the pending-report queue (shared across tests). */
class FakePendingReportDao : PendingReportDao {
    private val store = mutableListOf<PendingReportLocalEntity>()

    override suspend fun upsert(entity: PendingReportLocalEntity) {
        store.removeAll { it.interactionId == entity.interactionId }
        store.add(entity)
    }

    override suspend fun getAll(): List<PendingReportLocalEntity> = store.toList()
    override suspend fun getById(id: String): PendingReportLocalEntity? = store.firstOrNull { it.interactionId == id }
    override suspend fun incrementAttempt(id: String, now: Long) = Unit
    override suspend fun delete(id: String) {
        store.removeAll { it.interactionId == id }
    }

    override suspend fun delete(entity: PendingReportLocalEntity) {
        store.removeAll { it.interactionId == entity.interactionId }
    }

    override suspend fun count(): Int = store.size
}

/**
 * Report-submission navigation flow. The success/result is a one-time event:
 * after "کار بعدی / Next Action" the user must leave the result screen and the
 * stale result must never reappear when the overlay reopens — even on a fresh
 * first submission, and repeated without restarting the app.
 */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class ReportSubmissionFlowTest {

    private fun reportVm(): MandatoryReportViewModel {
        val repo = ReportRepository(
            api = MockHuginnApi(),
            pendingReportDao = FakePendingReportDao(),
            json = Json { ignoreUnknownKeys = true; coerceInputValues = true },
            context = RuntimeEnvironment.getApplication(),
        )
        return MandatoryReportViewModel(repo)
    }

    private fun runVmTest(block: suspend TestScope.() -> Unit) = runTest {
        Dispatchers.setMain(StandardTestDispatcher(testScheduler))
        try {
            block()
        } finally {
            Dispatchers.resetMain()
        }
    }

    private fun pending(interactionId: String, company: String = "شرکت نمونه") = PendingReport(
        interactionId = interactionId,
        leadId = "lead-$interactionId",
        companyName = company,
        channel = "CALL",
    )

    private fun pendingWithFacts(
        interactionId: String,
        correctPerson: Boolean? = true,
        decisionMaker: Boolean? = true,
        name: String? = "علی رضایی",
        role: String? = "مدیر خرید",
    ) = PendingReport(
        interactionId = interactionId,
        leadId = "lead-$interactionId",
        companyName = "شرکت آزمون",
        channel = "CALL",
        correctPersonReached = correctPerson,
        decisionMakerIdentified = decisionMaker,
        decisionMakerName = name,
        decisionMakerRole = role,
    )

    private fun fillAndSubmit(vm: MandatoryReportViewModel) {
        vm.onOutcomeSelected(ReportOutcome.INTERESTED)
        vm.setDraft(
            ReportDraft(
                outcome = ReportOutcome.INTERESTED,
                interestLevel = "HIGH",
                nextAction = "SEND_INFORMATION",
                summary = "خلاصه گفتگو",
            ),
        )
        vm.submit()
    }

    @Test
    fun `fresh launch first submission then next action leaves the result screen`() = runVmTest {
        val vm = reportVm()
        vm.init(pending("int-A"))
        assertNull(vm.state.value.result)

        fillAndSubmit(vm)
        advanceUntilIdle()
        assertNotNull(vm.state.value.result) // result / confirmation screen

        // User taps "کار بعدی": the one-time result event is consumed exactly once.
        vm.consumeResult()
        assertNull(vm.state.value.result)
        assertNull(vm.state.value.errorKey)
        assertFalse(vm.state.value.submitting)
        assertFalse(vm.state.value.queuedOffline)
    }

    @Test
    fun `flow repeats multiple times without restarting the app`() = runVmTest {
        val vm = reportVm()
        repeat(3) { i ->
            vm.init(pending("int-$i", "شرکت $i"))
            fillAndSubmit(vm)
            advanceUntilIdle()
            assertNotNull("report $i must reach the result screen", vm.state.value.result)
            vm.consumeResult()
            assertNull("result must be consumed after report $i", vm.state.value.result)
        }
    }

    @Test
    fun `a reopened overlay never shows the stale success card`() = runVmTest {
        val vm = reportVm()
        vm.init(pending("int-A", "شرکت الف"))
        fillAndSubmit(vm)
        advanceUntilIdle()
        assertNotNull(vm.state.value.result)
        vm.consumeResult()

        // The same interaction reopens (e.g. an overlapping gate check): the
        // stale result must NOT reappear.
        vm.init(pending("int-A", "شرکت الف"))
        assertNull(vm.state.value.result)

        // A different pending report opens: fresh form, no stale result.
        vm.init(pending("int-B", "شرکت ب"))
        assertNull(vm.state.value.result)
        assertEquals("شرکت ب", vm.state.value.companyName)
    }

    @Test
    fun `invalid draft never reaches the backend or the result screen`() = runVmTest {
        val vm = reportVm()
        vm.init(pending("int-A"))
        vm.onOutcomeSelected(ReportOutcome.INTERESTED) // interest + next action missing
        vm.submit()
        advanceUntilIdle()
        assertFalse(vm.state.value.validationErrors.isEmpty())
        assertNull(vm.state.value.result)
    }

    @Test
    fun `gate stays released after clear and the next resume recheck`() = runTest {
        var backendSaysPending = true
        val gate = PendingReportGate {
            AppResult.Success(
                if (backendSaysPending) {
                    PendingReportsResponse(requiresReport = true, interactionId = "int-A")
                } else {
                    PendingReportsResponse()
                },
            )
        }
        // Report opens.
        assertNotNull(gate.check())
        // onMandatoryReportDone clears the gate (no immediate re-check).
        gate.clear()
        assertNull(gate.pendingReport.value)
        // The backend has released the report; the next ON_RESUME check confirms.
        backendSaysPending = false
        gate.check()
        assertNull(gate.pendingReport.value)
        assertTrue(!gate.hasPendingReport)
    }

    // ---- Persistent-fact pre-fill (decision maker / correct person) ----

    @Test
    fun `second report pre-fills persistent facts from the lead`() = runVmTest {
        val vm = reportVm()
        // A previous report established these facts on the backend lead.
        vm.init(pendingWithFacts("int-2"))
        val draft = vm.state.value.draft
        assertEquals(true, draft.correctPersonReached)
        assertEquals(true, draft.decisionMakerIdentified)
        assertEquals("علی رضایی", draft.decisionMakerName)
        assertEquals("مدیر خرید", draft.decisionMakerRole)
    }

    @Test
    fun `first report with no established facts keeps the draft empty`() = runVmTest {
        val vm = reportVm()
        vm.init(pending("int-1"))
        val draft = vm.state.value.draft
        assertNull(draft.correctPersonReached)
        assertNull(draft.decisionMakerIdentified)
        assertNull(draft.decisionMakerName)
        assertNull(draft.decisionMakerRole)
    }

    @Test
    fun `changing outcome keeps the persistent facts prefilled`() = runVmTest {
        val vm = reportVm()
        vm.init(pendingWithFacts("int-3"))
        vm.onOutcomeSelected(ReportOutcome.INTERESTED)
        val draft = vm.state.value.draft
        assertEquals(ReportOutcome.INTERESTED, draft.outcome)
        assertEquals(true, draft.decisionMakerIdentified)
        assertEquals("علی رضایی", draft.decisionMakerName)
        assertEquals("مدیر خرید", draft.decisionMakerRole)

        // Switching again never drops the stable facts.
        vm.onOutcomeSelected(ReportOutcome.HIGHLY_INTERESTED)
        assertEquals("علی رضایی", vm.state.value.draft.decisionMakerName)
    }

    @Test
    fun `prefilled facts are editable and survive submission`() = runVmTest {
        val vm = reportVm()
        vm.init(pendingWithFacts("int-4"))
        vm.onOutcomeSelected(ReportOutcome.INTERESTED)
        // The salesperson corrects the decision-maker name on this call.
        vm.setDraft(
            vm.state.value.draft.copy(
                interestLevel = "HIGH",
                nextAction = "SEND_INFORMATION",
                summary = "گفتگوی خوبی بود",
                decisionMakerName = "مریم احمدی",
            ),
        )
        vm.submit()
        advanceUntilIdle()
        assertNotNull(vm.state.value.result)
        // The edited value is what gets submitted (not the stale pre-fill).
        assertEquals("مریم احمدی", vm.state.value.draft.decisionMakerName)
    }
}
