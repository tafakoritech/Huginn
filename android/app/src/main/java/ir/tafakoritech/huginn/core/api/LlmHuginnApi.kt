package ir.tafakoritech.huginn.core.api

import ir.tafakoritech.huginn.data.model.CallScenarioRequest
import ir.tafakoritech.huginn.data.model.CallScenarioResponse
import retrofit2.http.Body
import retrofit2.http.POST
import retrofit2.http.Path

/**
 * Retrofit contract for LLM-backed endpoints that can legitimately take far
 * longer than a normal API call (call-scenario generation waits on the LLM
 * provider, which can run 30-180+ seconds).
 *
 * These endpoints MUST be called through a dedicated long-running OkHttp client
 * (see NetworkModule.provideLlmClient) whose read/call timeouts comfortably
 * exceed the backend's LLM timeout — otherwise the shared 30s client would
 * cancel the request mid-generation and the backend would report a spurious
 * `context canceled` failure.
 *
 * Mirrors backend/internal/httpapi (see backend/api/openapi.yaml).
 */
interface LlmHuginnApi {

    /** Suggested call scenario for a lead (no interaction is created). */
    @POST("api/v1/leads/{id}/call-scenario")
    suspend fun callScenario(
        @Path("id") leadId: String,
        @Body request: CallScenarioRequest,
    ): CallScenarioResponse
}
