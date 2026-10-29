package ir.tafakoritech.huginn.feature.workqueue

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.FactCheck
import androidx.compose.material.icons.filled.Message
import androidx.compose.material.icons.filled.Report
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ScrollableTabRow
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.design.EmptyState
import ir.tafakoritech.huginn.core.design.ErrorState
import ir.tafakoritech.huginn.core.design.HuginnDimens
import ir.tafakoritech.huginn.core.design.HuginnTopBar
import ir.tafakoritech.huginn.core.design.TPCard
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.core.util.DateTimeUtil
import ir.tafakoritech.huginn.data.model.NextActionItem
import ir.tafakoritech.huginn.data.model.NextActionPriority
import ir.tafakoritech.huginn.data.model.NextActionType
import ir.tafakoritech.huginn.feature.followups.FollowUpDateTimeDialog

@Composable
fun WorkQueueScreen(
    vm: WorkQueueViewModel,
    onBack: () -> Unit,
    onOpenLead: (String) -> Unit,
    onOpenReport: (String) -> Unit,
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val tab by vm.tab.collectAsStateWithLifecycle()

    Column(modifier = Modifier.fillMaxSize()) {
        HuginnTopBar(title = stringResource(R.string.work_queue_title), onBack = onBack)
        WorkQueueTabRow(selected = tab, onSelect = vm::selectTab)
        WorkQueueContent(
            state = state,
            tab = tab,
            onOpenLead = onOpenLead,
            onOpenReport = onOpenReport,
            onSnooze = vm::snooze,
            onComplete = vm::complete,
        )
    }
}

/**
 * Horizontally scrollable filter row. Labels are always single-line (ellipsized
 * as a last resort), content-sized so long Persian labels keep their natural
 * width, and the whole row scrolls instead of shrinking or wrapping. RTL
 * ordering/indicator are handled by the standard Compose layout direction.
 */
@Composable
fun WorkQueueTabRow(
    selected: WorkQueueTab,
    onSelect: (WorkQueueTab) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = huginnColors()
    ScrollableTabRow(
        selectedTabIndex = WorkQueueTab.entries.indexOf(selected),
        containerColor = colors.surface,
        contentColor = colors.primary,
        edgePadding = HuginnDimens.space16,
        modifier = modifier,
    ) {
        WorkQueueTab.entries.forEach { t ->
            Tab(
                selected = t == selected,
                onClick = { onSelect(t) },
                text = {
                    Text(
                        text = stringResource(tabLabelRes(t)),
                        maxLines = 1,
                        softWrap = false,
                        overflow = TextOverflow.Ellipsis,
                    )
                },
            )
        }
    }
}

private fun tabLabelRes(tab: WorkQueueTab): Int = when (tab) {
    WorkQueueTab.ALL -> R.string.tab_all_work
    WorkQueueTab.URGENT -> R.string.tab_urgent_work
    WorkQueueTab.TODAY -> R.string.tab_today_work
    WorkQueueTab.OVERDUE -> R.string.tab_overdue_work
    WorkQueueTab.REPORTS -> R.string.tab_reports_work
    WorkQueueTab.CALL -> R.string.tab_call_work
    WorkQueueTab.MESSAGE -> R.string.tab_message_work
    WorkQueueTab.EMAIL -> R.string.tab_email_work
}

@Composable
private fun WorkQueueContent(
    state: WorkQueueUiState,
    tab: WorkQueueTab,
    onOpenLead: (String) -> Unit,
    onOpenReport: (String) -> Unit,
    onSnooze: (NextActionItem, String) -> Unit,
    onComplete: (NextActionItem) -> Unit,
) {
    if (state.loading && state.items.isEmpty()) {
        Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            CircularProgressIndicator(color = huginnColors().primary)
        }
        return
    }
    if (state.hasError && state.items.isEmpty()) {
        ErrorState(message = stringResource(R.string.error_network))
        return
    }
    val items = filterForTab(state.items, tab)
    if (items.isEmpty()) {
        EmptyState(title = stringResource(R.string.work_queue_empty))
        return
    }
    PullToRefreshBox(isRefreshing = false, onRefresh = {}, modifier = Modifier.fillMaxSize()) {
        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(HuginnDimens.space16),
            verticalArrangement = Arrangement.spacedBy(HuginnDimens.space12),
        ) {
            items(items, key = { it.id }) { action ->
                WorkQueueCard(
                    action = action,
                    onOpenLead = onOpenLead,
                    onOpenReport = onOpenReport,
                    onSnooze = onSnooze,
                    onComplete = onComplete,
                )
            }
        }
    }
}

