package ir.tafakoritech.huginn

import androidx.lifecycle.SavedStateHandle
import ir.tafakoritech.huginn.core.external.CallLauncher
import ir.tafakoritech.huginn.core.external.ContactInsertLauncher
import ir.tafakoritech.huginn.core.external.EmailLauncher
import ir.tafakoritech.huginn.core.external.LaunchResult
import ir.tafakoritech.huginn.core.external.MessengerLauncher
import ir.tafakoritech.huginn.core.external.SmsLauncher
import ir.tafakoritech.huginn.data.mock.MockHuginnApi
import ir.tafakoritech.huginn.data.model.CallScenario
import ir.tafakoritech.huginn.data.model.LoginRequest
import ir.tafakoritech.huginn.data.repository.FollowUpRepository
import ir.tafakoritech.huginn.data.repository.InteractionRepository
import ir.tafakoritech.huginn.data.repository.LeadRepository
import ir.tafakoritech.huginn.data.repository.QualificationRepository
import ir.tafakoritech.huginn.feature.lead_detail.LeadDetailViewModel
import ir.tafakoritech.huginn.feature.teleprompter.TeleprompterManager
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.serialization.json.Json
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

private class FakeCallLauncher : CallLauncher {
    var callCount = 0
    var lastPhone: String? = null
    override suspend fun launch(phone: String): LaunchResult {
        callCount++
        lastPhone = phone
        return LaunchResult.Launched
    }
}

private class NoOpSmsLauncher : SmsLauncher {
    override suspend fun launch(number: String, text: String): LaunchResult = LaunchResult.Launched
}

private class NoOpMessengerLauncher : MessengerLauncher {
    override suspend fun launch(
        channel: String,
        deepLink: ir.tafakoritech.huginn.data.model.DeepLink?,
        phone: String?,
        text: String?,
    ): LaunchResult = LaunchResult.Launched
}

private class NoOpEmailLauncher : EmailLauncher {
    override suspend fun launch(recipient: String?, subject: String?, body: String?): LaunchResult = LaunchResult.Launched
}

private class NoOpContactInsertLauncher : ContactInsertLauncher {
    override suspend fun launch(
        name: String,
        phone: String?,
        company: String?,
        jobTitle: String?,
        email: String?,
    ): LaunchResult = LaunchResult.Launched
}

