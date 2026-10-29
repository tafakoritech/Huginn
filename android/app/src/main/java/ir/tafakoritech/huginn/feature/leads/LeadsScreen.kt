package ir.tafakoritech.huginn.feature.leads

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.FilterList
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.design.EmptyState
import ir.tafakoritech.huginn.core.design.ErrorState
import ir.tafakoritech.huginn.core.design.HuginnDimens
import ir.tafakoritech.huginn.core.design.HuginnTopBar
import ir.tafakoritech.huginn.core.design.OfflineBanner
import ir.tafakoritech.huginn.core.design.statusLabel
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.data.model.LeadStatus
import androidx.compose.material3.pulltorefresh.PullToRefreshBox

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LeadsScreen(
    vm: LeadsViewModel,
    onOpenLead: (String) -> Unit,
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val colors = huginnColors()
    var showFilters by rememberSaveable { mutableStateOf(false) }
    val listState = rememberLazyListState()

    Column(modifier = Modifier.fillMaxSize()) {
        HuginnTopBar(title = stringResource(R.string.nav_leads)) {
            IconButton(onClick = { showFilters = true }) {
                Icon(
                    imageVector = Icons.Default.FilterList,
                    contentDescription = stringResource(R.string.filter),
                    tint = if (state.filters != LeadsFiltersUi()) colors.primary else colors.onSurface,
                )
            }
        }

        OutlinedTextField(
            value = state.query,
            onValueChange = vm::onQueryChange,
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = HuginnDimens.space16)
                .padding(bottom = HuginnDimens.space8),
            placeholder = { Text(stringResource(R.string.search_leads)) },
            singleLine = true,
            shape = androidx.compose.foundation.shape.RoundedCornerShape(10.dp),
        )

        PullToRefreshBox(
            isRefreshing = state.loading && state.leads.isNotEmpty(),
            onRefresh = vm::refresh,
            modifier = Modifier.fillMaxSize(),
        ) {
            LeadsListContent(
                state = state,
                listState = listState,
                onOpenLead = onOpenLead,
                onLoadMore = vm::loadMore,
            )
        }
    }

    if (showFilters) {
        FilterSheet(
            current = state.filters,
            onApply = {
                vm.applyFilters(it)
                showFilters = false
            },
            onReset = {
                vm.resetFilters()
                showFilters = false
            },
            onDismiss = { showFilters = false },
        )
    }
}

@Composable
private fun LeadsListContent(
    state: LeadsUiState,
    listState: LazyListState,
    onOpenLead: (String) -> Unit,
    onLoadMore: () -> Unit,
) {
    when {
        state.offline && state.leads.isEmpty() -> {
            Column(
                modifier = Modifier.fillMaxSize(),
                verticalArrangement = Arrangement.spacedBy(HuginnDimens.space12),
            ) {
                OfflineBanner()
                ErrorState(
                    message = stringResource(R.string.error_network),
                    onRetry = onLoadMore,
                )
            }
        }

        state.leads.isEmpty() && !state.loading -> {
            EmptyState(
                title = stringResource(R.string.no_leads),
                hint = stringResource(R.string.no_leads_hint),
            )
        }

        else -> {
            LeadList(
                state = state,
                listState = listState,
                onOpenLead = onOpenLead,
                onLoadMore = onLoadMore,
            )
        }
    }
}

