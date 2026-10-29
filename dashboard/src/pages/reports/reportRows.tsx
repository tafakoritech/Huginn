import type { ReactNode } from 'react'
import { EmptyValue } from '@/components/ui/misc'

export function DetailRows({ rows }: { rows: Array<{ label: string; value: ReactNode }> }) {
  return (
    <dl className="detail-list">
      {rows.map((r, i) => (
        <div className="detail-row" key={i}>
          <dt>{r.label}</dt>
          <dd>{r.value ?? <EmptyValue />}</dd>
        </div>
      ))}
    </dl>
  )
}
