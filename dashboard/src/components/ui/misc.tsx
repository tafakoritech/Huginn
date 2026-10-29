import type { ReactNode } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Icon } from '@/components/Icon'
import { initials } from '@/utils/format'
import { useI18n } from '@/i18n'

export function Avatar({ name, size = 30 }: { name?: string | null; size?: number }) {
  return (
    <span className="avatar" style={{ width: size, height: size, fontSize: size / 2.3 }} aria-hidden="true">
      {initials(name)}
    </span>
  )
}

export function Progress({ value, tone = 'accent', className }: { value: number; tone?: 'accent' | 'error' | 'warning'; className?: string }) {
  const pct = Math.min(100, Math.max(0, Math.round(value)))
  const toneClass = tone === 'error' ? 'progress-fill error' : tone === 'warning' ? 'progress-fill warning' : 'progress-fill'
  return (
    <div
      className={`progress-track ${className ?? ''}`.trim()}
      role="progressbar"
      aria-valuenow={pct}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-label={`${pct}%`}
    >
      <div className={toneClass} style={{ width: `${pct}%` }} />
    </div>
  )
}

export function Tabs({
  tabs,
  active,
  onChange,
}: {
  tabs: Array<{ key: string; label: ReactNode; count?: number }>
  active: string
  onChange: (key: string) => void
}) {
  return (
    <div className="tabs" role="tablist">
      {tabs.map((tab) => (
        <button
          key={tab.key}
          type="button"
          role="tab"
          aria-selected={active === tab.key}
          className={`tab ${active === tab.key ? 'active' : ''}`.trim()}
          onClick={() => onChange(tab.key)}
        >
          {tab.label}
          {tab.count !== undefined ? <span className="tab-count">{tab.count}</span> : null}
        </button>
      ))}
    </div>
  )
}

export function PageHeader({
  title,
  subtitle,
  actions,
  breadcrumbs,
}: {
  title: string
  subtitle?: string
  actions?: ReactNode
  breadcrumbs?: Array<{ label: string; to?: string }>
}) {
  return (
    <div className="page-header">
      <div>
        {breadcrumbs && breadcrumbs.length > 0 ? (
          <nav className="breadcrumbs" aria-label="breadcrumb">
            {breadcrumbs.map((b, i) => (
              <span key={i} className="flex items-center gap-1">
                {i > 0 ? <span className="sep">/</span> : null}
                {b.to ? <Link to={b.to}>{b.label}</Link> : <span className="current">{b.label}</span>}
              </span>
            ))}
          </nav>
        ) : null}
        <h1>{title}</h1>
        {subtitle ? <p className="page-subtitle">{subtitle}</p> : null}
      </div>
      {actions ? <div className="page-header-actions">{actions}</div> : null}
    </div>
  )
}

export function useCurrentPath() {
  return useLocation().pathname
}

export function Chip({ children, accent }: { children: ReactNode; accent?: boolean }) {
  return <span className={`chip ${accent ? 'chip-accent' : ''}`.trim()}>{children}</span>
}

export function DetailList({ items }: { items: Array<{ label: string; value: ReactNode }> }) {
  return (
    <dl className="detail-list">
      {items.map((it, i) => (
        <div className="detail-row" key={i}>
          <dt>{it.label}</dt>
          <dd>{it.value ?? <span className="muted">—</span>}</dd>
        </div>
      ))}
    </dl>
  )
}

export function EmptyValue({ label }: { label?: string }) {
  const text = label ?? 'نامشخص'
  return <span className="muted">{text}</span>
}

export function BackButton({ to }: { to: string }) {
  const { t } = useI18n()
  return (
    <Link to={to} className="btn btn-ghost btn-sm">
      <Icon name="arrowRight" size={15} />
      {t('common.back')}
    </Link>
  )
}
