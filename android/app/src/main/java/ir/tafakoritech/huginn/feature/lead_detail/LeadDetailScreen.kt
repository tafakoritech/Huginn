package ir.tafakoritech.huginn.feature.lead_detail

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Circle
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Message
import androidx.compose.material.icons.filled.SmartDisplay
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.Schedule
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.design.EmptyState
import ir.tafakoritech.huginn.core.design.ErrorState
import ir.tafakoritech.huginn.core.design.HuginnDimens
import ir.tafakoritech.huginn.core.design.HuginnTopBar
import ir.tafakoritech.huginn.core.design.InfoRow
import ir.tafakoritech.huginn.core.design.LeadScoreBadge
import ir.tafakoritech.huginn.core.design.LeadStatusBadge
import ir.tafakoritech.huginn.core.design.OfflineBanner
import ir.tafakoritech.huginn.core.design.PrimaryButton
import ir.tafakoritech.huginn.core.design.SectionHeader
import ir.tafakoritech.huginn.core.design.SecondaryButton
import ir.tafakoritech.huginn.core.design.TPCard
import ir.tafakoritech.huginn.core.design.VoiceInputButton
import ir.tafakoritech.huginn.core.design.VoiceInputStatus
import ir.tafakoritech.huginn.core.design.channelLabel
import ir.tafakoritech.huginn.core.design.errorMessage
import ir.tafakoritech.huginn.core.design.interestLabel
import ir.tafakoritech.huginn.core.design.nextActionLabel
import ir.tafakoritech.huginn.core.design.rememberVoiceInput
import ir.tafakoritech.huginn.core.design.scoreComponentLabel
import ir.tafakoritech.huginn.core.design.statusLabel
import ir.tafakoritech.huginn.core.design.timelineEventLabel
import ir.tafakoritech.huginn.core.speech.SpeechRecognizerManager
import ir.tafakoritech.huginn.core.speech.appendRecognizedText
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.core.util.DateTimeUtil
import ir.tafakoritech.huginn.core.util.PhoneFormatter
import ir.tafakoritech.huginn.core.util.RtlText
import ir.tafakoritech.huginn.data.model.CallScenario
import ir.tafakoritech.huginn.data.model.Contact
import ir.tafakoritech.huginn.data.model.EvidenceItem
import ir.tafakoritech.huginn.data.model.InteractionChannel
import ir.tafakoritech.huginn.data.model.LeadDetail
import ir.tafakoritech.huginn.data.model.LeadStatus
import ir.tafakoritech.huginn.data.model.TimelineEvent
import ir.tafakoritech.huginn.feature.messaging.ChannelAvailabilityEvaluator
import ir.tafakoritech.huginn.feature.followups.FollowUpDateTimeDialog
import ir.tafakoritech.huginn.feature.messaging.ChannelItem
import ir.tafakoritech.huginn.feature.messaging.CallScenarioPromptSheet
import ir.tafakoritech.huginn.feature.messaging.CallScenarioSheet
import ir.tafakoritech.huginn.feature.messaging.CommunicationSheet
import ir.tafakoritech.huginn.feature.messaging.ContactSavePromptSheet
import ir.tafakoritech.huginn.feature.messaging.SuggestedMessageSheet
import ir.tafakoritech.huginn.feature.messaging.channelIcon
import ir.tafakoritech.huginn.feature.teleprompter.TeleprompterManager

