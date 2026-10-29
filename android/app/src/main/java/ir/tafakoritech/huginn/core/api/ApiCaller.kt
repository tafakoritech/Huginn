package ir.tafakoritech.huginn.core.api

import ir.tafakoritech.huginn.core.network.ApiException
import ir.tafakoritech.huginn.core.network.NetworkException
import ir.tafakoritech.huginn.core.util.SafeLog
import ir.tafakoritech.huginn.data.model.ErrorEnvelope
import kotlinx.serialization.json.Json
import okhttp3.ResponseBody
import retrofit2.HttpException
import java.io.IOException

/**
 * Converts transport errors into [ApiException] / [NetworkException] with the
 * backend's machine-readable code. HTTP 2xx body parsing errors are treated as
 * server errors and never surfaced as business errors.
 */
object ApiCaller {

    suspend fun <T> call(json: Json, block: suspend () -> T): T {
        return try {
            block()
        } catch (e: HttpException) {
            val apiException = parseHttpError(json, e)
            SafeLog.d("ApiCaller: HTTP ${e.code()} -> ${apiException.code}")
            throw apiException
        } catch (e: ApiException) {
            throw e
        } catch (e: IOException) {
            SafeLog.d("ApiCaller: network failure")
            throw NetworkException(e)
        } catch (e: Exception) {
            // Response deserialization failure or any unexpected runtime error.
            throw ApiException(-1, null, e.message, e)
        }
    }

    private fun parseHttpError(json: Json, e: HttpException): ApiException {
        val code = e.code()
        var body: String? = null
        try {
            e.response()?.errorBody()?.let { body = readBody(it) }
        } catch (_: Throwable) {
            // errorBody may be consumed already
        }
        val (machineCode, message) = parseErrorEnvelope(json, body)
        return ApiException(code, machineCode, message, e)
    }

    private fun readBody(body: ResponseBody): String = body.string()

    private fun parseErrorEnvelope(json: Json, body: String?): Pair<String?, String?> {
        if (body.isNullOrBlank()) return null to null
        return try {
            val envelope = json.decodeFromString(ErrorEnvelope.serializer(), body)
            envelope.error.code to envelope.error.message
        } catch (_: Throwable) {
            null to body.take(200)
        }
    }
}
