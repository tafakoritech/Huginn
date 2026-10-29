package ir.tafakoritech.huginn.feature.messaging

import androidx.annotation.StringRes
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Chat
import androidx.compose.material.icons.filled.Circle
import androidx.compose.material.icons.filled.Contacts
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.Forum
import androidx.compose.material.icons.filled.Message
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.Saver
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.design.HuginnDimens
import ir.tafakoritech.huginn.core.design.PrimaryButton
import ir.tafakoritech.huginn.core.design.SecondaryButton
import ir.tafakoritech.huginn.core.design.channelLabel
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.core.util.copyToClipboard
import ir.tafakoritech.huginn.data.model.CallScenario
import ir.tafakoritech.huginn.data.model.InteractionChannel
import kotlinx.coroutines.delay

/** A selectable channel entry for the communication sheet. */
data class ChannelItem(
    val channel: String,
    val icon: ImageVector,
    val enabled: Boolean,
    @StringRes val disabledReasonRes: Int? = null,
)

fun channelIcon(channel: String): ImageVector = when (channel) {
    InteractionChannel.CALL -> Icons.Default.Call
    InteractionChannel.SMS -> Icons.Default.Message
    InteractionChannel.WHATSAPP -> Icons.Default.Chat
    InteractionChannel.TELEGRAM -> Icons.AutoMirrored.Filled.Send
    InteractionChannel.BALE -> Icons.Default.Circle
    InteractionChannel.EITAA -> Icons.Default.Share
    InteractionChannel.RUBIKA -> Icons.Default.Forum
    InteractionChannel.EMAIL -> Icons.Default.Email
    else -> Icons.Default.Message
}

/** Complete suggested message: optional subject followed by a blank line and the body. */
fun fullSuggestedMessage(subject: String?, body: String): String =
    (subject?.let { "$it\n\n" } ?: "") + body

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CommunicationSheet(
    contactName: String,
    channels: List<ChannelItem>,
    onSelect: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    val colors = huginnColors()
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = HuginnDimens.space16)
                .padding(bottom = HuginnDimens.space24),
            verticalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            Text(
                text = stringResource(R.string.action_message) + " — " + contactName,
                style = MaterialTheme.typography.titleMedium,
                color = colors.onSurface,
                fontWeight = FontWeight.SemiBold,
            )
            Spacer(Modifier.height(4.dp))
            channels.forEach { item ->
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable(enabled = item.enabled) { onSelect(item.channel) }
                        .padding(vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Icon(
                        imageVector = item.icon,
                        contentDescription = null,
                        tint = if (item.enabled) colors.onSurface else colors.mutedText,
                        modifier = Modifier.size(22.dp),
                    )
                    Column(Modifier.weight(1f)) {
                        Text(
                            text = channelLabel(item.channel),
                            style = MaterialTheme.typography.bodyLarge,
                            color = if (item.enabled) colors.onSurface else colors.mutedText,
                        )
                        item.disabledReasonRes?.let { reasonRes ->
                            Text(
                                text = stringResource(reasonRes),
                                style = MaterialTheme.typography.labelSmall,
                                color = colors.mutedText,
                            )
                        }
                    }
                }
            }
            Spacer(Modifier.height(8.dp))
            TextButton(
                onClick = onDismiss,
                modifier = Modifier.align(Alignment.End),
            ) {
                Text(stringResource(R.string.action_cancel))
            }
        }
    }
}

