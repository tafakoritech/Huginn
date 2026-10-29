import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import type { ColumnDef } from '@tanstack/react-table'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits, formatDateTime } from '@/utils/format'
import { PageHeader } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import { Select } from '@/components/ui/Field'
import type { ContactReport } from '@/api/types'

const REPORT_FILTERS = [
  { value: '', labelKey: 'common.all' },
  { value: 'PENDING', labelKey: 'reports.filterPending' },
  { value: 'SUBMITTED', labelKey: 'reports.filterSubmitted' },
  { value: 'OVERRIDE', labelKey: 'reports.filterOverride' },
]

export function ReportsPage() {
  const { t, locale } = useI18n()
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState('')
  const query = useMemo(() => ({ page, page_size: 20, status }), [page, status])
  const { data, isLoading } = useQuery({ queryKey: queryKeys.reports(query), queryFn: () => api.listReports(query) })

  const columns = useMemo<ColumnDef<ContactReport, unknown>[]>(() => {
    return [
      {
        id: 'lead',
        header: t('reports.lead'),
        cell: ({ row }) => row.original.interaction?.lead?.company_name ?? <span className="muted">—</span>,
        enableSorting: false,
      },
      {
        id: 'agent',
        header: t('reports.agent'),
        cell: ({ row }) => row.original.interaction?.user?.name ?? row.original.submitter?.name ?? <span className="muted">—</span>,
        enableSorting: false,
      },
      {
        id: 'channel',
        header: t('reports.channel'),
        cell: ({ row }) => row.original.interaction?.channel ?? '—',
        enableSorting: false,
      },
      {
        id: 'time',
        header: t('reports.time'),
        cell: ({ row }) => formatDateTime(row.original.interaction?.started_at ?? row.original.created_at, locale),
        enableSorting: true,
      },
      { accessorKey: 'outcome', header: t('reports.outcome') },
      { accessorKey: 'interest_level', header: t('reports.interest'), cell: ({ getValue }) => (getValue() as string) || '—' },
      { accessorKey: 'next_action', header: t('reports.nextAction'), cell: ({ getValue }) => (getValue() as string) || '—' },
      {
        id: 'report_status',
        header: t('reports.reportStatus'),
        cell: ({ row }) => {
          const r = row.original
          const overdue = r.interaction && !r.interaction.ended_at && r.created_at
          return (
            <div className="flex flex-col gap-1">
              {r.manager_override ? <span className="badge badge-info">{t('reports.managerOverride')}</span> : <span className="badge badge-success">{t('status.SUBMITTED')}</span>}
              {overdue ? <span className="report-overdue">{t('reports.overduePending')}</span> : null}
            </div>
          )
        },
        enableSorting: false,
      },
    ]
  }, [t, locale])

  return (
    <>
      <PageHeader title={t('reports.title')} subtitle={t('reports.subtitle')} actions={
        <div style={{ width: 200 }}>
          <Select value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }} aria-label={t('common.filters')}>
            {REPORT_FILTERS.map((f) => (
              <option key={f.value} value={f.value}>
                {t(f.labelKey)}
              </option>
            ))}
          </Select>
        </div>
      } />

      <DataTable
        columns={columns}
        data={data?.items ?? []}
        meta={data?.meta}
        onPage={setPage}
        loading={isLoading}
        emptyTitle={t('reports.noReports')}
        onRowClick={(r) => navigate(`/reports/${r.id}`)}
        rowCountLabel={`${t('common.total')}: ${toFaDigits(data?.meta.total ?? 0)}`}
      />
    </>
  )
}
