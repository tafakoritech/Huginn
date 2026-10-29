package ir.tafakoritech.huginn.core.speech

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/** UI phase of a voice-input session, derived from [SpeechEvent]s. */
sealed interface VoiceInputState {
    data object Idle : VoiceInputState
    data object Listening : VoiceInputState
    data object Processing : VoiceInputState
    data class Error(val kind: SpeechErrorKind) : VoiceInputState
}

/**
 * Holds the voice-input UI state for a text field and turns [SpeechEvent]s from
 * [SpeechRecognizerManager] into stable state transitions. The final recognized
 * utterance is handed to [onResult] so the host field can append it to its
 * current text (see [appendRecognizedText]); the controller never mutates the
 * field's text itself, which keeps it independent of any screen's draft model.
 */
class VoiceInputController(
    private val onResult: (String) -> Unit = {},
) {
    var state by mutableStateOf<VoiceInputState>(VoiceInputState.Idle)
        private set

    /** Live partial text while listening (may be blank). */
    var partialText by mutableStateOf("")
        private set

    /** Mark the beginning of a dictation attempt, before the recognizer starts. */
    fun begin() {
        state = VoiceInputState.Listening
        partialText = ""
    }

    /** Feed a recognizer event into the state machine. */
    fun onEvent(event: SpeechEvent) {
        when (event) {
            SpeechEvent.Listening -> {
                state = VoiceInputState.Listening
                partialText = ""
            }
            SpeechEvent.Processing -> state = VoiceInputState.Processing
            is SpeechEvent.Partial -> {
                state = VoiceInputState.Listening
                partialText = event.text
            }
            is SpeechEvent.Result -> {
                state = VoiceInputState.Idle
                partialText = ""
                if (event.text.isNotBlank()) onResult(event.text)
            }
            is SpeechEvent.Error -> {
                state = VoiceInputState.Error(event.kind)
                partialText = ""
            }
        }
    }

    /** The user denied the microphone permission. */
    fun onPermissionDenied() {
        state = VoiceInputState.Error(SpeechErrorKind.PERMISSION_DENIED)
    }

    /** Abort any active session (e.g. the host leaves composition). */
    fun reset() {
        state = VoiceInputState.Idle
        partialText = ""
    }
}
