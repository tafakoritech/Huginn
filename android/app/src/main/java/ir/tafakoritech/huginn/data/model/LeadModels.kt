package ir.tafakoritech.huginn.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Lead lifecycle status, matching backend/internal/leads/lead.go. */
object LeadStatus {
    const val NEW = "NEW"
    const val READY = "READY"
    const val ASSIGNED = "ASSIGNED"
    const val CONTACTING = "CONTACTING"
    const val FOLLOW_UP = "FOLLOW_UP"
    const val QUALIFIED = "QUALIFIED"
    const val DISQUALIFIED = "DISQUALIFIED"
    const val EXPORTED_TO_MUNINN = "EXPORTED_TO_MUNINN"
    const val ARCHIVED = "ARCHIVED"

    /** All valid statuses in pipeline order (backend leads.AllStatuses). */
    val all = listOf(
        NEW, READY, ASSIGNED, CONTACTING, FOLLOW_UP, QUALIFIED, DISQUALIFIED, EXPORTED_TO_MUNINN, ARCHIVED,
    )
}

/** Communication channels, matching backend/internal/interactions/interactions.go. */
object InteractionChannel {
    const val CALL = "CALL"
    const val SMS = "SMS"
    const val WHATSAPP = "WHATSAPP"
    const val TELEGRAM = "TELEGRAM"
    const val BALE = "BALE"
    const val EITAA = "EITAA"
    const val RUBIKA = "RUBIKA"
    const val EMAIL = "EMAIL"
}

object InteractionObjective {
    const val FIRST_CONTACT = "FIRST_CONTACT"
    const val FOLLOW_UP = "FOLLOW_UP"
    const val QUALIFICATION = "QUALIFICATION"
    const val INFO_REQUEST = "INFO_REQUEST"
}

/** Lead list item as returned by the backend list endpoint. */
@Serializable
data class Lead(
    val id: String,
    @SerialName("company_name") val companyName: String,
    @SerialName("display_name") val displayName: String? = null,
    @SerialName("business_type") val businessType: String? = null,
    val industry: String? = null,
    val description: String? = null,
    val province: String? = null,
    val city: String? = null,
    val address: String? = null,
    val website: String? = null,
    val instagram: String? = null,
    val source: String? = null,
    @SerialName("source_batch_id") val sourceBatchId: String? = null,
    @SerialName("source_document_id") val sourceDocumentId: String? = null,
    val status: String = "",
    @SerialName("quality_status") val qualityStatus: String = "",
    val score: Int = 0,
    @SerialName("owner_user_id") val ownerUserId: String? = null,
    @SerialName("owner_name") val ownerName: String? = null,
    @SerialName("primary_contact_name") val primaryContactName: String? = null,
    @SerialName("primary_contact_role") val primaryContactRole: String? = null,
    @SerialName("last_interaction_at") val lastInteractionAt: String? = null,
    @SerialName("last_interaction_channel") val lastInteractionChannel: String? = null,
    @SerialName("next_followup_at") val nextFollowupAt: String? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
    @SerialName("archived_at") val archivedAt: String? = null,
    val tags: List<String> = emptyList(),
)

@Serializable
data class LeadListResponse(
    val leads: List<Lead> = emptyList(),
    val total: Int = 0,
    val page: Int = 1,
    val size: Int = 20,
)

/** Prioritized work item returned by GET /me/work-queue (backend WorkQueueItem). */
@Serializable
data class WorkItem(
    @SerialName("lead_id") val leadId: String,
    @SerialName("company_name") val companyName: String,
    val priority: Int = 0,
    val reason: String = "",
    val status: String = "",
    val score: Int = 0,
    @SerialName("due_at") val dueAt: String? = null,
    @SerialName("primary_contact_name") val primaryContactName: String? = null,
    @SerialName("city") val city: String? = null,
    @SerialName("business_type") val businessType: String? = null,
    @SerialName("next_action") val nextAction: String? = null,
)

@Serializable
data class WorkQueueResponse(val items: List<WorkItem> = emptyList())

