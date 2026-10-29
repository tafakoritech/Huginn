package ir.tafakoritech.huginn.core.util

/**
 * RTL-safe text helpers.
 *
 * Persian is the default layout direction. Values such as phone numbers, email
 * addresses, URLs, Latin company names and codes must NOT be directionally
 * reversed. Wrapping such values with [ltrEmbedding] forces LTR ordering inside
 * a RTL paragraph without touching the surrounding Persian text.
 */
object RtlText {

    /** Left-to-right mark. */
    private const val LRM = "\u200E"

    /** Wraps a value so it renders LTR even inside RTL text. */
    fun ltrEmbedding(value: String): String = LRM + value

    /** LTR embedding for nullable values. */
    fun ltrEmbeddingOrBlank(value: String?): String = value?.let { LRM + it } ?: ""

    /**
     * Reverses a mixed string the way a reversed visual order would corrupt it.
     * Used only by tests to simulate a broken reversal.
     */
    fun reversed(value: String): String = value.reversed()

    /**
     * BiDi-aware formatting of "label: value" pairs where value is Latin.
     * Produces "label: value" with the value kept LTR. The label itself is RTL.
     */
    fun labelValue(label: String, value: String?): String {
        val v = value ?: return label
        return "$label: $v"
    }
}
