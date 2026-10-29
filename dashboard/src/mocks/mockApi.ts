/* =========================================================================
   Development mock layer.
   Activated ONLY when VITE_USE_MOCK_API=true. Kept entirely separate from
   production code paths. Generates deterministic, realistic-looking data so
   the UI can be developed without the backend.
   ========================================================================= */

import type {
  AgentPerformance,
  AuditEntry,
  AuthUser,
  BackendHealth,
  Contact,
  ContactReport,
  CrmExport,
  DashboardStats,
  DocumentDetail,
  DuplicatePair,
  FollowUp,
  FunnelStep,
  GlobalSearchResults,
  ImportBatch,
  Interaction,
  Lead,
  LeadCandidate,
  LLMSettings,
  MuninnSettings,
  NotificationItem,
  NextAction,
  Paginated,
  QualificationBySource,
  QualificationSettings,
  RejectionReason,
  ScoringSettings,
  SeriesPoint,
  SourceDocument,
  SystemConfig,
  TimelineEvent,
  LeadVersion,
  ChangeHistoryItem,
  User,
  WorkloadItem,
  Project,
  ProjectDetail,
  ProjectLeadRow,
  Campaign,
} from '@/api/types'
import type { ApiEndpoints, AuditQuery, LeadsQuery, ReportsQuery } from '@/api/interface'

const DAY = 24 * 60 * 60 * 1000
const HOUR = 60 * 60 * 1000

const iso = (d: Date) => d.toISOString()
const daysAgo = (n: number, hour = 10) => iso(new Date(Date.now() - n * DAY + hour * HOUR))
const hoursAgo = (h: number) => iso(new Date(Date.now() - h * HOUR))
const future = (n: number) => iso(new Date(Date.now() + n * DAY))

const companies = [
  'آریا پلیمر صنعت',
  'پخش مواد غذایی نوین',
  'تولیدی عروسک باران',
  'تجهیزات پزشکی سلامت گستر',
  'صنایع غذایی بهار',
  'آسان فناوری اطلاعات',
  'پتروشیمی امید فارس',
  'کاشی و سرامیک نقش جهان',
  'لبنیات کوهسار',
  'قطعات خودرو پارسیان',
  'نساجی گلستان',
  'فولاد آلیاژی مرکزی',
  'تولید کفش کوهستان',
  'مبلمان مدرن ایرانی',
  'بسته‌بندی صدف',
  'محصولات چوبی تیس',
  'داروسازی آریا',
  'تجهیزات مخابراتی رسا',
]

const cities = [
  { city: 'تهران', province: 'تهران' },
  { city: 'مشهد', province: 'خراسان رضوی' },
  { city: 'اصفهان', province: 'اصفهان' },
  { city: 'شیراز', province: 'فارس' },
  { city: 'تبریز', province: 'آذربایجان شرقی' },
  { city: 'کرج', province: 'البرز' },
  { city: 'قم', province: 'قم' },
  { city: 'اهواز', province: 'خوزستان' },
]

const businessTypes = ['تولیدکننده', 'پخش‌کننده', 'خدمات', 'خرده‌فروش', 'عمده‌فروش', 'شرکت دانش‌بنیان', 'کارگاه تولیدی']
const industries = ['صنایع غذایی', 'پلیمر و پلاستیک', 'فناوری اطلاعات', 'تجهیزات پزشکی', 'نساجی', 'خودرو', 'لوازم خانگی', 'صنایع دستی']
const sources = ['لیست تولیدکنندگان', 'نمایشگاه داخلی', 'وب‌سایت شرکت', 'اینستاگرام', 'اتاق بازرگانی', 'صفحه زرد', 'دیتای بازار']

const firstNames = ['علی', 'زهرا', 'محمد', 'فاطمه', 'رضا', 'مریم', 'حسین', 'نگار', 'امیر', 'سارا', 'مهدی', 'لیلا']
const lastNames = ['محمدی', 'حسینی', 'کریمی', 'رضایی', 'موسوی', 'جعفری', 'صادقی', 'نوری', 'اکبری', 'قاسمی']
const jobTitles = ['مدیرعامل', 'مدیر خرید', 'مدیر فروش', 'مدیر تولید', 'مالک', 'مدیر بازرگانی', 'مسئول تدارکات']

function pick<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)]
}
function pickN<T>(arr: T[], n: number): T[] {
  const copy = [...arr]
  const out: T[] = []
  for (let i = 0; i < n && copy.length; i++) out.push(copy.splice(Math.floor(Math.random() * copy.length), 1)[0])
  return out
}
function randInt(min: number, max: number) {
  return min + Math.floor(Math.random() * (max - min + 1))
}

const users: User[] = [
  { id: 'u-admin', name: 'مدیر کل سیستم', username: 'admin', email: 'admin@huginn.ir', mobile: '09120000001', role: 'super_admin', is_active: true, created_at: daysAgo(400), updated_at: daysAgo(2) },
  { id: 'u-manager', name: 'سارا رضایی', username: 'manager', email: 'manager@huginn.ir', mobile: '09120000002', role: 'sales_development_manager', is_active: true, created_at: daysAgo(300), updated_at: daysAgo(1) },
  { id: 'u-agent1', name: 'علی محمدی', username: 'agent1', email: 'ali@huginn.ir', mobile: '09120000003', role: 'sales_agent', is_active: true, created_at: daysAgo(200), updated_at: daysAgo(5) },
  { id: 'u-agent2', name: 'زهرا حسینی', username: 'agent2', email: 'zahra@huginn.ir', mobile: '09120000004', role: 'sales_agent', is_active: true, created_at: daysAgo(200), updated_at: daysAgo(5) },
  { id: 'u-agent3', name: 'رضا کریمی', username: 'agent3', email: 'reza@huginn.ir', mobile: '09120000005', role: 'sales_agent', is_active: true, created_at: daysAgo(180), updated_at: daysAgo(5) },
  { id: 'u-viewer', name: 'بیننده سیستم', username: 'viewer', email: 'viewer@huginn.ir', role: 'viewer', is_active: true, created_at: daysAgo(100), updated_at: daysAgo(10) },
]

const authUser = (u: User): AuthUser => ({ id: u.id, name: u.name, username: u.username, email: u.email, role: u.role })

let leadSeq = 0
function makeLead(over: Partial<Lead> = {}): Lead {
  const comp = companies[leadSeq % companies.length]
  const cityObj = cities[leadSeq % cities.length]
  const businessType = pick(businessTypes)
  const industry = pick(industries)
  const score = randInt(20, 95)
  const status: Lead['status'] =
    score >= 70
      ? pick<Lead['status']>(['QUALIFIED', 'QUALIFIED', 'CONTACTING', 'ASSIGNED'])
      : pick<Lead['status']>(['NEW', 'READY', 'ASSIGNED', 'FOLLOW_UP', 'DISQUALIFIED', 'DISQUALIFIED', 'ARCHIVED'])
  const owner = status === 'NEW' || status === 'READY' ? null : pick([users[2], users[3], users[4]])
  leadSeq++
  return {
    id: `lead-${leadSeq}`,
    company_name: `${comp} ${leadSeq}`,
    display_name: null,
    business_type: businessType,
    industry,
    description: `تولید و عرضه محصولات در حوزه ${industry}`,
    province: cityObj.province,
    city: cityObj.city,
    address: `خیابان اصلی، کوچه ${randInt(1, 40)}، پلاک ${randInt(1, 200)}`,
    website: `https://www.example${leadSeq}.ir`,
    instagram: `instagram.com/brand${leadSeq}`,
    source_batch_id: `batch-${(leadSeq % 4) + 1}`,
    source_document_id: `doc-${(leadSeq % 4) + 1}`,
    status,
    quality_status: pick(['NORMALIZED', 'VERIFIED', 'UNVERIFIED']),
    score,
    owner_user_id: owner?.id ?? null,
    owner: owner ? { id: owner.id, name: owner.name } : null,
    created_at: daysAgo(randInt(0, 120)),
    updated_at: daysAgo(randInt(0, 5)),
    archived_at: status === 'ARCHIVED' ? daysAgo(randInt(1, 20)) : null,
    tags: pickN(['فوری', 'پرارزش', 'منطقه مرکزی', 'فروش عمده'], randInt(0, 2)),
    last_contact_at: ['ASSIGNED', 'CONTACTING', 'FOLLOW_UP', 'QUALIFIED'].includes(status) ? daysAgo(randInt(1, 10)) : null,
    next_follow_up_at: ['CONTACTING', 'FOLLOW_UP'].includes(status) ? future(randInt(1, 5)) : null,
    primary_contact: {
      id: `contact-${leadSeq}`,
      lead_id: `lead-${leadSeq}`,
      full_name: `${pick(firstNames)} ${pick(lastNames)}`,
      job_title: pick(jobTitles),
      mobile: `0912${String(randInt(1000000, 9999999))}`,
      phone: `0${randInt(21, 35)}${String(randInt(1000000, 9999999))}`,
      is_decision_maker: Math.random() > 0.5,
      is_primary_contact: true,
      created_at: daysAgo(30),
      updated_at: daysAgo(3),
    },
    crm_export:
      status === 'EXPORTED_TO_MUNINN' || (status === 'QUALIFIED' && score > 85)
        ? {
            id: `exp-${leadSeq}`,
            huginn_lead_id: `lead-${leadSeq}`,
            muninn_lead_id: `MN-${randInt(1000, 9999)}`,
            status: status === 'EXPORTED_TO_MUNINN' ? 'SUCCEEDED' : pick(['PENDING', 'IN_PROGRESS', 'FAILED']),
            attempt_count: randInt(0, 2),
            last_error: Math.random() > 0.8 ? 'connection timeout to muninn' : null,
            exported_at: status === 'EXPORTED_TO_MUNINN' ? daysAgo(randInt(1, 15)) : null,
            created_at: daysAgo(randInt(1, 20)),
            updated_at: daysAgo(randInt(0, 5)),
          }
        : null,
    ...over,
  }
}

