package ir.tafakoritech.huginn.core.design

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CloudOff
import androidx.compose.material.icons.filled.Inbox
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.core.util.ScoreClassifier
import ir.tafakoritech.huginn.core.util.ScoreLevel
import androidx.compose.ui.res.stringResource

@Composable
fun LeadScoreBadge(score: Int, modifier: Modifier = Modifier) {
    val colors = huginnColors()
    val level = ScoreClassifier.classify(score)
    val bg = when (level) {
        ScoreLevel.HIGH -> colors.successContainer
        ScoreLevel.MEDIUM -> colors.warningContainer
        ScoreLevel.LOW -> colors.errorContainer
    }
    val fg = when (level) {
        ScoreLevel.HIGH -> colors.onSuccessContainer
        ScoreLevel.MEDIUM -> colors.onWarningContainer
        ScoreLevel.LOW -> colors.onErrorContainer
    }
    Row(
        modifier = modifier
            .background(bg, RoundedCornerShape(HuginnDimens.badgeRadius))
            .padding(horizontal = 10.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Text(
            text = stringResource(R.string.score, score),
            style = MaterialTheme.typography.labelMedium,
            color = fg,
            fontWeight = FontWeight.SemiBold,
        )
    }
}

@Composable
fun LeadStatusBadge(status: String?, modifier: Modifier = Modifier) {
    val colors = huginnColors()
    val fg = statusColor(status)
    Box(
        modifier = modifier
            .background(fg.copy(alpha = 0.14f), RoundedCornerShape(HuginnDimens.badgeRadius))
            .padding(horizontal = 10.dp, vertical = 4.dp),
    ) {
        Text(
            text = statusLabel(status),
            style = MaterialTheme.typography.labelSmall,
            color = fg,
            fontWeight = FontWeight.Medium,
        )
    }
}

@Composable
fun EmptyState(
    title: String,
    hint: String? = null,
    icon: ImageVector = Icons.Default.Inbox,
    modifier: Modifier = Modifier,
) {
    val colors = huginnColors()
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(HuginnDimens.space32),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = colors.mutedText,
            modifier = Modifier.size(48.dp),
        )
        Spacer(Modifier.height(HuginnDimens.space12))
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
            color = colors.onSurface,
            textAlign = TextAlign.Center,
        )
        if (hint != null) {
            Spacer(Modifier.height(HuginnDimens.space4))
            Text(
                text = hint,
                style = MaterialTheme.typography.bodySmall,
                color = colors.mutedText,
                textAlign = TextAlign.Center,
            )
        }
    }
}

@Composable
fun ErrorState(
    message: String,
    onRetry: (() -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    val colors = huginnColors()
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(HuginnDimens.space24),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = message,
            style = MaterialTheme.typography.bodyMedium,
            color = colors.error,
            textAlign = TextAlign.Center,
        )
        if (onRetry != null) {
            Spacer(Modifier.height(HuginnDimens.space12))
            SecondaryButton(text = stringResource(R.string.action_retry), onClick = onRetry)
        }
    }
}

@Composable
fun OfflineBanner(modifier: Modifier = Modifier) {
    val colors = huginnColors()
    Row(
        modifier = modifier
            .fillMaxWidth()
            .background(colors.warningContainer, RoundedCornerShape(HuginnDimens.controlRadius))
            .padding(horizontal = HuginnDimens.space12, vertical = HuginnDimens.space8),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Icon(
            imageVector = Icons.Default.CloudOff,
            contentDescription = null,
            tint = colors.onWarningContainer,
            modifier = Modifier.size(16.dp),
        )
        Text(
            text = stringResource(R.string.offline_banner),
            style = MaterialTheme.typography.bodySmall,
            color = colors.onWarningContainer,
        )
    }
}

@Composable
fun LoadingSkeleton(modifier: Modifier = Modifier) {
    val colors = huginnColors()
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(HuginnDimens.space16),
        verticalArrangement = Arrangement.spacedBy(HuginnDimens.space12),
    ) {
        repeat(4) {
            Surface(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(88.dp),
                color = colors.surfaceContainerHigh,
                shape = RoundedCornerShape(HuginnDimens.cardRadius),
            ) {}
        }
    }
}

@Composable
fun ColorDot(color: Color, size: Int = 10) {
    Box(
        Modifier
            .size(size.dp)
            .background(color, RoundedCornerShape(size = HuginnDimens.badgeRadius)),
    )
}
