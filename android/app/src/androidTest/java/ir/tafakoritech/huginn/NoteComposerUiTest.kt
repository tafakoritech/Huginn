package ir.tafakoritech.huginn

import android.Manifest
import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.speech.RecognitionListener
import android.speech.SpeechRecognizer
import androidx.appcompat.app.AppCompatDelegate
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.core.os.LocaleListCompat
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import ir.tafakoritech.huginn.core.speech.SpeechRecognizerLike
import ir.tafakoritech.huginn.core.speech.SpeechRecognizerManager
import ir.tafakoritech.huginn.core.theme.HuginnTheme
import ir.tafakoritech.huginn.feature.lead_detail.NoteComposer
import ir.tafakoritech.huginn.feature.lead_detail.NoteComposerFieldTag
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import java.util.Locale

/** Voice typing in the lead note composer. */
@RunWith(AndroidJUnit4::class)
class NoteComposerUiTest {

    @get:Rule
    val compose = createComposeRule()

    private class FakeRecognizer : SpeechRecognizerLike {
        var listener: RecognitionListener? = null

        override fun setRecognitionListener(listener: RecognitionListener) {
            this.listener = listener
        }

        override fun startListening(intent: Intent) {}

        override fun stopListening() {}

        override fun cancel() {}

        override fun destroy() {}

        fun simulateListening() = listener?.onReadyForSpeech(null)

        fun simulatePartial(text: String) = listener?.onPartialResults(resultsBundle(text))

        fun simulateResult(text: String) = listener?.onResults(resultsBundle(text))

        private fun resultsBundle(text: String): Bundle = Bundle().apply {
            putStringArrayList(SpeechRecognizer.RESULTS_RECOGNITION, arrayListOf(text))
        }
    }

    @Before
    fun setUpLocaleAndPermission() {
        AppCompatDelegate.setApplicationLocales(LocaleListCompat.create(Locale("fa")))
        InstrumentationRegistry.getInstrumentation().uiAutomation.grantRuntimePermission(
            InstrumentationRegistry.getInstrumentation().targetContext.packageName,
            Manifest.permission.RECORD_AUDIO,
        )
    }

    private fun manager(fake: FakeRecognizer, available: Boolean = true): (Context) -> SpeechRecognizerManager = { ctx ->
        SpeechRecognizerManager(
            context = ctx,
            recognizerFactory = { fake },
            availability = { available },
        )
    }

    @Test
    fun notes_composer_shows_voice_input_mic_button() {
        compose.setContent {
            HuginnTheme { NoteComposer(onAddNote = {}) }
        }
        compose.onNodeWithContentDescription("تایپ صوتی").assertIsDisplayed()
    }

    @Test
    fun voice_result_appends_to_existing_typed_text() {
        val fake = FakeRecognizer()
        compose.setContent {
            HuginnTheme {
                val context = LocalContext.current
                val speechManager = remember(context) { manager(fake)(context.applicationContext) }
                NoteComposer(onAddNote = {}, speechManager = speechManager)
            }
        }
        compose.onNodeWithTag(NoteComposerFieldTag).performTextInput("متن دستی")
        compose.onNodeWithContentDescription("تایپ صوتی").performClick()
        fake.simulateListening()
        fake.simulateResult("بخش گفتاری")
        compose.onNodeWithText("متن دستی بخش گفتاری").assertIsDisplayed()
    }

    @Test
    fun unavailable_recognition_shows_localized_error() {
        compose.setContent {
            HuginnTheme {
                val context = LocalContext.current
                val speechManager = remember(context) { manager(FakeRecognizer(), available = false)(context.applicationContext) }
                NoteComposer(onAddNote = {}, speechManager = speechManager)
            }
        }
        compose.onNodeWithContentDescription("تایپ صوتی").performClick()
        compose.onNodeWithText("تشخیص گفتار روی این دستگاه در دسترس نیست.").assertIsDisplayed()
    }
}
