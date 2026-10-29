package ir.tafakoritech.huginn.feature.qualification

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.getOrNull
import ir.tafakoritech.huginn.data.model.QualifyResponse
import ir.tafakoritech.huginn.data.repository.ChecklistKey
import ir.tafakoritech.huginn.data.repository.QualificationRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

/** Qualification checklist keys (backend is the authority on the rules). */
object QualificationChecklist {
    const val BUSINESS_RELEVANT = ChecklistKey.BUSINESS_RELEVANT
    const val CONTACT_VALID = ChecklistKey.CONTACT_VALID
    const val MEANINGFUL_CONTACT = ChecklistKey.MEANINGFUL_CONTACT
    const val NEED_OR_INTEREST = ChecklistKey.NEED_OR_INTEREST
    const val DECISION_MAKER = ChecklistKey.DECISION_MAKER
    const val NEXT_ACTION = ChecklistKey.NEXT_ACTION
    const val MINIMUM_SCORE = "minimum_score_reached"

    val all = listOf(
        BUSINESS_RELEVANT,
        CONTACT_VALID,
        MEANINGFUL_CONTACT,
        NEED_OR_INTEREST,
        DECISION_MAKER,
        NEXT_ACTION,
        MINIMUM_SCORE,
    )
}

data class QualifyUiState(
    val leadId: String,
    val checks: Map<String, Boolean> = emptyMap(),
    val submitting: Boolean = false,
    val result: QualifyResponse? = null,
    val errorKey: String? = null,
)

@HiltViewModel
class QualifyViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val qualificationRepository: QualificationRepository,
) : ViewModel() {

    private val leadId: String = checkNotNull(savedStateHandle["leadId"])

    private val _state = MutableStateFlow(
        QualifyUiState(leadId = leadId, checks = QualificationChecklist.all.associateWith { false }),
    )
    val state: StateFlow<QualifyUiState> = _state

    fun toggleCheck(key: String) {
        val checks = _state.value.checks.toMutableMap()
        checks[key] = !(checks[key] ?: false)
        _state.value = _state.value.copy(checks = checks)
    }

    fun submit() {
        viewModelScope.launch {
            _state.value = _state.value.copy(submitting = true, errorKey = null)
            when (val result = qualificationRepository.qualify(leadId, _state.value.checks, null)) {
                is AppResult.Success -> {
                    _state.value = _state.value.copy(submitting = false, result = result.data)
                }
                is AppResult.Failure -> {
                    _state.value = _state.value.copy(submitting = false, errorKey = result.error::class.simpleName)
                }
            }
        }
    }
}
