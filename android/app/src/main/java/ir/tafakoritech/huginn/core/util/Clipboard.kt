package ir.tafakoritech.huginn.core.util

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context

/**
 * Put [text] on the system clipboard. [ClipData.newPlainText] copies the text
 * verbatim, so Persian/Unicode characters are preserved exactly.
 */
fun Context.copyToClipboard(text: String) {
    val clipboard = getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
    clipboard.setPrimaryClip(ClipData.newPlainText("text", text))
}
