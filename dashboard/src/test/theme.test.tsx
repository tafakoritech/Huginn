import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider, useTheme } from '@/theme/ThemeProvider'
import { Button } from '@/components/ui/Button'

function ThemeHarness() {
  const { theme, toggleTheme } = useTheme()
  return (
    <Button onClick={toggleTheme} data-testid="toggle">
      {theme}
    </Button>
  )
}

describe('theme switching', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })

  it('applies the persisted theme to the document root', () => {
    localStorage.setItem('huginn.theme', 'dark')
    render(
      <ThemeProvider>
        <ThemeHarness />
      </ThemeProvider>,
    )
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(screen.getByText('dark')).toBeInTheDocument()
  })

  it('toggles between light and dark and persists the preference', async () => {
    localStorage.setItem('huginn.theme', 'light')
    const user = userEvent.setup()
    render(
      <ThemeProvider>
        <ThemeHarness />
      </ThemeProvider>,
    )
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')

    await user.click(screen.getByTestId('toggle'))
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(localStorage.getItem('huginn.theme')).toBe('dark')

    await user.click(screen.getByTestId('toggle'))
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })

  it('respects the default theme from config when nothing is stored', () => {
    render(
      <ThemeProvider>
        <ThemeHarness />
      </ThemeProvider>,
    )
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })
})
