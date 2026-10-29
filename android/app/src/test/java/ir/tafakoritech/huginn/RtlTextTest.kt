package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.util.RtlText
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

/** RTL correctness: Latin values must never be reversed inside RTL layouts. */
class RtlTextTest {

    @Test
    fun `ltrEmbedding prefixes LRM so value stays LTR`() {
        val wrapped = RtlText.ltrEmbedding("+989357578808")
        assertEquals("\u200E+989357578808", wrapped)
    }

    @Test
    fun `phone number is not corrupted by embedding`() {
        val value = "+989357578808"
        assertEquals(value, RtlText.ltrEmbedding(value).removePrefix("\u200E"))
    }

    @Test
    fun `reversed is different from original`() {
        assertNotEquals("09357578808", RtlText.reversed("09357578808"))
    }

    @Test
    fun `labelValue keeps value intact`() {
        assertEquals("تماس: 09357578808", RtlText.labelValue("تماس", "09357578808"))
    }

    @Test
    fun `labelValue with null returns label`() {
        assertEquals("تماس", RtlText.labelValue("تماس", null))
    }
}
