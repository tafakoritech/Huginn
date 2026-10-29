package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.util.ScoreClassifier
import ir.tafakoritech.huginn.core.util.ScoreLevel
import org.junit.Assert.assertEquals
import org.junit.Test

class ScoreClassifierTest {

    @Test
    fun `low range`() {
        assertEquals(ScoreLevel.LOW, ScoreClassifier.classify(0))
        assertEquals(ScoreLevel.LOW, ScoreClassifier.classify(39))
    }

    @Test
    fun `medium range`() {
        assertEquals(ScoreLevel.MEDIUM, ScoreClassifier.classify(40))
        assertEquals(ScoreLevel.MEDIUM, ScoreClassifier.classify(69))
    }

    @Test
    fun `high range`() {
        assertEquals(ScoreLevel.HIGH, ScoreClassifier.classify(70))
        assertEquals(ScoreLevel.HIGH, ScoreClassifier.classify(100))
    }

    @Test
    fun `custom thresholds from backend config respected`() {
        val thresholds = ir.tafakoritech.huginn.core.util.ScoreThresholds(lowMax = 49, mediumMax = 79)
        assertEquals(ScoreLevel.LOW, ScoreClassifier.classify(45, thresholds))
        assertEquals(ScoreLevel.MEDIUM, ScoreClassifier.classify(60, thresholds))
        assertEquals(ScoreLevel.HIGH, ScoreClassifier.classify(85, thresholds))
    }
}
