import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { Icon, type IconName } from '@/components/Icon'
import { Dropdown } from '@/components/ui/Dropdown'
import { formatDateTime } from '@/utils/format'
import type { NotificationItem } from '@/api/types'

function notifIcon(type: NotificationItem['type']): IconName {
  switch (type) {
    case 'batch_completed':
      return 'checkCircle'
    case 'batch_failed':
      return 'xCircle'
    case 'duplicates_review':
      return 'duplicates'
    case 'pending_reports':
      return 'reports'
    case 'overdue_followups':
      return 'clock'
    case 'muninn_export_failure':
      return 'alertTriangle'
    case 'high_value_qualified':
      return 'star'
  }
}

export function NotificationsBell() {
  const { t, locale } = useI18n()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { data: items = [] } = useQuery({ queryKey: queryKeys.notifications, queryFn: () => api.notifications(), refetchInterval: 60_000 })
  const unread = items.filter((n) => !n.read).length

  const markAll = useMutation({
    mutationFn: () => api.markNotificationsRead(),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notifications }),
  })

  return (
    <Dropdown
      align="end"
      className="notif-panel"
      trigger={
        <button type="button" className="topbar-icon-btn" aria-label={t('header.notifications')}>
          <Icon name="bell" size={19} />
          {unread > 0 ? <span className="topbar-badge">{unread > 9 ? '9+' : unread}</span> : null}
        </button>
      }
    >
      <div className="flex items-center justify-between" style={{ padding: '10px 12px 6px' }}>
        <span className="strong">{t('notifications.title')}</span>
        {unread > 0 ? (
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => markAll.mutate()}>
            {t('notifications.markAllRead')}
          </button>
        ) : null}
      </div>
      {items.length === 0 ? (
        <div className="state-block" style={{ padding: 24 }}>
          <p className="muted">{t('notifications.empty')}</p>
        </div>
      ) : (
        items.slice(0, 20).map((n) => (
          <button
            key={n.id}
            type="button"
            className="notif-item"
            style={{ width: '100%', textAlign: 'start', cursor: 'pointer' }}
            onClick={() => {
              if (n.link) navigate(n.link)
            }}
          >
            <Icon name={notifIcon(n.type)} size={18} style={{ color: 'var(--accent)', flexShrink: 0, marginTop: 2 }} />
            <span className="flex flex-col gap-1">
              <span className="notif-title">{n.title}</span>
              {n.body ? <span className="small secondary">{n.body}</span> : null}
              <span className="notif-time">{formatDateTime(n.created_at, locale, { time: true })}</span>
            </span>
          </button>
        ))
      )}
    </Dropdown>
  )
}
