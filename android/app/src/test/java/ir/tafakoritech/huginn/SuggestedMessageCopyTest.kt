package ir.tafakoritech.huginn

import android.content.ClipboardManager
import android.content.Context
import ir.tafakoritech.huginn.core.util.copyToClipboard
import ir.tafakoritech.huginn.feature.messaging.fullSuggestedMessage
import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

/** Suggested-message copy helper: Unicode-safe clipboard writes and composition. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class SuggestedMessageCopyTest {

    @Test
    fun `full message with subject prepends a blank line separator`() {
        assertEquals("موضوع پیام\n\nمتن پیام", fullSuggestedMessage("موضوع پیام", "متن پیام"))
    }

    @Test
    fun `full message without subject is just the body`() {
        assertEquals("متن پیام", fullSuggestedMessage(null, "متن پیام"))
        assertEquals("", fullSuggestedMessage(null, ""))
    }

    @Test
    fun `copy to clipboard preserves persian unicode exactly`() {
        val context = RuntimeEnvironment.getApplication()
        val message = "سلام آقای رضایی، از شرکت تفکر پردازان مزاحم شدم.\n— مدیر فروش"
        context.copyToClipboard(message)

        val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        val item = clipboard.primaryClip?.getItemAt(0)
        val copied = item?.coerceToText(context).toString()
        assertEquals(message, copied)
    }

    @Test
    fun `copy round-trips mixed latin and persian text`() {
        val context = RuntimeEnvironment.getApplication()
        val message = "گزارش ۱۴۰۳/۰۸/۰۱ — Call #42"
        context.copyToClipboard(message)

        val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        assertEquals(
            message,
            clipboard.primaryClip?.getItemAt(0)?.coerceToText(context).toString(),
        )
    }
}
