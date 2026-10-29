package ir.tafakoritech.huginn.feature.home

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class NextActionMapperTest {

    @Test
    fun `only follow-up actions are snoozable`() {
        assertTrue(actionSnoozable("follow_up"))
        assertFalse(actionSnoozable("pending_report"))
        assertFalse(actionSnoozable("first_contact"))
    }

    @Test
    fun `only follow-up actions are completable inline`() {
        assertTrue(actionCompletable("follow_up"))
        assertFalse(actionCompletable("pending_report"))
    }

    @Test
    fun `report action opens the mandatory report`() {
        assertTrue(opensReport("report", "pending_report"))
        assertTrue(opensReport("call", "pending_report"))
        assertTrue(opensReport("report", "follow_up"))
        assertFalse(opensReport("call", "follow_up"))
        assertFalse(opensReport("follow_up", "stale_lead"))
    }
}
