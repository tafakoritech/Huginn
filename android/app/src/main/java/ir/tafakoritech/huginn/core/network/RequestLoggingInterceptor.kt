package ir.tafakoritech.huginn.core.network

import ir.tafakoritech.huginn.BuildConfig
import ir.tafakoritech.huginn.core.util.SafeLog
import okhttp3.Interceptor
import okhttp3.Response
import java.util.concurrent.TimeUnit
import javax.inject.Inject

/**
 * Debug-only structured request diagnostics.
 *
 * Logs exactly what is needed to confirm requests reach Huginn and how the
 * backend answered:
 *
 *   API_REQUEST method=POST path=/api/v1/auth/login base_url=http://... status=201 latency_ms=...
 *
 * NEVER logs request/response bodies, headers, passwords, access/refresh
 * tokens, API keys or sensitive document contents. The interceptor is only
 * installed in debug builds; release/staging builds log nothing.
 */
class RequestLoggingInterceptor @Inject constructor() : Interceptor {

    override fun intercept(chain: Interceptor.Chain): Response {
        if (!BuildConfig.DEBUG) return chain.proceed(chain.request())
        val startNs = System.nanoTime()
        val request = chain.request()
        val httpUrl = request.url
        val isDefaultPort = (httpUrl.scheme == "https" && httpUrl.port == 443) ||
            (httpUrl.scheme == "http" && httpUrl.port == 80)
        val baseUrl = httpUrl.scheme + "://" + httpUrl.host + if (isDefaultPort) "" else ":${httpUrl.port}"
        val path = httpUrl.encodedPath
        val method = request.method
        var status = -1
        try {
            val response = chain.proceed(request)
            status = response.code
            return response
        } finally {
            val latencyMs = TimeUnit.NANOSECONDS.toMillis(System.nanoTime() - startNs)
            SafeLog.d(
                "API_REQUEST method=$method path=$path base_url=$baseUrl status=$status latency_ms=$latencyMs",
            )
        }
    }
}
