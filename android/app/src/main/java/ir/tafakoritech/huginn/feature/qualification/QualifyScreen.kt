package ir.tafakoritech.huginn.feature.qualification

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Checkbox
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
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
import ir.tafakoritech.huginn.core.design.errorMessage
import ir.tafakoritech.huginn.core.theme.huginnColors

@Composable
fun QualifyScreen(
    vm: QualifyViewModel,
    onDone: () -> Unit,
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val colors = huginnColors()

    Column(modifier = Modifier.fillMaxSize()) {
        HuginnTopBar(title = stringResource(R.string.qualify_lead), onBack = onDone)
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = HuginnDimens.space16, vertical = HuginnDimens.space16),
            verticalArrangement = Arrangement.spacedBy(HuginnDimens.space12),
        ) {
            QualifyChecklist(
                checks = state.checks,
                onToggle = vm::toggleCheck,
            )

            state.errorKey?.let {
                Text(
                    text = errorMessageForKey(it),
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.error,
                )
            }

            if (state.result?.qualified == true) {
                TPCard {
                    Text(
                        text = stringResource(R.string.lead_qualified),
                        style = MaterialTheme.typography.titleMedium,
                        color = colors.success,
                        fontWeight = FontWeight.SemiBold,
                    )
                    Spacer(Modifier.height(4.dp))
                    Text(
                        text = stringResource(R.string.ready_for_muninn),
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.secondaryText,
                    )
                }
            }

            if (state.result == null) {
                PrimaryButton(
                    text = stringResource(R.string.qualify_title),
                    onClick = vm::submit,
                    modifier = Modifier.fillMaxWidth().height(50.dp),
                    loading = state.submitting,
                )
            } else {
                PrimaryButton(
                    text = stringResource(R.string.action_done),
                    onClick = onDone,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }
    }
}


/** Stateless qualification checklist (UI-testable). */
@Composable
fun QualifyChecklist(
    checks: Map<String, Boolean>,
    onToggle: (String) -> Unit,
) {
    val colors = huginnColors()
    TPCard {
        Text(
            text = stringResource(R.string.qualify_lead),
            style = MaterialTheme.typography.titleSmall,
            color = colors.onSurface,
        )
        Spacer(Modifier.height(8.dp))
        checks.keys.forEach { key ->
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(
                    checked = checks[key] ?: false,
                    onCheckedChange = { onToggle(key) },
                )
                Text(
                    text = checklistLabel(key),
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.onSurface,
                )
            }
        }
    }
}

@Composable
private fun checklistLabel(key: String): String = when (key) {
    QualificationChecklist.BUSINESS_RELEVANT -> stringResource(R.string.checklist_business_relevant)
    QualificationChecklist.CONTACT_VALID -> stringResource(R.string.checklist_contact_valid)
    QualificationChecklist.MEANINGFUL_CONTACT -> stringResource(R.string.checklist_meaningful_contact)
    QualificationChecklist.NEED_OR_INTEREST -> stringResource(R.string.checklist_need_interest)
    QualificationChecklist.DECISION_MAKER -> stringResource(R.string.checklist_decision_maker)
    QualificationChecklist.NEXT_ACTION -> stringResource(R.string.checklist_next_action)
    QualificationChecklist.MINIMUM_SCORE -> stringResource(R.string.checklist_minimum_score)
    else -> key
}

@Composable
private fun errorMessageForKey(key: String): String = when (key) {
    "Forbidden" -> stringResource(R.string.error_forbidden)
    "AlreadyQualified" -> stringResource(R.string.error_already_qualified)
    "Network" -> stringResource(R.string.error_network)
    else -> stringResource(R.string.error_unknown)
}
