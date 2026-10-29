export const queryKeys = {
  dashboard: ['dashboard'] as const,
  dashboardStats: ['dashboard', 'stats'] as const,
  dashboardFunnel: ['dashboard', 'funnel'] as const,
  dashboardQualificationBySource: ['dashboard', 'qualification-by-source'] as const,
  dashboardAgentPerformance: ['dashboard', 'agent-performance'] as const,
  dashboardRejectionReasons: ['dashboard', 'rejection-reasons'] as const,
  dashboardLeadsByCity: ['dashboard', 'leads-by-city'] as const,
  dashboardLeadTrend: (period: string) => ['dashboard', 'lead-trend', period] as const,

  leads: (params: Record<string, unknown> = {}) => ['leads', params] as const,
  lead: (id: string) => ['lead', id] as const,
  leadContacts: (id: string) => ['lead', id, 'contacts'] as const,
  leadEvidence: (id: string) => ['lead', id, 'evidence'] as const,
  leadScore: (id: string) => ['lead', id, 'score'] as const,
  leadTimeline: (id: string) => ['lead', id, 'timeline'] as const,
  leadVersions: (id: string, page?: number) => ['lead', id, 'versions', page ?? 1] as const,
  leadVersion: (id: string, versionId: string) => ['lead', id, 'versions', versionId] as const,
  changeHistory: (page?: number) => ['change-history', page ?? 1] as const,
  leadInteractions: (id: string) => ['lead', id, 'interactions'] as const,
  leadReports: (id: string) => ['lead', id, 'reports'] as const,
  leadFollowups: (id: string) => ['lead', id, 'followups'] as const,
  leadNotes: (id: string) => ['lead', id, 'notes'] as const,
   leadExport: (id: string) => ['lead', id, 'export'] as const,

   projects: (params: Record<string, unknown> = {}) => ['projects', params] as const,
   project: (id: string) => ['project', id] as const,
   projectLeads: (id: string, params: Record<string, unknown> = {}) => ['project', id, 'leads', params] as const,

   campaigns: (params: Record<string, unknown> = {}) => ['campaigns', params] as const,
   campaign: (id: string) => ['campaign', id] as const,


  imports: (params: Record<string, unknown> = {}) => ['imports', params] as const,
  import: (id: string) => ['import', id] as const,
  importDocuments: (id: string) => ['import', id, 'documents'] as const,

  documents: (params: Record<string, unknown> = {}) => ['documents', params] as const,
  document: (id: string) => ['document', id] as const,

  duplicates: (params: Record<string, unknown> = {}) => ['duplicates', params] as const,

  assignmentOverview: ['assignments', 'overview'] as const,
  agents: ['agents'] as const,
  agent: (id: string) => ['agent', id] as const,
  reports: (params: Record<string, unknown> = {}) => ['reports', params] as const,
  report: (id: string) => ['report', id] as const,
  followups: (view: string) => ['followups', view] as const,

  nextActions: (limit?: number) => ['next-actions', limit ?? 'all'] as const,
  teamNextActions: (limit?: number) => ['team-next-actions', limit ?? 'all'] as const,

  users: ['users'] as const,
  user: (id: string) => ['user', id] as const,
  userSessions: (id: string) => ['user', id, 'sessions'] as const,

  scoringSettings: ['settings', 'scoring'] as const,
  qualificationSettings: ['settings', 'qualification'] as const,
  llmSettings: ['settings', 'llm'] as const,
  muninnSettings: ['settings', 'muninn'] as const,
  systemConfig: ['settings', 'system'] as const,

  audit: (params: Record<string, unknown> = {}) => ['audit', params] as const,
  notifications: ['notifications'] as const,
  pendingJobs: ['jobs', 'pending'] as const,
  health: ['health'] as const,
  search: (q: string) => ['search', q] as const,
}
