package ir.tafakoritech.huginn.feature.home

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.FactCheck
import androidx.compose.material.icons.filled.Message
import androidx.compose.material.icons.filled.Report
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.design.HuginnDimens
import ir.tafakoritech.huginn.core.design.LoadingSkeleton
import ir.tafakoritech.huginn.core.design.OfflineBanner
import ir.tafakoritech.huginn.core.design.PrimaryButton
import ir.tafakoritech.huginn.core.design.SecondaryButton
import ir.tafakoritech.huginn.core.design.TPCard
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.core.util.DateTimeUtil
import ir.tafakoritech.huginn.data.model.NextActionItem
import ir.tafakoritech.huginn.data.model.NextActionPriority
import ir.tafakoritech.huginn.data.model.NextActionSource
import ir.tafakoritech.huginn.data.model.NextActionType
import ir.tafakoritech.huginn.feature.followups.FollowUpDateTimeDialog

// --- Pure mapping helpers (unit-tested) -------------------------------------

/** Whether an action can be snoozed (only follow-up-backed work). */
internal fun actionSnoozable(sourceType: String): Boolean =
    sourceType == NextActionSource.FOLLOW_UP

/** Whether an action can be completed inline (follow-up-backed work). */
internal fun actionCompletable(sourceType: String): Boolean = actionSnoozable(sourceType)

/** Whether tapping the primary CTA should open the mandatory report. */
internal fun opensReport(actionType: String, sourceType: String): Boolean =
    actionType == NextActionType.REPORT || sourceType == NextActionSource.PENDING_REPORT

@Composable
fun HomeScreen(
    vm: HomeViewModel,
    onOpenLead: (String) -> Unit,
    onOpenWorkQueue: () -> Unit,
    onOpenReport: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val state by vm.state.collectAsStateWithLifecycle()

    val lifecycleOwner = androidx.lifecycle.compose.LocalLifecycleOwner.current
    androidx.compose.runtime.DisposableEffect(lifecycleOwner) {
        val observer = androidx.lifecycle.LifecycleEventObserver { _, event ->
            if (event == androidx.lifecycle.Lifecycle.Event.ON_RESUME) {
                vm.refresh()
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    PullToRefreshBox(
        isRefreshing = state.loading,
        onRefresh = vm::refresh,
        modifier = modifier.fillMaxSize(),
    ) {
        HomeContent(
            state = state,
            onOpenLead = onOpenLead,
            onOpenReport = onOpenReport,
            onOpenWorkQueue = onOpenWorkQueue,
            onSnooze = vm::snooze,
            onComplete = vm::complete,
        )
    }
}

@Composable
fun HomeContent(
    state: HomeUiState,
    onOpenLead: (String) -> Unit,
    onOpenReport: (String) -> Unit,
    onOpenWorkQueue: () -> Unit,
    onSnooze: (NextActionItem, String) -> Unit,
    onComplete: (NextActionItem) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = huginnColors()
    if (state.loading && state.nextActions.isEmpty()) {
        LoadingSkeleton(modifier = modifier)
    } else {
        LazyColumn(
            modifier = modifier.fillMaxSize(),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(
                horizontal = HuginnDimens.space16,
                vertical = HuginnDimens.space16,
            ),
            verticalArrangement = Arrangement.spacedBy(HuginnDimens.space16),
        ) {
            item {
                Text(
                    text = stringResource(R.string.home_greeting, state.userName),
                    style = MaterialTheme.typography.headlineSmall,
                    color = colors.onSurface,
                )
            }
            if (state.offline) {
                item { OfflineBanner() }
            }
            item {
                StatsGrid(state.stats)
            }
            item {
                NextActionsSection(
                    state = state,
                    onOpenLead = onOpenLead,
                    onOpenReport = onOpenReport,
                    onOpenWorkQueue = onOpenWorkQueue,
                    onSnooze = onSnooze,
                    onComplete = onComplete,
                )
            }
        }
    }
}

@Composable
private fun StatsGrid(stats: ir.tafakoritech.huginn.data.model.MeStats) {
    val items = listOf(
        stringResource(R.string.stat_today_leads) to stats.todayLeads,
        stringResource(R.string.stat_today_calls) to stats.todayCalls,
        stringResource(R.string.stat_pending_reports) to stats.pendingReports,
        stringResource(R.string.stat_today_followups) to stats.todayFollowups,
        stringResource(R.string.stat_overdue_followups) to stats.overdueFollowups,
        stringResource(R.string.stat_qualified_leads) to stats.qualifiedLeads,
    )
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        items.chunked(2).forEach { row: List<Pair<String, Int>> ->
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                row.forEach { (label, value) ->
                    StatCard(
                        label = label,
                        value = value,
                        modifier = Modifier.weight(1f),
                    )
                }
                if (row.size == 1) {
                    Spacer(Modifier.weight(1f))
                }
            }
        }
    }
}

@Composable
private fun StatCard(label: String, value: Int, modifier: Modifier = Modifier) {
    val colors = huginnColors()
    TPCard(modifier = modifier) {
        Text(
            text = DateTimeUtil.toLocalDigits(value.toString()),
            style = MaterialTheme.typography.titleLarge,
            color = colors.primary,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(2.dp))
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            color = colors.secondaryText,
        )
    }
}

