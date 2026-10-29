package ir.tafakoritech.huginn.data.repository

import ir.tafakoritech.huginn.core.api.ApiCaller
import ir.tafakoritech.huginn.core.api.HuginnApi
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.runCatchingAppError
import ir.tafakoritech.huginn.data.model.CompleteNextActionRequest
import ir.tafakoritech.huginn.data.model.MeStats
import ir.tafakoritech.huginn.data.model.NextActionsResponse
import ir.tafakoritech.huginn.data.model.PendingReportsResponse
import ir.tafakoritech.huginn.data.model.SnoozeNextActionRequest
import kotlinx.serialization.json.Json
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Next Best Action / work queue source. Every value comes from the real Huginn
 * API; the backend owns the ranking, this repository only fetches and executes.
 */
@Singleton
class WorkQueueRepository @Inject constructor(
    private val api: HuginnApi,
    private val json: Json,
) {

    /** Ranked next actions for the current user (backend single source of truth). */
    suspend fun nextActions(limit: Int? = null): AppResult<NextActionsResponse> =
        runCatchingAppError { ApiCaller.call(json) { api.nextActions(limit) } }

    suspend fun snooze(actionId: String, untilIso: String): AppResult<Unit> =
        runCatchingAppError { ApiCaller.call(json) { api.snoozeNextAction(actionId, SnoozeNextActionRequest(untilIso)) } }

    suspend fun complete(actionId: String): AppResult<Unit> =
        runCatchingAppError { ApiCaller.call(json) { api.completeNextAction(actionId, CompleteNextActionRequest()) } }

    /** Home/report KPIs from the real backend agent metrics (GET /dashboard/me). */
    suspend fun me(): AppResult<MeStats> =
        runCatchingAppError { ApiCaller.call(json) { api.dashboardMe().toMeStats() } }

    /** Source of truth for the mandatory-report gate (checked on app resume). */
    suspend fun pendingReports(): AppResult<PendingReportsResponse> =
        runCatchingAppError { ApiCaller.call(json) { api.pendingReports() } }
}
