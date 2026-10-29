import type {
  AgentPerformance,
  ManagerMetrics,
  AuditEntry,
  AuthUser,
  BackendHealth,
  Contact,
  ContactReport,
  ChangeHistoryItem,
  CrmExport,
  DocumentDetail,
  DuplicatePair,
  FollowUp,
  FollowUpCreateData,
  FollowUpCreateGlobalData,
  FollowUpUpdateData,
  FunnelStep,
  GlobalSearchResults,
  ImportBatch,
  Interaction,
  Lead,
  LeadCandidate,
  LeadVersion,
  LLMSettings,
  MuninnSettings,
  NotificationItem,
   Paginated,
   Project,
   ProjectDetail,
   ProjectLeadRow,
   Campaign,
   NextActionsResponse,
   QualificationBySource,
  QualificationSettings,
  RejectionReason,
  ScoringSettings,
  SeriesPoint,
  Session,
  SourceDocument,
  SystemConfig,
  TimelineEvent,
  User,
  WorkloadItem,
} from './types'

export type LeadsQuery = {
  page?: number
  page_size?: number
  search?: string
  status?: string
  owner?: string
  province?: string
  city?: string
  industry?: string
  business_type?: string
  source_batch_id?: string
  source?: string
  score_min?: number
  score_max?: number
  created_from?: string
  created_to?: string
  tag?: string
  sort_by?: string
  sort_dir?: 'asc' | 'desc'
  unassigned?: boolean
  qualified?: boolean
  disqualified?: boolean
  exported?: boolean
  review?: boolean
  project_id?: string
}

export type ReportsQuery = {
  page?: number
  page_size?: number
  status?: string
  user_id?: string
}

export type AuditQuery = {
  page?: number
  page_size?: number
  user_id?: string
  action?: string
  entity_type?: string
  date_from?: string
  date_to?: string
}

export interface ApiEndpoints {
  // Auth
  login(login: string, password: string): Promise<{ user: AuthUser; access_token: string; refresh_token: string }>
  logout(): Promise<void>
  me(): Promise<AuthUser>

  // Dashboard
  dashboardStats(): Promise<ManagerMetrics>
  dashboardFunnel(): Promise<FunnelStep[]>
  dashboardQualificationBySource(): Promise<QualificationBySource[]>
  dashboardAgentPerformance(): Promise<AgentPerformance[]>
  dashboardRejectionReasons(): Promise<RejectionReason[]>
  dashboardLeadsByCity(): Promise<SeriesPoint[]>
  dashboardLeadTrend(period: 'day' | 'week' | 'month'): Promise<SeriesPoint[]>
  dashboardLeadTrendRaw(period: 'day' | 'week' | 'month'): Promise<SeriesPoint[]>

  // Leads
  listLeads(query: LeadsQuery): Promise<Paginated<Lead>>
  createLead(data: Partial<Lead>): Promise<Lead>
  getLead(id: string): Promise<Lead>
  updateLead(id: string, patch: Partial<Lead>, note?: string): Promise<Lead>
  leadContacts(id: string): Promise<Contact[]>
  createContact(leadId: string, contact: Partial<Contact>, note?: string): Promise<Contact>
  updateContact(leadId: string, contactId: string, patch: Partial<Contact>, note?: string): Promise<Contact>
  archiveContact(leadId: string, contactId: string): Promise<void>
  leadVersions(id: string, page?: number, pageSize?: number): Promise<Paginated<LeadVersion>>
  leadVersion(id: string, versionId: string): Promise<LeadVersion>
  recentLeadChanges(limit?: number, offset?: number): Promise<Paginated<ChangeHistoryItem>>
  leadEvidence(id: string): Promise<{ items: Array<{ field_name: string; value: string; confidence: number; source: string; excerpt: string; page: number | null; row: number | null }> }>
  leadScore(id: string): Promise<{ total: number; max: number; components: Array<{ key: string; name: string; weight: number; contribution: number; value: number; max: number }> }>
  leadTimeline(id: string): Promise<TimelineEvent[]>
  leadInteractions(id: string): Promise<Interaction[]>
  leadReports(id: string): Promise<ContactReport[]>
  leadFollowups(id: string): Promise<FollowUp[]>
  leadNotes(id: string): Promise<unknown[]>
  addNote(id: string, text: string): Promise<unknown>
  addFollowUp(leadId: string, data: FollowUpCreateData): Promise<FollowUp>
  createFollowUp(data: FollowUpCreateGlobalData): Promise<FollowUp>
  updateFollowup(id: string, data: FollowUpUpdateData): Promise<FollowUp>
  rescheduleFollowup(id: string, dueAt: string): Promise<FollowUp>
  reopenFollowup(id: string): Promise<FollowUp>
  cancelFollowup(id: string): Promise<FollowUp>
  assignLeads(leadIds: string[], assigneeId: string, method: string): Promise<void>
  qualifyLead(id: string): Promise<{ qualified: boolean; lead_id: string }>
  disqualifyLead(id: string, reason: string): Promise<{ disqualified: boolean; lead_id: string }>
  exportLeadToMuninn(id: string): Promise<CrmExport>
  getCrmExport(leadId: string): Promise<CrmExport | null>
  archiveLead(id: string): Promise<void>

