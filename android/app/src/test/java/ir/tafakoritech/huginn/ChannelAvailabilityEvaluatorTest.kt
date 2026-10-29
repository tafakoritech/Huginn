package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.data.model.Contact
import ir.tafakoritech.huginn.data.model.InteractionChannel
import ir.tafakoritech.huginn.data.model.LeadPermissions
import ir.tafakoritech.huginn.feature.messaging.ChannelAvailability
import ir.tafakoritech.huginn.feature.messaging.ChannelAvailabilityEvaluator
import org.junit.Assert.assertEquals
import org.junit.Test

/** Communication channel state mapping (which channels are enabled per contact). */
class ChannelAvailabilityEvaluatorTest {

    private val fullContact = Contact(
        id = "c1", leadId = "l1", fullName = "محمد رضایی", mobileNormalized = "+989357578808",
        whatsapp = "+989357578808", telegram = "rezaei", bale = "rezaei", email = "info@pars.ir", eitaa = "rezaei",
    )
    private val phoneOnly = Contact(id = "c2", leadId = "l1", fullName = "علی", mobileNormalized = "+989121112233")
    private val usernameOnly = Contact(id = "c3", leadId = "l1", fullName = "مریم", telegram = "maryam", bale = "maryam")
    private val neither = Contact(id = "c4", leadId = "l1", fullName = "رضا")
    private val permissions = LeadPermissions()

    private fun evaluate(channel: String, contact: Contact?, installed: Boolean = true) =
        ChannelAvailabilityEvaluator.evaluate(channel, contact, permissions, messengerInstalled = installed).availability

    @Test
    fun `call requires a number`() {
        assertEquals(ChannelAvailability.ENABLED, evaluate(InteractionChannel.CALL, fullContact))
        assertEquals(ChannelAvailability.DISABLED_NO_NUMBER, evaluate(InteractionChannel.CALL, neither))
    }

    @Test
    fun `sms enabled when mobile present`() {
        assertEquals(ChannelAvailability.ENABLED, evaluate(InteractionChannel.SMS, fullContact))
        assertEquals(ChannelAvailability.DISABLED_NO_NUMBER, evaluate(InteractionChannel.SMS, neither))
    }

    @Test
    fun `email requires an email address`() {
        assertEquals(ChannelAvailability.ENABLED, evaluate(InteractionChannel.EMAIL, fullContact))
        assertEquals(ChannelAvailability.DISABLED_NO_EMAIL, evaluate(InteractionChannel.EMAIL, phoneOnly))
    }

    @Test
    fun `whatsapp falls back to mobile number and needs the app installed`() {
        assertEquals(ChannelAvailability.ENABLED, evaluate(InteractionChannel.WHATSAPP, phoneOnly))
        assertEquals(ChannelAvailability.DISABLED_NO_NUMBER, evaluate(InteractionChannel.WHATSAPP, neither))
        assertEquals(ChannelAvailability.DISABLED_NOT_INSTALLED, evaluate(InteractionChannel.WHATSAPP, phoneOnly, installed = false))
    }

    @Test
    fun `telegram and bale enable with phone only`() {
        assertEquals(ChannelAvailability.ENABLED, evaluate(InteractionChannel.TELEGRAM, phoneOnly))
        assertEquals(ChannelAvailability.ENABLED, evaluate(InteractionChannel.BALE, phoneOnly))
    }

    @Test
    fun `telegram and bale enable with username only`() {
        assertEquals(ChannelAvailability.ENABLED, evaluate(InteractionChannel.TELEGRAM, usernameOnly))
        assertEquals(ChannelAvailability.ENABLED, evaluate(InteractionChannel.BALE, usernameOnly))
    }

    @Test
    fun `telegram and bale enable with both phone and username`() {
        assertEquals(ChannelAvailability.ENABLED, evaluate(InteractionChannel.TELEGRAM, fullContact))
        assertEquals(ChannelAvailability.ENABLED, evaluate(InteractionChannel.BALE, fullContact))
    }

    @Test
    fun `telegram and bale disabled when neither phone nor username`() {
        // Clear disabled state, NOT save-contact.
        assertEquals(ChannelAvailability.DISABLED_NO_TARGET, evaluate(InteractionChannel.TELEGRAM, neither))
        assertEquals(ChannelAvailability.DISABLED_NO_TARGET, evaluate(InteractionChannel.BALE, neither))
    }

    @Test
    fun `telegram and bale disabled when the app is not installed`() {
        assertEquals(ChannelAvailability.DISABLED_NOT_INSTALLED, evaluate(InteractionChannel.TELEGRAM, phoneOnly, installed = false))
        assertEquals(ChannelAvailability.DISABLED_NOT_INSTALLED, evaluate(InteractionChannel.BALE, phoneOnly, installed = false))
    }

    @Test
    fun `eitaa and rubika are always save contact even with a username`() {
        assertEquals(ChannelAvailability.CONTACT_SAVE_REQUIRED, evaluate(InteractionChannel.EITAA, fullContact))
        assertEquals(ChannelAvailability.CONTACT_SAVE_REQUIRED, evaluate(InteractionChannel.RUBIKA, fullContact))
        assertEquals(ChannelAvailability.CONTACT_SAVE_REQUIRED, evaluate(InteractionChannel.EITAA, neither))
        assertEquals(ChannelAvailability.CONTACT_SAVE_REQUIRED, evaluate(InteractionChannel.RUBIKA, neither))
    }

    @Test
    fun `backend permissions gate call and message`() {
        val readOnly = LeadPermissions(canCall = false, canMessage = false)
        assertEquals(ChannelAvailability.DISABLED_FORBIDDEN, ChannelAvailabilityEvaluator.evaluate(InteractionChannel.CALL, fullContact, readOnly).availability)
        assertEquals(ChannelAvailability.DISABLED_FORBIDDEN, ChannelAvailabilityEvaluator.evaluate(InteractionChannel.WHATSAPP, fullContact, readOnly).availability)
        assertEquals(ChannelAvailability.DISABLED_FORBIDDEN, ChannelAvailabilityEvaluator.evaluate(InteractionChannel.RUBIKA, fullContact, readOnly).availability)
    }
}
