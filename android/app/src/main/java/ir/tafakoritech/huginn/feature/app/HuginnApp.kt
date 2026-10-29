package ir.tafakoritech.huginn.feature.app

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import ir.tafakoritech.huginn.core.auth.SessionState
import ir.tafakoritech.huginn.core.design.HuginnBottomNavigation
import ir.tafakoritech.huginn.core.design.HuginnDestination
import ir.tafakoritech.huginn.core.design.HuginnScaffold
import ir.tafakoritech.huginn.core.navigation.Routes
import ir.tafakoritech.huginn.feature.auth.LoginScreen
import ir.tafakoritech.huginn.feature.followups.FollowUpsScreen
import ir.tafakoritech.huginn.feature.followups.FollowUpsViewModel
import ir.tafakoritech.huginn.feature.home.HomeScreen
import ir.tafakoritech.huginn.feature.home.HomeViewModel
import ir.tafakoritech.huginn.feature.lead_detail.EditLeadScreen
import ir.tafakoritech.huginn.feature.lead_detail.EditLeadViewModel
import ir.tafakoritech.huginn.feature.lead_detail.LeadDetailScreen
import ir.tafakoritech.huginn.feature.lead_detail.LeadDetailViewModel
import ir.tafakoritech.huginn.feature.leads.LeadsScreen
import ir.tafakoritech.huginn.feature.leads.LeadsViewModel
import ir.tafakoritech.huginn.feature.profile.ProfileScreen
import ir.tafakoritech.huginn.feature.profile.ProfileViewModel
import ir.tafakoritech.huginn.feature.qualification.DisqualifyScreen
import ir.tafakoritech.huginn.feature.qualification.DisqualifyViewModel
import ir.tafakoritech.huginn.feature.qualification.QualifyScreen
import ir.tafakoritech.huginn.feature.qualification.QualifyViewModel
import ir.tafakoritech.huginn.feature.reports.MandatoryReportOverlay
import ir.tafakoritech.huginn.feature.reports.MandatoryReportViewModel
import ir.tafakoritech.huginn.feature.reports.ReportsScreen
import ir.tafakoritech.huginn.feature.reports.ReportsViewModel
import ir.tafakoritech.huginn.feature.settings.SettingsScreen
import ir.tafakoritech.huginn.feature.settings.SettingsViewModel
import ir.tafakoritech.huginn.feature.splash.SplashScreen
import ir.tafakoritech.huginn.feature.workqueue.WorkQueueScreen
import ir.tafakoritech.huginn.feature.workqueue.WorkQueueViewModel
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.hilt.navigation.compose.hiltViewModel

/** Root composable: routes between splash / login / main app and blocks on reports. */
@Composable
fun HuginnApp(gate: AppGateViewModel) {
    val session by gate.sessionState.collectAsStateWithLifecycle()
    val pendingReport by gate.pendingReport.collectAsStateWithLifecycle()
    val loginBusy by gate.loginBusy.collectAsStateWithLifecycle()
    val loginError by gate.loginError.collectAsStateWithLifecycle()
    val checking by gate.checkingPendingReport.collectAsStateWithLifecycle()

    val lifecycleOwner = LocalLifecycleOwner.current
    androidx.compose.runtime.DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                gate.onAppResumed()
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    // Watchdog: if session resolution ever stalls, fall back to the login screen
    // instead of staying on the splash forever.
    androidx.compose.runtime.LaunchedEffect(session) {
        if (session is SessionState.Unknown) {
            kotlinx.coroutines.delay(2500)
            gate.ensureSessionResolved()
        }
    }

    when (session) {
        SessionState.Unknown -> SplashScreen()

        SessionState.Anonymous -> LoginScreen(
            busy = loginBusy,
            error = loginError,
            onLogin = gate::login,
        )

        is SessionState.Authenticated -> {
            val report = pendingReport
            if (report != null) {
                MandatoryReportOverlay(
                    pendingReport = report,
                    checking = checking,
                    onSubmitted = gate::onMandatoryReportDone,
                    onRecheck = gate::checkPendingReports,
                )
            } else {
                MainNavHost(
                    onLogout = gate::logout,
                    onOpenReport = gate::showReport,
                    deepLinkFlow = gate.deepLinkRoute,
                    onDeepLinkConsumed = gate::consumeDeepLink,
                )
            }
        }
    }
}

