import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import type { ColumnDef } from '@tanstack/react-table'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { useAuth } from '@/auth/AuthContext'
import { toFaDigits, formatDateTime } from '@/utils/format'
import { jalaliToIsoDate, tehranTimeOf, toJalaliDate, type JalaliDate } from '@/utils/jalali'
import { JalaliDatePicker } from '@/components/ui/JalaliDatePicker'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader, Tabs } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import type { NextAction, NextActionPriority } from '@/api/types'

const FILTERS = [
  { key: 'all', labelKey: 'workQueue.filterAll' },
  { key: 'urgent', labelKey: 'workQueue.filterUrgent' },
  { key: 'today', labelKey: 'workQueue.filterToday' },
  { key: 'overdue', labelKey: 'workQueue.filterOverdue' },
  { key: 'reports', labelKey: 'workQueue.filterReports' },
  { key: 'call', labelKey: 'workQueue.filterCall' },
  { key: 'message', labelKey: 'workQueue.filterMessage' },
  { key: 'email', labelKey: 'workQueue.filterEmail' },
] as const

type FilterKey = (typeof FILTERS)[number]['key']

function filterItems(items: NextAction[], filter: FilterKey): NextAction[] {
  switch (filter) {
    case 'all':
      return items
    case 'urgent':
      return items.filter((i) => i.priority === 'urgent')
    case 'today':
      return items.filter((i) => i.due_at && new Date(i.due_at).toDateString() === new Date().toDateString())
    case 'overdue':
      return items.filter((i) => i.overdue)
    case 'reports':
      return items.filter((i) => i.source_type === 'pending_report')
    case 'call':
      return items.filter((i) => i.action_type === 'call')
    case 'message':
      return items.filter((i) => i.action_type === 'message')
    case 'email':
      return items.filter((i) => i.action_type === 'email')
  }
}

function priorityTone(priority: NextActionPriority): 'error' | 'warning' | 'neutral' {
  if (priority === 'urgent') return 'error'
  if (priority === 'important') return 'warning'
  return 'neutral'
}

function priorityLabel(priority: NextActionPriority, t: (k: string) => string): string {
  if (priority === 'urgent') return t('workQueue.urgent')
  if (priority === 'important') return t('workQueue.important')
  return t('workQueue.normal')
}