@Composable
fun LeadDetailScreen(
    vm: LeadDetailViewModel,
    onBack: () -> Unit,
    onEdit: () -> Unit,
    onQualify: (String) -> Unit,
    onDisqualify: (String) -> Unit,
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val colors = huginnColors()
    val context = LocalContext.current
    var loadedOnce by rememberSaveable { mutableStateOf(false) }

    // Reload from the backend when returning from the edit screen so the
    // updated values are shown immediately.
    LaunchedEffect(Unit) {
        if (loadedOnce) vm.reload()
        loadedOnce = true
    }

    Column(modifier = Modifier.fillMaxSize()) {
        HuginnTopBar(
            title = state.lead?.companyName ?: stringResource(R.string.detail),
            onBack = onBack,
            actions = {
                if (state.lead?.permissions?.canEdit == true) {
                    IconButton(onClick = onEdit) {
                        Icon(
                            imageVector = Icons.Default.Edit,
                            contentDescription = stringResource(R.string.edit_lead),
                            tint = colors.onSurface,
                        )
                    }
                }
            },
        )

        when {
            state.loading && state.lead == null -> {
                BoxCentered { CircularProgressIndicator(color = colors.primary) }
            }

            state.lead == null && state.error -> {
                ErrorState(
                    message = if (state.offline) stringResource(R.string.error_network) else stringResource(R.string.error_unknown),
                    onRetry = vm::reload,
                )
            }

            state.lead != null -> {
                val lead = state.lead!!
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = androidx.compose.foundation.layout.PaddingValues(HuginnDimens.space16),
                    verticalArrangement = Arrangement.spacedBy(HuginnDimens.space8),
                ) {
                    if (state.offline) {
                        item { OfflineBanner() }
                    }
                    item {
                        LeadHeader(
                            lead = lead,
                            onEdit = onEdit,
                            onQualify = { onQualify(lead.id) },
                            onDisqualify = { onDisqualify(lead.id) },
                        )
                    }
                    item {
                        Row(
                            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            PrimaryButton(
                                text = stringResource(R.string.action_call),
                                onClick = vm::startCall,
                                modifier = Modifier.weight(1f),
                                enabled = lead.permissions.canCall,
                                loading = state.callInProgress,
                            )
                            SecondaryButton(
                                text = stringResource(R.string.action_message),
                                onClick = vm::openMessageSheet,
                                modifier = Modifier.weight(1f),
                                enabled = lead.permissions.canMessage,
                            )
                        }
                    }
                    item { ContactSection(vm, lead) }
                    item { BasicSection(lead) }
                    item { BusinessSection(lead) }
                    item { LastInteractionSection(lead) }
                    item { NextFollowUpSection(lead) }
                    item { LeadFollowUpsSection(state.followUps, state.followUpsLoading, vm) }
                    item { ScoreSection(lead) }
                    item { EvidenceSection(lead) }
                    item { CallScenariosSection(vm, state.scenarios, state.loadingScenarios) }
                    item { NotesSection(vm, state.notes) }
                    item { TimelineSection(state.timeline) }
                    item { Spacer(Modifier.height(16.dp)) }
                }
            }
        }
    }

    state.notice?.let { key ->
        val text = noticeText(key)
        if (text != null) {
            Row(
                modifier = Modifier.fillMaxWidth()
                    .background(colors.warningContainer, RoundedCornerShape(8.dp))
                    .padding(HuginnDimens.space12),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = text,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.onWarningContainer,
                    modifier = Modifier.weight(1f),
                )
                TextButton(onClick = vm::consumeNotice) {
                    Text(stringResource(R.string.action_cancel), color = colors.onWarningContainer)
                }
            }
            androidx.compose.runtime.LaunchedEffect(key) {
                kotlinx.coroutines.delay(5000)
                vm.consumeNotice()
            }
        }
    }

    if (state.messageSheetVisible) {
        val sheetLead = state.lead
        if (sheetLead != null) {
            val contact = vm.selectedContact()
            val channels = buildChannelItems(sheetLead, contact, vm::messengerInstalled)
            CommunicationSheet(
                contactName = contact?.displayName?.ifBlank { null } ?: sheetLead.companyName,
                channels = channels,
                onSelect = vm::onSelectChannel,
                onDismiss = vm::dismissMessageSheet,
            )
        }
    }

    state.suggestedSheet?.let { draft ->
        SuggestedMessageSheet(
            channel = draft.channel,
            loading = state.loadingSuggested,
            text = draft.text,
            subject = draft.subject,
            hasError = state.suggestError,
            onTextChange = { vm.updateSuggestedText(it) },
            onRetry = vm::retrySuggested,
            onOpenMessenger = { edited -> vm.openMessenger(draft, edited) },
            onCancel = vm::dismissSuggestedSheet,
        )
    }

    state.contactSaveChannel?.let { channel ->
        ContactSavePromptSheet(
            channel = channel,
            onSaveContact = { vm.saveContactFor(channel) },
            onDismiss = vm::dismissContactSave,
        )
    }

    if (state.callPromptVisible) {
        val sheetLead = state.lead
        if (sheetLead != null) {
            CallScenarioPromptSheet(
                contactName = vm.selectedContact()?.displayName?.ifBlank { null } ?: sheetLead.companyName,
                onGenerate = vm::requestCallScenario,
                onCallNow = vm::proceedWithoutScenario,
                onDismiss = vm::dismissCallPrompt,
            )
        }
    }

    if (state.overlayPermissionDialogVisible) {
        AlertDialog(
            onDismissRequest = vm::dismissOverlayPermissionDialog,
            title = { Text(stringResource(R.string.teleprompter_permission_title)) },
            text = { Text(stringResource(R.string.teleprompter_permission_desc)) },
            confirmButton = {
                Button(
                    onClick = {
                        context.startActivity(TeleprompterManager.createOverlayPermissionIntent(context))
                        vm.dismissOverlayPermissionDialog()
                    },
                ) {
                    Text(stringResource(R.string.teleprompter_permission_grant))
                }
            },
            dismissButton = {
                TextButton(onClick = vm::proceedCallWithoutTeleprompter) {
                    Text(stringResource(R.string.teleprompter_permission_skip))
                }
            },
        )
    }

    state.deleteScenarioTarget?.let { target ->
        AlertDialog(
            onDismissRequest = vm::dismissDeleteScenario,
            title = { Text(stringResource(R.string.action_delete)) },
            text = { Text(stringResource(R.string.scenario_delete_confirm)) },
            confirmButton = {
                Button(
                    onClick = vm::confirmDeleteScenario,
                ) {
                    Text(stringResource(R.string.action_delete))
                }
            },
            dismissButton = {
                TextButton(onClick = vm::dismissDeleteScenario) {
                    Text(stringResource(R.string.action_cancel))
                }
            },
        )
    }

    state.scenario.let { ui ->
        if (ui.visible || ui.loading || ui.error || ui.scenario != null) {
            CallScenarioSheet(
                loading = ui.loading,
                scenario = ui.scenario,
                hasError = ui.error,
                isSaving = ui.saving,
                hasSaveError = ui.saveError,
                startInEditMode = ui.startInEditMode,
                onRetry = vm::retryCallScenario,
                onSave = vm::saveCallScenario,
                onStartCall = { vm.startCallWithScenario(ui.scenario) },
                onOpenTeleprompter = { vm.openTeleprompter(ui.scenario) },
                onCancel = vm::dismissCallScenario,
            )
        }
    }
}

