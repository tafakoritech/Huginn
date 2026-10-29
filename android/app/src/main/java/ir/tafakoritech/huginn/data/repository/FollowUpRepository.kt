package ir.tafakoritech.huginn.data.repository

import ir.tafakoritech.huginn.core.api.ApiCaller
import ir.tafakoritech.huginn.core.api.HuginnApi
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.runCatchingAppError
import ir.tafakoritech.huginn.data.model.FollowUp
import ir.tafakoritech.huginn.data.model.FollowUpCreateRequest
import ir.tafakoritech.huginn.data.model.FollowUpRescheduleRequest
import ir.tafakoritech.huginn.data.model.FollowUpUpdateRequest
import ir.tafakoritech.huginn.data.model.FollowUpsResponse
import kotlinx.serialization.json.Json
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class FollowUpRepository @Inject constructor(
    private val api: HuginnApi,
    private val json: Json,
) {

    suspend fun create(request: FollowUpCreateRequest): AppResult<FollowUp> =
        runCatchingAppError { ApiCaller.call(json) { api.createFollowUp(request) } }

    suspend fun get(id: String): AppResult<FollowUp> =
        runCatchingAppError { ApiCaller.call(json) { api.followUp(id) } }

    suspend fun update(id: String, request: FollowUpUpdateRequest): AppResult<FollowUp> =
        runCatchingAppError { ApiCaller.call(json) { api.updateFollowUp(id, request) } }

    suspend fun complete(id: String): AppResult<Unit> =
        runCatchingAppError { ApiCaller.call(json) { api.completeFollowUp(id) } }

    suspend fun reopen(id: String): AppResult<FollowUp> =
        runCatchingAppError { ApiCaller.call(json) { api.reopenFollowUp(id) } }

    suspend fun reschedule(id: String, dueAtIso: String): AppResult<FollowUp> =
        runCatchingAppError { ApiCaller.call(json) { api.rescheduleFollowUp(id, FollowUpRescheduleRequest(dueAtIso)) } }

    suspend fun cancel(id: String): AppResult<FollowUp> =
        runCatchingAppError { ApiCaller.call(json) { api.cancelFollowUp(id) } }

    /** Follow-ups for the current user; [view] maps to the backend scope (today/overdue/upcoming/completed/all). */
    suspend fun myFollowUps(view: String?): AppResult<FollowUpsResponse> =
        runCatchingAppError { ApiCaller.call(json) { api.followUps(view = view, limit = 200) } }

    /** Follow-ups belonging to a single lead (backend returns a bare array). */
    suspend fun forLead(leadId: String): AppResult<List<FollowUp>> =
        runCatchingAppError { ApiCaller.call(json) { api.leadFollowUps(leadId) } }
}
