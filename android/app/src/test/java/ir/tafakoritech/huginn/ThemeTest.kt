package ir.tafakoritech.huginn

import androidx.compose.ui.graphics.toArgb
import ir.tafakoritech.huginn.core.theme.DarkHuginnColors
import ir.tafakoritech.huginn.core.theme.LightHuginnColors
import ir.tafakoritech.huginn.core.theme.ThemeMode
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

class ThemeTest {

    @Test
    fun `dark palette matches TP tokens`() {
        assertEquals(0xFF021C25.toInt(), DarkHuginnColors.background.toArgb())
        assertEquals(0xFF091E27.toInt(), DarkHuginnColors.surface.toArgb())
        assertEquals(0xFF69B437.toInt(), DarkHuginnColors.primary.toArgb())
        assertEquals(0xFF91DA73.toInt(), DarkHuginnColors.primaryHighlight.toArgb())
    }

    @Test
    fun `light palette matches TP tokens`() {
        assertEquals(0xFFF5F9F7.toInt(), LightHuginnColors.background.toArgb())
        assertEquals(0xFFFFFFFF.toInt(), LightHuginnColors.card.toArgb())
        assertEquals(0xFF021C25.toInt(), LightHuginnColors.onBackground.toArgb())
        assertEquals(0xFF69B437.toInt(), LightHuginnColors.primary.toArgb())
    }

    @Test
    fun `dark and light backgrounds differ`() {
        assertNotEquals(DarkHuginnColors.background.toArgb(), LightHuginnColors.background.toArgb())
    }

    @Test
    fun `theme mode has three stable values`() {
        assertEquals(3, ThemeMode.entries.size)
        assertEquals(ThemeMode.SYSTEM, ThemeMode.entries[0])
    }
}
