import type { ReactNode } from 'react'
import { Icon, type IconName } from '@/components/Icon'
import { useI18n } from '@/i18n'

export function EmptyState({
  title,
  hint,
  icon = 'raven',
  actions,
}: {
  title: string
  hint?: string
  icon?: IconName
  actions?: ReactNode
}) {
  return (
    <div className="state-block">
      <Icon name={icon} size={46} />
      <h3>{title}</h3>
      {hint ? <p>{hint}</p> : null}
      {actions ? <div className="state-actions">{actions}</div> : null}
    </div>
  )
}

export function LoadingState({ label }: { label?: string }) {
  const { t } = useI18n()
  return (
    <div className="loading-block" role="status">
      <span className="spinner" aria-hidden="true" />
      <span>{label ?? t('common.loading')}</span>
    </div>
  )
}

export function ErrorState({
  title,
  detail,
  onRetry,
}: {
  title?: string
  detail?: string
  onRetry?: () => void
}) {
  const { t } = useI18n()
  return (
    <div className="state-block" role="alert">
      <Icon name="alertTriangle" size={42} />
      <h3>{title ?? t('errors.generic')}</h3>
      {detail ? <p className="mono">{detail}</p> : null}
      {onRetry ? (
        <div className="state-actions">
          <button className="btn btn-secondary" onClick={onRetry}>
            <Icon name="refresh" size={15} />
            {t('common.retry')}
          </button>
        </div>
      ) : null}
    </div>
  )
}

export function Skeleton({ className, style }: { className?: string; style?: React.CSSProperties }) {
  return <div className={`skeleton ${className ?? ''}`.trim()} style={style} />
}

export function TableSkeleton({ rows = 6, cols = 6 }: { rows?: number; cols?: number }) {
  return (
    <div className="table-wrap" aria-hidden="true">
      <table className="table">
        <thead>
          <tr>
            {Array.from({ length: cols }, (_, i) => (
              <th key={i}>
                <Skeleton className="skeleton-text" style={{ width: `${60 + ((i * 13) % 35)}%` }} />
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {Array.from({ length: rows }, (_, r) => (
            <tr key={r}>
              {Array.from({ length: cols }, (_, c) => (
                <td key={c}>
                  <Skeleton className="skeleton-row" />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function CardSkeletonGrid({ count = 8 }: { count?: number }) {
  return (
    <div className="metric-grid">
      {Array.from({ length: count }, (_, i) => (
        <div className="card" key={i} style={{ height: 96 }}>
          <Skeleton className="skeleton-text" style={{ width: '45%' }} />
          <Skeleton className="skeleton-title" style={{ marginTop: 12 }} />
        </div>
      ))}
    </div>
  )
}
