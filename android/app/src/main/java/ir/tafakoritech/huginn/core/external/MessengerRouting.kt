package ir.tafakoritech.huginn.core.external

import ir.tafakoritech.huginn.core.util.PhoneFormatter
import ir.tafakoritech.huginn.data.model.DeepLink
import ir.tafakoritech.huginn.data.model.InteractionChannel
import java.net.URLEncoder

/**
 * Pure, unit-testable messaging policy. Kept free of Android dependencies so the
 * per-app routing rules can be tested without a device.
 *
 * Rules:
 *  - Rubika / Eitaa  -> save-contact ONLY. A direct chat/deep-link is never
 *    attempted, even when a username exists.
 *  - WhatsApp / Telegram / Bale -> direct chat. For Telegram/Bale a valid mobile
 *    number (normalized to +98...) OR a username enables the action; the phone
 *    is preferred when both exist. When neither exists a null URL is returned
 *    (the caller surfaces a clear unavailable state).
 */
object MessengerPolicy {

    /** Channels that are intentionally save-contact-only. */
    fun requiresContactSave(channel: String): Boolean =
        channel == InteractionChannel.RUBIKA || channel == InteractionChannel.EITAA

    /** Channels that support a direct open/chat intent. */
    fun supportsDirectChat(channel: String): Boolean =
        channel == InteractionChannel.WHATSAPP ||
            channel == InteractionChannel.TELEGRAM ||
            channel == InteractionChannel.BALE
}

/**
 * Builds the direct open/chat URL for a channel. Returns null when no valid
 * target exists.
 *
 * Telegram/Bale: a valid mobile number produces the phone-based link
 * (https://t.me/+98..., https://ble.ir/+98...), preferring the phone over a
 * recorded username; without a phone the username deep link is the fallback.
 */
object MessengerDeepLinks {

    fun directUrl(channel: String, deepLink: DeepLink?, phone: String?, text: String?): String? {
        if (MessengerPolicy.requiresContactSave(channel)) return null
        val link = deepLink
        return when (channel) {
            InteractionChannel.WHATSAPP -> {
                val base = link?.target?.takeIf { it.isNotBlank() }
                    ?: link?.username?.let { DeepLink.waMe(it) }
                    ?: return null
                val withText = text?.takeIf { it.isNotBlank() }?.let {
                    if (base.contains("?")) "$base&text=${URLEncoder.encode(it, "UTF-8")}"
                    else "$base?text=${URLEncoder.encode(it, "UTF-8")}"
                } ?: base
                withText
            }

            InteractionChannel.TELEGRAM -> {
                val intl = PhoneFormatter.toInternationalMobile(phone)
                when {
                    intl != null -> "https://t.me/$intl"
                    !link?.username.isNullOrBlank() -> "tg://resolve?domain=${link.username}"
                    else -> null
                }
            }

            // Bale is the same platform as ble.ir — the phone-based link uses the
            // ble.ir domain, matching the username fallback's web identity.
            InteractionChannel.BALE -> {
                val intl = PhoneFormatter.toInternationalMobile(phone)
                when {
                    intl != null -> "https://ble.ir/$intl"
                    !link?.username.isNullOrBlank() -> "bale://profile/${link.username}"
                    else -> null
                }
            }

            else -> null
        }
    }
}
