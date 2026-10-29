package ir.tafakoritech.huginn.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** A person associated with a lead, matching backend/internal/contacts/contacts.go. */
@Serializable
data class Contact(
    val id: String,
    @SerialName("lead_id") val leadId: String,
    @SerialName("first_name") val firstName: String? = null,
    @SerialName("last_name") val lastName: String? = null,
    @SerialName("full_name") val fullName: String? = null,
    @SerialName("job_title") val jobTitle: String? = null,
    val department: String? = null,
    val mobile: String? = null,
    @SerialName("mobile_normalized") val mobileNormalized: String? = null,
    val phone: String? = null,
    @SerialName("phone_normalized") val phoneNormalized: String? = null,
    val email: String? = null,
    @SerialName("email_normalized") val emailNormalized: String? = null,
    val whatsapp: String? = null,
    val telegram: String? = null,
    val bale: String? = null,
    val eitaa: String? = null,
    val rubika: String? = null,
    @SerialName("is_decision_maker") val isDecisionMaker: Boolean = false,
    @SerialName("is_primary_contact") val isPrimaryContact: Boolean = false,
) {
    val displayName: String
        get() = fullName
            ?: listOfNotNull(firstName, lastName).joinToString(" ")

    val primaryNumber: String?
        get() = mobileNormalized ?: mobile ?: phoneNormalized ?: phone
}

/** A single outbound/inbound interaction (backend Interaction). */
@Serializable
data class Interaction(
    val id: String,
    @SerialName("lead_id") val leadId: String,
    @SerialName("contact_id") val contactId: String? = null,
    @SerialName("user_id") val userId: String? = null,
    val channel: String = "",
    val direction: String = "OUTBOUND",
    val objective: String? = null,
    @SerialName("started_at") val startedAt: String = "",
    @SerialName("ended_at") val endedAt: String? = null,
    @SerialName("duration_seconds") val durationSeconds: Int? = null,
    @SerialName("action_status") val actionStatus: String = "",
    @SerialName("report_status") val reportStatus: String = "",
    @SerialName("suggested_message") val suggestedMessage: String? = null,
    @SerialName("suggested_subject") val suggestedSubject: String? = null,
    @SerialName("final_message_if_known") val finalMessageIfKnown: String? = null,
    @SerialName("deep_link") val deepLink: String? = null,
    @SerialName("outcome") val outcome: String? = null,
    @SerialName("contact_name") val contactName: String? = null,
)