const leads: Lead[] = Array.from({ length: 640 }, () => makeLead({}))
leads.push(
  makeLead({ company_name: 'تولیدی عروسک باران', business_type: 'تولیدکننده', industry: 'اسباب‌بازی', city: 'تهران', province: 'تهران', score: 94, status: 'QUALIFIED', owner: users[2], created_at: daysAgo(2) }),
)
leads.push(
  makeLead({ company_name: 'پخش مواد غذایی نوین', business_type: 'پخش‌کننده', industry: 'صنایع غذایی', city: 'اصفهان', province: 'اصفهان', score: 82, status: 'QUALIFIED', owner: users[3], created_at: daysAgo(3) }),
)

// Deterministic export states for export UI tests.
const exportFixture: Lead | undefined = leads.find((l) => l.id === 'lead-3')
if (exportFixture) {
  exportFixture.status = 'QUALIFIED'
  exportFixture.score = 88
  exportFixture.crm_export = {
    id: 'exp-fix-failed',
    huginn_lead_id: exportFixture.id,
    muninn_lead_id: null,
    status: 'FAILED',
    attempt_count: 2,
    last_error: 'connection timeout to muninn',
    exported_at: null,
    created_at: daysAgo(1),
    updated_at: hoursAgo(1),
  }
}
const exportOkFixture: Lead | undefined = leads.find((l) => l.id === 'lead-2')
if (exportOkFixture) {
  exportOkFixture.status = 'EXPORTED_TO_MUNINN'
  exportOkFixture.crm_export = {
    id: 'exp-fix-ok',
    huginn_lead_id: exportOkFixture.id,
    muninn_lead_id: 'MN-1234',
    status: 'SUCCEEDED',
    attempt_count: 1,
    last_error: null,
    exported_at: daysAgo(2),
    created_at: daysAgo(2),
    updated_at: daysAgo(2),
  }
}

const batches: ImportBatch[] = [1, 2, 3, 4].map((n) => {
  const statuses: ImportBatch['status'][] = ['COMPLETED', 'COMPLETED', 'PROCESSING', 'REVIEW_REQUIRED']
  const s = statuses[n - 1]
  return {
    id: `batch-${n}`,
    name: `دسته بازار ${['لوازم خانگی', 'صنایع غذایی', 'پلیمر', 'تجهیزات پزشکی'][n - 1]} ${1400 + n}`,
    description: `اطلاعات بازار ${['لوازم خانگی', 'صنایع غذایی', 'پلیمر', 'تجهیزات پزشکی'][n - 1]}`,
    source_type: n === 3 ? 'PDF' : 'XLSX',
    status: s,
    created_by: 'u-manager',
    creator: { id: 'u-manager', name: 'سارا رضایی' },
    created_at: daysAgo(n * 6),
    processing_started_at: daysAgo(n * 6 - 1),
    processing_finished_at: s === 'COMPLETED' ? daysAgo(n * 6 - 2) : null,
    total_documents: randInt(4, 18),
    total_candidates: randInt(80, 500),
    created_leads: randInt(40, 320),
    duplicate_candidates: randInt(2, 40),
    failed_items: randInt(0, 8),
    error: s === 'FAILED' ? 'LLM provider timeout' : null,
    progress: s === 'COMPLETED' ? 100 : s === 'PROCESSING' ? 55 : s === 'REVIEW_REQUIRED' ? 88 : 20,
    timeline: [
      { key: 'uploaded', label: 'uploaded', status: 'done', started_at: daysAgo(6), finished_at: daysAgo(6) },
      { key: 'extracting', label: 'extracting', status: s === 'COMPLETED' ? 'done' : 'active', started_at: daysAgo(6), finished_at: daysAgo(5) },
      { key: 'analyzing', label: 'analyzing', status: s === 'COMPLETED' || s === 'REVIEW_REQUIRED' ? 'done' : s === 'FAILED' ? 'error' : 'active' },
      { key: 'deduplicating', label: 'deduplicating', status: s === 'COMPLETED' ? 'done' : 'pending' },
      { key: 'creating_leads', label: 'creating_leads', status: s === 'COMPLETED' ? 'done' : 'pending' },
      { key: 'completed', label: 'completed', status: s === 'COMPLETED' ? 'done' : 'pending' },
    ],
  }
})

const documents: SourceDocument[] = batches.flatMap((b, bi) =>
  Array.from({ length: b.total_documents }, () => {
    const statuses = ['UPLOADED', 'EXTRACTING', 'ANALYZING', 'COMPLETED', 'FAILED']
    return {
      id: `doc-${bi + 1}`,
      batch_id: b.id,
      object_key: `market/${b.id}/${bi + 1}.pdf`,
      mime_type: 'application/pdf',
      original_filename: `مشتریان-${bi + 1}-${['خانگی', 'غذایی', 'پلیمر', 'پزشکی'][bi]}.pdf`,
      size_bytes: randInt(50_000, 2_500_000),
      checksum: `sha256-${randInt(1000, 9999)}`,
      status: b.status === 'COMPLETED' ? 'COMPLETED' : pick(statuses),
      extracted_text: b.status === 'COMPLETED' ? `لیست ${randInt(10, 200)} شرکت فعال در حوزه بازار با مشخصات تماس...` : null,
      extraction_error: null,
      created_at: daysAgo(5),
      updated_at: daysAgo(4),
      batch_name: b.name,
      lead_count: b.status === 'COMPLETED' ? randInt(3, 40) : 0,
    }
  }),
)

const interactions: Interaction[] = leads.slice(0, 80).map((l, i) => {
  const agent = l.owner ?? users[2]
  const channels = ['CALL', 'CALL', 'SMS', 'WHATSAPP', 'TELEGRAM', 'EMAIL'] as const
  return {
    id: `interaction-${i + 1}`,
    lead_id: l.id,
    contact_id: null,
    user_id: agent.id,
    user: { id: agent.id, name: agent.name },
    channel: channels[i % channels.length],
    direction: 'OUTBOUND',
    objective: 'معرفی اولیه و بررسی نیاز',
    started_at: daysAgo(randInt(1, 9), randInt(9, 17)),
    ended_at: daysAgo(randInt(1, 9), randInt(9, 18)),
    duration_seconds: randInt(30, 900),
    action_status: pick(['COMPLETED', 'COMPLETED', 'FAILED']),
    report_status: i % 3 === 0 ? 'PENDING' : 'SUBMITTED',
    created_at: daysAgo(randInt(1, 9)),
    lead: { id: l.id, company_name: l.company_name },
  }
})

const reports: ContactReport[] = interactions
  .filter((it) => it.report_status === 'SUBMITTED')
  .slice(0, 50)
  .map((it, i) => ({
    id: `report-${i + 1}`,
    interaction_id: it.id,
    outcome: pick(['تماس موفق', 'تماس موفق', 'بدون پاسخ', 'مشغول بود', 'اشتباهی تماس گرفته شد']),
    contact_reached: i % 3 !== 0,
    correct_person_reached: i % 3 === 0 ? false : i % 4 !== 0,
    decision_maker_identified: i % 2 === 0,
    decision_maker_name: pick(firstNames),
    decision_maker_role: pick(jobTitles),
    interest_level: pick(['کم', 'متوسط', 'زیاد', 'خیلی زیاد']),
    need_level: pick(['کم', 'متوسط', 'زیاد']),
    estimated_purchase_capacity: pick(['کم', 'متوسط', 'زیاد', 'خیلی زیاد']),
    purchase_timing: pick(['۱-۳ ماه', '۳-۶ ماه', '۶ ماه به بعد', 'نامشخص']),
    main_objection: pick(['بودجه ندارد', 'نیاز ندارد', 'به فکر خرید نیست', 'رقبا قیمت بهتری دارند']),
    summary: `گفتگو با شرکت در خصوص محصولات و نیازهای احتمالی انجام شد.`,
    next_action: pick(['ارسال پیشنهاد قیمت', 'تماس مجدد بعد از دو هفته', 'ارسال کاتالوگ', 'بررسی نیاز تا ماه آینده']),
    follow_up_at: future(randInt(2, 20)),
    qualification_recommendation: i % 2 === 0 ? 'قابل احراز' : 'مناسب نیست',
    submitted_by: it.user_id,
    submitter: it.user,
    manager_override: i % 7 === 0,
    manager_override_reason: i % 7 === 0 ? 'نتیجه به‌درستی ثبت نشده بود' : null,
    overridden_by: i % 7 === 0 ? 'u-manager' : null,
    overrider: i % 7 === 0 ? { id: 'u-manager', name: 'سارا رضایی' } : null,
    created_at: it.started_at,
    updated_at: daysAgo(1),
    interaction: it,
  }))

