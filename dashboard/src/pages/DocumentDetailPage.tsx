import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { formatDateTime, formatBytes } from '@/utils/format'
import { PageHeader, BackButton, Tabs } from '@/components/ui/misc'
import { LoadingState, ErrorState } from '@/components/ui/States'
import { Icon } from '@/components/Icon'
import { ScoreBadge } from '@/components/ui/Badge'
import { useState } from 'react'

export function DocumentDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t, locale } = useI18n()
  const [tab, setTab] = useState('preview')

  const { data: doc, isLoading, isError, refetch } = useQuery({
    queryKey: queryKeys.document(id ?? ''),
    queryFn: () => api.getDocument(id ?? ''),
    enabled: !!id,
  })

  const { data: candidates = [] } = useQuery({
    queryKey: ['document', id ?? '', 'candidates'],
    queryFn: () => api.documentCandidates(id ?? ''),
    enabled: !!id,
  })

  if (isLoading || !doc) return <LoadingState />
  if (isError) return <ErrorState detail={t('errors.generic')} onRetry={() => void refetch()} />

  return (
    <>
      <BackButton to="/documents" />
      <PageHeader title={doc.original_filename ?? t('documents.detailTitle')} subtitle={t('documents.detailTitle')} />

      <div className="card mb-5" style={{ padding: 16 }}>
        <div className="meta-grid">
          <div className="meta-item">
            <div className="meta-label">{t('documents.batch')}</div>
            <div className="meta-value" style={{ fontSize: 12 }}>
              {doc.batch_name ?? doc.batch_id}
            </div>
          </div>
          <div className="meta-item">
            <div className="meta-label">{t('imports.sourceType')}</div>
            <div className="meta-value">{doc.mime_type ?? '—'}</div>
          </div>
          <div className="meta-item">
            <div className="meta-label">{t('documents.size')}</div>
            <div className="meta-value">{formatBytes(doc.size_bytes)}</div>
          </div>
          <div className="meta-item">
            <div className="meta-label">{t('common.createdAt')}</div>
            <div className="meta-value" style={{ fontSize: 12 }}>
              {formatDateTime(doc.created_at, locale)}
            </div>
          </div>
          <div className="meta-item">
            <div className="meta-label">{t('common.status')}</div>
            <div className="meta-value" style={{ fontSize: 12 }}>
              {doc.status}
            </div>
          </div>
        </div>
        <p className="small muted mt-4">
          <Icon name="info" size={13} />
          {t('documents.notEditable')}
        </p>
      </div>

      <Tabs
        tabs={[
          { key: 'preview', label: t('documents.preview') },
          { key: 'text', label: t('documents.extractedText') },
          { key: 'candidates', label: t('documents.candidates'), count: candidates.length },
        ]}
        active={tab}
        onChange={setTab}
      />

      <div className="mt-5">
        {tab === 'preview' ? (
          <div className="card" style={{ padding: 20 }}>
            {doc.extracted_text ? (
              <pre className="evidence-excerpt" dir="auto" style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit', fontStyle: 'normal', maxHeight: 480, overflow: 'auto' }}>
                {doc.extracted_text.slice(0, 4000)}
              </pre>
            ) : (
              <p className="muted">{t('common.noData')}</p>
            )}
          </div>
        ) : null}

        {tab === 'text' ? (
          <div className="card" style={{ padding: 20 }}>
            {doc.extracted_text ? (
              <pre className="evidence-excerpt" dir="auto" style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit', fontStyle: 'normal', maxHeight: 560, overflow: 'auto' }}>
                {doc.extracted_text}
              </pre>
            ) : doc.extraction_error ? (
              <p className="badge badge-error" role="alert">
                {doc.extraction_error}
              </p>
            ) : (
              <p className="muted">{t('common.noData')}</p>
            )}
          </div>
        ) : null}

        {tab === 'candidates' ? (
          <div className="card">
            <div className="table-scroll">
              <table className="table">
                <thead>
                  <tr>
                    <th>{t('leads.company')}</th>
                    <th>{t('leads.primaryContact')}</th>
                    <th>{t('leads.phone')}</th>
                    <th>{t('leads.score')}</th>
                    <th>{t('common.status')}</th>
                  </tr>
                </thead>
                <tbody>
                  {candidates.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="table-empty">
                        <p className="muted">{t('common.noData')}</p>
                      </td>
                    </tr>
                  ) : (
                    candidates.map((c) => (
                      <tr key={c.id}>
                        <td className="strong">{c.company_name}</td>
                        <td>{c.contact_name ?? '—'}</td>
                        <td>
                          {c.phone ? <span className="mono" dir="ltr">{c.phone}</span> : '—'}
                        </td>
                        <td>
                          <ScoreBadge score={c.score} />
                        </td>
                        <td>
                          {c.status === 'created' ? (
                            <span className="badge badge-success">{t('common.yes')}</span>
                          ) : c.status === 'duplicate' ? (
                            <span className="badge badge-warning">{t('status.MERGED')}</span>
                          ) : c.error ? (
                            <span className="badge badge-error" title={c.error}>
                              {t('status.FAILED')}
                            </span>
                          ) : (
                            <span className="badge badge-info">{t('status.NEW')}</span>
                          )}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        ) : null}
      </div>
    </>
  )
}
