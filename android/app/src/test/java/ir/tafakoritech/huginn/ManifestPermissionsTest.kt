package ir.tafakoritech.huginn

import android.Manifest
import android.content.pm.PackageManager
import ir.tafakoritech.huginn.core.permissions.PermissionsManager
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import org.junit.runner.RunWith

/**
 * Verifies the minimal-permission philosophy: the packaged manifest never
 * requests contact/SMS/call-log/media permissions. RECORD_AUDIO is allowed
 * solely for the user-initiated voice-typing feature.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class ManifestPermissionsTest {

    private val forbidden = setOf(
        Manifest.permission.READ_CONTACTS,
        Manifest.permission.WRITE_CONTACTS,
        Manifest.permission.READ_SMS,
        Manifest.permission.READ_CALL_LOG,
        Manifest.permission.WRITE_CALL_LOG,
        Manifest.permission.READ_PHONE_STATE,
        Manifest.permission.READ_MEDIA_IMAGES,
        Manifest.permission.READ_MEDIA_VIDEO,
    )

    private fun requestedPermissions(): Set<String> {
        val context = RuntimeEnvironment.getApplication()
        val info = context.packageManager.getPackageInfo(
            context.packageName,
            PackageManager.GET_PERMISSIONS,
        )
        return info.requestedPermissions?.toSet() ?: emptySet()
    }

    @Test
    fun `manifest does not request any forbidden permission`() {
        val overlap = requestedPermissions() intersect forbidden
        assertTrue("Forbidden permissions present: $overlap", overlap.isEmpty())
    }

    @Test
    fun `internet permission is present`() {
        assertTrue(requestedPermissions().contains(Manifest.permission.INTERNET))
    }

    @Test
    fun `record audio is declared for voice input`() {
        assertTrue(requestedPermissions().contains(Manifest.permission.RECORD_AUDIO))
    }

    @Test
    fun `only notifications and record audio are runtime requested by code`() {
        assertEquals(
            setOf(
                Manifest.permission.POST_NOTIFICATIONS,
                Manifest.permission.RECORD_AUDIO,
            ),
            PermissionsManager.RUNTIME_PERMISSIONS.toSet(),
        )
    }
}
