package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.data.model.ReportDraft
import ir.tafakoritech.huginn.data.model.ReportOutcome
import ir.tafakoritech.huginn.data.model.ReportValidationError
import ir.tafakoritech.huginn.data.model.ReportValidator
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ReportValidatorTest {

    @Test
    fun `outcome is required`() {
        val errors = ReportValidator.validate(ReportDraft())
        assertTrue(errors.contains(ReportValidationError.OUTCOME_REQUIRED))
    }

    @Test
    fun `no answer requires a note or followup time`() {
        val errors = ReportValidator.validate(ReportDraft(outcome = ReportOutcome.NO_ANSWER))
        assertTrue(errors.contains(ReportValidationError.SUMMARY_REQUIRED))
        val ok = ReportValidator.validate(
            ReportDraft(outcome = ReportOutcome.NO_ANSWER, summary = "زنگ خورد ولی جواب نداد"),
        )
        assertEquals(emptyList<ReportValidationError>(), ok)
    }

    @Test
    fun `callback requested requires followup time`() {
        val errors = ReportValidator.validate(ReportDraft(outcome = ReportOutcome.CALL_BACK_REQUESTED))
        assertTrue(errors.contains(ReportValidationError.FOLLOW_UP_TIME_REQUIRED))
        val ok = ReportValidator.validate(
            ReportDraft(outcome = ReportOutcome.CALL_BACK_REQUESTED, followUpAt = "2025-01-01T10:00:00Z"),
        )
        assertEquals(emptyList<ReportValidationError>(), ok)
    }

    @Test
    fun `interested requires interest and next action`() {
        val errors = ReportValidator.validate(ReportDraft(outcome = ReportOutcome.INTERESTED))
        assertTrue(errors.contains(ReportValidationError.INTEREST_REQUIRED))
        assertTrue(errors.contains(ReportValidationError.NEXT_ACTION_REQUIRED))
        val ok = ReportValidator.validate(
            ReportDraft(
                outcome = ReportOutcome.INTERESTED,
                interestLevel = "HIGH",
                nextAction = "SEND_INFORMATION",
            ),
        )
        assertEquals(emptyList<ReportValidationError>(), ok)
    }

    @Test
    fun `answered has no hard requirements`() {
        val errors = ReportValidator.validate(ReportDraft(outcome = ReportOutcome.ANSWERED))
        assertEquals(emptyList<ReportValidationError>(), errors)
    }

    @Test
    fun `not relevant requires summary or objection`() {
        val errors = ReportValidator.validate(ReportDraft(outcome = ReportOutcome.NOT_RELEVANT))
        assertTrue(errors.contains(ReportValidationError.SUMMARY_REQUIRED))
        val ok = ReportValidator.validate(
            ReportDraft(outcome = ReportOutcome.NOT_RELEVANT, mainObjection = "خارج از حوزه فعالیت"),
        )
        assertEquals(emptyList<ReportValidationError>(), ok)
    }

    @Test
    fun `toRequest strips blank optional fields`() {
        val draft = ReportDraft(
            outcome = ReportOutcome.INTERESTED,
            interestLevel = "MEDIUM",
            nextAction = "FOLLOW_UP",
            summary = "   ",
            decisionMakerName = "",
        )
        val request = draft.toRequest()
        assertEquals("INTERESTED", request.outcome)
        assertEquals("MEDIUM", request.interestLevel)
        assertEquals(null, request.summary)
        assertEquals(null, request.decisionMakerName)
    }
}