@Composable
private fun NextActionsSection(
    state: HomeUiState,
    onOpenLead: (String) -> Unit,
    onOpenReport: (String) -> Unit,
    onOpenWorkQueue: () -> Unit,
    onSnooze: (NextActionItem, String) -> Unit,
    onComplete: (NextActionItem) -> Unit,
) {
    val colors = huginnColors()
    Text(
        text = stringResource(R.string.next_work),
        style = MaterialTheme.typography.titleMedium,
        color = colors.onSurface,
        fontWeight = FontWeight.SemiBold,
    )
    Spacer(Modifier.height(HuginnDimens.space8))

    if (state.nextActions.isEmpty()) {
        if (state.hasError) {
            TPCard {
                Text(
                    text = stringResource(R.string.error_network),
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.error,
                )
            }
        } else {
            TPCard {
                Text(
                    text = stringResource(R.string.next_work_empty),
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.onSurface,
                )
                Spacer(Modifier.height(4.dp))
                Text(
                    text = stringResource(R.string.next_work_hint),
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.mutedText,
                )
            }
        }
        return
    }

    state.nextActions.forEachIndexed { index, action ->
        NextActionCard(
            action = action,
            onOpenLead = onOpenLead,
            onOpenReport = onOpenReport,
            onSnooze = onSnooze,
            onComplete = onComplete,
        )
        if (index != state.nextActions.lastIndex) {
            Spacer(Modifier.height(HuginnDimens.space8))
        }
    }

    if (state.totalActionable > state.nextActions.size) {
        Spacer(Modifier.height(HuginnDimens.space4))
        TextButton(onClick = onOpenWorkQueue) {
            Text(stringResource(R.string.view_all_work), color = colors.primary)
        }
    }
}

@Composable
private fun NextActionCard(
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
                tint = priorityColor(action.priority),
                modifier = Modifier.size(20.dp),
            )
            Spacer(Modifier.size(HuginnDimens.space8))
            PriorityBadge(action.priority)
            Spacer(Modifier.weight(1f))
            DueStateText(action)
        }
        Spacer(Modifier.height(HuginnDimens.space8))
        Text(
            text = action.title.ifBlank { action.companyName },
            style = MaterialTheme.typography.titleMedium,
            color = colors.onSurface,
            fontWeight = FontWeight.SemiBold,
        )
        val context = action.context?.takeIf { it.isNotBlank() } ?: action.reason
        if (context.isNotBlank()) {
            Spacer(Modifier.height(4.dp))
            Text(
                text = context,
                style = MaterialTheme.typography.bodySmall,
                color = colors.secondaryText,
            )
        }
        if (action.explainability.isNotEmpty()) {
            Spacer(Modifier.height(6.dp))
            Text(
                text = stringResource(R.string.why_now),
                style = MaterialTheme.typography.labelSmall,
                color = colors.mutedText,
                fontWeight = FontWeight.Medium,
            )
            Text(
                text = action.explainability.joinToString(" • "),
                style = MaterialTheme.typography.bodySmall,
                color = colors.secondaryText,
            )
        }
        action.lastInteractionAt?.let { last ->
            Spacer(Modifier.height(4.dp))
            Text(
                text = stringResource(R.string.last_interaction_prefix, DateTimeUtil.formatRelative(last)),
                style = MaterialTheme.typography.bodySmall,
                color = colors.mutedText,
            )
        }
        Spacer(Modifier.height(HuginnDimens.space12))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            PrimaryButton(
                text = action.ctaLabel.ifBlank { stringResource(R.string.action_view) },
                onClick = {
                    if (opensReport(action.actionType, action.sourceType)) {
                        onOpenReport(action.sourceId)
                    } else {
                        onOpenLead(action.leadId)
                    }
                },
                modifier = Modifier.weight(1f),
            )
            if (actionSnoozable(action.sourceType)) {
                SecondaryButton(
                    text = stringResource(R.string.action_snooze),
                    onClick = { showSnooze = true },
                )
            }
        }
    }

    if (showSnooze) {
        SnoozeDialog(
            onQuick = { iso ->
                showSnooze = false
                onSnooze(action, iso)
            },
            onCustom = {
                showSnooze = false
                showCustomPicker = true
            },
            onDismiss = { showSnooze = false },
        )
    }
    if (showCustomPicker) {
        FollowUpDateTimeDialog(
            onPick = { iso ->
                showCustomPicker = false
                onSnooze(action, iso)
            },
            onDismiss = { showCustomPicker = false },
        )
    }
}

