package ir.tafakoritech.huginn.feature.teleprompter

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.provider.Settings
import ir.tafakoritech.huginn.data.model.CallScenario

object TeleprompterManager {

    /** Checks if the app has permission to draw over other apps (SYSTEM_ALERT_WINDOW). */
    fun canDrawOverlays(context: Context): Boolean {
        return Settings.canDrawOverlays(context)
    }

    /** Intent to open the manage overlay permission settings for this package. */
    fun createOverlayPermissionIntent(context: Context): Intent {
        return Intent(
            Settings.ACTION_MANAGE_OVERLAY_PERMISSION,
            Uri.parse("package:${context.packageName}"),
        )
    }

    /** Starts the teleprompter overlay service with the given scenario. */
    fun start(
        context: Context,
        scenario: CallScenario,
        leadName: String? = null,
        contactName: String? = null,
    ) {
        if (!canDrawOverlays(context)) return
        val intent = Intent(context, TeleprompterService::class.java).apply {
            putExtra(TeleprompterService.EXTRA_OPENING, scenario.openingSentence)
            putExtra(TeleprompterService.EXTRA_INTRO, scenario.introduction)
            putExtra(TeleprompterService.EXTRA_REASON, scenario.reasonForCall)
            putExtra(TeleprompterService.EXTRA_VALUE, scenario.valueProposition)
            putStringArrayListExtra(TeleprompterService.EXTRA_QUESTIONS, ArrayList(scenario.discoveryQuestions))
            putStringArrayListExtra(TeleprompterService.EXTRA_OBJECTIONS, ArrayList(scenario.likelyObjections))
            putStringArrayListExtra(TeleprompterService.EXTRA_RESPONSES, ArrayList(scenario.objectionResponses))
            putExtra(TeleprompterService.EXTRA_NEXT_STEP, scenario.recommendedNextStep)
            putExtra(TeleprompterService.EXTRA_CLOSING, scenario.closingLine)
            putExtra(TeleprompterService.EXTRA_LEAD_NAME, leadName)
            putExtra(TeleprompterService.EXTRA_CONTACT_NAME, contactName)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            context.startService(intent)
        } else {
            context.startService(intent)
        }
    }

    /** Stops the teleprompter overlay. */
    fun stop(context: Context) {
        val intent = Intent(context, TeleprompterService::class.java)
        context.stopService(intent)
    }
}
