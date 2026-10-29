import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import type { ColumnDef } from '@tanstack/react-table'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { useAuth, PERMISSIONS } from '@/auth/AuthContext'
import { toFaDigits, formatDateTime } from '@/utils/format'
import { jalaliToIsoDate, tehranTimeOf, toJalaliDate, type JalaliDate } from '@/utils/jalali'
import { JalaliDatePicker } from '@/components/ui/JalaliDatePicker'
import { CreateFollowUpDialog } from '@/components/followups/CreateFollowUpDialog'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import { StatusBadge } from '@/components/ui/Badge'
import { Tabs } from '@/components/ui/misc'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import type { FollowUp } from '@/api/types'

const VIEWS = [
  { key: 'today', labelKey: 'followups.viewToday' },
  { key: 'overdue', labelKey: 'followups.viewOverdue' },
  { key: 'upcoming', labelKey: 'followups.viewUpcoming' },
  { key: 'completed', labelKey: 'followups.viewCompleted' },
] as const

export function FollowupsPage() {
  const { t, locale } = useI18n()
  const { can } = useAuth()
  const toast = useToastApi()
  const qc = useQueryClient()
  const [view, setView] = useState<(typeof VIEWS)[number]['key']>('today')
  const [reschedule, setReschedule] = useState<FollowUp | null>(null)
  const [rescheduleValue, setRescheduleValue] = useState<JalaliDate | null>(null)
  const [createOpen, setCreateOpen] = useState(false)

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: queryKeys.followups(view),
    queryFn: () => api.listFollowups(view),
  })

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['followups'] })
    void qc.invalidateQueries({ queryKey: queryKeys.leadFollowups('') })
  }

  const canCreate = can(PERMISSIONS.leadsContact)

  const complete = useMutation({
    mutationFn: (id: string) => api.completeFollowup(id),
    onSuccess: () => {
      toast.success(t('followups.complete'))
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const reopen = useMutation({
    mutationFn: (id: string) => api.reopenFollowup(id),
    onSuccess: () => {
      toast.success(t('followups.reopen'))
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const cancel = useMutation({
    mutationFn: (id: string) => api.cancelFollowup(id),
    onSuccess: () => {
      toast.success(t('followups.cancelled'))
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const rescheduleMut = useMutation({
    mutationFn: ({ id, fu, dueAt }: { id: string; fu: FollowUp; dueAt: JalaliDate }) => {
      const time = tehranTimeOf(fu.due_at)
      return api.rescheduleFollowup(id, jalaliToIsoDate(dueAt.year, dueAt.month, dueAt.day, time.hour, time.minute))
    },
    onSuccess: () => {
      toast.success(t('followups.rescheduled'))
      setReschedule(null)
      setRescheduleValue(null)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const columns = useMemo<ColumnDef<FollowUp, unknown>[]>(() => {
    return [
      {
        id: 'lead',
        header: t('followups.lead'),
        cell: ({ row }) => (
          <Link to={`/leads/${row.original.lead_id}`} className="strong" style={{ color: 'var(--text-1)' }}>
            {row.original.lead?.company_name ?? row.original.company_name ?? row.original.lead_id}
          </Link>
        ),
        enableSorting: false,
      },
      { id: 'agent', header: t('followups.agent'), cell: ({ row }) => row.original.assignee?.name ?? row.original.assignee_name ?? <span className="muted">—</span>, enableSorting: false },
      {
        accessorKey: 'due_at',
        header: t('followups.dueDate'),
        cell: ({ row }) => {
          const overdue = row.original.is_overdue
          return overdue ? <span className="fu-overdue">{formatDateTime(row.original.due_at, locale)}</span> : formatDateTime(row.original.due_at, locale)
        },
        enableSorting: true,
      },
      { accessorKey: 'type', header: t('followups.type'), cell: ({ getValue }) => <span className="chip">{t(`followups.types.${String(getValue())}`)}</span>, enableSorting: false },
      { accessorKey: 'description', header: t('followups.description'), cell: ({ getValue }) => (getValue() as string) || <span className="muted">—</span>, enableSorting: false },
      {
        accessorKey: 'status',
        header: t('followups.status'),
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
        enableSorting: false,
      },
      {
        id: 'actions',
        header: t('common.actions'),
        enableSorting: false,
        cell: ({ row }) => {
          const fu = row.original
          const pending = fu.status === 'PENDING'
          const completed = fu.status === 'COMPLETED'
          return (
            <span className="table-actions">
              {pending ? (
                <Button size="sm" icon="check" onClick={() => complete.mutate(fu.id)}>
                  {t('followups.complete')}
                </Button>
              ) : null}
              {pending ? (
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => {
                    setReschedule(fu)
                    setRescheduleValue(toJalaliDate(new Date(fu.due_at)))
                  }}
                >
                  {t('followups.reschedule')}
                </Button>
              ) : null}
              {pending ? (
                <Button size="sm" variant="ghost" icon="trash" onClick={() => cancel.mutate(fu.id)}>
                  {t('followups.cancel')}
                </Button>
              ) : null}
              {completed ? (
                <Button size="sm" variant="ghost" onClick={() => reopen.mutate(fu.id)}>
                  {t('followups.reopen')}
                </Button>
              ) : null}
            </span>
          )
        },
      },
    ]
  }, [t, locale, complete, reopen, cancel])

  return (
    <>
      <PageHeader
        title={t('followups.title')}
        subtitle={t('followups.subtitle')}
        actions={
          canCreate ? (
            <Button icon="plus" variant="primary" onClick={() => setCreateOpen(true)}>
              {t('followups.create')}
            </Button>
          ) : null
        }
      />
      <Tabs
        tabs={VIEWS.map((v) => ({ key: v.key, label: t(v.labelKey) }))}
        active={view}
        onChange={(k) => setView(k as (typeof VIEWS)[number]['key'])}
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
            data={data?.items ?? []}
            meta={data?.meta}
            loading={isLoading}
            emptyTitle={t('followups.noFollowups')}
            rowCountLabel={`${t('common.total')}: ${toFaDigits(data?.meta.total ?? 0)}`}
          />
        )}
      </div>

      <CreateFollowUpDialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        mode="global"
      />

      {reschedule ? (
        <Dialog open onClose={() => setReschedule(null)} title={t('followups.reschedule')}>
          <div className="field">
            <label className="field-label">{t('followups.dueDate')}</label>
            <JalaliDatePicker value={rescheduleValue} onChange={setRescheduleValue} />
          </div>
          <div className="mt-4 flex gap-2 justify-end">
            <Button variant="ghost" onClick={() => setReschedule(null)}>
              {t('common.cancel')}
            </Button>
            <Button
              loading={rescheduleMut.isPending}
              disabled={!rescheduleValue}
              onClick={() => {
                if (rescheduleValue) rescheduleMut.mutate({ id: reschedule.id, fu: reschedule, dueAt: rescheduleValue })
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
