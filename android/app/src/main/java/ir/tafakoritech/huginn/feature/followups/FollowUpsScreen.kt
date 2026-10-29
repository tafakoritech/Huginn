package ir.tafakoritech.huginn.feature.followups

import android.app.DatePickerDialog
import android.app.TimePickerDialog
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.Redo
import androidx.compose.material.icons.outlined.Schedule
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
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
import ir.tafakoritech.huginn.data.model.FollowUp
import ir.tafakoritech.huginn.data.model.FollowUpStatus
import java.time.LocalDateTime

@Composable
fun FollowUpsScreen(
    vm: FollowUpsViewModel,
    onOpenLead: (String) -> Unit,
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val colors = huginnColors()

    Column(modifier = Modifier.fillMaxSize()) {
        HuginnTopBar(title = stringResource(R.string.nav_followups))
        TabRow(
            selectedTabIndex = state.activeTab.ordinal,
            containerColor = colors.surface,
            contentColor = colors.primary,
        ) {
            FollowUpTab.entries.forEachIndexed { index, tab ->
                Tab(
                    selected = index == state.activeTab.ordinal,
                    onClick = { vm.selectTab(tab) },
                    text = { Text(stringResource(tabLabelRes(tab))) },
                )
            }
        }
        FollowUpsContent(state = state, onOpenLead = onOpenLead, vm = vm)
    }
}

@Composable
private fun FollowUpsContent(
    state: FollowUpsUiState,
    onOpenLead: (String) -> Unit,
    vm: FollowUpsViewModel,
) {
    when (state) {
        is FollowUpsUiState.Loading -> {
            Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = huginnColors().primary)
            }
        }
        is FollowUpsUiState.Error -> {
            ErrorState(message = state.message ?: stringResource(R.string.error_unknown), onRetry = vm::refresh)
        }
        is FollowUpsUiState.Empty -> {
            EmptyState(title = stringResource(R.string.empty))
        }
        is FollowUpsUiState.Loaded -> {
            PullToRefreshBox(
                isRefreshing = false,
                onRefresh = vm::refresh,
                modifier = Modifier.fillMaxSize(),
            ) {
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(HuginnDimens.space16),
                    verticalArrangement = Arrangement.spacedBy(HuginnDimens.space12),
                ) {
                    items(state.items, key = { it.id }) { followUp ->
                        FollowUpCard(
                            followUp = followUp,
                            onComplete = { vm.complete(followUp.id) },
                            onReopen = { vm.reopen(followUp.id) },
                            onReschedule = { vm.reschedule(followUp.id, it) },
                            onCancel = { vm.cancel(followUp.id) },
                            onOpenLead = { onOpenLead(followUp.leadId) },
                        )
                    }
                }
            }
        }
    }
}

private fun tabLabelRes(tab: FollowUpTab): Int = when (tab) {
    FollowUpTab.TODAY -> R.string.tab_today
    FollowUpTab.OVERDUE -> R.string.tab_overdue
    FollowUpTab.UPCOMING -> R.string.tab_future
    FollowUpTab.COMPLETED -> R.string.tab_completed
}

@Composable
fun FollowUpCard(
    followUp: FollowUp,
    onComplete: () -> Unit,
    onReopen: () -> Unit,
    onReschedule: (String) -> Unit,
    onCancel: () -> Unit,
    onOpenLead: () -> Unit,
) {
    val colors = huginnColors()
    var showReschedule by remember { mutableStateOf(false) }
    val completed = followUp.status == FollowUpStatus.COMPLETED

    TPCard(modifier = Modifier.clickable(onClick = onOpenLead)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(
                    text = followUp.displayName,
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.onSurface,
                    fontWeight = FontWeight.SemiBold,
                )
                Spacer(Modifier.height(2.dp))
                val detail = listOfNotNull(
                    followUp.title?.takeIf { it.isNotBlank() },
                    followUp.assigneeName,
                ).joinToString(" • ")
                if (detail.isNotBlank()) {
                    Text(
                        text = detail,
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.secondaryText,
                    )
                }
            }
            if (!completed) {
                Text(
                    text = DateTimeUtil.toLocalDigits(DateTimeUtil.formatDateTime(followUp.dueAt) ?: ""),
                    style = MaterialTheme.typography.labelSmall,
                    color = if (followUp.isOverdue) colors.error else colors.warning,
                    modifier = Modifier.padding(start = 6.dp),
                )
            }
        }
        followUp.description?.takeIf { it.isNotBlank() }?.let {
            Spacer(Modifier.height(4.dp))
            Text(
                text = it,
                style = MaterialTheme.typography.bodySmall,
                color = colors.mutedText,
            )
        }
        Spacer(Modifier.height(8.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(4.dp), verticalAlignment = Alignment.CenterVertically) {
            if (completed) {
                IconButton(onClick = onReopen) {
                    Icon(Icons.Outlined.Redo, contentDescription = stringResource(R.string.action_reopen), tint = colors.primary)
                }
            } else {
                TextButton(onClick = onComplete) {
                    Text(stringResource(R.string.action_done), color = colors.success)
                }
                IconButton(onClick = { showReschedule = true }) {
                    Icon(Icons.Outlined.Schedule, contentDescription = stringResource(R.string.action_reschedule), tint = colors.onSurfaceVariant)
                }
                IconButton(onClick = onCancel) {
                    Icon(Icons.Outlined.Delete, contentDescription = stringResource(R.string.action_cancel), tint = colors.mutedText)
                }
            }
        }
    }

    if (showReschedule) {
        FollowUpDateTimeDialog(
            onPick = { iso ->
                showReschedule = false
                onReschedule(iso)
            },
            onDismiss = { showReschedule = false },
        )
    }
}

/** Chains a date picker then a time picker; returns a UTC ISO timestamp (Tehran wall clock input). */
@Composable
fun FollowUpDateTimeDialog(
    onPick: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    val context = LocalContext.current
    var pickedDate by remember { mutableStateOf<LocalDateTime?>(null) }

    LaunchedEffect(Unit) {
        val now = DateTimeUtil.nowInTehran()
        var picked = false
        DatePickerDialog(
            context,
            { _, y, m, d ->
                picked = true
                pickedDate = LocalDateTime.of(y, m + 1, d, now.hour, now.minute)
            },
            now.year,
            now.monthValue - 1,
            now.dayOfMonth,
        ).apply {
            setOnDismissListener { if (!picked) onDismiss() }
            show()
        }
    }

    pickedDate?.let { date ->
        LaunchedEffect(date) {
            var picked = false
            TimePickerDialog(
                context,
                { _, h, m ->
                    picked = true
                    onPick(DateTimeUtil.toIso(date.withHour(h).withMinute(m)))
                },
                date.hour,
                date.minute,
                true,
            ).apply {
                setOnDismissListener { if (!picked) onDismiss() }
                show()
            }
        }
    }
}