/**
 * Suggested message composer. Opens immediately with a visible progress state
 * while the backend generates the text, auto-populates the field when it
 * arrives, keeps the composer open with a retry on failure, and lets the user
 * edit the text before handing it to the external messenger.
 *
 * The message text is only ever sent from the external messenger app by the
 * employee pressing Send there.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SuggestedMessageSheet(
    channel: String,
    loading: Boolean,
    text: String,
    subject: String?,
    hasError: Boolean = false,
    onTextChange: (String) -> Unit,
    onRetry: () -> Unit,
    onOpenMessenger: (String) -> Unit,
    onCancel: () -> Unit,
) {
    val colors = huginnColors()
    val context = LocalContext.current
    var edited by rememberSaveable { mutableStateOf(text) }
    var editing by rememberSaveable { mutableStateOf(false) }
    var copied by remember { mutableStateOf(false) }

    // Auto-populate the message field as soon as the suggested text arrives.
    LaunchedEffect(text) { edited = text }

    // Brief inline confirmation after a copy, then reset so it can be copied again.
    LaunchedEffect(copied) {
        if (copied) {
            delay(2000)
            copied = false
        }
    }

    val fullMessage = fullSuggestedMessage(subject, edited)
    val onCopy: () -> Unit = {
        context.copyToClipboard(fullMessage)
        copied = true
    }

    ModalBottomSheet(onDismissRequest = onCancel) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = HuginnDimens.space16)
                .padding(bottom = HuginnDimens.space24)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text(
                text = stringResource(R.string.suggested_message_title),
                style = MaterialTheme.typography.titleMedium,
                color = colors.onSurface,
                fontWeight = FontWeight.SemiBold,
            )
            when {
                loading -> {
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(20.dp),
                            strokeWidth = 2.dp,
                            color = colors.primary,
                        )
                        Text(
                            text = stringResource(R.string.loading_suggested_message),
                            style = MaterialTheme.typography.bodyMedium,
                            color = colors.mutedText,
                        )
                    }
                }
                hasError -> {
                    Text(
                        text = stringResource(R.string.suggest_failed),
                        style = MaterialTheme.typography.bodyMedium,
                        color = colors.error,
                    )
                    SecondaryButton(
                        text = stringResource(R.string.action_retry),
                        onClick = onRetry,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                else -> {
                    if (!editing) {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            verticalAlignment = Alignment.Top,
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            Text(
                                text = fullMessage,
                                style = MaterialTheme.typography.bodyMedium,
                                color = colors.onSurface,
                                modifier = Modifier.weight(1f),
                            )
                            IconButton(
                                onClick = onCopy,
                                enabled = fullMessage.isNotBlank(),
                            ) {
                                Icon(
                                    imageVector = Icons.Default.ContentCopy,
                                    contentDescription = stringResource(R.string.action_copy),
                                    tint = colors.onSurfaceVariant,
                                )
                            }
                        }
                        if (copied) {
                            Text(
                                text = stringResource(R.string.message_copied),
                                style = MaterialTheme.typography.labelSmall,
                                color = colors.success,
                                modifier = Modifier.align(Alignment.End),
                            )
                        }
                    } else {
                        if (subject != null) {
                            OutlinedTextField(
                                value = subject,
                                onValueChange = {},
                                enabled = false,
                                label = { Text(stringResource(R.string.next_action_send_info)) },
                                modifier = Modifier.fillMaxWidth(),
                                shape = RoundedCornerShape(10.dp),
                            )
                        }
                        OutlinedTextField(
                            value = edited,
                            onValueChange = { edited = it; onTextChange(it) },
                            modifier = Modifier.fillMaxWidth().height(160.dp),
                            shape = RoundedCornerShape(10.dp),
                        )
                    }
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        SecondaryButton(
                            text = stringResource(R.string.action_edit),
                            onClick = { editing = !editing },
                            modifier = Modifier.weight(1f),
                        )
                        PrimaryButton(
                            text = stringResource(R.string.open_messenger),
                            onClick = { onOpenMessenger(edited) },
                            modifier = Modifier.weight(1f),
                        )
                    }
                }
            }
            TextButton(
                onClick = onCancel,
                modifier = Modifier.align(Alignment.End),
            ) {
                Text(stringResource(R.string.action_cancel))
            }
        }
    }
}

/** Prompt shown when a messenger username is missing (Eitaa/Rubika/Telegram). */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ContactSavePromptSheet(
    channel: String,
    onSaveContact: () -> Unit,
    onDismiss: () -> Unit,
) {
    val colors = huginnColors()
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = HuginnDimens.space16)
                .padding(bottom = HuginnDimens.space24),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Icon(
                imageVector = Icons.Default.Contacts,
                contentDescription = null,
                tint = colors.primary,
                modifier = Modifier.size(32.dp),
            )
            Text(
                text = stringResource(R.string.contact_save_required),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.onSurface,
            )
            Text(
                text = stringResource(R.string.contact_save_hint),
                style = MaterialTheme.typography.labelSmall,
                color = colors.mutedText,
            )
            PrimaryButton(
                text = stringResource(R.string.save_to_contacts),
                onClick = onSaveContact,
                modifier = Modifier.fillMaxWidth(),
            )
            TextButton(
                onClick = onDismiss,
                modifier = Modifier.align(Alignment.End),
            ) {
                Text(stringResource(R.string.action_cancel))
            }
        }
    }
}

