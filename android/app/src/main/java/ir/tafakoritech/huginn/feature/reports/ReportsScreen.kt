package ir.tafakoritech.huginn.feature.reports

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.design.EmptyState
import ir.tafakoritech.huginn.core.design.ErrorState
import ir.tafakoritech.huginn.core.design.HuginnDimens
import ir.tafakoritech.huginn.core.design.HuginnTopBar
import ir.tafakoritech.huginn.core.design.LoadingSkeleton
import ir.tafakoritech.huginn.core.design.TPCard
import ir.tafakoritech.huginn.core.design.channelLabel
import ir.tafakoritech.huginn.core.design.interestLabel
import ir.tafakoritech.huginn.core.design.nextActionLabel
import ir.tafakoritech.huginn.core.design.outcomeLabel
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.core.util.DateTimeUtil
import ir.tafakoritech.huginn.data.model.SubmittedReport

/**
 * Global Reports screen. Shows submitted contact reports from the backend
 * (GET /me/reports). Pull to refresh + auto-refresh on every resume so a report
 * submitted moments ago appears immediately — no app restart or manual refresh
 * required.
 */
@Composable
fun ReportsScreen(
    vm: ReportsViewModel,
) {
    val state by vm.state.collectAsStateWithLifecycle()

    // Re-query whenever the tab becomes visible (ON_RESUME of this destination).
    // Navigation saves/restores the ViewModel, so without this the tab would keep
    // stale data after a report is submitted elsewhere.
    val lifecycleOwner = LocalLifecycleOwner.current
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                vm.refresh()
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    Column(modifier = Modifier.fillMaxSize()) {
        HuginnTopBar(title = stringResource(R.string.reports_title))
        PullToRefreshBox(
            isRefreshing = state is ReportsUiState.Loading,
            onRefresh = vm::refresh,
            modifier = Modifier.fillMaxSize(),
        ) {
            ReportsListContent(state = state, onRetry = vm::refresh)
        }
    }
}

@Composable
private fun ReportsListContent(state: ReportsUiState, onRetry: () -> Unit) {
    when (state) {
        is ReportsUiState.Loading -> LoadingSkeleton()
        is ReportsUiState.Error -> ErrorState(
            message = stringResource(R.string.reports_error),
            onRetry = onRetry,
        )
        ReportsUiState.Empty -> EmptyState(
            title = stringResource(R.string.no_reports),
            hint = stringResource(R.string.reports_empty_hint),
        )
        is ReportsUiState.Loaded -> LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(HuginnDimens.space16),
            verticalArrangement = Arrangement.spacedBy(HuginnDimens.space12),
        ) {
            items(state.reports, key = { it.id }) { report ->
                ReportCard(report = report)
            }
        }
    }
}

@Composable
private fun ReportCard(report: SubmittedReport) {
    val colors = huginnColors()
    TPCard {
        Column(Modifier.padding(HuginnDimens.space12), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = report.displayName,
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.onSurface,
                    fontWeight = FontWeight.SemiBold,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    text = channelLabel(report.channel),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.primary,
                )
            }

            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                if (report.outcome.isNotBlank()) {
                    Text(
                        text = stringResource(R.string.report_outcome_label_short) + " " + outcomeLabel(report.outcome),
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.onSurface,
                    )
                }
                if (!report.interestLevel.isNullOrBlank()) {
                    Text(
                        text = stringResource(R.string.report_interest_label_short) + " " + interestLabel(report.interestLevel),
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.onSurface,
                    )
                }
            }

            if (!report.summary.isNullOrBlank()) {
                Text(
                    text = report.summary,
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.mutedText,
                )
            }

            if (!report.mainObjection.isNullOrBlank()) {
                Text(
                    text = stringResource(R.string.report_objection_label) + " " + report.mainObjection,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.mutedText,
                )
            }

            if (!report.nextAction.isNullOrBlank()) {
                Text(
                    text = stringResource(R.string.report_next_action_label) + " " + nextActionLabel(report.nextAction),
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.mutedText,
                )
            }

            report.createdAt.takeIf { it.isNotBlank() }?.let {
                Text(
                    text = DateTimeUtil.formatDate(it) ?: it,
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.mutedText,
                )
            }
        }
    }
}
