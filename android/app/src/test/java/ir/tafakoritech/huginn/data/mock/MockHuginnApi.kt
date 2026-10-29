package ir.tafakoritech.huginn.data.mock

import ir.tafakoritech.huginn.core.api.HuginnApi
import ir.tafakoritech.huginn.core.api.LlmHuginnApi
import ir.tafakoritech.huginn.data.model.Agent
import ir.tafakoritech.huginn.data.model.AgentMetrics
import ir.tafakoritech.huginn.data.model.AssignRequest
import ir.tafakoritech.huginn.data.model.AuthResponse
import ir.tafakoritech.huginn.data.model.Contact
import ir.tafakoritech.huginn.data.model.ContactActionRequest
import ir.tafakoritech.huginn.data.model.CallScenario
import ir.tafakoritech.huginn.data.model.CallScenarioRequest
import ir.tafakoritech.huginn.data.model.CallScenarioResponse
import ir.tafakoritech.huginn.data.model.ContactActionResponse
import ir.tafakoritech.huginn.data.model.ContactUpdateRequest
import ir.tafakoritech.huginn.data.model.CreateNoteRequest
import ir.tafakoritech.huginn.data.model.DeepLink
import ir.tafakoritech.huginn.data.model.DisqualifyRequest
import ir.tafakoritech.huginn.data.model.DisqualifyResponse
import ir.tafakoritech.huginn.data.model.EvidenceItemDto
import ir.tafakoritech.huginn.data.model.EvidenceResponse
import ir.tafakoritech.huginn.data.model.FollowUp
import ir.tafakoritech.huginn.data.model.FollowUpCreateRequest
import ir.tafakoritech.huginn.data.model.FollowUpRescheduleRequest
import ir.tafakoritech.huginn.data.model.FollowUpStatus
import ir.tafakoritech.huginn.data.model.FollowUpUpdateRequest
import ir.tafakoritech.huginn.data.model.FollowUpsResponse
import ir.tafakoritech.huginn.data.model.Interaction
import ir.tafakoritech.huginn.data.model.InteractionChannel
import ir.tafakoritech.huginn.data.model.Lead
import ir.tafakoritech.huginn.data.model.LeadDetailResponse
import ir.tafakoritech.huginn.data.model.LeadListResponse
import ir.tafakoritech.huginn.data.model.LeadPermissions
import ir.tafakoritech.huginn.data.model.LeadStatus
import ir.tafakoritech.huginn.data.model.LoginRequest
import ir.tafakoritech.huginn.data.model.Note
import ir.tafakoritech.huginn.data.model.NotesResponse
import ir.tafakoritech.huginn.data.model.PendingReport
import ir.tafakoritech.huginn.data.model.PendingReportsResponse
import ir.tafakoritech.huginn.data.model.QualifyRequest
import ir.tafakoritech.huginn.data.model.QualifyResponse
import ir.tafakoritech.huginn.data.model.ReportSubmitRequest
import ir.tafakoritech.huginn.data.model.ReportSubmitResponse
import ir.tafakoritech.huginn.data.model.ReportsResponse
import ir.tafakoritech.huginn.data.model.ScoreComponentDto
import ir.tafakoritech.huginn.data.model.ScoreResponse
import ir.tafakoritech.huginn.data.model.TimelineEvent
import ir.tafakoritech.huginn.data.model.TimelineResponse
import ir.tafakoritech.huginn.data.model.User
import ir.tafakoritech.huginn.data.model.WorkItem
import ir.tafakoritech.huginn.data.model.WorkQueueResponse
import ir.tafakoritech.huginn.data.model.NextActionItem
import ir.tafakoritech.huginn.data.model.NextActionsResponse
import ir.tafakoritech.huginn.data.model.NextActionPriority
import ir.tafakoritech.huginn.data.model.NextActionSource
import ir.tafakoritech.huginn.data.model.NextActionType
import ir.tafakoritech.huginn.data.model.SnoozeNextActionRequest
import ir.tafakoritech.huginn.data.model.CompleteNextActionRequest
import ir.tafakoritech.huginn.data.model.SubmittedReport
import kotlinx.coroutines.delay
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import java.time.Instant
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter

/**
 * TEST-ONLY in-memory mirror of the Huginn API. Lives in the test source set and
 * is never reachable from any runtime build. Its response shapes follow the
 * real backend contract (backend/internal/httpapi).
 *
 * It implements both [HuginnApi] and [LlmHuginnApi]: call-scenario generation
 * is served by the dedicated LLM endpoint in the real app, and the mock covers
 * the same method for both contracts.
 */
class MockHuginnApi : HuginnApi, LlmHuginnApi {

    private data class MockNext(
        val sourceType: String,
        val actionType: String,
        val priority: String,
        val score: Int,
        val dueAt: String?,
        val overdue: Boolean,
    )

    private data class MockLead(
        var lead: Lead,
        var contacts: List<Contact>,
        val timeline: MutableList<TimelineEvent>,
        val notes: MutableList<Note>,
        val followUps: MutableList<FollowUp>,
        val callScenarios: MutableList<CallScenario> = mutableListOf(),
        var status: String,
        var score: Int,
    )

    private val now: String get() = DateTimeFormatter.ISO_INSTANT.format(Instant.now().atZone(ZoneOffset.UTC))

    private val interactions = mutableListOf<Interaction>()

    /** Test observability: how many suggested-message generations were requested. */
    var contactActionCallCount: Int = 0

    /** Test hook: when true, contactAction fails like a backend error. */
    var failContactAction: Boolean = false
    var failFollowUps: Boolean = false

    /** Test hook: when true, myReports() fails like a backend error. */
    var failReports: Boolean = false

    /** Submitted contact reports backing GET /me/reports (mirrors the backend). */
    private val submittedReports = mutableListOf<SubmittedReport>()

    /** Test seed: make the backend report a submitted report for the current user. */
    fun seedSubmittedReport(report: SubmittedReport) {
        submittedReports.removeAll { it.id == report.id }
        submittedReports.add(report)
    }

    /** Test seed: create a pending interaction so a report can be submitted for it. */
    fun seedPendingInteraction(interactionId: String, leadId: String, channel: String = "CALL") {
        interactions.add(
            Interaction(
                id = interactionId,
                leadId = leadId,
                userId = currentUser.id,
                channel = channel,
                startedAt = now,
                actionStatus = "COMPLETED",
                reportStatus = "PENDING",
            ),
        )
    }

