package ir.tafakoritech.huginn.core.api

import ir.tafakoritech.huginn.data.model.AgentMetrics
import ir.tafakoritech.huginn.data.model.Agent
import ir.tafakoritech.huginn.data.model.AuthResponse
import ir.tafakoritech.huginn.data.model.AssignRequest
import ir.tafakoritech.huginn.data.model.CallScenario
import ir.tafakoritech.huginn.data.model.Contact
import ir.tafakoritech.huginn.data.model.ContactActionRequest
import ir.tafakoritech.huginn.data.model.ContactActionResponse
import ir.tafakoritech.huginn.data.model.ContactUpdateRequest
import ir.tafakoritech.huginn.data.model.CreateNoteRequest
import ir.tafakoritech.huginn.data.model.DisqualifyRequest
import ir.tafakoritech.huginn.data.model.DisqualifyResponse
import ir.tafakoritech.huginn.data.model.EvidenceResponse
import ir.tafakoritech.huginn.data.model.FollowUp
import ir.tafakoritech.huginn.data.model.FollowUpCreateRequest
import ir.tafakoritech.huginn.data.model.FollowUpRescheduleRequest
import ir.tafakoritech.huginn.data.model.FollowUpUpdateRequest
import ir.tafakoritech.huginn.data.model.FollowUpsResponse
import ir.tafakoritech.huginn.data.model.Interaction
import ir.tafakoritech.huginn.data.model.LeadDetailResponse
import ir.tafakoritech.huginn.data.model.LeadListResponse
import ir.tafakoritech.huginn.data.model.LoginRequest
import ir.tafakoritech.huginn.data.model.NextActionsResponse
import ir.tafakoritech.huginn.data.model.CompleteNextActionRequest
import ir.tafakoritech.huginn.data.model.SnoozeNextActionRequest
import ir.tafakoritech.huginn.data.model.Note
import ir.tafakoritech.huginn.data.model.NotesResponse
import ir.tafakoritech.huginn.data.model.PendingReportsResponse
import ir.tafakoritech.huginn.data.model.QualifyRequest
import ir.tafakoritech.huginn.data.model.QualifyResponse
import ir.tafakoritech.huginn.data.model.ReportSubmitRequest
import ir.tafakoritech.huginn.data.model.ReportSubmitResponse
import ir.tafakoritech.huginn.data.model.ReportsResponse
import ir.tafakoritech.huginn.data.model.ScoreResponse
import ir.tafakoritech.huginn.data.model.TimelineResponse
import ir.tafakoritech.huginn.data.model.User
import ir.tafakoritech.huginn.data.model.WorkQueueResponse
import kotlinx.serialization.json.JsonObject
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query

/**
 * Retrofit contract for the real Huginn backend.
 *
 * Base URL is configured per build (HUGINN_API_BASE_URL); paths below include
 * the api/v1 prefix. Every path, field and response shape mirrors
 * backend/internal/httpapi (see backend/api/openapi.yaml). The client talks
 * ONLY to this HTTP API — never to DeepSeek, the queue, the database, MinIO or
 * Tika directly.
 */
interface HuginnApi {

    // ---- Auth ----
    @POST("api/v1/auth/login")
    suspend fun login(@Body request: LoginRequest): AuthResponse

    @POST("api/v1/auth/logout")
    suspend fun logout(@Header("X-Refresh-Token") refreshToken: String)

    // ---- Me ----
    @GET("api/v1/me")
    suspend fun me(): User

    @GET("api/v1/me/work-queue")
    suspend fun workQueue(): WorkQueueResponse

    @GET("api/v1/me/next-actions")
    suspend fun nextActions(@Query("limit") limit: Int? = null): NextActionsResponse

    @POST("api/v1/me/next-actions/{id}/snooze")
    suspend fun snoozeNextAction(@Path("id") id: String, @Body request: SnoozeNextActionRequest)

    @POST("api/v1/me/next-actions/{id}/complete")
    suspend fun completeNextAction(@Path("id") id: String, @Body request: CompleteNextActionRequest)

    @GET("api/v1/me/pending-reports")
    suspend fun pendingReports(): PendingReportsResponse

    /** Submitted contact reports visible to the current user (global Reports screen). */
    @GET("api/v1/me/reports")
    suspend fun myReports(
        @Query("limit") limit: Int? = null,
        @Query("offset") offset: Int? = null,
    ): ReportsResponse

    @GET("api/v1/dashboard/me")
    suspend fun dashboardMe(): AgentMetrics

    // ---- Lead follow-ups ----
    @GET("api/v1/leads/{id}/followups")
    suspend fun leadFollowUps(@Path("id") id: String): List<FollowUp>

    // ---- Leads ----
    @GET("api/v1/leads")
    suspend fun leads(
        @Query("search") search: String? = null,
        @Query("status") status: String? = null,
        @Query("scope") scope: String? = null,
        @Query("min_score") minScore: Int? = null,
        @Query("max_score") maxScore: Int? = null,
        @Query("city") city: String? = null,
        @Query("industry") industry: String? = null,
        @Query("business_type") businessType: String? = null,
        @Query("tag") tag: String? = null,
        @Query("qualified") qualified: Boolean? = null,
        @Query("page") page: Int = 1,
        @Query("page_size") pageSize: Int = 20,
        @Query("sort") sort: String? = null,
        @Query("order") order: String? = null,
    ): LeadListResponse

