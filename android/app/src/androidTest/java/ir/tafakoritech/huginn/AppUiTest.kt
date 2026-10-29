package ir.tafakoritech.huginn

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.test.ext.junit.runners.AndroidJUnit4
import ir.tafakoritech.huginn.core.design.LeadScoreBadge
import ir.tafakoritech.huginn.core.design.LeadStatusBadge
import ir.tafakoritech.huginn.core.theme.HuginnTheme
import ir.tafakoritech.huginn.core.util.AppError
import ir.tafakoritech.huginn.data.model.Contact
import ir.tafakoritech.huginn.data.model.FollowUp
import ir.tafakoritech.huginn.data.model.InteractionChannel
import ir.tafakoritech.huginn.data.model.Lead
import ir.tafakoritech.huginn.data.model.MeStats
import ir.tafakoritech.huginn.data.model.NextActionItem
import ir.tafakoritech.huginn.data.model.NextActionPriority
import ir.tafakoritech.huginn.data.model.NextActionSource
import ir.tafakoritech.huginn.data.model.NextActionType
import ir.tafakoritech.huginn.feature.auth.LoginScreen
import ir.tafakoritech.huginn.feature.followups.FollowUpCard
import ir.tafakoritech.huginn.feature.home.HomeContent
import ir.tafakoritech.huginn.feature.home.HomeUiState
import ir.tafakoritech.huginn.feature.leads.LeadCard
import ir.tafakoritech.huginn.feature.messaging.ChannelItem
import ir.tafakoritech.huginn.feature.messaging.CommunicationSheet
import ir.tafakoritech.huginn.feature.messaging.SuggestedMessageSheet
import ir.tafakoritech.huginn.feature.messaging.channelIcon
import ir.tafakoritech.huginn.feature.qualification.QualifyChecklist
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import androidx.appcompat.app.AppCompatDelegate
import androidx.core.os.LocaleListCompat
import java.util.Locale

@RunWith(AndroidJUnit4::class)
class AppUiTest {

    @get:Rule
    val compose = createComposeRule()

    @Before
    fun setUpLocale() {
        // Pin the app locale to Persian so string assertions are deterministic.
        AppCompatDelegate.setApplicationLocales(LocaleListCompat.create(Locale("fa")))
    }

    // ---- Login ----

    @Test
    fun login_validates_and_submits_credentials() {
        var submittedUsername: String? = null
        var submittedPassword: String? = null
        compose.setContent {
            HuginnTheme {
                LoginScreen(
                    busy = false,
                    error = null,
                    onLogin = { u, p ->
                        submittedUsername = u
                        submittedPassword = p
                    },
                )
            }
        }
        compose.onNodeWithText("نام کاربری").performTextInput("agent")
        compose.onNodeWithText("رمز عبور").performTextInput("secret")
        compose.onNodeWithText("ورود").performClick()
        assertTrue(submittedUsername == "agent" && submittedPassword == "secret")
    }

    @Test
    fun login_shows_invalid_credentials_error() {
        compose.setContent {
            HuginnTheme {
                LoginScreen(busy = false, error = AppError.Unauthorized, onLogin = { _, _ -> })
            }
        }
        compose.onNodeWithText("نام کاربری یا رمز عبور نادرست است.").assertIsDisplayed()
    }

    // ---- Home ----

    @Test
    fun home_shows_greeting_stats_and_next_task() {
        compose.setContent {
            HuginnTheme {
                HomeContent(
                    state = HomeUiState(
                        loading = false,
                        userName = "علی محمدی",
                        stats = MeStats(todayLeads = 4, todayCalls = 2, pendingReports = 1),
                        nextActions = listOf(
                            NextActionItem(
                                id = "follow_up:fu-1",
                                sourceType = NextActionSource.FOLLOW_UP,
                                sourceId = "fu-1",
                                leadId = "lead-1",
                                companyName = "عروسک‌سازی پارس",
                                actionType = NextActionType.CALL,
                                title = "تماس با عروسک‌سازی پارس",
                                reason = "پیگیری وعده داده‌شده",
                                priority = NextActionPriority.URGENT,
                                priorityScore = 92,
                                overdue = true,
                                ctaLabel = "تماس",
                                explainability = listOf("از زمان پیگیری گذشته است"),
                            ),
                        ),
                    ),
                    onOpenLead = {},
                    onOpenReport = {},
                    onOpenWorkQueue = {},
                    onSnooze = { _, _ -> },
                    onComplete = {},
                )
            }
        }
        compose.onNodeWithText("سلام، علی محمدی").assertIsDisplayed()
        compose.onNodeWithText("سرنخ‌های امروز").assertIsDisplayed()
        compose.onNodeWithText("۴").assertIsDisplayed()
    }

