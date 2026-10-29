package ir.tafakoritech.huginn.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Contact report outcomes (backend reports.Outcome). */
object ReportOutcome {
    const val NO_ANSWER = "NO_ANSWER"
    const val BUSY = "BUSY"
    const val WRONG_NUMBER = "WRONG_NUMBER"
    const val ANSWERED = "ANSWERED"
    const val CALL_BACK_REQUESTED = "CALL_BACK_REQUESTED"
    const val NOT_RELEVANT = "NOT_RELEVANT"
    const val INTERESTED = "INTERESTED"
    const val HIGHLY_INTERESTED = "HIGHLY_INTERESTED"
    const val DISQUALIFIED = "DISQUALIFIED"
}

/** Interest / need levels (backend reports.Interest). */
object InterestLevel {
    const val NONE = "NONE"
    const val LOW = "LOW"
    const val MEDIUM = "MEDIUM"
    const val HIGH = "HIGH"
}

/** Recommended next actions (backend reports.NextAction). */
object NextAction {
    const val CALL_AGAIN = "CALL_AGAIN"
    const val SEND_INFORMATION = "SEND_INFORMATION"
    const val SEND_SAMPLE = "SEND_SAMPLE"
    const val FOLLOW_UP = "FOLLOW_UP"
    const val QUALIFY = "QUALIFY"
    const val DISQUALIFY = "DISQUALIFY"
    const val TRANSFER_TO_MUNINN = "TRANSFER_TO_MUNINN"
    const val NONE = "NONE"
}

/** Follow-up statuses (backend followups.Status). */
object FollowUpStatus {
    const val PENDING = "PENDING"
    const val COMPLETED = "COMPLETED"
    const val CANCELLED = "CANCELLED"
    const val SNOOZED = "SNOOZED"
}

/** Follow-up priorities (backend followups.Priority). */
object FollowUpPriority {
    const val LOW = "LOW"
    const val MEDIUM = "MEDIUM"
    const val HIGH = "HIGH"
}

@Serializable
data class FollowUp(
    val id: String,
    @SerialName("lead_id") val leadId: String,
    @SerialName("contact_id") val contactId: String? = null,
    @SerialName("project_id") val projectId: String? = null,
    @SerialName("campaign_id") val campaignId: String? = null,
    @SerialName("interaction_id") val interactionId: String? = null,
    @SerialName("assigned_to") val assignedTo: String? = null,
    @SerialName("created_by") val createdBy: String? = null,
    val title: String? = null,
    val description: String? = null,
    val channel: String? = null,
    val type: String = "FOLLOW_UP",
    val priority: String = FollowUpPriority.MEDIUM,
    val status: String = FollowUpStatus.PENDING,
    @SerialName("due_at") val dueAt: String,
    @SerialName("completed_at") val completedAt: String? = null,
    @SerialName("snoozed_until") val snoozedUntil: String? = null,
    @SerialName("completion_note") val completionNote: String? = null,
    val outcome: String? = null,
    val source: String = "MANUAL",
    @SerialName("is_overdue") val isOverdue: Boolean = false,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
    @SerialName("company_name") val companyName: String? = null,
    @SerialName("contact_name") val contactName: String? = null,
    @SerialName("assignee_name") val assigneeName: String? = null,
) {
    val displayName: String
        get() = companyName?.takeIf { it.isNotBlank() } ?: title?.takeIf { it.isNotBlank() } ?: leadId
}

@Serializable
data class FollowUpCreateRequest(
    @SerialName("lead_id") val leadId: String,
    @SerialName("due_at") val dueAt: String,
    val type: String = "FOLLOW_UP",
    val title: String? = null,
    val description: String? = null,
    val channel: String? = null,
    val priority: String = FollowUpPriority.MEDIUM,
    @SerialName("assigned_to") val assignedTo: String? = null,
    @SerialName("contact_id") val contactId: String? = null,
)

@Serializable
data class FollowUpUpdateRequest(
    val title: String? = null,
    val description: String? = null,
    val channel: String? = null,
    val type: String? = null,
    val priority: String? = null,
    @SerialName("due_at") val dueAt: String? = null,
    @SerialName("assigned_to") val assignedTo: String? = null,
)

@Serializable
data class FollowUpRescheduleRequest(
    @SerialName("due_at") val dueAt: String,
)

@Serializable
data class FollowUpsResponse(
    @SerialName("followups") val items: List<FollowUp> = emptyList(),
    val count: Int = 0,
    val total: Int = 0,
)