/**
 * Pre-call prompt: "Do you need a suggested call scenario?" with Yes/No.
 * "Yes" opens the scenario screen; "No" launches the normal call flow.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CallScenarioPromptSheet(
    contactName: String,
    onGenerate: () -> Unit,
    onCallNow: () -> Unit,
    onDismiss: () -> Unit,
) {
    val colors = huginnColors()
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = HuginnDimens.space16)
                .padding(bottom = HuginnDimens.space24),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Icon(
                imageVector = Icons.Default.Call,
                contentDescription = null,
                tint = colors.primary,
                modifier = Modifier.size(32.dp),
            )
            Text(
                text = stringResource(R.string.call_scenario_title),
                style = MaterialTheme.typography.titleMedium,
                color = colors.onSurface,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                text = stringResource(R.string.call_scenario_ask, contactName),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.onSurface,
            )
            PrimaryButton(
                text = stringResource(R.string.call_scenario_generate),
                onClick = onGenerate,
                modifier = Modifier.fillMaxWidth(),
            )
            SecondaryButton(
                text = stringResource(R.string.call_scenario_call_now),
                onClick = onCallNow,
                modifier = Modifier.fillMaxWidth(),
            )
            TextButton(
                onClick = onDismiss,
                modifier = Modifier.align(Alignment.End),
            ) {
                Text(stringResource(R.string.action_cancel))
            }
        }
    }
}

/**
 * Saver for [CallScenario] so the editable copy survives configuration changes.
 */
private val callScenarioSaver = Saver<CallScenario, List<String>>(
    save = { s ->
        listOf(
            s.id.orEmpty(),
            s.leadId.orEmpty(),
            s.contactId.orEmpty(),
            s.userId.orEmpty(),
            s.openingSentence,
            s.introduction,
            s.reasonForCall,
            s.valueProposition,
            s.discoveryQuestions.joinToString(SCENARIO_LIST_SEPARATOR),
            s.likelyObjections.joinToString(SCENARIO_LIST_SEPARATOR),
            s.objectionResponses.joinToString(SCENARIO_LIST_SEPARATOR),
            s.recommendedNextStep,
            s.closingLine,
            s.isGenerated.toString(),
            s.createdAt.orEmpty(),
            s.updatedAt.orEmpty(),
        )
    },
    restore = { saved ->
        if (saved.size < 13) {
            CallScenario()
        } else {
            CallScenario(
                id = saved[0].ifBlank { null },
                leadId = saved[1].ifBlank { null },
                contactId = saved[2].ifBlank { null },
                userId = saved[3].ifBlank { null },
                openingSentence = saved[4],
                introduction = saved[5],
                reasonForCall = saved[6],
                valueProposition = saved[7],
                discoveryQuestions = saved[8].split(SCENARIO_LIST_SEPARATOR).filter { it.isNotBlank() },
                likelyObjections = saved[9].split(SCENARIO_LIST_SEPARATOR).filter { it.isNotBlank() },
                objectionResponses = saved[10].split(SCENARIO_LIST_SEPARATOR).filter { it.isNotBlank() },
                recommendedNextStep = saved[11],
                closingLine = saved[12],
                isGenerated = saved.getOrNull(13)?.toBooleanStrictOrNull() ?: false,
                createdAt = saved.getOrNull(14)?.ifBlank { null },
                updatedAt = saved.getOrNull(15)?.ifBlank { null },
            )
        }
    },
)

private const val SCENARIO_LIST_SEPARATOR = "\u0000|"

