package ir.tafakoritech.huginn.core.permissions

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.content.ContextCompat
import dagger.hilt.android.qualifiers.ApplicationContext
import ir.tafakoritech.huginn.R
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Minimal, explicit permission handling. Only two runtime permissions are ever
 * requested: POST_NOTIFICATIONS (Android 13+ when the user enables reminders)
 * and RECORD_AUDIO (only when the user taps the voice-input microphone).
 * Contact/SMS/call-log/phone-state permissions are deliberately never requested.
 */
@Singleton
class PermissionsManager @Inject constructor(
    @ApplicationContext private val context: Context,
) {

    @Suppress("InlinedApi")
    val notificationPermission: String = Manifest.permission.POST_NOTIFICATIONS

    val recordAudioPermission: String = Manifest.permission.RECORD_AUDIO

    fun isGranted(permission: String): Boolean =
        ContextCompat.checkSelfPermission(context, permission) == PackageManager.PERMISSION_GRANTED

    fun isNotificationPermissionGranted(): Boolean =
        Build.VERSION.SDK_INT < 33 || isGranted(notificationPermission)

    fun shouldRequestNotificationPermission(): Boolean =
        Build.VERSION.SDK_INT >= 33 && !isNotificationPermissionGranted()

    fun isRecordAudioGranted(): Boolean = isGranted(recordAudioPermission)

    fun shouldRequestRecordAudio(): Boolean = !isRecordAudioGranted()

    /** Permission -> Persian/English explanation keys for the UI. */
    fun rationaleKey(permission: String): Int? = when (permission) {
        Manifest.permission.POST_NOTIFICATIONS -> R.string.notification_permission_text
        Manifest.permission.RECORD_AUDIO -> R.string.record_audio_permission_text
        else -> null
    }

    companion object {
        // Runtime permissions this app can request, each only when its feature
        // is actively used by the user.
        val RUNTIME_PERMISSIONS = listOf(
            "android.permission.POST_NOTIFICATIONS",
            "android.permission.RECORD_AUDIO",
        )
    }
}
