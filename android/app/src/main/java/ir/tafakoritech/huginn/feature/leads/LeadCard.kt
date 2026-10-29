package ir.tafakoritech.huginn.feature.leads

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.filled.Message
import androidx.compose.material.icons.outlined.MoreVert
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.design.HuginnDimens
import ir.tafakoritech.huginn.core.design.LeadScoreBadge
import ir.tafakoritech.huginn.core.design.LeadStatusBadge
import ir.tafakoritech.huginn.core.design.TPCard
import ir.tafakoritech.huginn.core.design.channelLabel
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.core.util.DateTimeUtil
import ir.tafakoritech.huginn.core.util.PhoneFormatter
import ir.tafakoritech.huginn.core.util.RtlText
import ir.tafakoritech.huginn.data.model.Lead
import androidx.compose.ui.res.stringResource

@Composable
fun LeadCard(
    lead: Lead,
    onClick: () -> Unit,
    onCall: (() -> Unit)? = null,
    onMessage: (() -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    val colors = huginnColors()
    TPCard(
        modifier = modifier.clickable(onClick = onClick),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(
                    text = lead.companyName,
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.onSurface,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                if (lead.primaryContactName != null) {
                    Spacer(Modifier.height(2.dp))
                    Text(
                        text = lead.primaryContactName +
                            (lead.primaryContactRole?.let { " — $it" } ?: ""),
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.secondaryText,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            Spacer(Modifier.width(8.dp))
            LeadScoreBadge(score = lead.score)
        }

        Spacer(Modifier.height(8.dp))
        Text(
            text = listOfNotNull(lead.city, lead.businessType).joinToString(" • "),
            style = MaterialTheme.typography.bodySmall,
            color = colors.mutedText,
        )

        Spacer(Modifier.height(8.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                LeadStatusBadge(status = lead.status)
                Spacer(Modifier.width(8.dp))
                lead.lastInteractionChannel?.let { ch ->
                    Text(
                        text = channelLabel(ch),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.mutedText,
                    )
                }
            }
            lead.lastInteractionAt?.let {
                Text(
                    text = DateTimeUtil.formatRelative(it),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.mutedText,
                )
            }
        }

        if (onCall != null || onMessage != null) {
            Spacer(Modifier.height(8.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                onCall?.let { call ->
                    IconButton(
                        onClick = call,
                        modifier = Modifier
                            .height(HuginnDimens.touchTargetCompact)
                            .width(HuginnDimens.touchTargetCompact),
                    ) {
                        Icon(
                            imageVector = Icons.Default.Call,
                            contentDescription = stringResource(R.string.action_call),
                            tint = colors.primary,
                        )
                    }
                }
                onMessage?.let { message ->
                    IconButton(
                        onClick = message,
                        modifier = Modifier
                            .height(HuginnDimens.touchTargetCompact)
                            .width(HuginnDimens.touchTargetCompact),
                    ) {
                        Icon(
                            imageVector = Icons.Default.Message,
                            contentDescription = stringResource(R.string.action_message),
                            tint = colors.primary,
                        )
                    }
                }
            }
        }
    }
}
