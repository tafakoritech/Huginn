package ir.tafakoritech.huginn.core.external

import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.provider.ContactsContract
import android.util.Log
import ir.tafakoritech.huginn.data.model.DeepLink
import ir.tafakoritech.huginn.data.model.InteractionChannel

/**
 * Centralized external app launchers. All Intent creation for communication
 * actions lives here (never inside ViewModels).
 *
 * Privacy: no contact-reading permissions are requested. Saving a lead to
 * Android contacts delegates to the system Contacts app via ACTION_INSERT.
 */

sealed interface LaunchResult {
    data object Launched : LaunchResult
    data object ContactSaveRequired : LaunchResult
    data object NotInstalled : LaunchResult
}

interface CallLauncher {
    /** Opens the dialer with the number pre-filled (ACTION_DIAL, no CALL_PHONE permission). */
    suspend fun launch(phone: String): LaunchResult
}

interface SmsLauncher {
    /** Opens the messaging app with destination + body pre-filled. Never auto-sends. */
    suspend fun launch(number: String, text: String): LaunchResult
}

interface MessengerLauncher {
    /**
     * Opens the target messenger for the given channel. [phone] is the contact's
     * (normalized) mobile number, used to build phone-based Telegram/Bale links.
     */
    suspend fun launch(channel: String, deepLink: DeepLink?, phone: String?, text: String?): LaunchResult
}

interface EmailLauncher {
    /** Opens the email client pre-filled with recipient, subject and body. */
    suspend fun launch(recipient: String?, subject: String?, body: String?): LaunchResult
}

interface ContactInsertLauncher {
    /** Opens the system Contacts app to insert a new contact (user confirms). */
    suspend fun launch(
        name: String,
        phone: String?,
        company: String?,
        jobTitle: String?,
        email: String?,
    ): LaunchResult
}

/**
 * Known messenger packages and their web fallbacks. Availability is checked
 * with PackageManager so unavailable channels can be disabled in the UI.
 */
object MessengerChannels {
    data class AppInfo(
        val channel: String,
        val candidates: List<String>,
        val webUrl: String?,
    )

    val known = mapOf(
        InteractionChannel.WHATSAPP to AppInfo(InteractionChannel.WHATSAPP, listOf("com.whatsapp"), "https://web.whatsapp.com/"),
        InteractionChannel.TELEGRAM to AppInfo(InteractionChannel.TELEGRAM, listOf("org.telegram.messenger", "org.telegram.messenger.web"), "https://web.telegram.org/"),
        InteractionChannel.BALE to AppInfo(InteractionChannel.BALE, listOf("ir.bale.app", "ir.pad.bale"), "https://ble.ir/"),
        InteractionChannel.EITAA to AppInfo(InteractionChannel.EITAA, listOf("ir.jobinja.eitaa"), "https://eitaa.com/"),
        InteractionChannel.RUBIKA to AppInfo(InteractionChannel.RUBIKA, listOf("ir.irzo", "chat.rubika.ir"), "https://rubika.ir/"),
    )

    fun isInstalled(context: Context, channel: String): Boolean {
        val info = known[channel] ?: return false
        return info.candidates.any { installed(context, it) }
    }

    fun installed(context: Context, packageName: String): Boolean =
        runCatching {
            context.packageManager.getPackageInfo(packageName, 0)
            true
        }.getOrDefault(false)
}

class SystemIntentCallLauncher(
    private val context: Context,
) : CallLauncher {
    override suspend fun launch(phone: String): LaunchResult {
        val intent = Intent(Intent.ACTION_DIAL, Uri.parse("tel:${phone.filter { it.isDigit() || it == '+' }}"))
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        return if (context.startSafely(intent)) LaunchResult.Launched else LaunchResult.NotInstalled
    }
}

class SystemSmsLauncher(
    private val context: Context,
) : SmsLauncher {
    override suspend fun launch(number: String, text: String): LaunchResult {
        val intent = Intent(Intent.ACTION_SENDTO, Uri.parse("smsto:$number"))
            .putExtra(Intent.EXTRA_TEXT, text)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        return if (context.startSafely(intent)) LaunchResult.Launched else LaunchResult.NotInstalled
    }
}

