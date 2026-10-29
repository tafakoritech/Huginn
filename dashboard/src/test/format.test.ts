import { describe, it, expect } from 'vitest'
import { formatDate, formatDateTime, toFaDigits } from '@/utils/format'

describe('Shamsi (Jalali) date display in the Dashboard', () => {
  it('formatDate renders a Jalali date in Persian', () => {
    // 2024-03-20 00:00 UTC = 1 Farvardin 1403 in Tehran
    const out = formatDate('2024-03-20T00:00:00Z', 'fa')
    expect(out).toMatch(/۱ فروردین/)
    expect(out).toMatch(/۱۴۰۳/)
  })

  it('formatDate keeps the date stable across the Tehran/UTC boundary', () => {
    // 2024-09-22 21:00 UTC is already 23 Mehr 1403 in Tehran
    const out = formatDate('2024-09-22T21:00:00Z', 'fa')
    expect(out).toMatch(/مهر/)
    expect(out).toMatch(/۱۴۰۳/)
  })

  it('formatDateTime includes Tehran wall-clock time', () => {
    const out = formatDateTime('2024-03-20T09:30:00Z', 'fa', { date: true, time: true })
    expect(out).toMatch(/۱۳:۰۰/)
  })

  it('formatDate falls back gracefully on invalid input', () => {
    expect(formatDate(null, 'fa')).toBe('—')
    expect(formatDate('not-a-date', 'fa')).toBe('—')
  })

  it('toFaDigits converts latin digits to Persian', () => {
    expect(toFaDigits('2026/08/24')).toBe('۲۰۲۶/۰۸/۲۴')
  })
})
