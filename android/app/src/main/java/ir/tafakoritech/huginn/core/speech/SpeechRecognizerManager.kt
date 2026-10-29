package ir.tafakoritech.huginn.core.speech

import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.speech.RecognitionListener
import android.speech.RecognizerIntent
import android.speech.SpeechRecognizer
import androidx.annotation.StringRes
import ir.tafakoritech.huginn.R
import java.util.Locale

/**
 * Failure categories surfaced to the UI for Persian voice input.
 */
enum class SpeechErrorKind {
    PERMISSION_DENIED,
    NOT_AVAILABLE,
    TIMEOUT,
    NO_MATCH,
    FAILURE;

    companion object {
        /** Map an Android [SpeechRecognizer] error code to a stable UI category. */
        fun fromErrorCode(error: Int): SpeechErrorKind = when (error) {
            SpeechRecognizer.ERROR_INSUFFICIENT_PERMISSIONS -> PERMISSION_DENIED

            SpeechRecognizer.ERROR_SPEECH_TIMEOUT -> TIMEOUT
            SpeechRecognizer.ERROR_NO_MATCH -> NO_MATCH

            SpeechRecognizer.ERROR_CANNOT_CHECK_SUPPORT,
            SpeechRecognizer.ERROR_LANGUAGE_NOT_SUPPORTED,
            SpeechRecognizer.ERROR_LANGUAGE_UNAVAILABLE -> NOT_AVAILABLE

            else -> FAILURE
        }
    }
}

/** Localized label for a [SpeechErrorKind]. */
@StringRes
fun speechErrorLabelRes(kind: SpeechErrorKind): Int = when (kind) {
    SpeechErrorKind.PERMISSION_DENIED -> R.string.voice_permission_denied
    SpeechErrorKind.NOT_AVAILABLE -> R.string.voice_not_available
    SpeechErrorKind.TIMEOUT -> R.string.voice_timeout
    SpeechErrorKind.NO_MATCH -> R.string.voice_no_match
    SpeechErrorKind.FAILURE -> R.string.voice_failed
}

/** Events emitted by [SpeechRecognizerManager] while dictating. */
sealed interface SpeechEvent {
    data object Listening : SpeechEvent
    data object Processing : SpeechEvent
    data class Partial(val text: String) : SpeechEvent
    data class Result(val text: String) : SpeechEvent
    data class Error(val kind: SpeechErrorKind) : SpeechEvent
}

/**
 * Recognition language tag for the current app/device language (Persian ->
 * fa-IR, English -> en-US, anything else -> fa-IR). The platform
 * [SpeechRecognizer] needs a concrete BCP-47 tag; fa-IR matches the app's
 * Persian-first voice input.
 */
fun speechLanguageTag(locale: Locale = Locale.getDefault()): String = when (locale.language) {
    "fa" -> SpeechRecognizerManager.PERSIAN_LANGUAGE
    "en" -> "en-US"
    else -> SpeechRecognizerManager.PERSIAN_LANGUAGE
}

/**
 * Recognition language tag for the current app/device configuration, honoring
 * the app locale applied via AppCompatDelegate.setApplicationLocales.
 */
fun speechLanguageTag(context: Context): String {
    val locales = context.resources.configuration.locales
    val locale = if (locales.isEmpty) Locale.getDefault() else locales[0]
    return speechLanguageTag(locale)
}

/** Thin seam over [SpeechRecognizer] so the manager stays unit-testable. */
interface SpeechRecognizerLike {
    fun setRecognitionListener(listener: RecognitionListener)
    fun startListening(intent: Intent)
    fun stopListening()
    fun cancel()
    fun destroy()
}

private class RealSpeechRecognizer(
    private val delegate: SpeechRecognizer,
) : SpeechRecognizerLike {
    override fun setRecognitionListener(listener: RecognitionListener) = delegate.setRecognitionListener(listener)
    override fun startListening(intent: Intent) = delegate.startListening(intent)
    override fun stopListening() = delegate.stopListening()
    override fun cancel() = delegate.cancel()
    override fun destroy() = delegate.destroy()
}

/**
 * Persian (fa-IR) speech-to-text backed by the platform [SpeechRecognizer].
 *
 * The manager owns a single recognizer instance that is created lazily, reused
 * across dictations and released via [destroy] (e.g. when the host screen
 * leaves composition). Events are delivered on the main thread.
 */
