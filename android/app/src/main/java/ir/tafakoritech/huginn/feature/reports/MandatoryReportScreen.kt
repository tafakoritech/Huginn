package ir.tafakoritech.huginn.feature.reports

import android.app.TimePickerDialog
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.design.HuginnDimens
import ir.tafakoritech.huginn.core.design.PrimaryButton
import ir.tafakoritech.huginn.core.design.SecondaryButton
import ir.tafakoritech.huginn.core.design.VoiceTextField
import ir.tafakoritech.huginn.core.design.channelLabel
import ir.tafakoritech.huginn.core.design.interestLabel
import ir.tafakoritech.huginn.core.design.nextActionLabel
import ir.tafakoritech.huginn.core.design.outcomeLabel
import ir.tafakoritech.huginn.core.design.reportOutcomesInOrder
import ir.tafakoritech.huginn.core.design.interestLevelsInOrder
import ir.tafakoritech.huginn.core.design.nextActionsInOrder
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.core.util.DateTimeUtil
import ir.tafakoritech.huginn.data.model.PendingReport
import ir.tafakoritech.huginn.data.model.ReportDraft
import ir.tafakoritech.huginn.data.model.ReportOutcome
import ir.tafakoritech.huginn.data.model.ReportSubmitResponse
import ir.tafakoritech.huginn.data.model.ReportValidationError
import java.time.LocalTime

/**
 * Blocking mandatory-report overlay. Cannot be swiped, back-pressed or
 * tap-outside-dismissed. Only a successful backend submission releases it.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun MandatoryReportOverlay(
    pendingReport: PendingReport,
    checking: Boolean,
    onSubmitted: () -> Unit,
    onRecheck: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val vm: MandatoryReportViewModel = hiltViewModel()
    LaunchedEffect(pendingReport.interactionId) {
        vm.init(pendingReport)
    }
    val state by vm.state.collectAsStateWithLifecycle()

    // Back is intercepted: the report cannot be dismissed.
    BackHandler { /* intentionally empty — report is mandatory */ }

    // This blocking screen is rendered outside HuginnScaffold, so without insets
    // it would draw edge-to-edge underneath the status bar (and the bottom
    // navigation bar). safeDrawing pads the status bar on top, the navigation
    // bar / gesture insets at the bottom and any display cutouts, without
    // hardcoded values, in both gesture and 3-button navigation modes.
    Column(
        modifier = modifier
            .fillMaxSize()
            .windowInsetsPadding(WindowInsets.safeDrawing)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = HuginnDimens.space16, vertical = HuginnDimens.space16),
        verticalArrangement = Arrangement.spacedBy(HuginnDimens.space10),
    ) {
        val colors = huginnColors()

        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(
                    text = stringResource(R.string.report_title),
                    style = MaterialTheme.typography.headlineSmall,
                    color = colors.onSurface,
                    fontWeight = FontWeight.SemiBold,
                )
                Text(
                    text = state.companyName.ifBlank { pendingReport.companyName.orEmpty() },
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.secondaryText,
                )
            }
            Text(
                text = channelLabel(state.channel.ifBlank { pendingReport.channel }),
                style = MaterialTheme.typography.labelMedium,
                color = colors.primary,
            )
        }

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(colors.warningContainer, RoundedCornerShape(8.dp))
                .padding(HuginnDimens.space10),
        ) {
            Text(
                text = stringResource(R.string.error_mandatory_report),
                style = MaterialTheme.typography.bodySmall,
                color = colors.onWarningContainer,
            )
        }

        when {
            state.result != null -> ReportSuccessCard(
                result = state.result!!,
                onContinue = {
                    // Consume the one-time success event BEFORE navigating away so a
                    // re-entered overlay never re-shows the stale result screen.
                    vm.consumeResult()
                    onSubmitted()
                },
            )
            state.queuedOffline -> ReportQueuedCard(checking = checking, onRetry = vm::submit, onRecheck = onRecheck)
            else -> MandatoryReportForm(
                state = state,
                onOutcomeSelected = vm::onOutcomeSelected,
                onDraftChange = vm::setDraft,
                onSubmit = vm::submit,
            )
        }
    }
}

/**
 * Stateless mandatory-report form (UI-testable). Fields are driven by
 * [MandatoryReportUiState] and callbacks so the ViewModel stays out of tests.
 */
