import type { Locale } from '@/i18n/strings'

/**
 * Formats a number for display. In Persian locale it renders Persian digits,
 * but phone numbers, codes, URLs and IDs must keep Latin digits (callers use
 * the raw value or `formatRaw` for those).
 */
export function formatNumber(n: number | null | undefined, locale: Locale): string {
  if (n === null || n === undefined || Number.isNaN(n)) return '—'
  return new Intl.NumberFormat(locale === 'fa' ? 'fa-IR' : 'en-US').format(n)
}

export function formatPercent(n: number | null | undefined, locale: Locale, digits = 0): string {
  if (n === null || n === undefined || Number.isNaN(n)) return '—'
  return new Intl.NumberFormat(locale === 'fa' ? 'fa-IR' : 'en-US', {
    maximumFractionDigits: digits,
  }).format(n)
}

export const TEHRAN_TIME_ZONE = 'Asia/Tehran'

export function toTehranDateOnly(d: Date): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: TEHRAN_TIME_ZONE,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(d)
}

const isSameDay = (a: Date, b: Date) => toTehranDateOnly(a) === toTehranDateOnly(b)

export function formatDateTime(
  iso: string | null | undefined,
  locale: Locale,
  opts: { date?: boolean; time?: boolean; relativeToday?: boolean } = { date: true, time: true },
): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  const parts: string[] = []
  if (opts.relativeToday && isSameDay(d, new Date())) {
    parts.push(locale === 'fa' ? 'امروز' : 'today')
  } else if (opts.date !== false) {
    parts.push(
      new Intl.DateTimeFormat(locale === 'fa' ? 'fa-IR' : 'en-US', {
        timeZone: TEHRAN_TIME_ZONE,
        calendar: 'persian',
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      }).format(d),
    )
  }
  if (opts.time !== false) {
    parts.push(
      new Intl.DateTimeFormat(locale === 'fa' ? 'fa-IR' : 'en-US', {
        timeZone: TEHRAN_TIME_ZONE,
        hour: '2-digit',
        minute: '2-digit',
      }).format(d),
    )
  }
  return parts.join('، ')
}

export function formatDate(iso: string | null | undefined, locale: Locale): string {
  return formatDateTime(iso, locale, { time: false })
}

export function formatTrendLabel(label: string, locale: Locale): string {
  const day = /^\d{4}-\d{2}-\d{2}$/.exec(label)
  if (day) {
    return new Intl.DateTimeFormat(locale === 'fa' ? 'fa-IR' : 'en-US', {
      timeZone: TEHRAN_TIME_ZONE,
      calendar: 'persian',
      month: 'short',
      day: 'numeric',
    }).format(new Date(`${label}T00:00:00Z`))
  }
  const month = /^\d{4}-\d{2}$/.exec(label)
  if (month) {
    return new Intl.DateTimeFormat(locale === 'fa' ? 'fa-IR' : 'en-US', {
      timeZone: TEHRAN_TIME_ZONE,
      calendar: 'persian',
      month: 'short',
      year: 'numeric',
    }).format(new Date(`${label}-01T00:00:00Z`))
  }
  return label
}

export function formatDuration(seconds: number | null | undefined): string {
  if (seconds === null || seconds === undefined) return '—'
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return m > 0 ? `${m}:${String(s).padStart(2, '0')}` : `${s}s`
}

export function formatBytes(bytes: number | null | undefined): string {
  if (bytes === null || bytes === undefined) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function initials(name: string | null | undefined): string {
  if (!name) return '?'
  const parts = name.trim().split(/\s+/)
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

export function toFaDigits(input: string | number): string {
  const faDigits = '۰۱۲۳۴۵۶۷۸۹'
  return String(input).replace(/[0-9]/g, (d) => faDigits[Number(d)])
}

export function formatScore(n: number, locale: Locale): string {
  return locale === 'fa' ? toFaDigits(n) : String(n)
}

export function dirFor(locale: Locale): 'rtl' | 'ltr' {
  return locale === 'fa' ? 'rtl' : 'ltr'
}
