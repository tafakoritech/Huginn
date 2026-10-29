package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.design.UiMappers
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * Localization regression tests for the timeline event-type and score
 * component-key mappers. When Persian is active, the UI must never render raw
 * backend machine keys ("lead.created", "business_fit") or English prose.
 */
class UiMappersLocalizationTest {

    @Test
    fun timeline_event_types_map_to_localized_resources() {
        val expected = mapOf(
            "lead.created" to R.string.timeline_lead_created,
            "lead.updated" to R.string.timeline_lead_updated,
            "lead.assigned" to R.string.timeline_lead_assigned,
            "lead.qualified" to R.string.timeline_lead_qualified,
            "lead.disqualified" to R.string.timeline_lead_disqualified,
            "lead.exported" to R.string.timeline_lead_exported,
            "note.created" to R.string.timeline_note_created,
            "report.submitted" to R.string.timeline_report_submitted,
            "report.voided" to R.string.timeline_report_voided,
            "followup.created" to R.string.timeline_followup_created,
            "interaction.created" to R.string.timeline_interaction_created,
            "contact.created" to R.string.timeline_contact_created,
            "call_scenario.generated" to R.string.timeline_call_scenario_generated,
            "project.updated" to R.string.timeline_project_updated,
        )
        expected.forEach { (key, res) ->
            assertEquals("event type '$key'", res, UiMappers.timelineEventLabelRes(key))
        }
    }

    @Test
    fun unknown_timeline_event_type_returns_null_not_a_raw_key() {
        assertNull(UiMappers.timelineEventLabelRes("some.future.event"))
        assertNull(UiMappers.timelineEventLabelRes(null))
    }

    @Test
    fun score_component_keys_map_to_localized_resources() {
        val expected = mapOf(
            "business_fit" to R.string.score_business_fit,
            "contactability" to R.string.score_contactability,
            "scale" to R.string.score_scale,
            "engagement" to R.string.score_engagement,
            "authority" to R.string.score_authority,
            "need" to R.string.score_need,
            "timing" to R.string.score_timing,
        )
        expected.forEach { (key, res) ->
            assertEquals("score component '$key'", res, UiMappers.scoreComponentLabelRes(key))
        }
    }

    @Test
    fun unknown_score_component_key_returns_null_not_a_raw_key() {
        assertNull(UiMappers.scoreComponentLabelRes("has_phone"))
        assertNull(UiMappers.scoreComponentLabelRes(null))
    }

    // ---- Report enums: the Reports screen must never render raw backend codes ----

    @Test
    fun report_outcomes_map_to_localized_resources() {
        val expected = mapOf(
            "NO_ANSWER" to R.string.outcome_no_answer,
            "BUSY" to R.string.outcome_busy,
            "WRONG_NUMBER" to R.string.outcome_wrong_number,
            "ANSWERED" to R.string.outcome_answered,
            "CALL_BACK_REQUESTED" to R.string.outcome_callback_requested,
            "NOT_RELEVANT" to R.string.outcome_not_relevant,
            "INTERESTED" to R.string.outcome_interested,
            "HIGHLY_INTERESTED" to R.string.outcome_highly_interested,
            "DISQUALIFIED" to R.string.outcome_disqualified,
        )
        expected.forEach { (key, res) ->
            assertEquals("outcome '$key'", res, UiMappers.outcomeLabelRes(key))
        }
    }

    @Test
    fun interest_levels_map_to_localized_resources() {
        val expected = mapOf(
            "NONE" to R.string.interest_none,
            "LOW" to R.string.interest_low,
            "MEDIUM" to R.string.interest_medium,
            "HIGH" to R.string.interest_high,
        )
        expected.forEach { (key, res) ->
            assertEquals("interest '$key'", res, UiMappers.interestLabelRes(key))
        }
    }

    @Test
    fun channels_map_to_localized_resources() {
        val expected = mapOf(
            "CALL" to R.string.channel_call,
            "SMS" to R.string.channel_sms,
            "WHATSAPP" to R.string.channel_whatsapp,
            "TELEGRAM" to R.string.channel_telegram,
            "BALE" to R.string.channel_bale,
            "EITAA" to R.string.channel_eitaa,
            "RUBIKA" to R.string.channel_rubika,
            "EMAIL" to R.string.channel_email,
        )
        expected.forEach { (key, res) ->
            assertEquals("channel '$key'", res, UiMappers.channelLabelRes(key))
        }
    }

    @Test
    fun next_actions_map_to_localized_resources() {
        val expected = mapOf(
            "CALL_AGAIN" to R.string.next_action_call_again,
            "SEND_INFORMATION" to R.string.next_action_send_info,
            "SEND_SAMPLE" to R.string.next_action_send_sample,
            "FOLLOW_UP" to R.string.next_action_follow_up,
            "QUALIFY" to R.string.next_action_qualify,
            "DISQUALIFY" to R.string.next_action_disqualify,
            "TRANSFER_TO_MUNINN" to R.string.next_action_transfer,
            "NONE" to R.string.next_action_none,
        )
        expected.forEach { (key, res) ->
            assertEquals("next action '$key'", res, UiMappers.nextActionLabelRes(key))
        }
    }

    @Test
    fun unknown_report_enums_never_degrade_to_a_raw_code() {
        assertEquals(R.string.unknown, UiMappers.outcomeLabelRes(null))
        assertEquals(R.string.unknown, UiMappers.outcomeLabelRes("CUSTOM_EVENT"))
        assertEquals(R.string.unknown, UiMappers.interestLabelRes(null))
        assertEquals(R.string.unknown, UiMappers.channelLabelRes("SLACK"))
    }
}
