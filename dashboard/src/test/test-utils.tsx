import { render } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { ThemeProvider } from '@/theme/ThemeProvider'
import { I18nProvider } from '@/i18n'
import { ToastProvider } from '@/components/ui/Toast'
import { AuthProvider } from './authHarness'
import type { ReactNode } from 'react'
import type { AuthUser } from '@/api/types'

export function createQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 0, gcTime: Infinity } },
  })
}

type Options = {
  /** Route pattern used for useParams, e.g. "/leads/:id" */
  path?: string
  /** Initial URL the router navigates to. Defaults to `path`. */
  url?: string
  user?: AuthUser | null
}

export function renderWithProviders(ui: ReactNode, { path = '/', url = path, user = testUser() }: Options = {}) {
  const qc = createQueryClient()
  return render(
    <QueryClientProvider client={qc}>
      <ThemeProvider>
        <I18nProvider>
          <ToastProvider>
            <MemoryRouter initialEntries={[url]}>
              <AuthProvider user={user}>
                <Routes>
                  <Route path={path} element={ui} />
                </Routes>
              </AuthProvider>
            </MemoryRouter>
          </ToastProvider>
        </I18nProvider>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

export function testUser(role: AuthUser['role'] = 'super_admin'): AuthUser {
  return { id: 'u-test', name: 'مدیر تست', username: 'test', role }
}
