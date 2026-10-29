import { apiDelete, apiGet, apiPatch, apiPost, apiPut, buildQuery, tokenStore } from './client'
import type {
  AgentPerformance,
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
  FunnelStep,
  GlobalSearchResults,
  ImportBatch,
  Interaction,
  Lead,
  LeadCandidate,
  LeadVersion,
  LLMSettings,
  ManagerMetrics,
  MuninnSettings,
  NextActionsResponse,
   NotificationItem,
   Paginated,
   Project,
   ProjectDetail,
   ProjectLeadRow,
   Campaign,
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
} from './types'
import type { ApiEndpoints, AuditQuery, LeadsQuery, ReportsQuery } from './interface'

type PageMeta = { total: number; page: number; size: number }

function normalizeLead(l: Lead): Lead {
  return { ...l, tags: l.tags ?? [] }
}

function paginate<T>(items: T[], meta: PageMeta): Paginated<T> {
  const totalPages = Math.max(1, Math.ceil(meta.total / Math.max(meta.size, 1)))
  return {
    items,
    meta: { page: meta.page, page_size: meta.size, total: meta.total, total_pages: totalPages },
  }
}

function onePage<T>(items: T[], total?: number): Paginated<T> {
  const n = total ?? items.length
  return {
    items,
    meta: { page: 1, page_size: items.length, total: n, total_pages: Math.max(1, Math.ceil(n / Math.max(1, items.length))) },
  }
}