@Composable
private fun WorkQueueCard(
    action: NextActionItem,
    onOpenLead: (String) -> Unit,
    onOpenReport: (String) -> Unit,
    onSnooze: (NextActionItem, String) -> Unit,
    onComplete: (NextActionItem) -> Unit,
) {
    val colors = huginnColors()
    var showSnooze by remember { mutableStateOf(false) }
    var showCustomPicker by remember { mutableStateOf(false) }

    TPCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(
                imageVector = actionIcon(action.actionType),
                contentDescription = null,
                tint = when (action.priority) {
                    NextActionPriority.URGENT -> colors.error
                    NextActionPriority.IMPORTANT -> colors.warning
                    else -> colors.mutedText
                },
                modifier = Modifier.size(20.dp),
            )
            Spacer(Modifier.size(HuginnDimens.space8))
            Text(
                text = action.title.ifBlank { action.companyName },
                style = MaterialTheme.typography.titleMedium,
                color = colors.onSurface,
                fontWeight = FontWeight.SemiBold,
                modifier = Modifier.weight(1f),
            )
        }
        action.context?.takeIf { it.isNotBlank() }?.let {
            Spacer(Modifier.height(4.dp))
            Text(it, style = MaterialTheme.typography.bodySmall, color = colors.secondaryText)
        }
        Spacer(Modifier.height(4.dp))
        Text(
            text = action.explainability.joinToString(" • "),
            style = MaterialTheme.typography.bodySmall,
            color = colors.mutedText,
        )
        Spacer(Modifier.height(HuginnDimens.space8))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            TextButton(onClick = {
                if (action.actionType == NextActionType.REPORT) onOpenReport(action.sourceId) else onOpenLead(action.leadId)
            }) {
                Text(action.ctaLabel.ifBlank { stringResource(R.string.action_view) }, color = colors.primary)
            }
            if (action.sourceType == ir.tafakoritech.huginn.data.model.NextActionSource.FOLLOW_UP) {
                TextButton(onClick = { showSnooze = true }) {
                    Text(stringResource(R.string.action_snooze), color = colors.secondaryText)
                }
                TextButton(onClick = { onComplete(action) }) {
                    Text(stringResource(R.string.action_complete), color = colors.success)
                }
            }
        }
    }

    if (showSnooze) {
        SnoozeDialog(
            onQuick = { iso -> showSnooze = false; onSnooze(action, iso) },
            onCustom = { showSnooze = false; showCustomPicker = true },
            onDismiss = { showSnooze = false },
        )
    }
    if (showCustomPicker) {
        FollowUpDateTimeDialog(
            onPick = { iso -> showCustomPicker = false; onSnooze(action, iso) },
            onDismiss = { showCustomPicker = false },
        )
    }
}

@Composable
private fun SnoozeDialog(
    onQuick: (String) -> Unit,
    onCustom: () -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.snooze_title)) },
        text = {
            Column {
                SnoozeOption(stringResource(R.string.snooze_1h)) { onQuick(DateTimeUtil.toIso(DateTimeUtil.nowInTehran().plusHours(1))) }
                SnoozeOption(stringResource(R.string.snooze_evening)) { onQuick(DateTimeUtil.toIso(DateTimeUtil.todayInTehran().atTime(18, 0))) }
                SnoozeOption(stringResource(R.string.snooze_tomorrow)) { onQuick(DateTimeUtil.toIso(DateTimeUtil.todayInTehran().plusDays(1).atTime(9, 0))) }
                SnoozeOption(stringResource(R.string.snooze_pick_date), onCustom)
            }
        },
        confirmButton = {},
        dismissButton = {
            TextButton(onClick = onDismiss) { Text(stringResource(R.string.action_cancel)) }
        },
    )
}

@Composable
private fun SnoozeOption(label: String, onClick: () -> Unit) {
    TextButton(onClick = onClick, modifier = Modifier.fillMaxWidth()) {
        Text(label, modifier = Modifier.fillMaxWidth())
    }
}

private fun actionIcon(actionType: String): ImageVector = when (actionType) {
    NextActionType.CALL -> Icons.Filled.Call
    NextActionType.MESSAGE -> Icons.Filled.Message
    NextActionType.EMAIL -> Icons.Filled.Email
    NextActionType.REPORT -> Icons.Filled.Report
    NextActionType.QUALIFICATION -> Icons.Filled.FactCheck
    NextActionType.LEAD_REVIEW -> Icons.Filled.Visibility
    else -> Icons.Filled.Schedule
}
