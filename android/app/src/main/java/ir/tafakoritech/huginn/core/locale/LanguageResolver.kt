package ir.tafakoritech.huginn.core.locale

import java.util.Locale

/**
 * Resolves the initial application language from the device/system locale.
 *
 * Priority (first launch, no saved preference):
 *  - Persian device locale  -> fa
 *  - English device locale  -> en
 *  - anything else          -> en (fallback)
 */
object LanguageResolver {

    const val PERSIAN = "fa"
    const val ENGLISH = "en"

    val supportedLanguages = listOf(PERSIAN, ENGLISH)

    /** Effective language for the current default locale (device or app locale). */
    fun fromLocale(locale: Locale = Locale.getDefault()): String = when {
        locale.language.equals(PERSIAN, ignoreCase = true) -> PERSIAN
        locale.language.equals(ENGLISH, ignoreCase = true) -> ENGLISH
        else -> ENGLISH
    }

    /** User-visible name for a language code (native form). */
    fun displayName(language: String): String = when (language) {
        PERSIAN -> "فارسی"
        ENGLISH -> "English"
        else -> ENGLISH
    }
}
