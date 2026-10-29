package ir.tafakoritech.huginn.feature.lead_detail

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import ir.tafakoritech.huginn.core.util.AppError
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.data.model.Agent
import ir.tafakoritech.huginn.data.model.ContactUpdateRequest
import ir.tafakoritech.huginn.data.model.LeadDetail
import ir.tafakoritech.huginn.data.repository.LeadPatch
import ir.tafakoritech.huginn.data.repository.LeadRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class EditLeadFormState(
    val loading: Boolean = true,
    val lead: LeadDetail? = null,
    // Lead fields
    val companyName: String = "",
    val displayName: String = "",
    val businessType: String = "",
    val industry: String = "",
    val province: String = "",
    val city: String = "",
    val address: String = "",
    val website: String = "",
    val instagram: String = "",
    val source: String = "",
    val description: String = "",
    val status: String = "",
    val note: String = "",
    val saving: Boolean = false,
    val error: AppError? = null,
    val companyRequired: Boolean = false,
    // One-time save success (consumed by the UI to navigate back).
    val saved: Boolean = false,
    // Assignment
    val agents: List<Agent> = emptyList(),
    val agentsLoading: Boolean = false,
    val assigning: Boolean = false,
    val assigneeId: String = "",
) {
    val canAssign: Boolean get() = lead?.permissions?.canAssign == true
}

/**
 * Edit-Lead form backed by the real backend (PATCH /leads/{id}, contacts and
 * assign endpoints). Saves go through [LeadRepository]; after a successful save
 * the caller returns to the detail screen which reloads from the backend.
 */
@HiltViewModel
class EditLeadViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val leadRepository: LeadRepository,
) : ViewModel() {

    private val leadId: String = checkNotNull(savedStateHandle["leadId"])

    private val _state = MutableStateFlow(EditLeadFormState())
    val state: StateFlow<EditLeadFormState> = _state

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
            _state.value = _state.value.copy(loading = true, error = null)
            when (val result = leadRepository.leadDetail(leadId)) {
                is AppResult.Success -> {
                    val lead = result.data
                    _state.value = _state.value.copy(
                        loading = false,
                        lead = lead,
                        companyName = lead.companyName,
                        displayName = lead.displayName.orEmpty(),
                        businessType = lead.businessType.orEmpty(),
                        industry = lead.industry.orEmpty(),
                        province = lead.province.orEmpty(),
                        city = lead.city.orEmpty(),
                        address = lead.address.orEmpty(),
                        website = lead.website.orEmpty(),
                        instagram = lead.instagram.orEmpty(),
                        source = lead.source.orEmpty(),
                        description = lead.description.orEmpty(),
                        status = lead.status,
                        assigneeId = lead.ownerUserId.orEmpty(),
                    )
                    if (lead.permissions.canAssign) loadAgents()
                }
                is AppResult.Failure -> {
                    _state.value = _state.value.copy(loading = false, error = result.error)
                }
            }
        }
    }

    private fun loadAgents() {
        viewModelScope.launch {
            _state.value = _state.value.copy(agentsLoading = true)
            when (val result = leadRepository.agents()) {
                is AppResult.Success -> _state.value = _state.value.copy(agentsLoading = false, agents = result.data)
                is AppResult.Failure -> _state.value = _state.value.copy(agentsLoading = false)
            }
        }
    }

    fun setCompanyName(value: String) = _state.update { it.copy(companyName = value, companyRequired = false) }
    fun setDisplayName(value: String) = _state.update { it.copy(displayName = value) }
    fun setBusinessType(value: String) = _state.update { it.copy(businessType = value) }
    fun setIndustry(value: String) = _state.update { it.copy(industry = value) }
    fun setProvince(value: String) = _state.update { it.copy(province = value) }
    fun setCity(value: String) = _state.update { it.copy(city = value) }
    fun setAddress(value: String) = _state.update { it.copy(address = value) }
    fun setWebsite(value: String) = _state.update { it.copy(website = value) }
    fun setInstagram(value: String) = _state.update { it.copy(instagram = value) }
    fun setSource(value: String) = _state.update { it.copy(source = value) }
    fun setDescription(value: String) = _state.update { it.copy(description = value) }
    fun setStatus(value: String) = _state.update { it.copy(status = value) }
    fun setNote(value: String) = _state.update { it.copy(note = value) }

    fun consumeError() = _state.update { it.copy(error = null) }
    fun consumeSaved() = _state.update { it.copy(saved = false) }

    /** Save lead fields through the backend; a blank optional field clears it. */
    fun save() {
        val company = _state.value.companyName.trim()
        if (company.isEmpty()) {
            _state.value = _state.value.copy(companyRequired = true)
            return
        }
        if (_state.value.saving) return
        val patch = LeadPatch(
            companyName = company,
            displayName = blankToNull(_state.value.displayName),
            businessType = blankToNull(_state.value.businessType),
            industry = blankToNull(_state.value.industry),
            province = blankToNull(_state.value.province),
            city = blankToNull(_state.value.city),
            address = blankToNull(_state.value.address),
            website = blankToNull(_state.value.website),
            instagram = blankToNull(_state.value.instagram),
            source = blankToNull(_state.value.source),
            description = blankToNull(_state.value.description),
            status = _state.value.status.takeIf { it != _state.value.lead?.status },
        )
        viewModelScope.launch {
            _state.value = _state.value.copy(saving = true, error = null)
            when (val result = leadRepository.updateLead(leadId, patch, _state.value.note.takeIf { it.isNotBlank() })) {
                is AppResult.Success -> _state.value = _state.value.copy(saving = false, saved = true)
                is AppResult.Failure -> _state.value = _state.value.copy(saving = false, error = result.error)
            }
        }
    }

    /** Assign the lead to an agent immediately (Dashboard-style). */
    fun assign(assigneeId: String) {
        if (assigneeId.isBlank() || _state.value.assigning) return
        viewModelScope.launch {
            _state.value = _state.value.copy(assigning = true, error = null)
            when (val result = leadRepository.assignLead(leadId, assigneeId)) {
                is AppResult.Success -> {
                    _state.value = _state.value.copy(assigning = false, assigneeId = assigneeId)
                    load()
                }
                is AppResult.Failure -> _state.value = _state.value.copy(assigning = false, error = result.error)
            }
        }
    }

    /** Patch a contact on the lead, then refresh the form from the backend. */
    fun saveContact(contactId: String, request: ContactUpdateRequest) {
        if (_state.value.saving) return
        viewModelScope.launch {
            _state.value = _state.value.copy(saving = true, error = null)
            when (val result = leadRepository.updateContact(leadId, contactId, request)) {
                is AppResult.Success -> {
                    _state.value = _state.value.copy(saving = false)
                    load()
                }
                is AppResult.Failure -> _state.value = _state.value.copy(saving = false, error = result.error)
            }
        }
    }

    private fun blankToNull(value: String): String? = value.trim().takeIf { it.isNotEmpty() }
}