    private var currentUser: User = User(
        id = "u-agent-1",
        name = "علی محمدی",
        username = "agent",
        role = "sales_agent",
    )

    private fun lead(leadId: String, status: String, score: Int): MockLead = MockLead(
        lead = baseLead(leadId, score),
        contacts = contactsFor(leadId),
        timeline = mutableListOf(
            TimelineEvent(id = "t-$leadId-1", leadId = leadId, eventType = "lead.created", title = "ایجاد شد", createdAt = isoDaysAgo(12)),
            TimelineEvent(id = "t-$leadId-2", leadId = leadId, eventType = "lead.assigned", title = "به شما تخصیص داده شد", createdAt = isoDaysAgo(6)),
        ),
        notes = mutableListOf(
            Note(id = "n-$leadId-1", leadId = leadId, authorId = "u-agent-1", text = "در تماس تلفنی، مدیر مجموعه مشتری بالقوه است.", createdAt = isoDaysAgo(2)),
        ),
        followUps = mutableListOf(),
        status = status,
        score = score,
    )

    private fun baseLead(leadId: String, score: Int): Lead = when (leadId) {
        "lead-1" -> Lead(
            id = leadId, companyName = "عروسک‌سازی پارس", businessType = "تولیدکننده", industry = "صنایع اسباب‌بازی",
            province = "تهران", city = "تهران", status = LeadStatus.CONTACTING, score = score,
            source = "لیست تولیدکنندگان", ownerUserId = "u-agent-1", createdAt = isoDaysAgo(12), updatedAt = isoDaysAgo(1),
        )
        "lead-2" -> Lead(
            id = leadId, companyName = "صنایع چوب نوین", businessType = "تولیدکننده", industry = "صنایع چوب",
            province = "آذربایجان شرقی", city = "تبریز", status = LeadStatus.FOLLOW_UP, score = score,
            ownerUserId = "u-agent-1", createdAt = isoDaysAgo(20), updatedAt = isoDaysAgo(0),
        )
        "lead-3" -> Lead(
            id = leadId, companyName = "فروشگاه کالای خانگی نگین", businessType = "خرده‌فروشی", industry = "کالای خانگی",
            province = "خراسان رضوی", city = "مشهد", status = LeadStatus.READY, score = score,
            ownerUserId = null, createdAt = isoDaysAgo(5), updatedAt = isoDaysAgo(5),
        )
        "lead-4" -> Lead(
            id = leadId, companyName = "گروه صنعتی فولاد آریا", businessType = "صنعتی", industry = "فلزات",
            province = "اصفهان", city = "اصفهان", status = LeadStatus.ASSIGNED, score = score,
            ownerUserId = "u-agent-1", createdAt = isoDaysAgo(3), updatedAt = isoDaysAgo(1),
        )
        "lead-5" -> Lead(
            id = leadId, companyName = "پخش مواد غذایی البرز", businessType = "توزیع‌کننده", industry = "مواد غذایی",
            province = "البرز", city = "کرج", status = LeadStatus.CONTACTING, score = score,
            ownerUserId = "u-agent-1", createdAt = isoDaysAgo(8), updatedAt = isoDaysAgo(2),
        )
        "lead-6" -> Lead(
            id = leadId, companyName = "استارتاپ لجستیک هوشمند", businessType = "خدماتی", industry = "لجستیک",
            province = "فارس", city = "شیراز", status = LeadStatus.ASSIGNED, score = score,
            ownerUserId = "u-agent-1", createdAt = isoDaysAgo(2), updatedAt = isoDaysAgo(1),
        )
        "lead-7" -> Lead(
            id = leadId, companyName = "آموزشگاه زبان سفیر", businessType = "خدماتی", industry = "آموزش",
            province = "تهران", city = "تهران", status = LeadStatus.READY, score = score,
            ownerUserId = null, createdAt = isoDaysAgo(15), updatedAt = isoDaysAgo(15),
        )
        else -> Lead(
            id = leadId, companyName = "تولیدی پوشاک سپید", businessType = "تولیدکننده", industry = "پوشاک",
            province = "قم", city = "قم", status = LeadStatus.FOLLOW_UP, score = score,
            ownerUserId = "u-agent-1", createdAt = isoDaysAgo(10), updatedAt = isoDaysAgo(0),
        )
    }

    private fun contactsFor(leadId: String): List<Contact> = when (leadId) {
        "lead-1" -> listOf(
            Contact(
                id = "c-1", leadId = leadId, fullName = "محمد رضایی", jobTitle = "مدیر مجموعه",
                mobile = "0935 757 8808", mobileNormalized = phoneBase, whatsapp = phoneBase,
                telegram = "mohammad_rezaei", isPrimaryContact = true, isDecisionMaker = true,
            ),
            Contact(
                id = "c-2", leadId = leadId, fullName = "علی محمودی", jobTitle = "مدیر خرید",
                mobile = "09121112233", mobileNormalized = "+989121112233", whatsapp = "+989121112233",
                isDecisionMaker = false,
            ),
        )
        "lead-2" -> listOf(
            Contact(
                id = "c-4", leadId = leadId, fullName = "امیر کریمی", jobTitle = "مدیر فروش",
                mobile = "09154443322", mobileNormalized = "+989154443322", whatsapp = "+989154443322",
                telegram = "amir_karimi", isPrimaryContact = true, isDecisionMaker = true,
            ),
        )
        "lead-3" -> listOf(
            Contact(
                id = "c-5", leadId = leadId, fullName = "مریم احمدی", jobTitle = "مدیر خرید",
                mobile = "09123334455", mobileNormalized = "+989123334455", isPrimaryContact = true, isDecisionMaker = true,
            ),
        )
        "lead-4" -> listOf(
            Contact(
                id = "c-6", leadId = leadId, fullName = "رضا نادری", jobTitle = "مدیرعامل",
                mobile = "09123334455", mobileNormalized = "+989123334455", whatsapp = "+989123334455",
                telegram = "reza_naderi", bale = "reza_naderi", isPrimaryContact = true, isDecisionMaker = true,
            ),
        )
        "lead-5" -> listOf(
            Contact(
                id = "c-7", leadId = leadId, fullName = "سعید عباسی", jobTitle = "مدیر مجموعه",
                mobile = "09197776655", mobileNormalized = "+989197776655", isPrimaryContact = true, isDecisionMaker = true,
            ),
        )
        "lead-6" -> listOf(
            Contact(
                id = "c-8", leadId = leadId, fullName = "نگار شریفی", jobTitle = "مدیر عملیات",
                mobile = "09126665544", mobileNormalized = "+989126665544", whatsapp = "+989126665544",
                telegram = "negar_sharifi", isPrimaryContact = true, isDecisionMaker = true,
            ),
        )
        else -> listOf(
            Contact(
                id = "c-9", leadId = leadId, fullName = "زهرا کاظمی", jobTitle = "مدیر خرید",
                mobile = "09353336677", mobileNormalized = "+989353336677", isPrimaryContact = true, isDecisionMaker = true,
            ),
        )
    }

