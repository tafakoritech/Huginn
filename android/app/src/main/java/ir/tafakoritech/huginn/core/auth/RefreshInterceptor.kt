package ir.tafakoritech.huginn.core.auth

import kotlinx.coroutines.runBlocking
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Protocol
import okhttp3.Response
import okhttp3.ResponseBody.Companion.toResponseBody
import javax.inject.Inject

/**
 * Application interceptor that transparently refreshes the access token on 401.
 *
 * Behaviour:
 *  - auth/login and auth/refresh requests are never retried here (avoids loops).
 *  - a single refresh is shared by concurrent 401s (single-flight in SessionManager).
 *  - if refresh succeeds the original request is replayed once with the new token.
 *  - if refresh fails the session is cleared (SessionManager) and a synthetic
 *    401 response is returned so Retrofit surfaces it as a typed HTTP error.
 *
 * IMPORTANT: interceptors must never throw non-IOException exceptions. OkHttp's
 * async dispatcher re-throws them and the process crashes. A failed refresh is
 * therefore delivered as a normal 401 response, not an exception.
 */
class RefreshInterceptor @Inject constructor(
    private val sessionManager: SessionManager,
) : Interceptor {

    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()
        val response = chain.proceed(request)
        if (response.code != 401) return response
        if (sessionManager.accessToken() == null) return response

        val path = request.url.encodedPath
        if (path.endsWith("/auth/login") || path.endsWith("/auth/refresh")) {
            return response
        }

        response.close()

        val refreshed = runBlocking { sessionManager.refreshTokens() }
        if (!refreshed) {
            // Session cleared by SessionManager; the UI observes Anonymous and
            // returns to login. Hand the caller a real 401.
            return unauthorizedResponse(request)
        }

        val newToken = sessionManager.accessToken()
        if (newToken == null) {
            return unauthorizedResponse(request)
        }

        val retry = request.newBuilder()
            .header("Authorization", "Bearer $newToken")
            .build()
        return chain.proceed(retry)
    }

    private fun unauthorizedResponse(request: okhttp3.Request): Response =
        Response.Builder()
            .request(request)
            .protocol(Protocol.HTTP_1_1)
            .code(401)
            .message("Unauthorized")
            .body("""{"error":{"code":"UNAUTHORIZED","message":"session expired"}}""".toResponseBody(JSON_MEDIA_TYPE))
            .build()

    private companion object {
        val JSON_MEDIA_TYPE = "application/json".toMediaType()
    }
}
