package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.external.MessengerDeepLinks
import ir.tafakoritech.huginn.core.external.MessengerPolicy
import ir.tafakoritech.huginn.data.model.DeepLink
import ir.tafakoritech.huginn.data.model.InteractionChannel
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Per-app messenger routing rules (tested without a device):
 *  - Rubika/Eitaa -> save-contact only, never a direct chat.
 *  - Telegram/Bale -> phone-based link preferred, username is the fallback;
 *    a valid mobile OR username produces a link, neither returns null.
 *  - WhatsApp -> wa target with optional encoded text.
 */
class MessengerRoutingTest {

    private fun deepLink(channel: String, username: String? = null): DeepLink =
        DeepLink(channel = channel, type = "username", target = "", username = username)

    @Test
    fun `rubika and eitaa are save contact only`() {
        assertTrue(MessengerPolicy.requiresContactSave(InteractionChannel.RUBIKA))
        assertTrue(MessengerPolicy.requiresContactSave(InteractionChannel.EITAA))
        // A username must never produce a direct link for these channels.
        val dl = DeepLink(channel = InteractionChannel.RUBIKA, type = "username", target = "https://rubika.ir/x", username = "x")
        assertNull(MessengerDeepLinks.directUrl(InteractionChannel.RUBIKA, dl, phone = null, text = "سلام"))
        assertNull(MessengerDeepLinks.directUrl(InteractionChannel.EITAA, dl, phone = null, text = "سلام"))
    }

    @Test
    fun `bale telegram whatsapp support direct chat`() {
        assertFalse(MessengerPolicy.requiresContactSave(InteractionChannel.BALE))
        assertFalse(MessengerPolicy.requiresContactSave(InteractionChannel.TELEGRAM))
        assertFalse(MessengerPolicy.requiresContactSave(InteractionChannel.WHATSAPP))
        assertTrue(MessengerPolicy.supportsDirectChat(InteractionChannel.BALE))
        assertTrue(MessengerPolicy.supportsDirectChat(InteractionChannel.TELEGRAM))
        assertTrue(MessengerPolicy.supportsDirectChat(InteractionChannel.WHATSAPP))
    }

    // ---- Phone-based links (mobile only) ----

    @Test
    fun `telegram phone-only builds t me phone link`() {
        assertEquals(
            "https://t.me/+989121234567",
            MessengerDeepLinks.directUrl(InteractionChannel.TELEGRAM, deepLink(InteractionChannel.TELEGRAM), phone = "09121234567", text = null),
        )
    }

    @Test
    fun `bale phone-only builds ble ir phone link`() {
        assertEquals(
            "https://ble.ir/+989121234567",
            MessengerDeepLinks.directUrl(InteractionChannel.BALE, deepLink(InteractionChannel.BALE), phone = "09121234567", text = null),
        )
    }

    @Test
    fun `phone normalization handles local and international forms`() {
        val dl = deepLink(InteractionChannel.TELEGRAM)
        // Local, international-with-plus and 00-prefixed forms all normalize.
        assertEquals("https://t.me/+989121234567", MessengerDeepLinks.directUrl(InteractionChannel.TELEGRAM, dl, phone = "09121234567", text = null))
        assertEquals("https://t.me/+989121234567", MessengerDeepLinks.directUrl(InteractionChannel.TELEGRAM, dl, phone = "+989121234567", text = null))
        assertEquals("https://t.me/+989121234567", MessengerDeepLinks.directUrl(InteractionChannel.TELEGRAM, dl, phone = "989121234567", text = null))
        assertEquals("https://t.me/+989121234567", MessengerDeepLinks.directUrl(InteractionChannel.TELEGRAM, dl, phone = "00989121234567", text = null))
    }

    // ---- Username fallback (username only) ----

    @Test
    fun `telegram builds tg resolve from username when no phone`() {
        val dl = deepLink(InteractionChannel.TELEGRAM, username = "rezaei")
        assertEquals("tg://resolve?domain=rezaei", MessengerDeepLinks.directUrl(InteractionChannel.TELEGRAM, dl, phone = null, text = null))
    }

    @Test
    fun `bale builds bale profile from username when no phone`() {
        val dl = deepLink(InteractionChannel.BALE, username = "rezaei")
        assertEquals("bale://profile/rezaei", MessengerDeepLinks.directUrl(InteractionChannel.BALE, dl, phone = null, text = null))
    }

    // ---- Preference and unavailable ----

    @Test
    fun `phone is preferred over username when both exist`() {
        val dl = deepLink(InteractionChannel.TELEGRAM, username = "rezaei")
        assertEquals("https://t.me/+989121234567", MessengerDeepLinks.directUrl(InteractionChannel.TELEGRAM, dl, phone = "09121234567", text = null))
        val bale = deepLink(InteractionChannel.BALE, username = "rezaei")
        assertEquals("https://ble.ir/+989121234567", MessengerDeepLinks.directUrl(InteractionChannel.BALE, bale, phone = "09121234567", text = null))
    }

    @Test
    fun `telegram and bale return null when neither phone nor username exists`() {
        val empty = DeepLink(channel = InteractionChannel.TELEGRAM, type = "username", target = "", username = "")
        assertNull(MessengerDeepLinks.directUrl(InteractionChannel.TELEGRAM, empty, phone = null, text = "سلام"))
        assertNull(MessengerDeepLinks.directUrl(InteractionChannel.BALE, empty, phone = null, text = "سلام"))
        // A missing deep link never produces a direct URL.
        assertNull(MessengerDeepLinks.directUrl(InteractionChannel.WHATSAPP, null, phone = null, text = "سلام"))
    }

    @Test
    fun `non Iranian numbers do not produce phone links`() {
        val dl = deepLink(InteractionChannel.TELEGRAM)
        assertNull(MessengerDeepLinks.directUrl(InteractionChannel.TELEGRAM, dl, phone = "+15551234567", text = null))
    }

    // ---- WhatsApp ----

    @Test
    fun `whatsapp uses wa target with optional encoded text`() {
        val dl = DeepLink(channel = InteractionChannel.WHATSAPP, type = "wa.me", target = "https://wa.me/989357578808")
        assertEquals("https://wa.me/989357578808", MessengerDeepLinks.directUrl(InteractionChannel.WHATSAPP, dl, phone = null, text = null))
        assertEquals(
            "https://wa.me/989357578808?text=%D8%B3%D9%84%D8%A7%D9%85",
            MessengerDeepLinks.directUrl(InteractionChannel.WHATSAPP, dl, phone = null, text = "سلام"),
        )
    }
}
