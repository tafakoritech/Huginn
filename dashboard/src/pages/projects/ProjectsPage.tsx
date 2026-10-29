import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { ColumnDef } from '@tanstack/react-table'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits } from '@/utils/format'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog, ConfirmDialog } from '@/components/ui/Dialog'
import { Input, Textarea } from '@/components/ui/Field'
import { Icon } from '@/components/Icon'
import { useAuth, PERMISSIONS } from '@/auth/AuthContext'
import type { Project } from '@/api/types'

const projectSchema = z.object({
  title: z.string().min(1),
  description: z.string().min(1),
  product_name: z.string().optional(),
  product_description: z.string().optional(),
  target_market: z.string().optional(),
  ideal_customer_profile: z.string().optional(),
  customer_pain_points: z.string().optional(),
  value_proposition: z.string().optional(),
  sales_objective: z.string().optional(),
  common_objections: z.string().optional(),
  campaign_id: z.string().optional(),
})

type ProjectFormValues = z.infer<typeof projectSchema>

export function ProjectsPage() {
  const { t } = useI18n()
  const { can } = useAuth()
  const toast = useToastApi()
  const qc = useQueryClient()
  const [page, setPage] = useState(1)
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Project | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Project | null>(null)

  const query = useMemo(() => ({ page, page_size: 20 }), [page])
  const { data, isLoading } = useQuery({ queryKey: queryKeys.projects(query), queryFn: () => api.listProjects(query) })
  const { data: campaigns } = useQuery({ queryKey: queryKeys.campaigns({ page_size: 100 }), queryFn: () => api.listCampaigns({ page_size: 100 }), enabled: can(PERMISSIONS.projectsManage) })

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: queryKeys.projects() })
  }

  const createProject = useMutation({
    mutationFn: (v: ProjectFormValues) =>
      api.createProject({ title: v.title, description: v.description, product_name: v.product_name || undefined, product_description: v.product_description || undefined, target_market: v.target_market || undefined, ideal_customer_profile: v.ideal_customer_profile || undefined, customer_pain_points: v.customer_pain_points || undefined, value_proposition: v.value_proposition || undefined, sales_objective: v.sales_objective || undefined, common_objections: v.common_objections || undefined, campaign_id: v.campaign_id || undefined }),
    onSuccess: () => {
      toast.success(t('projects.createSuccess'))
      setCreateOpen(false)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const updateProject = useMutation({
    mutationFn: (v: ProjectFormValues) =>
      api.updateProject(editTarget?.id ?? '', {
        title: v.title,
        description: v.description,
        product_name: v.product_name || undefined,
        product_description: v.product_description || undefined,
        target_market: v.target_market || undefined,
        ideal_customer_profile: v.ideal_customer_profile || undefined,
        customer_pain_points: v.customer_pain_points || undefined,
        value_proposition: v.value_proposition || undefined,
        sales_objective: v.sales_objective || undefined,
        common_objections: v.common_objections || undefined,
        campaign_id: v.campaign_id || null,
      }),
    onSuccess: () => {
      toast.success(t('projects.updated'))
      setEditTarget(null)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const deleteProject = useMutation({
    mutationFn: (id: string) => api.deleteProject(id),
    onSuccess: () => {
      toast.success(t('projects.deleteSuccess'))
      setDeleteTarget(null)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const columns = useMemo<ColumnDef<Project & { lead_count: number }, unknown>[]>(
    () => [
      {
        accessorKey: 'title',
        header: t('projects.name'),
        cell: ({ row }) => (
          <Link to={`/projects/${row.original.id}`} className="strong" style={{ color: 'var(--text-1)' }}>
            {row.original.title}
          </Link>
        ),
      },
      {
        accessorKey: 'campaign',
        header: t('projects.campaign'),
        cell: ({ row }) =>
          row.original.campaign ? (
            <Link to={`/campaigns/${row.original.campaign.id}`} className="strong">
              {row.original.campaign.title}
            </Link>
          ) : (
            <span className="muted">—</span>
          ),
      },
      {
        accessorKey: 'description',
        header: t('projects.description'),
        cell: ({ getValue }) => <span className="muted">{String(getValue()) || '—'}</span>,
      },
      {
        accessorKey: 'lead_count',
        header: t('projects.leadCount'),
        cell: ({ row }) => toFaDigits(row.original.lead_count),
        enableSorting: false,
      },
      {
        accessorKey: 'status',
        header: t('common.status'),
        cell: ({ row }) => <Badge tone="accent">{t(`status.${row.original.status}`)}</Badge>,
        enableSorting: false,
      },
      {
        id: 'actions',
        header: t('common.actions'),
        enableSorting: false,
        cell: ({ row }) => {
          const p = row.original
          const canManage = can(PERMISSIONS.projectsManage)
          return (
            <div className="row-actions">
              {canManage ? (
                <Button size="sm" variant="ghost" icon="edit" onClick={() => setEditTarget(p)} aria-label={t('common.edit')} />
              ) : null}
              <Link to={`/projects/${p.id}`} className="btn btn-ghost btn-sm" aria-label={t('common.view')}>
                <Icon name="eye" size={16} />
              </Link>
              {canManage ? (
                <Button size="sm" variant="ghost" icon="trash" onClick={() => setDeleteTarget(p)} aria-label={t('common.delete')} />
              ) : null}
            </div>
          )
        },
      },
    ],
    [t, can],
  )

  return (
    <>
      <PageHeader
        title={t('projects.title')}
        subtitle={t('projects.subtitle')}
        actions={
          can(PERMISSIONS.projectsManage) ? (
            <Button icon="plus" onClick={() => setCreateOpen(true)}>
              {t('projects.newProject')}
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
        emptyTitle={t('projects.noProjects')}
        emptyHint={t('projects.noProjectsHint')}
        rowCountLabel={`${t('common.total')}: ${toFaDigits(data?.meta.total ?? 0)}`}
      />

      <ProjectFormDialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title={t('projects.newProject')}
        campaigns={campaigns?.items ?? []}
        onSubmit={(v) => createProject.mutate(v)}
        loading={createProject.isPending}
      />

      <ProjectFormDialog
        open={!!editTarget}
        onClose={() => setEditTarget(null)}
        title={t('projects.editProject')}
        initial={editTarget}
        campaigns={campaigns?.items ?? []}
        onSubmit={(v) => updateProject.mutate(v)}
        loading={updateProject.isPending}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteProject.mutate(deleteTarget.id)}
        title={t('confirm.title')}
        message={t('projects.confirmDelete')}
        loading={deleteProject.isPending}
      />
    </>
  )
}

function ProjectFormDialog({
  open,
  onClose,
  title,
  initial,
  campaigns,
  onSubmit,
  loading,
}: {
  open: boolean
  onClose: () => void
  title: string
  initial?: Project | null
  campaigns: Array<{ id: string; title: string }>
  onSubmit: (v: ProjectFormValues) => void
  loading: boolean
}) {
  const { t } = useI18n()
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<ProjectFormValues>({
    resolver: zodResolver(projectSchema),
    defaultValues: defaultValues(initial),
  })

  useEffect(() => {
    if (!open) return
    reset(defaultValues(initial))
  }, [open, initial, reset])

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      footer={
        <>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button type="submit" form="project-form" variant="primary" loading={loading}>
            {t('common.save')}
          </Button>
        </>
      }
    >
      <form id="project-form" onSubmit={handleSubmit(onSubmit)}>
        {Object.keys(errors).length > 0 ? (
          <div className="login-error" role="alert">
            {t('errors.validation')}
          </div>
        ) : null}
        <div className="form-row">
          <Input label={t('projects.name')} required error={errors.title?.message} {...register('title')} />
          <select className="select" {...register('campaign_id')}>
            <option value="">{t('projects.noCampaign')}</option>
            {campaigns.map((c) => (
              <option key={c.id} value={c.id}>{c.title}</option>
            ))}
          </select>
        </div>
        <div className="form-row">
          <Textarea label={t('projects.description')} required error={errors.description?.message} rows={3} {...register('description')} />
        </div>
        <div className="form-row">
          <Input label={t('projects.productName')} {...register('product_name')} />
          <Input label={t('projects.valueProposition')} {...register('value_proposition')} />
        </div>
        <div className="form-row">
          <Input label={t('projects.targetMarket')} {...register('target_market')} />
          <Input label={t('projects.salesObjective')} {...register('sales_objective')} />
        </div>
        <div className="form-row">
          <Textarea label={t('projects.idealCustomerProfile')} rows={2} {...register('ideal_customer_profile')} />
          <Textarea label={t('projects.painPoints')} rows={2} {...register('customer_pain_points')} />
        </div>
        <div className="form-row">
          <Textarea label={t('projects.commonObjections')} rows={2} {...register('common_objections')} />
        </div>
      </form>
    </Dialog>
  )
}

function defaultValues(initial?: Project | null): ProjectFormValues {
  if (!initial) {
    return { title: '', description: '' }
  }
  return {
    title: initial.title ?? '',
    description: initial.description ?? '',
    product_name: initial.product_name ?? '',
    product_description: initial.product_description ?? '',
    target_market: initial.target_market ?? '',
    ideal_customer_profile: initial.ideal_customer_profile ?? '',
    customer_pain_points: initial.customer_pain_points ?? '',
    value_proposition: initial.value_proposition ?? '',
    sales_objective: initial.sales_objective ?? '',
    common_objections: initial.common_objections ?? '',
    campaign_id: initial.campaign_id ?? '',
  }
}
