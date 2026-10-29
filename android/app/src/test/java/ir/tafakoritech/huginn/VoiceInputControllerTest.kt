package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.speech.SpeechErrorKind
import ir.tafakoritech.huginn.core.speech.SpeechEvent
import ir.tafakoritech.huginn.core.speech.VoiceInputController
import ir.tafakoritech.huginn.core.speech.VoiceInputState
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/** Voice-input state machine shared by the note composer and report summary. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class VoiceInputControllerTest {

    @Test
    fun `starts idle with no partial text`() {
        val controller = VoiceInputController()
        assertEquals(VoiceInputState.Idle, controller.state)
        assertEquals("", controller.partialText)
    }

    @Test
    fun `begin moves to listening and clears partial text`() {
        val controller = VoiceInputController()
        controller.begin()
        assertEquals(VoiceInputState.Listening, controller.state)
        assertEquals("", controller.partialText)
    }

    @Test
    fun `listening event keeps listening and clears partial text`() {
        val controller = VoiceInputController()
        controller.begin()
        controller.onEvent(SpeechEvent.Partial("قبلی"))
        controller.onEvent(SpeechEvent.Listening)
        assertEquals(VoiceInputState.Listening, controller.state)
        assertEquals("", controller.partialText)
    }

    @Test
    fun `partial event surfaces live text while listening`() {
        val controller = VoiceInputController()
        controller.onEvent(SpeechEvent.Partial("در حال گفتن"))
        assertEquals(VoiceInputState.Listening, controller.state)
        assertEquals("در حال گفتن", controller.partialText)
    }

    @Test
    fun `processing event reports the processing phase`() {
        val controller = VoiceInputController()
        controller.begin()
        controller.onEvent(SpeechEvent.Processing)
        assertEquals(VoiceInputState.Processing, controller.state)
    }

    @Test
    fun `result delivers the utterance and returns to idle`() {
        val results = mutableListOf<String>()
        val controller = VoiceInputController(onResult = { results.add(it) })
        controller.begin()
        controller.onEvent(SpeechEvent.Result("متن نهایی"))
        assertEquals(VoiceInputState.Idle, controller.state)
        assertEquals("", controller.partialText)
        assertEquals(listOf("متن نهایی"), results)
    }

    @Test
    fun `blank result is dropped without invoking onResult`() {
        var calls = 0
        val controller = VoiceInputController(onResult = { calls++ })
        controller.onEvent(SpeechEvent.Result("   "))
        assertEquals(0, calls)
        assertEquals(VoiceInputState.Idle, controller.state)
    }

    @Test
    fun `error event maps to a stable error state and clears partial text`() {
        val controller = VoiceInputController()
        controller.begin()
        controller.onEvent(SpeechEvent.Partial("کوتاه"))
        controller.onEvent(SpeechEvent.Error(SpeechErrorKind.TIMEOUT))
        assertEquals(VoiceInputState.Error(SpeechErrorKind.TIMEOUT), controller.state)
        assertEquals("", controller.partialText)
    }

    @Test
    fun `permission denial surfaces the permission error`() {
        val controller = VoiceInputController()
        controller.onPermissionDenied()
        assertEquals(VoiceInputState.Error(SpeechErrorKind.PERMISSION_DENIED), controller.state)
    }

    @Test
    fun `reset aborts an active session back to idle`() {
        val controller = VoiceInputController()
        controller.begin()
        controller.onEvent(SpeechEvent.Error(SpeechErrorKind.NO_MATCH))
        controller.reset()
        assertEquals(VoiceInputState.Idle, controller.state)
        assertEquals("", controller.partialText)
    }

    @Test
    fun `dictation can be retried after an error`() {
        val controller = VoiceInputController()
        controller.onEvent(SpeechEvent.Error(SpeechErrorKind.NOT_AVAILABLE))
        assertTrue(controller.state is VoiceInputState.Error)
        controller.begin()
        assertEquals(VoiceInputState.Listening, controller.state)
    }
}
