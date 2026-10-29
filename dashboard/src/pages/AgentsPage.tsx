import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits } from '@/utils/format'
import { PageHeader } from '@/components/ui/misc'
import { EmptyState, LoadingState } from '@/components/ui/States'
import { Icon } from '@/components/Icon'
import { Avatar } from '@/components/ui/misc'

export function AgentsPage() {
  const { t } = useI18n()
  const { data, isLoading } = useQuery({ queryKey: queryKeys.agents, queryFn: () => api.listAgents() })

  if (isLoading) return <LoadingState />

  return (
    <>
      <PageHeader title={t('agents.title')} subtitle={t('agents.subtitle')} />
      {!data || data.length === 0 ? (
        <div className="card">
          <EmptyState title={t('agents.noAgents')} icon="users" />
        </div>
      ) : (
        <div className="grid-3">
          {data.map((a) => (
            <Link to={`/agents/${a.user_id}`} className="card agent-card" key={a.user_id} style={{ textDecoration: 'none', color: 'inherit' }}>
              <div className="agent-card-head">
                <Avatar name={a.name} size={40} />
                <div className="flex flex-col">
                  <span className="agent-name">{a.name}</span>
                  <span className="small muted">{t('agents.name')}</span>
                </div>
                <span className="badge badge-success ms-auto" style={{ marginInlineStart: 'auto' }}>
                  {t('common.active')}
                </span>
              </div>
              <div className="agent-stats">
                <div className="agent-stat">
                  <div className="as-label">{t('agents.assignedLeads')}</div>
                  <div className="as-value">{toFaDigits(a.assigned_leads)}</div>
                </div>
                <div className="agent-stat">
                  <div className="as-label">{t('agents.callsToday')}</div>
                  <div className="as-value">{toFaDigits(a.calls_today)}</div>
                </div>
                <div className="agent-stat">
                  <div className="as-label">{t('agents.successfulContacts')}</div>
                  <div className="as-value">{toFaDigits(a.successful_contacts)}</div>
                </div>
                <div className="agent-stat">
                  <div className="as-label">{t('agents.pendingReports')}</div>
                  <div className="as-value" style={{ color: a.pending_reports > 0 ? 'var(--warning)' : undefined }}>
                    {toFaDigits(a.pending_reports)}
                  </div>
                </div>
                <div className="agent-stat">
                  <div className="as-label">{t('agents.overdueFollowups')}</div>
                  <div className="as-value" style={{ color: a.overdue_followups > 0 ? 'var(--error)' : undefined }}>
                    {toFaDigits(a.overdue_followups)}
                  </div>
                </div>
                <div className="agent-stat">
                  <div className="as-label">{t('agents.qualifiedLeads')}</div>
                  <div className="as-value">{toFaDigits(a.qualified_leads)}</div>
                </div>
              </div>
              <div className="flex items-center justify-between">
                <span className="small secondary">
                  {t('agents.qualificationRate')}
                </span>
                <span className="badge badge-accent">%{toFaDigits(a.qualification_rate)}</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </>
  )
}

export function AgentDetailPage() {
  const { t } = useI18n()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading } = useQuery({ queryKey: queryKeys.agent(id ?? ''), queryFn: () => api.getAgent(id ?? ''), enabled: !!id })

  if (isLoading || !data) return <LoadingState />

  return (
    <>
      <PageHeader
        title={data.user.name}
        subtitle={t('agents.title')}
        breadcrumbs={[{ label: t('nav.agents'), to: '/agents' }]}
      />
      <div className="metric-grid mb-6">
        <div className="metric-card">
          <span className="metric-label"><Icon name="leads" />{t('agents.assignedLeads')}</span>
          <span className="metric-value">{toFaDigits(data.assigned_leads)}</span>
        </div>
        <div className="metric-card">
          <span className="metric-label"><Icon name="phone" />{t('agents.callsToday')}</span>
          <span className="metric-value">{toFaDigits(data.calls_today)}</span>
        </div>
        <div className="metric-card">
          <span className="metric-label"><Icon name="checkCircle" />{t('agents.successfulContacts')}</span>
          <span className="metric-value">{toFaDigits(data.successful_contacts)}</span>
        </div>
        <div className="metric-card">
          <span className="metric-label"><Icon name="reports" />{t('agents.pendingReports')}</span>
          <span className="metric-value">{toFaDigits(data.pending_reports)}</span>
        </div>
        <div className="metric-card">
          <span className="metric-label"><Icon name="clock" />{t('agents.overdueFollowups')}</span>
          <span className="metric-value">{toFaDigits(data.overdue_followups)}</span>
        </div>
        <div className="metric-card">
          <span className="metric-label"><Icon name="checkCircle" />{t('agents.qualifiedLeads')}</span>
          <span className="metric-value">{toFaDigits(data.qualified_leads)}</span>
        </div>
      </div>

      <div className="grid-2">
        <section className="card">
          <div className="card-header">
            <h3>{t('agents.currentQueue')}</h3>
          </div>
          <div className="table-scroll">
            <table className="table">
              <thead>
                <tr>
                  <th>{t('leads.company')}</th>
                  <th>{t('leads.city')}</th>
                  <th>{t('leads.score')}</th>
                  <th>{t('leads.status')}</th>
                </tr>
              </thead>
              <tbody>
                {data.queue.slice(0, 12).map((l) => (
                  <tr key={l.id}>
                    <td>
                      <Link to={`/leads/${l.id}`} className="strong" style={{ color: 'var(--text-1)' }}>
                        {l.company_name}
                      </Link>
                    </td>
                    <td>{l.city ?? '—'}</td>
                    <td>{toFaDigits(l.score)}</td>
                    <td>{l.status}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
        <section className="card">
          <div className="card-header">
            <h3>{t('agents.recentActivity')}</h3>
          </div>
          <div className="card-body">
            {data.recent_activity.map((e) => (
              <div key={e.id} className="detail-row">
                <dt>{t(`leads.timeline.${e.event_type}`)}</dt>
                <dd>{e.description}</dd>
              </div>
            ))}
          </div>
        </section>
      </div>
    </>
  )
}
