package ir.tafakoritech.huginn.core.util

import android.util.Log
import ir.tafakoritech.huginn.BuildConfig
import ir.tafakoritech.huginn.core.network.ApiException
import ir.tafakoritech.huginn.core.network.toAppError

/**
 * Structured debug logging.
 *
 * Sensitive values (passwords, tokens, keys, private message bodies) must never
 * be passed to these helpers. The [redact] helper exists for defense in depth:
 * values wrapped with it are replaced by "<redacted>" unless explicitly allowed.
 */
object SafeLog {
    private const val TAG = "Huginn"

    private val alwaysRedact = BuildConfig.DEBUG.not()

    fun d(message: String, tag: String = TAG) {
        if (BuildConfig.DEBUG) Log.d(tag, message)
    }

    fun i(message: String, tag: String = TAG) {
        if (BuildConfig.DEBUG) Log.i(tag, message)
    }

    fun w(message: String, throwable: Throwable? = null, tag: String = TAG) {
        Log.w(tag, message, throwable)
    }

    fun e(message: String, throwable: Throwable? = null, tag: String = TAG) {
        Log.e(tag, message, throwable)
    }

    /** Wrap a sensitive value; the value is never logged in release builds. */
    fun redact(value: String): String = if (alwaysRedact) "<redacted>" else value

    fun redactToken(token: String?): String = if (token == null) "<null>" else redact("<token:${token.length}chars>")
}

/**
 * Generic result wrapper used across repositories.
 * Kept separate from kotlin.Result to allow carrying a rich error type.
 */
sealed class AppResult<out T> {
    data class Success<T>(val data: T) : AppResult<T>()
    data class Failure(val error: AppError) : AppResult<Nothing>()

    val isSuccess: Boolean get() = this is Success
}

/** High level error classification shared by the UI. */
sealed class AppError {
    data object Network : AppError()
    data object Unauthorized : AppError()
    data object Forbidden : AppError()
    data object NotFound : AppError()
    data object Conflict : AppError()
    data object Validation : AppError()
    data object RateLimited : AppError()
    data object MandatoryReportPending : AppError()
    data object InvalidState : AppError()
    data object AlreadyQualified : AppError()
    data object AlreadyExported : AppError()
    data object Unknown : AppError()
    data class Server(val code: String, val message: String) : AppError()
}

/** Thin suspend wrapper around a throwing call. */
suspend fun <T> runCatchingAppError(block: suspend () -> T): AppResult<T> {
    return try {
        AppResult.Success(block())
    } catch (e: ApiException) {
        AppResult.Failure(e.toAppError())
    } catch (e: Throwable) {
        AppResult.Failure(AppError.Network)
    }
}

/** Extracts the data from an [AppResult.Success], or null on failure. */
fun <T> AppResult<T>.getOrNull(): T? = (this as? AppResult.Success)?.data
