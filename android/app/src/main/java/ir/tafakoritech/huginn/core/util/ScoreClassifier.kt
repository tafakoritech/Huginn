package ir.tafakoritech.huginn.core.util

/**
 * Score classification. Business rules come from the backend when available
 * (ScoreThresholds); these defaults only apply when the server provides none.
 */
data class ScoreThresholds(
    val lowMax: Int = 39,
    val mediumMax: Int = 69,
) {
    companion object {
        val DEFAULT = ScoreThresholds()
    }
}

enum class ScoreLevel { LOW, MEDIUM, HIGH }

object ScoreClassifier {
    fun classify(score: Int, thresholds: ScoreThresholds = ScoreThresholds.DEFAULT): ScoreLevel =
        when {
            score <= thresholds.lowMax -> ScoreLevel.LOW
            score <= thresholds.mediumMax -> ScoreLevel.MEDIUM
            else -> ScoreLevel.HIGH
        }
}
