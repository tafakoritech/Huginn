package ir.tafakoritech.huginn

import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.os.Looper
import android.speech.RecognitionListener
import android.speech.RecognizerIntent
import android.speech.SpeechRecognizer
import ir.tafakoritech.huginn.core.speech.SpeechErrorKind
import ir.tafakoritech.huginn.core.speech.SpeechEvent
import ir.tafakoritech.huginn.core.speech.SpeechRecognizerLike
import ir.tafakoritech.huginn.core.speech.SpeechRecognizerManager
import ir.tafakoritech.huginn.core.speech.appendRecognizedText
import ir.tafakoritech.huginn.core.speech.speechErrorLabelRes
import ir.tafakoritech.huginn.core.speech.speechLanguageTag
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config
import java.util.Locale

/** Persian voice-input manager: event flow, error mapping and text insertion. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class SpeechRecognizerManagerTest {

    private class FakeRecognizer : SpeechRecognizerLike {
        var listener: RecognitionListener? = null
        var started = false
        var startIntent: Intent? = null

        override fun setRecognitionListener(listener: RecognitionListener) {
            this.listener = listener
        }

        override fun startListening(intent: Intent) {
            started = true
            startIntent = intent
        }

        override fun stopListening() {
            started = false
        }

        override fun cancel() {
            started = false
        }

        override fun destroy() {
            started = false
        }

        fun simulateListening() = listener?.onReadyForSpeech(null)
        fun simulatePartial(text: String) = listener?.onPartialResults(resultsBundle(text))
        fun simulateResult(text: String) = listener?.onResults(resultsBundle(text))
        fun simulateError(code: Int) = listener?.onError(code)

        private fun resultsBundle(text: String): Bundle = Bundle().apply {
            putStringArrayList(SpeechRecognizer.RESULTS_RECOGNITION, arrayListOf(text))
        }
    }

    private fun manager(fake: FakeRecognizer): SpeechRecognizerManager = SpeechRecognizerManager(
        context = RuntimeEnvironment.getApplication(),
        recognizerFactory = { fake },
        availability = { true },
    )

    private fun idleMainLooper() = shadowOf(Looper.getMainLooper()).idle()

    @Test
    fun `start emits listening then the final result`() {
        val fake = FakeRecognizer()
        val m = manager(fake)
        val events = mutableListOf<SpeechEvent>()
        assertTrue(m.start { events.add(it) })
        fake.simulateListening()
        fake.simulateResult("سلام")
        idleMainLooper()
        assertEquals(listOf(SpeechEvent.Listening, SpeechEvent.Result("سلام")), events)
        m.destroy()
    }

    @Test
    fun `partial results arrive before the final result`() {
        val fake = FakeRecognizer()
        val m = manager(fake)
        val events = mutableListOf<SpeechEvent>()
        m.start { events.add(it) }
        fake.simulateListening()
        fake.simulatePartial("سلام")
        fake.simulateResult("سلام، این است متن")
        idleMainLooper()
        assertEquals(
            listOf(
                SpeechEvent.Listening,
                SpeechEvent.Partial("سلام"),
                SpeechEvent.Result("سلام، این است متن"),
            ),
            events,
        )
        m.destroy()
    }

    @Test
    fun `start requests Persian recognition`() {
        val fake = FakeRecognizer()
        val m = manager(fake)
        m.start { }
        idleMainLooper()
        assertEquals("fa-IR", fake.startIntent?.getStringExtra(RecognizerIntent.EXTRA_LANGUAGE))
        assertEquals("fa-IR", fake.startIntent?.getStringExtra(RecognizerIntent.EXTRA_LANGUAGE_PREFERENCE))
        assertEquals(true, fake.startIntent?.getBooleanExtra(RecognizerIntent.EXTRA_PARTIAL_RESULTS, false))
        assertTrue(fake.started)
        m.destroy()
    }

    @Test
    fun `stop still delivers the final result`() {
        val fake = FakeRecognizer()
        val m = manager(fake)
        val events = mutableListOf<SpeechEvent>()
        m.start { events.add(it) }
        fake.simulateListening()
        m.stop()
        fake.simulateResult("متن نهایی")
        idleMainLooper()
        assertTrue(events.contains(SpeechEvent.Result("متن نهایی")))
        m.destroy()
    }

    @Test
    fun `unavailable recognizer emits not-available error`() {
        val m = SpeechRecognizerManager(
            context = RuntimeEnvironment.getApplication(),
            recognizerFactory = { null },
            availability = { false },
        )
        val events = mutableListOf<SpeechEvent>()
        assertFalse(m.start { events.add(it) })
        idleMainLooper()
        assertEquals(listOf(SpeechEvent.Error(SpeechErrorKind.NOT_AVAILABLE)), events)
        m.destroy()
    }

    @Test
    fun `android error codes map to stable error kinds`() {
        val cases = mapOf(
            SpeechRecognizer.ERROR_INSUFFICIENT_PERMISSIONS to SpeechErrorKind.PERMISSION_DENIED,
            SpeechRecognizer.ERROR_SPEECH_TIMEOUT to SpeechErrorKind.TIMEOUT,
            SpeechRecognizer.ERROR_NO_MATCH to SpeechErrorKind.NO_MATCH,
            SpeechRecognizer.ERROR_NETWORK to SpeechErrorKind.FAILURE,
            SpeechRecognizer.ERROR_NETWORK_TIMEOUT to SpeechErrorKind.FAILURE,
            SpeechRecognizer.ERROR_SERVER to SpeechErrorKind.FAILURE,
            SpeechRecognizer.ERROR_AUDIO to SpeechErrorKind.FAILURE,
            SpeechRecognizer.ERROR_RECOGNIZER_BUSY to SpeechErrorKind.FAILURE,
            SpeechRecognizer.ERROR_CLIENT to SpeechErrorKind.FAILURE,
            SpeechRecognizer.ERROR_LANGUAGE_UNAVAILABLE to SpeechErrorKind.NOT_AVAILABLE,
            SpeechRecognizer.ERROR_CANNOT_CHECK_SUPPORT to SpeechErrorKind.NOT_AVAILABLE,
        )
        cases.forEach { (code, expected) ->
            assertEquals(expected, SpeechErrorKind.fromErrorCode(code))
        }
    }

    @Test
    fun `every error kind has a non-blank localized label`() {
        val context = RuntimeEnvironment.getApplication()
        SpeechErrorKind.entries.forEach { kind ->
            val label = context.getString(speechErrorLabelRes(kind))
            assertFalse("Label for $kind is blank", label.isBlank())
        }
    }

    @Test
    fun `appendRecognizedText never overwrites existing text`() {
        assertEquals("", appendRecognizedText("", ""))
        assertEquals("متن", appendRecognizedText("", "متن"))
        assertEquals("قبلی", appendRecognizedText("قبلی", "   "))
        assertEquals("قبلی جدید", appendRecognizedText("قبلی", "جدید"))
        assertEquals("قبلی جدید", appendRecognizedText("قبلی ", "جدید"))
        assertEquals("قبلی\nجدید", appendRecognizedText("قبلی\n", "جدید"))
    }

    @Test
    fun `appendRecognizedText accumulates repeated dictations`() {
        val first = appendRecognizedText("", "گزارش اولیه")
        val second = appendRecognizedText(first, "و ادامه آن")
        assertEquals("گزارش اولیه و ادامه آن", second)
    }

    @Test
    fun `speech language tag follows the app language`() {
        assertEquals("fa-IR", speechLanguageTag(Locale("fa")))
        assertEquals("fa-IR", speechLanguageTag(Locale("fa", "IR")))
        assertEquals("en-US", speechLanguageTag(Locale.US))
        assertEquals("fa-IR", speechLanguageTag(Locale.GERMAN))
    }

    @Test
    fun `start requests the given language tag`() {
        val fake = FakeRecognizer()
        val m = manager(fake)
        m.start("en-US") { }
        idleMainLooper()
        assertEquals("en-US", fake.startIntent?.getStringExtra(RecognizerIntent.EXTRA_LANGUAGE))
        m.destroy()
    }
}