@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class CallScenarioFlowTest {

    private val testDispatcher = UnconfinedTestDispatcher()
    private val json = Json { ignoreUnknownKeys = true }
    private lateinit var mockApi: MockHuginnApi
    private lateinit var leadRepo: LeadRepository
    private lateinit var interactionRepo: InteractionRepository
    private lateinit var followUpRepo: FollowUpRepository
    private lateinit var qualRepo: QualificationRepository
    private val callLauncher = FakeCallLauncher()

    @Before
    fun setup() = runTest(testDispatcher) {
        Dispatchers.setMain(testDispatcher)
        mockApi = MockHuginnApi()
        mockApi.login(LoginRequest(login = "agent", password = "password"))
        leadRepo = LeadRepository(mockApi, json)
        interactionRepo = InteractionRepository(mockApi, mockApi, json)
        followUpRepo = FollowUpRepository(mockApi, json)
        qualRepo = QualificationRepository(mockApi, json)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private fun createViewModel(leadId: String = "lead-1"): LeadDetailViewModel {
        return LeadDetailViewModel(
            savedStateHandle = SavedStateHandle(mapOf("leadId" to leadId)),
            leadRepository = leadRepo,
            interactionRepository = interactionRepo,
            followUpRepository = followUpRepo,
            qualificationRepository = qualRepo,
            callLauncher = callLauncher,
            smsLauncher = NoOpSmsLauncher(),
            messengerLauncher = NoOpMessengerLauncher(),
            emailLauncher = NoOpEmailLauncher(),
            contactInsertLauncher = NoOpContactInsertLauncher(),
            appContext = RuntimeEnvironment.getApplication(),
        )
    }

    @Test
    fun `acceptance test 1 - generate scenario, edit text, save, restart, verify persistence`() = runTest(testDispatcher) {
        val vm1 = createViewModel("lead-1")
        advanceUntilIdle()

        // 1. Generate scenario
        vm1.requestCallScenario()
        advanceUntilIdle()

        val generated = vm1.state.value.scenario.scenario
        assertNotNull("Generated scenario must exist", generated)
        assertNotNull("Generated scenario must have an ID", generated?.id)
        assertTrue("Generated scenario is marked as generated", generated?.isGenerated == true)

        // 2. Edit the scenario
        val edited = generated!!.copy(
            openingSentence = "سلام جناب مهندس، وقتتون بخیر",
            reasonForCall = "تماس بابت همکاری در بخش بسته‌بندی",
            recommendedNextStep = "ارسال کاتالوگ در پیام‌رسان",
        )
        vm1.saveCallScenario(edited)
        advanceUntilIdle()

        assertEquals("scenario_saved", vm1.state.value.notice)
        assertEquals("سلام جناب مهندس، وقتتون بخیر", vm1.state.value.scenario.scenario?.openingSentence)

        // 3. Close app / re-instantiate VM against same backend
        val vm2 = createViewModel("lead-1")
        advanceUntilIdle()

        // Verify edited text is persisted in archive
        val archivedList = vm2.state.value.scenarios
        assertEquals(1, archivedList.size)
        val saved = archivedList.first()
        assertEquals("سلام جناب مهندس، وقتتون بخیر", saved.openingSentence)
        assertEquals("تماس بابت همکاری در بخش بسته‌بندی", saved.reasonForCall)
        assertEquals("ارسال کاتالوگ در پیام‌رسان", saved.recommendedNextStep)
    }

    @Test
    fun `acceptance test 2 - generate multiple scenarios for one lead, all appear in archive`() = runTest(testDispatcher) {
        val vm = createViewModel("lead-1")
        advanceUntilIdle()

        // Generate scenario 1
        vm.requestCallScenario()
        advanceUntilIdle()
        vm.dismissCallScenario()

        // Generate scenario 2
        vm.requestCallScenario()
        advanceUntilIdle()
        vm.dismissCallScenario()

        // Also create a manual scenario
        val manual = CallScenario(
            leadId = "lead-1",
            openingSentence = "سناریوی دستی مدیر فروش",
            introduction = "معرفی سازمانی",
            reasonForCall = "پیشنهاد تخفیف فصلی",
        )
        vm.saveCallScenario(manual)
        advanceUntilIdle()

        // Verify archive has all 3 scenarios
        val scenarios = vm.state.value.scenarios
        assertEquals(3, scenarios.size)
        assertTrue(scenarios.any { it.openingSentence == "سناریوی دستی مدیر فروش" && !it.isGenerated })
        assertEquals(2, scenarios.count { it.isGenerated })
    }

    @Test
    fun `acceptance test 3 - open archived scenario and start call`() = runTest(testDispatcher) {
        val vm = createViewModel("lead-1")
        advanceUntilIdle()

        // Create a scenario
        vm.requestCallScenario()
        advanceUntilIdle()
        val sc = vm.state.value.scenario.scenario!!
        vm.dismissCallScenario()

        // Open scenario from archive
        vm.openScenario(sc, editMode = false)
        assertTrue(vm.state.value.scenario.visible)
        assertEquals(sc.id, vm.state.value.scenario.scenario?.id)

        // Start call with this scenario
        vm.startCallWithScenario(sc)
        advanceUntilIdle()

        // In test environment without overlay permission granted, permission dialog explains limitation
        assertTrue(vm.state.value.overlayPermissionDialogVisible)
        assertEquals(sc.id, vm.state.value.pendingCallScenario?.id)
    }

    @Test
    fun `acceptance test 4 - close and reopen same scenario later`() = runTest(testDispatcher) {
        val vm = createViewModel("lead-1")
        advanceUntilIdle()

        vm.requestCallScenario()
        advanceUntilIdle()
        val original = vm.state.value.scenario.scenario!!

        // Close scenario sheet
        vm.dismissCallScenario()
        assertFalse(vm.state.value.scenario.visible)
        assertNull(vm.state.value.scenario.scenario)

        // Reopen later from archive
        vm.openScenario(original, editMode = false)
        assertTrue(vm.state.value.scenario.visible)
        assertEquals(original.id, vm.state.value.scenario.scenario?.id)
        assertEquals(original.openingSentence, vm.state.value.scenario.scenario?.openingSentence)
    }

    @Test
    fun `acceptance test 5 - permission denied flow explains limitation and allows normal calling`() = runTest(testDispatcher) {
        val vm = createViewModel("lead-1")
        advanceUntilIdle()

        val initialCallCount = callLauncher.callCount

        // Start call with scenario without overlay permission
        val scenario = CallScenario(leadId = "lead-1", openingSentence = "سلام")
        vm.startCallWithScenario(scenario)

        // Overlay permission dialog is shown
        assertTrue(vm.state.value.overlayPermissionDialogVisible)
        assertEquals(initialCallCount, callLauncher.callCount)

        // User chooses "Call without Teleprompter"
        vm.proceedCallWithoutTeleprompter()
        advanceUntilIdle()

        // Permission dialog dismissed and call proceeds normally
        assertFalse(vm.state.value.overlayPermissionDialogVisible)
        assertEquals(initialCallCount + 1, callLauncher.callCount)
    }

    @Test
    fun `delete scenario removes it from archive and backend`() = runTest(testDispatcher) {
        val vm = createViewModel("lead-1")
        advanceUntilIdle()

        vm.requestCallScenario()
        advanceUntilIdle()
        val sc = vm.state.value.scenario.scenario!!

        assertEquals(1, vm.state.value.scenarios.size)

        // Request delete
        vm.deleteCallScenario(sc)
        assertEquals(sc.id, vm.state.value.deleteScenarioTarget?.id)

        // Confirm delete
        vm.confirmDeleteScenario()
        advanceUntilIdle()

        assertEquals("scenario_deleted", vm.state.value.notice)
        assertEquals(0, vm.state.value.scenarios.size)
    }
}
