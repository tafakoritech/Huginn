package ir.tafakoritech.huginn.feature.messaging

import ir.tafakoritech.huginn.core.util.PhoneFormatter
import ir.tafakoritech.huginn.data.model.Contact
import ir.tafakoritech.huginn.data.model.InteractionChannel
import ir.tafakoritech.huginn.data.model.LeadPermissions

/** Per-channel availability for the communication sheet. */
enum class ChannelAvailability {
    ENABLED,
    /** No phone number recorded for this contact. */
    DISABLED_NO_NUMBER,
    /** No email recorded for this contact. */
    DISABLED_NO_EMAIL,
    /** The channel is save-contact only (Rubika/Eitaa): never a direct chat. */
    CONTACT_SAVE_REQUIRED,
    /** Backend forbids this channel for the current employee. */
    DISABLED_FORBIDDEN,
    /** Messenger not installed. */
    DISABLED_NOT_INSTALLED,
    /** No phone number and no username recorded (Telegram/Bale). */
    DISABLED_NO_TARGET,
}

data class ChannelStatus(
    val availability: ChannelAvailability,
    /** Deep link type the backend would produce for this channel (e.g. CONTACT_SAVE_REQUIRED). */
    val deepLinkType: String? = null,
)

/**
 * Pure channel-state evaluator (unit-tested). Business decisions still come
 * from the backend permissions object; this only turns contact data into
 * UI-visible availability.
 *
 * Messaging policy:
 *  - Rubika, Eitaa  -> save-contact only, never a direct chat/deep-link.
 *  - WhatsApp, Telegram, Bale -> direct chat when the target + app are
 *    available; otherwise a clear disabled state (never save-contact).
 */
object ChannelAvailabilityEvaluator {

    fun evaluate(
        channel: String,
        contact: Contact?,
        permissions: LeadPermissions,
        messengerInstalled: Boolean = true,
    ): ChannelStatus {
        val canMessage = permissions.canMessage
        return when (channel) {
            InteractionChannel.CALL -> {
                if (!permissions.canCall) return ChannelStatus(ChannelAvailability.DISABLED_FORBIDDEN)
                if (contact?.primaryNumber == null) ChannelStatus(ChannelAvailability.DISABLED_NO_NUMBER)
                else ChannelStatus(ChannelAvailability.ENABLED)
            }

            InteractionChannel.SMS -> {
                if (!canMessage) return ChannelStatus(ChannelAvailability.DISABLED_FORBIDDEN)
                if (contact?.primaryNumber == null) ChannelStatus(ChannelAvailability.DISABLED_NO_NUMBER)
                else ChannelStatus(ChannelAvailability.ENABLED)
            }

            InteractionChannel.EMAIL -> {
                if (!canMessage) return ChannelStatus(ChannelAvailability.DISABLED_FORBIDDEN)
                if (contact?.email.isNullOrBlank()) ChannelStatus(ChannelAvailability.DISABLED_NO_EMAIL)
                else ChannelStatus(ChannelAvailability.ENABLED)
            }

            InteractionChannel.WHATSAPP -> {
                if (!canMessage) return ChannelStatus(ChannelAvailability.DISABLED_FORBIDDEN)
                val target = contact?.whatsapp ?: contact?.mobileNormalized ?: contact?.mobile
                when {
                    target.isNullOrBlank() -> ChannelStatus(ChannelAvailability.DISABLED_NO_NUMBER)
                    !messengerInstalled -> ChannelStatus(ChannelAvailability.DISABLED_NOT_INSTALLED)
                    else -> ChannelStatus(ChannelAvailability.ENABLED)
                }
            }

            // Save-contact only: the employee finds the person inside Eitaa/Rubika.
            InteractionChannel.EITAA,
            InteractionChannel.RUBIKA,
            -> {
                if (!canMessage) return ChannelStatus(ChannelAvailability.DISABLED_FORBIDDEN)
                ChannelStatus(ChannelAvailability.CONTACT_SAVE_REQUIRED)
            }

            InteractionChannel.TELEGRAM,
            InteractionChannel.BALE,
            -> {
                if (!canMessage) return ChannelStatus(ChannelAvailability.DISABLED_FORBIDDEN)
                val username = when (channel) {
                    InteractionChannel.TELEGRAM -> contact?.telegram
                    InteractionChannel.BALE -> contact?.bale
                    else -> null
                }
                // A valid mobile number OR a username enables the action; the
                // phone-based link is preferred when both exist.
                val hasPhone = contact?.primaryNumber
                    ?.let { PhoneFormatter.toInternationalMobile(it) != null } == true
                when {
                    !hasPhone && username.isNullOrBlank() -> ChannelStatus(ChannelAvailability.DISABLED_NO_TARGET)
                    !messengerInstalled -> ChannelStatus(ChannelAvailability.DISABLED_NOT_INSTALLED)
                    else -> ChannelStatus(ChannelAvailability.ENABLED)
                }
            }

            else -> ChannelStatus(ChannelAvailability.DISABLED_FORBIDDEN)
        }
    }
}
