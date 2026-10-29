package ir.tafakoritech.huginn

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.width
import androidx.compose.ui.Modifier
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsNotDisplayed
import androidx.compose.ui.test.getBoundsInRoot
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.onRoot
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.swipeRight
import androidx.compose.ui.unit.dp
import androidx.test.ext.junit.runners.AndroidJUnit4
import ir.tafakoritech.huginn.core.theme.HuginnTheme
import ir.tafakoritech.huginn.core.theme.ThemeMode
import ir.tafakoritech.huginn.feature.workqueue.WorkQueueTab
import ir.tafakoritech.huginn.feature.workqueue.WorkQueueTabRow
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import androidx.appcompat.app.AppCompatDelegate
import androidx.core.os.LocaleListCompat
import java.util.Locale

/**
 * UI tests for the horizontally scrollable Work Queue tab row.
 *
 * Covers the visual regression where long Persian labels ("گزارش‌های در
 * انتظار", "عقب‌افتاده") wrapped onto multiple lines inside a fixed TabRow.
 */
@RunWith(AndroidJUnit4::class)
class WorkQueueTabRowUiTest {

    @get:Rule
    val compose = createComposeRule()

    private val allLabels = listOf("همه", "فوری", "امروز", "عقب‌افتاده", "گزارش‌های در انتظار", "تماس", "پیام", "ایمیل")

    @Before
    fun setUpLocale() {
        // Pin the app locale to Persian so label assertions are deterministic.
        AppCompatDelegate.setApplicationLocales(LocaleListCompat.create(Locale("fa")))
    }

    private fun render(widthDp: Int, selected: WorkQueueTab = WorkQueueTab.ALL, onSelect: (WorkQueueTab) -> Unit = {}) {
        compose.setContent {
            HuginnTheme {
                Box(Modifier.width(widthDp.dp)) {
                    WorkQueueTabRow(selected = selected, onSelect = onSelect)
                }
            }
        }
    }

    /** A single-line label is as tall as the short reference label "همه"; a wrapped one is ~2x. */
    private fun assertSingleLine(longLabel: String) {
        val reference = compose.onNodeWithText("همه").getBoundsInRoot()
        val long = compose.onNodeWithText(longLabel).getBoundsInRoot()
        val referenceHeight = reference.bottom - reference.top
        val longHeight = long.bottom - long.top
        assertTrue(
            "label '$longLabel' wrapped onto multiple lines ($longHeight vs $referenceHeight)",
            longHeight <= referenceHeight + 2.dp,
        )
    }

    @Test
    fun all_tab_labels_render() {
        render(411)
        allLabels.forEach { compose.onNodeWithText(it).assertIsDisplayed() }
    }

    @Test
    fun long_labels_do_not_wrap_at_360dp() {
        render(360)
        assertSingleLine("گزارش‌های در انتظار")
        assertSingleLine("عقب‌افتاده")
    }

    @Test
    fun long_labels_do_not_wrap_at_320dp() {
        render(320)
        assertSingleLine("گزارش‌های در انتظار")
        assertSingleLine("عقب‌افتاده")
    }

    @Test
    fun long_labels_do_not_wrap_at_411dp() {
        render(411)
        assertSingleLine("گزارش‌های در انتظار")
        assertSingleLine("عقب‌افتاده")
    }

    @Test
    fun row_scrolls_horizontally_when_tabs_overflow() {
        render(320)
        // With 8 content-sized tabs at 320dp, the last tab is off-screen (left in RTL).
        compose.onNodeWithText("ایمیل").assertIsNotDisplayed()
        // Swiping right reveals the leftmost content in RTL.
        compose.onRoot().performTouchInput { swipeRight() }
        compose.onNodeWithText("ایمیل").assertIsDisplayed()
    }

    @Test
    fun selecting_a_tab_reports_the_tab() {
        var selected: WorkQueueTab? = null
        render(360, onSelect = { selected = it })
        compose.onNodeWithText("فوری").performClick()
        assertTrue(selected == WorkQueueTab.URGENT)
    }

    @Test
    fun labels_render_in_dark_mode() {
        compose.setContent {
            HuginnTheme(themeMode = ThemeMode.DARK) {
                Box(Modifier.width(360.dp)) {
                    WorkQueueTabRow(selected = WorkQueueTab.ALL, onSelect = {})
                }
            }
        }
        compose.onNodeWithText("همه").assertIsDisplayed()
        compose.onNodeWithText("گزارش‌های در انتظار").assertIsDisplayed()
    }
}