    private val phoneBase = "+989357578808"

    private val mockLeads = mutableListOf(
        lead("lead-1", LeadStatus.CONTACTING, 78),
        lead("lead-2", LeadStatus.FOLLOW_UP, 82),
        lead("lead-3", LeadStatus.READY, 45),
        lead("lead-4", LeadStatus.ASSIGNED, 91),
        lead("lead-5", LeadStatus.CONTACTING, 63),
        lead("lead-6", LeadStatus.ASSIGNED, 88),
        lead("lead-7", LeadStatus.READY, 55),
        lead("lead-8", LeadStatus.FOLLOW_UP, 70),
    )

    init {
        val todayUtc = java.time.LocalDate.now(ZoneOffset.UTC)
        mockLeads.first { it.lead.id == "lead-1" }.followUps +=
            FollowUp(id = "fu-1", leadId = "lead-1", dueAt = todayUtc.atTime(14, 30).atZone(ZoneOffset.UTC).toInstant().toString(), type = "FOLLOW_UP", description = "پیگیری پیشنهاد قیمت", status = FollowUpStatus.PENDING, createdAt = today, companyName = "شرکت نمونه ۱")
        mockLeads.first { it.lead.id == "lead-2" }.followUps +=
            FollowUp(id = "fu-2", leadId = "lead-2", dueAt = todayUtc.plusDays(1).atTime(12, 0).atZone(ZoneOffset.UTC).toInstant().toString(), type = "FOLLOW_UP", description = "ارسال کاتالوگ", status = FollowUpStatus.PENDING, createdAt = today, companyName = "شرکت نمونه ۲")
        mockLeads.first { it.lead.id == "lead-5" }.followUps +=
            FollowUp(id = "fu-3", leadId = "lead-5", dueAt = twoDaysAgo, type = "FOLLOW_UP", description = "پیگیری جلسه حضوری", status = FollowUpStatus.PENDING, isOverdue = true, createdAt = threeDaysAgo, companyName = "شرکت نمونه ۵")
    }

    private val today: String get() = isoDaysAgo(0)
    private val twoDaysAgo: String get() = isoDaysAgo(2)
    private val threeDaysAgo: String get() = isoDaysAgo(3)

    // ---- helpers ----
    private fun mock(id: String): MockLead = mockLeads.first { it.lead.id == id }

    private fun ownedLeads(): List<MockLead> = mockLeads.filter { it.lead.ownerUserId == currentUser.id }

    private suspend fun simulate() = delay(1)

    private fun isoAt(hour: Int, minute: Int): String {
        val zoned = java.time.ZonedDateTime.now(java.time.ZoneId.systemDefault())
            .withHour(hour).withMinute(minute).withSecond(0).withNano(0)
        return DateTimeFormatter.ISO_INSTANT.format(zoned.toInstant())
    }

    private fun isoDaysAgo(daysAgo: Int): String =
        DateTimeFormatter.ISO_INSTANT.format(Instant.now().minusSeconds(daysAgo * 86400L).atOffset(ZoneOffset.UTC))

    // ---- Auth ----
    override suspend fun login(request: LoginRequest): AuthResponse {
        simulate()
        val role = when (request.login) {
            "manager" -> "sales_development_manager"
            "viewer" -> "viewer"
            else -> "sales_agent"
        }
        val name = when (role) {
            "sales_development_manager" -> "مدیر فروش نمونه"
            "viewer" -> "کاربر بیننده"
            else -> "علی محمدی"
        }
        val userId = when (role) {
            "sales_development_manager" -> "u-manager-1"
            "viewer" -> "u-viewer-1"
            else -> "u-agent-1"
        }
        currentUser = User(id = userId, name = name, username = request.login, role = role)
        return AuthResponse(
            accessToken = "mock.access.${request.login}",
            refreshToken = "mock.refresh.${request.login}",
            expiresIn = 3600,
            user = currentUser,
        )
    }

    override suspend fun logout(refreshToken: String) {
        simulate()
    }

    // ---- Me ----
    override suspend fun me(): User {
        simulate()
        return currentUser
    }

    override suspend fun dashboardMe(): AgentMetrics {
        simulate()
        val openFollowUps = ownedLeads().flatMap { it.followUps }.filter { it.status == FollowUpStatus.PENDING }
        return AgentMetrics(
            assignedLeads = ownedLeads().size,
            callsToday = interactions.count { it.channel == InteractionChannel.CALL && it.startedAt.startsWith(todayPrefix()) },
            pendingReports = interactions.count { it.reportStatus == "PENDING" },
            followUps = openFollowUps.size,
            qualifiedLeads = mockLeads.count { it.status == LeadStatus.QUALIFIED },
        )
    }

    private fun todayPrefix(): String = java.time.LocalDate.now().toString()

    /** UTC date of a due_at instant (robust across timezone boundaries). */
    private fun dueUtcDate(iso: String): java.time.LocalDate =
        runCatching { Instant.parse(iso).atZone(ZoneOffset.UTC).toLocalDate() }
            .getOrElse { java.time.LocalDate.now(ZoneOffset.UTC).plusDays(100) }

