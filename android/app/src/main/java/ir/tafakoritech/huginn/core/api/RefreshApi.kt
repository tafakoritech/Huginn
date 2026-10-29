package ir.tafakoritech.huginn.core.api

import ir.tafakoritech.huginn.data.model.Tokens
import retrofit2.http.Header
import retrofit2.http.POST

/**
 * Minimal refresh endpoint used only by SessionManager. Deliberately separate
 * from [HuginnApi] so the refresh call never runs through the auth interceptors
 * (preventing refresh loops).
 *
 * The backend reads the refresh token from the `X-Refresh-Token` header (or the
 * `refresh_token` query parameter) — it is NOT accepted in the JSON body.
 */
interface RefreshApi {
    @POST("api/v1/auth/refresh")
    suspend fun refresh(@Header("X-Refresh-Token") refreshToken: String): Tokens
}
