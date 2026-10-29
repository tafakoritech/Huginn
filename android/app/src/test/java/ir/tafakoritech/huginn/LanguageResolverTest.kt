package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.locale.LanguageResolver
import org.junit.Assert.assertEquals
import org.junit.Test
import java.util.Locale

/** Language detection: Persian / English / fallback. */
class LanguageResolverTest {

    @Test
    fun `persian device locale maps to fa`() {
        assertEquals("fa", LanguageResolver.fromLocale(Locale("fa", "IR")))
    }

    @Test
    fun `english device locale maps to en`() {
        assertEquals("en", LanguageResolver.fromLocale(Locale.ENGLISH))
    }

    @Test
    fun `unsupported device locale falls back to english`() {
        assertEquals("en", LanguageResolver.fromLocale(Locale.GERMAN))
        assertEquals("en", LanguageResolver.fromLocale(Locale("zh", "CN")))
    }

    @Test
    fun `persian variant languages resolve to fa`() {
        assertEquals("fa", LanguageResolver.fromLocale(Locale("fa", "AF")))
    }

    @Test
    fun `supported languages list is stable`() {
        assertEquals(listOf("fa", "en"), LanguageResolver.supportedLanguages)
    }

    @Test
    fun `display names are native forms`() {
        assertEquals("فارسی", LanguageResolver.displayName("fa"))
        assertEquals("English", LanguageResolver.displayName("en"))
    }
}
