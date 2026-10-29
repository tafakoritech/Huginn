import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import type { ColumnDef } from '@tanstack/react-table'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits, formatDateTime, formatBytes } from '@/utils/format'
import { PageHeader } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import type { SourceDocument } from '@/api/types'

export function DocumentsPage() {
  const { t, locale } = useI18n()
  const [page, setPage] = useState(1)
  const query = useMemo(() => ({ page, page_size: 20 }), [page])
  const { data, isLoading } = useQuery({ queryKey: queryKeys.documents(query), queryFn: () => api.listDocuments(query) })

  const columns = useMemo<ColumnDef<SourceDocument, unknown>[]>(() => {
    return [
      {
        accessorKey: 'original_filename',
        header: t('documents.filename'),
        cell: ({ row }) => (
          <Link to={`/documents/${row.original.id}`} className="strong" style={{ color: 'var(--text-1)' }}>
            {row.original.original_filename}
          </Link>
        ),
      },
      {
        accessorKey: 'batch_name',
        header: t('documents.batch'),
        cell: ({ row }) => (row.original.batch_id ? <span className="chip">{row.original.batch_name ?? row.original.batch_id}</span> : <span className="muted">—</span>),
        enableSorting: false,
      },
      {
        accessorKey: 'mime_type',
        header: t('imports.sourceType'),
        cell: ({ getValue }) => <span className="chip">{String(getValue() ?? '—')}</span>,
        enableSorting: false,
      },
      { accessorKey: 'size_bytes', header: t('documents.size'), cell: ({ row }) => formatBytes(row.original.size_bytes) },
      {
        accessorKey: 'status',
        header: t('documents.processingStatus'),
        cell: ({ row }) => {
          const s = row.original.status
          const tone = s === 'COMPLETED' ? 'success' : s === 'FAILED' ? 'error' : s === 'EXTRACTING' || s === 'ANALYZING' ? 'info' : 'neutral'
          return (
            <span className={`badge ${tone === 'success' ? 'badge-success' : tone === 'error' ? 'badge-error' : tone === 'info' ? 'badge-info' : 'badge-neutral'}`}>
              {s}
            </span>
          )
        },
        enableSorting: false,
      },
      {
        id: 'extracted_text_status',
        header: t('documents.extractedTextStatus'),
        cell: ({ row }) =>
          row.original.extracted_text ? (
            <span className="badge badge-success">{t('common.yes')}</span>
          ) : row.original.extraction_error ? (
            <span className="badge badge-error">{t('common.no')}</span>
          ) : (
            <span className="muted">—</span>
          ),
        enableSorting: false,
      },
      { accessorKey: 'lead_count', header: t('documents.leadCount'), cell: ({ row }) => toFaDigits(row.original.lead_count) },
      { accessorKey: 'created_at', header: t('common.createdAt'), cell: ({ row }) => formatDateTime(row.original.created_at, locale, { time: false }) },
    ]
  }, [t, locale])

  return (
    <>
      <PageHeader title={t('documents.title')} subtitle={t('documents.subtitle')} />
      <DataTable
        columns={columns}
        data={data?.items ?? []}
        meta={data?.meta}
        onPage={setPage}
        loading={isLoading}
        emptyTitle={t('documents.noDocuments')}
        rowCountLabel={`${t('common.total')}: ${toFaDigits(data?.meta.total ?? 0)}`}
      />
    </>
  )
}