@Composable
private fun MainNavHost(
    onLogout: () -> Unit,
    onOpenReport: (String) -> Unit,
    deepLinkFlow: kotlinx.coroutines.flow.StateFlow<String?>,
    onDeepLinkConsumed: () -> Unit,
) {
    val navController = rememberNavController()
    val backStack by navController.currentBackStackEntryAsState()
    val currentRoute = backStack?.destination?.route

    // Handle incoming deep links (huginn://lead/{id} etc.)
    val pendingRoute by deepLinkFlow.collectAsStateWithLifecycle()
    androidx.compose.runtime.LaunchedEffect(pendingRoute) {
        val route = pendingRoute
        if (route != null) {
            navController.navigate(route) {
                launchSingleTop = true
            }
            onDeepLinkConsumed()
        }
    }

    HuginnScaffold(
        bottomBar = {
            if (currentRoute != null && Routes.isTopLevel(currentRoute)) {
                HuginnBottomNavigation(
                    currentRoute = currentRoute,
                    onNavigate = { destination: HuginnDestination ->
                        navController.navigate(destination.route) {
                            popUpTo(Routes.HOME) { saveState = true }
                            launchSingleTop = true
                            restoreState = true
                        }
                    },
                )
            }
        },
    ) { padding ->
        NavHost(
            navController = navController,
            startDestination = Routes.HOME,
            // The Scaffold's padding carries the system-bar insets (status bar
            // on top, bottom navigation bar / gesture insets at the bottom) and
            // the bottom-navigation height. Every screen inside the NavHost must
            // be laid out inside it, otherwise content draws edge-to-edge
            // underneath the system UI.
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            composable(Routes.HOME) {
                val vm: HomeViewModel = hiltViewModel()
                HomeScreen(
                    vm = vm,
                    onOpenLead = { navController.navigate(Routes.leadDetail(it)) },
                    onOpenWorkQueue = { navController.navigate(Routes.WORK_QUEUE) },
                    onOpenReport = { navController.navigate(Routes.mandatoryReport(it)) },
                )
            }
            composable(Routes.LEADS) {
                val vm: LeadsViewModel = hiltViewModel()
                LeadsScreen(
                    vm = vm,
                    onOpenLead = { navController.navigate(Routes.leadDetail(it)) },
                )
            }
            composable(Routes.FOLLOWUPS) {
                val vm: FollowUpsViewModel = hiltViewModel()
                FollowUpsScreen(
                    vm = vm,
                    onOpenLead = { navController.navigate(Routes.leadDetail(it)) },
                )
            }
            composable(Routes.REPORTS) {
                val vm: ReportsViewModel = hiltViewModel()
                ReportsScreen(vm = vm)
            }
            composable(Routes.PROFILE) {
                val vm: ProfileViewModel = hiltViewModel()
                ProfileScreen(
                    vm = vm,
                    onLogout = onLogout,
                    onSettings = { navController.navigate(Routes.SETTINGS) },
                )
            }
            composable(Routes.LEAD_DETAIL) { entry ->
                val vm: LeadDetailViewModel = hiltViewModel()
                val leadId: String = entry.arguments?.getString("leadId").orEmpty()
                LeadDetailScreen(
                    vm = vm,
                    onBack = { navController.popBackStack() },
                    onEdit = { navController.navigate(Routes.editLead(leadId)) },
                    onQualify = { leadId -> navController.navigate(Routes.qualify(leadId)) },
                    onDisqualify = { leadId -> navController.navigate(Routes.disqualify(leadId)) },
                )
            }
            composable(Routes.EDIT_LEAD) {
                val vm: EditLeadViewModel = hiltViewModel()
                EditLeadScreen(
                    vm = vm,
                    onBack = { navController.popBackStack() },
                    onSaved = { navController.popBackStack() },
                )
            }
            composable(Routes.SETTINGS) {
                val vm: SettingsViewModel = hiltViewModel()
                SettingsScreen(vm = vm, onBack = { navController.popBackStack() })
            }
            composable(Routes.QUALIFY) {
                val vm: QualifyViewModel = hiltViewModel()
                QualifyScreen(vm = vm, onDone = { navController.popBackStack() })
            }
            composable(Routes.DISQUALIFY) {
                val vm: DisqualifyViewModel = hiltViewModel()
                DisqualifyScreen(vm = vm, onDone = { navController.popBackStack() })
            }
            composable(Routes.WORK_QUEUE) {
                val vm: WorkQueueViewModel = hiltViewModel()
                WorkQueueScreen(
                    vm = vm,
                    onBack = { navController.popBackStack() },
                    onOpenLead = { navController.navigate(Routes.leadDetail(it)) },
                    onOpenReport = { navController.navigate(Routes.mandatoryReport(it)) },
                )
            }
        }
    }
}
