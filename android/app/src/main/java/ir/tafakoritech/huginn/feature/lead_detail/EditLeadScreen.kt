package ir.tafakoritech.huginn.feature.lead_detail

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.design.HuginnDimens
import ir.tafakoritech.huginn.core.design.HuginnTopBar
import ir.tafakoritech.huginn.core.design.PrimaryButton
import ir.tafakoritech.huginn.core.design.SectionHeader
import ir.tafakoritech.huginn.core.design.TPCard
import ir.tafakoritech.huginn.core.design.VoiceTextField
import ir.tafakoritech.huginn.core.design.errorMessage
import ir.tafakoritech.huginn.core.design.statusLabel
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.data.model.Contact
import ir.tafakoritech.huginn.data.model.ContactUpdateRequest
import ir.tafakoritech.huginn.data.model.LeadStatus

/**
 * Edit-Lead screen. Mirrors the Dashboard's editable Lead fields (company /
 * business / source / description / website / instagram / status), assignment
 * and contact information, and persists through the real backend APIs.
 */
@Composable
fun EditLeadScreen(
    vm: EditLeadViewModel = hiltViewModel(),
    onBack: () -> Unit,
    onSaved: () -> Unit,
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val colors = huginnColors()

    LaunchedEffect(state.saved) {
        if (state.saved) {
            onSaved()
            vm.consumeSaved()
        }
    }

    Column(modifier = Modifier.fillMaxSize()) {
        HuginnTopBar(
            title = stringResource(R.string.edit_lead),
            onBack = onBack,
        )
        when {
            state.loading -> {
                androidx.compose.foundation.layout.Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center,
                ) { CircularProgressIndicator(color = colors.primary) }
            }

            state.error != null && state.lead == null -> {
                androidx.compose.foundation.layout.Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center,
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        Text(
                            text = errorMessage(state.error!!),
                            style = MaterialTheme.typography.bodyMedium,
                            color = colors.error,
                        )
                        TextButton(onClick = vm::load) {
                            Text(stringResource(R.string.action_retry))
                        }
                    }
                }
            }

            state.lead != null -> EditLeadForm(state = state, vm = vm)
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun EditLeadForm(
    state: EditLeadFormState,
    vm: EditLeadViewModel,
) {
    val colors = huginnColors()
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = HuginnDimens.space16, vertical = HuginnDimens.space16),
        verticalArrangement = Arrangement.spacedBy(HuginnDimens.space10),
    ) {
        state.error?.let {
            Text(
                text = errorMessage(it),
                style = MaterialTheme.typography.bodySmall,
                color = colors.error,
            )
        }

        EditSection(stringResource(R.string.edit_section_company)) {
            LabeledField(
                label = stringResource(R.string.edit_company_name) + " *",
                value = state.companyName,
                onValueChange = vm::setCompanyName,
                isError = state.companyRequired,
            )
            if (state.companyRequired) {
                Text(
                    text = stringResource(R.string.edit_company_required),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.error,
                )
            }
            LabeledField(stringResource(R.string.edit_display_name), state.displayName, vm::setDisplayName)
            LabeledField(stringResource(R.string.edit_business_type), state.businessType, vm::setBusinessType)
            LabeledField(stringResource(R.string.edit_industry), state.industry, vm::setIndustry)
            LabeledField(stringResource(R.string.edit_province), state.province, vm::setProvince)
            LabeledField(stringResource(R.string.edit_city), state.city, vm::setCity)
            LabeledField(stringResource(R.string.edit_address), state.address, vm::setAddress)
            LabeledField(stringResource(R.string.edit_website), state.website, vm::setWebsite)
            LabeledField(stringResource(R.string.edit_instagram), state.instagram, vm::setInstagram)
            LabeledField(stringResource(R.string.edit_source), state.source, vm::setSource)
            LabeledField(stringResource(R.string.edit_description), state.description, vm::setDescription, minLines = 3)
        }

        EditSection(stringResource(R.string.edit_section_status)) {
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                LeadStatus.all.forEach { status ->
                    FilterChip(
                        selected = state.status == status,
                        onClick = { vm.setStatus(status) },
                        label = { Text(statusLabel(status)) },
                        colors = FilterChipDefaults.filterChipColors(
                            selectedContainerColor = colors.primaryContainer,
                            selectedLabelColor = colors.onPrimaryContainer,
                        ),
                    )
                }
            }
        }

        if (state.canAssign) {
            EditSection(stringResource(R.string.edit_section_assignment)) {
                val currentName = state.lead?.ownerName ?: state.agents.firstOrNull { it.userId == state.assigneeId }?.name
                if (currentName.isNullOrBlank()) {
                    Text(
                        text = stringResource(R.string.edit_unassigned),
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.mutedText,
                    )
                } else {
                    Text(
                        text = currentName,
                        style = MaterialTheme.typography.bodyMedium,
                        color = colors.onSurface,
                        fontWeight = FontWeight.Medium,
                    )
                }
                if (state.agentsLoading) {
                    Text(
                        text = stringResource(R.string.edit_loading_agents),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.mutedText,
                    )
                } else if (state.agents.isNotEmpty()) {
                    Spacer(Modifier.height(4.dp))
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        state.agents.forEach { agent ->
                            FilterChip(
                                selected = agent.userId == state.assigneeId,
                                onClick = { vm.assign(agent.userId) },
                                label = { Text(agent.name) },
                                enabled = !state.assigning,
                            )
                        }
                    }
                }
            }
        }

        if (state.lead?.contacts.isNullOrEmpty().not()) {
            EditSection(stringResource(R.string.edit_section_contacts)) {
                state.lead?.contacts?.forEach { contact ->
                    ContactEditCard(
                        contact = contact,
                        saving = state.saving,
                        onSave = { request -> vm.saveContact(contact.id, request) },
                    )
                }
            }
        }

        EditSection(stringResource(R.string.edit_section_note)) {
            VoiceTextField(
                value = state.note,
                onValueChange = vm::setNote,
                label = stringResource(R.string.edit_change_note),
                minLines = 2,
            )
        }

        PrimaryButton(
            text = stringResource(R.string.action_save),
            onClick = vm::save,
            modifier = Modifier.fillMaxWidth().height(50.dp),
            loading = state.saving,
        )
        Spacer(Modifier.height(24.dp))
    }
}