    override suspend fun workQueue(): WorkQueueResponse {
        simulate()
        val items = ownedLeads()
            .filter { it.status !in listOf(LeadStatus.QUALIFIED, LeadStatus.DISQUALIFIED, LeadStatus.EXPORTED_TO_MUNINN, LeadStatus.ARCHIVED) }
            .map { ml ->
                val overdueFu = ml.followUps.firstOrNull { it.isOverdue && it.status == FollowUpStatus.PENDING }
                val todayFu = ml.followUps.firstOrNull { it.status == FollowUpStatus.PENDING && !it.isOverdue && dueUtcDate(it.dueAt) == java.time.LocalDate.now(ZoneOffset.UTC) }
                val priority = when {
                    overdueFu != null -> 10
                    todayFu != null -> 20
                    ml.status == LeadStatus.ASSIGNED -> 30
                    ml.status == LeadStatus.CONTACTING -> 40
                    ml.score >= 70 -> 50
                    else -> 60
                }
                val reason = when {
                    overdueFu != null -> "OVERDUE_FOLLOWUP"
                    todayFu != null -> "TODAY_FOLLOWUP"
                    ml.status == LeadStatus.ASSIGNED -> "NEW_ASSIGNED"
                    ml.status == LeadStatus.CONTACTING -> "NEEDS_ACTION"
                    ml.score >= 70 -> "HIGH_SCORE"
                    else -> "OTHER"
                }
                WorkItem(
                    leadId = ml.lead.id,
                    companyName = ml.lead.companyName,
                    priority = priority,
                    reason = reason,
                    status = ml.status,
                    score = ml.score,
                    dueAt = overdueFu?.dueAt ?: todayFu?.dueAt,
                )
            }
            .sortedWith(compareBy({ it.priority }, { -it.score }))
        return WorkQueueResponse(items)
    }

    override suspend fun nextActions(limit: Int?): NextActionsResponse {
        simulate()
        val items = ownedLeads()
            .filter { it.status !in listOf(LeadStatus.QUALIFIED, LeadStatus.DISQUALIFIED, LeadStatus.EXPORTED_TO_MUNINN, LeadStatus.ARCHIVED) }
            .map { ml ->
                val overdueFu = ml.followUps.firstOrNull { it.isOverdue && it.status == FollowUpStatus.PENDING }
                val todayFu = ml.followUps.firstOrNull { it.status == FollowUpStatus.PENDING && !it.isOverdue && dueUtcDate(it.dueAt) == java.time.LocalDate.now(ZoneOffset.UTC) }
                val (src, actionType, priority, score, due, overdue) = when {
                    overdueFu != null -> MockNext(NextActionSource.FOLLOW_UP, NextActionType.CALL, NextActionPriority.URGENT, 92, overdueFu.dueAt, true)
                    todayFu != null -> MockNext(NextActionSource.FOLLOW_UP, NextActionType.CALL, NextActionPriority.URGENT, 85, todayFu.dueAt, false)
                    ml.status == LeadStatus.ASSIGNED -> MockNext(NextActionSource.FIRST_CONTACT, NextActionType.CALL, NextActionPriority.IMPORTANT, 70, null, false)
                    ml.status == LeadStatus.CONTACTING -> MockNext(NextActionSource.STALE_LEAD, NextActionType.FOLLOW_UP, NextActionPriority.IMPORTANT, 60, null, false)
                    ml.score >= 70 -> MockNext(NextActionSource.STALE_LEAD, NextActionType.FOLLOW_UP, NextActionPriority.NORMAL, 45, null, false)
                    else -> MockNext(NextActionSource.STALE_LEAD, NextActionType.FOLLOW_UP, NextActionPriority.NORMAL, 30, null, false)
                }
                val cta = when (actionType) {
                    NextActionType.CALL -> "تماس"
                    NextActionType.REPORT -> "ثبت گزارش"
                    NextActionType.MESSAGE -> "پیام"
                    NextActionType.EMAIL -> "ایمیل"
                    else -> "پیگیری"
                }
                NextActionItem(
                    id = "$src:${ml.lead.id}",
                    sourceType = src,
                    sourceId = ml.lead.id,
                    leadId = ml.lead.id,
                    leadName = ml.lead.companyName,
                    companyName = ml.lead.companyName,
                    actionType = actionType,
                    title = if (actionType == NextActionType.CALL) "تماس با ${ml.lead.companyName}" else "پیگیری ${ml.lead.companyName}",
                    reason = ml.lead.companyName,
                    priority = priority,
                    priorityScore = score,
                    dueAt = due,
                    overdue = overdue,
                    suggestedAction = actionType,
                    ctaLabel = cta,
                    explainability = if (overdue) listOf("از زمان پیگیری گذشته است", "سرنخ گرم است") else listOf("سرنخ گرم است"),
                )
            }
            .sortedByDescending { it.priorityScore }
        val limited = if (limit != null && limit > 0) items.take(limit) else items
        return NextActionsResponse(
            items = limited,
            totalActionable = items.size,
            urgentCount = items.count { it.priority == NextActionPriority.URGENT },
            overdueCount = items.count { it.overdue },
        )
    }

    override suspend fun snoozeNextAction(id: String, request: SnoozeNextActionRequest) {
        simulate()
        val sourceId = id.substringAfter(':')
        rescheduleFollowUp(sourceId, FollowUpRescheduleRequest(request.until))
    }

    override suspend fun completeNextAction(id: String, request: CompleteNextActionRequest) {
        simulate()
        val sourceId = id.substringAfter(':')
        completeFollowUp(sourceId)
    }

    override suspend fun pendingReports(): PendingReportsResponse {
        simulate()
        val pending = interactions.filter { it.reportStatus == "PENDING" }.firstOrNull()
        return if (pending == null) {
            PendingReportsResponse()
        } else {
            PendingReportsResponse(
                requiresReport = true,
                interactionId = pending.id,
                leadId = pending.leadId,
                channel = pending.channel,
                createdAt = pending.startedAt,
            )
        }
    }

    override suspend fun myReports(limit: Int?, offset: Int?): ReportsResponse {
        simulate()
        if (failReports) throw IllegalStateException("backend unavailable")
        val all = submittedReports
            .filter { it.submittedBy == null || it.submittedBy == currentUser.id }
            .sortedByDescending { it.createdAt }
        val start = (offset ?: 0).coerceAtLeast(0)
        val paged = if (limit != null && limit > 0) all.drop(start).take(limit) else all.drop(start)
        return ReportsResponse(items = paged, total = all.size, count = paged.size)
    }

