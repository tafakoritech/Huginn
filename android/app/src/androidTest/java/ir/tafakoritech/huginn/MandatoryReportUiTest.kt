package ir.tafakoritech.huginn

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import ir.tafakoritech.huginn.core.theme.HuginnTheme
import ir.tafakoritech.huginn.data.model.ReportDraft
import ir.tafakoritech.huginn.data.model.ReportOutcome
import ir.tafakoritech.huginn.data.model.ReportValidationError
import ir.tafakoritech.huginn.feature.reports.MandatoryReportForm
import ir.tafakoritech.huginn.feature.reports.MandatoryReportUiState
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import androidx.appcompat.app.AppCompatDelegate
import androidx.core.os.LocaleListCompat
import java.util.Locale

/** Mandatory report UI: outcome selection drives the conditional fields. */
@RunWith(AndroidJUnit4::class)
class MandatoryReportUiTest {

    @get:Rule
    val compose = createComposeRule()

    @Before
    fun setUpLocale() {
        AppCompatDelegate.setApplicationLocales(LocaleListCompat.create(Locale("fa")))
    }

    private fun render(
        initialOutcome: String? = null,
        onSubmitted: () -> Unit = {},
    ) {
        var draft by mutableStateOf(ReportDraft(outcome = initialOutcome))
        var validationErrors by mutableStateOf(emptySet<ReportValidationError>())
        compose.setContent {
            HuginnTheme {
                MandatoryReportForm(
                    state = MandatoryReportUiState(
                        draft = draft,
                        validationErrors = validationErrors,
                    ),
                    onOutcomeSelected = { outcome ->
                        draft = ReportDraft(outcome = outcome)
                    },
                    onDraftChange = { d -> draft = d },
                    onSubmit = {
                        validationErrors = ir.tafakoritech.huginn.data.model.ReportValidator.validate(draft).toSet()
                        if (validationErrors.isEmpty()) onSubmitted()
                    },
                )
            }
        }
    }

    @Test
    fun noAnswer_outcome_shows_only_call_again_and_note() {
        render(initialOutcome = ReportOutcome.NO_ANSWER)
        compose.onNodeWithText("زمان تماس بعدی").assertIsDisplayed()
        compose.onNodeWithText("یادداشت").assertIsDisplayed()
    }

    @Test
    fun interested_outcome_shows_interest_and_next_action() {
        render(initialOutcome = ReportOutcome.INTERESTED)
        compose.onNodeWithText("میزان علاقه").assertIsDisplayed()
        compose.onNodeWithText("میزان نیاز").assertIsDisplayed()
        compose.onNodeWithText("اقدام بعدی").assertIsDisplayed()
        compose.onNodeWithText("خلاصه مکالمه").assertIsDisplayed()
    }

    @Test
    fun summary_field_offers_voice_input_button() {
        render(initialOutcome = ReportOutcome.INTERESTED)
        compose.onNodeWithText("خلاصه مکالمه").assertIsDisplayed()
        compose.onNodeWithContentDescription("تایپ صوتی").assertIsDisplayed()
    }

    @Test
    fun main_objection_field_offers_voice_input_button() {
        render(initialOutcome = ReportOutcome.NOT_RELEVANT)
        compose.onNodeWithText("مانع اصلی").assertIsDisplayed()
        compose.onNodeWithContentDescription("تایپ صوتی").assertIsDisplayed()
    }

    @Test
    fun callback_requested_requires_time() {
        var submitted = false
        render(initialOutcome = ReportOutcome.CALL_BACK_REQUESTED, onSubmitted = { submitted = true })
        compose.onNodeWithText("ثبت گزارش").performClick()
        assertTrue(!submitted) // validation prevents submit
        compose.onNodeWithText("زمان پیگیری").assertIsDisplayed()
    }

    @Test
    fun report_form_always_shows_outcome_and_submit() {
        var draft by mutableStateOf(ReportDraft())
        compose.setContent {
            HuginnTheme {
                MandatoryReportForm(
                    state = MandatoryReportUiState(draft = draft),
                    onOutcomeSelected = { draft = ReportDraft(outcome = it) },
                    onDraftChange = { draft = it },
                    onSubmit = {},
                )
            }
        }
        compose.onNodeWithText("نتیجه ارتباط").assertIsDisplayed()
        compose.onNodeWithText("ثبت گزارش").assertIsDisplayed()
    }
}
