package ir.tafakoritech.huginn

import android.content.Context
import android.content.Intent
import android.content.pm.ActivityInfo
import android.content.pm.PackageInfo
import android.content.pm.ResolveInfo
import android.net.Uri
import ir.tafakoritech.huginn.core.external.LaunchResult
import ir.tafakoritech.huginn.core.external.MessengerChannels
import ir.tafakoritech.huginn.core.external.SystemMessengerLauncher
import ir.tafakoritech.huginn.data.model.DeepLink
import ir.tafakoritech.huginn.data.model.InteractionChannel
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config

/**
 * Full runtime path for Telegram/Bale phone-based links:
 * messenger installed -> SystemMessengerLauncher -> normalized +98 number ->
 * generated link -> ACTION_VIEW Android Intent.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class SystemMessengerLauncherTest {

    /** Marks the messenger as installed so [MessengerChannels.isInstalled] passes. */
    private fun install(packageName: String) {
        shadowOf(RuntimeEnvironment.getApplication().packageManager)
            .addPackage(PackageInfo().apply { this.packageName = packageName })
    }

    /** Registers a web handler for the exact URL the launcher will open. */
    private fun registerWebResolver(context: Context, url: String) {
        val pm = shadowOf(context.packageManager)
        val intent = Intent(Intent.ACTION_VIEW, Uri.parse(url))
        pm.addResolveInfoForIntent(
            intent,
            ResolveInfo().apply {
                activityInfo = ActivityInfo().apply {
                    this.packageName = "org.example.browser"
                    name = "org.example.browser.BrowserActivity"
                }
            },
        )
    }

    private fun noUsernameDeepLink(channel: String): DeepLink =
        DeepLink(channel = channel, type = "username", target = "", username = null)

    @Test
    fun `telegram phone-only opens an ACTION_VIEW intent with t me phone link`() = runBlocking {
        install(MessengerChannels.known[InteractionChannel.TELEGRAM]!!.candidates.first())
        val context = RuntimeEnvironment.getApplication()
        registerWebResolver(context, "https://t.me/+989121234567")
        val launcher = SystemMessengerLauncher(context)

        val result = launcher.launch(InteractionChannel.TELEGRAM, noUsernameDeepLink(InteractionChannel.TELEGRAM), phone = "09121234567", text = null)

        assertEquals(LaunchResult.Launched, result)
        val intent = shadowOf(context).nextStartedActivity
        assertNotNull("an intent must be started", intent)
        assertEquals(Intent.ACTION_VIEW, intent.action)
        assertEquals("https://t.me/+989121234567", intent.dataString)
    }

    @Test
    fun `bale phone-only opens an ACTION_VIEW intent with ble ir phone link`() = runBlocking {
        install(MessengerChannels.known[InteractionChannel.BALE]!!.candidates.first())
        val context = RuntimeEnvironment.getApplication()
        registerWebResolver(context, "https://ble.ir/+989121234567")
        val launcher = SystemMessengerLauncher(context)

        val result = launcher.launch(InteractionChannel.BALE, noUsernameDeepLink(InteractionChannel.BALE), phone = "09121234567", text = null)

        assertEquals(LaunchResult.Launched, result)
        val intent = shadowOf(context).nextStartedActivity
        assertNotNull("an intent must be started", intent)
        assertEquals(Intent.ACTION_VIEW, intent.action)
        assertEquals("https://ble.ir/+989121234567", intent.dataString)
    }

    @Test
    fun `neither phone nor username does not launch anything`() = runBlocking {
        install(MessengerChannels.known[InteractionChannel.TELEGRAM]!!.candidates.first())
        val context = RuntimeEnvironment.getApplication()
        val launcher = SystemMessengerLauncher(context)

        val result = launcher.launch(InteractionChannel.TELEGRAM, noUsernameDeepLink(InteractionChannel.TELEGRAM), phone = null, text = null)

        assertEquals(LaunchResult.NotInstalled, result)
        assertNull(shadowOf(context).nextStartedActivity)
    }

    @Test
    fun `messenger not installed is reported even with a phone`() = runBlocking {
        // Do not install the app: the phone exists but the messenger is missing.
        val context = RuntimeEnvironment.getApplication()
        val launcher = SystemMessengerLauncher(context)

        val result = launcher.launch(InteractionChannel.TELEGRAM, noUsernameDeepLink(InteractionChannel.TELEGRAM), phone = "09121234567", text = null)

        assertEquals(LaunchResult.NotInstalled, result)
        assertTrue(MessengerChannels.isInstalled(context, InteractionChannel.TELEGRAM).not())
    }
}
