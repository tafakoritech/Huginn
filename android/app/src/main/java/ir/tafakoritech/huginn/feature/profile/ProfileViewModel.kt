package ir.tafakoritech.huginn.feature.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import ir.tafakoritech.huginn.core.auth.SessionManager
import ir.tafakoritech.huginn.core.auth.SessionState
import ir.tafakoritech.huginn.data.model.User
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class ProfileViewModel @Inject constructor(
    sessionManager: SessionManager,
) : ViewModel() {

    private val _user = MutableStateFlow<User?>(null)
    val user: StateFlow<User?> = _user

    init {
        viewModelScope.launch {
            sessionManager.state.collectLatest { state ->
                _user.value = (state as? SessionState.Authenticated)?.user
            }
        }
    }
}
