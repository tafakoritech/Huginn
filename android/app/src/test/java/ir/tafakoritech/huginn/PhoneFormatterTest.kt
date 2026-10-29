package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.util.PhoneFormatter
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PhoneFormatterTest {

    @Test
    fun `canonical to local drops plus 98 and prepends zero`() {
        assertEquals("09357578808", PhoneFormatter.canonicalToLocal("+989357578808"))
    }

    @Test
    fun `local number stays unchanged`() {
        assertEquals("09357578808", PhoneFormatter.canonicalToLocal("09357578808"))
    }

    @Test
    fun `null and empty return null`() {
        assertNull(PhoneFormatter.canonicalToLocal(null))
        assertNull(PhoneFormatter.canonicalToLocal(""))
        assertNull(PhoneFormatter.canonicalToLocal("   "))
    }

    @Test
    fun `display keeps raw when non Iranian`() {
        assertEquals("+1 555 000", PhoneFormatter.toDisplay("+1 555 000"))
    }

    @Test
    fun `waMe target strips leading zero`() {
        assertEquals("wa.me/989357578808", PhoneFormatter.waMeTarget("+989357578808"))
        assertEquals("wa.me/989357578808", PhoneFormatter.waMeTarget("09357578808"))
    }

    @Test
    fun `international mobile normalizes local iranian numbers`() {
        assertEquals("+989121234567", PhoneFormatter.toInternationalMobile("09121234567"))
        assertEquals("+989121234567", PhoneFormatter.toInternationalMobile("+989121234567"))
        assertEquals("+989121234567", PhoneFormatter.toInternationalMobile("989121234567"))
        assertEquals("+989121234567", PhoneFormatter.toInternationalMobile("00989121234567"))
        assertEquals("+989121234567", PhoneFormatter.toInternationalMobile("0912 123 4567"))
    }

    @Test
    fun `international mobile rejects invalid or non iranian values`() {
        assertNull(PhoneFormatter.toInternationalMobile(null))
        assertNull(PhoneFormatter.toInternationalMobile(""))
        assertNull(PhoneFormatter.toInternationalMobile("   "))
        assertNull(PhoneFormatter.toInternationalMobile("+15551234567"))
        assertNull(PhoneFormatter.toInternationalMobile("9121234567"))
    }

    @Test
    fun `search matches multiple representations`() {
        assertTrue(PhoneFormatter.matches("0935", "+989357578808"))
        assertTrue(PhoneFormatter.matches("98935", "0935 757 8808"))
        assertTrue(PhoneFormatter.matches("+989357578808", "+989357578808"))
    }

    @Test
    fun `digitsOnly strips separators`() {
        assertEquals("989357578808", PhoneFormatter.digitsOnly("+98 (935) 757-88-08"))
    }
}
