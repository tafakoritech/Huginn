package ir.tafakoritech.huginn.feature.settings

import android.Manifest
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
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
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.design.HuginnDimens
import ir.tafakoritech.huginn.core.design.HuginnTopBar
import ir.tafakoritech.huginn.core.design.SecondaryButton
import ir.tafakoritech.huginn.core.design.TPCard
import ir.tafakoritech.huginn.core.permissions.PermissionsManager
import ir.tafakoritech.huginn.core.theme.ThemeMode
import ir.tafakoritech.huginn.core.theme.huginnColors

@Composable
fun SettingsScreen(
    vm: SettingsViewModel,
    onBack: () -> Unit,
) {
    val colors = huginnColors()
    val themeMode by vm.themeMode.collectAsStateWithLifecycle(initialValue = ThemeMode.SYSTEM)
    val language by vm.language.collectAsStateWithLifecycle(initialValue = "fa")
    val notificationsEnabled by vm.notificationsEnabled.collectAsStateWithLifecycle(initialValue = false)

    val pm = vm.permissionsManager
    var permissionDenied by rememberSaveable { mutableStateOf(false) }
    val permissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted ->
        if (granted) {
            vm.setNotificationsEnabled(true)
            permissionDenied = false
        } else {
            vm.setNotificationsEnabled(false)
            permissionDenied = true
        }
    }

    Column(modifier = Modifier.fillMaxSize()) {
        HuginnTopBar(title = stringResource(R.string.settings_title), onBack = onBack)
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = HuginnDimens.space16, vertical = HuginnDimens.space16),
            verticalArrangement = Arrangement.spacedBy(HuginnDimens.space12),
        ) {
            // Theme
            TPCard {
                Text(
                    text = stringResource(R.string.theme),
                    style = MaterialTheme.typography.titleSmall,
                    color = colors.onSurface,
                )
                Spacer(Modifier.height(8.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    ThemeOption(label = stringResource(R.string.theme_system), selected = themeMode == ThemeMode.SYSTEM) {
                        vm.setThemeMode(ThemeMode.SYSTEM)
                    }
                    ThemeOption(label = stringResource(R.string.theme_light), selected = themeMode == ThemeMode.LIGHT) {
                        vm.setThemeMode(ThemeMode.LIGHT)
                    }
                    ThemeOption(label = stringResource(R.string.theme_dark), selected = themeMode == ThemeMode.DARK) {
                        vm.setThemeMode(ThemeMode.DARK)
                    }
                }
            }

            // Language
            TPCard {
                Text(
                    text = stringResource(R.string.language),
                    style = MaterialTheme.typography.titleSmall,
                    color = colors.onSurface,
                )
                Spacer(Modifier.height(8.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    ThemeOption(label = stringResource(R.string.language_fa), selected = language == "fa") {
                        vm.setLanguage("fa")
                    }
                    ThemeOption(label = stringResource(R.string.language_en), selected = language == "en") {
                        vm.setLanguage("en")
                    }
                }
            }

            // Notifications
            TPCard {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) {
                        Text(
                            text = stringResource(R.string.notifications_title),
                            style = MaterialTheme.typography.titleSmall,
                            color = colors.onSurface,
                        )
                        Spacer(Modifier.height(2.dp))
                        Text(
                            text = stringResource(R.string.notification_permission_text),
                            style = MaterialTheme.typography.bodySmall,
                            color = colors.mutedText,
                        )
                    }
                    Switch(
                        checked = notificationsEnabled,
                        onCheckedChange = { checked ->
                            if (checked && pm.shouldRequestNotificationPermission()) {
                                permissionLauncher.launch(pm.notificationPermission)
                            } else {
                                vm.setNotificationsEnabled(checked)
                            }
                        },
                    )
                }
                if (permissionDenied) {
                    Text(
                        text = stringResource(R.string.notification_permission_text),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.error,
                        modifier = Modifier.padding(top = 6.dp),
                    )
                }
            }

            // Privacy
            TPCard {
                Text(
                    text = stringResource(R.string.privacy_title),
                    style = MaterialTheme.typography.titleSmall,
                    color = colors.onSurface,
                    fontWeight = FontWeight.SemiBold,
                )
                Spacer(Modifier.height(6.dp))
                PrivacyLine(stringResource(R.string.privacy_contacts))
                PrivacyLine(stringResource(R.string.privacy_sms))
                PrivacyLine(stringResource(R.string.privacy_sms_send))
                PrivacyLine(stringResource(R.string.privacy_email))
            }

            Spacer(Modifier.height(8.dp))
            SecondaryButton(
                text = stringResource(R.string.action_back),
                onClick = onBack,
                modifier = Modifier.fillMaxWidth(),
            )
        }
    }
}

@Composable
private fun ThemeOption(label: String, selected: Boolean, onClick: () -> Unit) {
    val colors = huginnColors()
    androidx.compose.material3.FilterChip(
        selected = selected,
        onClick = onClick,
        label = { Text(label) },
        colors = androidx.compose.material3.FilterChipDefaults.filterChipColors(
            selectedContainerColor = colors.activeCard,
            selectedLabelColor = colors.onSurface,
        ),
    )
}

@Composable
private fun PrivacyLine(text: String) {
    val colors = huginnColors()
    Text(
        text = "• " + text,
        style = MaterialTheme.typography.bodySmall,
        color = colors.secondaryText,
        modifier = Modifier.padding(vertical = 3.dp),
    )
}