export const realApi: ApiEndpoints = {
  async login(login, password) {
    const data = await apiPost<{
      user: AuthUser
      access_token: string
      refresh_token: string
      expires_in?: number
    }>('/auth/login', { login, password }, { withAuth: false })
    return {
      user: data.user,
      access_token: data.access_token ?? '',
      refresh_token: data.refresh_token ?? '',
    }
  },
  logout: () =>
    apiPost(
      '/auth/logout',
      {},
      { withAuth: false, headers: tokenStore.getRefresh() ? { 'X-Refresh-Token': tokenStore.getRefresh() as string } : {} },
    ).then(() => undefined),
  me: () => apiGet<AuthUser>('/me'),

  dashboardStats: () => apiGet<ManagerMetrics>('/dashboard'),
  dashboardFunnel: () => apiGet<FunnelStep[]>('/dashboard/funnel'),
  dashboardQualificationBySource: () => apiGet<QualificationBySource[]>('/dashboard/qualification-by-source'),
  dashboardAgentPerformance: () => apiGet<AgentPerformance[]>('/dashboard/agent-performance'),
  dashboardRejectionReasons: () => apiGet<RejectionReason[]>('/dashboard/rejection-reasons'),
  dashboardLeadsByCity: () => apiGet<SeriesPoint[]>('/dashboard/leads-by-city'),
  dashboardLeadTrend: (period) => apiGet<SeriesPoint[]>(`/dashboard/lead-trend?period=${period}`),
  dashboardLeadTrendRaw: (period) => apiGet<SeriesPoint[]>(`/dashboard/lead-trend/raw?period=${period}`),

  listLeads: async (q: LeadsQuery) => {
    const params: Record<string, string | number | boolean> = {
      page: q.page ?? 1,
      page_size: q.page_size ?? 20,
    }
    if (q.search) params.search = q.search
    if (q.status) params.status = q.status
    if (q.owner) params.owner = q.owner
    if (q.province) params.province = q.province
    if (q.city) params.city = q.city
    if (q.industry) params.industry = q.industry
    if (q.business_type) params.business_type = q.business_type
    if (q.source) params.source = q.source
    if (q.source_batch_id) params.batch_id = q.source_batch_id
    if (q.score_min !== undefined) params.min_score = q.score_min
    if (q.score_max !== undefined) params.max_score = q.score_max
    if (q.tag) params.tag = q.tag
    if (q.unassigned) params.scope = 'unassigned'
    if (q.review) params.status = 'NEW'
    if (q.qualified) params.qualified = true
    if (q.project_id) params.project_id = q.project_id
    if (q.sort_by) {
      params.sort = q.sort_by === 'company' ? 'company' : q.sort_by === 'updated' ? 'updated' : q.sort_by
      params.order = q.sort_dir === 'asc' ? 'ASC' : 'DESC'
    }
    const data = await apiGet<{ leads: Lead[] } & PageMeta>(`/leads${buildQuery(params)}`)
    return paginate((data.leads ?? []).map(normalizeLead), { total: data.total, page: data.page, size: data.size })
  },
  createLead: (data) => apiPost<{ lead: Lead }>('/leads', data).then((r) => normalizeLead(r.lead)),
  getLead: async (id) => {
    const data = await apiGet<{ lead: Lead }>(`/leads/${id}`)
    return normalizeLead(data.lead)
  },
  updateLead: async (id, patch, note) => {
    const body = { ...patch } as Record<string, unknown>
    if (note) body.note = note
    const data = await apiPatch<{ lead: Lead }>(`/leads/${id}`, body)
    return normalizeLead(data.lead)
  },
  leadContacts: async (id) => {
    const data = await apiGet<{ contacts: Contact[] }>(`/leads/${id}/contacts`)
    return data.contacts ?? []
  },
  createContact: async (leadId, contact, note) => {
    const body = { ...contact } as Record<string, unknown>
    if (note) body.note = note
    return apiPost<Contact>(`/leads/${leadId}/contacts`, body)
  },
  updateContact: async (leadId, contactId, patch, note) => {
    const body = { ...patch } as Record<string, unknown>
    if (note) body.note = note
    await apiPatch<void>(`/leads/${leadId}/contacts/${contactId}`, body)
    return { id: contactId, lead_id: leadId, ...patch } as Contact
  },
  archiveContact: (leadId, contactId) => apiDelete<void>(`/leads/${leadId}/contacts/${contactId}`),
  leadVersions: async (id, page = 1, pageSize = 50) => {
    const offset = (page - 1) * pageSize
    const data = await apiGet<{ versions: LeadVersion[]; total: number; page_size: number; offset: number }>(
      `/leads/${id}/versions?limit=${pageSize}&offset=${offset}`,
    )
    return paginate(data.versions ?? [], { total: data.total, page, size: data.page_size })
  },
  leadVersion: async (id, versionId) => {
    const data = await apiGet<{ version: LeadVersion }>(`/leads/${id}/versions/${versionId}`)
    return data.version
  },
  recentLeadChanges: async (limit = 50, offset = 0) => {
    const data = await apiGet<{ items: ChangeHistoryItem[]; count: number }>(
      `/change-history?limit=${limit}&offset=${offset}`,
    )
    return onePage(data.items ?? [])
  },
  leadEvidence: (id) =>
    apiGet<{ items: Array<{ field_name: string; value: string; confidence: number; source: string; excerpt: string; page: number | null; row: number | null }> }>(
      `/leads/${id}/evidence`,
    ),
  leadScore: (id) =>
    apiGet<{ total: number; max: number; components: Array<{ key: string; name: string; weight: number; contribution: number; value: number; max: number }> }>(
      `/leads/${id}/score`,
    ),
  leadTimeline: async (id) => {
    const data = await apiGet<{ timeline: TimelineEvent[] }>(`/leads/${id}/timeline`)
    return data.timeline ?? []
  },
  leadInteractions: (id) => apiGet<Interaction[]>(`/leads/${id}/interactions`),
  leadReports: (id) => apiGet<ContactReport[]>(`/leads/${id}/reports`),
  leadFollowups: (id) => apiGet<FollowUp[]>(`/leads/${id}/followups`),
  leadNotes: async (id) => {
    const data = await apiGet<{ notes: unknown[] }>(`/leads/${id}/notes`)
    return data.notes ?? []
  },
  addNote: (id, text) => apiPost(`/leads/${id}/notes`, { text }),
  addFollowUp: (leadId, data) => apiPost<FollowUp>(`/leads/${leadId}/followups`, data),
  createFollowUp: (data) => apiPost<FollowUp>('/followups', data),
  updateFollowup: (id, data) => apiPatch<FollowUp>(`/followups/${id}`, data),
  rescheduleFollowup: (id, dueAt) => apiPost<FollowUp>(`/followups/${id}/reschedule`, { due_at: dueAt }),
  reopenFollowup: (id) => apiPost<FollowUp>(`/followups/${id}/reopen`),
  cancelFollowup: (id) => apiDelete<FollowUp>(`/followups/${id}`),
  assignLeads: async (leadIds, assigneeId, method) => {
    if (leadIds.length === 1) {
      await apiPost(`/leads/${leadIds[0]}/assign`, { assignee_id: assigneeId, method: method ?? 'manual' })
      return
    }
    await apiPost(`/leads/${leadIds[0]}/assign-bulk`, { lead_ids: leadIds, agent_ids: [assigneeId] })
  },
  qualifyLead: (id) =>
    apiPost<{ qualified: boolean; lead_id: string }>(`/leads/${id}/qualify`, {
      business_relevant: true,
      contact_info_valid: true,
      meaningful_contact_occurred: true,
      need_or_interest_exists: true,
      decision_maker_identified: true,
      next_sales_action_exists: true,
      reason: 'manager confirmation',
    }),
  disqualifyLead: (id, reason) =>
    apiPost<{ disqualified: boolean; lead_id: string }>(`/leads/${id}/disqualify`, { reason: 'OTHER', explanation: reason }),
  exportLeadToMuninn: (id) => apiPost<CrmExport>(`/leads/${id}/export/muninn`),
  getCrmExport: async (leadId) => {
    const data = await apiGet<{ export: CrmExport | null }>(`/leads/${leadId}/export`)
    return data.export ?? null
  },
   archiveLead: (id) => apiPost<void>(`/leads/${id}/archive`),

   listProjects: async (q) => {
     const params: Record<string, string | number> = { page: String(q?.page ?? 1), page_size: String(q?.page_size ?? 20) }
     if (q?.status) params.status = q.status
     if (q?.campaign_id) params.campaign_id = q.campaign_id
     const data = await apiGet<{ projects: Array<Project & { lead_count: number }>; total: number; page: number; size: number }>(`/projects${buildQuery(params)}`)
     return paginate(data.projects ?? [], { total: data.total, page: data.page, size: data.size })
   },
   getProject: (id) => apiGet<ProjectDetail>(`/projects/${id}`),
   createProject: (data) => apiPost<ProjectDetail>('/projects', data),
   updateProject: (id, patch) => apiPut<ProjectDetail>(`/projects/${id}`, patch),
   deleteProject: (id) => apiDelete<void>(`/projects/${id}`),
   listProjectLeads: async (projectId, q) => {
     const params: Record<string, string | number> = { page: String(q?.page ?? 1), page_size: String(q?.page_size ?? 50) }
     const data = await apiGet<{ leads: ProjectLeadRow[]; total: number; page: number; size: number }>(`/projects/${projectId}/leads${buildQuery(params)}`)
     return paginate(data.leads ?? [], { total: data.total, page: data.page, size: data.size })
   },
   attachProjectLeads: async (projectId, leadIds) => {
     const data = await apiPost<{ attached: number; project_id: string }>(`/projects/${projectId}/leads`, { lead_ids: leadIds })
     return data
   },
   attachProjectLead: async (projectId, leadId) => apiPost<{ success: boolean; project_id: string }>(`/projects/${projectId}/leads/${leadId}`, {}),
   detachProjectLead: async (projectId, leadId) => apiDelete<{ success: boolean }>(`/projects/${projectId}/leads/${leadId}`),
   updateProjectLead: async (projectId, leadId, patch) => apiPatch<ProjectLeadRow>(`/projects/${projectId}/leads/${leadId}`, patch),

   listCampaigns: async (q) => {
     const params: Record<string, string | number> = { page: String(q?.page ?? 1), page_size: String(q?.page_size ?? 20) }
     if (q?.status) params.status = q.status
     const data = await apiGet<{ campaigns: Campaign[]; total: number; page: number; size: number }>(`/campaigns${buildQuery(params)}`)
     return paginate(data.campaigns ?? [], { total: data.total, page: data.page, size: data.size })
   },
   getCampaign: (id) => apiGet<Campaign>(`/campaigns/${id}`),
   createCampaign: (data) => apiPost<Campaign>('/campaigns', data),
   updateCampaign: (id, patch) => apiPut<Campaign>(`/campaigns/${id}`, patch),
   deleteCampaign: (id) => apiDelete<void>(`/campaigns/${id}`),
   attachLeadToProject: (projectId, leadId) => apiPost<{ success: boolean; lead_id: string; project_id: string | null }>(`/projects/${projectId}/leads/${leadId}`),
   detachLeadFromProject: (projectId, leadId) => apiDelete<{ success: boolean; lead_id: string; project_id: string | null }>(`/projects/${projectId}/leads/${leadId}`),


  listImports: async (q) => {
    const params: Record<string, string | number> = { limit: String(q?.page_size ?? 50), offset: String(((q?.page ?? 1) - 1) * (q?.page_size ?? 50)) }
    const data = await apiGet<{ batches: ImportBatch[]; count: number }>(`/imports${buildQuery(params)}`)
    return onePage(data.batches ?? [])
  },
  getImport: async (id) => {
    const data = await apiGet<{ batch: ImportBatch; documents: SourceDocument[] }>(`/imports/${id}`)
    return data.batch
  },
  createImportBatch: (data) =>
    apiPost<ImportBatch>('/imports', {
      name: data.name,
      description: data.description ?? null,
      source_type: data.source_type,
      project_id: data.project_id ?? null,
    }),
  uploadFiles: async (batchId, files) => {
    for (const file of files) {
      const fd = new FormData()
      fd.append('file', file)
      await apiPost<SourceDocument>(`/imports/${batchId}/documents`, fd, { isFormData: true })
    }
    return { uploaded: files.length }
  },
  importDocuments: async (batchId) => {
    const data = await apiGet<{ batch: ImportBatch; documents: SourceDocument[] }>(`/imports/${batchId}`)
    return onePage(data.documents ?? [])
  },

  listDocuments: async (q) => {
    const params: Record<string, string | number> = { limit: String(q?.page_size ?? 50), offset: String(((q?.page ?? 1) - 1) * (q?.page_size ?? 50)) }
    const data = await apiGet<{ documents: SourceDocument[]; count: number }>(`/documents${buildQuery(params)}`)
    return onePage(data.documents ?? [])
  },
  getDocument: (id) => apiGet<DocumentDetail>(`/documents/${id}`),
  documentCandidates: async (id) => {
    const data = await apiGet<LeadCandidate[]>(`/documents/${id}/candidates`)
    return data ?? []
  },

  listDuplicates: async (q) => {
    const params: Record<string, string | number> = { limit: String(q?.page_size ?? 20), offset: String(((q?.page ?? 1) - 1) * (q?.page_size ?? 20)) }
    if (q?.status) params.status = q.status
    const data = await apiGet<{ duplicates: DuplicatePair[]; count: number }>(`/duplicates${buildQuery(params)}`)
    return onePage(data.duplicates ?? [])
  },
  markDuplicate: (id, action) => {
    if (action === 'merge') return apiPost<void>(`/duplicates/${id}/resolve`, { action: 'merge' })
    if (action === 'distinct') return apiPost<void>(`/duplicates/${id}/resolve`, { action: 'distinct' })
    return apiPost<void>(`/duplicates/${id}/resolve`, { action: 'later' })
  },
  mergeDuplicates: async (id, survivingLeadId) => {
    const data = await apiPost<{ target_lead_id: string }>(`/duplicates/${id}/resolve`, { action: 'merge', target_lead_id: survivingLeadId })
    const lead = await realApi.getLead(data.target_lead_id)
    return lead
  },

  assignmentOverview: async () => {
    const [unassigned_list, agents] = await Promise.all([
      realApi.listLeads({ unassigned: true, page_size: 100 }),
      realApi.listAgents(),
    ])
    const employees = agents.map((a) => ({
      user_id: a.user_id,
      name: a.name,
      current_load: a.assigned_leads,
      today_load: a.calls_today,
      qualification_performance: Math.round(a.qualification_rate),
    }))
    return {
      unassigned_leads: unassigned_list.meta.total,
      employees,
      unassigned_list,
    }
  },
  assignBulk: (leadIds, assigneeId) => realApi.assignLeads(leadIds, assigneeId, 'bulk'),
  assignRoundRobin: (leadIds, assigneeIds) =>
    apiPost(`/leads/${leadIds[0]}/assign-bulk`, { lead_ids: leadIds, agent_ids: assigneeIds }).then(() => undefined),

  listAgents: () => apiGet<AgentPerformance[]>('/agents'),
  getAgent: (id) =>
    apiGet<{
      user: Pick<User, 'id' | 'name'>
      assigned_leads: number
      calls_today: number
      successful_contacts: number
      pending_reports: number
      overdue_followups: number
      qualified_leads: number
      qualification_rate: number
      queue: Array<Pick<Lead, 'id' | 'company_name' | 'city' | 'score' | 'status'>>
      recent_activity: Array<{ id: string; event_type: string; title: string; description?: string | null; created_at: string; company_name?: string | null }>
    }>(`/agents/${id}`),

  listReports: async (q: ReportsQuery) => {
    const params: Record<string, string | number> = { limit: String(q?.page_size ?? 50), offset: String(((q?.page ?? 1) - 1) * (q?.page_size ?? 50)) }
    if (q?.status) params.status = q.status
    if (q?.user_id) params.user_id = q.user_id
    const data = await apiGet<{ reports: ContactReport[]; count: number }>(`/reports${buildQuery(params)}`)
    return onePage(data.reports ?? [])
  },
  getReport: (id) => apiGet<ContactReport>(`/reports/${id}`),
  overrideReport: (id, reason) => apiPost<ContactReport>(`/reports/${id}/override`, { reason }).then(() => ({ id } as ContactReport)),

  listFollowups: async (view) => {
    const data = await apiGet<{ followups: FollowUp[]; count: number; total: number }>(`/followups?view=${view}&limit=200`)
    return onePage(data.followups ?? [], data.total ?? data.count ?? data.followups?.length ?? 0)
  },
  completeFollowup: (id, note, outcome) => apiPost<FollowUp>(`/followups/${id}/complete`, { note, outcome }),

  nextActions: (limit = 3) => apiGet<NextActionsResponse>(`/me/next-actions${limit ? `?limit=${limit}` : ''}`),
  teamNextActions: (limit = 200) => apiGet<NextActionsResponse>(`/team/next-actions?limit=${limit}`),
  snoozeNextAction: (id, until) => apiPost<void>(`/me/next-actions/${encodeURIComponent(id)}/snooze`, { until }),
  completeNextAction: (id) => apiPost<void>(`/me/next-actions/${encodeURIComponent(id)}/complete`, {}),

  listUsers: async () => {
    const data = await apiGet<{ users: User[]; count: number }>('/users')
    return data.users ?? []
  },
  getUser: (id) => apiGet<User>(`/users/${id}`),
  createUser: (data) => apiPost<User>('/users', data),
  updateUser: (id, patch) => apiPatch<User>(`/users/${id}`, patch),
  setUserActive: (id, active) => apiPost<void>(`/users/${id}/${active ? 'reactivate' : 'deactivate'}`),
  resetUserPassword: (id, password) => apiPost<void>(`/users/${id}/reset-password`, { password }),
  listUserSessions: async (id) => {
    const data = await apiGet<{ sessions: Session[] }>(`/users/${id}/sessions`)
    return data.sessions ?? []
  },
  revokeSession: (userId, sessionId) => apiDelete<void>(`/users/${userId}/sessions/${sessionId}`),

  getScoringSettings: () => apiGet<ScoringSettings>('/settings/scoring'),
  updateScoringSettings: (weights) => apiPut<ScoringSettings>('/settings/scoring', { weights }),
  getQualificationSettings: () => apiGet<QualificationSettings>('/settings/qualification'),
  updateQualificationSettings: (patch) => apiPatch<QualificationSettings>('/settings/qualification', patch),
  getLlmSettings: () => apiGet<LLMSettings>('/settings/llm'),
  updateLlmSettings: (patch) => apiPatch<LLMSettings>('/settings/llm', patch),
  testLlmConnection: () => apiPost<{ ok: boolean; message: string }>('/settings/llm/test'),
  getMuninnSettings: () => apiGet<MuninnSettings>('/settings/muninn'),
  updateMuninnSettings: (patch) => apiPatch<MuninnSettings>('/settings/muninn', patch),
  getSystemConfig: () => apiGet<SystemConfig>('/settings/system'),

  listAudit: async (q: AuditQuery) => {
    const params: Record<string, string | number> = {
      page: String(q?.page ?? 1),
      page_size: String(q?.page_size ?? 20),
    }
    if (q?.user_id) params.user_id = q.user_id
    if (q?.action) params.action = q.action
    if (q?.entity_type) params.entity_type = q.entity_type
    const data = await apiGet<{ items: AuditEntry[]; total: number; page: number; page_size: number }>(`/audit${buildQuery(params)}`)
    return { items: data.items ?? [], meta: { page: data.page, page_size: data.page_size, total: data.total, total_pages: Math.max(1, Math.ceil(data.total / Math.max(data.page_size, 1))) } }
  },

  notifications: () => apiGet<NotificationItem[]>('/notifications'),
  markNotificationsRead: () => apiPost<void>('/notifications/read'),
  globalSearch: (q) => apiGet<GlobalSearchResults>(`/search?q=${encodeURIComponent(q)}`),
  backendHealth: () => apiGet<BackendHealth>('/health', { withAuth: false, root: true }),
  jobsPending: () => apiGet<{ count: number }>('/jobs/pending'),
}
