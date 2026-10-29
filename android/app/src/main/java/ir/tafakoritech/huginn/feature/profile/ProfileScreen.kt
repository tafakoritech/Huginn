package ir.tafakoritech.huginn.feature.profile

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import ir.tafakoritech.huginn.core.design.SecondaryButton
import ir.tafakoritech.huginn.core.design.TPCard
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.core.util.RtlText

@Composable
fun ProfileScreen(
    vm: ProfileViewModel,
    onLogout: () -> Unit,
    onSettings: () -> Unit,
) {
    val colors = huginnColors()
    val user by vm.user.collectAsStateWithLifecycle()

    Column(modifier = Modifier.fillMaxSize()) {
        HuginnTopBar(title = stringResource(R.string.profile_title))
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(HuginnDimens.space16),
            verticalArrangement = Arrangement.spacedBy(HuginnDimens.space12),
        ) {
            TPCard {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        imageVector = Icons.Default.Person,
                        contentDescription = null,
                        tint = colors.primary,
                    )
                    Column(Modifier.padding(start = 12.dp)) {
                        Text(
                            text = user?.name ?: "—",
                            style = MaterialTheme.typography.titleMedium,
                            color = colors.onSurface,
                            fontWeight = FontWeight.SemiBold,
                        )
                        Text(
                            text = RtlText.ltrEmbedding(user?.username ?: ""),
                            style = MaterialTheme.typography.bodySmall,
                            color = colors.mutedText,
                        )
                        user?.role?.let { role ->
                            Text(
                                text = roleLabel(role),
                                style = MaterialTheme.typography.labelSmall,
                                color = colors.secondaryText,
                            )
                        }
                    }
                }
            }

            TPCard(modifier = Modifier.clickable(onClick = onSettings)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        imageVector = Icons.Default.Settings,
                        contentDescription = null,
                        tint = colors.onSurface,
                    )
                    Text(
                        text = stringResource(R.string.settings_title),
                        style = MaterialTheme.typography.bodyLarge,
                        color = colors.onSurface,
                        modifier = Modifier.padding(start = 12.dp),
                    )
                }
            }

            Spacer(Modifier.height(16.dp))
            SecondaryButton(
                text = stringResource(R.string.logout),
                onClick = onLogout,
                modifier = Modifier.fillMaxWidth(),
            )
        }
    }
}

@Composable
private fun roleLabel(role: String): String = when (role) {
    "sales_agent" -> stringResource(R.string.agent)
    "sales_development_manager" -> stringResource(R.string.manager)
    else -> stringResource(R.string.unknown)
}
