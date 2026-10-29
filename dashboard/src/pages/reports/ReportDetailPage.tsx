import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { useAuth, PERMISSIONS } from '@/auth/AuthContext'
import { formatDateTime, formatDuration } from '@/utils/format'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader, BackButton } from '@/components/ui/misc'
import { LoadingState, ErrorState } from '@/components/ui/States'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Icon } from '@/components/Icon'
import { Textarea } from '@/components/ui/Field'
import { DetailRows } from './reportRows'

export function ReportDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t, locale } = useI18n()
  const { can } = useAuth()
  const toast = useToastApi()
  const qc = useQueryClient()
  const [overrideOpen, setOverrideOpen] = useState(false)
  const [overrideReason, setOverrideReason] = useState('')

  const { data: report, isLoading, isError, refetch } = useQuery({
    queryKey: queryKeys.report(id ?? ''),
    queryFn: () => api.getReport(id ?? ''),
    enabled: !!id,
  })

  const override = useMutation({
    mutationFn: () => api.overrideReport(id ?? '', overrideReason),
    onSuccess: () => {
      toast.success(t('reports.managerOverride'))
      setOverrideOpen(false)
      setOverrideReason('')
      void qc.invalidateQueries({ queryKey: queryKeys.report(id ?? '') })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  if (isLoading || !report) return <LoadingState />
  if (isError) return <ErrorState detail={t('errors.generic')} onRetry={() => void refetch()} />

  const it = report.interaction

  return (
    <>
      <BackButton to="/reports" />
      <PageHeader
        title={t('reports.detailTitle')}
        subtitle={it?.lead?.company_name ?? ''}
        actions={
          can(PERMISSIONS.reportsOverride) && !report.manager_override ? (
            <Button icon="edit" onClick={() => setOverrideOpen(true)}>
              {t('reports.managerOverride')}
            </Button>
          ) : report.manager_override ? (
            <Badge tone="info">{t('reports.managerOverride')}</Badge>
          ) : null
        }
      />

      <div className="grid-2">
        <div className="card">
          <div className="card-header">
            <h3>{t('reports.agent')}</h3>
          </div>
          <div className="card-body">
            <DetailRows
              rows={[
                { label: t('reports.agent'), value: it?.user?.name ?? report.submitter?.name },
                { label: t('reports.lead'), value: it?.lead?.company_name },
                { label: t('reports.channel'), value: it?.channel ? t(`status.${it.channel}`) : null },
                { label: t('reports.interactionTime'), value: it ? formatDateTime(it.started_at, locale) : null },
                { label: t('reports.duration'), value: it?.duration_seconds ? formatDuration(it.duration_seconds) : null },
              ]}
            />
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3>{t('reports.outcome')}</h3>
          </div>
          <div className="card-body">
            <DetailRows
              rows={[
                { label: t('reports.outcome'), value: report.outcome },
                { label: t('reports.contactReached'), value: bool(report.contact_reached, t) },
                { label: t('reports.correctPerson'), value: bool(report.correct_person_reached, t) },
                { label: t('reports.decisionMakerIdentified'), value: bool(report.decision_maker_identified, t) },
                { label: t('leads.decisionMakerLabel'), value: report.decision_maker_name ? `${report.decision_maker_name} — ${report.decision_maker_role ?? ''}` : null },
              ]}
            />
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3>{t('leads.businessProfile')}</h3>
          </div>
          <div className="card-body">
            <DetailRows
              rows={[
                { label: t('leads.interestLevel'), value: report.interest_level },
                { label: t('leads.needLevel'), value: report.need_level },
                { label: t('leads.estimatedCapacity'), value: report.estimated_purchase_capacity },
                { label: t('leads.purchaseTiming'), value: report.purchase_timing },
                { label: t('leads.mainObjection'), value: report.main_objection },
                { label: t('reports.followUp'), value: report.follow_up_at ? formatDateTime(report.follow_up_at, locale, { time: false }) : null },
                { label: t('reports.qualRecommendation'), value: report.qualification_recommendation },
              ]}
            />
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3>{t('reports.summary')}</h3>
          </div>
          <div className="card-body">
            <p>{report.summary ?? <span className="muted">—</span>}</p>
            <div className="mt-4">
              <DetailRows
                rows={[
                  { label: t('reports.nextAction'), value: report.next_action },
                ]}
              />
            </div>
          </div>
        </div>
      </div>

      {report.manager_override ? (
        <div className="card mt-5" style={{ borderInlineStart: '4px solid var(--info)' }}>
          <div className="card-header">
            <h3>
              <Icon name="alert" size={16} />
              {t('reports.managerOverride')}
            </h3>
            <Badge tone="info">{t('reports.managerOverrideNote')}</Badge>
          </div>
          <div className="card-body">
            <DetailRows
              rows={[
                { label: t('reports.overriddenBy'), value: report.overrider?.name ?? report.overridden_by },
                { label: t('reports.time'), value: report.updated_at ? formatDateTime(report.updated_at, locale) : null },
                { label: t('reports.overrideReason'), value: report.manager_override_reason },
              ]}
            />
          </div>
        </div>
      ) : null}

      <Dialog
        open={overrideOpen}
        onClose={() => setOverrideOpen(false)}
        title={t('reports.managerOverride')}
        footer={
          <>
            <Button onClick={() => setOverrideOpen(false)}>{t('common.cancel')}</Button>
            <Button variant="primary" disabled={!overrideReason.trim()} onClick={() => override.mutate()} loading={override.isPending}>
              {t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="field">
          <label className="field-label">{t('reports.overrideReason')}</label>
          <Textarea value={overrideReason} onChange={(e) => setOverrideReason(e.target.value)} />
        </div>
      </Dialog>
    </>
  )
}

function bool(v: boolean | null | undefined, t: (k: string) => string): string | null {
  if (v === null || v === undefined) return null
  return v ? t('common.yes') : t('common.no')
}