class SpeechRecognizerManager(
    context: Context,
    private val recognizerFactory: (Context) -> SpeechRecognizerLike? = { ctx ->
        SpeechRecognizer.createSpeechRecognizer(ctx)?.let { RealSpeechRecognizer(it) }
    },
    private val availability: () -> Boolean = {
        SpeechRecognizer.isRecognitionAvailable(context.applicationContext)
    },
    private val mainHandler: Handler = Handler(Looper.getMainLooper()),
) {
    private val appContext = context.applicationContext
    private var recognizer: SpeechRecognizerLike? = null
    private var listener: ((SpeechEvent) -> Unit)? = null
    private var listening = false

    fun isAvailable(): Boolean = availability()

    /** Begin dictation. Returns false (and emits [SpeechEvent.Error]) on failure. */
    fun start(language: String = PERSIAN_LANGUAGE, onEvent: (SpeechEvent) -> Unit): Boolean {
        if (listening) return true
        if (!isAvailable()) {
            post(onEvent, SpeechEvent.Error(SpeechErrorKind.NOT_AVAILABLE))
            return false
        }
        val rec = recognizer ?: recognizerFactory(appContext)?.also { recognizer = it }
        if (rec == null) {
            post(onEvent, SpeechEvent.Error(SpeechErrorKind.NOT_AVAILABLE))
            return false
        }
        this.listener = onEvent
        listening = true
        return runCatching {
            rec.setRecognitionListener(recognitionListener)
            rec.startListening(recognizerIntent(language))
            true
        }.getOrElse {
            listening = false
            this.listener = null
            post(onEvent, SpeechEvent.Error(SpeechErrorKind.FAILURE))
            false
        }
    }

    /** Stop dictation and finalize; the final [SpeechEvent.Result] still arrives. */
    fun stop() {
        listening = false
        recognizer?.stopListening()
    }

    /** Abort dictation without a result. */
    fun cancel() {
        listening = false
        recognizer?.cancel()
    }

    /** Release the recognizer. Safe to call multiple times. */
    fun destroy() {
        listening = false
        listener = null
        recognizer?.destroy()
        recognizer = null
    }

    private val recognitionListener = object : RecognitionListener {
        override fun onReadyForSpeech(params: Bundle?) = post(SpeechEvent.Listening)
        override fun onBeginningOfSpeech() {}
        override fun onRmsChanged(rmsdB: Float) {}
        override fun onBufferReceived(buffer: ByteArray?) {}
        override fun onEndOfSpeech() = post(SpeechEvent.Processing)

        override fun onError(error: Int) {
            listening = false
            post(SpeechEvent.Error(SpeechErrorKind.fromErrorCode(error)))
        }

        override fun onResults(results: Bundle?) {
            listening = false
            val text = results?.getStringArrayList(SpeechRecognizer.RESULTS_RECOGNITION)
                ?.firstOrNull().orEmpty()
            post(SpeechEvent.Result(text))
        }

        override fun onPartialResults(partialResults: Bundle?) {
            val text = partialResults?.getStringArrayList(SpeechRecognizer.RESULTS_RECOGNITION)
                ?.firstOrNull().orEmpty()
            post(SpeechEvent.Partial(text))
        }

        override fun onEvent(eventType: Int, params: Bundle?) {}
    }

    private fun post(event: SpeechEvent) {
        mainHandler.post { listener?.invoke(event) }
    }

    private fun post(onEvent: (SpeechEvent) -> Unit, event: SpeechEvent) {
        mainHandler.post { onEvent(event) }
    }

    private fun recognizerIntent(language: String): Intent =
        Intent(RecognizerIntent.ACTION_RECOGNIZE_SPEECH).apply {
            putExtra(RecognizerIntent.EXTRA_LANGUAGE_MODEL, RecognizerIntent.LANGUAGE_MODEL_FREE_FORM)
            putExtra(RecognizerIntent.EXTRA_LANGUAGE, language)
            putExtra(RecognizerIntent.EXTRA_LANGUAGE_PREFERENCE, language)
            putExtra(RecognizerIntent.EXTRA_PARTIAL_RESULTS, true)
            putExtra(RecognizerIntent.EXTRA_MAX_RESULTS, 1)
            putExtra(RecognizerIntent.EXTRA_CALLING_PACKAGE, appContext.packageName)
        }

    companion object {
        /** Persian (Farsi) language tag used for every dictation session. */
        const val PERSIAN_LANGUAGE = "fa-IR"
    }
}

/**
 * Append a recognized utterance to existing summary text without overwriting
 * it. A single space (or the existing line break) separates the parts, so
 * repeated dictations accumulate naturally.
 */
fun appendRecognizedText(current: String, recognized: String): String {
    val text = recognized.trim()
    if (text.isEmpty()) return current
    if (current.isBlank()) return text
    val separator = if (current.endsWith(" ") || current.endsWith("\n")) "" else " "
    return current + separator + text
}
