package ir.tafakoritech.huginn.feature.qualification

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.design.HuginnDimens
import ir.tafakoritech.huginn.core.design.HuginnTopBar
import ir.tafakoritech.huginn.core.design.PrimaryButton
import ir.tafakoritech.huginn.core.design.TPCard
import ir.tafakoritech.huginn.core.design.VoiceTextField
import ir.tafakoritech.huginn.core.design.disqualifyReasonLabel
import ir.tafakoritech.huginn.core.design.disqualifyReasonsInOrder
import ir.tafakoritech.huginn.core.theme.huginnColors

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun DisqualifyScreen(
    vm: DisqualifyViewModel,
    onDone: () -> Unit,
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val colors = huginnColors()

    Column(modifier = Modifier.fillMaxSize()) {
        HuginnTopBar(title = stringResource(R.string.disqualify_title), onBack = onDone)
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = HuginnDimens.space16, vertical = HuginnDimens.space16),
            verticalArrangement = Arrangement.spacedBy(HuginnDimens.space12),
        ) {
            TPCard {
                Text(
                    text = stringResource(R.string.disqualify_reason),
                    style = MaterialTheme.typography.titleSmall,
                    color = colors.onSurface,
                )
                Spacer(Modifier.height(8.dp))
                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    disqualifyReasonsInOrder.forEach { reason ->
                        FilterChip(
                            selected = state.reason == reason,
                            onClick = { vm.selectReason(reason) },
                            label = { Text(disqualifyReasonLabel(reason)) },
                            colors = FilterChipDefaults.filterChipColors(
                                selectedContainerColor = colors.errorContainer,
                                selectedLabelColor = colors.onErrorContainer,
                            ),
                        )
                    }
                }
            }

            VoiceTextField(
                value = state.explanation,
                onValueChange = vm::onExplanationChange,
                label = stringResource(R.string.disqualify_explanation),
                modifier = Modifier.fillMaxWidth(),
                minLines = 4,
            )

            if (state.reason == "OTHER" && state.explanation.isBlank()) {
                Text(
                    text = stringResource(R.string.disqualify_explanation_required),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.error,
                )
            }

            state.errorKey?.let { key ->
                Text(
                    text = disqualifyError(key),
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.error,
                )
            }

            if (state.done) {
                TPCard {
                    Text(
                        text = stringResource(R.string.outcome_disqualified),
                        style = MaterialTheme.typography.titleMedium,
                        color = colors.error,
                        fontWeight = FontWeight.SemiBold,
                    )
                }
            }

            PrimaryButton(
                text = if (state.done) stringResource(R.string.action_done) else stringResource(R.string.disqualify_title),
                onClick = { if (state.done) onDone() else vm.submit() },
                modifier = Modifier.fillMaxWidth().height(50.dp),
                enabled = state.reason != null,
                loading = state.submitting,
            )
        }
    }
}

@Composable
private fun disqualifyError(key: String): String = when (key) {
    "explanation_required" -> stringResource(R.string.disqualify_explanation_required)
    "Forbidden" -> stringResource(R.string.error_forbidden)
    "Network" -> stringResource(R.string.error_network)
    else -> stringResource(R.string.error_unknown)
}
