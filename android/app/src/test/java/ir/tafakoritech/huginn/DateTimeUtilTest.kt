package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.util.DateTimeUtil
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.time.LocalDateTime
import java.time.LocalTime
import java.time.ZoneId
import java.util.TimeZone

/**
 * DateTimeUtil is the single presentation-layer source of truth for timestamps:
 * everything renders in Asia/Tehran and uses Jalali dates for Persian, Gregorian
 * for English — regardless of the device locale or device timezone.
 */
class DateTimeUtilTest {

    private var defaultLanguage: String = "en"
    private var defaultTimeZone: TimeZone? = null

    @Before
    fun setUp() {
        defaultLanguage = DateTimeUtil.activeLanguage
        defaultTimeZone = TimeZone.getDefault()
    }

    @After
    fun tearDown() {
        DateTimeUtil.setLanguage(defaultLanguage)
        TimeZone.setDefault(defaultTimeZone)
    }

    private fun iso(local: LocalDateTime): String =
        DateTimeUtil.toIso(local)

    private fun tehran(local: LocalDateTime): String =
        DateTimeUtil.toIso(local)

    // ---------------------------------------------------------------- parsing

    @Test
    fun `parseIso handles Z and offset forms`() {
        assertTrue(DateTimeUtil.parseIso("2025-01-15T10:30:00Z") != null)
        assertTrue(DateTimeUtil.parseIso("2025-01-15T10:30:00+03:30") != null)
        assertNull(DateTimeUtil.parseIso("not a date"))
        assertNull(DateTimeUtil.parseIso(null))
    }

    // --------------------------------------------- Tehran timezone conversion

    @Test
    fun `formatTime renders Tehran time regardless of device timezone`() {
        // UTC+3:30 => 10:30Z is 14:00 in Tehran.
        val value = "2025-01-15T10:30:00Z"
        assertEquals("14:00", DateTimeUtil.formatTime(value))

        // Deliberately use a foreign device timezone: output must not change.
        TimeZone.setDefault(TimeZone.getTimeZone("America/New_York"))
        assertEquals("14:00", DateTimeUtil.formatTime(value))
        TimeZone.setDefault(TimeZone.getTimeZone("UTC"))
        assertEquals("14:00", DateTimeUtil.formatTime(value))
    }

    @Test
    fun `formatTime uses persian digits in persian language`() {
        DateTimeUtil.setLanguage("fa")
        assertEquals("۱۴:۰۰", DateTimeUtil.formatTime("2025-01-15T10:30:00Z"))
    }

    @Test
    fun `Tehran midnight crosses the UTC date`() {
        // 2024-03-19T20:30Z is already 2024-03-20 00:00 in Tehran.
        DateTimeUtil.setLanguage("en")
        assertEquals("2024/03/20 00:00", DateTimeUtil.formatDateTime("2024-03-19T20:30:00Z"))
    }

    @Test
    fun `toIso encodes Tehran wall time to the correct instant`() {
        assertEquals("2024-03-19T20:30:00Z", DateTimeUtil.toIso(LocalDateTime.of(2024, 3, 20, 0, 0)))
        assertEquals("2025-01-15T06:30:00Z", DateTimeUtil.toIso(LocalDateTime.of(2025, 1, 15, 10, 0)))
    }

    @Test
    fun `toIso accepts separate date and time`() {
        assertEquals(
            "2024-03-19T20:30:00Z",
            DateTimeUtil.toIso(java.time.LocalDate.of(2024, 3, 20), LocalTime.of(0, 0)),
        )
    }

    @Test
    fun `toIso round-trips through formatTime`() {
        val roundTripped = DateTimeUtil.toIso(LocalDateTime.of(2024, 3, 20, 14, 30))
        assertEquals("14:30", DateTimeUtil.formatTime(roundTripped))
    }

    // ------------------------------------------------------------ date formats

    @Test
    fun `formatDate shows Gregorian for English`() {
        DateTimeUtil.setLanguage("en")
        assertEquals("2025/01/15", DateTimeUtil.formatDate("2025-01-15T10:30:00Z"))
        assertNull(DateTimeUtil.formatDate(null))
    }

    @Test
    fun `formatDate shows Jalali for Persian`() {
        DateTimeUtil.setLanguage("fa")
        // Known anchors: 1 Farvardin 1403 = 20 Mar 2024, 22 Bahman 1357 = 11 Feb 1979.
        assertEquals("۱۴۰۳/۰۱/۰۱", DateTimeUtil.formatDate("2024-03-19T20:30:00Z"))
        assertEquals("۱۴۰۳/۱۰/۲۶", DateTimeUtil.formatDate("2025-01-15T10:30:00Z"))
        assertEquals("۱۳۵۷/۱۱/۲۲", DateTimeUtil.formatDate("1979-02-11T00:00:00Z"))
    }