const followups: FollowUp[] = [
  ...leads.slice(0, 30).map((l, i) => {
    const completed = i % 4 === 0
    const overdue = !completed && i % 6 === 0
    const due = i % 5 === 0 ? future(randInt(1, 6)) : daysAgo(randInt(0, 8))
    return {
      id: `fu-${i + 1}`,
      lead_id: l.id,
      lead: { id: l.id, company_name: l.company_name },
      company_name: l.company_name,
      assigned_to: l.owner?.id ?? null,
      assignee: l.owner ? { id: l.owner.id, name: l.owner.name } : null,
      due_at: due,
      type: i % 3 === 0 ? 'CALL' : i % 3 === 1 ? 'FOLLOW_UP' : 'MESSAGE',
      description: pick(['پیگیری پس از تماس اولیه', 'ارسال پیشنهاد', 'بررسی نیاز مجدد', 'هماهنگی جلسه']),
      priority: (i % 3 === 0 ? 'HIGH' : i % 3 === 1 ? 'MEDIUM' : 'LOW') as FollowUp['priority'],
      status: (completed ? 'COMPLETED' : 'PENDING') as FollowUp['status'],
      is_overdue: overdue,
      source: 'MANUAL' as const,
      created_at: daysAgo(5),
      completed_at: completed ? daysAgo(1) : null,
      created_by: 'u-manager',
    }
  }),
]

function mockNextActions(team = false): NextAction[] {
  const items: NextAction[] = []
  const pending = followups.filter((f) => f.status === 'PENDING').slice(0, 6)
  pending.forEach((f, i) => {
    const overdue = f.is_overdue ?? false
    const priority: NextAction['priority'] = overdue ? 'urgent' : i % 2 === 0 ? 'important' : 'normal'
    const actionType: NextAction['action_type'] =
      f.type === 'CALL' ? 'call' : f.type === 'MESSAGE' ? 'message' : 'follow_up'
    items.push({
      id: `follow_up:${f.id}`,
      source_type: 'follow_up',
      source_id: f.id,
      lead_id: f.lead_id,
      lead_name: f.company_name ?? '',
      company_name: f.company_name ?? '',
      action_type: actionType,
      title: `${actionType === 'call' ? 'تماس با' : 'پیگیری'} ${f.company_name ?? ''}`,
      reason: f.description ?? '',
      priority,
      priority_score: overdue ? 92 : i % 2 === 0 ? 65 : 40,
      due_at: f.due_at,
      overdue,
      suggested_action: actionType,
      cta_label: actionType === 'call' ? 'تماس' : actionType === 'message' ? 'پیام' : 'پیگیری',
      assignee_name: f.assignee?.name ?? null,
      explainability: overdue ? ['از زمان پیگیری گذشته است', 'سرنخ گرم است'] : ['سرنخ گرم است'],
    })
  })
  // A pending report item always ranks near the top.
  items.unshift({
    id: 'pending_report:int-1',
    source_type: 'pending_report',
    source_id: 'int-1',
    lead_id: leads[0].id,
    lead_name: leads[0].company_name,
    company_name: leads[0].company_name,
    action_type: 'report',
    title: `ثبت گزارش تماس با ${leads[0].company_name}`,
    reason: 'گزارش تماس ثبت نشده',
    priority: 'urgent',
    priority_score: 96,
    overdue: true,
    suggested_action: 'report',
    cta_label: 'ثبت گزارش',
    assignee_name: team ? leads[0].owner?.name ?? null : null,
    explainability: ['گزارش تماس ثبت نشده', 'گزارش برای مدت طولانی ثبت نشده است'],
  })
  return items
}

const duplicates: DuplicatePair[] = [
  {
    id: 'dup-1',
    lead_a_id: 'lead-1',
    lead_b_id: 'lead-2',
    reason: 'همان شرکت با دو تلفن',
    similarity: 0.93,
    status: 'POSSIBLE',
    detected_by: 'llm',
    created_at: daysAgo(2),
    lead_a: leads[0],
    lead_b: leads[1],
    matches: ['company_name', 'city', 'phone'],
  },
  {
    id: 'dup-2',
    lead_a_id: 'lead-3',
    lead_b_id: 'lead-4',
    reason: 'شماره موبایل یکسان',
    similarity: 0.88,
    status: 'POSSIBLE',
    detected_by: 'system',
    created_at: daysAgo(1),
    lead_a: leads[2],
    lead_b: leads[3],
    matches: ['mobile', 'email'],
  },
  {
    id: 'dup-3',
    lead_a_id: 'lead-5',
    lead_b_id: 'lead-6',
    reason: 'وب‌سایت یکسان',
    similarity: 0.81,
    status: 'POSSIBLE',
    detected_by: 'llm',
    created_at: hoursAgo(6),
    lead_a: leads[4],
    lead_b: leads[5],
    matches: ['website'],
  },
]

const auditEntries: AuditEntry[] = [
  { id: 'aud-1', actor_user_id: 'u-manager', action: 'lead.qualify', entity_type: 'lead', entity_id: 'lead-1', ip: '192.168.1.10', created_at: hoursAgo(3), actor: { id: 'u-manager', name: 'سارا رضایی' }, before_json: { status: 'CONTACTING' }, after_json: { status: 'QUALIFIED', score: 94 } },
  { id: 'aud-2', actor_user_id: 'u-admin', action: 'user.create', entity_type: 'user', entity_id: 'u-agent3', ip: '192.168.1.11', created_at: hoursAgo(20), actor: { id: 'u-admin', name: 'مدیر کل سیستم' }, after_json: { username: 'agent3', role: 'sales_agent' } },
  { id: 'aud-3', actor_user_id: 'u-manager', action: 'lead.assign', entity_type: 'lead', entity_id: 'lead-12', ip: '192.168.1.10', created_at: hoursAgo(5), actor: { id: 'u-manager', name: 'سارا رضایی' }, before_json: { owner_user_id: null }, after_json: { owner_user_id: 'u-agent1' } },
  { id: 'aud-4', actor_user_id: 'u-manager', action: 'duplicates.merge', entity_type: 'duplicate', entity_id: 'dup-9', ip: '192.168.1.10', created_at: daysAgo(1), actor: { id: 'u-manager', name: 'سارا رضایی' }, after_json: { surviving_lead_id: 'lead-77' } },
  { id: 'aud-5', actor_user_id: 'u-admin', action: 'import.create', entity_type: 'import_batch', entity_id: 'batch-4', ip: '192.168.1.12', created_at: daysAgo(1), actor: { id: 'u-admin', name: 'مدیر کل سیستم' }, after_json: { name: 'دسته بازار تجهیزات پزشکی' } },
  { id: 'aud-6', actor_user_id: 'u-manager', action: 'report.override', entity_type: 'contact_report', entity_id: 'report-7', ip: '192.168.1.10', created_at: hoursAgo(8), actor: { id: 'u-manager', name: 'سارا رضایی' }, before_json: { outcome: 'بدون پاسخ' }, after_json: { outcome: 'تماس موفق' }, metadata: { reason: 'ثبت اشتباه' } },
]

const notifications: NotificationItem[] = [
  { id: 'n1', type: 'batch_completed', title: 'پردازش «دسته بازار لوازم خانگی» تکمیل شد', body: '۳۲۰ سرنخ جدید ایجاد شد.', created_at: hoursAgo(2), read: false, link: '/imports/batch-1' },
  { id: 'n2', type: 'duplicates_review', title: 'نیاز به بررسی تکراری‌ها', body: '۳ مورد تکراری در انتظار بررسی.', created_at: hoursAgo(4), read: false, link: '/duplicates' },
  { id: 'n3', type: 'pending_reports', title: 'گزارش تماس در انتظار ثبت', body: '۶ گزارش در انتظار ثبت از سمت کارشناسان.', created_at: hoursAgo(6), read: false, link: '/reports' },
  { id: 'n4', type: 'overdue_followups', title: 'پیگیری عقب‌افتاده', body: '۴ پیگیری از موعد گذشته‌اند.', created_at: daysAgo(1), read: true, link: '/followups' },
  { id: 'n5', type: 'high_value_qualified', title: 'سرنخ پرارزش واجد شرایط شد', body: 'تولیدی عروسک باران (امتیاز ۹۴)', created_at: hoursAgo(9), read: false, link: '/leads/lead-642' },
]

function paginate<T>(items: T[], page: number, pageSize: number): Paginated<T> {
  const total = items.length
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const start = (page - 1) * pageSize
  return {
    items: items.slice(start, start + pageSize),
    meta: { page, page_size: pageSize, total, total_pages: totalPages },
  }
}

function onePage<T>(items: T[]): Paginated<T> {
  return {
    items,
    meta: { page: 1, page_size: items.length, total: items.length, total_pages: Math.max(1, items.length) },
  }
}

const wait = (ms = 250) => new Promise((r) => setTimeout(r, ms))

const mockCampaigns: Campaign[] = [
  {
    id: 'camp-1', title: 'Autumn 1405 Sales Campaign', description: 'Strategic push for the second half of the year',
    objective: 'Sign 40 new customers', status: 'ACTIVE', created_by: users[0].id, created_at: daysAgo(60), updated_at: daysAgo(10),
  },
]

