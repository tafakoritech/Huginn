package ir.tafakoritech.huginn.feature.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import ir.tafakoritech.huginn.core.datastore.SettingsRepository
import ir.tafakoritech.huginn.core.locale.LanguageResolver
import ir.tafakoritech.huginn.core.locale.LocaleManager
import ir.tafakoritech.huginn.core.permissions.PermissionsManager
import ir.tafakoritech.huginn.core.theme.ThemeMode
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class SettingsViewModel @Inject constructor(
    private val settingsRepository: SettingsRepository,
    private val localeManager: LocaleManager,
    val permissionsManager: PermissionsManager,
) : ViewModel() {

    val themeMode: Flow<ThemeMode> get() = settingsRepository.themeMode

    /** Effective language: saved user preference, else device-language detection. */
    val language: StateFlow<String> = settingsRepository.language
        .map { it ?: LanguageResolver.fromLocale() }
        .stateIn(viewModelScope, SharingStarted.Eagerly, LanguageResolver.fromLocale())

    val notificationsEnabled: Flow<Boolean> get() = settingsRepository.notificationsEnabled

    fun setThemeMode(mode: ThemeMode) {
        viewModelScope.launch { settingsRepository.setThemeMode(mode) }
    }

    fun setLanguage(lang: String) {
        viewModelScope.launch {
            settingsRepository.setLanguage(lang)
            localeManager.apply(lang)
        }
    }

    fun setNotificationsEnabled(enabled: Boolean) {
        viewModelScope.launch { settingsRepository.setNotificationsEnabled(enabled) }
    }
}