export function WorkQueuePage() {
  const { t, locale } = useI18n()
  const { isPrivileged } = useAuth()
  const toast = useToastApi()
  const qc = useQueryClient()
  const [view, setView] = useState<'my' | 'team'>('my')
  const [filter, setFilter] = useState<FilterKey>('all')
  const [snoozeTarget, setSnoozeTarget] = useState<NextAction | null>(null)
  const [snoozeValue, setSnoozeValue] = useState<JalaliDate | null>(null)

  const queryKey = view === 'team' ? queryKeys.teamNextActions(200) : queryKeys.nextActions(200)
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey,
    queryFn: () => (view === 'team' ? api.teamNextActions(200) : api.nextActions(200)),
  })

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['next-actions'] })
    void qc.invalidateQueries({ queryKey: ['team-next-actions'] })
  }

  const snooze = useMutation({
    mutationFn: ({ action, dueAt }: { action: NextAction; dueAt: JalaliDate }) => {
      const time = action.due_at ? tehranTimeOf(action.due_at) : { hour: 9, minute: 0 }
      return api.snoozeNextAction(action.id, jalaliToIsoDate(dueAt.year, dueAt.month, dueAt.day, time.hour, time.minute))
    },
    onSuccess: () => {
      toast.success(t('workQueue.snoozed'))
      setSnoozeTarget(null)
      setSnoozeValue(null)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const complete = useMutation({
    mutationFn: (id: string) => api.completeNextAction(id),
    onSuccess: () => {
      toast.success(t('workQueue.completed'))
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const items = useMemo(() => filterItems(data?.items ?? [], filter), [data, filter])

  const columns = useMemo<ColumnDef<NextAction, unknown>[]>(() => {
    const cols: ColumnDef<NextAction, unknown>[] = []
    if (view === 'team') {
      cols.push({
        id: 'agent',
        header: t('workQueue.agent'),
        cell: ({ row }) => row.original.assignee_name ?? <span className="muted">—</span>,
        enableSorting: false,
      })
    }
    cols.push(
      {
        id: 'lead',
        header: t('workQueue.lead'),
        cell: ({ row }) => (
          <Link to={`/leads/${row.original.lead_id}`} className="strong" style={{ color: 'var(--text-1)' }}>
            {row.original.company_name || row.original.lead_id}
          </Link>
        ),
        enableSorting: false,
      },
      {
        id: 'action',
        header: t('workQueue.action'),
        cell: ({ row }) => {
          const a = row.original
          const label = a.action_type === 'report' ? t('workQueue.actionTypes.report') : a.cta_label || t(`workQueue.actionTypes.${a.action_type}`)
          return <span className="chip">{label}</span>
        },
        enableSorting: false,
      },
      {
        id: 'priority',
        header: t('workQueue.priority'),
        cell: ({ row }) => <Badge tone={priorityTone(row.original.priority)}>{priorityLabel(row.original.priority, t)}</Badge>,
        enableSorting: false,
      },
      {
        id: 'due',
        header: t('workQueue.due'),
        cell: ({ row }) => {
          const d = row.original.due_at
          if (!d) return <span className="muted">—</span>
          const text = formatDateTime(d, locale, { relativeToday: true })
          return row.original.overdue ? <span className="fu-overdue">{text}</span> : text
        },
        enableSorting: true,
      },
      {
        id: 'reason',
        header: t('workQueue.reason'),
        cell: ({ row }) => {
          const reasons = row.original.explainability
          return reasons.length > 0 ? <span className="muted">{reasons.join(' • ')}</span> : <span className="muted">—</span>
        },
        enableSorting: false,
      },
      {
        id: 'actions',
        header: t('common.actions'),
        enableSorting: false,
        cell: ({ row }) => {
          const a = row.original
          return (
            <span className="table-actions">
              {a.source_type === 'follow_up' ? (
                <>
                  <Button size="sm" icon="check" onClick={() => complete.mutate(a.id)}>
                    {t('workQueue.complete')}
                  </Button>
                  <Button
                    size="sm"
                    variant="secondary"
                    onClick={() => {
                      setSnoozeTarget(a)
                      setSnoozeValue(a.due_at ? toJalaliDate(new Date(a.due_at)) : null)
                    }}
                  >
                    {t('workQueue.snooze')}
                  </Button>
                </>
              ) : null}
              <Link to={`/leads/${a.lead_id}`} className="btn btn-sm btn-ghost">
                {t('workQueue.openLead')}
              </Link>
            </span>
          )
        },
      },
    )
    return cols
  }, [t, locale, view, complete])

  return (
    <>
      <PageHeader
        title={view === 'team' ? t('workQueue.teamTitle') : t('workQueue.title')}
        subtitle={view === 'team' ? t('workQueue.teamSubtitle') : t('workQueue.subtitle')}
        actions={
          isPrivileged ? (
            <div className="flex gap-2">
              <Button variant={view === 'my' ? 'primary' : 'secondary'} onClick={() => setView('my')}>
                {t('workQueue.title')}
              </Button>
              <Button variant={view === 'team' ? 'primary' : 'secondary'} onClick={() => setView('team')}>
                {t('workQueue.teamTitle')}
              </Button>
            </div>
          ) : null
        }
      />
      <Tabs
        tabs={FILTERS.map((f) => ({ key: f.key, label: t(f.labelKey) }))}
        active={filter}
        onChange={(k) => setFilter(k as FilterKey)}
      />
      <div className="mt-5">
        {isError ? (
          <div className="alert error">
            <span>{t('common.loadFailed')}</span>
            <Button size="sm" variant="secondary" onClick={() => void refetch()}>
              {t('common.retry')}
            </Button>
          </div>
        ) : (
          <DataTable
            columns={columns}
            data={items}
            loading={isLoading}
            emptyTitle={t('workQueue.empty')}
            rowCountLabel={`${t('common.total')}: ${toFaDigits(items.length)}`}
          />
        )}
      </div>

      {snoozeTarget ? (
        <Dialog open onClose={() => setSnoozeTarget(null)} title={t('workQueue.snooze')}>
          <div className="field">
            <label className="field-label">{t('workQueue.due')}</label>
            <JalaliDatePicker value={snoozeValue} onChange={setSnoozeValue} />
          </div>
          <div className="mt-4 flex gap-2 justify-end">
            <Button variant="ghost" onClick={() => setSnoozeTarget(null)}>
              {t('common.cancel')}
            </Button>
            <Button
              loading={snooze.isPending}
              disabled={!snoozeValue}
              onClick={() => {
                if (snoozeValue) snooze.mutate({ action: snoozeTarget, dueAt: snoozeValue })
              }}
            >
              {t('common.save')}
            </Button>
          </div>
        </Dialog>
      ) : null}
    </>
  )
}
