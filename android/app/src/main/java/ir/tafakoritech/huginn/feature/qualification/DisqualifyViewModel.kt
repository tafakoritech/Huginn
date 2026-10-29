package ir.tafakoritech.huginn.feature.qualification

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.data.repository.QualificationRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

data class DisqualifyUiState(
    val leadId: String,
    val reason: String? = null,
    val explanation: String = "",
    val submitting: Boolean = false,
    val done: Boolean = false,
    val errorKey: String? = null,
)

@HiltViewModel
class DisqualifyViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val qualificationRepository: QualificationRepository,
) : ViewModel() {

    private val leadId: String = checkNotNull(savedStateHandle["leadId"])

    private val _state = MutableStateFlow(DisqualifyUiState(leadId = leadId))
    val state: StateFlow<DisqualifyUiState> = _state

    fun selectReason(reason: String) {
        _state.value = _state.value.copy(reason = reason, errorKey = null)
    }

    fun onExplanationChange(text: String) {
        _state.value = _state.value.copy(explanation = text)
    }

    fun submit() {
        val reason = _state.value.reason ?: return
        val explanation = _state.value.explanation
        if (reason == "OTHER" && explanation.isBlank()) {
            _state.value = _state.value.copy(errorKey = "explanation_required")
            return
        }
        viewModelScope.launch {
            _state.value = _state.value.copy(submitting = true, errorKey = null)
            when (val result = qualificationRepository.disqualify(leadId, reason, explanation.ifBlank { null })) {
                is AppResult.Success -> _state.value = _state.value.copy(submitting = false, done = true)
                is AppResult.Failure -> _state.value = _state.value.copy(submitting = false, errorKey = result.error::class.simpleName)
            }
        }
    }
}