const mockProjects: Project[] = [
  {
    id: 'proj-1', title: 'ERP Implementation', description: 'Roll out a new ERP system across the sales team.',
    product_name: 'HuginnERP', product_description: 'Modular ERP for manufacturing SMEs', value_proposition: 'Reduce manual data entry by 80%',
    sales_objective: 'Win 15 new ERP customers in 12 months', common_objections: 'Cost, integration effort', status: 'ACTIVE', created_by: users[0].id,
    campaign_id: 'camp-1', created_at: daysAgo(40), updated_at: daysAgo(40),
  },
  {
    id: 'proj-2', title: 'Online Store Launch', description: 'Launch an e-commerce presence for B2C products.',
    product_name: 'ShopSite', target_market: 'Online consumers 18-45', value_proposition: '3x online revenue in 18 months',
    status: 'ACTIVE', created_by: users[0].id, campaign_id: 'camp-1', created_at: daysAgo(20), updated_at: daysAgo(20),
  },
]

type MockMembership = {
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
}

// Project<->Lead memberships (project_leads in production).
const memberships: MockMembership[] = []
const membershipOf = (projectId: string, leadId: string) => memberships.find((m) => m.project_id === projectId && m.lead_id === leadId)
const membershipRows = (projectId: string): MockMembership[] => memberships.filter((m) => m.project_id === projectId)
const membershipOfLead = (leadId: string): MockMembership[] => memberships.filter((m) => m.lead_id === leadId)
function attachMembership(projectId: string, leadId: string): MockMembership {
  const existing = membershipOf(projectId, leadId)
  if (existing) return existing
  const m: MockMembership = {
    id: `m-${projectId}-${leadId}`,
    project_id: projectId,
    lead_id: leadId,
    stage: null,
    score: null,
    last_contact_at: null,
    next_action: null,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }
  memberships.push(m)
  return m
}
function attachLeadMemberships(projectId: string, leadIds: string[]): number {
  let n = 0
  for (const id of leadIds) {
    if (!membershipOf(projectId, id)) {
      attachMembership(projectId, id)
      n++
    }
  }
  return n
}

// Seed a few memberships so the dashboard has realistic Project<->Lead data.
for (const [pid, leadsArr, stage] of [
  ['proj-1', leads.slice(0, 3), 'MEETING_COMPLETED'],
  ['proj-1', leads.slice(3, 5), 'FIRST_CONTACT'],
  ['proj-2', leads.slice(5, 7), 'QUALIFIED'],
] as Array<[string, Lead[], string]>) {
  for (const l of leadsArr) {
    const m = attachMembership(pid, l.id)
    m.stage = stage
    m.score = l.score
    m.next_action = 'Call to follow up'
    m.last_contact_at = daysAgo(2)
  }
}


