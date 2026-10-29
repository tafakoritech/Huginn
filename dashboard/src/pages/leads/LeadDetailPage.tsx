import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { useAuth, PERMISSIONS } from '@/auth/AuthContext'
import { formatDateTime, toFaDigits } from '@/utils/format'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { CreateFollowUpDialog } from '@/components/followups/CreateFollowUpDialog'
import { Icon, type IconName } from '@/components/Icon'
import { ScoreBadge, StatusBadge, ConfidenceBadge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Tabs, Avatar, Chip, BackButton, EmptyValue } from '@/components/ui/misc'
import { ConfirmDialog, Dialog } from '@/components/ui/Dialog'
import { LoadingState, ErrorState } from '@/components/ui/States'
import { Select, Textarea, Input } from '@/components/ui/Field'
import { useClickOutside } from '@/hooks'
import type { Contact, Lead, LeadChange, LeadVersion } from '@/api/types'

export function LeadDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t, locale } = useI18n()
  const { can } = useAuth()
  const toast = useToastApi()
  const qc = useQueryClient()
  const [tab, setTab] = useState('overview')
  const [showActions, setShowActions] = useState(false)
  const [disqualifyOpen, setDisqualifyOpen] = useState(false)
  const [disqualifyReason, setDisqualifyReason] = useState('')
  const [exportOpen, setExportOpen] = useState(false)
  const [followupOpen, setFollowupOpen] = useState(false)
  const [noteText, setNoteText] = useState('')
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const { data: lead, isLoading, isError, refetch } = useQuery({
    queryKey: queryKeys.lead(id ?? ''),
    queryFn: () => api.getLead(id ?? ''),
    enabled: !!id,
  })

  const { data: contacts = [] } = useQuery({
    queryKey: queryKeys.leadContacts(id ?? ''),
    queryFn: () => api.leadContacts(id ?? ''),
    enabled: !!id,
  })

  const { data: evidence } = useQuery({
    queryKey: queryKeys.leadEvidence(id ?? ''),
    queryFn: () => api.leadEvidence(id ?? ''),
    enabled: !!id,
  })

  const { data: score } = useQuery({
    queryKey: queryKeys.leadScore(id ?? ''),
    queryFn: () => api.leadScore(id ?? ''),
    enabled: !!id,
  })

  const { data: timeline = [] } = useQuery({
    queryKey: queryKeys.leadTimeline(id ?? ''),
    queryFn: () => api.leadTimeline(id ?? ''),
    enabled: !!id,
  })

  const { data: versions } = useQuery({
    queryKey: queryKeys.leadVersions(id ?? ''),
    queryFn: () => api.leadVersions(id ?? ''),
    enabled: !!id && can(PERMISSIONS.leadsHistory),
  })

  const { data: interactions = [] } = useQuery({
    queryKey: queryKeys.leadInteractions(id ?? ''),
    queryFn: () => api.leadInteractions(id ?? ''),
    enabled: !!id,
  })

  const { data: reports = [] } = useQuery({
    queryKey: queryKeys.leadReports(id ?? ''),
    queryFn: () => api.leadReports(id ?? ''),
    enabled: !!id,
  })

  const { data: followups = [] } = useQuery({
    queryKey: queryKeys.leadFollowups(id ?? ''),
    queryFn: () => api.leadFollowups(id ?? ''),
    enabled: !!id,
  })

  const { data: notes = [] } = useQuery({
    queryKey: queryKeys.leadNotes(id ?? ''),
    queryFn: () => api.leadNotes(id ?? ''),
    enabled: !!id,
  })

  const { data: exportState } = useQuery({
    queryKey: queryKeys.leadExport(id ?? ''),
    queryFn: () => api.getCrmExport(id ?? ''),
    enabled: !!id,
  })

  const { data: users } = useQuery({ queryKey: queryKeys.users, queryFn: () => api.listUsers(), enabled: can(PERMISSIONS.leadsAssign) })
  const agents = useMemo(() => (users ?? []).filter((u) => u.role === 'sales_agent'), [users])

  const invalidateLead = () => {
    void qc.invalidateQueries({ queryKey: queryKeys.lead(id ?? '') })
    void qc.invalidateQueries({ queryKey: queryKeys.leadTimeline(id ?? '') })
    void qc.invalidateQueries({ queryKey: queryKeys.leadVersions(id ?? '') })
    void qc.invalidateQueries({ queryKey: queryKeys.leadExport(id ?? '') })
    void qc.invalidateQueries({ queryKey: queryKeys.leads() })
  }

  const qualify = useMutation({
    mutationFn: () => api.qualifyLead(id ?? ''),
    onSuccess: () => {
      toast.success(t('leads.qualifiedSuccess'))
      invalidateLead()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const disqualify = useMutation({
    mutationFn: () => api.disqualifyLead(id ?? '', disqualifyReason),
    onSuccess: () => {
      toast.success(t('leads.disqualifiedSuccess'))
      setDisqualifyOpen(false)
      setDisqualifyReason('')
      invalidateLead()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const exportMuninn = useMutation({
    mutationFn: () => api.exportLeadToMuninn(id ?? ''),
    onSuccess: () => {
      toast.success(t('leads.exportQueued'))
      setExportOpen(false)
      invalidateLead()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const assign = useMutation({
    mutationFn: (assigneeId: string) => api.assignLeads([id ?? ''], assigneeId, 'manual'),
    onSuccess: () => {
      toast.success(t('assignments.assignSuccess'))
      invalidateLead()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const updateLead = useMutation({
    mutationFn: (data: { patch: Partial<Lead>; note?: string }) => api.updateLead(id ?? '', data.patch, data.note),
    onSuccess: () => {
      toast.success(t('leads.updated'))
      setEditOpen(false)
      invalidateLead()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const deleteLead = useMutation({
    mutationFn: () => api.archiveLead(id ?? ''),
    onSuccess: () => {
      toast.success(t('leads.deleteSuccess'))
      setDeleteOpen(false)
      navigate('/leads')
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const completeFollowUp = useMutation({
    mutationFn: (fuId: string) => api.completeFollowup(fuId),
    onSuccess: () => {
      toast.success(t('followups.complete'))
      void qc.invalidateQueries({ queryKey: queryKeys.leadFollowups(id ?? '') })
      invalidateLead()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const reopenFollowUp = useMutation({
    mutationFn: (fuId: string) => api.reopenFollowup(fuId),
    onSuccess: () => {
      toast.success(t('followups.reopen'))
      void qc.invalidateQueries({ queryKey: queryKeys.leadFollowups(id ?? '') })
      invalidateLead()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const cancelFollowUp = useMutation({
    mutationFn: (fuId: string) => api.cancelFollowup(fuId),
    onSuccess: () => {
      toast.success(t('followups.cancelled'))
      void qc.invalidateQueries({ queryKey: queryKeys.leadFollowups(id ?? '') })
      invalidateLead()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const addNote = useMutation({
    mutationFn: () => api.addNote(id ?? '', noteText),
    onSuccess: () => {
      setNoteText('')
      void qc.invalidateQueries({ queryKey: queryKeys.leadNotes(id ?? '') })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const actionsRef = useClickOutside<HTMLDivElement>(() => setShowActions(false))

  if (isLoading || !lead) return <LoadingState />
  if (isError) return <ErrorState detail={t('errors.generic')} onRetry={() => void refetch()} />

  const canQualify = can(PERMISSIONS.leadsQualify) && ['NEW', 'READY', 'ASSIGNED', 'CONTACTING', 'FOLLOW_UP'].includes(lead.status)
  const canDisqualify = can(PERMISSIONS.leadsDisqualify) && !['DISQUALIFIED', 'ARCHIVED', 'EXPORTED_TO_MUNINN'].includes(lead.status)
  const canExport = can(PERMISSIONS.leadsExport) && lead.status === 'QUALIFIED'

  const tabs = [
    { key: 'overview', label: t('leads.overview') },
    { key: 'contacts', label: t('leads.contactsTab'), count: contacts.length },
    { key: 'history', label: t('leads.historyTab') },
    { key: 'interactions', label: t('leads.interactionsTab'), count: interactions.length },
    { key: 'reports', label: t('leads.reportsTab'), count: reports.length },
    { key: 'followups', label: t('leads.followupsTab'), count: followups.length },
    { key: 'evidence', label: t('leads.evidenceTab') },
    { key: 'score', label: t('leads.scoreTab') },
  ]
  if (can(PERMISSIONS.leadsHistory)) {
    tabs.push({ key: 'versions', label: t('leads.versionsTab'), count: versions?.meta.total })
  }

  return (
    <>
      <BackButton to="/leads" />

      <div className="lead-hero mt-3">
        <div>
          <div className="lead-hero-title">
            <h1>{lead.company_name}</h1>
            <StatusBadge status={lead.status} />
            <ScoreBadge score={lead.score} />
          </div>
          <div className="lead-meta-row">
            <span className="meta">
              <Icon name="user" />
              {lead.owner?.name ?? <EmptyValue />}
            </span>
            <span className="meta">
              <Icon name="mapPin" />
              {[lead.city, lead.province].filter(Boolean).join('، ') || <EmptyValue />}
            </span>
             <span className="meta">
               <Icon name="package" />
               {lead.business_type ?? <EmptyValue />}
             </span>
             <span className="meta">
               <Icon name="project" />
               {(lead.projects ?? []).length > 0 ? (
                 <Link to={`/projects/${lead.projects![0].project_id}`} className="strong" style={{ color: 'var(--text-1)' }}>
                   {lead.projects![0].project_title}
                   {(lead.projects ?? []).length > 1 ? ` +${toFaDigits(lead.projects!.length - 1)}` : ''}
                 </Link>
               ) : lead.project ? (
                 <Link to={`/projects/${lead.project.id}`} className="strong" style={{ color: 'var(--text-1)' }}>
                   {lead.project.title}
                 </Link>
               ) : (
                 <EmptyValue />
               )}
             </span>
            <span className="meta">
              <Icon name="layers" />
              {lead.industry ?? <EmptyValue />}
            </span>
            <span className="meta">
              <Icon name="calendar" />
              {formatDateTime(lead.created_at, locale, { time: false })}
            </span>
            <span className="meta">
              <Icon name="tag" />
              {lead.source ?? <EmptyValue />}
            </span>
          </div>
          {(lead.tags ?? []).length > 0 ? (
            <div className="flex gap-2 mt-3 wrap">
              {(lead.tags ?? []).map((tag) => (
                <Chip key={tag} accent>
                  {tag}
                </Chip>
              ))}
            </div>
          ) : null}
          {exportState && exportState.status === 'SUCCEEDED' ? (
            <div className="badge badge-brand mt-3">
              <Icon name="muninn" size={12} />
              Muninn ID: <span className="mono" dir="ltr">{exportState.muninn_lead_id}</span>
            </div>
          ) : null}
        </div>

        <div className="lead-hero-actions">
          {can(PERMISSIONS.leadsContact) ? (
            <Button icon="phone" onClick={() => toast.info(t('leads.actionDone'))}>
              {t('leads.contact')}
            </Button>
          ) : null}
          {can(PERMISSIONS.leadsContact) ? (
            <Button icon="message" variant="secondary" onClick={() => toast.info(t('leads.actionDone'))}>
              {t('leads.messageBtn')}
            </Button>
          ) : null}
          <Button icon="followups" variant="secondary" onClick={() => setFollowupOpen(true)}>
            {t('leads.followUp')}
          </Button>
          <div className="dropdown" ref={actionsRef}>
            <Button icon="more" variant="secondary" onClick={() => setShowActions((o) => !o)} aria-expanded={showActions}>
              {t('common.actions')}
            </Button>
            {showActions ? (
              <div className="menu" role="menu">
                {can(PERMISSIONS.leadsEdit) ? (
                  <button type="button" className="menu-item" role="menuitem" onClick={() => setEditOpen(true)}>
                    <Icon name="edit" />
                    {t('leads.edit')}
                  </button>
                ) : null}
                {can(PERMISSIONS.leadsAssign) ? (
                  <AssignMenuItem agents={agents} onAssign={(a) => assign.mutate(a)} />
                ) : null}
                {canQualify ? (
                  <button type="button" className="menu-item" role="menuitem" onClick={() => qualify.mutate()}>
                    <Icon name="checkCircle" />
                    {t('leads.qualify')}
                  </button>
                ) : null}
                {canDisqualify ? (
                  <button type="button" className="menu-item danger" role="menuitem" onClick={() => setDisqualifyOpen(true)}>
                    <Icon name="xCircle" />
                    {t('leads.disqualify')}
                  </button>
                ) : null}
                {canExport ? (
                  <button type="button" className="menu-item" role="menuitem" onClick={() => setExportOpen(true)}>
                    <Icon name="muninn" />
                    {t('leads.transferMuninn')}
                  </button>
                ) : null}
                {lead.status !== 'ARCHIVED' ? (
                  <button type="button" className="menu-item danger" role="menuitem" onClick={() => setDeleteOpen(true)}>
                    <Icon name="trash" />
                    {t('leads.deleteLead')}
                  </button>
                ) : null}
              </div>
            ) : null}
          </div>
        </div>
      </div>

      <Tabs tabs={tabs} active={tab} onChange={setTab} />

      <div className="mt-5">
        {tab === 'overview' ? <OverviewTab lead={lead} score={score?.total} evidenceCount={evidence?.items.length ?? 0} notes={notes as Array<{ id: string; text: string; author?: { name: string } | null; created_at: string }>} noteText={noteText} onNoteText={setNoteText} onAddNote={() => addNote.mutate()} /> : null}
        {tab === 'contacts' ? <ContactsTab leadId={lead.id} contacts={contacts} /> : null}
        {tab === 'history' ? <TimelineTab events={timeline} /> : null}
        {tab === 'interactions' ? <InteractionsTab interactions={interactions} /> : null}
        {tab === 'reports' ? <ReportsTab reports={reports} /> : null}
        {tab === 'followups' ? (
          <LeadFollowups
            followups={followups}
            onComplete={completeFollowUp.mutate}
            onReopen={reopenFollowUp.mutate}
            onCancel={cancelFollowUp.mutate}
          />
        ) : null}
        {tab === 'evidence' ? <EvidenceTab evidence={evidence?.items ?? []} /> : null}
        {tab === 'score' ? <ScoreTab total={score?.total ?? 0} max={score?.max ?? 100} components={score?.components ?? []} /> : null}
        {tab === 'versions' ? <VersionsTab versions={versions?.items ?? []} /> : null}
      </div>

      <ConfirmDialog
        open={disqualifyOpen}
        onClose={() => setDisqualifyOpen(false)}
        onConfirm={() => disqualify.mutate()}
        title={t('leads.disqualify')}
        message={t('leads.confirmDisqualify')}
        loading={disqualify.isPending}
      >
        <div className="mt-4">
          <Textarea label={t('leads.disqualifyReason')} placeholder={t('leads.disqualifyPlaceholder')} value={disqualifyReason} onChange={(e) => setDisqualifyReason(e.target.value)} />
        </div>
      </ConfirmDialog>

      <ConfirmDialog
        open={exportOpen}
        onClose={() => setExportOpen(false)}
        onConfirm={() => exportMuninn.mutate()}
        title={t('leads.transferMuninn')}
        message={t('leads.confirmExport')}
        danger={false}
        loading={exportMuninn.isPending}
      />

      <CreateFollowUpDialog
        open={followupOpen}
        onClose={() => setFollowupOpen(false)}
        mode="lead"
        leadId={lead.id}
        leadName={lead.company_name}
      />

      <LeadEditDialog
        open={editOpen}
        lead={lead}
        onClose={() => setEditOpen(false)}
        loading={updateLead.isPending}
        onSubmit={(patch, note) => updateLead.mutate({ patch, note })}
      />

      <ConfirmDialog
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => deleteLead.mutate()}
        title={t('leads.deleteLead')}
        message={t('leads.confirmDeleteLead')}
        loading={deleteLead.isPending}
      />
    </>
  )
}

const LEAD_SOURCES = ['لیست تولیدکنندگان', 'وب‌سایت شرکت', 'اینستاگرام', 'نمایشگاه', 'اتاق بازرگانی', 'دیتای بازار', 'دستی']

function LeadEditDialog({
  open,
  lead,
  onClose,
  loading,
  onSubmit,
}: {
  open: boolean
  lead: Lead
  onClose: () => void
  loading: boolean
  onSubmit: (data: Partial<Lead>, note?: string) => void
}) {
   const { t } = useI18n()
   const { can } = useAuth()
   const [form, setForm] = useState<Record<string, string>>(() => ({}))
   const [note, setNote] = useState('')

   const canManageProjects = can(PERMISSIONS.projectsManage)
   const { data: projectsPage } = useQuery({
     queryKey: queryKeys.projects({ page_size: 100 }),
     queryFn: () => api.listProjects({ page_size: 100 }),
     enabled: open && canManageProjects,
    retry: false,
  })
   const projectOptions = (projectsPage?.items ?? []).filter((p) => p.status === 'ACTIVE')


  useEffect(() => {
    if (open) {
      setForm({
        company_name: lead.company_name ?? '',
        business_type: lead.business_type ?? '',
        industry: lead.industry ?? '',
        province: lead.province ?? '',
        city: lead.city ?? '',
        source: lead.source ?? '',
        description: lead.description ?? '',
        website: lead.website ?? '',
        instagram: lead.instagram ?? '',
        project_id: lead.project_id ?? '',
      })
      setNote('')
    }
  }, [open, lead])

  const set = (k: string, v: string) => setForm((f) => ({ ...f, [k]: v }))

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t('leads.editLead')}
      footer={
        <>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button variant="primary" loading={loading} onClick={() => onSubmit(leadFormPayload(form), note || undefined)}>
            {t('common.save')}
          </Button>
        </>
      }
    >
      <div className="form-row">
        <Input label={`${t('leads.company')} *`} value={form.company_name ?? ''} onChange={(e) => set('company_name', e.target.value)} />
        <Input label={t('leads.businessType')} value={form.business_type ?? ''} onChange={(e) => set('business_type', e.target.value)} />
        <Input label={t('leads.industry')} value={form.industry ?? ''} onChange={(e) => set('industry', e.target.value)} />
        <Input label={t('leads.province')} value={form.province ?? ''} onChange={(e) => set('province', e.target.value)} />
        <Input label={t('leads.city')} value={form.city ?? ''} onChange={(e) => set('city', e.target.value)} />
        <div className="field">
          <label className="field-label" htmlFor="lead-source-edit">
            {t('leads.source')}
          </label>
          <input id="lead-source-edit" className="input" list="lead-sources-detail" value={form.source ?? ''} onChange={(e) => set('source', e.target.value)} />
          <datalist id="lead-sources-detail">
            {LEAD_SOURCES.map((s) => (
              <option key={s} value={s} />
            ))}
          </datalist>
        </div>
      </div>
       <div className="field">
         <Textarea label={t('leads.description')} value={form.description ?? ''} onChange={(e) => set('description', e.target.value)} style={{ minHeight: 70 }} />
       </div>
       {canManageProjects ? (
         <div className="field">
           <label className="field-label" htmlFor="lead-project-edit">
             {t('projects.title')}
           </label>
           <select id="lead-project-edit" className="select" value={form.project_id ?? ''} onChange={(e) => set('project_id', e.target.value)}>
             <option value="">{t('common.none')}</option>
             {projectOptions.map((p) => (
               <option key={p.id} value={p.id}>{p.title}</option>
             ))}
           </select>
         </div>
       ) : null}
       <div className="field">
         <Textarea label={t('versions.changeReason')} value={note} onChange={(e) => setNote(e.target.value)} style={{ minHeight: 50 }} />
       </div>
    </Dialog>
  )
}

function leadFormPayload(form: Record<string, string>): Partial<Lead> {
  return {
    company_name: form.company_name ?? '',
    business_type: form.business_type || null,
    industry: form.industry || null,
    province: form.province || null,
    city: form.city || null,
    source: form.source || null,
    description: form.description || null,
    website: form.website || null,
    instagram: form.instagram || null,
    project_id: form.project_id || null,
  }
}

function AssignMenuItem({ agents, onAssign }: { agents: Array<{ id: string; name: string }>; onAssign: (id: string) => void }) {
  const { t } = useI18n()
  return (
    <div>
      <div className="menu-label">{t('leads.assign')}</div>
      {agents.map((a) => (
        <button key={a.id} type="button" className="menu-item" role="menuitem" onClick={() => onAssign(a.id)}>
          <Icon name="user" />
          {a.name}
        </button>
      ))}
    </div>
  )
}

function OverviewTab({
  lead,
  score,
  evidenceCount,
  notes,
  noteText,
  onNoteText,
  onAddNote,
}: {
  lead: Lead
  score?: number
  evidenceCount: number
  notes: Array<{ id: string; text: string; author?: { name: string } | null; created_at: string }>
  noteText: string
  onNoteText: (v: string) => void
  onAddNote: () => void
}) {
  const { t, locale } = useI18n()
  const pc = lead.primary_contact
  const memberships = lead.projects ?? []
  return (
    <div className="grid-2">
      {memberships.length > 0 ? (
        <div className="card" style={{ gridColumn: '1 / -1' }}>
          <div className="card-header">
            <h3>{t('projects.title')}</h3>
          </div>
          <div className="table-scroll">
            <table className="table">
              <thead>
                <tr>
                  <th>{t('projects.name')}</th>
                  <th>{t('projects.stage')}</th>
                  <th>{t('projects.nextAction')}</th>
                </tr>
              </thead>
              <tbody>
                {memberships.map((m) => (
                  <tr key={m.project_id}>
                    <td>
                      <Link to={`/projects/${m.project_id}`} className="strong" style={{ color: 'var(--text-1)' }}>
                        {m.project_title}
                      </Link>
                    </td>
                    <td>{m.stage ? <Badge tone="accent">{m.stage}</Badge> : <span className="muted">—</span>}</td>
                    <td>{m.next_action ?? '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}
      <div className="card">
        <div className="card-header">
          <h3>{t('leads.companyInfo')}</h3>
        </div>
        <div className="card-body">
          <DetailRows
            rows={[
              { label: t('leads.company'), value: lead.company_name },
              { label: t('leads.businessType'), value: lead.business_type },
              { label: t('leads.industry'), value: lead.industry },
              { label: t('leads.address'), value: lead.address },
              { label: t('leads.website'), value: lead.website },
              { label: t('leads.socials'), value: lead.instagram ? `instagram.com/${lead.instagram}` : null },
            ]}
          />
        </div>
      </div>

      <div className="card">
        <div className="card-header">
          <h3>{t('leads.contactInfo')}</h3>
        </div>
        <div className="card-body">
          <DetailRows
            rows={[
              { label: t('leads.primaryContact'), value: pc?.full_name },
              { label: t('leads.contactRole'), value: pc?.job_title },
              { label: t('leads.mobile'), value: pc?.mobile ? <span className="mono" dir="ltr">{pc.mobile}</span> : null },
              { label: t('leads.phone'), value: pc?.phone ? <span className="mono" dir="ltr">{pc.phone}</span> : null },
              { label: t('leads.email'), value: pc?.email ? <span className="mono" dir="ltr">{pc.email}</span> : null },
              { label: t('leads.decisionMakerLabel'), value: pc ? (pc.is_decision_maker ? t('common.yes') : t('common.no')) : null },
            ]}
          />
        </div>
      </div>

      <div className="card">
        <div className="card-header">
          <h3>{t('leads.notes')}</h3>
        </div>
        <div className="card-body">
          <div className="flex gap-2 mb-4">
            <textarea
              className="textarea"
              placeholder={t('leads.addNote')}
              value={noteText}
              onChange={(e) => onNoteText(e.target.value)}
              style={{ minHeight: 60 }}
            />
            <Button variant="primary" onClick={onAddNote} disabled={!noteText.trim()}>
              {t('common.add')}
            </Button>
          </div>
          {notes.length === 0 ? (
            <p className="muted small">{t('common.noData')}</p>
          ) : (
            notes.map((n) => (
              <div key={n.id} className="detail-row" style={{ alignItems: 'flex-start' }}>
                <dd style={{ textAlign: 'start', fontWeight: 400 }}>{n.text}</dd>
                <dt>
                  {n.author?.name} · {formatDateTime(n.created_at, locale, { time: false })}
                </dt>
              </div>
            ))
          )}
        </div>
      </div>

      <div className="card">
        <div className="card-header">
          <h3>{t('leads.scoreBreakdown')}</h3>
          <span className="badge badge-accent">{score !== undefined ? toFaDigits(score) : '—'}</span>
        </div>
        <div className="card-body">
          <DetailRows
            rows={[
              { label: t('leads.interestLevel'), value: lead.qualification?.need_interest ? t('common.yes') : null },
              { label: t('leads.needLevel'), value: null },
              { label: t('leads.purchaseTiming'), value: null },
              { label: t('leads.estimatedCapacity'), value: null },
              { label: t('leads.mainObjection'), value: lead.qualification?.disqualify_reason },
              { label: t('leads.nextAction'), value: lead.qualification?.next_action ? t('common.yes') : null },
              { label: t('leads.evidenceTab'), value: evidenceCount > 0 ? `${toFaDigits(evidenceCount)} ${t('leads.evidenceValue')}` : '—' },
            ]}
          />
        </div>
      </div>
    </div>
  )
}

function DetailRows({ rows }: { rows: Array<{ label: string; value: ReactNode }> }) {
  return (
    <dl className="detail-list">
      {rows.map((r, i) => (
        <div className="detail-row" key={i}>
          <dt>{r.label}</dt>
          <dd>{r.value ?? <EmptyValue />}</dd>
        </div>
      ))}
    </dl>
  )
}

function ContactsTab({ leadId, contacts }: { leadId: string; contacts: Contact[] }) {
  const { t } = useI18n()
  const toast = useToastApi()
  const qc = useQueryClient()
  const [edit, setEdit] = useState<Contact | null>(null)
  const [createOpen, setCreateOpen] = useState(false)

  const update = useMutation({
    mutationFn: (patch: Partial<Contact>) => api.updateContact(leadId, edit?.id ?? '', patch),
    onSuccess: () => {
      toast.success(t('common.save'))
      setEdit(null)
      void qc.invalidateQueries({ queryKey: queryKeys.leadContacts(leadId) })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const archive = useMutation({
    mutationFn: (id: string) => api.archiveContact(leadId, id),
    onSuccess: () => {
      toast.success(t('common.save'))
      void qc.invalidateQueries({ queryKey: queryKeys.leadContacts(leadId) })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  return (
    <div className="card">
      <div className="card-header">
        <h3>{t('leads.contactsTab')}</h3>
        <Button size="sm" icon="plus" onClick={() => setCreateOpen(true)}>
          {t('leads.addContact')}
        </Button>
      </div>
      <div className="table-scroll">
        <table className="table">
          <thead>
            <tr>
              <th>{t('leads.contactName')}</th>
              <th>{t('leads.contactRole')}</th>
              <th>{t('leads.mobile')}</th>
              <th>{t('leads.phone')}</th>
              <th>{t('leads.email')}</th>
              <th>{t('leads.decisionMakerToggle')}</th>
              <th>{t('leads.primaryToggle')}</th>
              <th>{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {contacts.length === 0 ? (
              <tr>
                <td colSpan={8} className="table-empty">
                  <p className="muted">{t('common.noData')}</p>
                </td>
              </tr>
            ) : (
              contacts.map((c) => (
                <tr key={c.id}>
                  <td className="strong">{c.full_name ?? <EmptyValue />}</td>
                  <td>{c.job_title ?? <EmptyValue />}</td>
                  <td>{c.mobile ? <span className="mono" dir="ltr">{c.mobile}</span> : <EmptyValue />}</td>
                  <td>{c.phone ? <span className="mono" dir="ltr">{c.phone}</span> : <EmptyValue />}</td>
                  <td>{c.email ? <span className="mono" dir="ltr">{c.email}</span> : <EmptyValue />}</td>
                  <td>{c.is_decision_maker ? <Badge tone="accent">{t('common.yes')}</Badge> : <span className="muted">—</span>}</td>
                  <td>{c.is_primary_contact ? <Badge tone="success">{t('common.yes')}</Badge> : <span className="muted">—</span>}</td>
                  <td>
                    <div className="row-actions">
                      <Button size="sm" variant="ghost" icon="edit" onClick={() => setEdit(c)} aria-label={t('common.edit')} />
                      <Button size="sm" variant="ghost" icon="trash" onClick={() => archive.mutate(c.id)} aria-label={t('common.delete')} />
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <Dialog
        open={!!edit}
        onClose={() => setEdit(null)}
        title={t('common.edit')}
        footer={
          <>
            <Button onClick={() => setEdit(null)}>{t('common.cancel')}</Button>
            <Button variant="primary" onClick={() => update.mutate({})}>
              {t('common.save')}
            </Button>
          </>
        }
      >
        {edit ? (
          <div className="form-row">
            <Checkbox label={t('leads.decisionMakerToggle')} checked={edit.is_decision_maker} onChange={(v) => update.mutate({ is_decision_maker: v })} />
            <Checkbox label={t('leads.primaryToggle')} checked={edit.is_primary_contact} onChange={(v) => update.mutate({ is_primary_contact: v })} />
          </div>
        ) : null}
      </Dialog>

      <Dialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title={t('leads.addContact')}
        footer={<></>}
      >
        <CreateContactForm leadId={leadId} onDone={() => setCreateOpen(false)} />
      </Dialog>
    </div>
  )
}

import { Badge } from '@/components/ui/Badge'
import { Checkbox } from '@/components/ui/Field'

function CreateContactForm({ leadId, onDone }: { leadId: string; onDone: () => void }) {
  const { t } = useI18n()
  const toast = useToastApi()
  const qc = useQueryClient()
  const [form, setForm] = useState<Record<string, string>>({})

  const create = useMutation({
    mutationFn: () =>
      api.createContact(leadId, {
        first_name: form.firstName,
        last_name: form.lastName,
        full_name: `${form.firstName ?? ''} ${form.lastName ?? ''}`.trim(),
        job_title: form.jobTitle,
        mobile: form.mobile,
        phone: form.phone,
        email: form.email,
        is_decision_maker: form.isDecisionMaker === 'yes',
      }),
    onSuccess: () => {
      toast.success(t('common.save'))
      onDone()
      void qc.invalidateQueries({ queryKey: queryKeys.leadContacts(leadId) })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  return (
    <div>
      <div className="form-row">
        <div className="field">
          <label className="field-label">{t('leads.contactName')}</label>
          <input className="input" value={form.firstName ?? ''} onChange={(e) => setForm((f) => ({ ...f, firstName: e.target.value }))} />
        </div>
        <div className="field">
          <label className="field-label">{t('leads.contactRole')}</label>
          <input className="input" value={form.jobTitle ?? ''} onChange={(e) => setForm((f) => ({ ...f, jobTitle: e.target.value }))} />
        </div>
        <div className="field">
          <label className="field-label">{t('leads.mobile')}</label>
          <input className="input" dir="ltr" value={form.mobile ?? ''} onChange={(e) => setForm((f) => ({ ...f, mobile: e.target.value }))} />
        </div>
        <div className="field">
          <label className="field-label">{t('leads.email')}</label>
          <input className="input" dir="ltr" value={form.email ?? ''} onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))} />
        </div>
        <div className="field">
          <label className="field-label">{t('leads.decisionMakerToggle')}</label>
          <Select value={form.isDecisionMaker ?? 'no'} onChange={(e) => setForm((f) => ({ ...f, isDecisionMaker: e.target.value }))}>
            <option value="yes">{t('common.yes')}</option>
            <option value="no">{t('common.no')}</option>
          </Select>
        </div>
      </div>
      <div className="form-actions">
        <Button onClick={onDone}>{t('common.cancel')}</Button>
        <Button variant="primary" loading={create.isPending} onClick={() => create.mutate()}>
          {t('common.save')}
        </Button>
      </div>
    </div>
  )
}

function TimelineTab({ events }: { events: Array<{ id: string; event_type: string; title: string; description?: string | null; actor?: { name: string } | null; created_at: string }> }) {
  const { t, locale } = useI18n()
  return (
    <div className="card">
      <div className="card-body">
        {events.length === 0 ? (
          <p className="muted">{t('common.noData')}</p>
        ) : (
          <ul className="timeline">
            {events.map((e) => {
              const key = e.event_type
              const typeClass = key.includes('qualif') ? 'type-created' : key.includes('disqualif') || key.includes('fail') ? 'type-error' : key.includes('note') || key.includes('over') ? 'type-warn' : 'type-accent'
              return (
                <li key={e.id} className="timeline-item">
                  <span className={`timeline-marker ${typeClass}`} aria-hidden="true">
                    <Icon name={markerIcon(key)} size={9} />
                  </span>
                  <div className="timeline-title">{timelineLabel(key, e.title, t)}</div>
                  {e.description ? <div className="timeline-desc">{e.description}</div> : null}
                  <div className="timeline-meta">
                    {e.actor ? (
                      <span className="timeline-actor">
                        <Avatar name={e.actor.name} size={18} />
                        {e.actor.name}
                      </span>
                    ) : null}
                    <span>{formatDateTime(e.created_at, locale)}</span>
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}

/**
 * Maps a backend timeline `event_type` (e.g. "lead.created") to a dot-free i18n
 * key under `leads.timeline.*`. Unknown types return null so callers can fall
 * back to the raw title instead of rendering a machine key.
 */
const TIMELINE_EVENT_KEYS: Record<string, string> = {
  'lead.created': 'created',
  'lead.updated': 'updated',
  'lead.assigned': 'assigned',
  'lead.qualified': 'qualified',
  'lead.disqualified': 'disqualified',
  'lead.exported': 'exported',
  'lead.merged': 'merged',
  'lead.tag_added': 'tag_added',
  'lead.tag_removed': 'tag_removed',
  'lead.project.attached': 'project_attached',
  'lead.project.detached': 'project_detached',
  'note.created': 'note',
  'followup.created': 'follow_up',
  'report.submitted': 'report',
  'report.voided': 'report_voided',
  'interaction.created': 'interaction',
  'contact.created': 'contact_added',
  'contact.updated': 'contact_updated',
  'contact.deleted': 'contact_removed',
  'call_scenario.generated': 'call_scenario_generated',
  'call_scenario.created': 'call_scenario_created',
  'call_scenario.updated': 'call_scenario_updated',
  'call_scenario.deleted': 'call_scenario_deleted',
  'project.updated': 'project_updated',
}

/**
 * Resolves a timeline event type to its localized label. Falls back to the
 * backend title (and finally the raw key) for unknown event types, so a raw
 * machine key or English prose is never shown for known events.
 */
function timelineLabel(key: string, title: string, t: (k: string) => string): string {
  const mapped = TIMELINE_EVENT_KEYS[key]
  if (!mapped) return title || key
  const path = `leads.timeline.${mapped}`
  const translated = t(path)
  return translated === path ? (title || key) : translated
}

function markerIcon(eventType: string): IconName {
  if (eventType.includes('call')) return 'phone'
  if (eventType.includes('sms') || eventType.includes('messenger')) return 'message'
  if (eventType.includes('email')) return 'mail'
  if (eventType.includes('report')) return 'reports'
  if (eventType.includes('follow')) return 'followups'
  if (eventType.includes('note')) return 'edit'
  if (eventType.includes('score')) return 'scoring'
  if (eventType.includes('qualif')) return 'check'
  if (eventType.includes('disqualif')) return 'x'
  if (eventType.includes('export') || eventType.includes('muninn')) return 'muninn'
  if (eventType.includes('over')) return 'alert'
  if (eventType.includes('assign')) return 'users'
  return 'activity'
}

function InteractionsTab({ interactions }: { interactions: Array<{ id: string; channel: string; direction: string; objective?: string | null; started_at: string; duration_seconds?: number | null; action_status: string; report_status: string; user?: { name: string } | null }> }) {
  const { t, locale } = useI18n()
  return (
    <div className="card">
      <div className="table-scroll">
        <table className="table">
          <thead>
            <tr>
              <th>{t('reports.channel')}</th>
              <th>{t('reports.agent')}</th>
              <th>{t('reports.time')}</th>
              <th>{t('reports.duration')}</th>
              <th>{t('leads.nextAction')}</th>
              <th>{t('reports.reportStatus')}</th>
            </tr>
          </thead>
          <tbody>
            {interactions.length === 0 ? (
              <tr>
                <td colSpan={6} className="table-empty">
                  <p className="muted">{t('common.noData')}</p>
                </td>
              </tr>
            ) : (
              interactions.map((it) => (
                <tr key={it.id}>
                  <td>
                    <Badge tone="info">{t(`status.${it.channel}`)}</Badge>
                  </td>
                  <td>{it.user?.name}</td>
                  <td>{formatDateTime(it.started_at, locale)}</td>
                  <td>{it.duration_seconds ? formatDur(it.duration_seconds) : '—'}</td>
                  <td>{it.objective ?? '—'}</td>
                  <td>
                    <StatusBadge status={it.report_status} />
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function ReportsTab({ reports }: { reports: ContactReport[] }) {
  const { t, locale } = useI18n()
  return (
    <div className="card">
      <div className="table-scroll">
        <table className="table">
          <thead>
            <tr>
              <th>{t('reports.outcome')}</th>
              <th>{t('reports.interest')}</th>
              <th>{t('reports.nextAction')}</th>
              <th>{t('reports.time')}</th>
              <th>{t('reports.reportStatus')}</th>
            </tr>
          </thead>
          <tbody>
            {reports.length === 0 ? (
              <tr>
                <td colSpan={5} className="table-empty">
                  <p className="muted">{t('common.noData')}</p>
                </td>
              </tr>
            ) : (
              reports.map((r) => (
                <tr key={r.id}>
                  <td className="strong">{r.outcome}</td>
                  <td>{r.interest_level ?? '—'}</td>
                  <td>{r.next_action ?? '—'}</td>
                  <td>{formatDateTime(r.created_at, locale, { time: false })}</td>
                  <td>
                    {r.manager_override ? <Badge tone="info">{t('reports.managerOverride')}</Badge> : <StatusBadge status="SUBMITTED" />}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function LeadFollowups({
  followups,
  onComplete,
  onReopen,
  onCancel,
}: {
  followups: FollowUp[]
  onComplete: (id: string) => void
  onReopen: (id: string) => void
  onCancel: (id: string) => void
}) {
  const { t, locale } = useI18n()
  return (
    <div className="card">
      <div className="table-scroll">
        <table className="table">
          <thead>
            <tr>
              <th>{t('followups.dueDate')}</th>
              <th>{t('followups.type')}</th>
              <th>{t('followups.description')}</th>
              <th>{t('followups.status')}</th>
              <th>{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {followups.length === 0 ? (
              <tr>
                <td colSpan={5} className="table-empty">
                  <p className="muted">{t('common.noData')}</p>
                </td>
              </tr>
            ) : (
              followups.map((f) => (
                <tr key={f.id}>
                  <td>
                    {formatDateTime(f.due_at, locale, { time: false })}
                    {f.is_overdue ? <span className="fu-overdue"> · {t('followups.overdue')}</span> : null}
                  </td>
                  <td>{t(`followups.types.${f.type}`) ?? f.type}</td>
                  <td>{f.description ?? f.title ?? '—'}</td>
                  <td>
                    <StatusBadge status={f.status} />
                  </td>
                  <td>
                    <span className="table-actions">
                      {f.status === 'PENDING' ? (
                        <>
                          <Button size="sm" icon="check" onClick={() => onComplete(f.id)}>
                            {t('followups.complete')}
                          </Button>
                          <Button size="sm" variant="ghost" icon="trash" onClick={() => onCancel(f.id)}>
                            {t('followups.cancel')}
                          </Button>
                        </>
                      ) : null}
                      {f.status === 'COMPLETED' ? (
                        <Button size="sm" variant="ghost" onClick={() => onReopen(f.id)}>
                          {t('followups.reopen')}
                        </Button>
                      ) : null}
                    </span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function EvidenceTab({ evidence }: { evidence: Array<{ field_name: string; value: string; confidence: number; source: string; excerpt: string; page: number | null; row: number | null }> }) {
  const { t } = useI18n()
  return (
    <div className="card">
      <div className="card-body">
        {evidence.length === 0 ? (
          <p className="muted">{t('common.noData')}</p>
        ) : (
          <div className="table-scroll">
            <table className="table">
              <thead>
                <tr>
                  <th>{t('leads.evidenceField')}</th>
                  <th>{t('leads.evidenceValue')}</th>
                  <th>{t('leads.confidence')}</th>
                  <th>{t('leads.evidenceSource')}</th>
                  <th>{t('leads.excerpt')}</th>
                  <th>{t('leads.pageRow')}</th>
                </tr>
              </thead>
              <tbody>
                {evidence.map((e, i) => (
                  <tr key={i}>
                    <td className="strong">{t(`leads.${e.field_name}`)}</td>
                    <td>{e.value}</td>
                    <td>
                      <ConfidenceBadge value={e.confidence} />
                    </td>
                    <td>{e.source}</td>
                    <td>
                      <div className="evidence-excerpt">{e.excerpt}</div>
                    </td>
                    <td>
                      {e.page ? `${toFaDigits(e.page)} / ${e.row ? toFaDigits(e.row) : '—'}` : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

function ScoreTab({
  total,
  max,
  components,
}: {
  total: number
  max: number
  components: Array<{ key: string; name: string; weight: number; contribution: number; value: number; max: number }>
}) {
  const { t } = useI18n()
  return (
    <div className="card" style={{ maxWidth: 720 }}>
      <div className="card-header">
        <h3>{t('leads.scoreBreakdown')}</h3>
        <span className="badge badge-accent">
          {t('leads.totalScore')}: {toFaDigits(total)}/{toFaDigits(max)}
        </span>
      </div>
      <div className="card-body">
        {components.length === 0 ? (
          <p className="muted">{t('common.noData')}</p>
        ) : (
          <div>
            {components.map((c) => (
              <div className="score-dim" key={c.key}>
                <span className="dim-name">{t(`leads.scoreDimensions.${c.key}`)}</span>
                <span className="dim-weight">%{toFaDigits(c.weight)}</span>
                <span className="dim-score">{toFaDigits(c.contribution)}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function VersionsTab({ versions }: { versions: LeadVersion[] }) {
  const { t, locale } = useI18n()
  const [selected, setSelected] = useState<LeadVersion | null>(null)
  return (
    <>
      <div className="card">
        <div className="card-header">
          <h3>{t('versions.title')}</h3>
        </div>
        <div className="card-body">
          {versions.length === 0 ? (
            <p className="muted">{t('common.noData')}</p>
          ) : (
            <ul className="timeline">
              {versions.map((v) => (
                <li key={v.id} className="timeline-item">
                  <span className="timeline-marker type-accent" aria-hidden="true">
                    <Icon name="layers" size={9} />
                  </span>
                  <button type="button" className="version-row" onClick={() => setSelected(v)}>
                    <div className="timeline-title">
                      {t('versions.versionLabel', { n: String(v.version) })}
                    </div>
                    <div className="timeline-meta">
                      <span className="timeline-actor">
                        <Avatar name={v.actor?.name ?? t('common.system')} size={18} />
                        {v.actor?.name ?? t('common.system')}
                        {v.changed_by_role ? <Badge tone="neutral">{v.changed_by_role}</Badge> : null}
                      </span>
                      <span>{formatDateTime(v.created_at, locale)}</span>
                      <span className="change-count">
                        {t('versions.changeCount', { n: String((v.changes ?? []).length) })}
                      </span>
                    </div>
                    {v.note ? <div className="timeline-desc">«{v.note}»</div> : null}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      <Dialog
        open={!!selected}
        onClose={() => setSelected(null)}
        title={selected ? `${t('versions.versionLabel', { n: String(selected.version) })}` : ''}
        footer={
          <Button onClick={() => setSelected(null)}>{t('common.close')}</Button>
        }
      >
        {selected ? <VersionDetail version={selected} /> : null}
      </Dialog>
    </>
  )
}

function VersionDetail({ version }: { version: LeadVersion }) {
  const { t, locale } = useI18n()
  const changes = version.changes ?? []
  return (
    <div>
      <div className="flex items-center gap-2 mb-4 wrap">
        <Badge tone="brand">{t('versions.versionLabel', { n: String(version.version) })}</Badge>
        {version.changed_by_role ? <Badge tone="neutral">{version.changed_by_role}</Badge> : null}
        <span className="small secondary">
          {version.actor?.name ?? t('common.system')} · {formatDateTime(version.created_at, locale)}
        </span>
      </div>
      {version.note ? (
        <div className="note-callout mb-4">{version.note}</div>
      ) : null}
      {changes.length === 0 ? (
        <p className="muted">{t('versions.noChanges')}</p>
      ) : (
        <div className="table-scroll">
          <table className="table">
            <thead>
              <tr>
                <th>{t('versions.field')}</th>
                <th>{t('versions.old')}</th>
                <th>{t('versions.new')}</th>
              </tr>
            </thead>
            <tbody>
              {changes.map((ch, i) => {
                const view = describeChange(ch, version.data, t)
                return (
                  <tr key={i}>
                    <td className="strong">{view.label}</td>
                    <td className="diff-old">{view.oldLabel}</td>
                    <td className="diff-new">{view.newLabel}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function contactNameById(data: Record<string, unknown>, contactId: string): string | null {
  const contacts = Array.isArray(data?.contacts) ? (data.contacts as Array<Record<string, unknown>>) : []
  for (const c of contacts) {
    if (c.id === contactId && typeof c.full_name === 'string' && c.full_name) return c.full_name
    if (c.id === contactId && typeof c.first_name === 'string' && c.first_name) return c.first_name
  }
  return null
}

function fmtValue(v: unknown): string {
  if (v === null || v === undefined) return '—'
  if (typeof v === 'boolean') return v ? 'بله' : 'خیر'
  if (typeof v === 'object') return JSON.stringify(v)
  return String(v)
}

function fmtContact(v: unknown): string {
  if (v === null || v === undefined) return '—'
  if (typeof v === 'object') {
    const c = v as Record<string, unknown>
    return [c.full_name, c.job_title, c.mobile, c.email].filter(Boolean).join(' · ')
  }
  return String(v)
}

function describeChange(
  ch: LeadChange,
  data: Record<string, unknown>,
  t: (key: string, vars?: Record<string, string | number>) => string,
): { label: string; oldLabel: string; newLabel: string } {
  const field = ch.field
  if (field.startsWith('contact.')) {
    const parts = field.split('.')
    const contactId = parts[1]
    const sub = parts.slice(2).join('.')
    const name = contactNameById(data, contactId)
    const who = name ? `${t('leads.contact')} ${name}` : t('leads.contact')
    if (sub === 'added') return { label: `${who} ${t('versions.added')}`, oldLabel: '—', newLabel: fmtContact(ch.new) }
    if (sub === 'removed') return { label: `${who} ${t('versions.removed')}`, oldLabel: fmtContact(ch.old), newLabel: '—' }
    const fieldKey = `leads.versionFields.${sub}`
    const fieldLabel = t(fieldKey) === fieldKey ? sub : t(fieldKey)
    return { label: `${who} — ${fieldLabel}`, oldLabel: fmtValue(ch.old), newLabel: fmtValue(ch.new) }
  }
  if (field === 'lead.created') return { label: t('versions.created'), oldLabel: '—', newLabel: fmtValue(ch.new) }
  const fieldKey = `leads.versionFields.${field}`
  const fieldLabel = t(fieldKey) === fieldKey ? field : t(fieldKey)
  return { label: fieldLabel, oldLabel: fmtValue(ch.old), newLabel: fmtValue(ch.new) }
}

import type { ContactReport, FollowUp } from '@/api/types'
import { formatDur } from './leadFormat'
