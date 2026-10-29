import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { I18nProvider, useI18n } from '@/i18n'
import { Button } from '@/components/ui/Button'

function LangHarness() {
  const { locale, setLocale, t, dir } = useI18n()
  return (
    <div>
      <span data-testid="dir">{dir}</span>
      <span data-testid="locale">{locale}</span>
      <span data-testid="dashboard-label">{t('nav.dashboard')}</span>
      <Button data-testid="toggle" onClick={() => setLocale(locale === 'fa' ? 'en' : 'fa')}>
        toggle
      </Button>
    </div>
  )
}

describe('RTL / i18n', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('defaults to Persian (RTL) and sets document dir/lang', () => {
    render(
      <I18nProvider>
        <LangHarness />
      </I18nProvider>,
    )
    expect(document.documentElement.getAttribute('dir')).toBe('rtl')
    expect(document.documentElement.getAttribute('lang')).toBe('fa')
    expect(screen.getByTestId('dashboard-label')).toHaveTextContent('داشبورد')
  })

  it('switches to English (LTR) and back, persisting the preference', async () => {
    const user = userEvent.setup()
    render(
      <I18nProvider>
        <LangHarness />
      </I18nProvider>,
    )
    await user.click(screen.getByTestId('toggle'))
    expect(document.documentElement.getAttribute('dir')).toBe('ltr')
    expect(screen.getByTestId('dashboard-label')).toHaveTextContent('Dashboard')
    expect(localStorage.getItem('huginn.locale')).toBe('en')

    await user.click(screen.getByTestId('toggle'))
    expect(document.documentElement.getAttribute('dir')).toBe('rtl')
    expect(screen.getByTestId('dashboard-label')).toHaveTextContent('داشبورد')
  })
})