export const mockApi: ApiEndpoints = {
  async login(login, password) {
    await wait(400)
    const u = users.find((x) => x.username === login || x.email === login) ?? users[0]
    if (password && password.length < 4) throw new Error('INVALID_CREDENTIALS')
    return { user: authUser(u), access_token: 'mock-access-token', refresh_token: 'mock-refresh-token' }
  },
  async logout() {
    await wait(100)
  },
  async me() {
    await wait(100)
    return authUser(users[0])
  },

  async dashboardStats(): Promise<DashboardStats> {
    await wait()
    const exported = leads.filter((l) => l.status === 'EXPORTED_TO_MUNINN').length
    const disqualified = leads.filter((l) => l.status === 'DISQUALIFIED').length
    const qualifiedTotal = leads.filter((l) => l.status === 'QUALIFIED').length
    return {
      total_leads: leads.length,
      new_leads: leads.filter((l) => l.status === 'NEW').length,
      assigned_leads: leads.filter((l) => l.status === 'ASSIGNED' || l.status === 'CONTACTING' || l.status === 'FOLLOW_UP').length,
      unassigned_leads: leads.filter((l) => !l.owner_user_id && l.status !== 'DISQUALIFIED' && l.status !== 'ARCHIVED').length,
      review_required: leads.filter((l) => l.score >= 80 && l.status === 'NEW').length,
      calls_today: 38,
      successful_contacts: 21,
      no_answers: 11,
      pending_reports: interactions.filter((i) => i.report_status === 'PENDING').length,
      follow_ups_today: followups.filter((f) => f.status === 'PENDING' && !f.is_overdue).length,
      overdue_follow_ups: followups.filter((f) => f.status === 'PENDING' && f.is_overdue).length,
      qualified_leads: qualifiedTotal,
      disqualified_leads: disqualified,
      qualification_rate: qualifiedTotal + disqualified > 0 ? Math.round((qualifiedTotal / (qualifiedTotal + disqualified)) * 100) : 0,
      contact_rate: 55,
      exported_to_muninn: exported,
      by_employee: {},
      by_source: {},
      by_industry: {},
      by_city: {},
      by_business_type: {},
      by_reject_reason: {},
    }
  },
  async dashboardFunnel(): Promise<FunnelStep[]> {
    await wait(200)
    return [
      { stage: 'raw', count: 1850 },
      { stage: 'assigned', count: 1210 },
      { stage: 'contacted', count: 860 },
      { stage: 'interested', count: 420 },
      { stage: 'qualified', count: 190 },
      { stage: 'muninn', count: 87 },
    ]
  },
  async dashboardQualificationBySource(): Promise<QualificationBySource[]> {
    await wait(200)
    return sources.slice(0, 6).map((s) => ({ source: s, total: randInt(80, 300), qualified: randInt(10, 90) }))
  },
  async dashboardAgentPerformance(): Promise<AgentPerformance[]> {
    await wait(200)
    return [users[2], users[3], users[4]].map((u) => ({
      user_id: u.id,
      name: u.name,
      assigned_leads: randInt(40, 120),
      calls_today: randInt(8, 20),
      successful_contacts: randInt(4, 12),
      pending_reports: randInt(0, 6),
      overdue_followups: randInt(0, 4),
      qualified_leads: randInt(5, 30),
      qualification_rate: randInt(25, 70),
    }))
  },
  async dashboardRejectionReasons(): Promise<RejectionReason[]> {
    await wait(200)
    return [
      { reason: 'نامرتبط', count: 42 },
      { reason: 'شماره اشتباه', count: 36 },
      { reason: 'نیاز ندارد', count: 28 },
      { reason: 'بودجه ندارد', count: 17 },
      { reason: 'خیلی کوچک', count: 9 },
      { reason: 'پاسخ نمی‌دهد', count: 23 },
    ]
  },
  async dashboardLeadsByCity(): Promise<SeriesPoint[]> {
    await wait(200)
    return cities.map((c) => ({ label: c.city, value: leads.filter((l) => l.city === c.city).length || randInt(20, 120) }))
  },
  async dashboardLeadTrend(period): Promise<SeriesPoint[]> {
    await wait(200)
    const n = period === 'day' ? 14 : 12
    const step = period === 'day' ? DAY : period === 'week' ? 7 * DAY : 30 * DAY
    const fmt = (d: Date) => (period === 'month' ? d.toISOString().slice(0, 7) : d.toISOString().slice(0, 10))
    return Array.from({ length: n }, (_, i) => ({
      label: fmt(new Date(Date.now() - (n - 1 - i) * step)),
      value: randInt(15, 90),
    }))
  },
  async dashboardLeadTrendRaw(period): Promise<SeriesPoint[]> {
    return this.dashboardLeadTrend(period)
  },

  async listLeads(q: LeadsQuery): Promise<Paginated<Lead>> {
    await wait(350)
    let items = [...leads]
    if (q.search) {
      const s = q.search.toLowerCase()
      items = items.filter(
        (l) => l.company_name.toLowerCase().includes(s) || l.city?.toLowerCase().includes(s) || l.primary_contact?.full_name?.toLowerCase().includes(s) || l.primary_contact?.mobile?.includes(s) || (l.tags ?? []).some((t) => t.toLowerCase().includes(s)),
      )
    }
    if (q.status) items = items.filter((l) => l.status === q.status)
    if (q.owner) items = items.filter((l) => l.owner_user_id === q.owner)
    if (q.city) items = items.filter((l) => l.city === q.city)
    if (q.province) items = items.filter((l) => l.province === q.province)
    if (q.industry) items = items.filter((l) => l.industry === q.industry)
    if (q.business_type) items = items.filter((l) => l.business_type === q.business_type)
    if (q.score_min !== undefined) items = items.filter((l) => l.score >= (q.score_min as number))
    if (q.score_max !== undefined) items = items.filter((l) => l.score <= (q.score_max as number))
    if (q.tag) items = items.filter((l) => (l.tags ?? []).includes(q.tag as string))
    if (q.unassigned) items = items.filter((l) => !l.owner_user_id)
    if (q.qualified) items = items.filter((l) => l.status === 'QUALIFIED' || l.status === 'EXPORTED_TO_MUNINN')
    if (q.disqualified) items = items.filter((l) => l.status === 'DISQUALIFIED')
    if (q.exported) items = items.filter((l) => l.status === 'EXPORTED_TO_MUNINN')

    if (q.sort_by) {
      const dir = q.sort_dir === 'desc' ? -1 : 1
      const key = q.sort_by as keyof Lead
      items.sort((a, b) => {
        const av = a[key] ?? ''
        const bv = b[key] ?? ''
        return typeof av === 'number' ? (av - (bv as number)) * dir : String(av).localeCompare(String(bv)) * dir
      })
    }
    const page = q.page ?? 1
    const pageSize = q.page_size ?? 20
    return paginate(items, page, pageSize)
  },
  async getLead(id) {
    await wait(200)
    const l = leads.find((x) => x.id === id)
    if (!l) throw new Error('NOT_FOUND')
    const ms = membershipOfLead(id).map((m) => {
      const p = mockProjects.find((pp) => pp.id === m.project_id)
      return {
        project_id: m.project_id,
        project_title: p?.title ?? '',
        project_status: p?.status ?? 'ACTIVE',
        project_description: p?.description ?? null,
        stage: m.stage,
        score: m.score,
        assigned_user_id: m.assigned_user_id,
        last_contact_at: m.last_contact_at,
        next_action: m.next_action,
        assigned_at: m.assigned_at,
        updated_at: m.updated_at,
      }
    })
    return { ...l, projects: ms }
  },
  async createLead(data) {
    await wait(200)
    leadSeq++
    const l: Lead = {
      id: `lead-${leadSeq}`,
      company_name: data.company_name ?? 'سرنخ جدید',
      display_name: data.display_name ?? null,
      business_type: data.business_type ?? null,
      industry: data.industry ?? null,
      description: data.description ?? null,
      province: data.province ?? null,
      city: data.city ?? null,
      address: data.address ?? null,
      website: data.website ?? null,
      instagram: data.instagram ?? null,
      source: data.source ?? 'دستی',
      source_batch_id: null,
      source_document_id: null,
      status: 'NEW',
      quality_status: 'RAW',
      score: 0,
      owner_user_id: null,
      owner: null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      archived_at: null,
      tags: [],
    }
    leads.unshift(l)
    return l
  },
  async updateLead(id, patch) {
    await wait(150)
    const l = leads.find((x) => x.id === id)
    if (!l) throw new Error('NOT_FOUND')
    Object.assign(l, patch)
    return l
  },
  async leadContacts(id) {
    await wait(100)
    const l = leads.find((x) => x.id === id)
    const extra: Contact[] = []
    if (l?.primary_contact) extra.push(l.primary_contact)
    return [
      ...extra,
      ...Array.from({ length: randInt(0, 3) }, (_, i) => ({
        id: `contact-${id}-${i}`,
        lead_id: id,
        full_name: `${pick(firstNames)} ${pick(lastNames)}`,
        job_title: pick(jobTitles),
        mobile: `0912${String(randInt(1000000, 9999999))}`,
        is_decision_maker: i === 0,
        is_primary_contact: false,
        created_at: daysAgo(20),
        updated_at: daysAgo(2),
      })),
    ]
  },
  async createContact(leadId, contact) {
    await wait(150)
    return { id: `contact-${randInt(1000, 9999)}`, lead_id: leadId, is_decision_maker: false, is_primary_contact: false, created_at: new Date().toISOString(), updated_at: new Date().toISOString(), ...contact } as Contact
  },
  async updateContact(leadId, contactId, patch) {
    await wait(150)
    return { id: contactId, lead_id: leadId, ...patch } as Contact
  },
  async archiveContact() {
    await wait(100)
  },
  async leadEvidence(id) {
    await wait(150)
    const fields = [
      { field_name: 'company_name', value: leads.find((l) => l.id === id)?.company_name ?? '—', confidence: 96 },
      { field_name: 'business_type', value: 'تولیدکننده', confidence: 94 },
      { field_name: 'industry', value: 'پلیمر و پلاستیک', confidence: 89 },
      { field_name: 'city', value: 'تهران', confidence: 91 },
      { field_name: 'phone', value: '021-91234567', confidence: 82 },
      { field_name: 'contact_name', value: 'مدیر فروش', confidence: 71 },
    ]
    return {
      items: fields.map((f, i) => ({
        field_name: f.field_name,
        value: f.value,
        confidence: f.confidence,
        source: `لیست تولیدکنندگان / صفحه ${i + 1}`,
        excerpt: `"گروه صنعتی شامل ${f.value} با قابلیت تولید بالا در منطقه صنعتی..."`,
        page: i + 1,
        row: (i + 1) * 7,
      })),
    }
  },
  async leadScore(id) {
    await wait(150)
    const lead = leads.find((l) => l.id === id)
    const total = lead?.score ?? 70
    const dims = [
      { key: 'business_fit', name: 'business_fit', weight: 25, max: 25 },
      { key: 'contactability', name: 'contactability', weight: 15, max: 15 },
      { key: 'scale', name: 'scale', weight: 10, max: 10 },
      { key: 'engagement', name: 'engagement', weight: 20, max: 20 },
      { key: 'authority', name: 'authority', weight: 10, max: 10 },
      { key: 'need', name: 'need', weight: 10, max: 10 },
      { key: 'timing', name: 'timing', weight: 10, max: 10 },
    ]
    const contributions = dims.map((d) => ({ ...d, value: Math.round((d.weight / 100) * total), contribution: Math.round((d.weight / 100) * total) }))
    return { total, max: 100, components: contributions }
  },
  async leadTimeline(id) {
    await wait(150)
    const l = leads.find((x) => x.id === id)
    const events: TimelineEvent[] = [
      { id: `${id}-t1`, lead_id: id, event_type: 'lead.created', title: 'lead.created', created_at: l?.created_at ?? daysAgo(10), actor: null },
    ]
    if (l?.owner_user_id) {
      events.push({
        id: `${id}-t2`,
        lead_id: id,
        event_type: 'lead.assigned',
        title: 'lead.assigned',
        created_at: daysAgo(9),
        actor_user_id: 'u-manager',
        actor: { id: 'u-manager', name: 'سارا رضایی' },
        description: l.owner?.name,
      })
    }
    if (['CONTACTING', 'FOLLOW_UP', 'QUALIFIED'].includes(l?.status ?? '')) {
      events.push({ id: `${id}-t3`, lead_id: id, event_type: 'interaction.created', title: 'interaction.created', created_at: daysAgo(4), actor: l?.owner ? { id: l.owner.id, name: l.owner.name } : null, description: 'تماس اولیه انجام شد' })
      events.push({ id: `${id}-t4`, lead_id: id, event_type: 'report.submitted', title: 'report.submitted', created_at: daysAgo(4), actor: l?.owner ? { id: l.owner.id, name: l.owner.name } : null, description: 'گزارش تماس ثبت شد' })
    }
    if (l?.status === 'QUALIFIED') {
      events.push({ id: `${id}-t5`, lead_id: id, event_type: 'lead.qualified', title: 'lead.qualified', created_at: daysAgo(1), actor: { id: 'u-manager', name: 'سارا رضایی' } })
    }
    events.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    return events
  },
  async leadVersions(id, page = 1, pageSize = 50) {
    await wait(120)
    const l = leads.find((x) => x.id === id)
    const base = l?.company_name ?? 'شرکت'
    const versions: LeadVersion[] = [
      {
        id: `${id}-v3`, lead_id: id, version: 3,
        data: { lead: { company_name: base, city: l?.city ?? 'تهران' }, contacts: [], tags: [] },
        changes: [
          { field: 'contact.c1.is_primary_contact', old: false, new: true },
          { field: 'contact.c1.full_name', old: 'کارشناس فروش', new: 'رضا محمدی' },
        ],
        changed_by: 'u-agent1', changed_by_role: 'sales_agent',
        note: 'نام مسئول خرید مشخص شد', created_at: daysAgo(1),
        actor: { id: 'u-agent1', name: 'علی محمدی' },
      },
      {
        id: `${id}-v2`, lead_id: id, version: 2,
        data: { lead: { company_name: base, city: l?.city ?? 'تهران' }, contacts: [], tags: [] },
        changes: [{ field: 'city', old: 'کرج', new: l?.city ?? 'تهران' }],
        changed_by: 'u-manager', changed_by_role: 'sales_development_manager',
        note: '', created_at: daysAgo(5),
        actor: { id: 'u-manager', name: 'سارا رضایی' },
      },
      {
        id: `${id}-v1`, lead_id: id, version: 1,
        data: { lead: { company_name: base, city: l?.city ?? 'تهران' }, contacts: [], tags: [] },
        changes: [{ field: 'lead.created', new: base }],
        changed_by: null, changed_by_role: null,
        note: null, created_at: l?.created_at ?? daysAgo(10),
        actor: null,
      },
    ]
    const start = (page - 1) * pageSize
    return paginate(versions.slice(start, start + pageSize), page, pageSize)
  },
  async leadVersion(id, versionId) {
    await wait(100)
    const { items } = await mockApi.leadVersions(id, 1, 500)
    const v = items.find((x) => x.id === versionId)
    if (!v) throw new Error('NOT_FOUND')
    return v
  },
  async recentLeadChanges(limit = 50, offset = 0) {
    await wait(150)
    const items: ChangeHistoryItem[] = leads.slice(0, Math.min(limit, leads.length)).map((l, i) => ({
      id: `${l.id}-rc${i}`, lead_id: l.id, version: i + 3, lead_name: l.company_name,
      data: { lead: { company_name: l.company_name, city: l.city }, contacts: [], tags: [] },
      changes: [
        { field: 'status', old: 'CONTACTING', new: l.status },
        { field: 'contact.c1.is_primary_contact', old: false, new: true },
      ],
      changed_by: l.owner?.id ?? null, changed_by_role: 'sales_agent',
      note: null, created_at: hoursAgo(i + 1),
      actor: l.owner ?? null,
    }))
    return onePage(items.slice(offset, offset + limit))
  },
  async leadInteractions(id) {
    await wait(100)
    return interactions.filter((i) => i.lead_id === id)
  },
  async leadReports(id) {
    await wait(100)
    return reports.filter((r) => r.interaction?.lead_id === id)
  },
  async leadFollowups(id) {
    await wait(100)
    return followups.filter((f) => f.lead_id === id)
  },
  async leadNotes(id) {
    await wait(100)
    return [
      { id: `${id}-note1`, lead_id: id, author_id: 'u-manager', text: 'مشتری به پیشنهاد قیمت علاقه نشان داد.', created_at: daysAgo(2), updated_at: daysAgo(2), author: { id: 'u-manager', name: 'سارا رضایی' } },
      { id: `${id}-note2`, lead_id: id, author_id: 'u-agent1', text: 'تماس بعدی بعد از ارسال کاتالوگ.', created_at: daysAgo(1), updated_at: daysAgo(1), author: { id: 'u-agent1', name: 'علی محمدی' } },
    ]
  },
  async addNote(id, text) {
    await wait(150)
    return { id: `note-${randInt(1000, 9999)}`, lead_id: id, author_id: 'u-manager', text, created_at: new Date().toISOString(), updated_at: new Date().toISOString() }
  },
  async addFollowUp(leadId, data) {
    await wait(150)
    const fu: FollowUp = {
      id: `fu-${randInt(1000, 9999)}`,
      lead_id: leadId,
      due_at: data.due_at,
      type: data.type ?? 'FOLLOW_UP',
      title: data.title ?? null,
      description: data.description,
      priority: data.priority ?? 'MEDIUM',
      status: 'PENDING',
      source: 'MANUAL',
      created_at: new Date().toISOString(),
      created_by: 'u-manager',
      lead: { id: leadId, company_name: leads.find((l) => l.id === leadId)?.company_name ?? '' },
    }
    followups.push(fu)
    return fu
  },
  async createFollowUp(data) {
    await wait(150)
    const leadId = data.lead_id
    const fu: FollowUp = {
      id: `fu-${randInt(1000, 9999)}`,
      lead_id: leadId,
      due_at: data.due_at,
      type: data.type ?? 'FOLLOW_UP',
      title: data.title ?? null,
      description: data.description,
      priority: data.priority ?? 'MEDIUM',
      status: 'PENDING',
      source: 'MANUAL',
      created_at: new Date().toISOString(),
      created_by: 'u-manager',
      lead: { id: leadId, company_name: leads.find((l) => l.id === leadId)?.company_name ?? '' },
    }
    followups.push(fu)
    return fu
  },
  async updateFollowup(id, data) {
    await wait(150)
    const idx = followups.findIndex((f) => f.id === id)
    if (idx < 0) throw new Error('not found')
    followups[idx] = { ...followups[idx], ...data }
    return followups[idx]
  },
  async rescheduleFollowup(id, dueAt) {
    await wait(150)
    const idx = followups.findIndex((f) => f.id === id)
    if (idx < 0) throw new Error('not found')
    followups[idx] = { ...followups[idx], due_at: dueAt, status: 'PENDING', is_overdue: false, completed_at: null }
    return followups[idx]
  },
  async reopenFollowup(id) {
    await wait(150)
    const idx = followups.findIndex((f) => f.id === id)
    if (idx < 0) throw new Error('not found')
    followups[idx] = { ...followups[idx], status: 'PENDING', is_overdue: false, completed_at: null }
    return followups[idx]
  },
  async cancelFollowup(id) {
    await wait(150)
    const idx = followups.findIndex((f) => f.id === id)
    if (idx < 0) throw new Error('not found')
    followups[idx] = { ...followups[idx], status: 'CANCELLED', is_overdue: false }
    return followups[idx]
  },
  async assignLeads(leadIds, assigneeId) {
    await wait(200)
    const a = users.find((u) => u.id === assigneeId)
    for (const id of leadIds) {
      const l = leads.find((x) => x.id === id)
      if (l) {
        l.owner_user_id = assigneeId
        l.owner = a ? { id: a.id, name: a.name } : null
        if (l.status === 'NEW' || l.status === 'READY') l.status = 'ASSIGNED'
      }
    }
  },
  async qualifyLead(id) {
    await wait(200)
    const l = leads.find((x) => x.id === id)
    if (!l) throw new Error('NOT_FOUND')
    l.status = 'QUALIFIED'
    l.qualification = { minimum_score_met: true, business_relevance: true, meaningful_contact: true, decision_maker: true, need_interest: true, next_action: true, decided_at: new Date().toISOString(), decided_by: 'u-manager' }
    return { qualified: true, lead_id: id }
  },
  async disqualifyLead(id, reason) {
    await wait(200)
    const l = leads.find((x) => x.id === id)
    if (!l) throw new Error('NOT_FOUND')
    l.status = 'DISQUALIFIED'
    l.qualification = { minimum_score_met: false, business_relevance: false, meaningful_contact: false, decision_maker: false, need_interest: false, next_action: false, decided_at: new Date().toISOString(), decided_by: 'u-manager', disqualify_reason: reason }
    return { disqualified: true, lead_id: id }
  },
  async exportLeadToMuninn(id) {
    await wait(300)
    const l = leads.find((x) => x.id === id)
    if (!l) throw new Error('NOT_FOUND')
    if (l.status === 'EXPORTED_TO_MUNINN' || l.crm_export?.status === 'SUCCEEDED') {
      const err = new Error('DUPLICATE_EXPORT') as Error & { code: string }
      err.code = 'DUPLICATE_EXPORT'
      throw err
    }
    l.status = 'EXPORTED_TO_MUNINN'
    const exp: CrmExport = { id: `exp-${randInt(1000, 9999)}`, huginn_lead_id: id, muninn_lead_id: `MN-${randInt(1000, 9999)}`, status: 'PENDING', attempt_count: 0, last_error: null, exported_at: null, created_at: new Date().toISOString(), updated_at: new Date().toISOString() }
    l.crm_export = exp
    return exp
  },
  async getCrmExport(leadId) {
    await wait(100)
    return leads.find((l) => l.id === leadId)?.crm_export ?? null
  },
  async archiveLead(id) {
    await wait(150)
    const l = leads.find((x) => x.id === id)
    if (!l) throw new Error('NOT_FOUND')
    l.status = 'ARCHIVED'
    l.archived_at = new Date().toISOString()
  },

  async listProjects(q) {
    await wait(250)
    let items = mockProjects
    const status = q?.status
    if (status) items = items.filter((p) => p.status === status)
    const withCounts = items.map((p) => ({ ...p, lead_count: leads.filter((l) => l.project_id === p.id).length }))
    return paginate(withCounts, q?.page ?? 1, q?.page_size ?? 20)
  },
  async getProject(id) {
    await wait(200)
    const p = mockProjects.find((x) => x.id === id)
    if (!p) throw new Error('NOT_FOUND')
    const related = leads.filter((l) => l.project_id === id || (id === mockProjects[0]?.id && l.id.endsWith('1')))
    const activities: ProjectDetail['sales_activities'] = []
    return { project: p, lead_count: related.length, related_leads: { leads: related.slice(0, 50), total: related.length, page: 1, size: 50 }, sales_activities: activities } as ProjectDetail
  },
  async createProject(data) {
    await wait(200)
    const p: Project = {
      id: `proj-${mockProjects.length + 1}`,
      title: data.title ?? '',
      description: data.description ?? '',
      product_name: data.product_name ?? null,
      product_description: data.product_description ?? null,
      target_market: data.target_market ?? null,
      ideal_customer_profile: data.ideal_customer_profile ?? null,
      customer_pain_points: data.customer_pain_points ?? null,
      value_proposition: data.value_proposition ?? null,
      sales_objective: data.sales_objective ?? null,
      common_objections: data.common_objections ?? null,
      status: 'ACTIVE',
      created_by: users[0].id,
      campaign_id: data.campaign_id ?? null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    mockProjects.push(p)
    return { project: p, lead_count: 0, related_leads: { leads: [], total: 0, page: 1, size: 5 }, sales_activities: [] } as ProjectDetail
  },
  async updateProject(id, patch) {
    await wait(200)
    const p = mockProjects.find((x) => x.id === id)
    if (!p) throw new Error('NOT_FOUND')
    Object.assign(p, patch, { updated_at: new Date().toISOString() })
    return { project: p, lead_count: membershipRows(id).length, related_leads: { leads: [], total: 0, page: 1, size: 5 }, sales_activities: [] } as ProjectDetail
  },
  async deleteProject(id) {
    await wait(200)
    const idx = mockProjects.findIndex((x) => x.id === id)
    if (idx === -1) throw new Error('NOT_FOUND')
    mockProjects.splice(idx, 1)
  },
  async listProjectLeads(projectId, q) {
    await wait(200)
    const rows = membershipRows(projectId).map((m) => {
      const lead = leads.find((l) => l.id === m.lead_id)
      return {
        id: m.id, project_id: m.project_id, lead_id: m.lead_id, stage: m.stage, score: m.score,
        assigned_user_id: m.assigned_user_id, last_contact_at: m.last_contact_at, next_action: m.next_action,
        assigned_at: m.assigned_at, created_at: m.created_at, updated_at: m.updated_at,
        company_name: lead?.company_name ?? '', industry: lead?.industry ?? null, city: lead?.city ?? null,
        province: lead?.province ?? null, lead_status: lead?.status ?? 'NEW', lead_score: lead?.score ?? 0,
        owner_user_id: lead?.owner_user_id ?? null, owner_name: lead?.owner?.name ?? null,
      } as ProjectLeadRow
    })
    return paginate(rows, q?.page ?? 1, q?.page_size ?? 50)
  },
  async attachProjectLeads(projectId, leadIds) {
    await wait(200)
    const attached = attachLeadMemberships(projectId, leadIds)
    return { attached, project_id: projectId }
  },
  async attachProjectLead(projectId, leadId) {
    await wait(150)
    if (!mockProjects.some((p) => p.id === projectId)) throw new Error('NOT_FOUND')
    if (!leads.some((l) => l.id === leadId)) throw new Error('NOT_FOUND')
    attachMembership(projectId, leadId)
    return { success: true, project_id: projectId }
  },
  async detachProjectLead(projectId, leadId) {
    await wait(150)
    const idx = memberships.findIndex((m) => m.project_id === projectId && m.lead_id === leadId)
    if (idx !== -1) memberships.splice(idx, 1)
    return { success: true }
  },
  async updateProjectLead(projectId, leadId, patch) {
    await wait(150)
    const m = membershipOf(projectId, leadId)
    if (!m) throw new Error('NOT_FOUND')
    Object.assign(m, patch, { updated_at: new Date().toISOString() })
    const lead = leads.find((l) => l.id === leadId)
    return { ...m, company_name: lead?.company_name ?? '', industry: lead?.industry ?? null, city: lead?.city ?? null, lead_status: lead?.status ?? 'NEW', lead_score: lead?.score ?? 0 } as ProjectLeadRow
  },

  async listCampaigns(q) {
    await wait(200)
    let items = mockCampaigns
    if (q?.status) items = items.filter((c) => c.status === q.status)
    return paginate(items, q?.page ?? 1, q?.page_size ?? 20)
  },
  async getCampaign(id) {
    await wait(200)
    const c = mockCampaigns.find((x) => x.id === id)
    if (!c) throw new Error('NOT_FOUND')
    return { ...c, projects: mockProjects.filter((p) => p.campaign_id === id).map((p) => ({ id: p.id, title: p.title, status: p.status, created_at: p.created_at })) }
  },
  async createCampaign(data) {
    await wait(200)
    const c: Campaign = {
      id: `camp-${mockCampaigns.length + 1}`, title: data.title ?? '', description: data.description ?? null,
      objective: data.objective ?? null, status: 'ACTIVE', created_by: users[0].id,
      created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
    }
    mockCampaigns.push(c)
    return c
  },
  async updateCampaign(id, patch) {
    await wait(200)
    const c = mockCampaigns.find((x) => x.id === id)
    if (!c) throw new Error('NOT_FOUND')
    Object.assign(c, patch, { updated_at: new Date().toISOString() })
    return c
  },
  async deleteCampaign(id) {
    await wait(200)
    const idx = mockCampaigns.findIndex((x) => x.id === id)
    if (idx === -1) throw new Error('NOT_FOUND')
    mockCampaigns.splice(idx, 1)
  },
  async attachLeadToProject(projectId, leadId) {
    await wait(150)
    if (!mockProjects.some((p) => p.id === projectId)) throw new Error('NOT_FOUND')
    const l = leads.find((x) => x.id === leadId)
    if (!l) throw new Error('NOT_FOUND')
    l.project_id = projectId
    l.updated_at = new Date().toISOString()
    return { success: true, lead_id: leadId, project_id: projectId }
  },
  async detachLeadFromProject(projectId, leadId) {
    await wait(150)
    if (!mockProjects.some((p) => p.id === projectId)) throw new Error('NOT_FOUND')
    const l = leads.find((x) => x.id === leadId)
    if (!l) throw new Error('NOT_FOUND')
    l.project_id = null
    l.updated_at = new Date().toISOString()
    return { success: true, lead_id: leadId, project_id: null }
  },

  async listImports(q) {
    await wait(250)
    return paginate(batches, q?.page ?? 1, q?.page_size ?? 20)
  },
  async getImport(id) {
    await wait(200)
    const b = batches.find((x) => x.id === id)
    if (!b) throw new Error('NOT_FOUND')
    return b
  },
  async createImportBatch(data) {
    await wait(300)
    const b: ImportBatch = {
      id: `batch-${randInt(100, 999)}`,
      name: data.name,
      description: data.description,
      source_type: data.source_type as ImportBatch['source_type'],
      status: 'UPLOADED',
      created_by: 'u-manager',
      creator: { id: 'u-manager', name: 'سارا رضایی' },
      created_at: new Date().toISOString(),
      total_documents: 0,
      total_candidates: 0,
      created_leads: 0,
      duplicate_candidates: 0,
      failed_items: 0,
      project_id: data.project_id ?? null,
      progress: 0,
      timeline: [{ key: 'uploaded', label: 'uploaded', status: 'active', started_at: new Date().toISOString() }],
    }
    batches.unshift(b)
    return b
  },
  async uploadFiles(batchId) {
    await wait(400)
    const b = batches.find((x) => x.id === batchId)
    if (b) b.status = 'QUEUED'
    return { uploaded: 1 }
  },
  async importDocuments(batchId) {
    await wait(150)
    return paginate(documents.filter((d) => d.batch_id === batchId), 1, 100)
  },

  async listDocuments(q) {
    await wait(250)
    let items = documents
    if (q?.batch_id) items = items.filter((d) => d.batch_id === q.batch_id)
    return paginate(items, q?.page ?? 1, q?.page_size ?? 20)
  },
  async getDocument(id) {
    await wait(200)
    const d = documents.find((x) => x.id === id)
    if (!d) throw new Error('NOT_FOUND')
    return { ...d, candidates: [] } as DocumentDetail
  },
  async documentCandidates(id) {
    await wait(150)
    return Array.from({ length: randInt(2, 8) }, (_, i) => ({
      id: `cand-${id}-${i}`,
      lead_id: i % 3 === 0 ? `lead-${randInt(1, 50)}` : null,
      company_name: pick(companies),
      contact_name: `${pick(firstNames)} ${pick(lastNames)}`,
      phone: `0912${String(randInt(1000000, 9999999))}`,
      score: randInt(40, 95),
      status: (i % 3 === 0 ? 'duplicate' : i % 3 === 1 ? 'created' : 'new') as LeadCandidate['status'],
    }))
  },

  async listDuplicates(q) {
    await wait(250)
    let items = duplicates
    if (q?.status) items = items.filter((d) => d.status === q.status)
    return paginate(items, q?.page ?? 1, q?.page_size ?? 20)
  },
  async markDuplicate(id, action) {
    await wait(150)
    const d = duplicates.find((x) => x.id === id)
    if (d) d.status = action === 'merge' ? 'MERGED' : action === 'distinct' ? 'DISTINCT' : 'POSSIBLE'
  },
  async mergeDuplicates(id, survivingLeadId) {
    await wait(250)
    const d = duplicates.find((x) => x.id === id)
    if (d) d.status = 'MERGED'
    return leads.find((l) => l.id === survivingLeadId) ?? leads[0]
  },

  async assignmentOverview() {
    await wait(250)
    const unassignedList = leads.filter((l) => !l.owner_user_id && l.status !== 'DISQUALIFIED' && l.status !== 'ARCHIVED' && l.status !== 'EXPORTED_TO_MUNINN')
    const employees: WorkloadItem[] = [users[2], users[3], users[4]].map((u) => ({
      user_id: u.id,
      name: u.name,
      current_load: leads.filter((l) => l.owner_user_id === u.id).length,
      today_load: randInt(5, 15),
      qualification_performance: randInt(25, 70),
    }))
    return { unassigned_leads: unassignedList.length, employees, unassigned_list: paginate(unassignedList, 1, 100) }
  },
  async assignBulk(leadIds, assigneeId) {
    await this.assignLeads(leadIds, assigneeId, 'bulk')
  },
  async assignRoundRobin(leadIds, assigneeIds) {
    await wait(200)
    leadIds.forEach((id, i) => {
      const a = users.find((u) => u.id === assigneeIds[i % assigneeIds.length])
      const l = leads.find((x) => x.id === id)
      if (l && a) {
        l.owner_user_id = a.id
        l.owner = { id: a.id, name: a.name }
      }
    })
  },

  async listAgents() {
    await wait(200)
    return [users[2], users[3], users[4]].map((u) => ({
      user_id: u.id,
      name: u.name,
      assigned_leads: leads.filter((l) => l.owner_user_id === u.id).length,
      calls_today: randInt(8, 20),
      successful_contacts: randInt(4, 12),
      pending_reports: randInt(0, 6),
      overdue_followups: randInt(0, 4),
      qualified_leads: leads.filter((l) => l.owner_user_id === u.id && l.status === 'QUALIFIED').length,
      qualification_rate: randInt(25, 70),
    }))
  },
  async getAgent(id) {
    await wait(200)
    const u = users.find((x) => x.id === id)
    return {
      user: u ? { id: u.id, name: u.name } : { id, name: 'کارشناس' },
      assigned_leads: leads.filter((l) => l.owner_user_id === id).length,
      calls_today: randInt(8, 20),
      successful_contacts: randInt(4, 12),
      pending_reports: randInt(0, 6),
      overdue_followups: randInt(0, 4),
      qualified_leads: randInt(5, 30),
      qualification_rate: randInt(25, 70),
      queue: leads.filter((l) => l.owner_user_id === id).slice(0, 10),
      recent_activity: [users[2], users[3], users[4]].map((_, i) => ({
        id: `act-${i}`,
        lead_id: leads[i % leads.length].id,
        event_type: 'call',
        title: 'lead.call',
        created_at: daysAgo(i),
        actor_user_id: id,
        actor: { id, name: u?.name ?? '' },
        description: 'تماس با مشتری',
      })),
    }
  },

  async listReports(q: ReportsQuery) {
    await wait(250)
    let items = reports
    if (q?.status) items = items.filter((r) => (q.status === 'SUBMITTED' ? r.manager_override === false : q.status === 'OVERRIDE' ? r.manager_override : true))
    if (q?.user_id) items = items.filter((r) => r.interaction?.user_id === q.user_id)
    return paginate(items, q?.page ?? 1, q?.page_size ?? 20)
  },
  async getReport(id) {
    await wait(200)
    const r = reports.find((x) => x.id === id)
    if (!r) throw new Error('NOT_FOUND')
    return r
  },
  async overrideReport(id, reason) {
    await wait(200)
    const r = reports.find((x) => x.id === id)
    if (r) {
      r.manager_override = true
      r.manager_override_reason = reason
      r.overridden_by = 'u-manager'
    }
    return r!
  },

  async listFollowups(view) {
    await wait(250)
    let items = followups
    if (view === 'today') items = items.filter((f) => f.status === 'PENDING' && !f.is_overdue)
    if (view === 'overdue') items = items.filter((f) => f.status === 'PENDING' && f.is_overdue)
    if (view === 'upcoming') items = items.filter((f) => f.status === 'PENDING' && !f.is_overdue && new Date(f.due_at) > new Date())
    if (view === 'completed') items = items.filter((f) => f.status === 'COMPLETED')
    return paginate(items, 1, 50)
  },
  async completeFollowup(id) {
    await wait(150)
    const f = followups.find((x) => x.id === id)
    if (f) {
      f.status = 'COMPLETED'
      f.completed_at = new Date().toISOString()
    }
    return f!
  },

  async nextActions(limit = 3) {
    await wait(150)
    const items = mockNextActions()
    return {
      items: items.slice(0, limit),
      generated_at: new Date().toISOString(),
      total_actionable: items.length,
      urgent_count: items.filter((i) => i.priority === 'urgent').length,
      overdue_count: items.filter((i) => i.overdue).length,
    }
  },
  async teamNextActions(limit = 200) {
    await wait(150)
    const items = mockNextActions(true)
    return {
      items: items.slice(0, limit),
      generated_at: new Date().toISOString(),
      total_actionable: items.length,
      urgent_count: items.filter((i) => i.priority === 'urgent').length,
      overdue_count: items.filter((i) => i.overdue).length,
    }
  },
  async snoozeNextAction(id, until) {
    await wait(120)
    const fuId = id.split(':')[1]
    const f = followups.find((x) => x.id === fuId)
    if (f) f.due_at = until
  },
  async completeNextAction(id) {
    await wait(120)
    const fuId = id.split(':')[1]
    const f = followups.find((x) => x.id === fuId)
    if (f) f.status = 'COMPLETED'
  },

  async listUsers() {
    await wait(200)
    return users
  },
  async getUser(id) {
    await wait(100)
    return users.find((u) => u.id === id) ?? users[0]
  },
  async createUser(data) {
    await wait(200)
    const u: User = { id: `u-${randInt(100, 999)}`, name: data.name, username: data.username, email: data.email, mobile: data.mobile, role: data.role as User['role'], is_active: true, created_at: new Date().toISOString(), updated_at: new Date().toISOString() }
    users.push(u)
    return u
  },
  async updateUser(id, patch) {
    await wait(200)
    const u = users.find((x) => x.id === id)
    if (u) Object.assign(u, patch)
    return u!
  },
  async setUserActive(id, active) {
    await wait(150)
    const u = users.find((x) => x.id === id)
    if (u) u.is_active = active
  },
  async resetUserPassword() {
    await wait(150)
  },
  async listUserSessions(id) {
    await wait(150)
    return Array.from({ length: randInt(1, 3) }, (_, i) => ({
      id: `sess-${id}-${i}`,
      user_id: id,
      device: i === 0 ? 'Chrome / Windows' : 'Android / Huginn App',
      ip: `192.168.${randInt(1, 20)}.${randInt(2, 250)}`,
      user_agent: 'Mozilla/5.0',
      created_at: daysAgo(randInt(1, 20)),
      last_used_at: daysAgo(randInt(0, 2)),
      expires_at: future(20),
    }))
  },
  async revokeSession() {
    await wait(100)
  },

  async getScoringSettings(): Promise<ScoringSettings> {
    await wait(150)
    const dims = [
      { key: 'business_fit', name: 'business_fit', weight: 25, value: 0, contribution: 0, max: 25 },
      { key: 'contactability', name: 'contactability', weight: 15, value: 0, contribution: 0, max: 15 },
      { key: 'scale', name: 'scale', weight: 10, value: 0, contribution: 0, max: 10 },
      { key: 'engagement', name: 'engagement', weight: 20, value: 0, contribution: 0, max: 20 },
      { key: 'authority', name: 'authority', weight: 10, value: 0, contribution: 0, max: 10 },
      { key: 'need', name: 'need', weight: 10, value: 0, contribution: 0, max: 10 },
      { key: 'timing', name: 'timing', weight: 10, value: 0, contribution: 0, max: 10 },
    ]
    return { dimensions: dims, total_weight: 100 }
  },
  async updateScoringSettings(weights) {
    await wait(200)
    const s = await this.getScoringSettings()
    s.dimensions = s.dimensions.map((d) => ({ ...d, weight: weights[d.key] ?? d.weight }))
    s.total_weight = s.dimensions.reduce((a, b) => a + b.weight, 0)
    return s
  },
  async getQualificationSettings(): Promise<QualificationSettings> {
    await wait(150)
    return { minimum_score: 70, business_relevance_required: true, meaningful_contact_required: true, decision_maker_required: false, need_interest_required: true, next_action_required: true }
  },
  async updateQualificationSettings(patch) {
    await wait(200)
    return { ...(await this.getQualificationSettings()), ...patch }
  },
  async getLlmSettings(): Promise<LLMSettings> {
    await wait(150)
    return { provider: 'deepseek', model: 'deepseek-v4-flash', base_url: 'https://api.deepseek.com', timeout_seconds: 60, prompt_version: 'v3.2.1', structured_output_enabled: true, max_retries: 3 }
  },
  async updateLlmSettings(patch) {
    await wait(200)
    return { ...(await this.getLlmSettings()), ...patch }
  },
  async testLlmConnection() {
    await wait(600)
    return { ok: true, message: 'ok' }
  },
  async getMuninnSettings(): Promise<MuninnSettings> {
    await wait(150)
    return { enabled: true, base_url: 'https://muninn.example.com', timeout_seconds: 30, export_endpoint: '/api/v1/leads' }
  },
  async updateMuninnSettings(patch) {
    await wait(200)
    return { ...(await this.getMuninnSettings()), ...patch }
  },
  async getSystemConfig(): Promise<SystemConfig> {
    await wait(150)
    return { app_name: 'Huginn', version: '0.1.0', timezone: 'Asia/Tehran', max_upload_bytes: 25 * 1024 * 1024, storage_engine: 'MinIO', queue: 'RabbitMQ', worker_count: 2 }
  },

  async listAudit(q: AuditQuery) {
    await wait(250)
    let items = auditEntries
    if (q?.user_id) items = items.filter((e) => e.actor_user_id === q.user_id)
    if (q?.action) items = items.filter((e) => e.action.includes(q.action as string))
    if (q?.entity_type) items = items.filter((e) => e.entity_type === q.entity_type)
    return paginate(items, q?.page ?? 1, q?.page_size ?? 20)
  },

  async notifications() {
    await wait(150)
    return notifications
  },
  async markNotificationsRead() {
    await wait(100)
    notifications.forEach((n) => (n.read = true))
  },
  async globalSearch(q): Promise<GlobalSearchResults> {
    await wait(200)
    if (!q) return { leads: [], users: [] }
    const s = q.toLowerCase()
    return {
      leads: leads.filter((l) => l.company_name.toLowerCase().includes(s) || l.city?.toLowerCase().includes(s) || l.primary_contact?.full_name?.toLowerCase().includes(s) || l.primary_contact?.mobile?.includes(s) || (l.tags ?? []).some((t) => t.toLowerCase().includes(s))).slice(0, 6).map((l) => ({ id: l.id, company_name: l.company_name, city: l.city, status: l.status, score: l.score })),
      users: users.filter((u) => u.name.toLowerCase().includes(s) || u.username.toLowerCase().includes(s)).slice(0, 4).map((u) => ({ id: u.id, name: u.name, username: u.username })),
    }
  },
  async backendHealth(): Promise<BackendHealth> {
    await wait(150)
    return { status: 'ok', time: new Date().toISOString() }
  },
  async jobsPending() {
    await wait(100)
    return { count: batches.filter((b) => b.status === 'PROCESSING' || b.status === 'QUEUED').length }
  },
}
