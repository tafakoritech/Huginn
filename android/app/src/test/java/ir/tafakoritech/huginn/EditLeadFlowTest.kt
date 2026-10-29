package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.data.mock.MockHuginnApi
import ir.tafakoritech.huginn.data.model.AssignRequest
import ir.tafakoritech.huginn.data.model.ContactUpdateRequest
import ir.tafakoritech.huginn.data.model.LoginRequest
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * End-to-end Lead management flow against the test-only mock mirror of the
 * backend (response shapes follow backend/internal/httpapi):
 * open lead -> assignee name shown -> edit -> save -> reload shows new values.
 */
class EditLeadFlowTest {

    private fun api(login: String = "agent"): MockHuginnApi =
        MockHuginnApi().also { runBlocking { it.login(LoginRequest(login = login, password = "password")) } }

    @Test
    fun `lead detail shows assignee name not the raw id`() = runBlocking {
        val detail = api().lead("lead-1").toDetail(null, emptyList(), emptyList(), emptyList())
        assertEquals("علی محمدی", detail.ownerName)
        assertNotEquals(detail.ownerUserId, detail.ownerName)
    }

    @Test
    fun `authorized agent has edit permission`() = runBlocking {
        assertTrue(api().lead("lead-1").permissions.canEdit)
    }

    @Test
    fun `manager has assign permission but agent does not`() = runBlocking {
        assertTrue(api("manager").lead("lead-1").permissions.canAssign)
        assertFalse(api("agent").lead("lead-1").permissions.canAssign)
    }

    @Test
    fun `edit lead saves then reload shows updated values`() = runBlocking {
        val a = api()
        a.updateLead(
            "lead-1",
            buildJsonObject {
                put("company_name", JsonPrimitive("عروسک‌سازی پارس نوین"))
                put("business_type", JsonPrimitive("عمده‌فروشی"))
                put("source", JsonNull) // cleared field
                put("note", JsonPrimitive("اصلاح مشخصات"))
            },
        )
        val reloaded = a.lead("lead-1")
        assertEquals("عروسک‌سازی پارس نوین", reloaded.lead.companyName)
        assertEquals("عمده‌فروشی", reloaded.lead.businessType)
        assertNull(reloaded.lead.source)
    }

    @Test
    fun `editing a contact persists and is reflected after reload`() = runBlocking {
        val a = api()
        a.updateContact(
            "lead-1",
            "c-1",
            ContactUpdateRequest(jobTitle = "مدیرعامل", isDecisionMaker = false),
        )
        val reloaded = a.lead("lead-1")
        val contact = reloaded.contacts.first { it.id == "c-1" }
        assertEquals("مدیرعامل", contact.jobTitle)
        assertFalse(contact.isDecisionMaker)
        assertTrue(contact.isPrimaryContact) // untouched field preserved
    }

    @Test
    fun `assign changes owner and reload shows the new assignee name`() = runBlocking {
        val a = api()
        a.assignLead("lead-3", AssignRequest(assigneeId = "u-agent-2"))
        val reloaded = a.lead("lead-3")
        assertEquals("u-agent-2", reloaded.lead.ownerUserId)
        assertEquals("مریم رضایی", reloaded.lead.ownerName)
    }

    @Test
    fun `unauthorized viewer has no edit permission`() = runBlocking {
        assertFalse(api("viewer").lead("lead-1").permissions.canEdit)
    }
}
