export type Role = 'super_admin' | 'sales_development_manager' | 'sales_agent' | 'viewer'

export type LeadStatus =
  | 'NEW'
  | 'READY'
  | 'ASSIGNED'
  | 'CONTACTING'
  | 'FOLLOW_UP'
  | 'QUALIFIED'
  | 'DISQUALIFIED'
  | 'EXPORTED_TO_MUNINN'
  | 'ARCHIVED'

export type QualityStatus = 'RAW' | 'NORMALIZED' | 'VERIFIED' | 'UNVERIFIED'

export type BatchStatus = 'UPLOADED' | 'QUEUED' | 'PROCESSING' | 'REVIEW_REQUIRED' | 'COMPLETED' | 'FAILED'

export type SourceType = 'CSV' | 'XLSX' | 'PDF' | 'DOCX' | 'TXT' | 'HTML' | 'JSON' | 'IMAGE' | 'RAW_TEXT' | 'URL'

export type ChannelType = 'CALL' | 'SMS' | 'WHATSAPP' | 'TELEGRAM' | 'BALE' | 'EITAA' | 'RUBIKA' | 'EMAIL' | 'OTHER'

export type DirectionType = 'INBOUND' | 'OUTBOUND'

export type ActionStatus = 'CREATED' | 'OPENED' | 'COMPLETED' | 'FAILED' | 'CANCELLED'

export type ReportStatus = 'PENDING' | 'SUBMITTED' | 'VOIDED_BY_MANAGER'

export type FollowupStatus = 'PENDING' | 'COMPLETED' | 'CANCELLED' | 'SNOOZED'

export type DuplicateStatus = 'POSSIBLE' | 'IGNORED' | 'MERGED' | 'DISTINCT'

export type ExportStatus = 'PENDING' | 'IN_PROGRESS' | 'SUCCEEDED' | 'FAILED' | 'SKIPPED'

export type PaginationMeta = {
  page: number
  page_size: number
  total: number
  total_pages: number
}

export type Paginated<T> = {
  items: T[]
  meta: PaginationMeta
}

export type User = {
  id: string
  name: string
  username: string
  email?: string | null
  mobile?: string | null
  role: Role
  is_active: boolean
  created_at: string
  updated_at: string
  last_login_at?: string | null
}

export type Session = {
  id: string
  user_id: string
  device?: string | null
  ip?: string | null
  user_agent?: string | null
  created_at: string
  last_used_at: string
  expires_at: string
}

export type AuthUser = Pick<User, 'id' | 'name' | 'username' | 'email' | 'role'>

export type Lead = {
  id: string
  company_name: string
  display_name?: string | null
  business_type?: string | null
  industry?: string | null
  description?: string | null
  province?: string | null
  city?: string | null
  address?: string | null
  website?: string | null
  instagram?: string | null
  source?: string | null
  source_batch_id?: string | null
  source_document_id?: string | null
  status: LeadStatus
  quality_status: QualityStatus
  score: number
  owner_user_id?: string | null
  owner_name?: string | null
  project_id?: string | null
  project?: Pick<Project, 'id' | 'title' | 'description'> | null
  projects?: ProjectMembership[]
  owner?: Pick<User, 'id' | 'name'> | null
  created_at: string
  updated_at: string
  archived_at?: string | null
  tags?: string[]
  last_contact_at?: string | null
  next_follow_up_at?: string | null
  qualification?: LeadQualification | null
  primary_contact?: Contact | null
  crm_export?: CrmExport | null
}

// ProjectMembership is a Lead's membership in a Project with the
// project-specific sales state (from the project_leads relationship).
export type ProjectMembership = {
  project_id: string
  project_title: string
  project_status?: string
  project_description?: string | null
  stage?: string | null
  score?: number | null
  assigned_user_id?: string | null
  last_contact_at?: string | null
  next_action?: string | null
  assigned_at?: string | null
  updated_at: string
}

// ProjectLeadRow is a Project's attached Lead joined with the project-specific
// sales state.
export type ProjectLeadRow = {
  id: string
  project_id: string
  lead_id: string
  stage?: string | null
  score?: number | null
  assigned_user_id?: string | null
  last_contact_at?: string | null
  next_action?: string | null
  assigned_at?: string | null
  created_at: string
  updated_at: string
  company_name: string
  industry?: string | null
  city?: string | null
  province?: string | null
  lead_status: string
  lead_score: number
  owner_user_id?: string | null
  owner_name?: string | null
}

