package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.data.mock.MockHuginnApi
import ir.tafakoritech.huginn.data.model.ContactActionRequest
import ir.tafakoritech.huginn.data.model.LoginRequest
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/** Work queue / report mapping against the test-only mock mirror of the API. */
class WorkQueueMappingTest {

    private fun api(): MockHuginnApi {
        val api = MockHuginnApi()
        runBlocking { api.login(LoginRequest(login = "agent", password = "password")) }
        return api
    }

    @Test
    fun `next actions are ranked with overdue first`() = runBlocking {
        val resp = api().nextActions(limit = 3)
        val items = resp.items
        assertTrue(items.isNotEmpty())
        assertTrue(items.first().overdue)
        val scores = items.map { it.priorityScore }
        assertEquals(scores.sortedDescending(), scores)
    }

    @Test
    fun `next action exposes lead and company and explainability`() = runBlocking {
        val first = api().nextActions(limit = 3).items.first()
        assertTrue(first.leadId.isNotBlank())
        assertTrue(first.companyName.isNotBlank())
        assertTrue(first.explainability.isNotEmpty())
    }

    @Test
    fun `next actions limit and counts are consistent`() = runBlocking {
        val resp = api().nextActions(limit = 3)
        assertTrue(resp.items.size <= 3)
        assertTrue(resp.totalActionable >= resp.items.size)
        assertTrue(resp.urgentCount >= resp.items.count { it.priority == "urgent" })
    }

    @Test
    fun `starting a call creates a mandatory pending report`() = runBlocking {
        val api = api()
        val detail = api.lead("lead-1")
        val contact = detail.contacts.first { it.isPrimaryContact }

        val action = api.contactAction(
            "lead-1",
            ContactActionRequest(contactId = contact.id, channel = "CALL", objective = "FIRST_CONTACT"),
        )

        assertEquals("CALL", action.channel)
        assertTrue(action.interactionId.isNotBlank())

        val pending = api.pendingReports()
        assertTrue(pending.requiresReport)
        assertEquals(action.interactionId, pending.reports.first().interactionId)
    }

    @Test
    fun `submitting the report clears the pending report`() = runBlocking {
        val api = api()
        val detail = api.lead("lead-1")
        val contact = detail.contacts.first { it.isPrimaryContact }
        val action = api.contactAction(
            "lead-1",
            ContactActionRequest(contactId = contact.id, channel = "CALL", objective = "FIRST_CONTACT"),
        )

        val result = api.submitReport(
            action.interactionId,
            ir.tafakoritech.huginn.data.model.ReportSubmitRequest(
                outcome = "INTERESTED",
                interestLevel = "HIGH",
                nextAction = "SEND_INFORMATION",
                summary = "علاقه‌مند به دریافت معرفی محصول",
            ),
        )
        assertEquals("INTERESTED", result.outcome)
        assertEquals("SEND_INFORMATION", result.nextAction)

        val pending = api.pendingReports()
        assertTrue(!pending.requiresReport)
    }

    @Test
    fun `lead search by mobile digits works`() = runBlocking {
        val result = api().leads(search = "09357578808", page = 1, pageSize = 20)
        assertTrue(result.leads.isNotEmpty())
        assertTrue(result.leads.all { it.companyName.isNotBlank() })
    }
}
