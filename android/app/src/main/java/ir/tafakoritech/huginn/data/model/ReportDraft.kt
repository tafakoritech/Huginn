package ir.tafakoritech.huginn.data.model

/**
 * Mutable draft of a mandatory contact report. Kept in the ViewModel, the
 * conditional fields are filled depending on the chosen outcome.
 */
data class ReportDraft(
    var outcome: String? = null,
    var correctPersonReached: Boolean? = null,
    var decisionMakerIdentified: Boolean? = null,
    var decisionMakerName: String? = null,
    var decisionMakerRole: String? = null,
    var interestLevel: String? = null,
    var needLevel: String? = null,
    var purchaseCapacity: String? = null,
    var purchaseTiming: String? = null,
    var mainObjection: String? = null,
    var summary: String? = null,
    var nextAction: String? = null,
    var followUpAt: String? = null,
    var contactReached: Boolean? = null,
) {
    fun toRequest(): ReportSubmitRequest = ReportSubmitRequest(
        outcome = outcome.orEmpty(),
        contactReached = contactReached,
        correctPersonReached = correctPersonReached,
        decisionMakerIdentified = decisionMakerIdentified,
        decisionMakerName = decisionMakerName?.takeIf { it.isNotBlank() },
        decisionMakerRole = decisionMakerRole?.takeIf { it.isNotBlank() },
        interestLevel = interestLevel,
        needLevel = needLevel,
        estimatedPurchaseCapacity = purchaseCapacity?.takeIf { it.isNotBlank() },
        purchaseTiming = purchaseTiming?.takeIf { it.isNotBlank() },
        mainObjection = mainObjection?.takeIf { it.isNotBlank() },
        summary = summary?.takeIf { it.isNotBlank() },
        nextAction = nextAction,
        followUpAt = followUpAt,
    )
}

/** Validation failure reasons (stable keys used to look up UI messages). */
enum class ReportValidationError {
    OUTCOME_REQUIRED,
    SUMMARY_REQUIRED,
    FOLLOW_UP_TIME_REQUIRED,
    INTEREST_REQUIRED,
    NEXT_ACTION_REQUIRED,
}

/**
 * Outcome-dependent validation, mirroring the backend rules
 * (backend/internal/reports/reports.go Validate).
 */
object ReportValidator {

    fun validate(draft: ReportDraft): List<ReportValidationError> {
        val errors = mutableListOf<ReportValidationError>()
        val outcome = draft.outcome
        if (outcome.isNullOrBlank()) {
            errors += ReportValidationError.OUTCOME_REQUIRED
            return errors
        }
        when (outcome) {
            ReportOutcome.NO_ANSWER,
            ReportOutcome.BUSY,
            ReportOutcome.WRONG_NUMBER,
            -> {
                if (draft.summary.isNullOrBlank() && draft.followUpAt.isNullOrBlank()) {
                    errors += ReportValidationError.SUMMARY_REQUIRED
                }
            }
            ReportOutcome.NOT_RELEVANT,
            ReportOutcome.DISQUALIFIED,
            -> {
                if (draft.summary.isNullOrBlank() && draft.mainObjection.isNullOrBlank()) {
                    errors += ReportValidationError.SUMMARY_REQUIRED
                }
            }
            ReportOutcome.CALL_BACK_REQUESTED -> {
                if (draft.followUpAt.isNullOrBlank()) {
                    errors += ReportValidationError.FOLLOW_UP_TIME_REQUIRED
                }
            }
            ReportOutcome.INTERESTED,
            ReportOutcome.HIGHLY_INTERESTED,
            -> {
                if (draft.interestLevel.isNullOrBlank()) {
                    errors += ReportValidationError.INTEREST_REQUIRED
                }
                if (draft.nextAction.isNullOrBlank()) {
                    errors += ReportValidationError.NEXT_ACTION_REQUIRED
                }
            }
        }
        return errors
    }
}
