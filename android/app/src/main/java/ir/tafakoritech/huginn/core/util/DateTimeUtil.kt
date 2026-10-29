package ir.tafakoritech.huginn.core.util

import ir.tafakoritech.huginn.core.locale.LanguageResolver
import java.time.Instant
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.LocalTime
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.time.temporal.ChronoUnit

/**
 * Single, centralized date/time service for the whole app.
 *
 * Backend timestamps (RFC3339, UTC) are never mutated. Every displayed date and
 * time is produced here and follows two fixed rules:
 *
 *  - All times are rendered in the business timezone **Asia/Tehran**, never the
 *    device timezone.
 *  - Dates use the **Jalali (Shamsi)** calendar when the app language is
 *    Persian, and the **Gregorian** calendar when it is English.
 *
 * The active language must be kept in sync via [setLanguage] (the
 * [ir.tafakoritech.huginn.core.locale.LocaleManager] and workers do this), so
 * formatting never depends on the device locale.
 */
object DateTimeUtil {

    /** Business timezone for every displayed timestamp. */
    val TEHRAN_ZONE: ZoneId = ZoneId.of("Asia/Tehran")

    /** Active UI language ("fa"/"en"), initialized from the device language. */
    var activeLanguage: String = LanguageResolver.fromLocale()
        private set

    /** Update the active language; call whenever the app language changes. */
    fun setLanguage(language: String) {
        activeLanguage = language
    }

    private val isPersian: Boolean get() = activeLanguage == LanguageResolver.PERSIAN

    private val TIME_FMT = DateTimeFormatter.ofPattern("HH:mm")
    private val GREGORIAN_DATE_FMT = DateTimeFormatter.ofPattern("yyyy/MM/dd")

    fun parseIso(value: String?): Instant? {
        if (value.isNullOrBlank()) return null
        return runCatching {
            when {
                value.endsWith("Z") -> Instant.parse(value)
                else -> OffsetDateTime.parse(value).toInstant()
            }
        }.getOrNull()
    }

    /**
     * Backend timestamp converted to Tehran wall-clock time.
     * Presentation-only: the original instant (and any stored value) is untouched.
     */
    fun toTehranDateTime(value: String?): LocalDateTime? =
        parseIso(value)?.atZone(TEHRAN_ZONE)?.toLocalDateTime()

    /** Current wall-clock time in Tehran (for "now"/relative comparisons). */
    fun nowInTehran(): LocalDateTime = ZonedDateTime.now(TEHRAN_ZONE).toLocalDateTime()

    /** Current date in Tehran (for "today"/presets). */
    fun todayInTehran(): LocalDate = ZonedDateTime.now(TEHRAN_ZONE).toLocalDate()

    /**
     * Encode a Tehran wall-clock value as an RFC3339 (UTC) instant for the
     * backend. Use this whenever a timestamp is created from user input.
     */
    fun toIso(tehran: LocalDateTime): String =
        DateTimeFormatter.ISO_INSTANT.format(tehran.atZone(TEHRAN_ZONE).toInstant())

    fun toIso(tehran: LocalDate, time: LocalTime): String = toIso(tehran.atTime(time))

    fun formatTime(value: String?): String? =
        toTehranDateTime(value)?.format(TIME_FMT)?.let { toLocalDigits(it) }

    fun formatDate(value: String?): String? {
        val tehran = toTehranDateTime(value) ?: return null
        return if (isPersian) {
            val (jy, jm, jd) = toJalali(tehran.year, tehran.monthValue, tehran.dayOfMonth)
            toLocalDigits("%04d/%02d/%02d".format(jy, jm, jd))
        } else {
            tehran.format(GREGORIAN_DATE_FMT)
        }
    }

    fun formatDateTime(value: String?): String? {
        val time = formatTime(value) ?: return null
        val date = formatDate(value) ?: return null
        return "$date $time"
    }

    fun isToday(value: String?): Boolean {
        val tehran = toTehranDateTime(value) ?: return false
        return tehran.toLocalDate() == todayInTehran()
    }

    fun isPast(value: String?): Boolean {
        val tehran = toTehranDateTime(value) ?: return false
        return tehran.isBefore(nowInTehran())
    }

    fun isFuture(value: String?): Boolean = !isToday(value) && !isPast(value)

