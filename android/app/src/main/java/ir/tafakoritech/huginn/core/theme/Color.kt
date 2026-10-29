package ir.tafakoritech.huginn.core.theme

import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

// ---------------------------------------------------------------------------
// Tafakor Pardazan official palettes.
// ---------------------------------------------------------------------------

// Dark palette tokens.
internal val TpDarkBackground = Color(0xFF021C25)
internal val TpDarkSurface = Color(0xFF091E27)
internal val TpDarkCard = Color(0xFF0E3338)
internal val TpDarkElevated = Color(0xFF053A3D)
internal val TpDarkBorder = Color(0xFF165A54)
internal val TpDarkTextPrimary = Color(0xFFF1F7F4)
internal val TpDarkTextSecondary = Color(0xFFA9C2B8)
internal val TpDarkTextMuted = Color(0xFF7C908B)

// Light palette tokens.
internal val TpLightBackground = Color(0xFFF5F9F7)
internal val TpLightSecondaryBackground = Color(0xFFEDF5F1)
internal val TpLightCard = Color(0xFFFFFFFF)
internal val TpLightActiveCard = Color(0xFFE3F2E8)
internal val TpLightDivider = Color(0xFFD7E4DE)
internal val TpLightBorder = Color(0xFFC5D8CF)
internal val TpLightTextPrimary = Color(0xFF021C25)
internal val TpLightTextSecondary = Color(0xFF49635F)
internal val TpLightTextMuted = Color(0xFF7C908B)

// Brand / semantic colors shared across themes.
internal val TpPrimary = Color(0xFF69B437)
internal val TpPrimaryHighlight = Color(0xFF91DA73)
internal val TpDeepGreen = Color(0xFF23643D)
internal val TpDarkGreen = Color(0xFF165A54)
internal val TpTeal = Color(0xFF0E3338)

internal val TpSuccess = Color(0xFF49A85A)
internal val TpInfo = Color(0xFF3C82C4)
internal val TpWarning = Color(0xFFD99A28)
internal val TpError = Color(0xFFD95C5C)
internal val TpNeutral = Color(0xFF80918C)

// On-primary label color: dark green keeps a strong contrast on the bright brand green.
internal val OnPrimary = Color(0xFF022C22)

internal val ScoreHigh = TpSuccess
internal val ScoreMedium = TpWarning
internal val ScoreLow = TpError

/**
 * Semantic color set for Huginn. Additional tokens on top of Material 3.
 */
@Immutable
data class HuginnColors(
    val primary: Color,
    val onPrimary: Color,
    val primaryContainer: Color,
    val onPrimaryContainer: Color,
    val primaryHighlight: Color,
    val secondary: Color,
    val onSecondary: Color,
    val background: Color,
    val onBackground: Color,
    val surface: Color,
    val onSurface: Color,
    val surfaceContainerLowest: Color,
    val surfaceContainerLow: Color,
    val surfaceContainer: Color,
    val surfaceContainerHigh: Color,
    val surfaceContainerHighest: Color,
    val surfaceVariant: Color,
    val onSurfaceVariant: Color,
    val outline: Color,
    val outlineVariant: Color,
    val card: Color,
    val activeCard: Color,
    val elevated: Color,
    val border: Color,
    val divider: Color,
    val secondaryText: Color,
    val mutedText: Color,
    val deepGreen: Color,
    val darkGreen: Color,
    val teal: Color,
    val success: Color,
    val onSuccess: Color,
    val successContainer: Color,
    val onSuccessContainer: Color,
    val info: Color,
    val onInfo: Color,
    val infoContainer: Color,
    val onInfoContainer: Color,
    val warning: Color,
    val onWarning: Color,
    val warningContainer: Color,
    val onWarningContainer: Color,
    val error: Color,
    val onError: Color,
    val errorContainer: Color,
    val onErrorContainer: Color,
    val neutral: Color,
)

