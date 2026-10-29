package ir.tafakoritech.huginn.data.repository

import ir.tafakoritech.huginn.core.api.ApiCaller
import ir.tafakoritech.huginn.core.api.HuginnApi
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.runCatchingAppError
import ir.tafakoritech.huginn.data.model.Agent
import ir.tafakoritech.huginn.data.model.AssignRequest
import ir.tafakoritech.huginn.data.model.Contact
import ir.tafakoritech.huginn.data.model.ContactUpdateRequest
import ir.tafakoritech.huginn.data.model.CreateNoteRequest
import ir.tafakoritech.huginn.data.model.EvidenceItem
import ir.tafakoritech.huginn.data.model.LeadDetail
import ir.tafakoritech.huginn.data.model.LeadListResponse
import ir.tafakoritech.huginn.data.model.Note
import ir.tafakoritech.huginn.data.model.NotesResponse
import ir.tafakoritech.huginn.data.model.TimelineResponse
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import javax.inject.Inject
import javax.inject.Singleton

/** Query filters for the lead list screen. */
data class LeadFilters(
    val search: String? = null,
    val scope: String? = null,
    val status: String? = null,
    val minScore: Int? = null,
    val city: String? = null,
    val industry: String? = null,
    val businessType: String? = null,
    val tag: String? = null,
    val qualified: Boolean? = null,
    val sort: String? = null,
    val order: String? = null,
)

/**
 * Lead browsing. All data comes from the real Huginn API — there is no local
 * fallback: a network failure propagates as a typed error so the UI can show
 * the appropriate error/retry state instead of stale or fabricated data.
 */
@Singleton
class LeadRepository @Inject constructor(
    private val api: HuginnApi,
    private val json: Json,
) {

    suspend fun listLeads(filters: LeadFilters, page: Int, pageSize: Int): AppResult<LeadListResponse> =
        runCatchingAppError {
            ApiCaller.call(json) {
                api.leads(
                    search = filters.search,
                    status = filters.status,
                    scope = filters.scope,
                    minScore = filters.minScore,
                    city = filters.city,
                    industry = filters.industry,
                    businessType = filters.businessType,
                    tag = filters.tag,
                    qualified = filters.qualified,
                    page = page,
                    pageSize = pageSize,
                    sort = filters.sort,
                    order = filters.order,
                )
            }
        }

    /**
     * Loads a lead detail from the backend and enriches it with the real
     * per-lead resources (score breakdown, evidence, interactions, follow-ups).
     */
    suspend fun leadDetail(id: String): AppResult<LeadDetail> =
        runCatchingAppError {
            ApiCaller.call(json) { enrich(id) }
        }

    private suspend fun enrich(id: String): LeadDetail = coroutineScope {
        val detail = async { ApiCaller.call(json) { api.lead(id) } }
        val score = async { ApiCaller.call(json) { api.leadScore(id) } }
        val evidence = async { ApiCaller.call(json) { api.leadEvidence(id) } }
        val interactions = async { ApiCaller.call(json) { api.leadInteractions(id) } }
        val followUps = async { ApiCaller.call(json) { api.leadFollowUps(id) } }

        detail.await().toDetail(
            score = score.await(),
            evidence = evidence.await().items.map { it.toEvidenceItem() },
            interactions = interactions.await(),
            followUps = followUps.await(),
        )
    }

    suspend fun timeline(id: String): AppResult<TimelineResponse> =
        runCatchingAppError { ApiCaller.call(json) { api.timeline(id) } }

    suspend fun notes(id: String): AppResult<NotesResponse> =
        runCatchingAppError { ApiCaller.call(json) { api.notes(id) } }

    suspend fun addNote(id: String, text: String): AppResult<Note> =
        runCatchingAppError { ApiCaller.call(json) { api.createNote(id, CreateNoteRequest(text.trim())) } }

    // ---- Editing ----

    /**
     * PATCHes a lead through the real backend (versioned edit). Only fields
     * provided in [patch] are sent; a cleared (null) value is encoded as an
     * explicit JSON null so the backend removes the field. The caller then
     * refreshes the detail from the backend.
     */
    suspend fun updateLead(id: String, patch: LeadPatch, note: String?): AppResult<Unit> =
        runCatchingAppError {
            ApiCaller.call(json) { api.updateLead(id, buildLeadPatchBody(patch, note)) }
        }

    /** PATCHes a contact on a lead. Only non-null fields are changed by the backend. */
    suspend fun updateContact(leadId: String, contactId: String, request: ContactUpdateRequest): AppResult<Contact> =
        runCatchingAppError {
            ApiCaller.call(json) { api.updateContact(leadId, contactId, request) }
        }

    /** Assigns the lead to an agent through POST /leads/{id}/assign. */
    suspend fun assignLead(leadId: String, assigneeId: String): AppResult<Unit> =
        runCatchingAppError {
            ApiCaller.call(json) { api.assignLead(leadId, AssignRequest(assigneeId = assigneeId)) }
        }

    /** Agents available for assignment (GET /agents). */
    suspend fun agents(): AppResult<List<Agent>> =
        runCatchingAppError { ApiCaller.call(json) { api.agents() } }

    private fun buildLeadPatchBody(patch: LeadPatch, note: String?): JsonObject = buildJsonObject {
        patchText("company_name", patch.companyName)
        patchText("display_name", patch.displayName)
        patchText("business_type", patch.businessType)
        patchText("industry", patch.industry)
        patchText("province", patch.province)
        patchText("city", patch.city)
        patchText("address", patch.address)
        patchText("website", patch.website)
        patchText("instagram", patch.instagram)
        patchText("source", patch.source)
        patchText("description", patch.description)
        patch.status?.takeIf { it.isNotBlank() }?.let { put("status", JsonPrimitive(it)) }
        if (!note.isNullOrBlank()) put("note", JsonPrimitive(note))
    }

    private fun kotlinx.serialization.json.JsonObjectBuilder.patchText(key: String, value: String?) {
        if (value == null) put(key, JsonNull) else put(key, JsonPrimitive(value))
    }
}

/** Editable lead fields (blank/null clears the field on the backend). */
data class LeadPatch(
    val companyName: String? = null,
    val displayName: String? = null,
    val businessType: String? = null,
    val industry: String? = null,
    val province: String? = null,
    val city: String? = null,
    val address: String? = null,
    val website: String? = null,
    val instagram: String? = null,
    val source: String? = null,
    val description: String? = null,
    val status: String? = null,
)

private fun ir.tafakoritech.huginn.data.model.EvidenceItemDto.toEvidenceItem(): EvidenceItem =
    EvidenceItem(
        id = "",
        leadId = "",
        fieldName = fieldName,
        sourceExcerpt = excerpt,
        confidence = confidence.toDouble() / 100.0,
        sourcePage = page,
        sourceRow = row,
        sourceDocument = source,
    )
