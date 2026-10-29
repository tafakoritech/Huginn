import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits, formatDateTime } from '@/utils/format'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader, Tabs } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/Dialog'
import { Icon } from '@/components/Icon'
import { useAuth, PERMISSIONS } from '@/auth/AuthContext'
import type { ColumnDef } from '@tanstack/react-table'
import type { Lead, ProjectDetail } from '@/api/types'
import { LeadPickerDialog } from './LeadPickerDialog'

export function ProjectDetailPage() {
  const { t, locale } = useI18n()
  const { can } = useAuth()
  const toast = useToastApi()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const [tab, setTab] = useState('leads')
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [pickerOpen, setPickerOpen] = useState(false)
  const [detachTarget, setDetachTarget] = useState<Lead | null>(null)

  const { data: detail, isLoading } = useQuery({
    queryKey: queryKeys.project(id ?? ''),
    queryFn: () => api.getProject(id ?? ''),
    enabled: !!id,
    staleTime: 30_000,
  })

  const project = detail?.project

  const invalidateProject = () => {
    void qc.invalidateQueries({ queryKey: queryKeys.project(id ?? '') })
    void qc.invalidateQueries({ queryKey: queryKeys.projects() })
  }

  const attachLeads = useMutation({
    mutationFn: (leadIds: string[]) => Promise.all(leadIds.map((leadId) => api.attachLeadToProject(id ?? '', leadId))),
    onSuccess: () => {
      toast.success(t('projects.attachSuccess'))
      setPickerOpen(false)
      invalidateProject()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const detachLead = useMutation({
    mutationFn: (leadId: string) => api.detachLeadFromProject(id ?? '', leadId),
    onSuccess: () => {
      toast.success(t('projects.detachSuccess'))
      setDetachTarget(null)
      invalidateProject()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const deleteProject = useMutation({
    mutationFn: (pid: string) => api.deleteProject(pid),
    onSuccess: () => {
      toast.success(t('projects.deleteSuccess'))
      void qc.invalidateQueries({ queryKey: queryKeys.projects() })
      navigate('/projects')
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const tabs: Array<{ key: string; label: string }> = []
  tabs.push({ key: 'leads', label: `${t('projects.leadsTab')} (${toFaDigits(detail?.related_leads.total ?? 0)})` })
  tabs.push({ key: 'activities', label: `${t('projects.activitiesTab')} (${toFaDigits(detail?.sales_activities.length ?? 0)})` })

  const leadColumns = useMemo<ColumnDef<Lead, unknown>[]>(
    () => [
      {
        accessorKey: 'company_name',
        header: t('leads.company'),
        cell: ({ row }) => (
          <Link to={`/leads/${row.original.id}`} className="strong" style={{ color: 'var(--text-1)' }}>
            {row.original.company_name}
          </Link>
        ),
      },
      { accessorKey: 'industry', header: t('leads.industry'), cell: ({ getValue }) => <span>{String(getValue()) || '—'}</span> },
      { accessorKey: 'city', header: t('leads.city'), cell: ({ getValue }) => <span>{String(getValue()) || '—'}</span> },
      {
        accessorKey: 'status',
        header: t('leads.status'),
        cell: ({ row }) => <Badge tone="accent">{t(`status.${row.original.status}`)}</Badge>,
      },
      {
        id: 'owner',
        header: t('leads.owner'),
        cell: ({ row }) => row.original.owner?.name ?? row.original.owner_name ?? <span className="muted">—</span>,
        enableSorting: false,
      },
      {
        id: 'last_activity',
        header: t('projects.lastActivity'),
        cell: ({ row }) => {
          const at = row.original.last_contact_at ?? row.original.updated_at
          return at ? formatDateTime(at, locale, { time: false }) : <span className="muted">—</span>
        },
        enableSorting: false,
      },
      {
        id: 'actions',
        header: t('common.actions'),
        enableSorting: false,
        cell: ({ row }) => {
          const lead = row.original
          return (
            <div className="row-actions">
              <Link to={`/leads/${lead.id}`} className="btn btn-ghost btn-sm" aria-label={t('common.view')}>
                <Icon name="eye" size={16} />
              </Link>
              {can(PERMISSIONS.projectsManage) ? (
                <Button size="sm" variant="ghost" icon="trash" onClick={() => setDetachTarget(lead)} aria-label={t('projects.removeConnection')} />
              ) : null}
            </div>
          )
        },
      },
    ],
    [t, locale, can],
  )

  const activityColumns = useMemo<ColumnDef<ProjectDetail['sales_activities'][number], unknown>[]>(
    () => [
      {
        accessorKey: 'opening_sentence',
        header: t('projects.openingSentence'),
        cell: ({ getValue }) => <span>{String(getValue())}</span>,
      },
      {
        accessorKey: 'is_generated',
        header: '#',
        cell: ({ row }) =>
          row.original.generation_status === 'degraded' || row.original.fallback_used ? (
            <Badge tone="warning">{t('projects.scenarioFallback')}</Badge>
          ) : row.original.is_generated ? (
            <Badge tone="success">{t('projects.isGenerated')}</Badge>
          ) : (
            <Badge tone="neutral">—</Badge>
          ),
        enableSorting: false,
      },
      {
        accessorKey: 'lead_id',
        header: t('leads.lead'),
        cell: ({ row }) => (
          <Link to={`/leads/${row.original.lead_id}`} className="strong" style={{ color: 'var(--text-1)' }}>
            {row.original.lead_id}
          </Link>
        ),
      },
      {
        accessorKey: 'created_at',
        header: t('common.createdAt'),
        cell: ({ row }) => formatDateTime(row.original.created_at, locale, { time: false }),
      },
    ],
    [t, locale],
  )

  if (isLoading || !project) {
    return (
      <>
        <PageHeader title={t('projects.title')} subtitle={t('projects.subtitle')} breadcrumbs={[{ label: t('nav.projects'), to: '/projects' }]} />
        <div className="p-6">{t('common.loading')}</div>
      </>
    )
  }

  const manage = can(PERMISSIONS.projectsManage)
  const relatedLeads = detail?.related_leads.leads ?? []

  return (
    <>
      <PageHeader
        title={project.title}
        subtitle={project.description}
        breadcrumbs={[{ label: t('nav.projects'), to: '/projects' }]}
        actions={
          manage ? (
            <div className="row-actions">
              <Button size="sm" variant="primary" icon="plus" onClick={() => setPickerOpen(true)}>
                {t('projects.addExistingLead')}
              </Button>
              <Link to={`/imports/new?project=${encodeURIComponent(id ?? '')}`} className="btn btn-ghost btn-sm">
                <Icon name="import" size={16} />
                {t('imports.newImport')}
              </Link>
              <Button size="sm" variant="ghost" icon="trash" onClick={() => setDeleteOpen(true)}>
                {t('common.delete')}
              </Button>
            </div>
          ) : null
        }
      />

      <Tabs tabs={tabs} active={tab} onChange={setTab} />

      <div className="mt-5">
        {tab === 'leads' ? (
          <DataTable columns={leadColumns} data={relatedLeads} emptyTitle={t('projects.noProjects')} emptyHint={t('projects.addExistingLead')} />
        ) : (
          <DataTable columns={activityColumns} data={detail?.sales_activities ?? []} emptyTitle={t('projects.emptySalesActivities')} />
        )}
      </div>

      <LeadPickerDialog
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        attachedLeadIds={relatedLeads.map((l) => l.id)}
        onAttach={async (leadIds) => {
          await attachLeads.mutateAsync(leadIds)
        }}
      />

      <ConfirmDialog
        open={!!detachTarget}
        onClose={() => setDetachTarget(null)}
        onConfirm={() => detachTarget && detachLead.mutate(detachTarget.id)}
        title={t('projects.removeConnection')}
        message={detachTarget ? `${t('projects.detachConfirm')} (${detachTarget.company_name})` : ''}
        loading={detachLead.isPending}
      />

      <ConfirmDialog
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => {
          if (id) {
            deleteProject.mutate(id)
          }
        }}
        title={t('confirm.title')}
        message={t('projects.confirmDelete')}
        loading={deleteProject.isPending}
      />
    </>
  )
}
