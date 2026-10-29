package ir.tafakoritech.huginn

import android.content.pm.PackageInfo
import androidx.lifecycle.SavedStateHandle
import ir.tafakoritech.huginn.core.external.CallLauncher
import ir.tafakoritech.huginn.core.external.ContactInsertLauncher
import ir.tafakoritech.huginn.core.external.EmailLauncher
import ir.tafakoritech.huginn.core.external.LaunchResult
import ir.tafakoritech.huginn.core.external.MessengerChannels
import ir.tafakoritech.huginn.core.external.MessengerLauncher
import ir.tafakoritech.huginn.core.external.SmsLauncher
import ir.tafakoritech.huginn.data.mock.MockHuginnApi
import ir.tafakoritech.huginn.data.model.DeepLink
import ir.tafakoritech.huginn.data.model.InteractionChannel
import ir.tafakoritech.huginn.data.repository.FollowUpRepository
import ir.tafakoritech.huginn.data.repository.InteractionRepository
import ir.tafakoritech.huginn.data.repository.LeadRepository
import ir.tafakoritech.huginn.data.repository.QualificationRepository
import ir.tafakoritech.huginn.feature.lead_detail.LeadDetailViewModel
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.UnconfinedTestDispatcher
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
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config

/** Test-only fakes for the external launchers. */
private class FakeSmsLauncher : SmsLauncher {
    var lastText: String? = null
    override suspend fun launch(number: String, text: String): LaunchResult {
        lastText = text
        return LaunchResult.Launched
    }
}

private class FakeMessengerLauncher : MessengerLauncher {
    var lastChannel: String? = null
    var lastPhone: String? = null
    var lastText: String? = null
    override suspend fun launch(channel: String, deepLink: DeepLink?, phone: String?, text: String?): LaunchResult {
        lastChannel = channel
        lastPhone = phone
        lastText = text
        return LaunchResult.Launched
    }
}

private object NoopCallLauncher : CallLauncher {
    override suspend fun launch(phone: String): LaunchResult = LaunchResult.Launched
}

private object NoopEmailLauncher : EmailLauncher {
    override suspend fun launch(recipient: String?, subject: String?, body: String?): LaunchResult = LaunchResult.Launched
}

private object NoopContactInsertLauncher : ContactInsertLauncher {
    override suspend fun launch(name: String, phone: String?, company: String?, jobTitle: String?, email: String?): LaunchResult =
        LaunchResult.Launched
}

