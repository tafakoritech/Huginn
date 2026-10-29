package ir.tafakoritech.huginn.data.repository

import ir.tafakoritech.huginn.core.api.ApiCaller
import ir.tafakoritech.huginn.core.api.HuginnApi
import ir.tafakoritech.huginn.core.api.LlmHuginnApi
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.runCatchingAppError
import ir.tafakoritech.huginn.data.model.CallScenario
import ir.tafakoritech.huginn.data.model.CallScenarioRequest
import ir.tafakoritech.huginn.data.model.CallScenarioResponse
import ir.tafakoritech.huginn.data.model.Contact
import ir.tafakoritech.huginn.data.model.ContactActionRequest
import ir.tafakoritech.huginn.data.model.ContactActionResponse
import kotlinx.serialization.json.Json
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Communication interaction repository. Every outbound attempt is recorded
 * server-side before any external app is launched (contact-actions endpoint).
 * The backend is also responsible for generating suggested messages/deep links
 * and pre-call scenarios; the client only sends the lead/contact identifiers.
 *
 * Call-scenario generation waits on the backend's LLM and can legitimately take
 * minutes, so it goes through [LlmHuginnApi] (dedicated long-running client)
 * instead of the shared 30s [HuginnApi] client.
 */
@Singleton
class InteractionRepository @Inject constructor(
    private val api: HuginnApi,
    private val llmApi: LlmHuginnApi,
    private val json: Json,
) {

    suspend fun startAction(
        leadId: String,
        contact: Contact,
        channel: String,
        objective: String?,
    ): AppResult<ContactActionResponse> = runCatchingAppError {
        ApiCaller.call(json) {
            api.contactAction(
                leadId,
                ContactActionRequest(contactId = contact.id, channel = channel, objective = objective),
            )
        }
    }

    /** Fetches a backend-generated call scenario for a lead (no interaction created). */
    suspend fun callScenario(
        leadId: String,
        contact: Contact?,
    ): AppResult<CallScenarioResponse> = runCatchingAppError {
        ApiCaller.call(json) {
            llmApi.callScenario(
                leadId,
                CallScenarioRequest(contactId = contact?.id),
            )
        }
    }

    /** Lists all scenarios archived for a lead. */
    suspend fun getCallScenarios(leadId: String): AppResult<List<CallScenario>> = runCatchingAppError {
        ApiCaller.call(json) {
            api.getCallScenarios(leadId)
        }
    }

    /** Creates a manual scenario for a lead. */
    suspend fun createCallScenario(leadId: String, scenario: CallScenario): AppResult<CallScenario> = runCatchingAppError {
        ApiCaller.call(json) {
            api.createCallScenario(leadId, scenario)
        }
    }

    /** Fetches a single scenario. */
    suspend fun getCallScenario(leadId: String, scenarioId: String): AppResult<CallScenario> = runCatchingAppError {
        ApiCaller.call(json) {
            api.getCallScenario(leadId, scenarioId)
        }
    }

    /** Updates an existing scenario. */
    suspend fun updateCallScenario(leadId: String, scenario: CallScenario): AppResult<CallScenario> = runCatchingAppError {
        val scenarioId = requireNotNull(scenario.id) { "Scenario ID is required for update" }
        ApiCaller.call(json) {
            api.updateCallScenario(leadId, scenarioId, scenario)
        }
    }

    /** Deletes a scenario. */
    suspend fun deleteCallScenario(leadId: String, scenarioId: String): AppResult<Unit> = runCatchingAppError {
        ApiCaller.call(json) {
            api.deleteCallScenario(leadId, scenarioId)
        }
    }
}
