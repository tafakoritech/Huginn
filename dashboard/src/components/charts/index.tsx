import { useId, useMemo } from 'react'
import { useI18n } from '@/i18n'
import { toFaDigits } from '@/utils/format'
import type { SeriesPoint } from '@/api/types'

const PALETTE = ['#69b437', '#91da73', '#23643d', '#165a54', '#3c82c4', '#d99a28', '#49a85a', '#80918c', '#5aa22e', '#0e3338']
const GRID = 'var(--divider)'
const TEXT = 'var(--text-3)'
const INK = 'var(--text-1)'

export function useChartColors(): string[] {
  return PALETTE
}

/* ================= BarChart (HTML-based, RTL-safe) ================= */
export function BarChart({ data, height = 220, color = PALETTE[0] }: { data: SeriesPoint[]; height?: number; color?: string }) {
  const max = Math.max(...data.map((d) => d.value), 1)
  return (
    <div role="img" aria-label={data.map((d) => `${d.label}: ${d.value}`).join('، ')}>
      <div className="flex items-end gap-2" style={{ height, direction: 'ltr' }}>
        {data.map((d, i) => (
          <div key={i} className="flex-1 flex flex-col justify-end" style={{ height: '100%' }}>
            <div
              style={{
                height: `${(d.value / max) * 100}%`,
                background: color,
                borderRadius: '4px 4px 0 0',
                minHeight: 2,
                opacity: 0.9,
              }}
              title={`${d.label}: ${d.value}`}
            />
          </div>
        ))}
      </div>
      <div className="flex gap-2" style={{ direction: 'ltr' }}>
        {data.map((d, i) => (
          <div key={i} className="flex-1 text-center small muted ellipsis" style={{ fontSize: 10.5, paddingTop: 6 }}>
            {d.label}
          </div>
        ))}
      </div>
    </div>
  )
}

/* ================= LineChart (SVG, RTL-aware) ================= */
export function LineChart({
  data,
  height = 220,
  color = PALETTE[0],
  showValues = true,
}: {
  data: SeriesPoint[]
  height?: number
  color?: string
  showValues?: boolean
}) {
  const { fa } = useI18n()
  const uid = useId()
  const W = 620
  const H = 240
  const padL = 8
  const padR = 8
  const padT = 18
  const padB = 26

  const { points, yLabels, xLabels } = useMemo(() => {
    const n = data.length
    const max = Math.max(...data.map((d) => d.value), 1)
    const innerW = W - padL - padR
    const innerH = H - padT - padB
    const stepX = n > 1 ? innerW / (n - 1) : 0
    // Reverse X order in RTL so time flows right-to-left.
    const pts = data.map((d, i) => {
      const xi = fa ? W - padR - i * stepX : padL + i * stepX
      const yi = padT + innerH - (d.value / max) * innerH
      return { x: xi, y: yi, value: d.value, label: d.label }
    })
    const yTicks = 4
    const yl = Array.from({ length: yTicks + 1 }, (_, i) => Math.round((max / yTicks) * i))
    const xl = pts
    return { points: pts, yLabels: yl, xLabels: xl }
  }, [data, fa, W, H, padL, padR, padT, padB])

  const path = useMemo(() => points.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x},${p.y}`).join(' '), [points])
  const area = useMemo(() => `${path} L${points[points.length - 1]?.x ?? 0},${H - padB} L${points[0]?.x ?? 0},${H - padB} Z`, [path, points, H, padB])

  return (
    <div role="img" aria-label={data.map((d) => `${d.label}: ${d.value}`).join('، ')}>
      <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height, display: 'block' }}>
        <defs>
          <linearGradient id={`lg-${uid}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity="0.25" />
            <stop offset="100%" stopColor={color} stopOpacity="0.02" />
          </linearGradient>
        </defs>
        {yLabels.map((v, i) => {
          const y = padT + ((H - padT - padB) * (1 - v / Math.max(...yLabels, 1)))
          return (
            <g key={i}>
              <line x1={padL} y1={y} x2={W - padR} y2={y} stroke={GRID} strokeWidth="1" strokeDasharray={i === 0 ? '' : '3 4'} />
              <text x={fa ? W - 4 : 4} y={y - 3} fontSize="9" fill={TEXT} textAnchor={fa ? 'end' : 'start'} style={{ direction: 'ltr' }}>
                {toFaDigits(v)}
              </text>
            </g>
          )
        })}
        <path d={area} fill={`url(#lg-${uid})`} />
        <path d={path} fill="none" stroke={color} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
        {points.map((p, i) => (
          <g key={i}>
            <circle cx={p.x} cy={p.y} r="3.5" fill={color} stroke={INK} strokeWidth="1.5" />
            {showValues ? (
              <text x={p.x} y={p.y - 8} fontSize="9" fill={TEXT} textAnchor="middle" style={{ direction: 'ltr' }}>
                {toFaDigits(p.value)}
              </text>
            ) : null}
          </g>
        ))}
        {xLabels.map((p, i) => (
          <text key={i} x={p.x} y={H - 6} fontSize="9" fill={TEXT} textAnchor="middle" style={{ direction: 'ltr' }}>
            {p.label}
          </text>
        ))}
      </svg>
    </div>
  )
}

