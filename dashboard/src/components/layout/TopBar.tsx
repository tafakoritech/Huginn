import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { useTheme } from '@/theme/ThemeProvider'
import { useAuth } from '@/auth/AuthContext'
import { Icon } from '@/components/Icon'
import { Avatar } from '@/components/ui/misc'
import { Dropdown, MenuItem, MenuSeparator } from '@/components/ui/Dropdown'
import { GlobalSearch } from './GlobalSearch'
import { NotificationsBell } from './Notifications'

export function TopBar({
  onMobileMenu,
  currentPage,
}: {
  onMobileMenu: () => void
  currentPage: string
}) {
  const { t, locale, setLocale, fa } = useI18n()
  const { theme, toggleTheme } = useTheme()
  const { user, logout, isPrivileged, can } = useAuth()
  const navigate = useNavigate()

  const { data: health } = useQuery({
    queryKey: queryKeys.health,
    queryFn: () => api.backendHealth(),
    refetchInterval: 30_000,
    enabled: isPrivileged,
  })

  const { data: jobs } = useQuery({
    queryKey: queryKeys.pendingJobs,
    queryFn: () => api.jobsPending(),
    refetchInterval: 30_000,
    enabled: isPrivileged,
  })

  return (
    <header className="topbar">
      <button type="button" className="topbar-icon-btn mobile-menu-btn" onClick={onMobileMenu} aria-label={t('common.more')}>
        <Icon name="menu" size={20} />
      </button>

      <span className="topbar-current-page">{currentPage}</span>

      <GlobalSearch />

      <div className="topbar-actions">
        {isPrivileged ? (
          <div className="flex items-center gap-2" title={health?.status ? t(`header.health${health.status === 'ok' ? 'Healthy' : 'Down'}`) : undefined}>
            <span className={`health-dot ${health?.status === 'ok' ? '' : 'down'}`.trim()} aria-hidden="true" />
            {jobs && jobs.count > 0 ? (
              <span className="badge badge-info" title={t('header.processingJobs')}>
                <Icon name="activity" size={12} />
                {jobs.count}
              </span>
            ) : null}
          </div>
        ) : null}

        <button type="button" className="topbar-icon-btn" onClick={toggleTheme} aria-label={theme === 'light' ? t('theme.dark') : t('theme.light')}>
          <Icon name={theme === 'light' ? 'moon' : 'sun'} size={19} />
        </button>

        <button
          type="button"
          className="topbar-icon-btn"
          onClick={() => setLocale(locale === 'fa' ? 'en' : 'fa')}
          aria-label={locale === 'fa' ? 'English' : 'فارسی'}
          title={locale === 'fa' ? 'English' : 'فارسی'}
          style={{ fontSize: 12, fontWeight: 800 }}
        >
          {fa ? 'EN' : 'فا'}
        </button>

        <NotificationsBell />

        <Dropdown
          align="end"
          trigger={
            <button type="button" className="topbar-user" aria-label={t('header.profile')}>
              <Avatar name={user?.name} size={30} />
              <span className="small strong nowrap" style={{ maxWidth: 120 }}>
                {user?.name}
              </span>
              <Icon name="chevronDown" size={14} style={{ color: 'var(--text-3)' }} />
            </button>
          }
        >
          <div className="menu-label">{user?.role ? t(`roles.${user.role}`) : ''}</div>
          {can('users.manage') ? (
            <MenuItem icon="settings" onClick={() => navigate('/settings/users')}>
              {t('header.settings')}
            </MenuItem>
          ) : null}
          <MenuSeparator />
          <MenuItem icon="logout" danger onClick={() => void logout()}>
            {t('auth.logout')}
          </MenuItem>
        </Dropdown>
      </div>
    </header>
  )
}
