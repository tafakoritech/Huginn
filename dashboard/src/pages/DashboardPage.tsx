import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits, formatTrendLabel } from '@/utils/format'
import { Icon, type IconName } from '@/components/Icon'
import { PageHeader } from '@/components/ui/misc'
import { CardSkeletonGrid, ErrorState, Skeleton } from '@/components/ui/States'
import { FunnelChart, BarChart, LineChart, QualifiedBarChart } from '@/components/charts'

function MetricCard({
  label,
  value,
  icon,
  foot,
  tone = 'accent',
}: {
  label: string
  value: number
  icon: IconName
  foot?: string
  tone?: 'accent' | 'success' | 'warning' | 'error' | 'info' | 'neutral'
}) {
  const colorMap: Record<string, string> = {
    accent: 'var(--accent)',
    success: 'var(--success)',
    warning: 'var(--warning)',
    error: 'var(--error)',
    info: 'var(--info)',
    neutral: 'var(--text-3)',
  }
  return (
    <div className="metric-card">
      <span className="metric-label">
        <Icon name={icon} style={{ color: colorMap[tone] }} />
        {label}
      </span>
      <span className="metric-value">{toFaDigits(value)}</span>
      {foot ? <span className="metric-foot">{foot}</span> : null}
    </div>
  )
}

export function DashboardPage() {
  const { t, locale } = useI18n()
  const { data: stats, isLoading, isError, refetch } = useQuery({ queryKey: queryKeys.dashboardStats, queryFn: () => api.dashboardStats() })
  const { data: funnel } = useQuery({ queryKey: queryKeys.dashboardFunnel, queryFn: () => api.dashboardFunnel() })
  const { data: qualBySource } = useQuery({ queryKey: queryKeys.dashboardQualificationBySource, queryFn: () => api.dashboardQualificationBySource() })
  const { data: agents } = useQuery({ queryKey: queryKeys.dashboardAgentPerformance, queryFn: () => api.dashboardAgentPerformance() })
  const { data: rejections } = useQuery({ queryKey: queryKeys.dashboardRejectionReasons, queryFn: () => api.dashboardRejectionReasons() })
  const { data: cities } = useQuery({ queryKey: queryKeys.dashboardLeadsByCity, queryFn: () => api.dashboardLeadsByCity() })
  const { data: trendRaw } = useQuery({ queryKey: queryKeys.dashboardLeadTrend('day'), queryFn: () => api.dashboardLeadTrend('day') })
  const trend = useMemo(() => trendRaw?.map((p) => ({ ...p, label: formatTrendLabel(p.label, locale) })), [trendRaw, locale])

  if (isError) {
    return <ErrorState detail={t('errors.generic')} onRetry={() => void refetch()} />
  }

  if (isLoading || !stats) {
    return (
      <>
        <PageHeader title={t('dashboard.title')} subtitle={t('dashboard.subtitle')} />
        <CardSkeletonGrid count={12} />
      </>
    )
  }

  return (
    <>
      <PageHeader title={t('dashboard.title')} subtitle={t('dashboard.subtitle')} />

      <div className="metric-grid mb-6">
        <MetricCard label={t('dashboard.totalLeads')} value={stats.total_leads} icon="leads" tone="accent" />
        <MetricCard label={t('dashboard.newLeads')} value={stats.new_leads} icon="plus" tone="info" />
        <MetricCard label={t('dashboard.unassigned')} value={stats.unassigned_leads} icon="users" tone="warning" />
        <MetricCard label={t('dashboard.reviewRequired')} value={stats.review_required} icon="alert" tone="warning" />
      </div>

      <div className="metric-grid mb-6">
        <MetricCard label={t('dashboard.callsToday')} value={stats.calls_today} icon="phone" />
        <MetricCard label={t('dashboard.successfulContacts')} value={stats.successful_contacts} icon="checkCircle" tone="success" />
        <MetricCard label={t('dashboard.unanswered')} value={stats.no_answers} icon="phone" tone="error" />
        <MetricCard label={t('dashboard.missingReports')} value={stats.pending_reports} icon="reports" tone="error" />
      </div>

      <div className="metric-grid mb-6">
        <MetricCard label={t('dashboard.followupsToday')} value={stats.follow_ups_today} icon="calendar" />
        <MetricCard label={t('dashboard.followupsOverdue')} value={stats.overdue_follow_ups} icon="clock" tone="error" />
        <MetricCard label={t('dashboard.qualified')} value={stats.qualified_leads} icon="checkCircle" tone="success" />
        <MetricCard label={t('dashboard.disqualified')} value={stats.disqualified_leads} icon="xCircle" tone="error" />
        <MetricCard label={t('dashboard.transferredMuninn')} value={stats.exported_to_muninn} icon="muninn" tone="accent" />
      </div>

      <div className="grid-2 mb-6">
        <section className="card chart-card">
          <h4>{t('dashboard.leadFunnel')}</h4>
          {funnel ? <FunnelChart steps={funnel.map((f) => f.count)} /> : <Skeleton className="skeleton-row" style={{ height: 180 }} />}
        </section>
        <section className="card chart-card">
          <h4>{t('dashboard.leadCreationTrend')}</h4>
          {trend ? <LineChart data={trend} /> : <Skeleton className="skeleton-row" style={{ height: 200 }} />}
        </section>
      </div>

      <div className="grid-2 mb-6">
        <section className="card chart-card">
          <h4>{t('dashboard.qualificationBySource')}</h4>
          {qualBySource ? (
            <QualifiedBarChart data={qualBySource.map((q) => ({ label: q.source, total: q.total, qualified: q.qualified }))} />
          ) : (
            <Skeleton className="skeleton-row" style={{ height: 200 }} />
          )}
        </section>
        <section className="card chart-card">
          <h4>{t('dashboard.rejectionReasons')}</h4>
          {rejections ? <BarChart data={rejections.map((r) => ({ label: r.reason, value: r.count }))} color="var(--error)" /> : <Skeleton className="skeleton-row" style={{ height: 200 }} />}
        </section>
      </div>

      <div className="grid-2">
        <section className="card chart-card">
          <h4>{t('dashboard.leadsByCity')}</h4>
          {cities ? <BarChart data={cities} color="var(--info)" /> : <Skeleton className="skeleton-row" style={{ height: 200 }} />}
        </section>
        <section className="card chart-card">
          <h4>{t('dashboard.agentPerformance')}</h4>
          {agents ? (
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>{t('agents.name')}</th>
                    <th>{t('agents.callsToday')}</th>
                    <th>{t('agents.successfulContacts')}</th>
                    <th>{t('agents.qualifiedLeads')}</th>
                    <th>{t('agents.qualificationRate')}</th>
                  </tr>
                </thead>
                <tbody>
                  {agents.map((a) => (
                    <tr key={a.user_id}>
                      <td className="strong">{a.name}</td>
                      <td>{toFaDigits(a.calls_today)}</td>
                      <td>{toFaDigits(a.successful_contacts)}</td>
                      <td>{toFaDigits(a.qualified_leads)}</td>
                      <td>
                        <span className="badge badge-accent">%{toFaDigits(a.qualification_rate)}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <Skeleton className="skeleton-row" style={{ height: 200 }} />
          )}
        </section>
      </div>
    </>
  )
}