export type Project = {
  id: string
  title: string
  description: string
  product_name?: string | null
  product_description?: string | null
  target_market?: string | null
  ideal_customer_profile?: string | null
  customer_pain_points?: string | null
  value_proposition?: string | null
  sales_objective?: string | null
  common_objections?: string | null
  created_by?: string | null
  status: 'ACTIVE' | 'ARCHIVED'
  campaign_id?: string | null
  campaign?: Pick<Campaign, 'id' | 'title'> | null
  created_at: string
  updated_at: string
}

export type Campaign = {
  id: string
  title: string
  description?: string | null
  objective?: string | null
  status: 'ACTIVE' | 'ARCHIVED'
  start_at?: string | null
  end_at?: string | null
  created_by?: string | null
  created_at: string
  updated_at: string
  projects?: Array<{ id: string; title: string; status: string; created_at: string }>
}

export type ProjectDetail = {
  project: Project
  lead_count: number
  related_leads: {
    leads: Lead[]
    total: number
    page: number
    size: number
  }
  sales_activities: Array<{
    id: string
    lead_id: string
    contact_id?: string | null
    is_generated: boolean
    generation_status?: string
    fallback_used?: boolean
    created_at: string
    opening_sentence: string
  }>
}

export type ProjectList = {
  projects: Array<Project & { lead_count: number }>
  total: number
  page: number
  size: number
}

export type LeadQualification = {
  minimum_score_met: boolean
  business_relevance: boolean
  meaningful_contact: boolean
  decision_maker: boolean
  need_interest: boolean
  next_action: boolean
  decided_at?: string | null
  decided_by?: string | null
  disqualify_reason?: string | null
}

export type Contact = {
  id: string
  lead_id: string
  first_name?: string | null
  last_name?: string | null
  full_name?: string | null
  job_title?: string | null
  department?: string | null
  mobile?: string | null
  phone?: string | null
  email?: string | null
  whatsapp?: string | null
  telegram?: string | null
  bale?: string | null
  eitaa?: string | null
  rubika?: string | null
  is_decision_maker: boolean
  is_primary_contact: boolean
  created_at: string
  updated_at: string
  archived?: boolean
}

export type Evidence = {
  id: string
  lead_id: string
  source_document_id?: string | null
  field_name: string
  source_page?: number | null
  source_row?: number | null
  source_excerpt?: string | null
  confidence: number
}

export type ScoreComponent = {
  key: string
  name: string
  weight: number
  value: number
  contribution: number
  max: number
}

export type ScoreBreakdown = {
  total: number
  max: number
  components: ScoreComponent[]
}

export type ImportBatch = {
  id: string
  name: string
  description?: string | null
  source_type: SourceType
  status: BatchStatus
  created_by?: string | null
  creator?: Pick<User, 'id' | 'name'> | null
  created_at: string
  processing_started_at?: string | null
  processing_finished_at?: string | null
  total_documents: number
  total_candidates: number
  created_leads: number
  duplicate_candidates: number
  failed_items: number
  project_id?: string | null
  project?: Pick<Project, 'id' | 'title'> | null
  error?: string | null
  progress?: number
  timeline?: ProcessingStage[]
}

export type ProcessingStage = {
  key: string
  label: string
  status: 'pending' | 'active' | 'done' | 'error'
  started_at?: string | null
  finished_at?: string | null
}

export type SourceDocument = {
  id: string
  batch_id: string
  object_key?: string | null
  mime_type?: string | null
  original_filename?: string | null
  size_bytes?: number | null
  checksum?: string | null
  status: string
  extracted_text?: string | null
  extraction_error?: string | null
  created_by?: string | null
  created_at: string
  updated_at: string
  batch_name?: string | null
  lead_count: number
}

export type DocumentDetail = SourceDocument & {
  candidates: LeadCandidate[]
}

export type LeadCandidate = {
  id: string
  lead_id?: string | null
  company_name: string
  contact_name?: string | null
  phone?: string | null
  score: number
  status: 'new' | 'created' | 'duplicate' | 'error'
  error?: string | null
}

export type DuplicatePair = {
  id: string
  lead_a_id: string
  lead_b_id: string
  reason: string
  similarity: number
  status: DuplicateStatus
  detected_by: string
  created_at: string
  lead_a?: Lead | null
  lead_b?: Lead | null
  matches: string[]
}