    override suspend fun followUps(
        view: String?, status: String?, assignedUser: String?, leadId: String?,
        from: String?, to: String?, overdue: Boolean?, limit: Int?, offset: Int?,
    ): FollowUpsResponse {
        simulate()
        if (failFollowUps) throw IllegalStateException("backend unavailable")
        var all = mockLeads.flatMap { ml -> ml.followUps.map { fu -> fu.copy() } }
        if (leadId != null) all = all.filter { it.leadId == leadId }
        val todayUtc = java.time.LocalDate.now(ZoneOffset.UTC)
        val filtered = when (view) {
            "today" -> all.filter { it.status == FollowUpStatus.PENDING && !it.isOverdue && dueUtcDate(it.dueAt) == todayUtc }
            "overdue" -> all.filter { it.status == FollowUpStatus.PENDING && it.isOverdue }
            "upcoming" -> all.filter { it.status == FollowUpStatus.PENDING && !it.isOverdue && dueUtcDate(it.dueAt).isAfter(todayUtc) }
            "completed" -> all.filter { it.status == FollowUpStatus.COMPLETED }
            "all" -> all
            else -> all
        }.sortedBy { it.dueAt }
        return FollowUpsResponse(items = filtered, count = filtered.size, total = filtered.size)
    }

    override suspend fun createFollowUp(request: FollowUpCreateRequest): FollowUp {
        simulate()
        val ml = mock(request.leadId)
        val fu = FollowUp(
            id = "fu-new",
            leadId = request.leadId,
            dueAt = request.dueAt,
            type = request.type,
            title = request.title,
            description = request.description,
            priority = request.priority,
            status = FollowUpStatus.PENDING,
            companyName = ml.lead.companyName,
            createdAt = today,
        )
        ml.followUps.add(fu)
        return fu
    }

    override suspend fun followUp(id: String): FollowUp =
        mockLeads.flatMap { it.followUps }.first { it.id == id }

    override suspend fun updateFollowUp(id: String, request: FollowUpUpdateRequest): FollowUp {
        val ml = mockLeads.first { ml -> ml.followUps.any { it.id == id } }
        val idx = ml.followUps.indexOfFirst { it.id == id }
        val old = ml.followUps[idx]
        val updated = old.copy(
            title = request.title ?: old.title,
            description = request.description ?: old.description,
            channel = request.channel ?: old.channel,
            type = request.type ?: old.type,
            priority = request.priority ?: old.priority,
            dueAt = request.dueAt ?: old.dueAt,
        )
        ml.followUps[idx] = updated
        return updated
    }

    override suspend fun reopenFollowUp(id: String): FollowUp {
        val ml = mockLeads.first { ml -> ml.followUps.any { it.id == id } }
        val idx = ml.followUps.indexOfFirst { it.id == id }
        val updated = ml.followUps[idx].copy(status = FollowUpStatus.PENDING, completedAt = null, isOverdue = false)
        ml.followUps[idx] = updated
        return updated
    }

    override suspend fun rescheduleFollowUp(id: String, request: FollowUpRescheduleRequest): FollowUp {
        val ml = mockLeads.first { ml -> ml.followUps.any { it.id == id } }
        val idx = ml.followUps.indexOfFirst { it.id == id }
        val updated = ml.followUps[idx].copy(dueAt = request.dueAt, status = FollowUpStatus.PENDING, completedAt = null, isOverdue = false)
        ml.followUps[idx] = updated
        return updated
    }

    override suspend fun cancelFollowUp(id: String): FollowUp {
        val ml = mockLeads.first { ml -> ml.followUps.any { it.id == id } }
        val idx = ml.followUps.indexOfFirst { it.id == id }
        val updated = ml.followUps[idx].copy(status = FollowUpStatus.CANCELLED, isOverdue = false)
        ml.followUps[idx] = updated
        return updated
    }

    // ---- Leads ----
    override suspend fun leads(
        search: String?, status: String?, scope: String?, minScore: Int?, maxScore: Int?,
        city: String?, industry: String?, businessType: String?, tag: String?, qualified: Boolean?,
        page: Int, pageSize: Int, sort: String?, order: String?,
    ): LeadListResponse {
        simulate()
        var result = mockLeads.map { it.lead }
        if (search != null && search.isNotBlank()) {
            val q = search.trim()
            result = result.filter {
                it.companyName.contains(q, ignoreCase = true) ||
                    it.city?.contains(q, ignoreCase = true) == true ||
                    contactsFor(it.id).any { c ->
                        ir.tafakoritech.huginn.core.util.PhoneFormatter.matches(q, c.mobileNormalized) ||
                            ir.tafakoritech.huginn.core.util.PhoneFormatter.matches(q, c.phoneNormalized)
                    }
            }
        }
        if (scope == "mine") result = result.filter { it.ownerUserId == currentUser.id }
        if (scope == "team") result = result.filter { it.ownerUserId != null }
        status?.let { s -> result = result.filter { it.status == s } }
        minScore?.let { s -> result = result.filter { it.score >= s } }
        maxScore?.let { s -> result = result.filter { it.score <= s } }
        city?.let { c -> result = result.filter { it.city == c } }
        industry?.let { i -> result = result.filter { it.industry == i } }
        businessType?.let { b -> result = result.filter { it.businessType == b } }

        val sorted = when (sort) {
            "score" -> if (order == "desc") result.sortedByDescending { it.score } else result.sortedBy { it.score }
            "created_at" -> if (order == "desc") result.sortedByDescending { it.createdAt } else result.sortedBy { it.createdAt }
            else -> result
        }

        val start = ((page - 1) * pageSize).coerceAtLeast(0)
        val paged = sorted.drop(start).take(pageSize)
        return LeadListResponse(
            leads = paged.map { lead ->
                mock(lead.id).lead.let { it.copy(ownerName = ownerNameFor(it.ownerUserId)) }
            },
            total = sorted.size,
            page = page,
            size = pageSize,
        )
    }

    override suspend fun lead(id: String): LeadDetailResponse {
        simulate()
        val ml = mock(id)
        return LeadDetailResponse(
            lead = ml.lead.copy(
                status = ml.status,
                score = ml.score,
                ownerName = ownerNameFor(ml.lead.ownerUserId),
            ),
            contacts = ml.contacts,
            permissions = permissionsFor(ml),
            workflow = ir.tafakoritech.huginn.data.model.WorkflowDto(mandatoryReportPending = hasPendingReportFor(id)),
            recentEvents = ml.timeline,
        )
    }

