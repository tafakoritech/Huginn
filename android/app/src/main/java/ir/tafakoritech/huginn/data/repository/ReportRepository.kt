package ir.tafakoritech.huginn.data.repository

import android.content.Context
import androidx.work.BackoffPolicy
import androidx.work.ExistingWorkPolicy
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import dagger.hilt.android.qualifiers.ApplicationContext
import ir.tafakoritech.huginn.core.api.ApiCaller
import ir.tafakoritech.huginn.core.api.HuginnApi
import ir.tafakoritech.huginn.core.database.PendingReportDao
import ir.tafakoritech.huginn.core.database.PendingReportLocalEntity
import ir.tafakoritech.huginn.core.network.toAppError
import ir.tafakoritech.huginn.core.util.AppError
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.SafeLog
import ir.tafakoritech.huginn.core.util.runCatchingAppError
import ir.tafakoritech.huginn.data.model.ReportSubmitRequest
import ir.tafakoritech.huginn.data.model.ReportSubmitResponse
import ir.tafakoritech.huginn.data.model.ReportsResponse
import ir.tafakoritech.huginn.work.ReportSyncWorker
import kotlinx.serialization.json.Json
import java.util.concurrent.TimeUnit
import javax.inject.Inject
import javax.inject.Singleton

/** Outcome of a report submission attempt. */
sealed interface ReportSubmissionResult {
    data class Submitted(val response: ReportSubmitResponse) : ReportSubmissionResult

    /** Saved locally; will be sent when connectivity returns. */
    data class Queued(val interactionId: String) : ReportSubmissionResult

    data class Failed(val error: AppError) : ReportSubmissionResult
}

/**
 * Submits mandatory contact reports.
 *
 * Online path: POST /interactions/{id}/report.
 * Offline path: the report is persisted to the Room queue and a WorkManager
 * worker retries it. Until the backend confirms submission, the workflow stays
 * pending (never treated as completed merely because it exists locally).
 */
@Singleton
class ReportRepository @Inject constructor(
    private val api: HuginnApi,
    private val pendingReportDao: PendingReportDao,
    private val json: Json,
    @ApplicationContext private val context: Context,
) {

    suspend fun submitReport(interactionId: String, request: ReportSubmitRequest): ReportSubmissionResult {        return try {
            val response = ApiCaller.call(json) { api.submitReport(interactionId, request) }
            pendingReportDao.delete(interactionId)
            ReportSubmissionResult.Submitted(response)
        } catch (e: Throwable) {
            val appError = (e as? ir.tafakoritech.huginn.core.network.ApiException)?.toAppError() ?: AppError.Network
            if (appError == AppError.Network) {
                queueLocally(interactionId, request)
                ReportSubmissionResult.Queued(interactionId)
            } else {
                ReportSubmissionResult.Failed(appError)
            }
        }
    }

    private suspend fun queueLocally(interactionId: String, request: ReportSubmitRequest) {
        val serialized = json.encodeToString(ReportSubmitRequest.serializer(), request)
        pendingReportDao.upsert(
            PendingReportLocalEntity(
                interactionId = interactionId,
                leadId = "",
                reportJson = serialized,
            ),
        )
        SafeLog.d("ReportRepository: report queued locally for interaction $interactionId")
        enqueueSync()
    }

    /** Returns locally queued reports that still await backend confirmation. */
    suspend fun pendingLocalReports(): List<PendingReportLocalEntity> = pendingReportDao.getAll()

    /**
     * Submitted contact reports visible to the current user (GET /me/reports).
     * The backend is the single source of truth for the global Reports screen.
     */
    suspend fun myReports(): AppResult<ReportsResponse> =
        runCatchingAppError { ApiCaller.call(json) { api.myReports() } }

    suspend fun hasPendingLocalReports(): Boolean = pendingReportDao.count() > 0

    fun enqueueSync() {
        val request = OneTimeWorkRequestBuilder<ReportSyncWorker>()
            .setBackoffCriteria(BackoffPolicy.EXPONENTIAL, 30, TimeUnit.SECONDS)
            .build()
        WorkManager.getInstance(context).enqueueUniqueWork(
            ReportSyncWorker.WORK_NAME,
            ExistingWorkPolicy.APPEND_OR_REPLACE,
            request,
        )
    }
}