    // ---- Lead list ----

    @Test
    fun lead_card_shows_essential_fields() {
        compose.setContent {
            HuginnTheme {
                LeadCard(
                    lead = Lead(
                        id = "lead-1",
                        companyName = "عروسک‌سازی پارس",
                        primaryContactName = "محمد رضایی",
                        primaryContactRole = "مدیر مجموعه",
                        city = "تهران",
                        businessType = "تولیدکننده",
                        status = "CONTACTING",
                        score = 78,
                    ),
                    onClick = {},
                )
            }
        }
        compose.onNodeWithText("عروسک‌سازی پارس").assertIsDisplayed()
        compose.onNodeWithText("محمد رضایی — مدیر مجموعه").assertIsDisplayed()
        compose.onNodeWithText("تهران • تولیدکننده").assertIsDisplayed()
    }

    @Test
    fun score_and_status_badges_render() {
        compose.setContent {
            HuginnTheme {
                androidx.compose.foundation.layout.Column {
                    LeadScoreBadge(score = 82)
                    LeadStatusBadge(status = "QUALIFIED")
                }
            }
        }
        compose.onNodeWithText("امتیاز: ۸۲").assertIsDisplayed()
        compose.onNodeWithText("واجد شرایط").assertIsDisplayed()
    }

    // ---- Communication sheet ----

    @Test
    fun communication_sheet_selects_channel() {
        var selected: String? = null
        compose.setContent {
            HuginnTheme {
                CommunicationSheet(
                    contactName = "محمد رضایی",
                    channels = listOf(
                        ChannelItem(InteractionChannel.SMS, channelIcon(InteractionChannel.SMS), enabled = true),
                        ChannelItem(InteractionChannel.WHATSAPP, channelIcon(InteractionChannel.WHATSAPP), enabled = false),
                    ),
                    onSelect = { selected = it },
                    onDismiss = {},
                )
            }
        }
        compose.onNodeWithText("پیامک").performClick()
        assertTrue(selected == InteractionChannel.SMS)
    }

    @Test
    fun suggested_message_sheet_offers_edit_and_open() {
        var opened = false
        compose.setContent {
            HuginnTheme {
                SuggestedMessageSheet(
                    channel = InteractionChannel.WHATSAPP,
                    loading = false,
                    text = "سلام آقای رضایی، از شرکت تفکر پردازان مزاحم شدم.",
                    subject = null,
                    onTextChange = {},
                    onRetry = {},
                    onOpenMessenger = { opened = true },
                    onCancel = {},
                )
            }
        }
        compose.onNodeWithText("متن پیشنهادی").assertIsDisplayed()
        compose.onNodeWithText("باز کردن پیام‌رسان").performClick()
        assertTrue(opened)
    }

    // ---- Follow-up card ----

    @Test
    fun follow_up_card_shows_company_and_due_time() {
        var completed = false
        compose.setContent {
            HuginnTheme {
                FollowUpCard(
                    followUp = FollowUp(
                        id = "fu-1",
                        leadId = "lead-2",
                        companyName = "صنایع چوب نوین",
                        contactName = "امیر کریمی",
                        dueAt = java.time.LocalDateTime.now().plusDays(1)
                            .atZone(java.time.ZoneId.systemDefault())
                            .format(java.time.format.DateTimeFormatter.ISO_INSTANT),
                        type = "FOLLOW_UP",
                        status = "PENDING",
                    ),
                    onComplete = { completed = true },
                    onReopen = {},
                    onReschedule = {},
                    onCancel = {},
                    onOpenLead = {},
                )
            }
        }
        compose.onNodeWithText("صنایع چوب نوین").assertIsDisplayed()
        compose.onNodeWithText("انجام شد").performClick()
        assertTrue(completed)
    }

    // ---- Qualification checklist ----

    @Test
    fun qualification_checklist_toggles_items() {
        var toggled: String? = null
        compose.setContent {
            HuginnTheme {
                var checks by androidx.compose.runtime.remember {
                    mutableStateOf(mapOf("business_is_relevant" to false))
                }
                QualifyChecklist(
                    checks = checks,
                    onToggle = { key ->
                        toggled = key
                        checks = checks.toMutableMap().apply { this[key] = !(this[key] ?: false) }
                    },
                )
            }
        }
        compose.onNodeWithText("تناسب کسب‌وکار").performClick()
        assertTrue(toggled == "business_is_relevant")
    }
}
