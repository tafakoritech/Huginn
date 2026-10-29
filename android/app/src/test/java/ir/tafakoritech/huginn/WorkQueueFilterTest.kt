package ir.tafakoritech.huginn.feature.workqueue

import ir.tafakoritech.huginn.data.model.NextActionItem
import org.junit.Assert.assertEquals
import org.junit.Test

class WorkQueueFilterTest {

    private fun item(
        id: String,
        priority: String = "normal",
        overdue: Boolean = false,
        sourceType: String = "follow_up",
        actionType: String = "follow_up",
        dueAt: String? = null,
    ) = NextActionItem(
        id = id,
        sourceType = sourceType,
        leadId = id,
        companyName = id,
        priority = priority,
        overdue = overdue,
        actionType = actionType,
        dueAt = dueAt,
    )

    @Test
    fun `all tab returns everything`() {
        val items = listOf(item("a"), item("b", priority = "urgent"))
        assertEquals(2, filterForTab(items, WorkQueueTab.ALL).size)
    }

    @Test
    fun `urgent tab filters by priority`() {
        val items = listOf(item("a"), item("b", priority = "urgent"), item("c", priority = "important"))
        assertEquals(1, filterForTab(items, WorkQueueTab.URGENT).size)
    }

    @Test
    fun `overdue tab filters overdue only`() {
        val items = listOf(item("a", overdue = true), item("b", overdue = false))
        assertEquals(1, filterForTab(items, WorkQueueTab.OVERDUE).size)
    }

    @Test
    fun `reports tab filters pending reports`() {
        val items = listOf(item("a"), item("b", sourceType = "pending_report", actionType = "report"))
        assertEquals(1, filterForTab(items, WorkQueueTab.REPORTS).size)
    }

    @Test
    fun `call tab filters call actions`() {
        val items = listOf(item("a", actionType = "call"), item("b", actionType = "message"))
        assertEquals(1, filterForTab(items, WorkQueueTab.CALL).size)
    }
}