@Serializable
data class TimelineEvent(
    val id: String = "",
    @SerialName("lead_id") val leadId: String = "",
    @SerialName("event_type") val eventType: String = "",
    val title: String = "",
    val description: String? = null,
    @SerialName("actor_user_id") val actorUserId: String? = null,
    @SerialName("actor_name") val actorName: String? = null,
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class TimelineResponse(
    @SerialName("timeline") val items: List<TimelineEvent> = emptyList(),
)

/** Score breakdown; backend ScoreSnapshot.components maps criterion -> points. */
@Serializable
data class ScoreBreakdown(
    val total: Int = 0,
    val components: Map<String, Int> = emptyMap(),
)

/** LLM extraction evidence (backend evidence.Item). */
@Serializable
data class EvidenceItem(
    val id: String = "",
    @SerialName("lead_id") val leadId: String = "",
    @SerialName("field_name") val fieldName: String = "",
    @SerialName("source_excerpt") val sourceExcerpt: String? = null,
    val confidence: Double = 0.0,
    @SerialName("source_page") val sourcePage: Int? = null,
    @SerialName("source_row") val sourceRow: Int? = null,
    @SerialName("source_document") val sourceDocument: String? = null,
)

/** One row from GET /leads/{id}/evidence (backend handler shape). */
@Serializable
data class EvidenceItemDto(
    @SerialName("field_name") val fieldName: String = "",
    val value: String? = null,
    val confidence: Int = 0,
    val source: String? = null,
    val excerpt: String? = null,
    val page: Int? = null,
    val row: Int? = null,
)

@Serializable
data class EvidenceResponse(val items: List<EvidenceItemDto> = emptyList())

/** Score component row from GET /leads/{id}/score. */
@Serializable
data class ScoreComponentDto(
    val key: String = "",
    val name: String? = null,
    val weight: Int = 0,
    val value: Int = 0,
    val max: Int = 0,
)

@Serializable
data class ScoreResponse(
    val total: Int = 0,
    val max: Int = 100,
    val components: List<ScoreComponentDto> = emptyList(),
)

@Serializable
data class Note(
    val id: String,
    @SerialName("lead_id") val leadId: String,
    @SerialName("author_id") val authorId: String,
    @SerialName("author_name") val authorName: String? = null,
    val text: String,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class NotesResponse(
    @SerialName("notes") val items: List<Note> = emptyList(),
)

@Serializable
data class CreateNoteRequest(val text: String)

/** Server-driven permission set for the current employee on a lead. */
@Serializable
data class LeadPermissions(
    @SerialName("can_call") val canCall: Boolean = true,
    @SerialName("can_message") val canMessage: Boolean = true,
    @SerialName("can_edit") val canEdit: Boolean = false,
    @SerialName("can_assign") val canAssign: Boolean = false,
    @SerialName("can_qualify") val canQualify: Boolean = false,
    @SerialName("can_export_to_muninn") val canExportToMuninn: Boolean = false,
    @SerialName("can_disqualify") val canDisqualify: Boolean = true,
)

@Serializable
data class WorkflowDto(
    @SerialName("mandatory_report_pending") val mandatoryReportPending: Boolean = false,
    @SerialName("next_action") val nextAction: String? = null,
)

@Serializable
data class ScoreComponent(
    val key: String = "",
    val label: String? = null,
    val value: Int = 0,
    val max: Int = 100,
)

/**
 * Lead detail as returned by GET /leads/{id}. The backend nests the lead under
 * `lead` and provides `contacts`, `permissions`, `workflow` and `recent_events`
 * alongside it.
 */
@Serializable
data class LeadDetailResponse(
    val lead: Lead = Lead(id = "", companyName = ""),
    val contacts: List<Contact> = emptyList(),
    val permissions: LeadPermissions = LeadPermissions(),
    val workflow: WorkflowDto = WorkflowDto(),
    @SerialName("recent_events") val recentEvents: List<TimelineEvent> = emptyList(),
) {
    /**
     * Flattens into the [LeadDetail] shape used by the UI, enriched with the
     * real per-lead data served by the backend's dedicated endpoints
     * (score breakdown, evidence, interactions and follow-ups).
     */
    fun toDetail(
        score: ScoreResponse?,
        evidence: List<EvidenceItem>,
        interactions: List<Interaction>,
        followUps: List<FollowUp>,
    ): LeadDetail = LeadDetail(
        id = lead.id,
        companyName = lead.companyName,
        displayName = lead.displayName,
        businessType = lead.businessType,
        industry = lead.industry,
        description = lead.description,
        province = lead.province,
        city = lead.city,
        address = lead.address,
        website = lead.website,
        instagram = lead.instagram,
        source = lead.source,
        status = lead.status,
        qualityStatus = lead.qualityStatus,
        score = lead.score,
        ownerUserId = lead.ownerUserId,
        ownerName = lead.ownerName,
        createdAt = lead.createdAt,
        updatedAt = lead.updatedAt,
        tags = lead.tags,
        contacts = contacts,
        permissions = permissions,
        workflow = workflow,
        scoreBreakdown = score?.let {
            ScoreBreakdown(
                total = it.total,
                components = it.components.associate { c -> c.key to c.value },
            )
        },
        scoreExplanation = score?.components?.map {
            ScoreComponent(key = it.key, label = it.name, value = it.value, max = it.max)
        }.orEmpty(),
        nextFollowup = followUps
            .filter { it.status == "PENDING" }
            .minByOrNull { it.dueAt },
        lastInteraction = interactions.firstOrNull(),
        evidence = evidence,
    )
}

/** Full lead detail returned by GET /leads/{id} (flattened by [LeadDetailResponse.toDetail]). */
@Serializable
data class LeadDetail(
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
    val status: String = "",
    @SerialName("quality_status") val qualityStatus: String = "",
    val score: Int = 0,
    @SerialName("owner_user_id") val ownerUserId: String? = null,
    @SerialName("owner_name") val ownerName: String? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
    val tags: List<String> = emptyList(),
    val contacts: List<Contact> = emptyList(),
    val permissions: LeadPermissions = LeadPermissions(),
    val workflow: WorkflowDto = WorkflowDto(),
    @SerialName("score_breakdown") val scoreBreakdown: ScoreBreakdown? = null,
    @SerialName("score_explanation") val scoreExplanation: List<ScoreComponent> = emptyList(),
    @SerialName("next_followup") val nextFollowup: FollowUp? = null,
    @SerialName("last_interaction") val lastInteraction: Interaction? = null,
    val evidence: List<EvidenceItem> = emptyList(),
)

/**
 * Patch body for PATCH /leads/{id}/contacts/{contactId}. Only fields present in
 * the JSON are changed by the backend; a null value clears the field.
 */
@Serializable
data class ContactUpdateRequest(
    @SerialName("first_name") val firstName: String? = null,
    @SerialName("last_name") val lastName: String? = null,
    @SerialName("full_name") val fullName: String? = null,
    @SerialName("job_title") val jobTitle: String? = null,
    val department: String? = null,
    val mobile: String? = null,
    val phone: String? = null,
    val email: String? = null,
    val whatsapp: String? = null,
    val telegram: String? = null,
    val bale: String? = null,
    val eitaa: String? = null,
    val rubika: String? = null,
    @SerialName("is_decision_maker") val isDecisionMaker: Boolean? = null,
    @SerialName("is_primary_contact") val isPrimaryContact: Boolean? = null,
)

/** Body for POST /leads/{id}/assign. */
@Serializable
data class AssignRequest(
    @SerialName("assignee_id") val assigneeId: String,
    val method: String = "manual",
)

/** One agent row from GET /agents (used to pick an assignee). */
@Serializable
data class Agent(
    @SerialName("user_id") val userId: String,
    val name: String,
    @SerialName("assigned_leads") val assignedLeads: Int = 0,
)
