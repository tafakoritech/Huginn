package ir.tafakoritech.huginn.core.design

import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.util.AppError

/** Maps machine errors to clear Persian messages (never parses human strings). */
@Composable
fun errorMessage(error: AppError): String = when (error) {
    AppError.Network -> stringResource(R.string.error_network)
    AppError.Unauthorized -> stringResource(R.string.session_expired)
    AppError.Forbidden -> stringResource(R.string.error_forbidden)
    AppError.NotFound -> stringResource(R.string.error_not_found)
    AppError.Conflict -> stringResource(R.string.error_conflict)
    AppError.Validation -> stringResource(R.string.error_validation)
    AppError.RateLimited -> stringResource(R.string.error_rate_limited)
    AppError.MandatoryReportPending -> stringResource(R.string.error_mandatory_report)
    AppError.InvalidState -> stringResource(R.string.error_invalid_state)
    AppError.AlreadyQualified -> stringResource(R.string.error_already_qualified)
    AppError.AlreadyExported -> stringResource(R.string.error_already_exported)
    AppError.Unknown -> stringResource(R.string.error_unknown)
    is AppError.Server -> stringResource(R.string.error_unknown)
}