@Composable
private fun EditSection(title: String, content: @Composable () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        SectionHeader(title)
        TPCard {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                content()
            }
        }
    }
}

@Composable
private fun LabeledField(
    label: String,
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    minLines: Int = 1,
    isError: Boolean = false,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        label = { Text(label) },
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(10.dp),
        minLines = minLines,
        isError = isError,
    )
}

@Composable
private fun ContactEditCard(
    contact: Contact,
    saving: Boolean,
    onSave: (ContactUpdateRequest) -> Unit,
) {
    val colors = huginnColors()
    var expanded by rememberSaveable(contact.id) { mutableStateOf(false) }
    var fullName by rememberSaveable(contact.id) { mutableStateOf(contact.fullName.orEmpty()) }
    var jobTitle by rememberSaveable(contact.id) { mutableStateOf(contact.jobTitle.orEmpty()) }
    var mobile by rememberSaveable(contact.id) { mutableStateOf(contact.mobile.orEmpty()) }
    var phone by rememberSaveable(contact.id) { mutableStateOf(contact.phone.orEmpty()) }
    var email by rememberSaveable(contact.id) { mutableStateOf(contact.email.orEmpty()) }
    var whatsapp by rememberSaveable(contact.id) { mutableStateOf(contact.whatsapp.orEmpty()) }
    var telegram by rememberSaveable(contact.id) { mutableStateOf(contact.telegram.orEmpty()) }
    var bale by rememberSaveable(contact.id) { mutableStateOf(contact.bale.orEmpty()) }
    var isDecisionMaker by rememberSaveable(contact.id) { mutableStateOf(contact.isDecisionMaker) }
    var isPrimary by rememberSaveable(contact.id) { mutableStateOf(contact.isPrimaryContact) }

    TPCard(modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clickable { expanded = !expanded },
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.weight(1f)) {
                Text(
                    text = contact.displayName.ifBlank { stringResource(R.string.unknown) },
                    style = MaterialTheme.typography.bodyLarge,
                    color = colors.onSurface,
                    fontWeight = FontWeight.Medium,
                )
                contact.jobTitle?.let {
                    Text(
                        text = it,
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.secondaryText,
                    )
                }
            }
            Text(
                text = stringResource(if (expanded) R.string.edit_collapse else R.string.edit_expand),
                style = MaterialTheme.typography.labelSmall,
                color = colors.primary,
            )
        }
        if (expanded) {
            Spacer(Modifier.height(10.dp))
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                LabeledField(stringResource(R.string.edit_full_name), fullName, onValueChange = { fullName = it })
                LabeledField(stringResource(R.string.edit_job_title), jobTitle, onValueChange = { jobTitle = it })
                LabeledField(stringResource(R.string.edit_mobile), mobile, onValueChange = { mobile = it })
                LabeledField(stringResource(R.string.edit_phone), phone, onValueChange = { phone = it })
                LabeledField(stringResource(R.string.edit_email), email, onValueChange = { email = it })
                LabeledField(stringResource(R.string.edit_whatsapp), whatsapp, onValueChange = { whatsapp = it })
                LabeledField(stringResource(R.string.edit_telegram), telegram, onValueChange = { telegram = it })
                LabeledField(stringResource(R.string.edit_bale), bale, onValueChange = { bale = it })
                ToggleRow(
                    label = stringResource(R.string.edit_decision_maker),
                    checked = isDecisionMaker,
                    onCheckedChange = { isDecisionMaker = it },
                )
                ToggleRow(
                    label = stringResource(R.string.edit_primary_contact),
                    checked = isPrimary,
                    onCheckedChange = { isPrimary = it },
                )
                PrimaryButton(
                    text = stringResource(R.string.action_save),
                    onClick = {
                        onSave(
                            ContactUpdateRequest(
                                fullName = fullName.trim().takeIf { it.isNotEmpty() },
                                jobTitle = jobTitle.trim().takeIf { it.isNotEmpty() },
                                mobile = mobile.trim().takeIf { it.isNotEmpty() },
                                phone = phone.trim().takeIf { it.isNotEmpty() },
                                email = email.trim().takeIf { it.isNotEmpty() },
                                whatsapp = whatsapp.trim().takeIf { it.isNotEmpty() },
                                telegram = telegram.trim().takeIf { it.isNotEmpty() },
                                bale = bale.trim().takeIf { it.isNotEmpty() },
                                isDecisionMaker = isDecisionMaker,
                                isPrimaryContact = isPrimary,
                            ),
                        )
                    },
                    modifier = Modifier.fillMaxWidth(),
                    loading = saving,
                )
            }
        }
    }
}

@Composable
private fun ToggleRow(
    label: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
) {
    val colors = huginnColors()
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = colors.onSurface,
            modifier = Modifier.weight(1f),
        )
        Switch(checked = checked, onCheckedChange = onCheckedChange)
    }
}
