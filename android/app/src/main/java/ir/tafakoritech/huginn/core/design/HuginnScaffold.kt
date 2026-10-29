package ir.tafakoritech.huginn.core.design

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.QueryStats
import androidx.compose.material.icons.filled.WorkOutline
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import ir.tafakoritech.huginn.R
import ir.tafakoritech.huginn.core.theme.huginnColors
import androidx.compose.material3.IconButton
import androidx.compose.material.icons.automirrored.filled.ArrowBack

/** Bottom navigation destinations. */
enum class HuginnDestination(
    val route: String,
    @StringRes val labelRes: Int,
    val icon: ImageVector,
) {
    HOME("home", R.string.nav_home, Icons.Default.Home),
    LEADS("leads", R.string.nav_leads, Icons.Default.WorkOutline),
    FOLLOWUPS("followups", R.string.nav_followups, Icons.Default.Notifications),
    REPORTS("reports", R.string.nav_reports, Icons.Default.QueryStats),
    PROFILE("profile", R.string.nav_profile, Icons.Default.Person),
}

@Composable
fun HuginnScaffold(
    topBar: @Composable () -> Unit = {},
    bottomBar: @Composable () -> Unit = {},
    snackbarHostState: SnackbarHostState? = null,
    content: @Composable (androidx.compose.foundation.layout.PaddingValues) -> Unit,
) {
    val colors = huginnColors()
    Scaffold(
        containerColor = colors.background,
        topBar = topBar,
        bottomBar = bottomBar,
        // safeDrawing covers the status bar, navigation bar (gesture + 3-button)
        // and display cutouts; the caller applies the produced PaddingValues to
        // the NavHost so no screen ever draws under the system UI.
        contentWindowInsets = WindowInsets.safeDrawing,
        snackbarHost = { snackbarHostState?.let { SnackbarHost(it) } },
        content = content,
    )
}

@OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)
@Composable
fun HuginnTopBar(
    title: String,
    onBack: (() -> Unit)? = null,
    actions: @Composable () -> Unit = {},
) {
    val colors = huginnColors()
    TopAppBar(
        title = {
            Text(text = title, fontWeight = FontWeight.SemiBold)
        },
        navigationIcon = {
            if (onBack != null) {
                IconButton(onClick = onBack) {
                    Icon(
                        imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                        contentDescription = null,
                        tint = colors.onSurface,
                    )
                }
            }
        },
        actions = { actions() },
        colors = TopAppBarDefaults.topAppBarColors(
            containerColor = colors.background,
            titleContentColor = colors.onSurface,
        ),
        // Screen-level top bars live INSIDE the scaffolded NavHost, which already
        // applies the status-bar inset as top padding. Disabling the TopAppBar's
        // own window insets prevents a double status-bar inset on every screen.
        windowInsets = WindowInsets(0),
    )
}

@Composable
fun HuginnBottomNavigation(
    currentRoute: String,
    onNavigate: (HuginnDestination) -> Unit,
) {
    val colors = huginnColors()
    NavigationBar(
        containerColor = colors.surface,
        tonalElevation = 0.dp,
    ) {
        HuginnDestination.entries.forEach { destination ->
            val selected = currentRoute == destination.route
            NavigationBarItem(
                selected = selected,
                onClick = { onNavigate(destination) },
                icon = {
                    Icon(
                        imageVector = destination.icon,
                        contentDescription = null,
                    )
                },
                label = {
                    Text(text = androidx.compose.ui.res.stringResource(destination.labelRes))
                },
                colors = NavigationBarItemDefaults.colors(
                    selectedIconColor = colors.primary,
                    selectedTextColor = colors.primary,
                    unselectedIconColor = colors.mutedText,
                    unselectedTextColor = colors.mutedText,
                    indicatorColor = colors.activeCard,
                ),
            )
        }
    }
}
