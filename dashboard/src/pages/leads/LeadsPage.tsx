import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import type { ColumnDef } from '@tanstack/react-table'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { useUrlState, useDebouncedValue } from '@/hooks'
import { useAuth, PERMISSIONS } from '@/auth/AuthContext'
import { toFaDigits, formatDateTime } from '@/utils/format'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import { StatusBadge, ScoreBadge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dropdown, MenuItem } from '@/components/ui/Dropdown'
import { Icon } from '@/components/Icon'
import { ConfirmDialog, Dialog } from '@/components/ui/Dialog'
import { Select, Input, Textarea } from '@/components/ui/Field'

import type { Lead, Project } from '@/api/types'

type LeadFilters = {
  page?: string
  search?: string
  status?: string
  owner?: string
  province?: string
  city?: string
  industry?: string
  business_type?: string
  source?: string
  source_batch_id?: string
  score_min?: string
  score_max?: string
  tag?: string
  sort_by?: string
  sort_dir?: 'asc' | 'desc'
  unassigned?: string
  review?: string
  project_id?: string
}

const defaultFilters: LeadFilters = { page: '1' }

export function LeadsPage() {
  const { t, locale } = useI18n()
  const { can } = useAuth()
  const toast = useToastApi()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [url, setUrl] = useUrlState<LeadFilters>(defaultFilters)
  const [filtersOpen, setFiltersOpen] = useState(false)
  const [draft, setDraft] = useState<LeadFilters>(url)
  const [selected, setSelected] = useState<Record<string, boolean>>({})
  const [assignTarget, setAssignTarget] = useState<{ leadIds: string[]; assigneeId?: string } | null>(null)
  const [disqualifyTarget, setDisqualifyTarget] = useState<Lead | null>(null)
  const [disqualifyReason, setDisqualifyReason] = useState('')
  const [exportTarget, setExportTarget] = useState<Lead | null>(null)
  const [archiveTarget, setArchiveTarget] = useState<Lead | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [editLead, setEditLead] = useState<Lead | null>(null)

  const debouncedSearch = useDebouncedValue(url.search ?? '', 400)

  const query = useMemo(() => {
    const q: Record<string, string> = { page_size: '20' }
    if (url.page) q.page = url.page
    if (debouncedSearch) q.search = debouncedSearch
    if (url.status) q.status = url.status
    if (url.owner) q.owner = url.owner
    if (url.province) q.province = url.province
    if (url.city) q.city = url.city
    if (url.industry) q.industry = url.industry
    if (url.business_type) q.business_type = url.business_type
    if (url.source) q.source = url.source
    if (url.source_batch_id) q.source_batch_id = url.source_batch_id
    if (url.score_min) q.score_min = url.score_min
    if (url.score_max) q.score_max = url.score_max
    if (url.tag) q.tag = url.tag
    if (url.unassigned) q.unassigned = '1'
    if (url.review) q.review = '1'
    if (url.project_id) q.project_id = url.project_id
    if (url.sort_by) {
      q.sort_by = url.sort_by
      q.sort_dir = url.sort_dir ?? 'desc'
    }
    return q
  }, [url, debouncedSearch])

  const { data, isLoading, isFetching, isError, refetch } = useQuery({
    queryKey: queryKeys.leads(query),
    queryFn: () => api.listLeads(query as Parameters<typeof api.listLeads>[0]),
  })

  const { data: users } = useQuery({
    queryKey: queryKeys.users,
    queryFn: () => api.listUsers(),
    enabled: can(PERMISSIONS.leadsAssign),
  })

  const agents = useMemo(() => (users ?? []).filter((u) => u.role === 'sales_agent'), [users])

  const { data: projectsPage } = useQuery({
    queryKey: queryKeys.projects({ page_size: 100 }),
    queryFn: () => api.listProjects({ page_size: 100 }),
    enabled: can(PERMISSIONS.projectsManage),
  })
  const projectOptions = useMemo(
    () => (projectsPage?.items ?? []).filter((p) => p.status === 'ACTIVE'),
    [projectsPage],
  )

  const listLeads = useMutation({
    mutationFn: (vars: { leadIds: string[]; assigneeId: string }) => api.assignLeads(vars.leadIds, vars.assigneeId, 'manual'),
    onSuccess: () => {
      toast.success(t('assignments.assignSuccess'))
      setAssignTarget(null)
      setSelected({})
      void qc.invalidateQueries({ queryKey: queryKeys.leads() })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const disqualify = useMutation({
    mutationFn: (vars: { id: string; reason: string }) => api.disqualifyLead(vars.id, vars.reason),
    onSuccess: () => {
      toast.success(t('leads.disqualifiedSuccess'))
      setDisqualifyTarget(null)
      setDisqualifyReason('')
      void qc.invalidateQueries({ queryKey: queryKeys.lead(disqualifyTarget?.id ?? '') })
      void qc.invalidateQueries({ queryKey: queryKeys.leads() })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const exportMuninn = useMutation({
    mutationFn: (id: string) => api.exportLeadToMuninn(id),
    onSuccess: () => {
      toast.success(t('leads.exportQueued'))
      setExportTarget(null)
      void qc.invalidateQueries({ queryKey: queryKeys.lead(exportTarget?.id ?? '') })
      void qc.invalidateQueries({ queryKey: queryKeys.leads() })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const archive = useMutation({
    mutationFn: (id: string) => api.archiveLead(id),
    onSuccess: () => {
      toast.success(t('leads.deleteSuccess'))
      setArchiveTarget(null)
      void qc.invalidateQueries({ queryKey: queryKeys.leads() })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const createLead = useMutation({
    mutationFn: (data: Partial<Lead>) => api.createLead(data),
    onSuccess: () => {
      toast.success(t('leads.createSuccess'))
      setCreateOpen(false)
      void qc.invalidateQueries({ queryKey: queryKeys.leads() })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const updateLead = useMutation({
    mutationFn: (data: Partial<Lead>) => api.updateLead(editLead?.id ?? '', data),
    onSuccess: () => {
      toast.success(t('leads.updated'))
      setEditLead(null)
      void qc.invalidateQueries({ queryKey: queryKeys.leads() })
      if (editLead) void qc.invalidateQueries({ queryKey: queryKeys.lead(editLead.id) })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const columns = useMemo<ColumnDef<Lead, unknown>[]>(() => {
    const cols: ColumnDef<Lead, unknown>[] = [
      {
        id: 'select',
        header: () => (
          <input
            type="checkbox"
            aria-label={t('leads.bulkSelect')}
            checked={data ? data.items.length > 0 && Object.keys(selected).length === data.items.length : false}
            onChange={(e) => {
              if (!data) return
              if (e.target.checked) {
                const next: Record<string, boolean> = {}
                for (const it of data.items) next[it.id] = true
                setSelected(next)
              } else setSelected({})
            }}
          />
        ),
        cell: ({ row }) => (
          <input
            type="checkbox"
            aria-label={t('common.selected')}
            checked={!!selected[row.original.id]}
            onChange={(e) => {
              setSelected((prev) => {
                const next = { ...prev }
                if (e.target.checked) next[row.original.id] = true
                else delete next[row.original.id]
                return next
              })
            }}
          />
        ),
        enableSorting: false,
      },
      {
        accessorKey: 'company_name',
        header: t('leads.company'),
        cell: ({ row }) => (
          <div className="flex flex-col">
            <Link to={`/leads/${row.original.id}`} className="strong" style={{ color: 'var(--text-1)' }}>
              {row.original.company_name}
            </Link>
            {(row.original.tags ?? []).length > 0 ? (
              <span className="flex gap-1 mt-1 wrap">
                {(row.original.tags ?? []).slice(0, 2).map((tag) => (
                  <span key={tag} className="chip chip-accent">
                    {tag}
                  </span>
                ))}
              </span>
            ) : null}
          </div>
        ),
      },
      {
        id: 'primary_contact',
        header: t('leads.primaryContact'),
        cell: ({ row }) => row.original.primary_contact?.full_name ?? <span className="muted">—</span>,
        enableSorting: false,
      },
      { accessorKey: 'business_type', header: t('leads.businessType'), cell: ({ getValue }) => (getValue() as string) || '—' },
      { accessorKey: 'industry', header: t('leads.industry'), cell: ({ getValue }) => (getValue() as string) || '—' },
      {
        id: 'project',
        header: t('projects.title'),
        enableSorting: false,
        cell: ({ row }) =>
          row.original.project ? (
            <Link to={`/projects/${row.original.project.id}`} style={{ color: 'var(--text-1)' }}>{row.original.project.title}</Link>
          ) : (
            <span className="muted">—</span>
          ),
      },
      { accessorKey: 'city', header: t('leads.city'), cell: ({ getValue }) => (getValue() as string) || '—' },
      {
        id: 'phone',
        header: t('leads.phone'),
        cell: ({ row }) => {
          const p = row.original.primary_contact
          const val = p?.mobile ?? p?.phone
          return val ? <span className="mono" dir="ltr">{val}</span> : <span className="muted">—</span>
        },
        enableSorting: false,
      },
      {
        accessorKey: 'score',
        header: t('leads.score'),
        cell: ({ row }) => <ScoreBadge score={row.original.score} />,
        enableSorting: true,
      },
      {
        accessorKey: 'status',
        header: t('leads.status'),
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
        enableSorting: false,
      },
      {
        id: 'owner',
        header: t('leads.owner'),
        cell: ({ row }) => row.original.owner?.name ?? <span className="muted">{t('common.none')}</span>,
        enableSorting: false,
      },
      {
        id: 'last_contact_at',
        header: t('leads.lastContact'),
        cell: ({ row }) => (row.original.last_contact_at ? formatDateTime(row.original.last_contact_at, locale, { time: false }) : <span className="muted">—</span>),
        enableSorting: true,
      },
      {
        id: 'next_follow_up_at',
        header: t('leads.nextFollowUp'),
        cell: ({ row }) => (row.original.next_follow_up_at ? formatDateTime(row.original.next_follow_up_at, locale, { time: false }) : <span className="muted">—</span>),
        enableSorting: true,
      },
      {
        id: 'source',
        header: t('leads.source'),
        cell: ({ row }) => row.original.source ?? row.original.source_batch_id ?? <span className="muted">—</span>,
        enableSorting: false,
      },
      {
        id: 'actions',
        header: t('common.actions'),
        enableSorting: false,
        cell: ({ row }) => {
          const lead = row.original
          return (
            <div className="row-actions" onClick={(e) => e.stopPropagation()}>
              <Dropdown
                align="end"
                trigger={
                  <button type="button" className="btn btn-ghost btn-sm btn-icon" aria-label={t('common.actions')}>
                    <Icon name="more" size={16} />
                  </button>
                }
              >
                <MenuItem icon="eye" onClick={() => navigate(`/leads/${lead.id}`)}>
                  {t('leads.view')}
                </MenuItem>
                {can(PERMISSIONS.leadsEdit) ? (
                  <MenuItem icon="edit" onClick={() => setEditLead(lead)}>
                    {t('leads.edit')}
                  </MenuItem>
                ) : null}
                {can(PERMISSIONS.leadsAssign) ? (
                  <MenuItem icon="users" onClick={() => setAssignTarget({ leadIds: [lead.id] })}>
                    {lead.owner_user_id ? t('leads.reassign') : t('leads.assign')}
                  </MenuItem>
                ) : null}
                {lead.status === 'QUALIFIED' && can(PERMISSIONS.leadsExport) ? (
                  <MenuItem icon="muninn" onClick={() => setExportTarget(lead)}>
                    {t('leads.transferMuninn')}
                  </MenuItem>
                ) : null}
                {can(PERMISSIONS.leadsDisqualify) && lead.status !== 'DISQUALIFIED' && lead.status !== 'ARCHIVED' && lead.status !== 'EXPORTED_TO_MUNINN' ? (
                  <MenuItem icon="xCircle" danger onClick={() => setDisqualifyTarget(lead)}>
                    {t('leads.disqualify')}
                  </MenuItem>
                ) : null}
                {lead.status !== 'ARCHIVED' ? (
                  <MenuItem icon="archive" danger onClick={() => setArchiveTarget(lead)}>
                    {t('leads.deleteLead')}
                  </MenuItem>
                ) : null}
              </Dropdown>
            </div>
          )
        },
      },
    ]
    return cols
  }, [t, navigate, can, data, selected, locale])

  const selectedIds = Object.keys(selected).filter((k) => selected[k])

  return (
    <>
      <PageHeader
        title={t('leads.title')}
        subtitle={t('leads.subtitle')}
        actions={
          <>
            <Button variant="ghost" icon="filter" onClick={() => setFiltersOpen((o) => !o)} aria-expanded={filtersOpen}>
              {t('leads.filters')}
            </Button>
            {can(PERMISSIONS.leadsCreate) ? (
              <Button icon="plus" onClick={() => setCreateOpen(true)}>
                {t('leads.newLead')}
              </Button>
            ) : null}
          </>
        }
      />

      <div className="toolbar mb-4">
        <div className="search-input-wrap" style={{ width: 320, maxWidth: '100%' }}>
          <Icon name="search" />
          <input
            type="search"
            className="input"
            placeholder={t('leads.searchCompany')}
            value={url.search ?? ''}
            onChange={(e) => setUrl({ search: e.target.value, page: '1' })}
            aria-label={t('common.search')}
          />
        </div>
        {isFetching ? <span className="spinner" aria-hidden="true" /> : null}
        <div className="spacer" />
        {selectedIds.length > 0 ? (
          <span className="small secondary">
            {t('common.selected')}: {toFaDigits(selectedIds.length)}
          </span>
        ) : null}
        {selectedIds.length > 0 && can(PERMISSIONS.leadsAssign) ? (
          <Button size="sm" icon="users" onClick={() => setAssignTarget({ leadIds: selectedIds })}>
            {t('assignments.assignSelected')}
          </Button>
        ) : null}
        {url.status ? (
          <Button size="sm" variant="ghost" onClick={() => setUrl({ status: '', page: '1' })}>
            <Icon name="x" size={13} />
            {t(`status.${url.status}`)}
          </Button>
        ) : null}
        {url.unassigned ? (
          <Button size="sm" variant="ghost" onClick={() => setUrl({ unassigned: '', page: '1' })}>
            <Icon name="x" size={13} />
            {t('nav.unassignedLeads')}
          </Button>
        ) : null}
        {url.review ? (
          <Button size="sm" variant="ghost" onClick={() => setUrl({ review: '', page: '1' })}>
            <Icon name="x" size={13} />
            {t('nav.reviewRequired')}
          </Button>
        ) : null}
      </div>

      {filtersOpen ? (
        <div className="filter-bar mb-4">
          <Select label={t('leads.status')} value={draft.status ?? ''} onChange={(e) => setDraft((d) => ({ ...d, status: e.target.value }))}>
            <option value="">{t('leads.allStatuses')}</option>
            {['NEW', 'READY', 'ASSIGNED', 'CONTACTING', 'FOLLOW_UP', 'QUALIFIED', 'DISQUALIFIED', 'EXPORTED_TO_MUNINN', 'ARCHIVED'].map((s) => (
              <option key={s} value={s}>
                {t(`status.${s}`)}
              </option>
            ))}
          </Select>
          <Select label={t('leads.businessType')} value={draft.business_type ?? ''} onChange={(e) => setDraft((d) => ({ ...d, business_type: e.target.value }))}>
            <option value="">{t('common.all')}</option>
            {['تولیدکننده', 'پخش‌کننده', 'خدمات', 'خرده‌فروش', 'عمده‌فروش'].map((b) => (
              <option key={b} value={b}>
                {b}
              </option>
            ))}
          </Select>
          <Input label={`${t('leads.score')} ${t('common.total')}`} placeholder={t('leads.score')} value={draft.score_min ?? ''} onChange={(e) => setDraft((d) => ({ ...d, score_min: e.target.value }))} />
          <Select label={t('leads.province')} value={draft.province ?? ''} onChange={(e) => setDraft((d) => ({ ...d, province: e.target.value }))}>
            <option value="">{t('common.all')}</option>
            {['تهران', 'خراسان رضوی', 'اصفهان', 'فارس', 'آذربایجان شرقی', 'البرز', 'قم', 'خوزستان'].map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </Select>
          <Input label={t('leads.city')} placeholder={t('leads.city')} value={draft.city ?? ''} onChange={(e) => setDraft((d) => ({ ...d, city: e.target.value }))} />
          <Input label={t('leads.source')} placeholder={t('leads.source')} value={draft.source ?? ''} onChange={(e) => setDraft((d) => ({ ...d, source: e.target.value }))} />
          <Select label={t('leads.owner')} value={draft.owner ?? ''} onChange={(e) => setDraft((d) => ({ ...d, owner: e.target.value }))}>
            <option value="">{t('common.all')}</option>
            {(users ?? []).map((u) => (
              <option key={u.id} value={u.id}>
                {u.name}
              </option>
            ))}
          </Select>
          <Input label={t('leads.tag')} placeholder={t('leads.tag')} value={draft.tag ?? ''} onChange={(e) => setDraft((d) => ({ ...d, tag: e.target.value }))} />
          {can(PERMISSIONS.projectsManage) ? (
            <Select label={t('projects.title')} value={draft.project_id ?? ''} onChange={(e) => setDraft((d) => ({ ...d, project_id: e.target.value }))}>
              <option value="">{t('common.all')}</option>
              {projectOptions.map((p) => (
                <option key={p.id} value={p.id}>{p.title}</option>
              ))}
            </Select>
          ) : null}
          <div className="filter-actions">
            <Button variant="ghost" onClick={() => setDraft(defaultFilters)}>
              {t('leads.resetFilters')}
            </Button>
            <Button
              icon="check"
              onClick={() => {
                setUrl({ ...draft, page: '1' })
                setFiltersOpen(false)
              }}
            >
              {t('common.apply')}
            </Button>
          </div>
        </div>
      ) : null}

      {isError ? (
        <div className="card">
          <div className="state-block">
            <Icon name="alertTriangle" size={42} />
            <h3>{t('errors.generic')}</h3>
            <button className="btn btn-secondary" onClick={() => void refetch()}>
              {t('common.retry')}
            </button>
          </div>
        </div>
      ) : (
        <DataTable
          columns={columns}
          data={data?.items ?? []}
          meta={data?.meta}
          onPage={(p) => setUrl({ page: String(p) })}
          loading={isLoading}
          emptyTitle={t('leads.noLeads')}
          emptyHint={url.status || url.search ? undefined : t('leads.noLeadsHint')}
          rowCountLabel={`${t('leads.total')}: ${toFaDigits(data?.meta.total ?? 0)}`}
        />
      )}

      <Dialog
        open={!!assignTarget}
        onClose={() => setAssignTarget(null)}
        title={t('leads.assign')}
        footer={
          <>
            <Button onClick={() => setAssignTarget(null)}>{t('common.cancel')}</Button>
            {assignTarget ? (
              <Button
                variant="primary"
                disabled={!assignTarget.assigneeId}
                loading={listLeads.isPending}
                onClick={() => assignTarget.assigneeId && listLeads.mutate({ leadIds: assignTarget.leadIds, assigneeId: assignTarget.assigneeId })}
              >
                {t('common.confirm')}
              </Button>
            ) : null}
          </>
        }
      >
        {assignTarget ? (
          <AssignForm
            agents={agents}
            onPick={(assigneeId) => setAssignTarget({ ...assignTarget, assigneeId })}
          />
        ) : null}
      </Dialog>

      <ConfirmDialog
        open={!!disqualifyTarget}
        onClose={() => setDisqualifyTarget(null)}
        onConfirm={() => disqualifyTarget && disqualify.mutate({ id: disqualifyTarget.id, reason: disqualifyReason })}
        title={t('leads.disqualify')}
        message={t('leads.confirmDisqualify')}
        loading={disqualify.isPending}
      >
        <div className="mt-4">
          <Textarea label={t('leads.disqualifyReason')} placeholder={t('leads.disqualifyPlaceholder')} value={disqualifyReason} onChange={(e) => setDisqualifyReason(e.target.value)} />
        </div>
      </ConfirmDialog>

      <ConfirmDialog
        open={!!exportTarget}
        onClose={() => setExportTarget(null)}
        onConfirm={() => exportTarget && exportMuninn.mutate(exportTarget.id)}
        title={t('leads.transferMuninn')}
        message={t('leads.confirmExport')}
        danger={false}
        loading={exportMuninn.isPending}
      />

      <ConfirmDialog
        open={!!archiveTarget}
        onClose={() => setArchiveTarget(null)}
        onConfirm={() => archiveTarget && archive.mutate(archiveTarget.id)}
        title={t('leads.deleteLead')}
        message={t('leads.confirmDeleteLead')}
        loading={archive.isPending}
      />

      <LeadFormDialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title={t('leads.newLead')}
        loading={createLead.isPending}
        projectOptions={projectOptions}
        onSubmit={(data) => createLead.mutate(data)}
      />

      <LeadFormDialog
        open={!!editLead}
        onClose={() => setEditLead(null)}
        title={t('leads.editLead')}
        initial={editLead}
        loading={updateLead.isPending}
        projectOptions={projectOptions}
        onSubmit={(data) => updateLead.mutate(data)}
      />
    </>
  )
}

const LEAD_SOURCES = ['لیست تولیدکنندگان', 'وب‌سایت شرکت', 'اینستاگرام', 'نمایشگاه', 'اتاق بازرگانی', 'دیتای بازار', 'دستی']

function LeadFormDialog({
  open,
  onClose,
  title,
  initial,
  loading,
  projectOptions,
  onSubmit,
}: {
  open: boolean
  onClose: () => void
  title: string
  initial?: Lead | null
  loading: boolean
  projectOptions: Array<Project & { lead_count: number }>
  onSubmit: (data: Partial<Lead>) => void
}) {
  const { t } = useI18n()
  const { can } = useAuth()
  const [form, setForm] = useState<Record<string, string>>(() => leadFormDefaults(initial))

  useEffect(() => {
    if (open) setForm(leadFormDefaults(initial))
  }, [open, initial])

  const set = (k: string, v: string) => setForm((f) => ({ ...f, [k]: v }))
  const canSubmit = (form.company_name ?? '').trim().length >= 2
  const showProject = can(PERMISSIONS.projectsManage)

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      footer={
        <>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button variant="primary" disabled={!canSubmit} loading={loading} onClick={() => onSubmit(leadFormPayload(form))}>
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
          <label className="field-label" htmlFor="lead-source-create">
            {t('leads.source')}
          </label>
          <input id="lead-source-create" className="input" list="lead-sources" placeholder={t('leads.sourcePlaceholder')} value={form.source ?? ''} onChange={(e) => set('source', e.target.value)} />
          <datalist id="lead-sources">
            {LEAD_SOURCES.map((s) => (
              <option key={s} value={s} />
            ))}
          </datalist>
        </div>
        {showProject ? (
          <div className="field">
            <Select label={t('projects.title')} value={form.project_id ?? ''} onChange={(e) => set('project_id', e.target.value)}>
              <option value="">{t('common.none')}</option>
              {projectOptions.map((p) => (
                <option key={p.id} value={p.id}>{p.title}</option>
              ))}
            </Select>
          </div>
        ) : null}
      </div>
      <div className="field">
        <Textarea label={t('leads.description')} value={form.description ?? ''} onChange={(e) => set('description', e.target.value)} style={{ minHeight: 70 }} />
      </div>
      <div className="form-row">
        <Input label={t('leads.website')} dir="ltr" value={form.website ?? ''} onChange={(e) => set('website', e.target.value)} />
        <Input label={t('leads.instagram')} dir="ltr" value={form.instagram ?? ''} onChange={(e) => set('instagram', e.target.value)} />
      </div>
    </Dialog>
  )
}

function leadFormDefaults(initial?: Lead | null): Record<string, string> {
  return {
    company_name: initial?.company_name ?? '',
    business_type: initial?.business_type ?? '',
    industry: initial?.industry ?? '',
    province: initial?.province ?? '',
    city: initial?.city ?? '',
    source: initial?.source ?? '',
    description: initial?.description ?? '',
    website: initial?.website ?? '',
    instagram: initial?.instagram ?? '',
    project_id: initial?.project_id ?? '',
  }
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

function AssignForm({ agents, onPick }: { agents: Array<{ id: string; name: string }>; onPick: (id: string) => void }) {
  const { t } = useI18n()
  return (
    <div className="field">
      <label className="field-label">{t('leads.owner')}</label>
      <Select onChange={(e) => onPick(e.target.value)} defaultValue="">
        <option value="" disabled>
          {t('common.choose')}
        </option>
        {agents.map((a) => (
          <option key={a.id} value={a.id}>
            {a.name}
          </option>
        ))}
      </Select>
    </div>
  )
}
