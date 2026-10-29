package ir.tafakoritech.huginn.work

import android.content.Context
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import ir.tafakoritech.huginn.core.api.ApiCaller
import ir.tafakoritech.huginn.core.api.HuginnApi
import ir.tafakoritech.huginn.core.database.PendingReportDao
import ir.tafakoritech.huginn.core.network.ApiErrorCodes
import ir.tafakoritech.huginn.core.network.ApiException
import ir.tafakoritech.huginn.core.util.SafeLog
import ir.tafakoritech.huginn.data.model.ReportSubmitRequest
import kotlinx.serialization.json.Json

/**
 * Retries locally queued mandatory reports until the backend confirms them.
 * The Room queue entry is only removed after a successful API submission.
 */
@HiltWorker
class ReportSyncWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val api: HuginnApi,
    private val pendingReportDao: PendingReportDao,
    private val json: Json,
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        val pending = pendingReportDao.getAll()
        if (pending.isEmpty()) return Result.success()

        for (item in pending) {
            val request = runCatching {
                json.decodeFromString(ReportSubmitRequest.serializer(), item.reportJson)
            }.getOrNull() ?: run {
                pendingReportDao.delete(item.interactionId)
                continue
            }
            try {
                ApiCaller.call(json) { api.submitReport(item.interactionId, request) }
                pendingReportDao.delete(item.interactionId)
                SafeLog.d("ReportSyncWorker: submitted queued report ${item.interactionId}")
            } catch (e: ApiException) {
                when {
                    e.code == ApiErrorCodes.UNAUTHORIZED || e.httpStatus == 401 ->
                        return Result.failure() // session invalid; do not spin
                    e.httpStatus in 400..499 && e.httpStatus != 429 -> {
                        // Permanently rejected (validation/state) — remove to avoid a retry loop.
                        SafeLog.w("ReportSyncWorker: permanent failure for ${item.interactionId}: ${e.code}")
                        pendingReportDao.delete(item.interactionId)
                    }
                    else -> return Result.retry()
                }
            } catch (e: Throwable) {
                return Result.retry()
            }
        }
        return Result.success()
    }

    companion object {
        const val WORK_NAME = "huginn_report_sync"
    }
}