export type Interaction = {
  id: string
  lead_id: string
  contact_id?: string | null
  user_id: string
  channel: ChannelType
  direction: DirectionType
  objective?: string | null
  started_at: string
  ended_at?: string | null
  duration_seconds?: number | null
  action_status: ActionStatus
  report_status: ReportStatus
  suggested_message?: string | null
  suggested_subject?: string | null
  deep_link?: string | null
  created_at: string
  user?: Pick<User, 'id' | 'name'> | null
  lead?: Pick<Lead, 'id' | 'company_name'> | null
}

export type ContactReport = {
  id: string
  interaction_id: string
  outcome: string
  contact_reached?: boolean | null
  correct_person_reached?: boolean | null
  decision_maker_identified?: boolean | null
  decision_maker_name?: string | null
  decision_maker_role?: string | null
  interest_level?: string | null
  need_level?: string | null
  estimated_purchase_capacity?: string | null
  purchase_timing?: string | null
  main_objection?: string | null
  summary?: string | null
  next_action?: string | null
  follow_up_at?: string | null
  qualification_recommendation?: string | null
  submitted_by?: string | null
  manager_override: boolean
  manager_override_reason?: string | null
  overridden_by?: string | null
  created_at: string
  updated_at: string
  interaction?: Interaction | null
  submitter?: Pick<User, 'id' | 'name'> | null
  overrider?: Pick<User, 'id' | 'name'> | null
}

export type NextActionSource = 'follow_up' | 'pending_report' | 'first_contact' | 'stale_lead'

export type NextActionType = 'call' | 'message' | 'email' | 'report' | 'follow_up' | 'qualification' | 'lead_review'

export type NextActionPriority = 'urgent' | 'important' | 'normal'

export type NextAction = {
  id: string
  source_type: NextActionSource
  source_id: string
  lead_id: string
  lead_name: string
  company_name: string
  contact_name?: string | null
  action_type: NextActionType
  title: string
  reason: string
  context?: string | null
  priority: NextActionPriority
  priority_score: number
  due_at?: string | null
  overdue: boolean
  suggested_action: string
  cta_label: string
  project_id?: string | null
  project_title?: string | null
  last_interaction_at?: string | null
  assignee_name?: string | null
  explainability: string[]
}

export type NextActionsResponse = {
  items: NextAction[]
  generated_at: string
  total_actionable: number
  urgent_count: number
  overdue_count: number
}

export type FollowUp = {
  id: string
  lead_id: string
  contact_id?: string | null
  project_id?: string | null
  campaign_id?: string | null
  interaction_id?: string | null
  assigned_to?: string | null
  created_by?: string | null
  title?: string | null
  description?: string | null
  channel?: string | null
  type: string
  priority: 'LOW' | 'MEDIUM' | 'HIGH'
  status: FollowupStatus
  due_at: string
  completed_at?: string | null
  snoozed_until?: string | null
  completion_note?: string | null
  outcome?: string | null
  source: 'MANUAL' | 'REPORT' | 'SYSTEM'
  is_overdue?: boolean
  created_at: string
  updated_at?: string
  company_name?: string | null
  contact_name?: string | null
  assignee_name?: string | null
  lead?: Pick<Lead, 'id' | 'company_name'> | null
  assignee?: Pick<User, 'id' | 'name'> | null
}

export type FollowUpCreateData = {
  due_at: string
  type?: string
  title?: string | null
  description?: string | null
  channel?: string | null
  priority?: 'LOW' | 'MEDIUM' | 'HIGH'
  assigned_to?: string | null
}

export type FollowUpCreateGlobalData = FollowUpCreateData & {
  lead_id: string
}

export type FollowUpUpdateData = {
  title?: string | null
  description?: string | null
  channel?: string | null
  type?: string
  priority?: 'LOW' | 'MEDIUM' | 'HIGH'
  due_at?: string
  assigned_to?: string | null
}

export type Assignment = {
  id: string
  lead_id: string
  assignee_id: string
  assigned_by?: string | null
  method: 'manual' | 'bulk' | 'round_robin' | 'reassignment'
  created_at: string
  lead?: Pick<Lead, 'id' | 'company_name' | 'status' | 'score'> | null
  assignee?: Pick<User, 'id' | 'name'> | null
}

