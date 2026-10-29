import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ThemeProvider } from '@/theme/ThemeProvider'
import { I18nProvider } from '@/i18n'
import { ToastProvider } from '@/components/ui/Toast'
import { AuthProvider } from '@/auth/AuthContext'
import { LoginPage } from '@/pages/LoginPage'
import { config } from '@/config'

function renderLogin() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <ThemeProvider>
        <I18nProvider>
          <ToastProvider>
            <MemoryRouter initialEntries={['/login']}>
              <AuthProvider>
                <Routes>
                  <Route path="/login" element={<LoginPage />} />
                  <Route path="/dashboard" element={<div data-testid="dashboard-page">Dashboard</div>} />
                </Routes>
              </AuthProvider>
            </MemoryRouter>
          </ToastProvider>
        </I18nProvider>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

describe('authentication', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('renders the login form with Persian labels', () => {
    renderLogin()
    expect(screen.getByRole('heading', { name: /خوش آمدید/ })).toBeInTheDocument()
    expect(screen.getByLabelText(/نام کاربری/)).toBeInTheDocument()
    expect(screen.getByLabelText(/گذرواژه/)).toBeInTheDocument()
  })

  it('logs in and navigates to the dashboard, persisting the user', async () => {
    const user = userEvent.setup()
    renderLogin()
    await user.type(screen.getByLabelText(/نام کاربری/), 'admin')
    await user.type(screen.getByLabelText(/گذرواژه/), 'password123')
    await user.click(screen.getByRole('button', { name: /ورود/ }))

    await waitFor(() => expect(screen.getByTestId('dashboard-page')).toBeInTheDocument(), { timeout: 3000 })
    expect(localStorage.getItem(config.storageKeys.accessToken)).toBeTruthy()
    const stored = JSON.parse(localStorage.getItem(config.storageKeys.user) ?? '{}')
    expect(stored.role).toBe('super_admin')
  })

  it('shows an error for a too-short password', async () => {
    const user = userEvent.setup()
    renderLogin()
    await user.type(screen.getByLabelText(/نام کاربری/), 'admin')
    await user.type(screen.getByLabelText(/گذرواژه/), '12')
    await user.click(screen.getByRole('button', { name: /ورود/ }))
    await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument(), { timeout: 3000 })
  })
})
