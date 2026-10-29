package ir.tafakoritech.huginn.core.design

import androidx.annotation.StringRes
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.data.model.InteractionChannel
import ir.tafakoritech.huginn.data.model.InteractionObjective
import ir.tafakoritech.huginn.data.model.LeadStatus
import ir.tafakoritech.huginn.data.model.NextAction
import ir.tafakoritech.huginn.data.model.ReportOutcome

/** Code -> string resource mappings for display. Unknown values => "نامشخص". */
object UiMappers {

    @StringRes
    fun statusLabelRes(status: String?): Int = when (status) {
        LeadStatus.NEW -> R.string.status_new
        LeadStatus.READY -> R.string.status_ready
        LeadStatus.ASSIGNED -> R.string.status_assigned
        LeadStatus.CONTACTING -> R.string.status_contacting
        LeadStatus.FOLLOW_UP -> R.string.status_followup
        LeadStatus.QUALIFIED -> R.string.status_qualified
        LeadStatus.DISQUALIFIED -> R.string.status_disqualified
        LeadStatus.EXPORTED_TO_MUNINN -> R.string.status_exported
        LeadStatus.ARCHIVED -> R.string.status_archived
        else -> R.string.unknown
    }

    @StringRes
    fun channelLabelRes(channel: String?): Int = when (channel) {
        InteractionChannel.CALL -> R.string.channel_call
        InteractionChannel.SMS -> R.string.channel_sms
        InteractionChannel.WHATSAPP -> R.string.channel_whatsapp
        InteractionChannel.TELEGRAM -> R.string.channel_telegram
        InteractionChannel.BALE -> R.string.channel_bale
        InteractionChannel.EITAA -> R.string.channel_eitaa
        InteractionChannel.RUBIKA -> R.string.channel_rubika
        InteractionChannel.EMAIL -> R.string.channel_email
        else -> R.string.unknown
    }

    @StringRes
    fun objectiveLabelRes(objective: String?): Int = when (objective) {
        InteractionObjective.FIRST_CONTACT -> R.string.action_call
        InteractionObjective.FOLLOW_UP -> R.string.next_followup
        InteractionObjective.QUALIFICATION -> R.string.qualify_title
        InteractionObjective.INFO_REQUEST -> R.string.next_action_send_info
        else -> R.string.unknown
    }

    @StringRes
    fun nextActionLabelRes(action: String?): Int = when (action) {
        NextAction.CALL_AGAIN -> R.string.next_action_call_again
        NextAction.SEND_INFORMATION -> R.string.next_action_send_info
        NextAction.SEND_SAMPLE -> R.string.next_action_send_sample
        NextAction.FOLLOW_UP -> R.string.next_action_follow_up
        NextAction.QUALIFY -> R.string.next_action_qualify
        NextAction.DISQUALIFY -> R.string.next_action_disqualify
        NextAction.TRANSFER_TO_MUNINN -> R.string.next_action_transfer
        NextAction.NONE -> R.string.next_action_none
        // Status-derived next actions served by the backend work queue / lead
        // workflow (backend httpapi.nextActionFor). Mapped explicitly so a
        // valid backend value is never degraded to "نامشخص".
        "ASSIGN" -> R.string.next_action_assign
        "CALL" -> R.string.next_action_call
        "REPORT" -> R.string.next_action_report
        "EXPORT" -> R.string.next_action_export
        else -> R.string.unknown
    }

    @StringRes
    fun interestLabelRes(level: String?): Int = when (level) {
        "NONE" -> R.string.interest_none
        "LOW" -> R.string.interest_low
        "MEDIUM" -> R.string.interest_medium
        "HIGH" -> R.string.interest_high
        else -> R.string.unknown
    }