    fun daysUntil(value: String?): Long? {
        val tehran = toTehranDateTime(value) ?: return null
        return ChronoUnit.DAYS.between(todayInTehran(), tehran.toLocalDate())
    }

    /**
     * Compact relative label for interaction history / follow-up cards,
     * localized for the active UI language and relative to the Tehran day.
     * fa: "امروز 14:30", "دیروز", "۳ روز پیش"
     * en: "Today 14:30", "Yesterday", "3 days ago"
     */
    fun formatRelative(value: String?): String {
        val tehran = toTehranDateTime(value) ?: return unknownLabel()
        val now = nowInTehran()
        val today = now.toLocalDate()
        val date = tehran.toLocalDate()
        return when {
            date == today -> todayLabel(tehran)
            date == today.minusDays(1) -> yesterdayLabel()
            !date.isBefore(today.minusDays(7)) -> daysAgoLabel(ChronoUnit.DAYS.between(date, today))
            else -> formatDate(value) ?: unknownLabel()
        }
    }

    private fun todayLabel(local: LocalDateTime): String =
        if (isPersian) "امروز ${local.format(TIME_FMT)}" else "Today ${local.format(TIME_FMT)}"

    private fun yesterdayLabel(): String = if (isPersian) "دیروز" else "Yesterday"

    private fun daysAgoLabel(days: Long): String =
        if (isPersian) "${toLocalDigits(days.toString())} روز پیش" else "$days days ago"

    private fun unknownLabel(): String = if (isPersian) "نامشخص" else "Unknown"

    /**
     * Convert digits to the active locale's numeral system:
     *  - Persian UI -> Persian digits (۰-۹)
     *  - otherwise   -> Latin digits unchanged
     */
    fun toLocalDigits(value: String?): String {
        if (value.isNullOrBlank()) return value ?: ""
        if (!isPersian) return value
        return value.map { c ->
            when (c) {
                in '0'..'9' -> '۰' + (c - '0')
                else -> c
            }
        }.joinToString("")
    }

    // ------------------------------------------------------------------
    // Jalali (Shamsi) calendar conversion — presentation-only helpers.
    // Verified anchors: 1 Farvardin 1 = 21 Mar 622 (proleptic Gregorian),
    // 1 Farvardin 1403 = 20 Mar 2024, 22 Bahman 1357 = 11 Feb 1979.
    // ------------------------------------------------------------------

    private const val JALALI_EPOCH_JDN = 1_948_320L // 1 Farvardin 1

    private fun toJalali(gy: Int, gm: Int, gd: Int): Triple<Int, Int, Int> {
        var days = toJdn(gy, gm, gd) - JALALI_EPOCH_JDN
        var jy = (days / 365.24).toInt() + 1
        if (jy < 1) jy = 1
        while (startOfYear(jy) > days) jy--
        while (startOfYear(jy + 1) <= days) jy++
        val dayOfYear = (days - startOfYear(jy)).toInt()
        return if (dayOfYear < 186) {
            Triple(jy, dayOfYear / 31 + 1, dayOfYear % 31 + 1)
        } else {
            val offset = dayOfYear - 186
            Triple(jy, offset / 30 + 7, offset % 30 + 1)
        }
    }

    /** Days from the Jalaali epoch to 1 Farvardin of [jy]. */
    private fun startOfYear(jy: Int): Long = (jy - 1) * 365L + leapsUpTo(jy - 1)

    /** Number of leap years in 1..[jy]. */
    private fun leapsUpTo(jy: Int): Long {
        var count = 0L
        for (y in 1..jy) if (isJalaliLeap(y)) count++
        return count
    }

    /** Arithmetical Jalaali leap rule (33-year cycle; positions 1,5,9,13,17,22,26,30). */
    private fun isJalaliLeap(jy: Int): Boolean = ((25 * jy + 11) % 33) < 8

    /** Gregorian calendar date -> Julian day number (integer, noon-based). */
    private fun toJdn(gy: Int, gm: Int, gd: Int): Long {
        val a = (14 - gm) / 12
        val y = gy + 4800 - a
        val m = gm + 12 * a - 3
        return gd + (153L * m + 2) / 5 + 365L * y + y / 4L - y / 100L + y / 400L - 32045L
    }
}