@Composable
private fun BoxCentered(content: @Composable () -> Unit) {
    androidx.compose.foundation.layout.Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) { content() }
}

@Composable
private fun buildChannelItems(
    lead: LeadDetail,
    contact: Contact?,
    messengerInstalled: (String) -> Boolean,
): List<ChannelItem> {
    val channels = listOf(
        InteractionChannel.CALL,
        InteractionChannel.SMS,
        InteractionChannel.WHATSAPP,
        InteractionChannel.TELEGRAM,
        InteractionChannel.BALE,
        InteractionChannel.EITAA,
        InteractionChannel.RUBIKA,
        InteractionChannel.EMAIL,
    )
    return channels.map { channel ->
        val status = ChannelAvailabilityEvaluator.evaluate(
            channel = channel,
            contact = contact,
            permissions = lead.permissions,
            messengerInstalled = when (channel) {
                InteractionChannel.WHATSAPP,
                InteractionChannel.TELEGRAM,
                InteractionChannel.BALE,
                -> messengerInstalled(channel)
                else -> true
            },
        )
        val reasonRes = when (status.availability) {
            ir.tafakoritech.huginn.feature.messaging.ChannelAvailability.DISABLED_NO_NUMBER -> R.string.channel_no_number
            ir.tafakoritech.huginn.feature.messaging.ChannelAvailability.DISABLED_NO_EMAIL -> R.string.channel_no_email
            ir.tafakoritech.huginn.feature.messaging.ChannelAvailability.DISABLED_NO_TARGET -> R.string.channel_no_target
            ir.tafakoritech.huginn.feature.messaging.ChannelAvailability.DISABLED_NOT_INSTALLED -> R.string.channel_not_available
            ir.tafakoritech.huginn.feature.messaging.ChannelAvailability.DISABLED_FORBIDDEN -> R.string.error_forbidden
            else -> null
        }
        ChannelItem(
            channel = channel,
            icon = channelIcon(channel),
            enabled = status.availability == ir.tafakoritech.huginn.feature.messaging.ChannelAvailability.ENABLED ||
                status.availability == ir.tafakoritech.huginn.feature.messaging.ChannelAvailability.CONTACT_SAVE_REQUIRED,
            disabledReasonRes = reasonRes,
        )
    }
}

