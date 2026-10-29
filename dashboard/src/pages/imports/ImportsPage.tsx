import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import type { ColumnDef } from '@tanstack/react-table'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits, formatDateTime } from '@/utils/format'
import { PageHeader } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import { ImportStatusBadge } from '@/components/ui/Badge'
import { Icon } from '@/components/Icon'
import type { ImportBatch } from '@/api/types'

export function ImportsPage() {
  const { t, locale } = useI18n()
  const [page, setPage] = useState(1)
  const query = useMemo(() => ({ page, page_size: 20 }), [page])
  const { data, isLoading } = useQuery({ queryKey: queryKeys.imports(query), queryFn: () => api.listImports(query) })

  const columns = useMemo<ColumnDef<ImportBatch, unknown>[]>(() => {
    return [
      {
        accessorKey: 'name',
        header: t('imports.batchName'),
        cell: ({ row }) => (
          <Link to={`/imports/${row.original.id}`} className="strong" style={{ color: 'var(--text-1)' }}>
            {row.original.name}
          </Link>
        ),
      },
      {
        accessorKey: 'source_type',
        header: t('imports.sourceType'),
        cell: ({ getValue }) => <span className="chip">{String(getValue())}</span>,
      },
      {
        id: 'creator',
        header: t('imports.creator'),
        cell: ({ row }) => row.original.creator?.name ?? <span className="muted">—</span>,
        enableSorting: false,
      },
      {
        accessorKey: 'total_documents',
        header: t('imports.documents'),
        cell: ({ row }) => toFaDigits(row.original.total_documents),
      },
      {
        accessorKey: 'total_candidates',
        header: t('imports.candidates'),
        cell: ({ row }) => toFaDigits(row.original.total_candidates),
      },
      {
        accessorKey: 'created_leads',
        header: t('imports.leadsCreated'),
        cell: ({ row }) => toFaDigits(row.original.created_leads),
      },
      {
        accessorKey: 'duplicate_candidates',
        header: t('imports.duplicates'),
        cell: ({ row }) => toFaDigits(row.original.duplicate_candidates),
      },
      {
        accessorKey: 'failed_items',
        header: t('imports.errors'),
        cell: ({ row }) =>
          row.original.failed_items > 0 ? (
            <span className="badge badge-error">{toFaDigits(row.original.failed_items)}</span>
          ) : (
            <span className="muted">0</span>
          ),
      },
      {
        accessorKey: 'status',
        header: t('common.status'),
        cell: ({ row }) => <ImportStatusBadge status={row.original.status} />,
        enableSorting: false,
      },
      {
        accessorKey: 'created_at',
        header: t('common.createdAt'),
        cell: ({ row }) => formatDateTime(row.original.created_at, locale, { time: false }),
        enableSorting: true,
      },
    ]
  }, [t, locale])

  return (
    <>
      <PageHeader
        title={t('imports.title')}
        subtitle={t('imports.subtitle')}
        actions={
          <Link to="/imports/new" className="btn btn-primary">
            <Icon name="plus" size={16} />
            {t('imports.newImport')}
          </Link>
        }
      />

      <DataTable
        columns={columns}
        data={data?.items ?? []}
        meta={data?.meta}
        onPage={setPage}
        loading={isLoading}
        emptyTitle={t('imports.noImports')}
        emptyHint={t('imports.noImportsHint')}
        rowCountLabel={`${t('common.total')}: ${toFaDigits(data?.meta.total ?? 0)}`}
      />
    </>
  )
}