    @Test
    fun `formatDate respects Jalali leap years`() {
        DateTimeUtil.setLanguage("fa")
        // 1403 is a leap year: Esfand has 30 days and 1 Farvardin 1404 = 21 Mar 2025.
        assertEquals("۱۴۰۳/۱۲/۳۰", DateTimeUtil.formatDate("2025-03-20T00:00:00Z"))
        assertEquals("۱۴۰۴/۰۱/۰۱", DateTimeUtil.formatDate("2025-03-21T00:00:00Z"))
    }

    @Test
    fun `formatDateTime combines date and Tehran time`() {
        DateTimeUtil.setLanguage("en")
        assertEquals("2025/01/15 14:00", DateTimeUtil.formatDateTime("2025-01-15T10:30:00Z"))
        DateTimeUtil.setLanguage("fa")
        assertEquals("۱۴۰۳/۱۰/۲۶ ۱۴:۰۰", DateTimeUtil.formatDateTime("2025-01-15T10:30:00Z"))
    }

    // --------------------------------------------------------------- language

    @Test
    fun `setLanguage drives formatting independent of device locale`() {
        // English regardless of a Persian device locale.
        DateTimeUtil.setLanguage("en")
        assertEquals("2025/01/15 14:00", DateTimeUtil.formatDateTime("2025-01-15T10:30:00Z"))
        // Persian regardless of an English device locale.
        DateTimeUtil.setLanguage("fa")
        assertEquals("۱۴۰۳/۱۰/۲۶ ۱۴:۰۰", DateTimeUtil.formatDateTime("2025-01-15T10:30:00Z"))
    }

    // --------------------------------------------------------- relative labels

    @Test
    fun `isToday and isPast use the Tehran day`() {
        val now = DateTimeUtil.nowInTehran()
        assertTrue(DateTimeUtil.isToday(iso(now)))
        assertTrue(DateTimeUtil.isPast(iso(now.minusMinutes(5))))
        assertFalse(DateTimeUtil.isPast(iso(now.plusMinutes(5))))
        assertFalse(DateTimeUtil.isToday(iso(now.minusDays(1))))
    }

    @Test
    fun `daysUntil counts calendar days in Tehran`() {
        val now = DateTimeUtil.nowInTehran()
        assertEquals(1L, DateTimeUtil.daysUntil(iso(now.plusDays(1))))
        assertEquals(0L, DateTimeUtil.daysUntil(iso(now)))
    }

    @Test
    fun `relative label marks today and yesterday in persian`() {
        DateTimeUtil.setLanguage("fa")
        val now = DateTimeUtil.nowInTehran()
        assertTrue(DateTimeUtil.formatRelative(iso(now)).startsWith("امروز"))
        assertEquals("دیروز", DateTimeUtil.formatRelative(iso(now.minusDays(1))))
        assertEquals("نامشخص", DateTimeUtil.formatRelative(null))
    }

    @Test
    fun `relative label is english in english language`() {
        DateTimeUtil.setLanguage("en")
        val now = DateTimeUtil.nowInTehran()
        assertTrue(DateTimeUtil.formatRelative(iso(now)).startsWith("Today"))
        assertEquals("Yesterday", DateTimeUtil.formatRelative(iso(now.minusDays(1))))
        assertEquals("Unknown", DateTimeUtil.formatRelative(null))
    }

    @Test
    fun `relative label shows days ago within a week`() {
        DateTimeUtil.setLanguage("fa")
        val now = DateTimeUtil.nowInTehran()
        assertEquals("۳ روز پیش", DateTimeUtil.formatRelative(iso(now.minusDays(3))))
        assertEquals("۷ روز پیش", DateTimeUtil.formatRelative(iso(now.minusDays(7))))

        DateTimeUtil.setLanguage("en")
        assertEquals("3 days ago", DateTimeUtil.formatRelative(iso(now.minusDays(3))))
        assertEquals("7 days ago", DateTimeUtil.formatRelative(iso(now.minusDays(7))))
    }

    @Test
    fun `relative label falls back to full date outside a week`() {
        DateTimeUtil.setLanguage("en")
        val now = DateTimeUtil.nowInTehran()
        val tenDaysAgo = iso(now.minusDays(10))
        val expected = tenDaysAgo.let { DateTimeUtil.formatDate(it) }
        assertEquals(expected, DateTimeUtil.formatRelative(tenDaysAgo))
    }

    // ----------------------------------------------------------------- digits

    @Test
    fun `toLocalDigits converts to persian digits in persian language`() {
        DateTimeUtil.setLanguage("fa")
        assertEquals("۰۹۳۵۷۵۷۸۸۰۸", DateTimeUtil.toLocalDigits("09357578808"))
        assertEquals("", DateTimeUtil.toLocalDigits(""))
        assertEquals("", DateTimeUtil.toLocalDigits(null))
    }

    @Test
    fun `toLocalDigits keeps latin digits in english language`() {
        DateTimeUtil.setLanguage("en")
        assertEquals("09357578808", DateTimeUtil.toLocalDigits("09357578808"))
    }

    // ------------------------------------------------------ timezone metadata

    @Test
    fun `Tehran zone is the fixed business timezone`() {
        assertEquals(ZoneId.of("Asia/Tehran"), DateTimeUtil.TEHRAN_ZONE)
    }
}
