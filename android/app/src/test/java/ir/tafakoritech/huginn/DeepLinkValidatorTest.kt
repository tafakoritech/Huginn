package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.external.DeepLinkValidator
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class DeepLinkValidatorTest {

    @Test
    fun `lead deep link maps to route`() {
        assertEquals("lead/abc-123", DeepLinkValidator.parse("huginn://lead/abc-123"))
    }

    @Test
    fun `report deep link maps to mandatory report route`() {
        assertEquals("mandatory-report/int_9", DeepLinkValidator.parse("huginn://report/int_9"))
    }

    @Test
    fun `followup deep link maps to followups tab`() {
        assertEquals("followups", DeepLinkValidator.parse("huginn://followup/fu-1"))
    }

    @Test
    fun `unknown host is rejected`() {
        assertNull(DeepLinkValidator.parse("huginn://unknown/xyz"))
    }

    @Test
    fun `blank or oversized ids are rejected`() {
        assertNull(DeepLinkValidator.parse("huginn://lead/"))
        assertNull(DeepLinkValidator.parse("huginn://lead/" + "x".repeat(200)))
    }

    @Test
    fun `path traversal or special chars rejected`() {
        assertNull(DeepLinkValidator.parse("huginn://lead/../secret"))
        assertNull(DeepLinkValidator.parse("huginn://lead/a b"))
        assertNull(DeepLinkValidator.parse("not a deep link"))
    }
}