@Composable
private fun PriorityBadge(priority: String) {
    val colors = huginnColors()
    val (label, fg, bg) = when (priority) {
        NextActionPriority.URGENT -> Triple(stringResource(R.string.priority_urgent), colors.error, colors.error.copy(alpha = 0.14f))
        NextActionPriority.IMPORTANT -> Triple(stringResource(R.string.priority_important), colors.warning, colors.warning.copy(alpha = 0.14f))
        else -> Triple(stringResource(R.string.priority_normal), colors.mutedText, colors.surfaceContainerHighest)
    }
    androidx.compose.foundation.layout.Box(
        modifier = Modifier
            .background(bg, RoundedCornerShape(HuginnDimens.badgeRadius))
            .padding(horizontal = 8.dp, vertical = 3.dp),
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall,
            color = fg,
            fontWeight = FontWeight.SemiBold,
        )
    }
}

@Composable
private fun DueStateText(action: NextActionItem) {
    val colors = huginnColors()
    val text = dueStateText(action)
    if (text.isBlank()) return
    val color = if (action.overdue) colors.error else colors.warning
    Text(
        text = text,
        style = MaterialTheme.typography.labelSmall,
        color = color,
        fontWeight = FontWeight.Medium,
    )
}

/** Compact due-state label; empty when unknown. */
private fun dueStateText(action: NextActionItem): String {
    val due = action.dueAt ?: return ""
    if (action.overdue) {
        val rel = DateTimeUtil.formatRelative(due)
        return "$rel • عقب‌افتاده"
    }
    if (DateTimeUtil.isToday(due)) {
        val time = DateTimeUtil.formatTime(due) ?: return "امروز"
        val now = DateTimeUtil.nowInTehran()
        val dueTehran = DateTimeUtil.toTehranDateTime(due) ?: return "امروز $time"
        return if (dueTehran.minusHours(2).isBefore(now)) {
            "تا ۲ ساعت دیگر"
        } else {
            "امروز $time"
        }
    }
    return DateTimeUtil.formatDateTime(due) ?: ""
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

@Composable
private fun priorityColor(priority: String) = when (priority) {
    NextActionPriority.URGENT -> huginnColors().error
    NextActionPriority.IMPORTANT -> huginnColors().warning
    else -> huginnColors().mutedText
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
                SnoozeOption(stringResource(R.string.snooze_1h)) {
                    onQuick(DateTimeUtil.toIso(DateTimeUtil.nowInTehran().plusHours(1)))
                }
                SnoozeOption(stringResource(R.string.snooze_evening)) {
                    onQuick(DateTimeUtil.toIso(DateTimeUtil.todayInTehran().atTime(18, 0)))
                }
                SnoozeOption(stringResource(R.string.snooze_tomorrow)) {
                    onQuick(DateTimeUtil.toIso(DateTimeUtil.todayInTehran().plusDays(1).atTime(9, 0)))
                }
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
