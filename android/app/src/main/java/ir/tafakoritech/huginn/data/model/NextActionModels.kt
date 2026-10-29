package ir.tafakoritech.huginn.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Source of a next action (backend nextactions.SourceType). */
object NextActionSource {
    const val FOLLOW_UP = "follow_up"
    const val PENDING_REPORT = "pending_report"
    const val FIRST_CONTACT = "first_contact"
    const val STALE_LEAD = "stale_lead"
}

/** Executable action types (backend nextactions.ActionType). */
object NextActionType {
    const val CALL = "call"
    const val MESSAGE = "message"
    const val EMAIL = "email"
    const val REPORT = "report"
    const val FOLLOW_UP = "follow_up"
    const val QUALIFICATION = "qualification"
    const val LEAD_REVIEW = "lead_review"
}

/** Semantic priority tiers (backend nextactions.Tier). */
object NextActionPriority {
    const val URGENT = "urgent"
    const val IMPORTANT = "important"
    const val NORMAL = "normal"
}

/**
 * Canonical work item returned by GET /me/next-actions. The backend is the
 * single source of truth for ranking; the app only renders and executes.
 */
@Serializable
data class NextActionItem(
    val id: String,
    @SerialName("source_type") val sourceType: String = "",
    @SerialName("source_id") val sourceId: String = "",
    @SerialName("lead_id") val leadId: String,
    @SerialName("lead_name") val leadName: String = "",
    @SerialName("company_name") val companyName: String = "",
    @SerialName("contact_name") val contactName: String? = null,
    @SerialName("action_type") val actionType: String = "",
    val title: String = "",
    val reason: String = "",
    val context: String? = null,
    val priority: String = NextActionPriority.NORMAL,
    @SerialName("priority_score") val priorityScore: Int = 0,
    @SerialName("due_at") val dueAt: String? = null,
    val overdue: Boolean = false,
    @SerialName("suggested_action") val suggestedAction: String = "",
    @SerialName("cta_label") val ctaLabel: String = "",
    @SerialName("project_id") val projectId: String? = null,
    @SerialName("project_title") val projectTitle: String? = null,
    @SerialName("last_interaction_at") val lastInteractionAt: String? = null,
    @SerialName("assignee_name") val assigneeName: String? = null,
    val explainability: List<String> = emptyList(),
)

@Serializable
data class NextActionsResponse(
    val items: List<NextActionItem> = emptyList(),
    @SerialName("generated_at") val generatedAt: String? = null,
    @SerialName("total_actionable") val totalActionable: Int = 0,
    @SerialName("urgent_count") val urgentCount: Int = 0,
    @SerialName("overdue_count") val overdueCount: Int = 0,
)

@Serializable
data class SnoozeNextActionRequest(
    val until: String,
)

@Serializable
data class CompleteNextActionRequest(
    val note: String? = null,
    val outcome: String? = null,
)