@Composable
fun MandatoryReportForm(
    state: MandatoryReportUiState,
    onOutcomeSelected: (String) -> Unit,
    onDraftChange: (ReportDraft) -> Unit,
    onSubmit: () -> Unit,
) {
    val colors = huginnColors()
    Column(verticalArrangement = Arrangement.spacedBy(HuginnDimens.space10)) {
        Text(
            text = stringResource(R.string.report_outcome),
            style = MaterialTheme.typography.titleSmall,
            color = colors.onSurface,
        )
        FlowRow(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            reportOutcomesInOrder.forEach { outcome ->
                FilterChip(
                    selected = state.outcome == outcome,
                    onClick = { onOutcomeSelected(outcome) },
                    label = { Text(outcomeLabel(outcome)) },
                    colors = FilterChipDefaults.filterChipColors(
                        selectedContainerColor = colors.primaryContainer,
                        selectedLabelColor = colors.onPrimaryContainer,
                    ),
                )
            }
        }

        state.validationErrors.forEach { error ->
            Text(
                text = validationMessage(error),
                style = MaterialTheme.typography.labelSmall,
                color = colors.error,
            )
        }

        OutcomeFields(state, onDraftChange)

        PrimaryButton(
            text = stringResource(R.string.report_submit),
            onClick = onSubmit,
            modifier = Modifier.fillMaxWidth().height(50.dp),
            loading = state.submitting,
        )
        Spacer(Modifier.height(24.dp))
    }
}

/** Outcome-conditional form fields, extracted so the `when` stays unambiguous. */
@Composable
private fun OutcomeFields(
    state: MandatoryReportUiState,
    onDraftChange: (ReportDraft) -> Unit,
) {
    when (state.outcome) {
        ReportOutcome.NO_ANSWER,
        ReportOutcome.BUSY,
        ReportOutcome.WRONG_NUMBER,
        -> {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                FollowUpTimeField(onDraftChange, state, R.string.report_call_again_time)
                SummaryField(state, onDraftChange)
            }
        }

        ReportOutcome.CALL_BACK_REQUESTED -> {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                FollowUpTimeField(onDraftChange, state, R.string.report_call_again_time)
                SummaryField(state, onDraftChange)
            }
        }

        ReportOutcome.NOT_RELEVANT,
        ReportOutcome.DISQUALIFIED,
        -> {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                VoiceTextField(
                    value = state.draft.mainObjection.orEmpty(),
                    onValueChange = { newValue -> onDraftChange(state.draft.copy(mainObjection = newValue)) },
                    label = stringResource(R.string.report_main_objection),
                )
                SummaryField(state, onDraftChange)
            }
        }

        ReportOutcome.ANSWERED,
        ReportOutcome.INTERESTED,
        ReportOutcome.HIGHLY_INTERESTED,
        -> AnswerDetailsForm(state, onDraftChange)

        else -> {}
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun AnswerDetailsForm(
    state: MandatoryReportUiState,
    onDraftChange: (ReportDraft) -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        YesNoSelector(
            question = stringResource(R.string.report_correct_person_reached),
            value = state.draft.correctPersonReached,
            onChange = { newValue -> onDraftChange(state.draft.copy(correctPersonReached = newValue)) },
        )
        YesNoSelector(
            question = stringResource(R.string.report_decision_maker_identified),
            value = state.draft.decisionMakerIdentified,
            onChange = { newValue -> onDraftChange(state.draft.copy(decisionMakerIdentified = newValue)) },
        )
        LevelSelector(
            title = stringResource(R.string.report_interest_level),
            options = interestLevelsInOrder,
            selected = state.draft.interestLevel,
            label = { interestLabel(it) },
            onChange = { newValue -> onDraftChange(state.draft.copy(interestLevel = newValue)) },
        )
        LevelSelector(
            title = stringResource(R.string.report_need_level),
            options = interestLevelsInOrder,
            selected = state.draft.needLevel,
            label = { interestLabel(it) },
            onChange = { newValue -> onDraftChange(state.draft.copy(needLevel = newValue)) },
        )
        VoiceTextField(
            value = state.draft.decisionMakerName.orEmpty(),
            onValueChange = { newValue -> onDraftChange(state.draft.copy(decisionMakerName = newValue)) },
            label = stringResource(R.string.report_decision_maker_name),
        )
        VoiceTextField(
            value = state.draft.decisionMakerRole.orEmpty(),
            onValueChange = { newValue -> onDraftChange(state.draft.copy(decisionMakerRole = newValue)) },
            label = stringResource(R.string.report_decision_maker_role),
        )
        VoiceTextField(
            value = state.draft.purchaseCapacity.orEmpty(),
            onValueChange = { newValue -> onDraftChange(state.draft.copy(purchaseCapacity = newValue)) },
            label = stringResource(R.string.report_purchase_capacity),
        )
        VoiceTextField(
            value = state.draft.purchaseTiming.orEmpty(),
            onValueChange = { newValue -> onDraftChange(state.draft.copy(purchaseTiming = newValue)) },
            label = stringResource(R.string.report_purchase_timing),
        )
        VoiceTextField(
            value = state.draft.mainObjection.orEmpty(),
            onValueChange = { newValue -> onDraftChange(state.draft.copy(mainObjection = newValue)) },
            label = stringResource(R.string.report_main_objection),
        )
        LevelSelector(
            title = stringResource(R.string.report_next_action),
            options = nextActionsInOrder,
            selected = state.draft.nextAction,
            label = { nextActionLabel(it) },
            onChange = { newValue -> onDraftChange(state.draft.copy(nextAction = newValue)) },
        )
        FollowUpTimeField(onDraftChange, state, R.string.report_followup_time)
        SummaryField(state, onDraftChange)
    }
}