/** Report submission body: POST /interactions/{id}/report. */
@Serializable
data class ReportSubmitRequest(
    val outcome: String,
    @SerialName("contact_reached") val contactReached: Boolean? = null,
    @SerialName("correct_person_reached") val correctPersonReached: Boolean? = null,
    @SerialName("decision_maker_identified") val decisionMakerIdentified: Boolean? = null,
    @SerialName("decision_maker_name") val decisionMakerName: String? = null,
    @SerialName("decision_maker_role") val decisionMakerRole: String? = null,
    @SerialName("interest_level") val interestLevel: String? = null,
    @SerialName("need_level") val needLevel: String? = null,
    @SerialName("estimated_purchase_capacity") val estimatedPurchaseCapacity: String? = null,
    @SerialName("purchase_timing") val purchaseTiming: String? = null,
    @SerialName("main_objection") val mainObjection: String? = null,
    val summary: String? = null,
    @SerialName("next_action") val nextAction: String? = null,
    @SerialName("follow_up_at") val followUpAt: String? = null,
    @SerialName("qualification_recommendation") val qualificationRecommendation: String? = null,
)

/**
 * Submission response: the persisted backend contact report
 * (backend reports.Report). The backend does NOT report a score delta; the
 * success card renders outcome, next action and follow-up time instead.
 */
@Serializable
data class ReportSubmitResponse(
    val id: String = "",
    @SerialName("interaction_id") val interactionId: String = "",
    val outcome: String = "",
    @SerialName("contact_reached") val contactReached: Boolean? = null,
    @SerialName("correct_person_reached") val correctPersonReached: Boolean? = null,
    @SerialName("decision_maker_identified") val decisionMakerIdentified: Boolean? = null,
    @SerialName("decision_maker_name") val decisionMakerName: String? = null,
    @SerialName("decision_maker_role") val decisionMakerRole: String? = null,
    @SerialName("interest_level") val interestLevel: String? = null,
    @SerialName("need_level") val needLevel: String? = null,
    @SerialName("estimated_purchase_capacity") val estimatedPurchaseCapacity: String? = null,
    @SerialName("purchase_timing") val purchaseTiming: String? = null,
    @SerialName("main_objection") val mainObjection: String? = null,
    val summary: String? = null,
    @SerialName("next_action") val nextAction: String? = null,
    @SerialName("follow_up_at") val followUpAt: String? = null,
    @SerialName("qualification_recommendation") val qualificationRecommendation: String? = null,
    @SerialName("submitted_by") val submittedBy: String = "",
    @SerialName("manager_override") val managerOverride: Boolean = false,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

/** Qualification checklist booleans, matching the backend's flat request body. */
@Serializable
data class QualifyRequest(
    @SerialName("business_relevant") val businessRelevant: Boolean = false,
    @SerialName("contact_info_valid") val contactInfoValid: Boolean = false,
    @SerialName("meaningful_contact_occurred") val meaningfulContactOccurred: Boolean = false,
    @SerialName("need_or_interest_exists") val needOrInterestExists: Boolean = false,
    @SerialName("decision_maker_identified") val decisionMakerIdentified: Boolean = false,
    @SerialName("next_sales_action_exists") val nextSalesActionExists: Boolean = false,
    val reason: String? = null,
)

@Serializable
data class QualifyResponse(
    val qualified: Boolean = false,
    @SerialName("lead_id") val leadId: String? = null,
)

@Serializable
data class DisqualifyRequest(
    val reason: String,
    val explanation: String? = null,
)

@Serializable
data class DisqualifyResponse(
    val disqualified: Boolean = false,
    @SerialName("lead_id") val leadId: String? = null,
)

/**
 * A submitted contact report for the global Reports screen, as returned by
 * GET /api/v1/me/reports. The backend returns persisted contact_reports
 * enriched with lead/interaction context in a single flat shape — the app must
 * NOT reconstruct this from per-lead calls.
 */
@Serializable
data class SubmittedReport(
    val id: String = "",
    @SerialName("interaction_id") val interactionId: String = "",
    @SerialName("lead_id") val leadId: String = "",
    @SerialName("lead_name") val leadName: String? = null,
    @SerialName("company_name") val companyName: String? = null,
    val channel: String = "",
    val outcome: String = "",
    @SerialName("contact_reached") val contactReached: Boolean? = null,
    @SerialName("interest_level") val interestLevel: String? = null,
    @SerialName("need_level") val needLevel: String? = null,
    @SerialName("main_objection") val mainObjection: String? = null,
    val summary: String? = null,
    @SerialName("next_action") val nextAction: String? = null,
    @SerialName("follow_up_at") val followUpAt: String? = null,
    @SerialName("submitted_by") val submittedBy: String? = null,
    @SerialName("submitter_name") val submitterName: String? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
) {
    val displayName: String
        get() = companyName?.takeIf { it.isNotBlank() }
            ?: leadName?.takeIf { it.isNotBlank() }
            ?: leadId
}

/** Paged response from GET /api/v1/me/reports. */
@Serializable
data class ReportsResponse(
    val items: List<SubmittedReport> = emptyList(),
    val total: Int = 0,
    val count: Int = 0,
)
