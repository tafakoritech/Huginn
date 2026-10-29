package ir.tafakoritech.huginn.core.util

/**
 * Phone formatting helpers.
 *
 * The backend stores canonical numbers as +989357578808; employees should see
 * the local form 09357578808. Links and API calls always use the canonical form.
 */
object PhoneFormatter {

    /** Normalize to digits only, dropping any separators/spaces. */
    fun digitsOnly(value: String?): String = value?.filter { it.isDigit() } ?: ""

    /** +989357578808 -> 09357578808. Returns null for non-Iranian or empty input. */
    fun canonicalToLocal(canonical: String?): String? {
        val digits = digitsOnly(canonical)
        return when {
            digits.isEmpty() -> null
            digits.length == 12 && digits.startsWith("98") -> "0" + digits.substring(2)
            digits.length == 11 && digits.startsWith("0") -> digits
            else -> canonical
        }
    }

    /** Display form used on cards and headers (local Iranian format when possible). */
    fun toDisplay(value: String?): String? = canonicalToLocal(value) ?: value

    /** Best-effort wa.me target for a mobile number (0935.. -> 98935..). */
    fun waMeTarget(value: String?): String? {
        var digits = digitsOnly(value).removePrefix("+")
        if (digits.startsWith("0") && digits.length == 11) {
            digits = "98" + digits.substring(1)
        }
        return if (digits.isNotEmpty()) "wa.me/$digits" else null
    }

    /**
     * Normalize an Iranian mobile number to the canonical international form
     * (+98...). Accepts 09121234567, +989121234567, 989121234567 and
     * 00989121234567; returns null for anything that is not a valid Iranian
     * mobile. Phone-based messenger links (Telegram/Bale) use this value.
     */
    fun toInternationalMobile(value: String?): String? {
        var digits = digitsOnly(value)
        if (digits.isEmpty()) return null
        if (digits.startsWith("00")) digits = digits.substring(2)
        return when {
            digits.startsWith("98") && digits.length == 12 -> "+$digits"
            digits.startsWith("0") && digits.length == 11 -> "+98" + digits.substring(1)
            else -> null
        }
    }

    /** Whether [query] matches a number in any common representation. */
    fun matches(query: String, number: String?): Boolean {
        if (number.isNullOrBlank()) return false
        val q = digitsOnly(query)
        if (q.isEmpty()) return true
        val n = digitsOnly(number)
        val canonical = if (n.startsWith("0") && n.length == 11) "98" + n.substring(1) else n
        return queryVariants(q).any { canonical.contains(it) }
    }

    /** Normalize a query so 0935.., 98935.. and +98935.. all match a canonical number. */
    private fun queryVariants(query: String): Set<String> {
        val out = mutableSetOf(query)
        if (query.startsWith("0") && query.length >= 3) out += "98" + query.substring(1)
        if (query.startsWith("98") && query.length >= 4) out += "0" + query.substring(2)
        return out
    }
}
