import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits } from '@/utils/format'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader } from '@/components/ui/misc'
import { EmptyState } from '@/components/ui/States'
import { Button } from '@/components/ui/Button'
import { Dialog, ConfirmDialog } from '@/components/ui/Dialog'
import { Icon } from '@/components/Icon'
import { StatusBadge } from '@/components/ui/Badge'

export function AssignmentsPage() {
  const { t } = useI18n()
  const toast = useToastApi()
  const qc = useQueryClient()
  const [selected, setSelected] = useState<Record<string, boolean>>({})
  const [assignee, setAssignee] = useState('')
  const [bulkOpen, setBulkOpen] = useState(false)
  const [roundRobinOpen, setRoundRobinOpen] = useState(false)
  const [confirmBulk, setConfirmBulk] = useState(false)
  const [confirmRR, setConfirmRR] = useState(false)

  const { data } = useQuery({ queryKey: queryKeys.assignmentOverview, queryFn: () => api.assignmentOverview() })

  const invalidate = () => void qc.invalidateQueries({ queryKey: queryKeys.assignmentOverview })

  const bulkAssign = useMutation({
    mutationFn: () => api.assignBulk(leadIds(), assignee),
    onSuccess: () => {
      toast.success(t('assignments.assignSuccess'))
      setSelected({})
      setBulkOpen(false)
      setConfirmBulk(false)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const roundRobin = useMutation({
    mutationFn: () => api.assignRoundRobin(leadIds(), roundRobinAgentIds()),
    onSuccess: () => {
      toast.success(t('assignments.assignSuccess'))
      setSelected({})
      setRoundRobinOpen(false)
      setConfirmRR(false)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const leadIds = () => Object.keys(selected).filter((k) => selected[k])
  const roundRobinAgentIds = () => data?.employees.map((e) => e.user_id) ?? []

  const unassignedList = data?.unassigned_list.items ?? []
  const selectedCount = leadIds().length

  return (
    <>
      <PageHeader title={t('assignments.title')} subtitle={t('assignments.subtitle')} />

      <div className="metric-grid mb-6">
        <div className="metric-card">
          <span className="metric-label">
            <Icon name="users" />
            {t('assignments.unassignedLeads')}
          </span>
          <span className="metric-value">{toFaDigits(data?.unassigned_leads ?? 0)}</span>
        </div>
      </div>

      <div className="grid-2 mb-6">
        <section className="card">
          <div className="card-header">
            <h3>{t('assignments.employees')}</h3>
          </div>
          <div className="table-scroll">
            <table className="table">
              <thead>
                <tr>
                  <th>{t('agents.name')}</th>
                  <th>{t('assignments.currentWorkload')}</th>
                  <th>{t('assignments.todayWorkload')}</th>
                  <th>{t('assignments.qualificationPerformance')}</th>
                </tr>
              </thead>
              <tbody>
                {data?.employees.map((e) => (
                  <tr key={e.user_id}>
                    <td className="strong">{e.name}</td>
                    <td>{toFaDigits(e.current_load)}</td>
                    <td>{toFaDigits(e.today_load)}</td>
                    <td>
                      <span className="badge badge-accent">%{toFaDigits(e.qualification_performance)}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <section className="card">
          <div className="card-header">
            <h3>{t('assignments.unassignedLeads')}</h3>
            {selectedCount > 0 ? (
              <div className="flex gap-2">
                <Button size="sm" icon="users" onClick={() => setBulkOpen(true)}>
                  {t('assignments.assignSelected')}
                </Button>
                <Button size="sm" icon="refresh" variant="ghost" onClick={() => setRoundRobinOpen(true)}>
                  {t('assignments.roundRobin')}
                </Button>
              </div>
            ) : null}
          </div>
          {unassignedList.length === 0 ? (
            <EmptyState title={t('assignments.noUnassigned')} icon="checkCircle" />
          ) : (
            <div className="table-scroll" style={{ maxHeight: 420, overflowY: 'auto' }}>
              <table className="table">
                <thead>
                  <tr>
                    <th>
                      <input
                        type="checkbox"
                        aria-label={t('leads.bulkSelect')}
                        checked={unassignedList.length > 0 && Object.keys(selected).length === unassignedList.length}
                        onChange={(e) => {
                          if (e.target.checked) {
                            const next: Record<string, boolean> = {}
                            for (const l of unassignedList) next[l.id] = true
                            setSelected(next)
                          } else setSelected({})
                        }}
                      />
                    </th>
                    <th>{t('leads.company')}</th>
                    <th>{t('leads.city')}</th>
                    <th>{t('leads.score')}</th>
                    <th>{t('leads.status')}</th>
                  </tr>
                </thead>
                <tbody>
                  {unassignedList.map((l) => (
                    <tr key={l.id}>
                      <td>
                        <input type="checkbox" checked={!!selected[l.id]} onChange={(e) => setSelected((prev) => ({ ...prev, [l.id]: e.target.checked }))} aria-label={t('common.selected')} />
                      </td>
                      <td>
                        <Link to={`/leads/${l.id}`} className="strong" style={{ color: 'var(--text-1)' }}>
                          {l.company_name}
                        </Link>
                      </td>
                      <td>{l.city ?? '—'}</td>
                      <td>{toFaDigits(l.score)}</td>
                      <td>
                        <StatusBadge status={l.status} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>

      <Dialog
        open={bulkOpen}
        onClose={() => setBulkOpen(false)}
        title={t('assignments.bulkAssign')}
        footer={
          <>
            <Button onClick={() => setBulkOpen(false)}>{t('common.cancel')}</Button>
            <Button variant="primary" disabled={!assignee} onClick={() => setConfirmBulk(true)}>
              {t('assignments.assignSelected')} ({toFaDigits(selectedCount)})
            </Button>
          </>
        }
      >
        <div className="field">
          <label className="field-label">{t('leads.owner')}</label>
          <select className="select" value={assignee} onChange={(e) => setAssignee(e.target.value)}>
            <option value="">{t('common.choose')}</option>
            {data?.employees.map((e) => (
              <option key={e.user_id} value={e.user_id}>
                {e.name}
              </option>
            ))}
          </select>
        </div>
      </Dialog>

      <Dialog
        open={roundRobinOpen}
        onClose={() => setRoundRobinOpen(false)}
        title={t('assignments.roundRobin')}
        footer={
          <>
            <Button onClick={() => setRoundRobinOpen(false)}>{t('common.cancel')}</Button>
            <Button variant="primary" onClick={() => setConfirmRR(true)}>
              {t('common.confirm')}
            </Button>
          </>
        }
      >
        <p className="small secondary">
          {t('assignments.roundRobin')} — {toFaDigits(selectedCount)} {t('leads.title')} / {toFaDigits(roundRobinAgentIds().length)} {t('agents.name')}
        </p>
      </Dialog>

      <ConfirmDialog
        open={confirmBulk}
        onClose={() => setConfirmBulk(false)}
        onConfirm={() => bulkAssign.mutate()}
        title={t('assignments.bulkAssign')}
        message={t('assignments.confirmBulk', { count: toFaDigits(selectedCount) })}
        loading={bulkAssign.isPending}
      />

      <ConfirmDialog
        open={confirmRR}
        onClose={() => setConfirmRR(false)}
        onConfirm={() => roundRobin.mutate()}
        title={t('assignments.roundRobin')}
        message={t('assignments.confirmBulk', { count: toFaDigits(selectedCount) })}
        loading={roundRobin.isPending}
      />
    </>
  )
}
