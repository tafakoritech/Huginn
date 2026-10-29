package ir.tafakoritech.huginn.core.design

import android.Manifest
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.Stop
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.permissions.PermissionsManager
import ir.tafakoritech.huginn.core.speech.SpeechErrorKind
import ir.tafakoritech.huginn.core.speech.SpeechEvent
import ir.tafakoritech.huginn.core.speech.SpeechRecognizerManager
import ir.tafakoritech.huginn.core.speech.VoiceInputController
import ir.tafakoritech.huginn.core.speech.VoiceInputState
import ir.tafakoritech.huginn.core.speech.appendRecognizedText
import ir.tafakoritech.huginn.core.speech.speechErrorLabelRes
import ir.tafakoritech.huginn.core.speech.speechLanguageTag
import ir.tafakoritech.huginn.core.theme.huginnColors

/** Handlers surfaced by [rememberVoiceInput] so a field can render the mic. */
data class VoiceInputHandlers(
    val state: VoiceInputState,
    val partialText: String,
    val onMicClick: () -> Unit,
    val reset: () -> Unit,
)

/**
 * Wires platform speech-to-text into a text field: owns the recognizer, the
 * runtime microphone permission and the [VoiceInputController]. [onResult]
 * receives each final utterance so the host field can append it to its current
 * text (see [appendRecognizedText]). The recognizer is released when the host
 * leaves composition. Pass a [speechManager] to inject a fake in tests.
 */
@Composable
fun rememberVoiceInput(
    onResult: (String) -> Unit,
    speechManager: SpeechRecognizerManager? = null,
): VoiceInputHandlers {
    val context = LocalContext.current
    val manager = speechManager ?: remember(context) { SpeechRecognizerManager(context.applicationContext) }
    val permissionsManager = remember(context) { PermissionsManager(context.applicationContext) }
    val controller = remember { VoiceInputController(onResult = onResult) }

    DisposableEffect(manager) {
        onDispose { manager.destroy() }
    }

    val onSpeechEvent: (SpeechEvent) -> Unit = { controller.onEvent(it) }

    val startRecognition: () -> Unit = {
        controller.begin()
        manager.start(speechLanguageTag(context), onSpeechEvent)
    }

    val permissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted ->
        if (granted) startRecognition()
        else controller.onPermissionDenied()
    }

    val onMicClick: () -> Unit = {
        when {
            !manager.isAvailable() ->
                controller.onEvent(SpeechEvent.Error(SpeechErrorKind.NOT_AVAILABLE))
            !permissionsManager.isRecordAudioGranted() ->
                permissionLauncher.launch(Manifest.permission.RECORD_AUDIO)
            controller.state == VoiceInputState.Listening -> manager.stop()
            else -> startRecognition()
        }
    }

    return VoiceInputHandlers(
        state = controller.state,
        partialText = controller.partialText,
        onMicClick = onMicClick,
        reset = { controller.reset() },
    )
}

/**
 * Labeled [OutlinedTextField] with a voice-typing mic button beside it and an
 * inline status line below. Recognized utterances are appended to the current
 * [value], never overwriting it. Reusable across every free-text report field.
 */
@Composable
fun VoiceTextField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    modifier: Modifier = Modifier,
    minLines: Int = 1,
    speechManager: SpeechRecognizerManager? = null,
) {
    val colors = huginnColors()
    val currentValue by rememberUpdatedState(value)
    val currentOnValueChange by rememberUpdatedState(onValueChange)
    val voice = rememberVoiceInput(
        onResult = { recognized ->
            currentOnValueChange(appendRecognizedText(currentValue, recognized))
        },
        speechManager = speechManager,
    )

    Column(modifier = modifier) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            OutlinedTextField(
                value = value,
                onValueChange = onValueChange,
                label = { Text(label) },
                modifier = Modifier.weight(1f),
                shape = RoundedCornerShape(10.dp),
                minLines = minLines,
            )
            VoiceInputButton(
                voiceState = voice.state,
                onClick = voice.onMicClick,
                tintColor = colors.onSurfaceVariant,
            )
        }
        VoiceInputStatus(
            voiceState = voice.state,
            partialText = voice.partialText,
        )
    }
}

/**
 * Mic icon button that reflects the current [VoiceInputState]: an idle mic, a
 * stop button while listening, a progress spinner while processing, and an
 * error-tinted mic after a failure so a fresh tap can retry.
 */
@Composable
fun VoiceInputButton(
    voiceState: VoiceInputState,
    onClick: () -> Unit,
    tintColor: Color,
) {
    val colors = huginnColors()
    IconButton(onClick = onClick) {
        when (voiceState) {
            VoiceInputState.Idle -> Icon(
                imageVector = Icons.Default.Mic,
                contentDescription = stringResource(R.string.voice_input),
                tint = tintColor,
            )
            VoiceInputState.Listening -> Icon(
                imageVector = Icons.Default.Stop,
                contentDescription = stringResource(R.string.voice_stop),
                tint = colors.error,
            )
            VoiceInputState.Processing -> CircularProgressIndicator(
                modifier = Modifier.size(20.dp),
                strokeWidth = 2.dp,
                color = colors.primary,
            )
            is VoiceInputState.Error -> Icon(
                imageVector = Icons.Default.Mic,
                contentDescription = stringResource(R.string.voice_input),
                tint = colors.error,
            )
        }
    }
}

/** Status/hint line shown under a voice-enabled field. */
@Composable
fun VoiceInputStatus(
    voiceState: VoiceInputState,
    partialText: String,
) {
    val colors = huginnColors()
    val text: String
    val color: Color
    when (voiceState) {
        VoiceInputState.Idle -> {
            text = stringResource(R.string.voice_input_hint)
            color = colors.mutedText
        }
        VoiceInputState.Listening -> {
            text = partialText.ifBlank { stringResource(R.string.voice_listening) }
            color = colors.primary
        }
        VoiceInputState.Processing -> {
            text = stringResource(R.string.voice_processing)
            color = colors.mutedText
        }
        is VoiceInputState.Error -> {
            text = stringResource(speechErrorLabelRes(voiceState.kind))
            color = colors.error
        }
    }
    Text(
        text = text,
        style = MaterialTheme.typography.labelSmall,
        color = color,
    )
}