@Composable
private fun LeadList(
    state: LeadsUiState,
    listState: LazyListState,
    onOpenLead: (String) -> Unit,
    onLoadMore: () -> Unit,
) {
    LaunchedEffect(listState.layoutInfo.totalItemsCount) {
        val layoutInfo = listState.layoutInfo
        if (layoutInfo.totalItemsCount > 0 &&
            (layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: 0) >= layoutInfo.totalItemsCount - 3
        ) {
            onLoadMore()
        }
    }
    LazyColumn(
        state = listState,
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(HuginnDimens.space16),
        verticalArrangement = Arrangement.spacedBy(HuginnDimens.space12),
    ) {
        items(
            items = state.leads,
            key = { it.id },
        ) { lead ->
            LeadCard(
                lead = lead,
                onClick = { onOpenLead(lead.id) },
            )
        }
        item {
            if (state.loadingMore) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.Center,
                ) {
                    CircularProgressIndicator(modifier = Modifier.height(28.dp), strokeWidth = 2.dp)
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun FilterSheet(
    current: LeadsFiltersUi,
    onApply: (LeadsFiltersUi) -> Unit,
    onReset: () -> Unit,
    onDismiss: () -> Unit,
) {
    val colors = huginnColors()
    var scope by remember { mutableStateOf(current.scope) }
    var status by remember { mutableStateOf(current.status) }
    var minScore by remember { mutableStateOf(current.minScore) }
    var sort by remember { mutableStateOf(current.sort) }

    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = HuginnDimens.space16)
                .padding(bottom = HuginnDimens.space24)
                .imePadding(),
            verticalArrangement = Arrangement.spacedBy(HuginnDimens.space10),
        ) {
            Text(
                text = stringResource(R.string.filter),
                style = MaterialTheme.typography.titleMedium,
                color = colors.onSurface,
            )

            FilterGroup(title = stringResource(R.string.owner)) {
                listOf(
                    null to stringResource(R.string.all),
                    "mine" to stringResource(R.string.mine),
                    "team" to stringResource(R.string.team),
                ).forEach { (value, label) ->
                    RadioRow(label = label, selected = scope == value) { scope = value }
                }
            }

            FilterGroup(title = stringResource(R.string.status)) {
                val options = listOf<String?>(null) + listOf(
                    LeadStatus.ASSIGNED, LeadStatus.CONTACTING, LeadStatus.FOLLOW_UP,
                    LeadStatus.QUALIFIED, LeadStatus.DISQUALIFIED,
                )
                Row(
                    modifier = Modifier.horizontalScroll(androidx.compose.foundation.rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    options.forEach { value ->
                        FilterChip(
                            selected = status == value,
                            onClick = { status = value },
                            label = { Text(if (value == null) stringResource(R.string.all) else statusLabel(value)) },
                            colors = FilterChipDefaults.filterChipColors(
                                selectedContainerColor = colors.activeCard,
                                selectedLabelColor = colors.onSurface,
                            ),
                        )
                    }
                }
            }

            FilterGroup(title = stringResource(R.string.score)) {
                listOf(
                    null to stringResource(R.string.all),
                    70 to "70+",
                    40 to "40+",
                ).forEach { (value, label) ->
                    RadioRow(label = label, selected = minScore == value) { minScore = value }
                }
            }

            FilterGroup(title = stringResource(R.string.sort)) {
                listOf(
                    null to stringResource(R.string.status),
                    "score" to stringResource(R.string.sort_by_score),
                    "created_at" to stringResource(R.string.sort_by_date),
                ).forEach { (value, label) ->
                    RadioRow(label = label, selected = sort == value) { sort = value }
                }
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                TextButton(onClick = onReset, modifier = Modifier.weight(1f)) {
                    Text(stringResource(R.string.filter_reset), color = colors.mutedText)
                }
                androidx.compose.material3.Button(
                    onClick = {
                        onApply(
                            LeadsFiltersUi(
                                scope = scope,
                                status = status,
                                minScore = minScore,
                                sort = sort,
                                order = if (sort != null) "desc" else null,
                            ),
                        )
                    },
                    modifier = Modifier.weight(2f),
                ) {
                    Text(stringResource(R.string.action_confirm))
                }
            }
        }
    }
}

@Composable
private fun FilterGroup(title: String, content: @Composable () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Text(
            text = title,
            style = MaterialTheme.typography.labelMedium,
            color = huginnColors().secondaryText,
        )
        content()
    }
}

@Composable
private fun RadioRow(label: String, selected: Boolean, onSelect: () -> Unit) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.fillMaxWidth(),
    ) {
        RadioButton(selected = selected, onClick = onSelect)
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = huginnColors().onSurface,
        )
    }
}
