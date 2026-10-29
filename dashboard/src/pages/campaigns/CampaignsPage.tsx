import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { ColumnDef } from '@tanstack/react-table'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits, formatDateTime } from '@/utils/format'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader, BackButton } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog, ConfirmDialog } from '@/components/ui/Dialog'
import { Input, Textarea } from '@/components/ui/Field'
import { useAuth, PERMISSIONS } from '@/auth/AuthContext'
import type { Campaign } from '@/api/types'

const campaignSchema = z.object({
  title: z.string().min(1),
  description: z.string().optional(),
  objective: z.string().optional(),
})

type CampaignFormValues = z.infer<typeof campaignSchema>

export function CampaignsPage() {
  const { t, locale } = useI18n()
  const { can } = useAuth()
  const toast = useToastApi()
  const qc = useQueryClient()
  const [page, setPage] = useState(1)
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Campaign | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Campaign | null>(null)

  const query = useMemo(() => ({ page, page_size: 20 }), [page])
  const { data, isLoading } = useQuery({ queryKey: queryKeys.campaigns(query), queryFn: () => api.listCampaigns(query) })

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: queryKeys.campaigns() })
  }

  const createCampaign = useMutation({
    mutationFn: (v: CampaignFormValues) =>
      api.createCampaign({ title: v.title, description: v.description || null, objective: v.objective || null }),
    onSuccess: () => {
      toast.success(t('campaigns.createSuccess'))
      setCreateOpen(false)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const updateCampaign = useMutation({
    mutationFn: (v: CampaignFormValues) =>
      api.updateCampaign(editTarget?.id ?? '', {
        title: v.title,
        description: v.description || null,
        objective: v.objective || null,
      }),
    onSuccess: () => {
      toast.success(t('campaigns.updated'))
      setEditTarget(null)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const deleteCampaign = useMutation({
    mutationFn: (id: string) => api.deleteCampaign(id),
    onSuccess: () => {
      toast.success(t('campaigns.deleteSuccess'))
      setDeleteTarget(null)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const columns = useMemo<ColumnDef<Campaign, unknown>[]>(
    () => [
      {
        accessorKey: 'title',
        header: t('campaigns.name'),
        cell: ({ row }) => (
          <Link to={`/campaigns/${row.original.id}`} className="strong" style={{ color: 'var(--text-1)' }}>
            {row.original.title}
          </Link>
        ),
      },
      { accessorKey: 'objective', header: t('campaigns.objective'), cell: ({ getValue }) => <span className="muted">{String(getValue()) || '—'}</span> },
      { accessorKey: 'description', header: t('campaigns.description'), cell: ({ getValue }) => <span className="muted">{String(getValue()) || '—'}</span> },
      {
        accessorKey: 'status',
        header: t('campaigns.status'),
        cell: ({ row }) => <Badge tone="accent">{t(`status.${row.original.status}`)}</Badge>,
        enableSorting: false,
      },
      {
        accessorKey: 'start_at',
        header: t('campaigns.startAt'),
        cell: ({ row }) => (row.original.start_at ? formatDateTime(row.original.start_at, locale, { time: false }) : <span className="muted">—</span>),
      },
      {
        id: 'actions',
        header: t('common.actions'),
        enableSorting: false,
        cell: ({ row }) => {
          const canManage = can(PERMISSIONS.projectsManage)
          return (
            <div className="row-actions">
              {canManage ? (
                <Button size="sm" variant="ghost" icon="edit" onClick={() => setEditTarget(row.original)} aria-label={t('common.edit')} />
              ) : null}
              {canManage ? (
                <Button size="sm" variant="ghost" icon="trash" onClick={() => setDeleteTarget(row.original)} aria-label={t('common.delete')} />
              ) : null}
            </div>
          )
        },
      },
    ],
    [t, can, locale],
  )

  return (
    <>
      <PageHeader
        title={t('campaigns.title')}
        subtitle={t('campaigns.subtitle')}
        actions={
          can(PERMISSIONS.projectsManage) ? (
            <Button icon="plus" onClick={() => setCreateOpen(true)}>
              {t('campaigns.newCampaign')}
            </Button>
          ) : null
        }
      />

      <DataTable
        columns={columns}
        data={data?.items ?? []}
        meta={data?.meta}
        onPage={setPage}
        loading={isLoading}
        emptyTitle={t('campaigns.noCampaigns')}
        emptyHint={t('campaigns.noCampaignsHint')}
        rowCountLabel={`${t('common.total')}: ${toFaDigits(data?.meta.total ?? 0)}`}
      />

      <CampaignFormDialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title={t('campaigns.newCampaign')}
        onSubmit={(v) => createCampaign.mutate(v)}
        loading={createCampaign.isPending}
      />

      <CampaignFormDialog
        open={!!editTarget}
        onClose={() => setEditTarget(null)}
        title={t('campaigns.editCampaign')}
        initial={editTarget}
        onSubmit={(v) => updateCampaign.mutate(v)}
        loading={updateCampaign.isPending}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteCampaign.mutate(deleteTarget.id)}
        title={t('confirm.title')}
        message={t('campaigns.confirmDelete')}
        loading={deleteCampaign.isPending}
      />
    </>
  )
}

export function CampaignDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t, locale } = useI18n()
  const { data: campaign, isLoading } = useQuery({
    queryKey: queryKeys.campaign(id ?? ''),
    queryFn: () => api.getCampaign(id ?? ''),
    enabled: !!id,
  })

  if (isLoading || !campaign) {
    return (
      <>
        <BackButton to="/campaigns" />
        <PageHeader title={t('campaigns.title')} subtitle={t('campaigns.subtitle')} breadcrumbs={[{ label: t('nav.projects'), to: '/campaigns' }]} />
        <div className="p-6">{t('common.loading')}</div>
      </>
    )
  }

  return (
    <>
      <BackButton to="/campaigns" />
      <PageHeader title={campaign.title} subtitle={campaign.description ?? undefined} breadcrumbs={[{ label: t('campaigns.title'), to: '/campaigns' }]} />

      <div className="card mb-4">
        <div className="card-body">
          <div className="grid-2">
            <div className="detail-row">
              <dt>{t('campaigns.objective')}</dt>
              <dd>{campaign.objective ?? '—'}</dd>
            </div>
            <div className="detail-row">
              <dt>{t('campaigns.status')}</dt>
              <dd><Badge tone="accent">{t(`status.${campaign.status}`)}</Badge></dd>
            </div>
            <div className="detail-row">
              <dt>{t('campaigns.startAt')}</dt>
              <dd>{campaign.start_at ? formatDateTime(campaign.start_at, locale, { time: false }) : '—'}</dd>
            </div>
            <div className="detail-row">
              <dt>{t('campaigns.endAt')}</dt>
              <dd>{campaign.end_at ? formatDateTime(campaign.end_at, locale, { time: false }) : '—'}</dd>
            </div>
          </div>
        </div>
      </div>

      <div className="card">
        <div className="card-header">
          <h3>{t('campaigns.projects')} ({toFaDigits((campaign.projects ?? []).length)})</h3>
        </div>
        <div className="table-scroll">
          <table className="table">
            <thead>
              <tr>
                <th>{t('projects.name')}</th>
                <th>{t('common.status')}</th>
              </tr>
            </thead>
            <tbody>
              {(campaign.projects ?? []).length === 0 ? (
                <tr>
                  <td colSpan={2} className="table-empty">
                    <p className="muted">{t('common.noData')}</p>
                  </td>
                </tr>
              ) : (
                (campaign.projects ?? []).map((p) => (
                  <tr key={p.id}>
                    <td>
                      <Link to={`/projects/${p.id}`} className="strong" style={{ color: 'var(--text-1)' }}>
                        {p.title}
                      </Link>
                    </td>
                    <td><Badge tone="accent">{t(`status.${p.status}`)}</Badge></td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </>
  )
}

function CampaignFormDialog({
  open,
  onClose,
  title,
  initial,
  onSubmit,
  loading,
}: {
  open: boolean
  onClose: () => void
  title: string
  initial?: Campaign | null
  onSubmit: (v: CampaignFormValues) => void
  loading: boolean
}) {
  const { t } = useI18n()
  const { register, handleSubmit, reset, formState: { errors } } = useForm<CampaignFormValues>({
    resolver: zodResolver(campaignSchema),
    defaultValues: { title: initial?.title ?? '', description: initial?.description ?? '', objective: initial?.objective ?? '' },
  })
  useEffect(() => {
    if (open) reset({ title: initial?.title ?? '', description: initial?.description ?? '', objective: initial?.objective ?? '' })
  }, [open, initial, reset])

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      footer={
        <>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button type="submit" form="campaign-form" variant="primary" loading={loading}>
            {t('common.save')}
          </Button>
        </>
      }
    >
      <form id="campaign-form" onSubmit={handleSubmit(onSubmit)}>
        {Object.keys(errors).length > 0 ? (
          <div className="login-error" role="alert">{t('errors.validation')}</div>
        ) : null}
        <div className="form-row">
          <Input label={`${t('campaigns.name')} *`} required error={errors.title?.message} {...register('title')} />
        </div>
        <div className="form-row">
          <Textarea label={t('campaigns.objective')} rows={2} {...register('objective')} />
        </div>
        <div className="form-row">
          <Textarea label={t('campaigns.description')} rows={3} {...register('description')} />
        </div>
      </form>
    </Dialog>
  )
}
