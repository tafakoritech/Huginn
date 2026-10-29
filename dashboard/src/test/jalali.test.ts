import { describe, it, expect } from 'vitest'
import {
  toJalali,
  toGregorian,
  jalaliToIsoDate,
  addDaysJalali,
  jalaliMonthLength,
  formatJalali,
  formatJalaliNumeric,
} from '@/utils/jalali'

describe('jalali (Shamsi) conversion', () => {
  it('converts Nowruz 1403 (2024-03-20) to 1 Farvardin', () => {
    expect(toJalali(2024, 3, 20)).toEqual({ year: 1403, month: 1, day: 1 })
  })

  it('converts 1 Mehr 1403 back to 2024-09-22', () => {
    expect(toGregorian(1403, 7, 1)).toEqual({ gy: 2024, gm: 9, gd: 22 })
  })

  it('roundtrips Gregorian -> Jalali -> Gregorian', () => {
    const samples: Array<[number, number, number]> = [
      [2026, 8, 24],
      [2025, 1, 1],
      [2024, 12, 21],
      [1999, 12, 31],
      [1980, 6, 30],
    ]
    for (const [gy, gm, gd] of samples) {
      const j = toJalali(gy, gm, gd)
      const g = toGregorian(j.year, j.month, j.day)
      expect([g.gy, g.gm, g.gd]).toEqual([gy, gm, gd])
    }
  })

  it('addDaysJalali crosses months and years', () => {
    expect(addDaysJalali({ year: 1403, month: 1, day: 1 }, 1)).toEqual({ year: 1403, month: 1, day: 2 })
    expect(addDaysJalali({ year: 1403, month: 6, day: 31 }, 1)).toEqual({ year: 1403, month: 7, day: 1 })
    expect(addDaysJalali({ year: 1403, month: 12, day: 30 }, 1)).toEqual({ year: 1404, month: 1, day: 1 })
  })

  it('reports correct month lengths (31 for first six, 30 after)', () => {
    expect(jalaliMonthLength(1403, 1)).toBe(31)
    expect(jalaliMonthLength(1403, 6)).toBe(31)
    expect(jalaliMonthLength(1403, 7)).toBe(30)
    expect(jalaliMonthLength(1403, 12)).toBe(30)
  })

  it('formats in Persian digits and month names', () => {
    expect(formatJalali({ year: 1403, month: 1, day: 1 }, 'fa')).toBe('۱ فروردین ۱۴۰۳')
    expect(formatJalaliNumeric({ year: 1403, month: 1, day: 1 }, 'fa')).toBe('۱۴۰۳/۰۱/۰۱')
  })
})

describe('UTC preservation (UI layer conversion only)', () => {
  it('jalaliToIsoDate emits a valid RFC3339 UTC instant', () => {
    const iso = jalaliToIsoDate(1403, 1, 1)
    expect(iso.endsWith('Z')).toBe(true)
    const d = new Date(iso)
    expect(Number.isNaN(d.getTime())).toBe(false)
  })

  it('jalaliToIsoDate keeps the Tehran wall clock intact', () => {
    const iso = jalaliToIsoDate(1403, 1, 1)
    const parts = new Intl.DateTimeFormat('en-CA', {
      timeZone: 'Asia/Tehran',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hourCycle: 'h23',
    }).formatToParts(new Date(iso))
    const get = (t: string) => Number(parts.find((p) => p.type === t)?.value)
    expect([get('year'), get('month'), get('day')]).toEqual([2024, 3, 20])
    expect([get('hour'), get('minute')]).toEqual([0, 0])
  })

  it('jalaliToIsoDate supports an explicit Tehran time', () => {
    const iso = jalaliToIsoDate(1403, 1, 1, 15, 30)
    const parts = new Intl.DateTimeFormat('en-CA', {
      timeZone: 'Asia/Tehran',
      hour: '2-digit',
      minute: '2-digit',
      hourCycle: 'h23',
    }).formatToParts(new Date(iso))
    const get = (t: string) => Number(parts.find((p) => p.type === t)?.value)
    expect([get('hour'), get('minute')]).toEqual([15, 30])
  })
})