/**
 * Suggested call scenario screen. Shows a loading state while the backend
 * generates or loads it, keeps the screen open with a retry on failure, allows
 * full editing with persistent saving to the backend, and provides action buttons
 * for starting a call with teleprompter or opening teleprompter mode.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CallScenarioSheet(
    loading: Boolean,
    scenario: CallScenario?,
    hasError: Boolean = false,
    isSaving: Boolean = false,
    hasSaveError: Boolean = false,
    startInEditMode: Boolean = false,
    onRetry: () -> Unit,
    onSave: (CallScenario) -> Unit = {},
    onStartCall: () -> Unit,
    onOpenTeleprompter: (() -> Unit)? = null,
    onCancel: () -> Unit,
) {
    val colors = huginnColors()
    var editing by rememberSaveable { mutableStateOf(startInEditMode) }
    var edited by rememberSaveable(stateSaver = callScenarioSaver) { mutableStateOf(scenario ?: CallScenario()) }

    // A "degraded" scenario is the deterministic fallback the backend returns
    // when the LLM failed. It must NEVER be presented as a successful
    // personalized generation — the UI shows an explicit failure state with a
    // retry instead of the fallback content masquerading as a real scenario.
    val isDegraded = scenario != null && (scenario.generationStatus == "degraded" || scenario.fallbackUsed)

    // Sync the editable copy whenever a freshly generated or reloaded scenario arrives.
    LaunchedEffect(scenario) {
        if (scenario != null && (!editing || edited.id == null || edited.id != scenario.id)) {
            edited = scenario
        }
    }

    ModalBottomSheet(onDismissRequest = onCancel) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = HuginnDimens.space16)
                .padding(bottom = HuginnDimens.space24)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = stringResource(R.string.call_scenario_screen_title),
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.onSurface,
                    fontWeight = FontWeight.SemiBold,
                )
                if (scenario != null && !loading && !editing) {
                    val badgeText = when {
                        scenario.generationStatus == "degraded" || scenario.fallbackUsed ->
                            stringResource(R.string.scenario_source_fallback)
                        scenario.isGenerated ->
                            stringResource(R.string.scenario_source_ai)
                        else ->
                            stringResource(R.string.scenario_source_manual)
                    }
                    Text(
                        text = badgeText,
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.primary,
                        fontWeight = FontWeight.Medium,
                    )
                }
            }

            when {
                loading -> {
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(20.dp),
                            strokeWidth = 2.dp,
                            color = colors.primary,
                        )
                        Text(
                            text = stringResource(R.string.loading_call_scenario),
                            style = MaterialTheme.typography.bodyMedium,
                            color = colors.mutedText,
                        )
                    }
                }
                hasError -> {
                    Text(
                        text = stringResource(R.string.scenario_failed),
                        style = MaterialTheme.typography.bodyMedium,
                        color = colors.error,
                    )
                    SecondaryButton(
                        text = stringResource(R.string.action_retry),
                        onClick = onRetry,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                isDegraded -> {
                    Text(
                        text = stringResource(R.string.scenario_generation_failed),
                        style = MaterialTheme.typography.bodyMedium,
                        color = colors.error,
                    )
                    val detail = scenario?.generationError
                    if (!detail.isNullOrBlank()) {
                        Text(
                            text = stringResource(R.string.scenario_generation_failed_detail, detail),
                            style = MaterialTheme.typography.bodySmall,
                            color = colors.mutedText,
                        )
                    }
                    SecondaryButton(
                        text = stringResource(R.string.action_retry),
                        onClick = onRetry,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                scenario != null || edited.openingSentence.isNotBlank() || editing -> {
                    val currentScenario = scenario ?: edited

                    if (editing) {
                        ScenarioBodyEditable(
                            scenario = edited,
                            onField = { edited = it },
                        )

                        if (hasSaveError) {
                            Text(
                                text = stringResource(R.string.scenario_save_failed),
                                style = MaterialTheme.typography.bodySmall,
                                color = colors.error,
                            )
                        }

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            SecondaryButton(
                                text = stringResource(R.string.action_cancel),
                                onClick = {
                                    if (scenario != null) {
                                        edited = scenario
                                        editing = false
                                    } else {
                                        onCancel()
                                    }
                                },
                                modifier = Modifier.weight(1f),
                            )
                            PrimaryButton(
                                text = if (isSaving) stringResource(R.string.scenario_saving) else stringResource(R.string.action_save),
                                onClick = { onSave(edited) },
                                modifier = Modifier.weight(1f),
                                enabled = !isSaving,
                            )
                        }
                    } else {
                        ScenarioBody(currentScenario)

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            SecondaryButton(
                                text = stringResource(R.string.action_edit),
                                onClick = { editing = true },
                                modifier = Modifier.weight(1f),
                            )
                            if (onOpenTeleprompter != null) {
                                SecondaryButton(
                                    text = stringResource(R.string.teleprompter_mode),
                                    onClick = onOpenTeleprompter,
                                    modifier = Modifier.weight(1f),
                                )
                            }
                        }
                        Spacer(Modifier.height(4.dp))
                        PrimaryButton(
                            text = stringResource(R.string.start_call),
                            onClick = onStartCall,
                            modifier = Modifier.fillMaxWidth().height(48.dp),
                        )
                    }
                }
            }

            TextButton(
                onClick = onCancel,
                modifier = Modifier.align(Alignment.End),
            ) {
                Text(stringResource(R.string.action_cancel))
            }
        }
    }
}

@Composable
private fun ScenarioBody(scenario: CallScenario) {
    val colors = huginnColors()
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        ScenarioField(stringResource(R.string.scenario_opening), scenario.openingSentence)
        ScenarioField(stringResource(R.string.scenario_introduction), scenario.introduction)
        ScenarioField(stringResource(R.string.scenario_reason), scenario.reasonForCall)
        ScenarioField(stringResource(R.string.scenario_value), scenario.valueProposition)
        scenario.discoveryQuestions.takeIf { it.isNotEmpty() }?.let {
            ScenarioList(stringResource(R.string.scenario_questions), it)
        }
        scenario.likelyObjections.takeIf { it.isNotEmpty() }?.let {
            ScenarioList(stringResource(R.string.scenario_objections), it)
        }
        scenario.objectionResponses.takeIf { it.isNotEmpty() }?.let {
            ScenarioList(stringResource(R.string.scenario_responses), it)
        }
        ScenarioField(stringResource(R.string.scenario_next_step), scenario.recommendedNextStep)
        ScenarioField(stringResource(R.string.scenario_closing), scenario.closingLine)
    }
}

@Composable
private fun ScenarioBodyEditable(
    scenario: CallScenario,
    onField: (CallScenario) -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        ScenarioEditField(stringResource(R.string.scenario_opening), scenario.openingSentence) {
            onField(scenario.copy(openingSentence = it))
        }
        ScenarioEditField(stringResource(R.string.scenario_introduction), scenario.introduction) {
            onField(scenario.copy(introduction = it))
        }
        ScenarioEditField(stringResource(R.string.scenario_reason), scenario.reasonForCall) {
            onField(scenario.copy(reasonForCall = it))
        }
        ScenarioEditField(stringResource(R.string.scenario_value), scenario.valueProposition) {
            onField(scenario.copy(valueProposition = it))
        }
        scenario.discoveryQuestions.forEachIndexed { i, q ->
            ScenarioEditField(stringResource(R.string.scenario_question_n, i + 1), q) {
                onField(scenario.copy(discoveryQuestions = scenario.discoveryQuestions.toMutableList().apply { this[i] = it }))
            }
        }
        TextButton(
            onClick = {
                onField(scenario.copy(discoveryQuestions = scenario.discoveryQuestions + ""))
            },
            modifier = Modifier.align(Alignment.Start),
        ) {
            Text(stringResource(R.string.scenario_add_question))
        }

        scenario.likelyObjections.forEachIndexed { i, o ->
            ScenarioEditField(stringResource(R.string.scenario_objection_n, i + 1), o) {
                onField(scenario.copy(likelyObjections = scenario.likelyObjections.toMutableList().apply { this[i] = it }))
            }
            val resp = scenario.objectionResponses.getOrNull(i).orEmpty()
            ScenarioEditField(stringResource(R.string.scenario_response_n, i + 1), resp) {
                val list = scenario.objectionResponses.toMutableList()
                while (list.size <= i) list.add("")
                list[i] = it
                onField(scenario.copy(objectionResponses = list))
            }
        }
        TextButton(
            onClick = {
                onField(scenario.copy(
                    likelyObjections = scenario.likelyObjections + "",
                    objectionResponses = scenario.objectionResponses + "",
                ))
            },
            modifier = Modifier.align(Alignment.Start),
        ) {
            Text(stringResource(R.string.scenario_add_objection))
        }
        ScenarioEditField(stringResource(R.string.scenario_next_step), scenario.recommendedNextStep) {
            onField(scenario.copy(recommendedNextStep = it))
        }
        ScenarioEditField(stringResource(R.string.scenario_closing), scenario.closingLine) {
            onField(scenario.copy(closingLine = it))
        }
    }
}

@Composable
private fun ScenarioEditField(label: String, value: String, onChange: (String) -> Unit) {
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        label = { Text(label) },
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(10.dp),
        minLines = 1,
    )
}

@Composable
private fun ScenarioField(label: String, value: String) {
    if (value.isBlank()) return
    val colors = huginnColors()
    Column {
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            color = colors.primary,
            fontWeight = FontWeight.SemiBold,
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            color = colors.onSurface,
        )
    }
}

@Composable
private fun ScenarioList(label: String, items: List<String>) {
    if (items.isEmpty()) return
    val colors = huginnColors()
    Column {
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            color = colors.primary,
            fontWeight = FontWeight.SemiBold,
        )
        Spacer(Modifier.height(2.dp))
        items.forEach { item ->
            Row(modifier = Modifier.fillMaxWidth().padding(vertical = 2.dp)) {
                Text(
                    text = "•",
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.mutedText,
                )
                Spacer(Modifier.size(6.dp))
                Text(
                    text = item,
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.onSurface,
                )
            }
        }
    }
}