/* ================= DonutChart (conic gradient, RTL-safe) ================= */
export function DonutChart({ data, size = 180, thickness = 26 }: { data: SeriesPoint[]; size?: number; thickness?: number }) {
  const { locale } = useI18n()
  const total = data.reduce((a, b) => a + b.value, 0)
  const segments = data.map((d, i) => ({ ...d, color: PALETTE[i % PALETTE.length] }))
  const allColors = segments.map((s) => `${s.color} ${(s.value / total) * 360}deg`).join(', ')
  const bg = segments.length === 1 && segments[0].value > 0 ? segments[0].color : `conic-gradient(${allColors})`

  return (
    <div className="flex flex-col items-center gap-3">
      <div style={{ position: 'relative', width: size, height: size }} role="img" aria-label={data.map((d) => `${d.label}: ${d.value}`).join('، ')}>
        <div style={{ width: '100%', height: '100%', borderRadius: '50%', background: bg }} />
        <div
          style={{
            position: 'absolute',
            inset: thickness,
            borderRadius: '50%',
            background: 'var(--bg-surface)',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <span style={{ fontSize: 20, fontWeight: 800, fontVariantNumeric: 'tabular-nums' }}>{locale === 'fa' ? toFaDigits(total) : total}</span>
          <span className="small muted" style={{ fontSize: 10.5 }}>
            {locale === 'fa' ? 'مجموع' : 'total'}
          </span>
        </div>
      </div>
      <div className="chart-legend">
        {segments.map((s, i) => (
          <span key={i} className="legend-item">
            <span className="legend-swatch" style={{ background: s.color }} />
            {s.label} · <b>{locale === 'fa' ? toFaDigits(s.value) : s.value}</b>
          </span>
        ))}
      </div>
    </div>
  )
}

/* ================= FunnelChart (HTML, RTL-safe) ================= */
export function FunnelChart({ steps, labels }: { steps: number[]; labels?: string[] }) {
  const { t, fa } = useI18n()
  const max = Math.max(...steps, 1)
  const defaultLabels = ['raw', 'assigned', 'contacted', 'interested', 'qualified', 'muninn'].map((k) => t(`dashboard.funnel.${k}`))
  const labelMap = labels ?? defaultLabels

  return (
    <div className="flex flex-col gap-2" role="img" aria-label={steps.map((s, i) => `${labelMap[i]}: ${s}`).join('، ')}>
      {steps.map((s, i) => {
        const pct = Math.max(6, Math.round((s / max) * 100))
        return (
          <div key={i} className="flex items-center gap-3">
            <span className="small secondary" style={{ width: 84, flexShrink: 0, textAlign: 'start' }}>
              {labelMap[i]}
            </span>
            <div className="flex-1 flex items-center gap-2" style={{ direction: 'ltr' }}>
              <div className="progress-track" style={{ height: 18 }}>
                <div
                  className="progress-fill"
                  style={{
                    width: `${pct}%`,
                    borderRadius: 4,
                    opacity: 0.95 - i * 0.09,
                  }}
                />
              </div>
            </div>
            <b style={{ width: 44, textAlign: 'end', fontVariantNumeric: 'tabular-nums', fontSize: 13 }}>{fa ? toFaDigits(s) : s}</b>
          </div>
        )
      })}
    </div>
  )
}

/* ================= Stacked/Grouped bar chart for qualification by source ================= */
export function QualifiedBarChart({
  data,
  height = 200,
}: {
  data: Array<{ label: string; total: number; qualified: number }>
  height?: number
}) {
  const { locale } = useI18n()
  const max = Math.max(...data.map((d) => d.total), 1)
  return (
    <div role="img" aria-label={data.map((d) => `${d.label}: ${d.qualified}/${d.total}`).join('، ')}>
      <div className="flex items-end gap-2" style={{ height, direction: 'ltr' }}>
        {data.map((d, i) => (
          <div key={i} className="flex-1 flex flex-col justify-end" style={{ height: '100%' }}>
            <div className="flex flex-col" style={{ width: '100%', height: `${(d.total / max) * 100}%`, minHeight: 4 }}>
              <div style={{ flex: d.qualified, background: '#69b437', borderRadius: '4px 4px 0 0' }} title={`${d.qualified} qualified`} />
              <div style={{ flex: Math.max(d.total - d.qualified, 0), background: 'var(--border-default)' }} title={`${d.total - d.qualified} rest`} />
            </div>
          </div>
        ))}
      </div>
      <div className="flex gap-2" style={{ direction: 'ltr' }}>
        {data.map((d, i) => (
          <div key={i} className="flex-1 text-center small muted ellipsis" style={{ fontSize: 10.5, paddingTop: 6 }} title={d.label}>
            {d.label}
          </div>
        ))}
      </div>
      <div className="chart-legend mt-2">
        <span className="legend-item">
          <span className="legend-swatch" style={{ background: '#69b437' }} />
          {locale === 'fa' ? 'واجد شرایط' : 'qualified'}
        </span>
        <span className="legend-item">
          <span className="legend-swatch" style={{ background: 'var(--border-default)' }} />
          {locale === 'fa' ? 'سایر' : 'rest'}
        </span>
      </div>
    </div>
  )
}

export function chartPalette(index: number): string {
  return PALETTE[index % PALETTE.length]
}
