package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.data.model.Lead
import ir.tafakoritech.huginn.data.model.LeadDetailResponse
import ir.tafakoritech.huginn.data.model.LeadPermissions
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Lead-detail mapping: the assignee is surfaced by human-readable name, not a
 * raw database ID, and editable fields (source, canAssign) flow through the
 * real response shapes.
 */
class LeadDetailMappingTest {

    private fun response(ownerUserId: String?, ownerName: String?): LeadDetailResponse = LeadDetailResponse(
        lead = Lead(
            id = "lead-1",
            companyName = "عروسک‌سازی پارس",
            ownerUserId = ownerUserId,
            ownerName = ownerName,
            source = "لیست تولیدکنندگان",
            status = "CONTACTING",
        ),
    )

    private fun toDetail(r: LeadDetailResponse) =
        r.toDetail(score = null, evidence = emptyList(), interactions = emptyList(), followUps = emptyList())

    @Test
    fun `assignee name is carried instead of the raw database id`() {
        val detail = toDetail(response(ownerUserId = "u-agent-1", ownerName = "علی محمدی"))
        assertEquals("علی محمدی", detail.ownerName)
        assertEquals("u-agent-1", detail.ownerUserId)
    }

    @Test
    fun `unassigned lead has no owner name`() {
        val detail = toDetail(response(ownerUserId = null, ownerName = null))
        assertNull(detail.ownerName)
        assertNull(detail.ownerUserId)
    }

    @Test
    fun `lead source flows into the detail`() {
        val detail = toDetail(response("u-agent-1", "علی محمدی"))
        assertEquals("لیست تولیدکنندگان", detail.source)
    }

    @Test
    fun `can_assign permission parses from the backend payload`() {
        val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }
        val perms = json.decodeFromString<LeadPermissions>("""{"can_edit":true,"can_assign":true}""")
        assertTrue(perms.canEdit)
        assertTrue(perms.canAssign)
    }

    @Test
    fun `explicit json null is encoded as null for field clearing`() {
        // NetworkModule config: explicitNulls = false, encodeDefaults = true.
        val json = Json { ignoreUnknownKeys = true; explicitNulls = false; encodeDefaults = true }
        val body: JsonObject = buildJsonObject {
            put("business_type", JsonNull)
            put("city", JsonPrimitive("تهران"))
        }
        val encoded = json.encodeToString(JsonObject.serializer(), body)
        // A cleared field must reach the backend as an explicit null.
        assertEquals("""{"business_type":null,"city":"تهران"}""", encoded)
    }
}