  // Projects
   listProjects(query: { page?: number; page_size?: number; status?: string; campaign_id?: string }): Promise<Paginated<Project & { lead_count: number }>>
  getProject(id: string): Promise<ProjectDetail>
  createProject(data: Partial<Project>): Promise<ProjectDetail>
  updateProject(id: string, patch: Partial<Project>): Promise<ProjectDetail>
  deleteProject(id: string): Promise<void>
  listProjectLeads(projectId: string, query?: { page?: number; page_size?: number }): Promise<Paginated<ProjectLeadRow>>
  attachProjectLeads(projectId: string, leadIds: string[]): Promise<{ attached: number; project_id: string }>
  attachProjectLead(projectId: string, leadId: string): Promise<{ success: boolean; project_id: string }>
  detachProjectLead(projectId: string, leadId: string): Promise<{ success: boolean }>
  updateProjectLead(projectId: string, leadId: string, patch: { stage?: string | null; score?: number | null; next_action?: string | null; assigned_user_id?: string | null }): Promise<ProjectLeadRow>

  // Campaigns
  listCampaigns(query: { page?: number; page_size?: number; status?: string }): Promise<Paginated<Campaign>>
  getCampaign(id: string): Promise<Campaign>
  createCampaign(data: Partial<Campaign>): Promise<Campaign>
  updateCampaign(id: string, patch: Partial<Campaign>): Promise<Campaign>
  deleteCampaign(id: string): Promise<void>
  attachLeadToProject(projectId: string, leadId: string): Promise<{ success: boolean; lead_id: string; project_id: string | null }>
  detachLeadFromProject(projectId: string, leadId: string): Promise<{ success: boolean; lead_id: string; project_id: string | null }>

  // Imports
  listImports(query: { page?: number; page_size?: number }): Promise<Paginated<ImportBatch>>
  getImport(id: string): Promise<ImportBatch>
  createImportBatch(data: {
    name: string
    description?: string
    source_type: string
    tags?: string[]
    project_id?: string | null
  }): Promise<ImportBatch>
  uploadFiles(batchId: string, files: File[]): Promise<{ uploaded: number }>
  importDocuments(batchId: string): Promise<Paginated<SourceDocument>>

  // Documents
  listDocuments(query: { page?: number; page_size?: number; batch_id?: string }): Promise<Paginated<SourceDocument>>
  getDocument(id: string): Promise<DocumentDetail>
  documentCandidates(id: string): Promise<LeadCandidate[]>

  // Duplicates
  listDuplicates(query: { page?: number; page_size?: number; status?: string }): Promise<Paginated<DuplicatePair>>
  markDuplicate(id: string, action: 'merge' | 'distinct' | 'later'): Promise<void>
  mergeDuplicates(id: string, survivingLeadId: string, options: { fields: string[] }): Promise<Lead>

  // Assignments
  assignmentOverview(): Promise<{ unassigned_leads: number; employees: WorkloadItem[]; unassigned_list: Paginated<Lead> }>
  assignBulk(leadIds: string[], assigneeId: string): Promise<void>
  assignRoundRobin(leadIds: string[], assigneeIds: string[]): Promise<void>

  // Agents
  listAgents(): Promise<AgentPerformance[]>
  getAgent(id: string): Promise<{ user: Pick<User, 'id' | 'name'>; assigned_leads: number; calls_today: number; successful_contacts: number; pending_reports: number; overdue_followups: number; qualified_leads: number; qualification_rate: number; queue: Array<Pick<Lead, 'id' | 'company_name' | 'city' | 'score' | 'status'>>; recent_activity: Array<{ id: string; event_type: string; title: string; description?: string | null; created_at: string; company_name?: string | null }> }>

  // Reports
  listReports(query: ReportsQuery): Promise<Paginated<ContactReport>>
  getReport(id: string): Promise<ContactReport>
  overrideReport(id: string, reason: string): Promise<ContactReport>

  // Followups
  listFollowups(view: 'today' | 'overdue' | 'upcoming' | 'completed' | 'all' | 'pending' | 'my'): Promise<Paginated<FollowUp>>
  completeFollowup(id: string, note?: string, outcome?: string): Promise<FollowUp>

  // Next Best Action / Work queue
  nextActions(limit?: number): Promise<NextActionsResponse>
  teamNextActions(limit?: number): Promise<NextActionsResponse>
  snoozeNextAction(id: string, until: string): Promise<void>
  completeNextAction(id: string): Promise<void>

  // Users
  listUsers(): Promise<User[]>
  getUser(id: string): Promise<User>
  createUser(data: { name: string; username: string; email?: string; mobile?: string; password: string; role: string }): Promise<User>
  updateUser(id: string, patch: { name?: string; email?: string; mobile?: string; role?: string }): Promise<User>
  setUserActive(id: string, active: boolean): Promise<void>
  resetUserPassword(id: string, password: string): Promise<void>
  listUserSessions(id: string): Promise<Session[]>
  revokeSession(userId: string, sessionId: string): Promise<void>

  // Settings
  getScoringSettings(): Promise<ScoringSettings>
  updateScoringSettings(weights: Record<string, number>): Promise<ScoringSettings>
  getQualificationSettings(): Promise<QualificationSettings>
  updateQualificationSettings(patch: Partial<QualificationSettings>): Promise<QualificationSettings>
  getLlmSettings(): Promise<LLMSettings>
  updateLlmSettings(patch: Partial<LLMSettings>): Promise<LLMSettings>
  testLlmConnection(): Promise<{ ok: boolean; message: string }>
  getMuninnSettings(): Promise<MuninnSettings>
  updateMuninnSettings(patch: Partial<MuninnSettings>): Promise<MuninnSettings>
  getSystemConfig(): Promise<SystemConfig>

  // Audit
  listAudit(query: AuditQuery): Promise<Paginated<AuditEntry>>

  // Misc
  notifications(): Promise<NotificationItem[]>
  markNotificationsRead(): Promise<void>
  globalSearch(q: string): Promise<GlobalSearchResults>
  backendHealth(): Promise<BackendHealth>
  jobsPending(): Promise<{ count: number }>
}

export type ImportCreateResult = { uploaded: number }