    @StringRes
    fun outcomeLabelRes(outcome: String?): Int = when (outcome) {
        ReportOutcome.NO_ANSWER -> R.string.outcome_no_answer
        ReportOutcome.BUSY -> R.string.outcome_busy
        ReportOutcome.WRONG_NUMBER -> R.string.outcome_wrong_number
        ReportOutcome.ANSWERED -> R.string.outcome_answered
        ReportOutcome.CALL_BACK_REQUESTED -> R.string.outcome_callback_requested
        ReportOutcome.NOT_RELEVANT -> R.string.outcome_not_relevant
        ReportOutcome.INTERESTED -> R.string.outcome_interested
        ReportOutcome.HIGHLY_INTERESTED -> R.string.outcome_highly_interested
        ReportOutcome.DISQUALIFIED -> R.string.outcome_disqualified
        else -> R.string.unknown
    }

    @StringRes
    fun disqualifyReasonLabelRes(reason: String?): Int = when (reason) {
        "NOT_RELEVANT" -> R.string.reason_not_relevant
        "INVALID_CONTACT" -> R.string.reason_invalid_contact
        "TOO_SMALL" -> R.string.reason_too_small
        "NO_INTEREST" -> R.string.reason_no_interest
        "NO_NEED" -> R.string.reason_no_need
        "NO_BUDGET" -> R.string.reason_no_budget
        "DUPLICATE" -> R.string.reason_duplicate
        "BUSINESS_CLOSED" -> R.string.reason_closed
        "OTHER" -> R.string.reason_other
        else -> R.string.unknown
    }

    /**
     * Maps a backend timeline `event_type` (a stable machine key such as
     * "lead.created") to a localized label. Returns null for unknown types so
     * callers can fall back to the raw title instead of showing the key.
     */
    @StringRes
    fun timelineEventLabelRes(eventType: String?): Int? = when (eventType) {
        "lead.created" -> R.string.timeline_lead_created
        "lead.updated" -> R.string.timeline_lead_updated
        "lead.assigned" -> R.string.timeline_lead_assigned
        "lead.qualified" -> R.string.timeline_lead_qualified
        "lead.disqualified" -> R.string.timeline_lead_disqualified
        "lead.exported" -> R.string.timeline_lead_exported
        "lead.merged" -> R.string.timeline_lead_merged
        "lead.tag_added" -> R.string.timeline_lead_tag_added
        "lead.tag_removed" -> R.string.timeline_lead_tag_removed
        "lead.project.attached" -> R.string.timeline_lead_project_attached
        "lead.project.detached" -> R.string.timeline_lead_project_detached
        "note.created" -> R.string.timeline_note_created
        "followup.created" -> R.string.timeline_followup_created
        "report.submitted" -> R.string.timeline_report_submitted
        "report.voided" -> R.string.timeline_report_voided
        "interaction.created" -> R.string.timeline_interaction_created
        "contact.created" -> R.string.timeline_contact_created
        "contact.updated" -> R.string.timeline_contact_updated
        "contact.deleted" -> R.string.timeline_contact_deleted
        "call_scenario.generated" -> R.string.timeline_call_scenario_generated
        "call_scenario.created" -> R.string.timeline_call_scenario_created
        "call_scenario.updated" -> R.string.timeline_call_scenario_updated
        "call_scenario.deleted" -> R.string.timeline_call_scenario_deleted
        "project.updated" -> R.string.timeline_project_updated
        else -> null
    }

    /**
     * Maps a backend score component key (e.g. "business_fit") to a localized
     * label. Returns null for unknown keys so callers can fall back safely.
     */
    @StringRes
    fun scoreComponentLabelRes(key: String?): Int? = when (key) {
        "business_fit" -> R.string.score_business_fit
        "contactability" -> R.string.score_contactability
        "scale" -> R.string.score_scale
        "engagement" -> R.string.score_engagement
        "authority" -> R.string.score_authority
        "need" -> R.string.score_need
        "timing" -> R.string.score_timing
        else -> null
    }
}

@Composable
fun statusLabel(status: String?): String = stringResource(UiMappers.statusLabelRes(status))