@Composable
private fun SummaryField(
    state: MandatoryReportUiState,
    onDraftChange: (ReportDraft) -> Unit,
) {
    val label = if (state.outcome in setOf(ReportOutcome.NO_ANSWER, ReportOutcome.BUSY, ReportOutcome.WRONG_NUMBER)) {
        stringResource(R.string.report_note)
    } else {
        stringResource(R.string.report_summary)
    }
    VoiceTextField(
        value = state.draft.summary.orEmpty(),
        onValueChange = { newValue ->
            onDraftChange(state.draft.copy(summary = newValue))
        },
        label = label,
        minLines = 5,
    )
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun LevelSelector(
    title: String,
    options: List<String>,
    selected: String?,
    label: @Composable (String) -> String,
    onChange: (String) -> Unit,
) {
    val colors = huginnColors()
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleSmall,
            color = colors.onSurface,
        )
        FlowRow(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            options.forEach { option ->
                FilterChip(
                    selected = selected == option,
                    onClick = { onChange(option) },
                    label = { Text(label(option)) },
                    colors = FilterChipDefaults.filterChipColors(
                        selectedContainerColor = colors.primaryContainer,
                        selectedLabelColor = colors.onPrimaryContainer,
                    ),
                )
            }
        }
    }
}

@Composable
private fun YesNoSelector(question: String, value: Boolean?, onChange: (Boolean) -> Unit) {
    val colors = huginnColors()
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(
            text = question,
            style = MaterialTheme.typography.titleSmall,
            color = colors.onSurface,
        )
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            FilterChip(
                selected = value == true,
                onClick = { onChange(true) },
                label = { Text(stringResource(R.string.yes)) },
            )
            FilterChip(
                selected = value == false,
                onClick = { onChange(false) },
                label = { Text(stringResource(R.string.no)) },
            )
        }
    }
}

@Composable
private fun FollowUpTimeField(
    onDraftChange: (ReportDraft) -> Unit,
    state: MandatoryReportUiState,
    labelRes: Int,
) {
    val colors = huginnColors()
    val context = LocalContext.current
    val value = state.draft.followUpAt

    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(
            text = stringResource(labelRes),
            style = MaterialTheme.typography.titleSmall,
            color = colors.onSurface,
        )
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            listOf(
                "today" to stringResource(R.string.preset_today),
                "tomorrow" to stringResource(R.string.preset_tomorrow),
                "3d" to stringResource(R.string.preset_3_days),
                "week" to stringResource(R.string.preset_next_week),
            ).forEach { (key, label) ->
                FilterChip(
                    selected = presetSelected(key, value),
                    onClick = { onDraftChange(state.draft.copy(followUpAt = applyPreset(key, value))) },
                    label = { Text(label) },
                )
            }
        }
        TextButton(
            onClick = {
                val now = DateTimeUtil.nowInTehran()
                TimePickerDialog(
                    context,
                    { _, hour, minute ->
                        onDraftChange(state.draft.copy(followUpAt = combineDateAndTime(value, hour, minute)))
                    },
                    now.hour,
                    now.minute,
                    true,
                ).show()
            },
        ) {
            Text(
                text = value?.let { stringResource(R.string.time_prefix) + (DateTimeUtil.formatTime(it) ?: "") }
                    ?: stringResource(R.string.followup_time),
                color = colors.primary,
            )
        }
    }
}

