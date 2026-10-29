import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import type { ColumnDef } from '@tanstack/react-table'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { useAuth, PERMISSIONS } from '@/auth/AuthContext'
import { toFaDigits, formatDateTime } from '@/utils/format'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import { ScoreBadge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/Icon'
import { ConfirmDialog } from '@/components/ui/Dialog'
import { Select } from '@/components/ui/Field'
import type { Lead } from '@/api/types'

const EXPORT_TONES: Record<string, 'neutral' | 'info' | 'success' | 'error' | 'warning'> = {
  PENDING: 'neutral',
  IN_PROGRESS: 'info',
  SUCCEEDED: 'success',
  FAILED: 'error',
  SKIPPED: 'warning',
}

export function QualifiedPage() {
  const { t, locale } = useI18n()
  const { can } = useAuth()
  const toast = useToastApi()
  const qc = useQueryClient()
  const [page, setPage] = useState(1)
  const [owner, setOwner] = useState('')
  const [exportTarget, setExportTarget] = useState<Lead | null>(null)

  const query = useMemo(() => {
    const q: Record<string, string> = { page: String(page), page_size: '25', qualified: '1' }
    if (owner) q.owner = owner
    return q
  }, [page, owner])

  const { data, isLoading } = useQuery({ queryKey: queryKeys.leads(query), queryFn: () => api.listLeads(query as Parameters<typeof api.listLeads>[0]) })

  const { data: users } = useQuery({ queryKey: queryKeys.users, queryFn: () => api.listUsers(), enabled: can(PERMISSIONS.leadsAssign) })

  const exportMuninn = useMutation({
    mutationFn: (id: string) => api.exportLeadToMuninn(id),
    onSuccess: () => {
      toast.success(t('leads.exportQueued'))
      setExportTarget(null)
      void qc.invalidateQueries({ queryKey: queryKeys.leads(query) })
      void qc.invalidateQueries({ queryKey: queryKeys.lead(exportTarget?.id ?? '') })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const columns = useMemo<ColumnDef<Lead, unknown>[]>(() => {
    const base: ColumnDef<Lead, unknown>[] = [
      {
        accessorKey: 'company_name',
        header: t('leads.company'),
        cell: ({ row }) => (
          <Link to={`/leads/${row.original.id}`} className="strong" style={{ color: 'var(--text-1)' }}>
            {row.original.company_name}
          </Link>
        ),
      },
      { accessorKey: 'score', header: t('leads.score'), cell: ({ row }) => <ScoreBadge score={row.original.score} />, enableSorting: true },
      {
        id: 'decision_maker',
        header: t('leads.decisionMaker'),
        cell: ({ row }) => (row.original.primary_contact?.is_decision_maker ? <span>{row.original.primary_contact.full_name}</span> : <span className="muted">—</span>),
        enableSorting: false,
      },
      { id: 'need', header: t('leads.need'), cell: ({ row }) => row.original.qualification?.need_interest ? <span>✓</span> : <span className="muted">—</span>, enableSorting: false },
      { id: 'interest', header: t('leads.interest'), cell: ({ row }) => row.original.qualification?.need_interest ? <span>✓</span> : <span className="muted">—</span>, enableSorting: false },
      { id: 'next_action', header: t('leads.nextAction'), cell: ({ row }) => row.original.qualification?.next_action ? <span>✓</span> : <span className="muted">—</span>, enableSorting: false },
      {
        id: 'qual_date',
        header: t('leads.qualificationDate'),
        cell: ({ row }) => (row.original.qualification?.decided_at ? formatDateTime(row.original.qualification.decided_at, locale, { time: false }) : <span className="muted">—</span>),
        enableSorting: false,
      },
      { id: 'owner', header: t('leads.owner'), cell: ({ row }) => row.original.owner?.name ?? <span className="muted">—</span>, enableSorting: false },
      {
        id: 'export',
        header: t('leads.muninnId'),
        cell: ({ row }) => {
          const exp = row.original.crm_export
          if (!exp) return <span className="muted">—</span>
          if (exp.status === 'SUCCEEDED') {
            return (
              <span className="badge badge-brand">
                {exp.muninn_lead_id ?? '✓'}
              </span>
            )
          }
          const tone = EXPORT_TONES[exp.status] ?? 'neutral'
          return (
            <div className="flex flex-col gap-1">
              <span className={`badge ${tone === 'success' ? 'badge-success' : tone === 'error' ? 'badge-error' : tone === 'info' ? 'badge-info' : tone === 'warning' ? 'badge-warning' : 'badge-neutral'}`}>
                {t(`status.${exp.status}`)}
              </span>
              {exp.status === 'FAILED' ? (
                <span className="small muted">
                  {t('common.total')}: {toFaDigits(exp.attempt_count)} · {exp.last_error}
                </span>
              ) : null}
            </div>
          )
        },
        enableSorting: false,
      },
    ]
    if (can(PERMISSIONS.leadsExport)) {
      base.push({
        id: 'exported_at',
        header: t('leads.exportedAt'),
        cell: ({ row }) => {
          const exp = row.original.crm_export
          return exp?.exported_at ? formatDateTime(exp.exported_at, locale) : <span className="muted">—</span>
        },
        enableSorting: false,
      })
      base.push({
        id: 'actions',
        header: t('common.actions'),
        enableSorting: false,
        cell: ({ row }) => {
          const exp = row.original.crm_export
          const isExported = exp?.status === 'SUCCEEDED' || row.original.status === 'EXPORTED_TO_MUNINN'
          const exporting = exp && (exp.status === 'IN_PROGRESS' || exp.status === 'PENDING')
          return (
            <div className="row-actions">
              {exporting ? (
                <span className="badge badge-info">
                  <Icon name="activity" size={12} />
                  {t('status.PENDING')}
                </span>
              ) : (
                <Button size="sm" variant="primary" icon="muninn" disabled={isExported} onClick={() => setExportTarget(row.original)}>
                  {isExported ? t('leads.muninnId') : t('leads.exportToMuninn')}
                </Button>
              )}
            </div>
          )
        },
      })
    }
    return base
  }, [t, locale, can])

  return (
    <>
      <PageHeader
        title={t('leads.qualifiedTitle')}
        subtitle={t('leads.qualifiedSubtitle')}
        actions={
          users && users.filter((u) => u.role === 'sales_agent').length > 0 ? (
            <div style={{ width: 200 }}>
              <Select value={owner} onChange={(e) => { setOwner(e.target.value); setPage(1) }} label={t('leads.owner')}>
                <option value="">{t('common.all')}</option>
                {users.filter((u) => u.role === 'sales_agent').map((u) => (
                  <option key={u.id} value={u.id}>
                    {u.name}
                  </option>
                ))}
              </Select>
            </div>
          ) : null
        }
      />

      <DataTable
        columns={columns}
        data={data?.items ?? []}
        meta={data?.meta}
        onPage={setPage}
        loading={isLoading}
        emptyTitle={t('leads.noQualified')}
        emptyActions={
          <Link to="/leads" className="btn btn-secondary btn-sm">
            {t('leads.view')}
          </Link>
        }
        rowCountLabel={`${t('leads.total')}: ${toFaDigits(data?.meta.total ?? 0)}`}
      />

      <ConfirmDialog
        open={!!exportTarget}
        onClose={() => setExportTarget(null)}
        onConfirm={() => exportTarget && exportMuninn.mutate(exportTarget.id)}
        title={t('leads.transferMuninn')}
        message={t('leads.confirmExport')}
        danger={false}
        loading={exportMuninn.isPending}
      />
    </>
  )
}
