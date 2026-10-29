import type { ReactNode } from 'react'
import { useI18n } from '@/i18n'
import { toFaDigits } from '@/utils/format'
import type { BatchStatus, LeadStatus, ReportStatus } from '@/api/types'

type Tone = 'neutral' | 'success' | 'info' | 'warning' | 'error' | 'accent' | 'brand'

const toneClass: Record<Tone, string> = {
  neutral: 'badge-neutral',
  success: 'badge-success',
  info: 'badge-info',
  warning: 'badge-warning',
  error: 'badge-error',
  accent: 'badge-accent',
  brand: 'badge-brand',
}

export function Badge({
  tone = 'neutral',
  dot,
  children,
  className,
}: {
  tone?: Tone
  dot?: boolean
  children: ReactNode
  className?: string
}) {
  return (
    <span className={`badge ${toneClass[tone]} ${className ?? ''}`.trim()}>
      {dot ? <span className="dot" /> : null}
      {children}
    </span>
  )
}

const statusTone: Record<string, Tone> = {
  NEW: 'info',
  READY: 'info',
  ASSIGNED: 'accent',
  CONTACTING: 'info',
  FOLLOW_UP: 'warning',
  QUALIFIED: 'success',
  DISQUALIFIED: 'error',
  EXPORTED_TO_MUNINN: 'brand',
  ARCHIVED: 'neutral',
  UPLOADED: 'neutral',
  QUEUED: 'neutral',
  PROCESSING: 'info',
  REVIEW_REQUIRED: 'warning',
  COMPLETED: 'success',
  FAILED: 'error',
  PENDING: 'warning',
  SUBMITTED: 'success',
  VOIDED_BY_MANAGER: 'info',
  OPEN: 'info',
  OVERDUE: 'error',
  CANCELLED: 'neutral',
  IN_PROGRESS: 'info',
  SUCCEEDED: 'success',
  SKIPPED: 'neutral',
  POSSIBLE: 'warning',
  MERGED: 'success',
  DISTINCT: 'neutral',
  IGNORED: 'neutral',
  SUCCEEDED2: 'success',
}

export function StatusBadge({ status, label }: { status: string; label?: string }) {
  const { t } = useI18n()
  const tone = statusTone[status] ?? 'neutral'
  return (
    <Badge tone={tone} dot={status === 'PROCESSING' || status === 'QUEUED' || status === 'IN_PROGRESS' || status === 'OPEN' || status === 'PENDING' || status === 'CONTACTING'}>
      {label ?? t(`status.${status}`)}
    </Badge>
  )
}

export function LeadStatusBadge({ status }: { status: LeadStatus }) {
  return <StatusBadge status={status} />
}

export function ScoreBadge({ score }: { score: number }) {
  const cls = score >= 75 ? 'score-high' : score >= 50 ? 'score-mid' : 'score-low'
  return (
    <span className={`score-badge ${cls}`} title={score >= 75 ? 'high' : score >= 50 ? 'medium' : 'low'}>
      {toFaDigits(score)}
    </span>
  )
}

export function ConfidenceBadge({ value }: { value: number }) {
  const pct = Math.round(value * 100)
  return (
    <span className="confidence-bar" role="img" aria-label={`${toFaDigits(pct)}%`}>
      <span className="confidence-track">
        <span className="confidence-fill" style={{ width: `${Math.min(100, Math.max(0, pct))}%` }} />
      </span>
      <span className="confidence-value">{toFaDigits(pct)}٪</span>
    </span>
  )
}

export function ImportStatusBadge({ status }: { status: BatchStatus }) {
  return <StatusBadge status={status} />
}

export function ReportStatusBadge({ status }: { status: ReportStatus }) {
  return <StatusBadge status={status} />
}