    override suspend fun updateLead(id: String, body: JsonObject): LeadDetailResponse {
        simulate()
        val ml = mock(id)
        ml.lead = patchLead(ml.lead, body)
        ml.status = ml.lead.status
        return LeadDetailResponse(
            lead = ml.lead.copy(status = ml.status, ownerName = ownerNameFor(ml.lead.ownerUserId)),
            contacts = ml.contacts,
            permissions = permissionsFor(ml),
            workflow = ir.tafakoritech.huginn.data.model.WorkflowDto(mandatoryReportPending = hasPendingReportFor(id)),
            recentEvents = ml.timeline,
        )
    }

    override suspend fun updateContact(leadId: String, contactId: String, request: ContactUpdateRequest): Contact {
        simulate()
        val ml = mock(leadId)
        val idx = ml.contacts.indexOfFirst { it.id == contactId }
        if (idx < 0) error("contact not found")
        val c = ml.contacts[idx]
        val updated = c.copy(
            firstName = request.firstName ?: c.firstName,
            lastName = request.lastName ?: c.lastName,
            fullName = request.fullName ?: c.fullName,
            jobTitle = request.jobTitle ?: c.jobTitle,
            department = request.department ?: c.department,
            mobile = request.mobile ?: c.mobile,
            phone = request.phone ?: c.phone,
            email = request.email ?: c.email,
            whatsapp = request.whatsapp ?: c.whatsapp,
            telegram = request.telegram ?: c.telegram,
            bale = request.bale ?: c.bale,
            eitaa = request.eitaa ?: c.eitaa,
            rubika = request.rubika ?: c.rubika,
            isDecisionMaker = request.isDecisionMaker ?: c.isDecisionMaker,
            isPrimaryContact = request.isPrimaryContact ?: c.isPrimaryContact,
        )
        ml.contacts = ml.contacts.mapIndexed { i, existing -> if (i == idx) updated else existing }
        return updated
    }

    override suspend fun assignLead(id: String, request: AssignRequest) {
        simulate()
        val ml = mock(id)
        val status = when (ml.status) {
            LeadStatus.NEW, LeadStatus.READY, LeadStatus.DISQUALIFIED -> LeadStatus.ASSIGNED
            else -> ml.status
        }
        ml.lead = ml.lead.copy(ownerUserId = request.assigneeId, status = status)
        ml.status = status
    }

    override suspend fun agents(): List<Agent> {
        simulate()
        return listOf(
            Agent(userId = "u-agent-1", name = "علی محمدی", assignedLeads = 3),
            Agent(userId = "u-agent-2", name = "مریم رضایی", assignedLeads = 2),
        )
    }

    private fun ownerNameFor(userId: String?): String? = when (userId) {
        "u-agent-1" -> "علی محمدی"
        "u-agent-2" -> "مریم رضایی"
        "u-manager-1" -> "مدیر فروش نمونه"
        else -> null
    }

    private fun patchLead(l: Lead, body: JsonObject): Lead {
        fun str(key: String): String? {
            val el = body[key] ?: return null
            return if (el is JsonPrimitive && el.isString) el.content else null
        }
        return l.copy(
            companyName = str("company_name") ?: l.companyName,
            displayName = if (body.containsKey("display_name")) str("display_name") else l.displayName,
            businessType = if (body.containsKey("business_type")) str("business_type") else l.businessType,
            industry = if (body.containsKey("industry")) str("industry") else l.industry,
            province = if (body.containsKey("province")) str("province") else l.province,
            city = if (body.containsKey("city")) str("city") else l.city,
            address = if (body.containsKey("address")) str("address") else l.address,
            website = if (body.containsKey("website")) str("website") else l.website,
            instagram = if (body.containsKey("instagram")) str("instagram") else l.instagram,
            source = if (body.containsKey("source")) str("source") else l.source,
            description = if (body.containsKey("description")) str("description") else l.description,
            status = if (body.containsKey("status")) str("status") ?: l.status else l.status,
        )
    }

    private fun permissionsFor(ml: MockLead): LeadPermissions {
        val isOwner = ml.lead.ownerUserId == currentUser.id
        val role = currentUser.role
        val canEdit = role != "viewer"
        val qualified = ml.status == LeadStatus.QUALIFIED
        return LeadPermissions(
            canCall = isOwner && ml.status != LeadStatus.DISQUALIFIED,
            canMessage = isOwner && ml.status != LeadStatus.DISQUALIFIED,
            canEdit = canEdit,
            canAssign = role == "sales_development_manager" || role == "super_admin",
            canQualify = isOwner && !qualified && ml.score >= 70,
            canExportToMuninn = qualified && role != "sales_agent",
            canDisqualify = isOwner && ml.status !in listOf(LeadStatus.QUALIFIED, LeadStatus.DISQUALIFIED, LeadStatus.EXPORTED_TO_MUNINN),
        )
    }

    private fun hasPendingReportFor(leadId: String): Boolean =
        interactions.any { it.leadId == leadId && it.reportStatus == "PENDING" }

    override suspend fun timeline(id: String): TimelineResponse {
        simulate()
        return TimelineResponse(mock(id).timeline)
    }

    override suspend fun notes(id: String): NotesResponse {
        simulate()
        return NotesResponse(mock(id).notes)
    }

    override suspend fun createNote(id: String, request: CreateNoteRequest): Note {
        simulate()
        val note = Note(
            id = "n-${id}-${System.currentTimeMillis()}",
            leadId = id,
            authorId = currentUser.id,
            text = request.text,
            createdAt = now,
        )
        mock(id).notes.add(0, note)
        return note
    }

    override suspend fun leadEvidence(id: String): EvidenceResponse {
        simulate()
        return EvidenceResponse(
            items = listOf(
                EvidenceItemDto(fieldName = "business_type", value = "تولیدکننده", confidence = 94, source = "لیست تولیدکنندگان", excerpt = "این مجموعه به صورت کارگاهی، محصولات عروسکی را تولید می‌کند.", page = 12, row = 4),
                EvidenceItemDto(fieldName = "city", value = "تهران", confidence = 98, source = "لیست تولیدکنندگان", excerpt = "آدرس: تهران، خیابان انقلاب، پلاک ۴۰", page = 2, row = 9),
            ),
        )
    }