class SystemEmailLauncher(
    private val context: Context,
) : EmailLauncher {
    override suspend fun launch(recipient: String?, subject: String?, body: String?): LaunchResult {
        val uri = Uri.Builder().scheme("mailto").opaquePart(recipient.orEmpty()).build()
        val intent = Intent(Intent.ACTION_SENDTO, uri)
            .putExtra(Intent.EXTRA_SUBJECT, subject.orEmpty())
            .putExtra(Intent.EXTRA_TEXT, body.orEmpty())
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        return if (context.startSafely(intent)) LaunchResult.Launched else LaunchResult.NotInstalled
    }
}

class SystemMessengerLauncher(
    private val context: Context,
) : MessengerLauncher {

    override suspend fun launch(channel: String, deepLink: DeepLink?, phone: String?, text: String?): LaunchResult {
        // Rubika/Eitaa are intentionally save-contact only — never a direct chat.
        if (MessengerPolicy.requiresContactSave(channel)) {
            return LaunchResult.ContactSaveRequired
        }
        if (!MessengerPolicy.supportsDirectChat(channel)) {
            return LaunchResult.NotInstalled
        }

        // Direct chat requires the app to be installed; otherwise a clear error.
        if (!MessengerChannels.isInstalled(context, channel)) {
            return LaunchResult.NotInstalled
        }

        val url = MessengerDeepLinks.directUrl(channel, deepLink, phone, text)
            ?: return LaunchResult.NotInstalled

        val intent = Intent(Intent.ACTION_VIEW, Uri.parse(url)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        return if (context.startSafely(intent)) LaunchResult.Launched else LaunchResult.NotInstalled
    }
}

/** Starts an external intent if any activity can handle it. */
internal fun Context.startSafely(intent: Intent): Boolean =
    runCatching {
        if (intent.resolveActivity(packageManager) != null) {
            startActivity(intent)
            true
        } else {
            false
        }
    }.getOrElse {
        Log.w("ExternalLauncher", "start failed", it)
        false
    }

class SystemContactInsertLauncher(
    private val context: Context,
) : ContactInsertLauncher {
    override suspend fun launch(
        name: String,
        phone: String?,
        company: String?,
        jobTitle: String?,
        email: String?,
    ): LaunchResult {
        val intent = Intent(Intent.ACTION_INSERT, ContactsContract.Contacts.CONTENT_URI).apply {
            putExtra(ContactsContract.Intents.Insert.NAME, name)
            if (!phone.isNullOrBlank()) {
                putExtra(ContactsContract.Intents.Insert.PHONE, phone)
                putExtra(ContactsContract.Intents.Insert.PHONE_TYPE, ContactsContract.CommonDataKinds.Phone.TYPE_MOBILE)
            }
            if (!email.isNullOrBlank()) {
                putExtra(ContactsContract.Intents.Insert.EMAIL, email)
                putExtra(ContactsContract.Intents.Insert.EMAIL_TYPE, ContactsContract.CommonDataKinds.Email.TYPE_WORK)
            }
            company?.takeIf { it.isNotBlank() }?.let { putExtra(ContactsContract.Intents.Insert.COMPANY, it) }
            jobTitle?.takeIf { it.isNotBlank() }?.let { putExtra(ContactsContract.Intents.Insert.JOB_TITLE, it) }
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        return if (runCatching { context.startActivity(intent) }.isSuccess) LaunchResult.Launched else LaunchResult.NotInstalled
    }
}

/** Validates and normalizes a huginn:// deep link into a navigation route. */
object DeepLinkValidator {
    private val PATTERN = Regex("^huginn://([a-z]+)/([A-Za-z0-9_-]{1,100})$")

    fun parse(raw: String): String? {
        val match = PATTERN.matchEntire(raw.trim()) ?: return null
        val host = match.groupValues[1]
        val id = match.groupValues[2]
        return when (host) {
            "lead" -> "lead/$id"
            "report" -> "mandatory-report/$id"
            "followup" -> "followups"
            else -> null
        }
    }
}
