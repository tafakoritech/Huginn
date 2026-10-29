package ir.tafakoritech.huginn.core.design

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import ir.tafakoritech.huginn.core.theme.huginnColors
import ir.tafakoritech.huginn.core.util.RtlText

@Composable
fun TPCard(
    modifier: Modifier = Modifier,
    content: @Composable ColumnScope.() -> Unit,
) {
    val colors = huginnColors()
    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(HuginnDimens.cardRadius),
        color = colors.card,
        border = BorderStroke(1.dp, colors.border.copy(alpha = 0.6f)),
    ) {
        Column(modifier = Modifier.padding(HuginnDimens.space16), content = content)
    }
}

@Composable
fun SectionHeader(title: String, modifier: Modifier = Modifier) {
    val colors = huginnColors()
    Text(
        text = title,
        style = MaterialTheme.typography.titleSmall,
        color = colors.onSurface,
        fontWeight = FontWeight.SemiBold,
        modifier = modifier.padding(vertical = HuginnDimens.space8),
    )
}

/**
 * Label/value row. Latin values (phones, emails, URLs, codes) are wrapped so
 * they render LTR correctly inside the RTL layout.
 */
@Composable
fun InfoRow(
    label: String,
    value: String?,
    modifier: Modifier = Modifier,
    valueIsLtr: Boolean = true,
) {
    val colors = huginnColors()
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(HuginnDimens.space12),
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = colors.secondaryText,
            modifier = Modifier.weight(0.42f),
        )
        Text(
            text = if (value.isNullOrBlank()) "—" else if (valueIsLtr) RtlText.ltrEmbedding(value) else value,
            style = MaterialTheme.typography.bodyMedium,
            color = if (value.isNullOrBlank()) colors.mutedText else colors.onSurface,
            fontWeight = if (value.isNullOrBlank()) FontWeight.Normal else FontWeight.Medium,
            modifier = Modifier.weight(0.58f),
        )
    }
}

@Composable
fun InfoSpacer() = Spacer(Modifier.height(4.dp))
