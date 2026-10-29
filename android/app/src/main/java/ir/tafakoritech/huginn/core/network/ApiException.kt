package ir.tafakoritech.huginn.core.network

import ir.tafakoritech.huginn.core.util.AppError

/**
 * Machine-readable error codes returned by the Huginn backend
 * (see backend/internal/httpapi/response.go).
 */
object ApiErrorCodes {
    const val BAD_REQUEST = "BAD_REQUEST"
    const val UNAUTHORIZED = "UNAUTHORIZED"
    const val FORBIDDEN = "FORBIDDEN"
    const val NOT_FOUND = "NOT_FOUND"
    const val CONFLICT = "CONFLICT"
    const val VALIDATION_ERROR = "VALIDATION_ERROR"
    const val INTERNAL_ERROR = "INTERNAL_ERROR"
    const val MANDATORY_REPORT_PENDING = "MANDATORY_REPORT_PENDING"
    const val INVALID_STATE_TRANSITION = "INVALID_STATE_TRANSITION"
    const val DUPLICATE_EXPORT = "DUPLICATE_EXPORT"
    const val RATE_LIMITED = "RATE_LIMITED"
    const val METHOD_NOT_ALLOWED = "METHOD_NOT_ALLOWED"
}

/**
 * Raised for any non-2xx API response. Carries the machine code and a human
 * message from the backend (used only for diagnostics, never parsed).
 */
class ApiException(
    val httpStatus: Int,
    val code: String?,
    val backendMessage: String?,
    cause: Throwable? = null,
) : Exception(backendMessage ?: "HTTP $httpStatus", cause)

/** Raised when a request cannot be served and no cached data exists. */
class NetworkException(cause: Throwable? = null) : Exception("network failure", cause)

fun ApiException.toAppError(): AppError = when {
    code == ApiErrorCodes.MANDATORY_REPORT_PENDING -> AppError.MandatoryReportPending
    code == ApiErrorCodes.INVALID_STATE_TRANSITION -> AppError.InvalidState
    code == ApiErrorCodes.CONFLICT || httpStatus == 409 -> AppError.Conflict
    code == ApiErrorCodes.VALIDATION_ERROR || httpStatus == 400 -> AppError.Validation
    code == ApiErrorCodes.RATE_LIMITED || httpStatus == 429 -> AppError.RateLimited
    code == ApiErrorCodes.UNAUTHORIZED || httpStatus == 401 -> AppError.Unauthorized
    code == ApiErrorCodes.FORBIDDEN || httpStatus == 403 -> AppError.Forbidden
    code == ApiErrorCodes.NOT_FOUND || httpStatus == 404 -> AppError.NotFound
    code == ApiErrorCodes.DUPLICATE_EXPORT -> AppError.AlreadyExported
    else -> AppError.Server(code ?: "", backendMessage ?: "")
}