    override suspend fun leadScore(id: String): ScoreResponse {
        simulate()
        val ml = mock(id)
        return ScoreResponse(
            total = ml.score,
            components = listOf(
                ScoreComponentDto(key = "business_fit", name = "business_fit", value = 22, max = 25),
                ScoreComponentDto(key = "contactability", name = "contactability", value = 18, max = 20),
                ScoreComponentDto(key = "scale", name = "scale", value = 14, max = 20),
                ScoreComponentDto(key = "engagement", name = "engagement", value = 10, max = 15),
                ScoreComponentDto(key = "authority", name = "authority", value = 12, max = 15),
                ScoreComponentDto(key = "need", name = "need", value = 8, max = 10),
                ScoreComponentDto(key = "timing", name = "timing", value = ml.score - 84, max = 10),
            ),
        )
    }

    override suspend fun leadInteractions(id: String): List<Interaction> {
        simulate()
        return interactions.filter { it.leadId == id }.sortedByDescending { it.startedAt }
    }

    override suspend fun leadFollowUps(id: String): List<FollowUp> {
        simulate()
        return mock(id).followUps
    }

    // ---- Communication ----
    override suspend fun contactAction(leadId: String, request: ContactActionRequest): ContactActionResponse {
        simulate()
        contactActionCallCount++
        if (failContactAction) throw IllegalStateException("backend unavailable")
        val ml = mock(leadId)
        val contact = ml.contacts.firstOrNull { it.id == request.contactId }
            ?: ml.contacts.firstOrNull { it.isPrimaryContact }
            ?: error("no contact")
        val channel = request.channel
        val interactionId = "int-${System.currentTimeMillis()}"
        val interaction = Interaction(
            id = interactionId,
            leadId = leadId,
            contactId = contact.id,
            userId = currentUser.id,
            channel = channel,
            objective = request.objective ?: "FIRST_CONTACT",
            startedAt = now,
            actionStatus = "CREATED",
            reportStatus = "PENDING",
            contactName = contact.displayName,
        )
        interactions.add(interaction)
        val deepLink = deepLinkFor(channel, contact)
        return ContactActionResponse(
            interactionId = interactionId,
            contactId = contact.id,
            channel = channel,
            phone = contact.primaryNumber,
            deepLink = deepLink,
            suggestedMessage = if (channel == InteractionChannel.CALL) null else suggestedTextFor(channel, contact, ml.lead),
        )
    }

    var callScenarioCallCount: Int = 0

    /** Test hook: when true, callScenario fails like a backend error. */
    var failCallScenario: Boolean = false

    override suspend fun callScenario(leadId: String, request: CallScenarioRequest): CallScenarioResponse {
        simulate()
        callScenarioCallCount++
        if (failCallScenario) throw IllegalStateException("backend unavailable")
        val ml = mock(leadId)
        val contact = ml.contacts.firstOrNull { it.id == request.contactId }
            ?: ml.contacts.firstOrNull { it.isPrimaryContact }
        val sc = CallScenario(
            id = "sc-${java.util.UUID.randomUUID()}",
            leadId = leadId,
            contactId = contact?.id,
            userId = currentUser.id,
            openingSentence = "سلام، وقت بخیر، ${contact?.displayName?.ifBlank { null } ?: ml.lead.companyName}؟",
            introduction = "اینجا " + currentUser.name + " هستم؛ بابت بررسی فرصت همکاری با " + ml.lead.companyName + " تماس گرفتم.",
            reasonForCall = "مجموعه شما را برای همکاری تجاری شناسایی کرده‌ایم و می‌خواستیم بیشتر آشنا شویم.",
            valueProposition = "می‌توانیم درباره راه‌هایی که برای هر دو طرف ارزش ایجاد می‌کند گفتگو کنیم.",
            discoveryQuestions = listOf(
                "مجموعه شما در حال حاضر روی چه حوزه‌ای تمرکز دارد؟",
                "بزرگ‌ترین چالش فعلی شما در تأمین یا فروش چیست؟",
            ),
            likelyObjections = listOf("الان وقت ندارم.", "در حال حاضر نیازی نداریم."),
            objectionResponses = listOf(
                "حتماً، چه زمانی برای گفتگو مناسب است؟",
                "اشکالی ندارد؛ اگر بعداً نیاز شد در خدمتیم.",
            ),
            recommendedNextStep = "پیشنهاد می‌کنم اطلاعات تکمیلی را ارسال و در روزهای آینده دوباره تماس بگیریم.",
            closingLine = "ممنون از وقتی که گذاشتید. روز خوبی داشته باشید.",
            isGenerated = true,
            createdAt = now,
            updatedAt = now,
        )
        ml.callScenarios.add(sc)
        return CallScenarioResponse(
            leadId = leadId,
            scenario = sc,
        )
    }

    override suspend fun getCallScenarios(leadId: String): List<CallScenario> {
        simulate()
        val ml = mock(leadId)
        return ml.callScenarios.reversed()
    }

    override suspend fun createCallScenario(leadId: String, request: CallScenario): CallScenario {
        simulate()
        val ml = mock(leadId)
        val sc = request.copy(
            id = "sc-${java.util.UUID.randomUUID()}",
            leadId = leadId,
            userId = currentUser.id,
            isGenerated = false,
            createdAt = now,
            updatedAt = now,
        )
        ml.callScenarios.add(sc)
        return sc
    }

    override suspend fun getCallScenario(leadId: String, scenarioId: String): CallScenario {
        simulate()
        val ml = mock(leadId)
        return ml.callScenarios.firstOrNull { it.id == scenarioId }
            ?: throw IllegalStateException("scenario not found")
    }

    override suspend fun updateCallScenario(
        leadId: String,
        scenarioId: String,
        request: CallScenario,
    ): CallScenario {
        simulate()
        val ml = mock(leadId)
        val idx = ml.callScenarios.indexOfFirst { it.id == scenarioId }
        val updated = request.copy(
            id = scenarioId,
            leadId = leadId,
            updatedAt = now,
        )
        if (idx >= 0) {
            ml.callScenarios[idx] = updated
        } else {
            ml.callScenarios.add(updated)
        }
        return updated
    }