export type TimelineEvent = {
  id: string
  lead_id: string
  event_type: string
  title: string
  description?: string | null
  actor_user_id?: string | null
  payload?: Record<string, unknown> | null
  created_at: string
  actor?: Pick<User, 'id' | 'name'> | null
}

export type Note = {
  id: string
  lead_id: string
  author_id: string
  text: string
  created_at: string
  updated_at: string
  author?: Pick<User, 'id' | 'name'> | null
}

export type LeadChange = {
  field: string
  old?: unknown
  new?: unknown
}

export type LeadVersion = {
  id: string
  lead_id: string
  version: number
  data: Record<string, unknown>
  changes: LeadChange[]
  changed_by?: string | null
  changed_by_role?: string | null
  note?: string | null
  created_at: string
  actor?: Pick<User, 'id' | 'name'> | null
}

export type ChangeHistoryItem = LeadVersion & {
  lead_name: string
}

export type CrmExport = {
  id: string
  huginn_lead_id: string
  muninn_lead_id?: string | null
  status: ExportStatus
  attempt_count: number
  last_error?: string | null
  exported_at?: string | null
  created_at: string
  updated_at: string
}

export type AgentPerformance = {
  user_id: string
  name: string
  assigned_leads: number
  calls_today: number
  successful_contacts: number
  pending_reports: number
  overdue_followups: number
  qualified_leads: number
  qualification_rate: number
}

export type WorkloadItem = {
  user_id: string
  name: string
  current_load: number
  today_load: number
  qualification_performance: number
}

export type DashboardStats = {
  total_leads: number
  new_leads: number
  assigned_leads: number
  unassigned_leads: number
  review_required: number
  calls_today: number
  successful_contacts: number
  no_answers: number
  pending_reports: number
  follow_ups_today: number
  overdue_follow_ups: number
  qualified_leads: number
  disqualified_leads: number
  qualification_rate: number
  contact_rate: number
  exported_to_muninn: number
  by_employee: Record<string, number>
  by_source: Record<string, number>
  by_industry: Record<string, number>
  by_city: Record<string, number>
  by_business_type: Record<string, number>
  by_reject_reason: Record<string, number>
}

export type ManagerMetrics = DashboardStats

export type FunnelStep = { stage: string; count: number }

export type SeriesPoint = { label: string; value: number }

export type Series = { label: string; value: number; group?: string }

export type QualificationBySource = { source: string; total: number; qualified: number }

export type RejectionReason = { reason: string; count: number }

export type AgentPerformanceRow = AgentPerformance

export type AuditEntry = {
  id: string
  actor_user_id?: string | null
  action: string
  entity_type: string
  entity_id: string
  before_json?: Record<string, unknown> | null
  after_json?: Record<string, unknown> | null
  ip?: string | null
  user_agent?: string | null
  request_id?: string | null
  metadata?: Record<string, unknown> | null
  created_at: string
  actor?: Pick<User, 'id' | 'name'> | null
}

export type NotificationItem = {
  id: string
  type:
    | 'batch_completed'
    | 'batch_failed'
    | 'duplicates_review'
    | 'pending_reports'
    | 'overdue_followups'
    | 'muninn_export_failure'
    | 'high_value_qualified'
  title: string
  body?: string | null
  created_at: string
  read: boolean
  link?: string | null
}

export type GlobalSearchResults = {
  leads: Pick<Lead, 'id' | 'company_name' | 'city' | 'status' | 'score'>[]
  users: Pick<User, 'id' | 'name' | 'username'>[]
}

export type BackendHealth = {
  status: string
  time?: string | null
}

export type SystemConfig = {
  app_name: string
  version: string
  timezone: string
  max_upload_bytes: number
  storage_engine: string
  queue: string
  worker_count: number
}

export type LLMSettings = {
  provider: string
  model: string
  base_url: string
  timeout_seconds: number
  prompt_version: string
  structured_output_enabled: boolean
  max_retries: number
}

export type ScoringSettings = {
  dimensions: ScoreComponent[]
  total_weight: number
}

export type QualificationSettings = {
  minimum_score: number
  business_relevance_required: boolean
  meaningful_contact_required: boolean
  decision_maker_required: boolean
  need_interest_required: boolean
  next_action_required: boolean
}

export type MuninnSettings = {
  enabled: boolean
  base_url: string
  timeout_seconds: number
  export_endpoint: string
}

export type ApiErrorShape = {
  code: string
  message: string
  details?: Record<string, unknown>
}
