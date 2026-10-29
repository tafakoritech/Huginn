package ir.tafakoritech.huginn

import android.content.res.Configuration
import android.content.res.Resources
import java.util.Locale
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

/**
 * The Reports screen strings must be fully localized: Persian when the app
 * language is Persian, English otherwise. The empty-state text is exactly
 * "گزارشی نیست" in fa and "No reports" in en.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class ReportsLocalizationTest {

    private fun resourcesFor(language: String): Resources {
        val config = Configuration()
        config.setLocale(Locale(language))
        return RuntimeEnvironment.getApplication()
            .createConfigurationContext(config).resources
    }

    private fun fa() = resourcesFor("fa")
    private fun en() = resourcesFor("en")

    @Test
    fun `fa empty state is exactly گزاری نیست`() {
        assertEquals("گزارشی نیست", fa().getString(R.string.no_reports))
    }

    @Test
    fun `en empty state is No reports`() {
        assertEquals("No reports", en().getString(R.string.no_reports))
    }

    @Test
    fun `fa report card labels are Persian`() {
        val res = fa()
        assertEquals("نتیجه:", res.getString(R.string.report_outcome_label_short))
        assertEquals("علاقه:", res.getString(R.string.report_interest_label_short))
        assertEquals("ایراد:", res.getString(R.string.report_objection_label))
        assertEquals("اقدام بعدی:", res.getString(R.string.report_next_action_label))
        assertEquals("خطا در دریافت گزارش‌ها. دوباره تلاش کنید.", res.getString(R.string.reports_error))
        assertEquals("تماس", res.getString(R.string.channel_call))
        assertEquals("متوسط", res.getString(R.string.interest_medium))
        assertEquals("علاقه‌مند", res.getString(R.string.outcome_interested))
        assertEquals("ارسال معرفی", res.getString(R.string.next_action_send_info))
    }

    @Test
    fun `en report card labels are English`() {
        val res = en()
        assertEquals("Outcome:", res.getString(R.string.report_outcome_label_short))
        assertEquals("Interest:", res.getString(R.string.report_interest_label_short))
        assertEquals("Objection:", res.getString(R.string.report_objection_label))
        assertEquals("Next action:", res.getString(R.string.report_next_action_label))
        assertEquals("Call", res.getString(R.string.channel_call))
        assertEquals("Medium", res.getString(R.string.interest_medium))
        assertEquals("Interested", res.getString(R.string.outcome_interested))
    }

    @Test
    fun `fa strings never fall back to english or raw enums`() {
        val faRes = fa()
        // Persian file must define these (not inherit a raw english/machine value).
        assertFalse("no_reports must not contain english", faRes.getString(R.string.no_reports).contains("No reports"))
        assertFalse("channel_call must not be the raw enum CALL", faRes.getString(R.string.channel_call).contains("CALL"))
        assertFalse("interest_medium must not be the raw enum MEDIUM", faRes.getString(R.string.interest_medium).contains("MEDIUM"))
        assertFalse("outcome_interested must not be the raw enum INTERESTED", faRes.getString(R.string.outcome_interested).contains("INTERESTED"))
        assertTrue("fa no_reports should be the Persian empty state", faRes.getString(R.string.no_reports).isNotBlank())
    }
}
