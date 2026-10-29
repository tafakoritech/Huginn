import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits, formatDateTime } from '@/utils/format'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader } from '@/components/ui/misc'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog, ConfirmDialog, Drawer } from '@/components/ui/Dialog'
import { Input, Select } from '@/components/ui/Field'
import { Icon } from '@/components/Icon'
import { Tabs } from '@/components/ui/misc'
import type { ColumnDef, User } from './types'

const baseUserFields = {
  name: z.string().min(2),
  username: z.string().min(3),
  email: z.string().email().or(z.literal('')).optional(),
  mobile: z.string().optional(),
  role: z.string().min(1),
}

// Create requires a password; edit never touches it, so an empty password must
// be accepted there or the save button silently fails validation.
const userSchema = z.object({ ...baseUserFields, password: z.string().min(6) })
const editUserSchema = z.object({ ...baseUserFields, password: z.string().or(z.literal('')).optional() })

type UserFormValues = z.infer<typeof userSchema>

const ROLES = ['super_admin', 'sales_development_manager', 'sales_agent', 'viewer']

export function UsersPage() {
  const { t, locale } = useI18n()
  const toast = useToastApi()
  const qc = useQueryClient()
  const [tab, setTab] = useState('users')
  const [createOpen, setCreateOpen] = useState(false)
  const [editUser, setEditUser] = useState<User | null>(null)
  const [resetTarget, setResetTarget] = useState<User | null>(null)
  const [deactivateTarget, setDeactivateTarget] = useState<User | null>(null)
  const [sessionsUser, setSessionsUser] = useState<User | null>(null)

  const { data: users, isLoading } = useQuery({ queryKey: queryKeys.users, queryFn: () => api.listUsers() })

  const invalidate = () => void qc.invalidateQueries({ queryKey: queryKeys.users })

  const createUser = useMutation({
    mutationFn: (v: UserFormValues) =>
      api.createUser({ name: v.name, username: v.username, email: v.email || undefined, mobile: v.mobile || undefined, password: v.password ?? '', role: v.role }),
    onSuccess: () => {
      toast.success(t('common.save'))
      setCreateOpen(false)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const updateUser = useMutation({
    mutationFn: (v: UserFormValues) => api.updateUser(editUser?.id ?? '', { name: v.name, email: v.email || undefined, mobile: v.mobile || undefined, role: v.role }),
    onSuccess: () => {
      toast.success(t('common.save'))
      setEditUser(null)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const setActive = useMutation({
    mutationFn: (v: { id: string; active: boolean }) => api.setUserActive(v.id, v.active),
    onSuccess: () => {
      toast.success(t('common.save'))
      setDeactivateTarget(null)
      invalidate()
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const resetPassword = useMutation({
    mutationFn: (v: { id: string; password: string }) => api.resetUserPassword(v.id, v.password),
    onSuccess: () => {
      toast.success(t('common.save'))
      setResetTarget(null)
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const columns = useMemo<ColumnDef<User, unknown>[]>(() => {
    return [
      {
        id: 'name',
        header: t('common.name'),
        cell: ({ row }) => (
          <div className="flex flex-col">
            <span className="strong">{row.original.name}</span>
            <span className="small muted" dir="ltr">{row.original.username}</span>
          </div>
        ),
      },
      { id: 'role', header: t('settings.role'), cell: ({ row }) => <Badge tone="accent">{t(`roles.${row.original.role}`)}</Badge>, enableSorting: false },
      { id: 'email', header: t('settings.email'), cell: ({ row }) => row.original.email ?? <span className="muted">—</span>, enableSorting: false },
      { id: 'mobile', header: t('settings.mobile'), cell: ({ row }) => row.original.mobile ?? <span className="muted">—</span>, enableSorting: false },
      {
        id: 'active',
        header: t('common.status'),
        cell: ({ row }) =>
          row.original.is_active ? <Badge tone="success">{t('common.active')}</Badge> : <Badge tone="neutral">{t('common.inactive')}</Badge>,
        enableSorting: false,
      },
      {
        id: 'last_login',
        header: t('settings.lastLogin'),
        cell: ({ row }) => (row.original.last_login_at ? formatDateTime(row.original.last_login_at, locale, { time: false }) : <span className="muted">—</span>),
        enableSorting: false,
      },
      {
        id: 'actions',
        header: t('common.actions'),
        enableSorting: false,
        cell: ({ row }) => {
          const u = row.original
          return (
            <div className="row-actions">
              <Button size="sm" variant="ghost" icon="edit" onClick={() => setEditUser(u)} aria-label={t('common.edit')} />
              <Button size="sm" variant="ghost" icon="key" onClick={() => setResetTarget(u)} aria-label={t('settings.resetPassword')} />
              <Button size="sm" variant="ghost" icon="activity" onClick={() => setSessionsUser(u)} aria-label={t('settings.sessions')} />
              <Button
                size="sm"
                variant="ghost"
                icon={u.is_active ? 'lock' : 'check'}
                onClick={() => (u.is_active ? setDeactivateTarget(u) : setActive.mutate({ id: u.id, active: true }))}
                aria-label={u.is_active ? t('settings.deactivate') : t('settings.activate')}
              />
            </div>
          )
        },
      },
    ]
  }, [t, locale, setActive])

  return (
    <>
      <PageHeader
        title={t('settings.usersTitle')}
        subtitle={t('settings.usersSubtitle')}
        breadcrumbs={[{ label: t('nav.settings'), to: '/settings/users' }]}
        actions={
          <Button icon="plus" onClick={() => setCreateOpen(true)}>
            {t('settings.createUser')}
          </Button>
        }
      />

      <Tabs
        tabs={[
          { key: 'users', label: t('nav.users') },
          { key: 'roles', label: t('nav.roles') },
        ]}
        active={tab}
        onChange={setTab}
      />

      <div className="mt-5">
        {tab === 'users' ? (
          <DataTable
            columns={columns}
            data={users ?? []}
            loading={isLoading}
            emptyTitle={t('settings.noUsers')}
          />
        ) : (
          <RolesTable t={t} />
        )}
      </div>

      <UserFormDialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title={t('settings.createUser')}
        onSubmit={(v) => createUser.mutate(v)}
        loading={createUser.isPending}
      />

      <UserFormDialog
        open={!!editUser}
        onClose={() => setEditUser(null)}
        title={t('settings.editUser')}
        initial={editUser}
        onSubmit={(v) => updateUser.mutate(v)}
        loading={updateUser.isPending}
        hidePassword
      />

      <ResetPasswordDialog open={!!resetTarget} user={resetTarget} onClose={() => setResetTarget(null)} onConfirm={(p) => resetPassword.mutate({ id: resetTarget?.id ?? '', password: p })} loading={resetPassword.isPending} />

      <ConfirmDialog
        open={!!deactivateTarget}
        onClose={() => setDeactivateTarget(null)}
        onConfirm={() => deactivateTarget && setActive.mutate({ id: deactivateTarget.id, active: false })}
        title={t('settings.deactivate')}
        message={t('settings.confirmDeactivate')}
        loading={setActive.isPending}
      />

      <SessionsDrawer open={!!sessionsUser} user={sessionsUser} onClose={() => setSessionsUser(null)} />
    </>
  )
}

function UserFormDialog({
  open,
  onClose,
  title,
  initial,
  onSubmit,
  loading,
  hidePassword,
}: {
  open: boolean
  onClose: () => void
  title: string
  initial?: User | null
  onSubmit: (v: UserFormValues) => void
  loading: boolean
  hidePassword?: boolean
}) {
  const { t } = useI18n()
  const defaultValues = (): UserFormValues =>
    initial
      ? { name: initial.name, username: initial.username, email: initial.email ?? '', mobile: initial.mobile ?? '', role: initial.role, password: '' }
      : { name: '', username: '', email: '', mobile: '', role: 'sales_agent', password: '' }

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<UserFormValues>({
    resolver: zodResolver(hidePassword ? editUserSchema : userSchema),
    defaultValues: defaultValues(),
  })

  // The dialog stays mounted; reinitialize the form whenever it opens with a
  // (possibly different) target user so "ویرایش" never shows stale/empty data.
  useEffect(() => {
    if (!open) return
    reset(
      initial
        ? { name: initial.name, username: initial.username, email: initial.email ?? '', mobile: initial.mobile ?? '', role: initial.role, password: '' }
        : { name: '', username: '', email: '', mobile: '', role: 'sales_agent', password: '' },
    )
  }, [open, initial, reset])

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      footer={
        <>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button type="submit" form="user-form" variant="primary" loading={loading}>
            {t('common.save')}
          </Button>
        </>
      }
    >
      <form id="user-form" onSubmit={handleSubmit(onSubmit)}>
        {Object.keys(errors).length > 0 ? (
          <div className="login-error" role="alert">
            {t('errors.validation')}
          </div>
        ) : null}
        <div className="form-row">
          <Input label={t('common.name')} required error={errors.name?.message} {...register('name')} />
          <Input label="Username" required error={errors.username?.message} dir="ltr" {...register('username')} />
        </div>
        <div className="form-row">
          <Input label={t('settings.email')} dir="ltr" error={errors.email?.message} {...register('email')} />
          <Input label={t('settings.mobile')} dir="ltr" {...register('mobile')} />
        </div>
        <div className="form-row">
          <Select label={t('settings.role')} required {...register('role')}>
            {ROLES.map((r) => (
              <option key={r} value={r}>
                {t(`roles.${r}`)}
              </option>
            ))}
          </Select>
          {!hidePassword ? <Input label={t('auth.password')} type="password" dir="ltr" error={errors.password?.message} {...register('password')} /> : null}
        </div>
      </form>
    </Dialog>
  )
}

function ResetPasswordDialog({
  open,
  user,
  onClose,
  onConfirm,
  loading,
}: {
  open: boolean
  user: User | null
  onClose: () => void
  onConfirm: (password: string) => void
  loading: boolean
}) {
  const { t } = useI18n()
  const [password, setPassword] = useState('')
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t('settings.resetPassword')}
      footer={
        <>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button variant="primary" disabled={password.length < 6} loading={loading} onClick={() => onConfirm(password)}>
            {t('common.confirm')}
          </Button>
        </>
      }
    >
      <div className="field">
        <label className="field-label">{t('auth.password')} — {user?.name}</label>
        <input type="password" className="input" dir="ltr" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" />
      </div>
    </Dialog>
  )
}

function SessionsDrawer({ open, user, onClose }: { open: boolean; user: User | null; onClose: () => void }) {
  const { t, locale } = useI18n()
  const toast = useToastApi()
  const qc = useQueryClient()
  const { data: sessions = [] } = useQuery({
    queryKey: queryKeys.userSessions(user?.id ?? ''),
    queryFn: () => api.listUserSessions(user?.id ?? ''),
    enabled: !!user,
  })

  const revoke = useMutation({
    mutationFn: (sid: string) => api.revokeSession(user?.id ?? '', sid),
    onSuccess: () => {
      toast.success(t('common.save'))
      void qc.invalidateQueries({ queryKey: queryKeys.userSessions(user?.id ?? '') })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  return (
    <Drawer open={open} onClose={onClose} title={`${t('settings.sessions')} — ${user?.name ?? ''}`}>
      {sessions.map((s) => (
        <div className="upload-item mb-3" key={s.id}>
          <Icon name="activity" size={16} style={{ color: 'var(--text-3)' }} />
          <div className="flex-1 flex flex-col">
            <span className="strong">{s.device ?? '—'}</span>
            <span className="small muted" dir="ltr">
              {s.ip} · {formatDateTime(s.created_at, locale)}
            </span>
          </div>
          <Button size="sm" variant="danger-ghost" icon="x" onClick={() => revoke.mutate(s.id)}>
            {t('common.remove')}
          </Button>
        </div>
      ))}
    </Drawer>
  )
}

function RolesTable({ t }: { t: (k: string) => string }) {
  const roles = [
    { key: 'super_admin', perms: 14 },
    { key: 'sales_development_manager', perms: 12 },
    { key: 'sales_agent', perms: 3 },
    { key: 'viewer', perms: 1 },
  ]
  return (
    <div className="table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th>{t('settings.role')}</th>
            <th>{t('common.status')}</th>
            <th>{t('common.total')}</th>
          </tr>
        </thead>
        <tbody>
          {roles.map((r) => (
            <tr key={r.key}>
              <td className="strong">{t(`roles.${r.key}`)}</td>
              <td>
                <Badge tone="success">{t('common.active')}</Badge>
              </td>
              <td>{toFaDigits(r.perms)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
