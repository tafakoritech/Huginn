package ir.tafakoritech.huginn.work

import androidx.work.Configuration
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import dagger.hilt.android.qualifiers.ApplicationContext
import java.util.concurrent.TimeUnit
import javax.inject.Inject
import javax.inject.Singleton

/** Schedules background workers at the right moments. */
@Singleton
class WorkScheduler @Inject constructor(
    @ApplicationContext private val context: android.content.Context,
) {

    /** Coarse periodic reminder cadence (respects Android background limits). */
    fun scheduleFollowUpReminders() {
        val request = PeriodicWorkRequestBuilder<FollowUpReminderWorker>(2, TimeUnit.HOURS)
            .setInitialDelay(15, TimeUnit.MINUTES)
            .build()
        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            FollowUpReminderWorker.WORK_NAME,
            ExistingPeriodicWorkPolicy.KEEP,
            request,
        )
    }

    fun cancelFollowUpReminders() {
        WorkManager.getInstance(context).cancelUniqueWork(FollowUpReminderWorker.WORK_NAME)
    }
}
