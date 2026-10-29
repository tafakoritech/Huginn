package ir.tafakoritech.huginn

import androidx.lifecycle.SavedStateHandle
import ir.tafakoritech.huginn.core.speech.appendRecognizedText
import ir.tafakoritech.huginn.data.mock.MockHuginnApi
import ir.tafakoritech.huginn.data.repository.QualificationRepository
import ir.tafakoritech.huginn.feature.qualification.DisqualifyViewModel
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
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class DisqualifyFlowTest {

    private fun json() = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private fun runVmTest(block: suspend TestScope.() -> Unit) = runTest {
        Dispatchers.setMain(StandardTestDispatcher(testScheduler))
        try {
            block()
        } finally {
            Dispatchers.resetMain()
        }
    }

    private fun vm(api: MockHuginnApi = MockHuginnApi()): DisqualifyViewModel =
        DisqualifyViewModel(
            SavedStateHandle(mapOf("leadId" to "lead-1")),
            QualificationRepository(api, json()),
        )

    @Test
    fun `reject lead submits the reason and explanation to the backend`() = runVmTest {
        val viewModel = vm()
        viewModel.selectReason("NOT_RELEVANT")
        viewModel.onExplanationChange("سرنخ نامرتبط با حوزه کاری")
        viewModel.submit()
        advanceUntilIdle()
        val state = viewModel.state.value
        assertTrue("expected done after successful submit, got $state", state.done)
        assertEquals("NOT_RELEVANT", state.reason)
    }

    @Test
    fun `reject with OTHER requires a non-blank explanation`() = runVmTest {
        val viewModel = vm()
        viewModel.selectReason("OTHER")
        viewModel.submit()
        val state = viewModel.state.value
        assertEquals("explanation_required", state.errorKey)
        assertTrue("must not submit without explanation", !state.done && !state.submitting)
    }

    @Test
    fun `voice typed text is appended to the existing reject description`() = runVmTest {
        val viewModel = vm()
        viewModel.selectReason("OTHER")
        viewModel.onExplanationChange("نام تجاری تکراری است")
        val appended = appendRecognizedText(viewModel.state.value.explanation, "قبلا به فروشنده دیگری فروخته شده")
        viewModel.onExplanationChange(appended)
        assertEquals(
            "نام تجاری تکراری است قبلا به فروشنده دیگری فروخته شده",
            viewModel.state.value.explanation,
        )
    }

    @Test
    fun `voice typing never clears previously typed reject description`() = runVmTest {
        val viewModel = vm()
        viewModel.onExplanationChange("متن قبلی")
        val withVoice = appendRecognizedText(viewModel.state.value.explanation, "متن گفتاری")
        viewModel.onExplanationChange(withVoice)
        assertTrue(viewModel.state.value.explanation.startsWith("متن قبلی"))
        assertTrue(viewModel.state.value.explanation.endsWith("متن گفتاری"))
    }
}
