package ir.tafakoritech.huginn.work

import android.content.Context
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.datastore.SettingsRepository
import ir.tafakoritech.huginn.core.locale.LanguageResolver
import ir.tafakoritech.huginn.core.notifications.NotificationHelper
import ir.tafakoritech.huginn.core.util.DateTimeUtil
import ir.tafakoritech.huginn.core.util.SafeLog
import ir.tafakoritech.huginn.core.util.getOrNull
import ir.tafakoritech.huginn.data.repository.FollowUpRepository
import ir.tafakoritech.huginn.data.model.FollowUpStatus
import kotlinx.coroutines.flow.first

/**
 * Local follow-up reminders. Respects Android background limits: runs on a
 * coarse periodic cadence with a network constraint, never constant polling.
 * (Server push can replace this later through NotificationHelper.)
 */
@HiltWorker
class FollowUpReminderWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val followUpRepository: FollowUpRepository,
    private val notificationHelper: NotificationHelper,
    private val settingsRepository: SettingsRepository,
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        if (!notificationHelper.canNotify()) return Result.success()
        // A fresh WorkManager process has no default locale from the UI; resolve
        // the persisted language so relative labels are formatted correctly.
        val lang = settingsRepository.language.first() ?: LanguageResolver.fromLocale()
        DateTimeUtil.setLanguage(lang)
        val today = followUpRepository.myFollowUps(view = "today")
        val overdue = followUpRepository.myFollowUps(view = "overdue")
        val shown = mutableSetOf<String>()
        today.getOrNull()?.items?.forEach { fu ->
            if (shown.add(fu.leadId)) {
                val company = fu.companyName ?: fu.leadId
                val body = (fu.description ?: "").trim()
                notificationHelper.notifyFollowUp(
                    company = company,
                    body = body.ifBlank { applicationContext.getString(R.string.notif_followup_due, DateTimeUtil.formatRelative(fu.dueAt)) },
                    notificationId = (fu.leadId.hashCode() and 0x7fffffff) % 100000,
                )
            }
        }
        overdue.getOrNull()?.items?.forEach { fu ->
            if (fu.isOverdue && shown.add(fu.leadId)) {
                val company = fu.companyName ?: fu.leadId
                notificationHelper.notifyFollowUp(
                    company = company,
                    body = applicationContext.getString(R.string.notif_followup_overdue, DateTimeUtil.formatRelative(fu.dueAt)),
                    notificationId = ((fu.leadId.hashCode() and 0x7fffffff) % 100000) + 50000,
                )
            }
        }
        SafeLog.d("FollowUpReminderWorker: notified ${shown.size} leads")
        return Result.success()
    }

    companion object {
        const val WORK_NAME = "huginn_followup_reminders"
    }
}