    override suspend fun deleteCallScenario(
        leadId: String,
        scenarioId: String,
    ): retrofit2.Response<Unit> {
        simulate()
        val ml = mock(leadId)
        ml.callScenarios.removeAll { it.id == scenarioId }
        return retrofit2.Response.success(Unit)
    }

    private fun deepLinkFor(channel: String, contact: Contact): DeepLink? {
        val number = contact.mobileNormalized ?: contact.mobile
        return when (channel) {
            InteractionChannel.CALL -> DeepLink(InteractionChannel.CALL, "tel", "tel:${number.orEmpty()}")
            InteractionChannel.SMS -> DeepLink(InteractionChannel.SMS, "sms", contact.primaryNumber.orEmpty())
            InteractionChannel.WHATSAPP -> {
                val wa = contact.whatsapp ?: contact.mobileNormalized
                DeepLink(InteractionChannel.WHATSAPP, "wa.me", wa?.let { DeepLink.waMe(it) }.orEmpty())
            }
            InteractionChannel.TELEGRAM ->
                contact.telegram?.let { DeepLink(InteractionChannel.TELEGRAM, "username", "https://t.me/$it", username = it) }
                    ?: DeepLink(InteractionChannel.TELEGRAM, "CONTACT_SAVE_REQUIRED", "")
            InteractionChannel.BALE ->
                contact.bale?.let { DeepLink(InteractionChannel.BALE, "username", "https://ble.ir/$it", username = it) }
                    ?: DeepLink(InteractionChannel.BALE, "CONTACT_SAVE_REQUIRED", "")
            InteractionChannel.EITAA ->
                contact.eitaa?.let { DeepLink(InteractionChannel.EITAA, "username", "https://eitaa.com/$it", username = it) }
                    ?: DeepLink(InteractionChannel.EITAA, "CONTACT_SAVE_REQUIRED", "")
            InteractionChannel.RUBIKA ->
                contact.rubika?.let { DeepLink(InteractionChannel.RUBIKA, "username", "https://rubika.ir/$it", username = it) }
                    ?: DeepLink(InteractionChannel.RUBIKA, "CONTACT_SAVE_REQUIRED", "")
            InteractionChannel.EMAIL -> DeepLink(InteractionChannel.EMAIL, "mailto", contact.email.orEmpty())
            else -> null
        }
    }

    private fun suggestedTextFor(channel: String, contact: Contact, lead: Lead): String {
        val greeting = when (channel) {
            InteractionChannel.EMAIL -> "جناب $contact.displayName"
            else -> "سلام ${contact.displayName}"
        }
        return when (channel) {
            InteractionChannel.EMAIL ->
                "$greeting،\n\nما در شرکت تفکر پردازان، سامانه هوشمند شناسایی و پالایش سرنخ فروش را توسعه داده‌ایم. با توجه به فعالیت مجموعه شما در حوزه «${lead.businessType ?: "کسب‌وکار"}»، مایل به معرفی محصولات خود هستیم.\n\nبا احترام،\nتفکر پردازان"
            else -> "$greeting،\nمن از شرکت تفکر پردازان هستم و در مورد همکاری در زمینه معرفی محصولات هوشمند فروش با شما در ارتباط هستم. اگر امکانش باشد زمان مناسبی برای گفتگو معرفی می‌فرمایید؟"
        }
    }

    // ---- Follow-ups ----

    override suspend fun completeFollowUp(id: String) {
        simulate()
        for (ml in mockLeads) {
            val idx = ml.followUps.indexOfFirst { it.id == id }
            if (idx >= 0) {
                ml.followUps[idx] = ml.followUps[idx].copy(status = FollowUpStatus.COMPLETED, isOverdue = false)
                return
            }
        }
    }

    // ---- Reports ----
    override suspend fun submitReport(interactionId: String, request: ReportSubmitRequest): ReportSubmitResponse {
        simulate()
        val idx = interactions.indexOfFirst { it.id == interactionId }
        if (idx >= 0) {
            interactions[idx] = interactions[idx].copy(reportStatus = "SUBMITTED")
        }
        val leadId = interactions.getOrNull(idx)?.leadId ?: "lead-unknown"
        val response = ReportSubmitResponse(
            id = "rep-${interactionId}",
            interactionId = interactionId,
            outcome = request.outcome,
            contactReached = request.contactReached,
            interestLevel = request.interestLevel,
            needLevel = request.needLevel,
            mainObjection = request.mainObjection,
            summary = request.summary,
            nextAction = request.nextAction ?: "CALL_AGAIN",
            followUpAt = request.followUpAt,
            submittedBy = currentUser.id,
            createdAt = now,
            updatedAt = now,
        )
        // Mirror the backend: a submitted report immediately becomes visible in
        // the global reports list (GET /me/reports).
        submittedReports.removeAll { it.interactionId == interactionId }
        submittedReports.add(
            SubmittedReport(
                id = response.id,
                interactionId = response.interactionId,
                leadId = leadId,
                leadName = null,
                companyName = mockLeads.firstOrNull { it.lead.id == leadId }?.lead?.companyName,
                channel = interactions.getOrNull(idx)?.channel ?: "CALL",
                outcome = response.outcome,
                contactReached = response.contactReached,
                interestLevel = response.interestLevel,
                needLevel = response.needLevel,
                mainObjection = response.mainObjection,
                summary = response.summary,
                nextAction = response.nextAction,
                followUpAt = response.followUpAt,
                submittedBy = response.submittedBy,
                submitterName = currentUser.name,
                createdAt = response.createdAt,
                updatedAt = response.updatedAt,
            ),
        )
        return response
    }

    // ---- Qualification ----
    override suspend fun qualify(leadId: String, request: QualifyRequest): QualifyResponse {
        simulate()
        val ml = mock(leadId)
        ml.status = LeadStatus.QUALIFIED
        return QualifyResponse(qualified = true, leadId = leadId)
    }

    override suspend fun disqualify(leadId: String, request: DisqualifyRequest): DisqualifyResponse {
        simulate()
        val ml = mock(leadId)
        ml.status = LeadStatus.DISQUALIFIED
        return DisqualifyResponse(disqualified = true, leadId = leadId)
    }
}
