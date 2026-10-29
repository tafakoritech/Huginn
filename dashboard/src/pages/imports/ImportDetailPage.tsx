import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import type { ColumnDef } from '@tanstack/react-table'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits, formatDateTime, formatBytes } from '@/utils/format'
import { PageHeader, BackButton } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import { ImportStatusBadge } from '@/components/ui/Badge'
import { LoadingState, ErrorState } from '@/components/ui/States'
import { Progress, Tabs } from '@/components/ui/misc'
import { Icon } from '@/components/Icon'
import type { SourceDocument, ProcessingStage } from '@/api/types'

export function ImportDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t, locale } = useI18n()
  const [tab, setTab] = useState('overview')

  const { data: batch, isLoading, isError, refetch } = useQuery({
    queryKey: queryKeys.import(id ?? ''),
    queryFn: () => api.getImport(id ?? ''),
    enabled: !!id,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'PROCESSING' || status === 'QUEUED' || status === 'UPLOADED' ? 4000 : false
    },
  })

  const { data: docsData } = useQuery({
    queryKey: queryKeys.importDocuments(id ?? ''),
    queryFn: () => api.importDocuments(id ?? ''),
    enabled: !!id,
  })

  const docs = docsData?.items ?? []

  const docColumns = useMemo<ColumnDef<SourceDocument, unknown>[]>(() => {
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
        accessorKey: 'mime_type',
        header: t('imports.sourceType'),
        cell: ({ getValue }) => <span className="chip">{String(getValue() ?? '—')}</span>,
      },
      {
        accessorKey: 'size_bytes',
        header: t('documents.size'),
        cell: ({ row }) => formatBytes(row.original.size_bytes),
      },
      {
        accessorKey: 'status',
        header: t('common.status'),
        cell: ({ row }) => (
          <span className={`badge ${row.original.status === 'COMPLETED' ? 'badge-success' : row.original.status === 'FAILED' ? 'badge-error' : row.original.status === 'UPLOADED' ? 'badge-neutral' : 'badge-info'}`}>
            {row.original.status}
          </span>
        ),
        enableSorting: false,
      },
      {
        accessorKey: 'created_at',
        header: t('common.createdAt'),
        cell: ({ row }) => formatDateTime(row.original.created_at, locale, { time: false }),
      },
      { accessorKey: 'lead_count', header: t('documents.leadCount'), cell: ({ row }) => toFaDigits(row.original.lead_count) },
    ]
  }, [t, locale])

  if (isLoading || !batch) return <LoadingState />
  if (isError) return <ErrorState detail={t('errors.generic')} onRetry={() => void refetch()} />

  const stages: Array<{ key: string; label: string; status: ProcessingStage['status'] }> = [
    { key: 'uploaded', label: t('imports.stages.uploaded'), status: 'done' },
    { key: 'extracting', label: t('imports.stages.extracting'), status: stageStatus(batch, 'extracting') },
    { key: 'analyzing', label: t('imports.stages.analyzing'), status: stageStatus(batch, 'analyzing') },
    { key: 'deduplicating', label: t('imports.stages.deduplicating'), status: stageStatus(batch, 'deduplicating') },
    { key: 'creating_leads', label: t('imports.stages.creating_leads'), status: stageStatus(batch, 'creating_leads') },
    { key: 'completed', label: t('imports.stages.completed'), status: batch.status === 'COMPLETED' ? 'done' : 'pending' },
  ]

  const activeIndex = stages.findIndex((s) => s.status === 'active')

  return (
    <>
      <BackButton to="/imports" />

      <PageHeader
        title={batch.name}
        subtitle={t('imports.detailTitle')}
        actions={<ImportStatusBadge status={batch.status} />}
      />

      <div className="processing-hero card">
        <div className="flex-1">
          <div className="flex justify-between small secondary mb-2">
            <span>{t('imports.progress')}</span>
            <span className="strong">{toFaDigits(batch.progress ?? progressFor(batch.status))}٪</span>
          </div>
          <Progress value={batch.progress ?? progressFor(batch.status)} tone={batch.status === 'FAILED' ? 'error' : batch.status === 'REVIEW_REQUIRED' ? 'warning' : 'accent'} />
          {batch.status === 'PROCESSING' || batch.status === 'QUEUED' || batch.status === 'UPLOADED' ? (
            <p className="small muted mt-2">
              <Icon name="activity" size={12} />
              {t('imports.processingNote')}
            </p>
          ) : null}
          {batch.error ? (
            <p className="badge badge-error mt-3" role="alert">
              {batch.error}
            </p>
          ) : null}
        </div>
        <div className="meta-grid">
          <div className="meta-item">
            <div className="meta-label">{t('imports.documents')}</div>
            <div className="meta-value">{toFaDigits(batch.total_documents)}</div>
          </div>
          <div className="meta-item">
            <div className="meta-label">{t('imports.candidates')}</div>
            <div className="meta-value">{toFaDigits(batch.total_candidates)}</div>
          </div>
          <div className="meta-item">
            <div className="meta-label">{t('imports.leadsCreated')}</div>
            <div className="meta-value">{toFaDigits(batch.created_leads)}</div>
          </div>
          <div className="meta-item">
            <div className="meta-label">{t('imports.duplicates')}</div>
            <div className="meta-value">{toFaDigits(batch.duplicate_candidates)}</div>
          </div>
          <div className="meta-item">
            <div className="meta-label">{t('imports.errors')}</div>
            <div className="meta-value" style={{ color: batch.failed_items > 0 ? 'var(--error)' : undefined }}>
              {toFaDigits(batch.failed_items)}
            </div>
          </div>
          <div className="meta-item">
            <div className="meta-label">{t('common.createdAt')}</div>
            <div className="meta-value" style={{ fontSize: 12 }}>
              {formatDateTime(batch.created_at, locale, { time: false })}
            </div>
          </div>
        </div>
      </div>

      <div className="card mb-5" style={{ padding: 16 }}>
        <div className="step-indicator">
          {stages.map((s, i) => (
            <span key={s.key} className="flex items-center">
              <span className={`step ${s.status === 'done' ? 'done' : s.status === 'active' ? 'active' : s.status === 'error' ? 'error' : ''}`.trim()}>
                <span className="step-dot" />
                {s.label}
              </span>
              {i < stages.length - 1 ? <span className="step-sep" /> : null}
            </span>
          ))}
        </div>
        {activeIndex >= 0 ? <p className="small muted mt-2">{stages[activeIndex].label}...</p> : null}
      </div>

      <Tabs tabs={[{ key: 'overview', label: t('imports.documentsTab'), count: docs.length }]} active={tab} onChange={setTab} />

      <div className="mt-5">
        <DataTable
          columns={docColumns}
          data={docs}
          loading={false}
          emptyTitle={t('documents.noDocuments')}
          rowCountLabel={`${t('common.total')}: ${toFaDigits(docs.length)}`}
        />
      </div>
    </>
  )
}

function progressFor(status: string): number {
  switch (status) {
    case 'COMPLETED':
      return 100
    case 'FAILED':
      return 100
    case 'REVIEW_REQUIRED':
      return 85
    case 'PROCESSING':
      return 45
    case 'QUEUED':
      return 15
    default:
      return 5
  }
}

function stageStatus(batch: { status: string; timeline?: ProcessingStage[] }, key: string): ProcessingStage['status'] {
  if (batch.timeline) {
    const found = batch.timeline.find((s) => s.key === key)
    if (found) return found.status
  }
  const order = ['extracting', 'analyzing', 'deduplicating', 'creating_leads']
  const idx = order.indexOf(key)
  const done = batch.status === 'COMPLETED'
  const failed = batch.status === 'FAILED'
  if (done) return 'done'
  if (failed && idx === 0) return 'error'
  if (batch.status === 'PROCESSING' && idx === 0) return 'active'
  return 'pending'
}
