package ir.tafakoritech.huginn.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Deep link target produced by the backend (messaging/deeplink.go). */
@Serializable
data class DeepLink(
    val channel: String = "",
    val type: String = "",
    val target: String = "",
    val text: String? = null,
    val username: String? = null,
    val note: String? = null,
) {
    companion object {
        const val TYPE_TEL = "tel"
        const val TYPE_SMS = "sms"
        const val TYPE_WA_ME = "wa.me"
        const val TYPE_USERNAME = "username"
        const val TYPE_MAILTO = "mailto"
        const val TYPE_CONTACT_SAVE_REQUIRED = "CONTACT_SAVE_REQUIRED"
        const val TYPE_OTHER = "other"

        /** Build a wa.me link client-side only when the backend provides none. */
        fun waMe(mobile: String): String {
            val digits = mobile.filter { it.isDigit() }
            return "https://wa.me/$digits"
        }
    }
}

/** Request to record a communication attempt: POST /leads/{id}/contact-actions. */
@Serializable
data class ContactActionRequest(
    @SerialName("contact_id") val contactId: String,
    val channel: String,
    val objective: String? = null,
)

@Serializable
data class ContactActionResponse(
    @SerialName("interaction_id") val interactionId: String,
    @SerialName("contact_id") val contactId: String? = null,
    val channel: String = "",
    val phone: String? = null,
    @SerialName("deep_link") val deepLink: DeepLink? = null,
    @SerialName("suggested_message") val suggestedMessage: String? = null,
    @SerialName("suggested_subject") val suggestedSubject: String? = null,
    val workflow: WorkflowDto = WorkflowDto(),
)

/** Request for POST /leads/{id}/call-scenario. */
@Serializable
data class CallScenarioRequest(
    @SerialName("contact_id") val contactId: String? = null,
)

/** A practical, spoken phone-call script generated or edited by the user/backend. */
@Serializable
data class CallScenario(
    val id: String? = null,
    @SerialName("lead_id") val leadId: String? = null,
    @SerialName("contact_id") val contactId: String? = null,
    @SerialName("user_id") val userId: String? = null,
    @SerialName("opening_sentence") val openingSentence: String = "",
    val introduction: String = "",
    @SerialName("reason_for_call") val reasonForCall: String = "",
    @SerialName("value_proposition") val valueProposition: String = "",
    @SerialName("discovery_questions") val discoveryQuestions: List<String> = emptyList(),
    @SerialName("likely_objections") val likelyObjections: List<String> = emptyList(),
    @SerialName("objection_responses") val objectionResponses: List<String> = emptyList(),
    @SerialName("recommended_next_step") val recommendedNextStep: String = "",
    @SerialName("closing_line") val closingLine: String = "",
    @SerialName("is_generated") val isGenerated: Boolean = false,
    @SerialName("generation_status") val generationStatus: String? = null,
    @SerialName("fallback_used") val fallbackUsed: Boolean = false,
    @SerialName("generation_error") val generationError: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class CallScenarioResponse(
    @SerialName("lead_id") val leadId: String = "",
    val scenario: CallScenario = CallScenario(),
)
