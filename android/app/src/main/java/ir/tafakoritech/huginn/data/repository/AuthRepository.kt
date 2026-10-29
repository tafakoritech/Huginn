package ir.tafakoritech.huginn.data.repository

import ir.tafakoritech.huginn.core.api.ApiCaller
import ir.tafakoritech.huginn.core.api.HuginnApi
import ir.tafakoritech.huginn.core.auth.SessionManager
import ir.tafakoritech.huginn.core.auth.SessionState
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.runCatchingAppError
import ir.tafakoritech.huginn.data.model.LoginRequest
import kotlinx.coroutines.flow.StateFlow
import kotlinx.serialization.json.Json
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Authentication repository: login, logout, session restoration and the
 * authenticated-user validation used by the splash flow.
 */
@Singleton
class AuthRepository @Inject constructor(
    private val api: HuginnApi,
    private val sessionManager: SessionManager,
    private val json: Json,
) {
    val sessionState: StateFlow<SessionState> = sessionManager.state

    suspend fun login(username: String, password: String): AppResult<Unit> = runCatchingAppError {
        val response = ApiCaller.call(json) { api.login(LoginRequest(login = username.trim(), password = password)) }
        sessionManager.onTokensRefreshed(response.accessToken, response.refreshToken)
        val user = response.user
        if (user != null) {
            sessionManager.onAuthenticated(user)
        } else {
            sessionManager.restoreFromStorage()
        }
    }

    suspend fun logout() {
        // Best-effort server-side revocation of the refresh token (the backend
        // reads it from the X-Refresh-Token header), then local cleanup.
        runCatchingAppError { ApiCaller.call(json) { api.logout(sessionManager.refreshToken().orEmpty()) } }
        sessionManager.logout()
    }

    /**
     * Validate a persisted session against the backend (GET /me). On 401 the
     * refresh interceptor transparently refreshes; if refresh also fails the
     * session is cleared and Anonymous is emitted.
     */
    suspend fun restoreSession(): AppResult<Unit> = runCatchingAppError {
        sessionManager.restoreFromStorage()
        if (sessionManager.state.value is SessionState.Authenticated) {
            val me = ApiCaller.call(json) { api.me() }
            sessionManager.onAuthenticated(me)
        }
    }

    fun clearLocalSession() = sessionManager.clearSession()
}
