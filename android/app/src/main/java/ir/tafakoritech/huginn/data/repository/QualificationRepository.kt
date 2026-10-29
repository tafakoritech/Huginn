package ir.tafakoritech.huginn.data.repository

import ir.tafakoritech.huginn.core.api.ApiCaller
import ir.tafakoritech.huginn.core.api.HuginnApi
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.runCatchingAppError
import ir.tafakoritech.huginn.data.model.DisqualifyRequest
import ir.tafakoritech.huginn.data.model.DisqualifyResponse
import ir.tafakoritech.huginn.data.model.QualifyRequest
import ir.tafakoritech.huginn.data.model.QualifyResponse
import kotlinx.serialization.json.Json
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Qualification and disqualification. Critical actions require confirmed
 * backend success; they are never optimistic locally.
 */
@Singleton
class QualificationRepository @Inject constructor(
    private val api: HuginnApi,
    private val json: Json,
) {

    /**
     * Qualify a lead. [checks] uses the UI checklist keys; they are translated
     * to the backend's flat request booleans before sending.
     */
    suspend fun qualify(leadId: String, checks: Map<String, Boolean>, reason: String?): AppResult<QualifyResponse> =
        runCatchingAppError {
            ApiCaller.call(json) { api.qualify(leadId, buildQualifyRequest(checks, reason)) }
        }

    suspend fun disqualify(leadId: String, reason: String, explanation: String?): AppResult<DisqualifyResponse> =
        runCatchingAppError { ApiCaller.call(json) { api.disqualify(leadId, DisqualifyRequest(reason = reason, explanation = explanation)) } }
}

private fun buildQualifyRequest(checks: Map<String, Boolean>, reason: String?): QualifyRequest {
    // Map the UI checklist keys to the backend's qualification input names.
    fun enabled(key: String): Boolean = checks[key] == true
    return QualifyRequest(
        businessRelevant = enabled(ChecklistKey.BUSINESS_RELEVANT),
        contactInfoValid = enabled(ChecklistKey.CONTACT_VALID),
        meaningfulContactOccurred = enabled(ChecklistKey.MEANINGFUL_CONTACT),
        needOrInterestExists = enabled(ChecklistKey.NEED_OR_INTEREST),
        decisionMakerIdentified = enabled(ChecklistKey.DECISION_MAKER),
        nextSalesActionExists = enabled(ChecklistKey.NEXT_ACTION),
        reason = reason,
    )
}

/** UI checklist keys (backend is the authority on the rules; names here are display-only). */
object ChecklistKey {
    const val BUSINESS_RELEVANT = "business_is_relevant"
    const val CONTACT_VALID = "contact_information_valid"
    const val MEANINGFUL_CONTACT = "meaningful_contact_occurred"
    const val NEED_OR_INTEREST = "need_or_interest_exists"
    const val DECISION_MAKER = "decision_maker_or_influencer_identified"
    const val NEXT_ACTION = "next_sales_action_exists"
}
