import type { ReactNode } from 'react'
import { useI18n } from '@/i18n'
import { Icon } from '@/components/Icon'
import { useUrlState } from '@/hooks'
import type { PaginationMeta } from '@/api/types'

export function Pagination({
  meta,
  onPage,
}: {
  meta: PaginationMeta
  onPage: (page: number) => void
}) {
  const { t } = useI18n()
  const { page, total_pages: totalPages, total } = meta
  if (total === 0) return null

  const pages: (number | '…')[] = []
  const max = 5
  let start = Math.max(1, page - Math.floor(max / 2))
  const end = Math.min(totalPages, start + max - 1)
  start = Math.max(1, end - max + 1)
  if (start > 1) pages.push(1, '…')
  for (let i = start; i <= end; i++) pages.push(i)
  if (end < totalPages) pages.push('…', totalPages)

  return (
    <div className="pagination" role="navigation" aria-label={t('common.page')}>
      <button className="page-btn" disabled={page <= 1} onClick={() => onPage(page - 1)} aria-label="previous">
        <Icon name="chevronRight" size={15} />
      </button>
      {pages.map((p, i) =>
        p === '…' ? (
          <span key={`e${i}`} className="muted small">
            …
          </span>
        ) : (
          <button key={p} className={`page-btn ${p === page ? 'active' : ''}`} aria-current={p === page ? 'page' : undefined} onClick={() => onPage(p)}>
            {p}
          </button>
        ),
      )}
      <button className="page-btn" disabled={page >= totalPages} onClick={() => onPage(page + 1)} aria-label="next">
        <Icon name="chevronLeft" size={15} />
      </button>
    </div>
  )
}

export function PaginatedFooter({
  meta,
  onPage,
  totalLabel,
}: {
  meta: PaginationMeta
  onPage: (page: number) => void
  totalLabel: string
}) {
  return (
    <div className="table-footer">
      <span className="small secondary">
        {totalLabel} {meta.total}
      </span>
      <Pagination meta={meta} onPage={onPage} />
    </div>
  )
}

export function UrlPagination({ meta }: { meta: PaginationMeta }) {
  const [, setUrl] = useUrlState({ page: '1' })
  return <PaginatedFooter meta={meta} totalLabel="" onPage={(p) => setUrl({ page: String(p) })} />
}

export function SortIndicator({ dir, active }: { dir: 'asc' | 'desc' | null; active: boolean }) {
  if (!active) {
    return <Icon name="chevronsUpDown" size={13} style={{ color: 'var(--text-3)', opacity: 0.6 }} />
  }
  return (
    <span className="sort-ind">
      <Icon name={dir === 'asc' ? 'arrowUp' : 'arrowDown'} size={13} />
    </span>
  )
}

export function EmptyRow({ colSpan, children }: { colSpan: number; children: ReactNode }) {
  return (
    <tr>
      <td colSpan={colSpan} className="table-empty">
        {children}
      </td>
    </tr>
  )
}