internal val DarkHuginnColors = HuginnColors(
    primary = TpPrimary,
    onPrimary = OnPrimary,
    primaryContainer = TpDeepGreen,
    onPrimaryContainer = TpPrimaryHighlight,
    primaryHighlight = TpPrimaryHighlight,
    secondary = TpDarkGreen,
    onSecondary = TpDarkTextPrimary,
    background = TpDarkBackground,
    onBackground = TpDarkTextPrimary,
    surface = TpDarkSurface,
    onSurface = TpDarkTextPrimary,
    surfaceContainerLowest = TpDarkBackground,
    surfaceContainerLow = TpDarkCard,
    surfaceContainer = TpDarkCard,
    surfaceContainerHigh = TpDarkElevated,
    surfaceContainerHighest = TpDarkElevated,
    surfaceVariant = TpDarkCard,
    onSurfaceVariant = TpDarkTextSecondary,
    outline = TpDarkBorder,
    outlineVariant = TpDarkBorder,
    card = TpDarkCard,
    activeCard = Color(0xFF0D2D23),
    elevated = TpDarkElevated,
    border = TpDarkBorder,
    divider = TpDarkBorder,
    secondaryText = TpDarkTextSecondary,
    mutedText = TpDarkTextMuted,
    deepGreen = TpDeepGreen,
    darkGreen = TpDarkGreen,
    teal = TpTeal,
    success = TpSuccess,
    onSuccess = TpDarkBackground,
    successContainer = TpDeepGreen,
    onSuccessContainer = TpPrimaryHighlight,
    info = TpInfo,
    onInfo = TpDarkBackground,
    infoContainer = Color(0xFF0F3147),
    onInfoContainer = Color(0xFF9CC8EA),
    warning = TpWarning,
    onWarning = TpDarkBackground,
    warningContainer = Color(0xFF4A3412),
    onWarningContainer = Color(0xFFF0C98A),
    error = TpError,
    onError = TpDarkBackground,
    errorContainer = Color(0xFF4A1F22),
    onErrorContainer = Color(0xFFF0A9A9),
    neutral = TpNeutral,
)

internal val LightHuginnColors = HuginnColors(
    primary = TpPrimary,
    onPrimary = OnPrimary,
    primaryContainer = TpLightActiveCard,
    onPrimaryContainer = TpLightTextPrimary,
    primaryHighlight = TpPrimaryHighlight,
    secondary = TpDarkGreen,
    onSecondary = TpLightCard,
    background = TpLightBackground,
    onBackground = TpLightTextPrimary,
    surface = TpLightCard,
    onSurface = TpLightTextPrimary,
    surfaceContainerLowest = TpLightCard,
    surfaceContainerLow = TpLightSecondaryBackground,
    surfaceContainer = TpLightSecondaryBackground,
    surfaceContainerHigh = TpLightActiveCard,
    surfaceContainerHighest = TpLightDivider,
    surfaceVariant = TpLightSecondaryBackground,
    onSurfaceVariant = TpLightTextSecondary,
    outline = TpLightBorder,
    outlineVariant = TpLightDivider,
    card = TpLightCard,
    activeCard = TpLightActiveCard,
    elevated = TpLightCard,
    border = TpLightBorder,
    divider = TpLightDivider,
    secondaryText = TpLightTextSecondary,
    mutedText = TpLightTextMuted,
    deepGreen = TpDeepGreen,
    darkGreen = TpDarkGreen,
    teal = TpTeal,
    success = TpSuccess,
    onSuccess = TpLightCard,
    successContainer = TpLightActiveCard,
    onSuccessContainer = TpDeepGreen,
    info = TpInfo,
    onInfo = TpLightCard,
    infoContainer = Color(0xFFE3EEF7),
    onInfoContainer = Color(0xFF1C5E8F),
    warning = TpWarning,
    onWarning = TpLightCard,
    warningContainer = Color(0xFFFAEDD5),
    onWarningContainer = Color(0xFF7A5310),
    error = TpError,
    onError = TpLightCard,
    errorContainer = Color(0xFFFBE5E5),
    onErrorContainer = Color(0xFF8C2E2E),
    neutral = TpNeutral,
)

val LocalHuginnColors = staticCompositionLocalOf { LightHuginnColors }