private fun presetSelected(key: String, value: String?): Boolean {
    val local = DateTimeUtil.toTehranDateTime(value) ?: return false
    val today = DateTimeUtil.todayInTehran()
    return when (key) {
        "today" -> local.toLocalDate() == today
        "tomorrow" -> local.toLocalDate() == today.plusDays(1)
        "3d" -> local.toLocalDate() == today.plusDays(3)
        "week" -> local.toLocalDate() == today.plusDays(7)
        else -> false
    }
}

private fun applyPreset(key: String, current: String?): String? {
    val time = DateTimeUtil.toTehranDateTime(current)?.toLocalTime() ?: LocalTime.of(10, 0)
    val today = DateTimeUtil.todayInTehran()
    val date = when (key) {
        "today" -> today
        "tomorrow" -> today.plusDays(1)
        "3d" -> today.plusDays(3)
        "week" -> today.plusDays(7)
        else -> today
    }
    return DateTimeUtil.toIso(date, time)
}

private fun combineDateAndTime(current: String?, hour: Int, minute: Int): String? {
    val date = DateTimeUtil.toTehranDateTime(current)?.toLocalDate() ?: DateTimeUtil.todayInTehran()
    return DateTimeUtil.toIso(date, LocalTime.of(hour, minute))
}

@Composable
private fun ReportSuccessCard(
    result: ReportSubmitResponse,
    onContinue: () -> Unit,
) {
    val colors = huginnColors()
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Card(colors = CardDefaults.cardColors(containerColor = colors.successContainer)) {
            Column(Modifier.padding(HuginnDimens.space16), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(
                    text = stringResource(R.string.report_submitted),
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.onSuccessContainer,
                    fontWeight = FontWeight.SemiBold,
                )
                Text(
                    text = stringResource(R.string.report_outcome) + " " + outcomeLabel(result.outcome),
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.onSuccessContainer,
                )
                result.nextAction?.let {
                    Text(
                        text = stringResource(R.string.report_next_action_label) + " " + nextActionLabel(it),
                        style = MaterialTheme.typography.bodyMedium,
                        color = colors.onSuccessContainer,
                    )
                }
                result.followUpAt?.let {
                    Text(
                        text = stringResource(R.string.report_followup_label) + " " +
                            DateTimeUtil.formatDateTime(it),
                        style = MaterialTheme.typography.bodyMedium,
                        color = colors.onSuccessContainer,
                    )
                }
            }
        }
        PrimaryButton(
            text = stringResource(R.string.next_work),
            onClick = onContinue,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@Composable
private fun ReportQueuedCard(
    checking: Boolean,
    onRetry: () -> Unit,
    onRecheck: () -> Unit,
) {
    val colors = huginnColors()
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Card(colors = CardDefaults.cardColors(containerColor = colors.warningContainer)) {
            Column(Modifier.padding(HuginnDimens.space16), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(
                    text = stringResource(R.string.error_network),
                    style = MaterialTheme.typography.titleSmall,
                    color = colors.onWarningContainer,
                )
                Text(
                    text = stringResource(R.string.report_queued_offline),
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.onWarningContainer,
                )
            }
        }
        SecondaryButton(
            text = stringResource(R.string.action_retry),
            onClick = onRetry,
            modifier = Modifier.fillMaxWidth(),
        )
        SecondaryButton(
            text = stringResource(R.string.error_network_show_cached),
            onClick = onRecheck,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@Composable
private fun validationMessage(error: ReportValidationError): String = when (error) {
    ReportValidationError.OUTCOME_REQUIRED -> stringResource(R.string.report_outcome)
    ReportValidationError.SUMMARY_REQUIRED -> stringResource(R.string.report_summary)
    ReportValidationError.FOLLOW_UP_TIME_REQUIRED -> stringResource(R.string.report_followup_time)
    ReportValidationError.INTEREST_REQUIRED -> stringResource(R.string.report_interest_level)
    ReportValidationError.NEXT_ACTION_REQUIRED -> stringResource(R.string.report_next_action)
}