    @GET("api/v1/leads/{id}")
    suspend fun lead(@Path("id") id: String): LeadDetailResponse

    /**
     * PATCH a lead (versioned edit). Fields present in the body are applied;
     * an explicit null clears the field; `note` records the change reason.
     */
    @PATCH("api/v1/leads/{id}")
    suspend fun updateLead(@Path("id") id: String, @Body body: JsonObject): LeadDetailResponse

    @PATCH("api/v1/leads/{id}/contacts/{contactId}")
    suspend fun updateContact(
        @Path("id") leadId: String,
        @Path("contactId") contactId: String,
        @Body request: ContactUpdateRequest,
    ): Contact

    @POST("api/v1/leads/{id}/assign")
    suspend fun assignLead(@Path("id") id: String, @Body request: AssignRequest)

    @GET("api/v1/agents")
    suspend fun agents(): List<Agent>

    @GET("api/v1/leads/{id}/timeline")
    suspend fun timeline(@Path("id") id: String): TimelineResponse

    @GET("api/v1/leads/{id}/notes")
    suspend fun notes(@Path("id") id: String): NotesResponse

    @POST("api/v1/leads/{id}/notes")
    suspend fun createNote(@Path("id") id: String, @Body request: CreateNoteRequest): Note

    @GET("api/v1/leads/{id}/evidence")
    suspend fun leadEvidence(@Path("id") id: String): EvidenceResponse

    @GET("api/v1/leads/{id}/score")
    suspend fun leadScore(@Path("id") id: String): ScoreResponse

    @GET("api/v1/leads/{id}/interactions")
    suspend fun leadInteractions(@Path("id") id: String): List<Interaction>

    // ---- Follow-ups ----
    @GET("api/v1/followups")
    suspend fun followUps(
        @Query("view") view: String? = null,
        @Query("status") status: String? = null,
        @Query("assigned_user") assignedUser: String? = null,
        @Query("lead_id") leadId: String? = null,
        @Query("from") from: String? = null,
        @Query("to") to: String? = null,
        @Query("overdue") overdue: Boolean? = null,
        @Query("limit") limit: Int? = null,
        @Query("offset") offset: Int? = null,
    ): FollowUpsResponse

    @POST("api/v1/followups")
    suspend fun createFollowUp(@Body request: FollowUpCreateRequest): FollowUp

    @GET("api/v1/followups/{id}")
    suspend fun followUp(@Path("id") id: String): FollowUp

    @PATCH("api/v1/followups/{id}")
    suspend fun updateFollowUp(@Path("id") id: String, @Body request: FollowUpUpdateRequest): FollowUp

    @POST("api/v1/followups/{id}/complete")
    suspend fun completeFollowUp(@Path("id") id: String)

    @POST("api/v1/followups/{id}/reopen")
    suspend fun reopenFollowUp(@Path("id") id: String): FollowUp

    @POST("api/v1/followups/{id}/reschedule")
    suspend fun rescheduleFollowUp(@Path("id") id: String, @Body request: FollowUpRescheduleRequest): FollowUp

    @DELETE("api/v1/followups/{id}")
    suspend fun cancelFollowUp(@Path("id") id: String): FollowUp

    // ---- Communication ----
    @POST("api/v1/leads/{id}/contact-actions")
    suspend fun contactAction(
        @Path("id") leadId: String,
        @Body request: ContactActionRequest,
    ): ContactActionResponse

    /** Suggested call scenario for a lead (no interaction is created). */
    // NOTE: call-scenario generation waits on the LLM and can take minutes.
    // It is served by LlmHuginnApi through the dedicated long-running OkHttp
    // client (NetworkModule.provideLlmClient), never by this 30s client.

    @GET("api/v1/leads/{id}/call-scenarios")
    suspend fun getCallScenarios(@Path("id") leadId: String): List<CallScenario>

    @POST("api/v1/leads/{id}/call-scenarios")
    suspend fun createCallScenario(
        @Path("id") leadId: String,
        @Body request: CallScenario,
    ): CallScenario

    @GET("api/v1/leads/{id}/call-scenarios/{scenarioId}")
    suspend fun getCallScenario(
        @Path("id") leadId: String,
        @Path("scenarioId") scenarioId: String,
    ): CallScenario

    @PUT("api/v1/leads/{id}/call-scenarios/{scenarioId}")
    suspend fun updateCallScenario(
        @Path("id") leadId: String,
        @Path("scenarioId") scenarioId: String,
        @Body request: CallScenario,
    ): CallScenario

    @DELETE("api/v1/leads/{id}/call-scenarios/{scenarioId}")
    suspend fun deleteCallScenario(
        @Path("id") leadId: String,
        @Path("scenarioId") scenarioId: String,
    ): retrofit2.Response<Unit>

    // ---- Reports ----
    @POST("api/v1/interactions/{id}/report")
    suspend fun submitReport(
        @Path("id") interactionId: String,
        @Body request: ReportSubmitRequest,
    ): ReportSubmitResponse

    // ---- Qualification ----
    @POST("api/v1/leads/{id}/qualify")
    suspend fun qualify(
        @Path("id") leadId: String,
        @Body request: QualifyRequest,
    ): QualifyResponse

    @POST("api/v1/leads/{id}/disqualify")
    suspend fun disqualify(
        @Path("id") leadId: String,
        @Body request: DisqualifyRequest,
    ): DisqualifyResponse
}
