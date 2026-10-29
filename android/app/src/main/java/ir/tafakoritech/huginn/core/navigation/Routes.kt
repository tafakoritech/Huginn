package ir.tafakoritech.huginn.core.navigation

/** Navigation route constants. String routes are used for reliable, simple navigation. */
object Routes {
    const val LOGIN = "login"

    // Bottom-tab destinations
    const val HOME = "home"
    const val LEADS = "leads"
    const val FOLLOWUPS = "followups"
    const val REPORTS = "reports"
    const val PROFILE = "profile"

    // Detail / secondary destinations
    const val LEAD_DETAIL = "lead/{leadId}"
    const val EDIT_LEAD = "edit/{leadId}"
    const val SETTINGS = "settings"
    const val MANDATORY_REPORT = "mandatory-report/{interactionId}"
    const val QUALIFY = "qualify/{leadId}"
    const val DISQUALIFY = "disqualify/{leadId}"
    const val WORK_QUEUE = "work-queue"

    fun leadDetail(leadId: String) = "lead/$leadId"
    fun editLead(leadId: String) = "edit/$leadId"
    fun mandatoryReport(interactionId: String) = "mandatory-report/$interactionId"
    fun qualify(leadId: String) = "qualify/$leadId"
    fun disqualify(leadId: String) = "disqualify/$leadId"

    val topLevelDestinations = setOf(HOME, LEADS, FOLLOWUPS, REPORTS, PROFILE)

    fun isTopLevel(route: String?): Boolean = route in topLevelDestinations
}