@Composable
private fun LeadHeader(
    lead: LeadDetail,
    onEdit: () -> Unit,
    onQualify: () -> Unit,
    onDisqualify: () -> Unit,
) {
    val colors = huginnColors()
    TPCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(
                    text = lead.companyName,
                    style = MaterialTheme.typography.titleLarge,
                    color = colors.onSurface,
                    fontWeight = FontWeight.SemiBold,
                )
                Spacer(Modifier.height(2.dp))
                Text(
                    text = listOfNotNull(lead.city, lead.province).joinToString(" — ").ifBlank { null } ?: "—",
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.mutedText,
                )
            }
            Column(horizontalAlignment = Alignment.End) {
                LeadScoreBadge(score = lead.score)
                Spacer(Modifier.height(4.dp))
                LeadStatusBadge(status = lead.status)
            }
        }
        Spacer(Modifier.height(10.dp))
        InfoRow(stringResource(R.string.owner), lead.ownerName ?: lead.ownerUserId)
        InfoRow(stringResource(R.string.business_type), lead.businessType)
        InfoRow(stringResource(R.string.industry), lead.industry)

        if (lead.status == LeadStatus.QUALIFIED && lead.permissions.canExportToMuninn) {
            Spacer(Modifier.height(10.dp))
            Row(
                modifier = Modifier.fillMaxWidth().background(colors.successContainer, RoundedCornerShape(8.dp)).padding(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(Icons.Default.Star, contentDescription = null, tint = colors.success)
                Spacer(Modifier.size(8.dp))
                Text(
                    text = stringResource(R.string.ready_for_muninn),
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.onSuccessContainer,
                )
            }
        }

        if (lead.permissions.canEdit) {
            Spacer(Modifier.height(10.dp))
            SecondaryButton(
                text = stringResource(R.string.edit_lead),
                onClick = onEdit,
                modifier = Modifier.fillMaxWidth(),
            )
        }

        if (lead.permissions.canQualify || lead.permissions.canDisqualify) {
            Spacer(Modifier.height(10.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                if (lead.permissions.canQualify) {
                    TextButton(onClick = onQualify, modifier = Modifier.weight(1f)) {
                        Text(stringResource(R.string.qualify_lead), color = colors.success)
                    }
                }
                if (lead.permissions.canDisqualify) {
                    TextButton(onClick = onDisqualify, modifier = Modifier.weight(1f)) {
                        Text(stringResource(R.string.disqualify_title), color = colors.error)
                    }
                }
            }
        }
    }
}