/** Pending mandatory report, derived from the flat backend pending-reports response. */
@Serializable
data class PendingReport(
    @SerialName("interaction_id") val interactionId: String,
    @SerialName("lead_id") val leadId: String,
    @SerialName("company_name") val companyName: String? = null,
    @SerialName("contact_name") val contactName: String? = null,
    val channel: String = "",
    @SerialName("started_at") val startedAt: String? = null,
    // Canonical persistent facts promoted from earlier reports: the app pre-fills
    // these on the next report so stable information is never re-asked.
    @SerialName("correct_person_reached") val correctPersonReached: Boolean? = null,
    @SerialName("decision_maker_identified") val decisionMakerIdentified: Boolean? = null,
    @SerialName("decision_maker_name") val decisionMakerName: String? = null,
    @SerialName("decision_maker_role") val decisionMakerRole: String? = null,
)

/**
 * Mandatory-report state returned by GET /me/pending-reports. The backend
 * responds with flat fields (no `reports` array); [reports] derives a single
 * [PendingReport] so the app gate/UI contract stays stable.
 */
@Serializable
data class PendingReportsResponse(
    @SerialName("requires_report") val requiresReport: Boolean = false,
    @SerialName("interaction_id") val interactionId: String? = null,
    @SerialName("lead_id") val leadId: String? = null,
    val channel: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("company_name") val companyName: String? = null,
    @SerialName("contact_name") val contactName: String? = null,
    @SerialName("correct_person_reached") val correctPersonReached: Boolean? = null,
    @SerialName("decision_maker_identified") val decisionMakerIdentified: Boolean? = null,
    @SerialName("decision_maker_name") val decisionMakerName: String? = null,
    @SerialName("decision_maker_role") val decisionMakerRole: String? = null,
) {
    val reports: List<PendingReport>
        get() = if (requiresReport && !interactionId.isNullOrBlank()) {
            listOf(
                PendingReport(
                    interactionId = interactionId,
                    leadId = leadId.orEmpty(),
                    companyName = companyName,
                    contactName = contactName,
                    channel = channel.orEmpty(),
                    startedAt = createdAt,
                    correctPersonReached = correctPersonReached,
                    decisionMakerIdentified = decisionMakerIdentified,
                    decisionMakerName = decisionMakerName,
                    decisionMakerRole = decisionMakerRole,
                ),
            )
        } else {
            emptyList()
        }
}

/** Per-employee KPI snapshot returned by GET /dashboard/me (backend AgentMetrics). */
@Serializable
data class AgentMetrics(
    @SerialName("assigned_leads") val assignedLeads: Int = 0,
    @SerialName("calls_today") val callsToday: Int = 0,
    @SerialName("remaining_work_queue") val remainingWorkQueue: Int = 0,
    @SerialName("pending_reports") val pendingReports: Int = 0,
    @SerialName("follow_ups") val followUps: Int = 0,
    @SerialName("qualified_leads") val qualifiedLeads: Int = 0,
    @SerialName("contact_rate") val contactRate: Double = 0.0,
) {
    /** Maps the backend agent metrics onto the home-screen [MeStats] layout. */
    fun toMeStats(): MeStats = MeStats(
        todayLeads = assignedLeads,
        todayCalls = callsToday,
        pendingReports = pendingReports,
        todayFollowups = followUps,
        overdueFollowups = 0,
        qualifiedLeads = qualifiedLeads,
    )
}

/** Dashboard statistics for the home screen (fed from AgentMetrics). */
@Serializable
data class MeStats(
    @SerialName("today_leads") val todayLeads: Int = 0,
    @SerialName("today_calls") val todayCalls: Int = 0,
    @SerialName("pending_reports") val pendingReports: Int = 0,
    @SerialName("today_followups") val todayFollowups: Int = 0,
    @SerialName("overdue_followups") val overdueFollowups: Int = 0,
    @SerialName("qualified_leads") val qualifiedLeads: Int = 0,
)