/**
 * Send Message flow. Selecting a channel must open the composer immediately
 * (with a visible progress state), auto-populate the suggested text when the
 * backend responds, keep the composer open with a retry on failure, and never
 * fire duplicate generation requests. Uses SMS so the test is independent of
 * the device messenger-install check.
 */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class SendMessageFlowTest {

    private fun json(): Json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private fun vm(api: MockHuginnApi = MockHuginnApi(), sms: FakeSmsLauncher = FakeSmsLauncher(), messenger: FakeMessengerLauncher = FakeMessengerLauncher()): Triple<LeadDetailViewModel, FakeSmsLauncher, FakeMessengerLauncher> {
        val handle = SavedStateHandle(mapOf("leadId" to "lead-1"))
        val vm = LeadDetailViewModel(
            savedStateHandle = handle,
            leadRepository = LeadRepository(api, json()),
            interactionRepository = InteractionRepository(api, api, json()),
            followUpRepository = FollowUpRepository(api, json()),
            qualificationRepository = QualificationRepository(api, json()),
            callLauncher = NoopCallLauncher,
            smsLauncher = sms,
            messengerLauncher = messenger,
            emailLauncher = NoopEmailLauncher,
            contactInsertLauncher = NoopContactInsertLauncher,
            appContext = RuntimeEnvironment.getApplication(),
        )
        return Triple(vm, sms, messenger)
    }

    private fun runVmTest(block: suspend TestScope.() -> Unit) = runTest {
        Dispatchers.setMain(UnconfinedTestDispatcher(testScheduler))
        try {
            block()
        } finally {
            Dispatchers.resetMain()
        }
    }

    @Test
    fun `selecting a channel opens the composer immediately with a loading state`() = runVmTest {
        val (vm, _, _) = vm()
        advanceUntilIdle() // initial lead load

        vm.openMessageSheet()
        vm.onSelectChannel(InteractionChannel.SMS)

        // Composer open on the first tap, before the backend responds.
        val draft = vm.state.value.suggestedSheet
        assertNotNull(draft)
        assertEquals(InteractionChannel.SMS, draft?.channel)
        assertEquals("", draft?.text)
        assertTrue(vm.state.value.loadingSuggested)

        advanceUntilIdle()
        // Backend text auto-populates.
        assertFalse(vm.state.value.loadingSuggested)
        assertTrue(vm.state.value.suggestedSheet?.text.isNullOrBlank().not())
        assertFalse(vm.state.value.suggestedSheet?.interactionId.isNullOrBlank())
    }

    @Test
    fun `generated text can be edited before sending and the edit is used`() = runVmTest {
        val (vm, sms, _) = vm()
        advanceUntilIdle()
        vm.openMessageSheet()
        vm.onSelectChannel(InteractionChannel.SMS)
        advanceUntilIdle()

        val original = vm.state.value.suggestedSheet?.text.orEmpty()
        assertTrue(original.isNotBlank())

        vm.updateSuggestedText("پیام ویرایش‌شده")
        assertEquals("پیام ویرایش‌شده", vm.state.value.suggestedSheet?.text)

        vm.openMessenger(vm.state.value.suggestedSheet!!, "پیام ویرایش‌شده")
        advanceUntilIdle()
        assertEquals("پیام ویرایش‌شده", sms.lastText)
        assertNull(vm.state.value.suggestedSheet) // composer dismissed after launch
    }

    @Test
    fun `repeated channel taps do not trigger duplicate backend requests`() = runVmTest {
        val api = MockHuginnApi()
        val (vm, _, _) = vm(api)
        advanceUntilIdle()
        vm.openMessageSheet()
        vm.onSelectChannel(InteractionChannel.SMS)

        // While the first generation is still in flight, tapping again is ignored.
        vm.openMessageSheet()
        vm.onSelectChannel(InteractionChannel.SMS)
        assertTrue(vm.state.value.loadingSuggested)

        advanceUntilIdle()
        assertEquals(1, api.contactActionCallCount)
        assertFalse(vm.state.value.suggestedSheet?.text.isNullOrBlank())
    }

    @Test
    fun `generation failure keeps the composer open with a retry`() = runVmTest {
        val api = MockHuginnApi().apply { failContactAction = true }
        val (vm, _, _) = vm(api)
        advanceUntilIdle()
        vm.openMessageSheet()
        vm.onSelectChannel(InteractionChannel.SMS)
        advanceUntilIdle()

        assertTrue(vm.state.value.suggestError)
        assertNotNull(vm.state.value.suggestedSheet) // composer stays open
        assertFalse(vm.state.value.loadingSuggested)

        api.failContactAction = false
        vm.retrySuggested()
        advanceUntilIdle()
        assertFalse(vm.state.value.suggestError)
        assertTrue(vm.state.value.suggestedSheet?.text.isNullOrBlank().not())
        assertEquals(2, api.contactActionCallCount)
    }

    @Test
    fun `cancelling during loading discards the stale response`() = runVmTest {
        val (vm, _, _) = vm()
        advanceUntilIdle()
        vm.openMessageSheet()
        vm.onSelectChannel(InteractionChannel.SMS)
        assertTrue(vm.state.value.loadingSuggested)

        vm.dismissSuggestedSheet()
        assertNull(vm.state.value.suggestedSheet)
        assertFalse(vm.state.value.loadingSuggested)

        advanceUntilIdle()
        // The late response must not reopen the composer.
        assertNull(vm.state.value.suggestedSheet)
    }

    @Test
    fun `telegram flow passes the contact mobile to the messenger launcher`() = runVmTest {
        // Mark Telegram as installed so the channel becomes eligible via the phone.
        val appContext = RuntimeEnvironment.getApplication()
        shadowOf(appContext.packageManager).addPackage(
            PackageInfo().apply { packageName = MessengerChannels.known[InteractionChannel.TELEGRAM]!!.candidates.first() },
        )

        val (vm, _, messenger) = vm()
        advanceUntilIdle()
        vm.openMessageSheet()
        vm.onSelectChannel(InteractionChannel.TELEGRAM)
        advanceUntilIdle()

        val draft = vm.state.value.suggestedSheet
        assertNotNull("composer must open", draft)
        assertEquals(InteractionChannel.TELEGRAM, draft?.channel)

        vm.openMessenger(draft!!, draft.text)
        advanceUntilIdle()

        // The launcher receives the contact's real backend mobile (primaryNumber).
        assertEquals(InteractionChannel.TELEGRAM, messenger.lastChannel)
        assertEquals("+989357578808", messenger.lastPhone)
        assertNull(vm.state.value.suggestedSheet)
    }
}
