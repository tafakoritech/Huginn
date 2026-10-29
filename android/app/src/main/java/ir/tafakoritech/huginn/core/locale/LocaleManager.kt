package ir.tafakoritech.huginn.core.locale

import android.content.Context
import androidx.appcompat.app.AppCompatDelegate
import androidx.core.os.LocaleListCompat
import dagger.hilt.android.qualifiers.ApplicationContext
import ir.tafakoritech.huginn.core.util.DateTimeUtil
import java.util.Locale
import javax.inject.Inject
import javax.inject.Singleton

/**
 * App language handling (fa / en). Uses AppCompatDelegate.setApplicationLocales,
 * which recreates the activity configuration so Compose picks up the correct
 * layout direction (RTL for fa) automatically.
 */
@Singleton
class LocaleManager @Inject constructor(
    @ApplicationContext private val context: Context,
) {

    fun apply(language: String) {
        DateTimeUtil.setLanguage(language)
        val locale = when (language) {
            "fa" -> Locale("fa")
            "en" -> Locale.ENGLISH
            else -> Locale.getDefault()
        }
        // Skip if the app is already on the requested locale: avoids a pointless
        // activity recreation at startup.
        val current = runCatching { AppCompatDelegate.getApplicationLocales().get(0) }.getOrNull()
            ?: Locale.getDefault()
        if (current.language == locale.language) return
        AppCompatDelegate.setApplicationLocales(LocaleListCompat.create(locale))
    }

    val defaultLanguage: String get() = "fa"
}