@Composable
fun statusColor(status: String?): Color {
    val c = huginnColors()
    return when (status) {
        LeadStatus.QUALIFIED -> c.success
        LeadStatus.DISQUALIFIED -> c.error
        LeadStatus.EXPORTED_TO_MUNINN -> c.info
        LeadStatus.ARCHIVED -> c.neutral
        LeadStatus.FOLLOW_UP -> c.warning
        else -> c.primary
    }
}

@Composable
fun channelLabel(channel: String?): String = stringResource(UiMappers.channelLabelRes(channel))

@Composable
fun nextActionLabel(action: String?): String = stringResource(UiMappers.nextActionLabelRes(action))

@Composable
fun interestLabel(level: String?): String = stringResource(UiMappers.interestLabelRes(level))

@Composable
fun outcomeLabel(outcome: String?): String = stringResource(UiMappers.outcomeLabelRes(outcome))

@Composable
fun disqualifyReasonLabel(reason: String?): String = stringResource(UiMappers.disqualifyReasonLabelRes(reason))

/**
 * Localized timeline event label. Falls back to [fallback] (typically the
 * backend-provided title) when the event type is unknown, and finally to a
 * generic label, so raw machine keys are never shown.
 */
@Composable
fun timelineEventLabel(eventType: String?, fallback: String? = null): String {
    val res = UiMappers.timelineEventLabelRes(eventType)
    if (res == null && !eventType.isNullOrBlank()) {
        androidx.compose.runtime.remember(eventType) {
            ir.tafakoritech.huginn.core.util.SafeLog.d("Unknown timeline event type: $eventType")
        }
    }
    return res
        ?.let { stringResource(it) }
        ?: fallback?.takeIf { it.isNotBlank() }
        ?: stringResource(R.string.unknown)
}

/**
 * Localized score component label. Falls back to [fallback] (the backend
 * `name`) when the key is unknown, and finally to a generic label.
 */
@Composable
fun scoreComponentLabel(key: String?, fallback: String? = null): String {
    val res = UiMappers.scoreComponentLabelRes(key)
    if (res == null && !key.isNullOrBlank()) {
        androidx.compose.runtime.remember(key) {
            ir.tafakoritech.huginn.core.util.SafeLog.d("Unknown score component key: $key")
        }
    }
    return res
        ?.let { stringResource(it) }
        ?: fallback?.takeIf { it.isNotBlank() }
        ?: stringResource(R.string.unknown)
}

/** All report outcomes in display order for the outcome selector. */
val reportOutcomesInOrder: List<String> = listOf(
    ReportOutcome.NO_ANSWER,
    ReportOutcome.BUSY,
    ReportOutcome.WRONG_NUMBER,
    ReportOutcome.ANSWERED,
    ReportOutcome.CALL_BACK_REQUESTED,
    ReportOutcome.NOT_RELEVANT,
    ReportOutcome.INTERESTED,
    ReportOutcome.HIGHLY_INTERESTED,
    ReportOutcome.DISQUALIFIED,
)

val interestLevelsInOrder: List<String> = listOf("NONE", "LOW", "MEDIUM", "HIGH")

val nextActionsInOrder: List<String> = listOf(
    NextAction.CALL_AGAIN,
    NextAction.SEND_INFORMATION,
    NextAction.SEND_SAMPLE,
    NextAction.FOLLOW_UP,
    NextAction.QUALIFY,
    NextAction.DISQUALIFY,
    NextAction.TRANSFER_TO_MUNINN,
    NextAction.NONE,
)

val disqualifyReasonsInOrder: List<String> = listOf(
    "NOT_RELEVANT",
    "INVALID_CONTACT",
    "TOO_SMALL",
    "NO_INTEREST",
    "NO_NEED",
    "NO_BUDGET",
    "DUPLICATE",
    "BUSINESS_CLOSED",
    "OTHER",
)
