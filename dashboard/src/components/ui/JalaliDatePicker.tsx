import { useState } from 'react'
import { useI18n } from '@/i18n'
import { Icon } from '@/components/Icon'
import { useClickOutside } from '@/hooks'
import {
  JALALI_MONTH_NAMES,
  type JalaliDate,
  formatJalali,
  formatJalaliNumeric,
  jalaliMonthLength,
  jalaliToday,
  jalaliWeekdayIndex,
} from '@/utils/jalali'
import { toFaDigits } from '@/utils/format'

const FA_WEEKDAYS = ['ش', 'ی', 'د', 'س', 'چ', 'پ', 'ج']
const EN_WEEKDAYS = ['Sa', 'Su', 'Mo', 'Tu', 'We', 'Th', 'Fr']

export function JalaliDatePicker({
  value,
  onChange,
  placeholder,
  id,
}: {
  value: JalaliDate | null
  onChange: (value: JalaliDate) => void
  placeholder?: string
  id?: string
}) {
  const { t, locale } = useI18n()
  const [open, setOpen] = useState(false)
  const initial = value ?? jalaliToday()
  const [view, setView] = useState<JalaliDate>({ year: initial.year, month: initial.month, day: 1 })

  const openPicker = () => {
    if (value) setView({ year: value.year, month: value.month, day: 1 })
    else {
      const today = jalaliToday()
      setView({ year: today.year, month: today.month, day: 1 })
    }
    setOpen(true)
  }

  const ref = useClickOutside<HTMLDivElement>(() => setOpen(false))

  const prevMonth = () => setView((v) => (v.month === 1 ? { year: v.year - 1, month: 12, day: 1 } : { year: v.year, month: v.month - 1, day: 1 }))
  const nextMonth = () => setView((v) => (v.month === 12 ? { year: v.year + 1, month: 1, day: 1 } : { year: v.year, month: v.month + 1, day: 1 }))
  const prevYear = () => setView((v) => ({ ...v, year: v.year - 1 }))
  const nextYear = () => setView((v) => ({ ...v, year: v.year + 1 }))

  const today = jalaliToday()
  const length = jalaliMonthLength(view.year, view.month)
  const firstWeekday = jalaliWeekdayIndex({ year: view.year, month: view.month, day: 1 })
  const cells: Array<number | null> = [
    ...Array.from({ length: firstWeekday }, () => null),
    ...Array.from({ length }, (_, i) => i + 1),
  ]
  const weekdays = locale === 'fa' ? FA_WEEKDAYS : EN_WEEKDAYS

  return (
    <div className="jalali-picker" ref={ref}>
      <button
        type="button"
        id={id}
        className={`jalali-picker-trigger ${open ? 'is-open' : ''}`}
        onClick={() => (open ? setOpen(false) : openPicker())}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-label={t('followups.dueDate')}
      >
        <Icon name="calendar" size={16} />
        <span className={value ? '' : 'is-placeholder'}>
          {value ? formatJalaliNumeric(value, locale) : (placeholder ?? t('followups.dueDate'))}
        </span>
        <Icon name="chevronDown" size={14} className="jalali-picker-caret" />
      </button>

      {open ? (
        <div className="jalali-popover" role="dialog" aria-label={t('followups.dueDate')}>
          <div className="jalali-popover-nav">
            <div className="jalali-nav-years">
              <button type="button" className="jalali-nav-btn" onClick={prevYear} aria-label={t('common.prevYear')}>
                <Icon name="chevronRight" size={14} />
              </button>
              <span className="jalali-nav-year">{locale === 'fa' ? toFaDigits(view.year) : String(view.year)}</span>
              <button type="button" className="jalali-nav-btn" onClick={nextYear} aria-label={t('common.nextYear')}>
                <Icon name="chevronLeft" size={14} />
              </button>
            </div>
            <div className="jalali-nav-month">
              <button type="button" className="jalali-nav-btn" onClick={prevMonth} aria-label={t('common.prevMonth')}>
                <Icon name="chevronRight" size={14} />
              </button>
              <span className="jalali-nav-month-name">{JALALI_MONTH_NAMES[view.month - 1]}</span>
              <button type="button" className="jalali-nav-btn" onClick={nextMonth} aria-label={t('common.nextMonth')}>
                <Icon name="chevronLeft" size={14} />
              </button>
            </div>
          </div>

          <div className="jalali-grid">
            {weekdays.map((w, i) => (
              <span key={`${w}-${i}`} className="jalali-weekday">
                {w}
              </span>
            ))}
            {cells.map((day, i) => {
              if (day === null) return <span key={`blank-${i}`} className="jalali-cell" />
              const isSelected = value?.year === view.year && value?.month === view.month && value?.day === day
              const isToday = today.year === view.year && today.month === view.month && today.day === day
              return (
                <button
                  key={day}
                  type="button"
                  className={`jalali-cell ${isSelected ? 'is-selected' : ''} ${isToday ? 'is-today' : ''}`}
                  onClick={() => {
                    onChange({ year: view.year, month: view.month, day })
                    setOpen(false)
                  }}
                  aria-label={formatJalali({ year: view.year, month: view.month, day }, locale)}
                  aria-selected={isSelected}
                >
                  {locale === 'fa' ? toFaDigits(day) : String(day)}
                </button>
              )
            })}
          </div>

          <div className="jalali-popover-footer">
            <button
              type="button"
              className="jalali-today-btn"
              onClick={() => {
                onChange(today)
                setOpen(false)
              }}
            >
              {locale === 'fa' ? `امروز · ${formatJalali(today, 'fa')}` : `Today · ${formatJalali(today, 'en')}`}
            </button>
          </div>
        </div>
      ) : null}
    </div>
  )
}
