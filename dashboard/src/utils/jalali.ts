import type { Locale } from '@/i18n/strings'
import { TEHRAN_TIME_ZONE, toFaDigits } from './format'

export type JalaliDate = { year: number; month: number; day: number }

export const JALALI_MONTH_NAMES = [
  'فروردین',
  'اردیبهشت',
  'خرداد',
  'تیر',
  'مرداد',
  'شهریور',
  'مهر',
  'آبان',
  'آذر',
  'دی',
  'بهمن',
  'اسفند',
]

const div = (a: number, b: number) => ~~(a / b)
const mod = (a: number, b: number) => a - ~~(a / b) * b

const BREAKS = [-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178]

function jalCal(jy: number): { leap: number; gy: number; march: number } {
  const gy = jy + 621
  let leapJ = -14
  let jp = BREAKS[0]
  let jump = 0
  for (let i = 1; i < BREAKS.length; i += 1) {
    const jm = BREAKS[i]
    jump = jm - jp
    if (jy < jm) break
    leapJ = leapJ + div(jump, 33) * 8 + div(mod(jump, 33), 4)
    jp = jm
  }
  let n = jy - jp
  leapJ = leapJ + div(n, 33) * 8 + div(mod(n, 33) + 3, 4)
  if (mod(jump, 33) === 4 && jump - n === 4) leapJ += 1
  const leapG = div(gy, 4) - div((div(gy, 100) + 1) * 3, 4) - 150
  const march = 20 + leapJ - leapG
  if (jump - n < 6) n = n - jump + div(jump + 4, 33) * 33
  let leap = mod(mod(n + 1, 33) - 1, 4)
  if (leap === -1) leap = 4
  return { leap, gy, march }
}

function g2d(gy: number, gm: number, gd: number): number {
  let d = div((gy + div(gm - 8, 6) + 100100) * 1461, 4) + div(153 * mod(gm + 9, 12) + 2, 5) + gd - 34840408
  d = d - div(div(gy + 100100 + div(gm - 8, 6), 100) * 3, 4) + 752
  return d
}

function d2g(jdn: number): { gy: number; gm: number; gd: number } {
  let j = 4 * jdn + 139361631
  j = j + div(div(4 * jdn + 183187720, 146097) * 3, 4) * 4 - 3908
  const i = div(mod(j, 1461), 4) * 5 + 308
  const gd = div(mod(i, 153), 5) + 1
  const gm = mod(div(i, 153), 12) + 1
  const gy = div(j, 1461) - 100100 + div(8 - gm, 6)
  return { gy, gm, gd }
}

function j2d(jy: number, jm: number, jd: number): number {
  const r = jalCal(jy)
  return g2d(r.gy, 3, r.march) + (jm - 1) * 31 - div(jm, 7) * (jm - 7) + jd - 1
}

function d2j(jdn: number): { jy: number; jm: number; jd: number } {
  const gy = d2g(jdn).gy
  let jy = gy - 621
  const r = jalCal(jy)
  const jdn1f = g2d(gy, 3, r.march)
  let k = jdn - jdn1f
  if (k >= 0) {
    if (k <= 185) return { jy, jm: 1 + div(k, 31), jd: mod(k, 31) + 1 }
    k -= 186
  } else {
    jy -= 1
    k += 179
    if (r.leap === 1) k += 1
  }
  return { jy, jm: 7 + div(k, 30), jd: mod(k, 30) + 1 }
}

export function toJalali(gy: number, gm: number, gd: number): JalaliDate {
  const { jy, jm, jd } = d2j(g2d(gy, gm, gd))
  return { year: jy, month: jm, day: jd }
}

export function toGregorian(jy: number, jm: number, jd: number): { gy: number; gm: number; gd: number } {
  return d2g(j2d(jy, jm, jd))
}

function tehranParts(date: Date): { year: number; month: number; day: number; hour: number; minute: number; second: number } {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: TEHRAN_TIME_ZONE,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).formatToParts(date)
  const get = (t: Intl.DateTimeFormatPartTypes): number => Number(parts.find((p) => p.type === t)?.value ?? 0)
  return { year: get('year'), month: get('month'), day: get('day'), hour: get('hour'), minute: get('minute'), second: get('second') }
}

export function toJalaliDate(date: Date): JalaliDate {
  const p = tehranParts(date)
  return toJalali(p.year, p.month, p.day)
}

export function jalaliToday(): JalaliDate {
  return toJalaliDate(new Date())
}

export function addDaysJalali(date: JalaliDate, days: number): JalaliDate {
  const g = toGregorian(date.year, date.month, date.day)
  const { jy, jm, jd } = d2j(g2d(g.gy, g.gm, g.gd) + days)
  return { year: jy, month: jm, day: jd }
}

function tehranOffsetMsAt(utc: number): number {
  const p = tehranParts(new Date(utc))
  const asUtc = Date.UTC(p.year, p.month - 1, p.day, p.hour, p.minute, p.second)
  return asUtc - utc
}

export function jalaliToIsoDate(jy: number, jm: number, jd: number, hour = 0, minute = 0): string {
  const g = toGregorian(jy, jm, jd)
  const targetUtc = Date.UTC(g.gy, g.gm - 1, g.gd, hour, minute, 0, 0)
  const noonUtc = Date.UTC(g.gy, g.gm - 1, g.gd, 12, 0, 0)
  return new Date(targetUtc - tehranOffsetMsAt(noonUtc)).toISOString()
}

export function tehranTimeOf(iso: string): { hour: number; minute: number } {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return { hour: 0, minute: 0 }
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: TEHRAN_TIME_ZONE,
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(d)
  const get = (t: Intl.DateTimeFormatPartTypes): number => Number(parts.find((p) => p.type === t)?.value ?? 0)
  return { hour: get('hour'), minute: get('minute') }
}

export function jalaliMonthLength(jy: number, jm: number): number {
  const first = j2d(jy, jm, 1)
  const next = jm === 12 ? j2d(jy + 1, 1, 1) : j2d(jy, jm + 1, 1)
  return next - first
}

export function jalaliWeekdayIndex(date: JalaliDate): number {
  const g = toGregorian(date.year, date.month, date.day)
  const weekday = new Date(Date.UTC(g.gy, g.gm - 1, g.gd)).getUTCDay()
  return (weekday + 1) % 7
}

export function formatJalali(date: JalaliDate, locale: Locale): string {
  const monthName = JALALI_MONTH_NAMES[date.month - 1]
  const day = locale === 'fa' ? toFaDigits(date.day) : String(date.day)
  const year = locale === 'fa' ? toFaDigits(date.year) : String(date.year)
  return `${day} ${monthName} ${year}`
}

export function formatJalaliNumeric(date: JalaliDate, locale: Locale): string {
  const pad = (n: number) => (n < 10 ? `0${n}` : String(n))
  const day = pad(date.day)
  const month = pad(date.month)
  const year = String(date.year)
  return locale === 'fa' ? toFaDigits(`${year}/${month}/${day}`) : `${year}/${month}/${day}`
}