@Composable
private fun ContactSection(vm: LeadDetailViewModel, lead: LeadDetail) {
    Column {
        SectionHeader(stringResource(R.string.section_contacts))
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            lead.contacts.forEach { contact ->
                val selected = contact.id == vm.selectedContact()?.id
                val colors = huginnColors()
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable { vm.selectContact(contact) }
                        .background(
                            if (selected) colors.activeCard else colors.card,
                            RoundedCornerShape(10.dp),
                        )
                        .padding(HuginnDimens.space12),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column(Modifier.weight(1f)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                text = contact.displayName.ifBlank { stringResource(R.string.unknown) },
                                style = MaterialTheme.typography.bodyLarge,
                                color = colors.onSurface,
                                fontWeight = if (contact.isPrimaryContact) FontWeight.SemiBold else FontWeight.Normal,
                            )
                            if (contact.isPrimaryContact) {
                                Spacer(Modifier.size(6.dp))
                                Text(
                                    text = "• " + stringResource(R.string.primary_contact),
                                    style = MaterialTheme.typography.labelSmall,
                                    color = colors.primary,
                                )
                            }
                            if (contact.isDecisionMaker) {
                                Spacer(Modifier.size(6.dp))
                                Text(
                                    text = "• " + stringResource(R.string.decision_maker),
                                    style = MaterialTheme.typography.labelSmall,
                                    color = colors.warning,
                                )
                            }
                        }
                        contact.jobTitle?.let {
                            Text(
                                text = it,
                                style = MaterialTheme.typography.bodySmall,
                                color = colors.secondaryText,
                            )
                        }
                        (contact.primaryNumber ?: contact.email)?.let {
                            Text(
                                text = RtlText.ltrEmbedding(PhoneFormatter.toDisplay(it) ?: it),
                                style = MaterialTheme.typography.bodySmall,
                                color = colors.mutedText,
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun BasicSection(lead: LeadDetail) {
    Column {
        SectionHeader(stringResource(R.string.section_basic))
        TPCard {
            InfoRow(stringResource(R.string.status), statusLabel(lead.status), valueIsLtr = false)
            InfoRow(stringResource(R.string.city), lead.city)
            lead.address?.let { InfoRow(stringResource(R.string.unknown), it) }
            lead.website?.let { InfoRow(stringResource(R.string.unknown), RtlText.ltrEmbedding(it)) }
            lead.tags.takeIf { it.isNotEmpty() }?.let {
                InfoRow(stringResource(R.string.tag), it.joinToString(stringResource(R.string.tag_separator)), valueIsLtr = false)
            }
        }
    }
}

@Composable
private fun BusinessSection(lead: LeadDetail) {
    if (lead.businessType == null && lead.industry == null && lead.description == null) return
    Column {
        SectionHeader(stringResource(R.string.section_business))
        TPCard {
            InfoRow(stringResource(R.string.business_type), lead.businessType)
            InfoRow(stringResource(R.string.industry), lead.industry)
            lead.description?.let {
                Text(
                    text = it,
                    style = MaterialTheme.typography.bodyMedium,
                    color = huginnColors().secondaryText,
                    modifier = Modifier.padding(top = 6.dp),
                )
            }
        }
    }
}

@Composable
private fun LastInteractionSection(lead: LeadDetail) {
    val interaction = lead.lastInteraction ?: return
    val colors = huginnColors()
    Column {
        SectionHeader(stringResource(R.string.section_last_interaction))
        TPCard {
            InfoRow(stringResource(R.string.channel_call), channelLabel(interaction.channel), valueIsLtr = false)
            InfoRow(stringResource(R.string.detail), DateTimeUtil.formatDateTime(interaction.startedAt), valueIsLtr = false)
            interaction.contactName?.let { InfoRow(stringResource(R.string.primary_contact), it, valueIsLtr = false) }
            interaction.outcome?.let {
                InfoRow(stringResource(R.string.report_outcome), statusLabel(it), valueIsLtr = false)
            }
            interaction.suggestedMessage?.takeIf { it.isNotBlank() }?.let {
                Text(
                    text = it,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.secondaryText,
                    modifier = Modifier.padding(top = 6.dp),
                )
            }
        }
    }
}

@Composable
private fun NextFollowUpSection(lead: LeadDetail) {
    val followUp = lead.nextFollowup ?: return
    val colors = huginnColors()
    Column {
        SectionHeader(stringResource(R.string.section_next_followup))
        TPCard {
            InfoRow(
                stringResource(R.string.followup_time),
                DateTimeUtil.formatDateTime(followUp.dueAt),
                valueIsLtr = false,
            )
            followUp.description?.let {
                Text(
                    text = it,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.secondaryText,
                    modifier = Modifier.padding(top = 6.dp),
                )
            }
        }
    }
}

/** Lead-scoped follow-ups with create / complete / reschedule / cancel. */
@Composable
private fun LeadFollowUpsSection(
    followUps: List<ir.tafakoritech.huginn.data.model.FollowUp>,
    loading: Boolean,
    vm: LeadDetailViewModel,
) {
    val colors = huginnColors()
    var showCreate by rememberSaveable { mutableStateOf(false) }
    var rescheduleTarget by rememberSaveable { mutableStateOf<ir.tafakoritech.huginn.data.model.FollowUp?>(null) }

    Column {
        SectionHeader(stringResource(R.string.section_followups))
        SecondaryButton(
            text = stringResource(R.string.followup_create_title),
            onClick = { showCreate = true },
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        when {
            loading -> {
                Box(
                    modifier = Modifier.fillMaxWidth().padding(vertical = 16.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator(modifier = Modifier.size(24.dp), strokeWidth = 2.dp, color = colors.primary)
                }
            }
            followUps.isEmpty() -> {
                Text(
                    text = stringResource(R.string.empty),
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.mutedText,
                )
            }
            else -> {
                followUps.forEach { fu ->
                    TPCard {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Column(Modifier.weight(1f)) {
                                Text(
                                    text = fu.title?.takeIf { it.isNotBlank() }
                                        ?: fu.type,
                                    style = MaterialTheme.typography.titleSmall,
                                    color = colors.onSurface,
                                    fontWeight = FontWeight.Medium,
                                )
                                Text(
                                    text = DateTimeUtil.toLocalDigits(DateTimeUtil.formatDateTime(fu.dueAt) ?: ""),
                                    style = MaterialTheme.typography.labelSmall,
                                    color = if (fu.isOverdue) colors.error else colors.secondaryText,
                                )
                                fu.description?.takeIf { it.isNotBlank() }?.let {
                                    Text(
                                        text = it,
                                        style = MaterialTheme.typography.bodySmall,
                                        color = colors.mutedText,
                                    )
                                }
                            }
                            if (fu.status == ir.tafakoritech.huginn.data.model.FollowUpStatus.PENDING) {
                                IconButton(onClick = { vm.completeFollowUp(fu.id) }) {
                                    Icon(
                                        imageVector = Icons.Default.Check,
                                        contentDescription = stringResource(R.string.action_done),
                                        tint = colors.success,
                                    )
                                }
                                IconButton(onClick = { rescheduleTarget = fu }) {
                                    Icon(
                                        imageVector = Icons.Outlined.Schedule,
                                        contentDescription = stringResource(R.string.action_reschedule),
                                        tint = colors.onSurfaceVariant,
                                    )
                                }
                                IconButton(onClick = { vm.cancelFollowUp(fu.id) }) {
                                    Icon(
                                        imageVector = Icons.Outlined.Delete,
                                        contentDescription = stringResource(R.string.action_cancel),
                                        tint = colors.mutedText,
                                    )
                                }
                            } else {
                                Text(
                                    text = stringResource(R.string.action_reopen),
                                    style = MaterialTheme.typography.labelMedium,
                                    color = colors.primary,
                                    modifier = Modifier
                                        .clickable { vm.reopenFollowUp(fu.id) }
                                        .padding(8.dp),
                                )
                            }
                        }
                    }
                    Spacer(Modifier.height(8.dp))
                }
            }
        }
    }

    if (showCreate) {
        FollowUpDateTimeDialog(
            onPick = { iso -> vm.createFollowUp(iso, description = null) },
            onDismiss = { showCreate = false },
        )
    }
    rescheduleTarget?.let { target ->
        FollowUpDateTimeDialog(
            onPick = { iso ->
                vm.rescheduleFollowUp(target.id, iso)
                rescheduleTarget = null
            },
            onDismiss = { rescheduleTarget = null },
        )
    }
}

@Composable
private fun ScoreSection(lead: LeadDetail) {
    val colors = huginnColors()
    val components = lead.scoreExplanation.ifEmpty {
        lead.scoreBreakdown?.components?.map { (k, v) ->
            ir.tafakoritech.huginn.data.model.ScoreComponent(key = k, value = v, max = 25)
        }.orEmpty()
    }
    Column {
        SectionHeader(stringResource(R.string.section_score))
        TPCard {
            Text(
                text = stringResource(R.string.score_full, lead.score),
                style = MaterialTheme.typography.titleMedium,
                color = colors.onSurface,
                fontWeight = FontWeight.SemiBold,
            )
            Spacer(Modifier.height(8.dp))
            components.forEach { component ->
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = scoreComponentLabel(component.key, fallback = component.label),
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.secondaryText,
                        modifier = Modifier.weight(1f),
                    )
                    Text(
                        text = DateTimeUtil.toLocalDigits(component.value.toString()),
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.onSurface,
                        fontWeight = FontWeight.Medium,
                    )
                }
                Spacer(Modifier.height(2.dp))
            }
        }
    }
}

@Composable
private fun EvidenceSection(lead: LeadDetail) {
    if (lead.evidence.isEmpty()) return
    val colors = huginnColors()
    Column {
        SectionHeader(stringResource(R.string.section_evidence))
        lead.evidence.forEach { item ->
            TPCard {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) {
                        Text(
                            text = item.fieldName,
                            style = MaterialTheme.typography.labelMedium,
                            color = colors.primary,
                            fontWeight = FontWeight.SemiBold,
                        )
                        item.sourceExcerpt?.let {
                            Spacer(Modifier.height(2.dp))
                            Text(
                                text = it,
                                style = MaterialTheme.typography.bodySmall,
                                color = colors.secondaryText,
                            )
                        }
                        Spacer(Modifier.height(4.dp))
                        Text(
                            text = stringResource(R.string.evidence_confidence) + ": " +
                                DateTimeUtil.toLocalDigits(((item.confidence * 100).toInt()).toString()) + "%",
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.mutedText,
                        )
                        (item.sourceDocument ?: "—").let {
                            Text(
                                text = stringResource(R.string.evidence_source) + ": " + it,
                                style = MaterialTheme.typography.labelSmall,
                                color = colors.mutedText,
                            )
                        }
                    }
                }
            }
            Spacer(Modifier.height(8.dp))
        }
    }
}

@Composable
private fun CallScenariosSection(
    vm: LeadDetailViewModel,
    scenarios: List<CallScenario>,
    loading: Boolean,
) {
    val colors = huginnColors()
    Column {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            SectionHeader(stringResource(R.string.section_call_scenarios))
            TextButton(onClick = vm::createNewScenario) {
                Icon(
                    imageVector = Icons.Default.Add,
                    contentDescription = null,
                    tint = colors.primary,
                    modifier = Modifier.size(16.dp),
                )
                Spacer(Modifier.size(4.dp))
                Text(
                    text = stringResource(R.string.scenario_new),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.primary,
                )
            }
        }
        Spacer(Modifier.height(4.dp))

        if (loading && scenarios.isEmpty()) {
            CircularProgressIndicator(
                modifier = Modifier.size(24.dp).align(Alignment.CenterHorizontally),
                strokeWidth = 2.dp,
                color = colors.primary,
            )
        } else if (scenarios.isEmpty()) {
            Text(
                text = stringResource(R.string.scenario_empty),
                style = MaterialTheme.typography.bodySmall,
                color = colors.mutedText,
            )
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                scenarios.forEach { sc ->
                    TPCard {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            val badgeText = when {
                                sc.generationStatus == "degraded" || sc.fallbackUsed ->
                                    stringResource(R.string.scenario_source_fallback)
                                sc.isGenerated ->
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
                            val dateStr = sc.updatedAt ?: sc.createdAt
                            if (!dateStr.isNullOrBlank()) {
                                Text(
                                    text = DateTimeUtil.formatRelative(dateStr),
                                    style = MaterialTheme.typography.labelSmall,
                                    color = colors.mutedText,
                                )
                            }
                        }
                        Spacer(Modifier.height(6.dp))
                        val previewText = sc.openingSentence.ifBlank { sc.reasonForCall.ifBlank { "—" } }
                        Text(
                            text = previewText,
                            style = MaterialTheme.typography.bodyMedium,
                            color = colors.onSurface,
                            fontWeight = FontWeight.Normal,
                            maxLines = 2,
                        )
                        Spacer(Modifier.height(10.dp))
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(6.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            TextButton(
                                onClick = { vm.openScenario(sc, editMode = false) },
                                modifier = Modifier.weight(1f),
                            ) {
                                Icon(Icons.Default.Visibility, contentDescription = null, modifier = Modifier.size(16.dp))
                                Spacer(Modifier.size(4.dp))
                                Text(stringResource(R.string.action_view), style = MaterialTheme.typography.labelSmall)
                            }
                            TextButton(
                                onClick = { vm.openScenario(sc, editMode = true) },
                                modifier = Modifier.weight(1f),
                            ) {
                                Icon(Icons.Default.Edit, contentDescription = null, modifier = Modifier.size(16.dp))
                                Spacer(Modifier.size(4.dp))
                                Text(stringResource(R.string.action_edit), style = MaterialTheme.typography.labelSmall)
                            }
                            TextButton(
                                onClick = { vm.openTeleprompter(sc) },
                                modifier = Modifier.weight(1.2f),
                            ) {
                                Icon(Icons.Default.SmartDisplay, contentDescription = null, modifier = Modifier.size(16.dp))
                                Spacer(Modifier.size(4.dp))
                                Text(stringResource(R.string.teleprompter_mode), style = MaterialTheme.typography.labelSmall)
                            }
                            IconButton(
                                onClick = { vm.deleteCallScenario(sc) },
                                modifier = Modifier.size(32.dp),
                            ) {
                                Icon(
                                    Icons.Default.Delete,
                                    contentDescription = stringResource(R.string.action_delete),
                                    tint = colors.error,
                                    modifier = Modifier.size(18.dp),
                                )
                            }
                        }
                        Spacer(Modifier.height(4.dp))
                        PrimaryButton(
                            text = stringResource(R.string.start_call),
                            onClick = { vm.startCallWithScenario(sc) },
                            modifier = Modifier.fillMaxWidth().height(40.dp),
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun NotesSection(vm: LeadDetailViewModel, notes: List<ir.tafakoritech.huginn.data.model.Note>) {
    val colors = huginnColors()
    Column {
        SectionHeader(stringResource(R.string.section_notes))
        NoteComposer(onAddNote = vm::addNote)
        Spacer(Modifier.height(8.dp))
        if (notes.isEmpty()) {
            Text(
                text = stringResource(R.string.notes_empty),
                style = MaterialTheme.typography.bodySmall,
                color = colors.mutedText,
            )
        } else {
            notes.forEach { note ->
                TPCard {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            text = note.authorName ?: "—",
                            style = MaterialTheme.typography.labelMedium,
                            color = colors.primary,
                            fontWeight = FontWeight.Medium,
                            modifier = Modifier.weight(1f),
                        )
                        Text(
                            text = DateTimeUtil.formatRelative(note.createdAt),
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.mutedText,
                        )
                    }
                    Spacer(Modifier.height(4.dp))
                    Text(
                        text = note.text,
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.onSurface,
                    )
                }
                Spacer(Modifier.height(8.dp))
            }
        }
    }
}

/**
 * Add-note row: text field, voice-typing mic button and submit button. Tapping
 * the mic starts platform speech-to-text in the current app language (Persian
 * by default); the recognized utterance is appended to any existing typed text
 * so nothing is ever overwritten. Permission, availability, timeout and
 * no-match failures are surfaced inline and a second tap while listening stops
 * dictation and commits the partial result.
 */
@Composable
fun NoteComposer(
    onAddNote: (String) -> Unit,
    modifier: Modifier = Modifier,
    speechManager: SpeechRecognizerManager? = null,
) {
    val colors = huginnColors()
    var newNote by rememberSaveable { mutableStateOf("") }
    val voice = rememberVoiceInput(
        onResult = { recognized -> newNote = appendRecognizedText(newNote, recognized) },
        speechManager = speechManager,
    )

    Column(modifier = modifier) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            OutlinedTextField(
                value = newNote,
                onValueChange = { newNote = it },
                modifier = Modifier.weight(1f).testTag(NoteComposerFieldTag),
                placeholder = { Text(stringResource(R.string.notes_add)) },
                shape = RoundedCornerShape(10.dp),
            )
            VoiceInputButton(
                voiceState = voice.state,
                onClick = voice.onMicClick,
                tintColor = colors.onSurfaceVariant,
            )
            IconButton(onClick = {
                if (newNote.isNotBlank()) {
                    onAddNote(newNote)
                    newNote = ""
                    voice.reset()
                }
            }) {
                Icon(Icons.Default.Message, contentDescription = stringResource(R.string.notes_add), tint = colors.primary)
            }
        }
        VoiceInputStatus(
            voiceState = voice.state,
            partialText = voice.partialText,
        )
    }
}

/** Test tag for the note text field inside [NoteComposer]. */
const val NoteComposerFieldTag = "note_composer_field"

@Composable
private fun TimelineSection(events: List<TimelineEvent>) {
    val colors = huginnColors()
    Column {
        SectionHeader(stringResource(R.string.section_timeline))
        if (events.isEmpty()) {
            EmptyState(title = stringResource(R.string.empty))
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(0.dp)) {
                events.forEach { event ->
                    Row(modifier = Modifier.fillMaxWidth().padding(vertical = 6.dp)) {
                        Icon(
                            imageVector = Icons.Default.Circle,
                            contentDescription = null,
                            tint = colors.primary,
                            modifier = Modifier.size(10.dp).padding(top = 4.dp),
                        )
                        Spacer(Modifier.size(8.dp))
                        Column(Modifier.weight(1f)) {
                            Text(
                                text = timelineEventLabel(event.eventType, fallback = event.title),
                                style = MaterialTheme.typography.bodyMedium,
                                color = colors.onSurface,
                            )
                            event.description?.let {
                                Text(
                                    text = it,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = colors.secondaryText,
                                )
                            }
                            Text(
                                text = DateTimeUtil.formatRelative(event.createdAt),
                                style = MaterialTheme.typography.labelSmall,
                                color = colors.mutedText,
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun noticeText(key: String): String? = when (key) {
    "no_number" -> stringResource(R.string.error_no_number)
    "no_email" -> stringResource(R.string.channel_no_email)
    "no_target" -> stringResource(R.string.channel_no_target)
    "not_installed" -> stringResource(R.string.channel_not_available)
    "contact_saved" -> stringResource(R.string.contact_saved)
    "call_failed" -> stringResource(R.string.channel_not_available)
    "channel_unavailable" -> stringResource(R.string.channel_not_available)
    "conflict" -> stringResource(R.string.error_conflict)
    "forbidden" -> stringResource(R.string.error_forbidden)
    "network" -> stringResource(R.string.error_network)
    "scenario_saved" -> stringResource(R.string.scenario_saved)
    "scenario_save_failed" -> stringResource(R.string.scenario_save_failed)
    "scenario_deleted" -> stringResource(R.string.scenario_deleted)
    "scenario_delete_failed" -> stringResource(R.string.scenario_delete_failed)
    "error" -> stringResource(R.string.error_unknown)
    else -> null
}
