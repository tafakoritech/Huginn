import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits, formatDateTime } from '@/utils/format'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader } from '@/components/ui/misc'
import { LoadingState, EmptyState } from '@/components/ui/States'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog, Dialog } from '@/components/ui/Dialog'
import { Icon } from '@/components/Icon'
import { ScoreBadge } from '@/components/ui/Badge'
import type { DuplicatePair, Lead } from '@/api/types'

const FIELDS: Array<{ key: keyof Lead; labelKey: string }> = [
  { key: 'company_name', labelKey: 'leads.company' },
  { key: 'business_type', labelKey: 'leads.businessType' },
  { key: 'industry', labelKey: 'leads.industry' },
  { key: 'city', labelKey: 'leads.city' },
  { key: 'province', labelKey: 'leads.province' },
  { key: 'status', labelKey: 'leads.status' },
  { key: 'score', labelKey: 'leads.score' },
  { key: 'website', labelKey: 'leads.website' },
  { key: 'created_at', labelKey: 'common.createdAt' },
]

export function DuplicatesPage() {
  const { t } = useI18n()
  const toast = useToastApi()
  const qc = useQueryClient()
  const [page] = useState(1)
  const [mergeTarget, setMergeTarget] = useState<DuplicatePair | null>(null)
  const [confirmMerge, setConfirmMerge] = useState(false)
  const [survivor, setSurvivor] = useState<'a' | 'b'>('a')

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.duplicates({ page, page_size: 10, status: 'POSSIBLE' }),
    queryFn: () => api.listDuplicates({ page, page_size: 10, status: 'POSSIBLE' }),
  })

  const invalidate = () => void qc.invalidateQueries({ queryKey: queryKeys.duplicates({}) })

  const markAction = useMutation({
    mutationFn: (vars: { id: string; action: 'merge' | 'distinct' | 'later' }) => api.markDuplicate(vars.id, vars.action),
    onSuccess: () => {
      toast.success(t('duplicates.merged'))
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const doMerge = useMutation({
    mutationFn: () => {
      const pair = mergeTarget!
      const survivingLeadId = survivor === 'a' ? pair.lead_a_id : pair.lead_b_id
      return api.mergeDuplicates(pair.id, survivingLeadId, { fields: FIELDS.map((f) => f.key) })
    },
    onSuccess: () => {
      toast.success(t('duplicates.merged'))
      setMergeTarget(null)
      setConfirmMerge(false)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  if (isLoading) return <LoadingState />

  const items = data?.items ?? []

  return (
    <>
      <PageHeader title={t('duplicates.title')} subtitle={t('duplicates.subtitle')} />

      {items.length === 0 ? (
        <div className="card">
          <EmptyState title={t('duplicates.noDuplicates')} icon="checkCircle" />
        </div>
      ) : (
        <div className="flex flex-col gap-5">
          {items.map((pair) => {
            const a = pair.lead_a
            const b = pair.lead_b
            return (
              <div className="card" key={pair.id}>
                <div className="card-header">
                  <div className="flex items-center gap-3">
                    <span className="badge badge-warning">
                      <Icon name="duplicates" size={12} />
                      {t('duplicates.similarity')}: %{toFaDigits(Math.round(pair.similarity * 100))}
                    </span>
                    <span className="small muted">{pair.reason}</span>
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" icon="check" variant="primary" onClick={() => { setMergeTarget(pair); setSurvivor('a') }}>
                      {t('duplicates.merge')}
                    </Button>
                    <Button size="sm" icon="x" variant="ghost" onClick={() => markAction.mutate({ id: pair.id, action: 'distinct' })}>
                      {t('duplicates.distinct')}
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => markAction.mutate({ id: pair.id, action: 'later' })}>
                      {t('duplicates.later')}
                    </Button>
                  </div>
                </div>

                <div className="card-body">
                  <div className="compare-grid">
                    <CompareColumn title={`${t('duplicates.leadA')} — ${a?.company_name ?? '—'}`} lead={a} />
                    <CompareColumn title={`${t('duplicates.leadB')} — ${b?.company_name ?? '—'}`} lead={b} />
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      <Dialog
        open={!!mergeTarget}
        onClose={() => setMergeTarget(null)}
        title={t('duplicates.mergeTitle')}
        size="lg"
        footer={
          <>
            <Button onClick={() => setMergeTarget(null)}>{t('common.cancel')}</Button>
            <Button variant="primary" onClick={() => setConfirmMerge(true)}>
              {t('duplicates.confirmMerge')}
            </Button>
          </>
        }
      >
        {mergeTarget && (
          <div className="flex flex-col gap-4">
            <p className="small secondary">{t('duplicates.timelinePreserved')}</p>
            <div>
              <div className="menu-label" style={{ padding: 0, marginBottom: 6 }}>
                {t('duplicates.survivingLead')}
              </div>
              <div className="flex gap-2">
                <Button size="sm" variant={survivor === 'a' ? 'primary' : 'secondary'} onClick={() => setSurvivor('a')}>
                  {t('duplicates.leadA')} — {mergeTarget.lead_a?.company_name}
                </Button>
                <Button size="sm" variant={survivor === 'b' ? 'primary' : 'secondary'} onClick={() => setSurvivor('b')}>
                  {t('duplicates.leadB')} — {mergeTarget.lead_b?.company_name}
                </Button>
              </div>
            </div>
            <div>
              <div className="menu-label" style={{ padding: 0, marginBottom: 6 }}>
                {t('duplicates.fieldsFromA')} / {t('duplicates.fieldsFromB')}
              </div>
              <div className="table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th>{t('leads.evidenceField')}</th>
                      <th>A</th>
                      <th>B</th>
                    </tr>
                  </thead>
                  <tbody>
                    {FIELDS.map((f) => (
                      <tr key={f.key}>
                        <td>{t(f.labelKey)}</td>
                        <td>{displayValue(mergeTarget.lead_a?.[f.key])}</td>
                        <td>{displayValue(mergeTarget.lead_b?.[f.key])}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}
      </Dialog>

      <ConfirmDialog
        open={confirmMerge}
        onClose={() => setConfirmMerge(false)}
        onConfirm={() => doMerge.mutate()}
        title={t('duplicates.confirmMerge')}
        message={t('duplicates.confirmMergeText')}
        loading={doMerge.isPending}
      />
    </>
  )
}

function displayValue(v: unknown): string {
  if (v === null || v === undefined || v === '') return '—'
  if (typeof v === 'object') return '…'
  return String(v)
}

function CompareColumn({ title, lead }: { title: string; lead?: Lead | null }) {
  const { t, locale } = useI18n()
  if (!lead) return <div className="compare-col"><div className="compare-col-head">{title}</div><div className="card-body muted">{t('common.noData')}</div></div>
  return (
    <div className="compare-col">
      <div className="compare-col-head">
        <span>{title}</span>
        <ScoreBadge score={lead.score} />
      </div>
      {FIELDS.map((f) => {
        const value = lead[f.key]
        const isMatch = value != null && value !== ''
        return (
          <div key={f.key} className={`compare-field ${isMatch ? 'match' : ''}`}>
            <span className="cmp-label">{t(f.labelKey)}</span>
            <span className="cmp-value">
              {f.key === 'created_at' && typeof value === 'string' ? formatDateTime(value, locale, { time: false }) : f.key === 'status' && typeof value === 'string' ? t(`status.${value}`) : String(value ?? '—')}
            </span>
          </div>
        )
      })}
    </div>
  )
}
