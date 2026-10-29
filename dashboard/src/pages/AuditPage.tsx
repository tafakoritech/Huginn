import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits, formatDateTime } from '@/utils/format'
import { PageHeader } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Input, Select } from '@/components/ui/Field'
import type { AuditEntry } from '@/api/types'

export function AuditPage() {
  const { t, locale } = useI18n()
  const [page, setPage] = useState(1)
  const [action, setAction] = useState('')
  const [entityType, setEntityType] = useState('')
  const [userFilter, setUserFilter] = useState('')
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})

  const query = useMemo(() => ({ page, page_size: 20, action, entity_type: entityType, user_id: userFilter }), [page, action, entityType, userFilter])
  const { data, isLoading } = useQuery({ queryKey: queryKeys.audit(query), queryFn: () => api.listAudit(query) })

  const columns = useMemo<ColumnDef<AuditEntry, unknown>[]>(() => {
    return [
      { accessorKey: 'created_at', header: t('audit.time'), cell: ({ row }) => formatDateTime(row.original.created_at, locale), enableSorting: true },
      { id: 'actor', header: t('audit.actor'), cell: ({ row }) => row.original.actor?.name ?? row.original.actor_user_id ?? 'system', enableSorting: false },
      { accessorKey: 'action', header: t('audit.action'), cell: ({ row }) => <span className="mono" dir="ltr">{row.original.action}</span>, enableSorting: false },
      { accessorKey: 'entity_type', header: t('audit.entity'), cell: ({ row }) => <Badge tone="neutral">{row.original.entity_type}</Badge>, enableSorting: false },
      { accessorKey: 'entity_id', header: t('audit.entityId'), cell: ({ row }) => <span className="mono small" dir="ltr">{row.original.entity_id}</span>, enableSorting: false },
      { accessorKey: 'ip', header: t('audit.ip'), cell: ({ row }) => row.original.ip ?? <span className="muted">—</span>, enableSorting: false },
      {
        id: 'expand',
        header: '',
        enableSorting: false,
        cell: ({ row }) =>
          row.original.before_json || row.original.after_json ? (
            <Button size="sm" variant="ghost" icon="chevronDown" onClick={() => setExpanded((e) => ({ ...e, [row.original.id]: !e[row.original.id] }))} aria-expanded={!!expanded[row.original.id]}>
              {t('audit.expand')}
            </Button>
          ) : null,
      },
    ]
  }, [t, locale, expanded])

  return (
    <>
      <PageHeader title={t('audit.title')} subtitle={t('audit.subtitle')} />

      <div className="filter-bar mb-4">
        <Select label={t('audit.filterAction')} value={action} onChange={(e) => { setAction(e.target.value); setPage(1) }}>
          <option value="">{t('common.all')}</option>
          {['lead.qualify', 'lead.disqualify', 'lead.assign', 'lead.export', 'user.create', 'import.create', 'duplicates.merge', 'report.override'].map((a) => (
            <option key={a} value={a}>
              {a}
            </option>
          ))}
        </Select>
        <Select label={t('audit.filterEntity')} value={entityType} onChange={(e) => { setEntityType(e.target.value); setPage(1) }}>
          <option value="">{t('common.all')}</option>
          {['lead', 'user', 'import_batch', 'contact_report', 'duplicate', 'assignment'].map((et) => (
            <option key={et} value={et}>
              {et}
            </option>
          ))}
        </Select>
        <Input label={t('audit.filterUser')} placeholder={t('audit.actor')} value={userFilter} onChange={(e) => { setUserFilter(e.target.value); setPage(1) }} />
      </div>

      <DataTable
        columns={columns}
        data={data?.items ?? []}
        meta={data?.meta}
        onPage={setPage}
        loading={isLoading}
        emptyTitle={t('audit.noAudit')}
        footer={
          <div className="flex items-center justify-between w-full">
            <span className="small secondary">
              {t('common.total')}: {toFaDigits(data?.meta.total ?? 0)}
            </span>
          </div>
        }
      />

      {Object.keys(expanded).filter((k) => expanded[k]).length > 0 ? (
        <div className="mt-5 flex flex-col gap-4">
          {(data?.items ?? [])
            .filter((e) => expanded[e.id])
            .map((e) => (
              <div className="card" key={e.id}>
                <div className="card-header">
                  <h3>
                    <span className="mono" dir="ltr">{e.action}</span>
                  </h3>
                </div>
                <div className="card-body grid-2">
                  <AuditDiff title={t('audit.before')} data={e.before_json} />
                  <AuditDiff title={t('audit.after')} data={e.after_json} />
                </div>
              </div>
            ))}
        </div>
      ) : null}
    </>
  )
}

function AuditDiff({ title, data }: { title: string; data?: Record<string, unknown> | null }) {
  return (
    <div>
      <div className="small strong mb-2 secondary">{title}</div>
      {!data || Object.keys(data).length === 0 ? (
        <p className="muted small">—</p>
      ) : (
        <div className="diff-json">
          {Object.entries(data).map(([k, v]) => (
            <div key={k} className="diff-line key">
              <span className="diff-line key">{k}:</span> <span className="diff-line added">{typeof v === 'string' ? `"${v}"` : JSON.stringify(v)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
