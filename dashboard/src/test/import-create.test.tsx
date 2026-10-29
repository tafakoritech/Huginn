import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ThemeProvider } from '@/theme/ThemeProvider'
import { I18nProvider } from '@/i18n'
import { ToastProvider } from '@/components/ui/Toast'
import { AuthProvider } from '@/test/authHarness'
import { ImportNewPage } from '@/pages/imports/ImportNewPage'
import { ImportDetailPage } from '@/pages/imports/ImportDetailPage'
import { testUser } from '@/test/test-utils'

function renderImportFlow() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <ThemeProvider>
        <I18nProvider>
          <ToastProvider>
            <MemoryRouter initialEntries={['/imports/new']}>
              <AuthProvider user={testUser()}>
                <Routes>
                  <Route path="/imports/new" element={<ImportNewPage />} />
                  <Route path="/imports/:id" element={<ImportDetailPage />} />
                </Routes>
              </AuthProvider>
            </MemoryRouter>
          </ToastProvider>
        </I18nProvider>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

describe('import creation', () => {
  it('creates a batch and navigates to its detail page', async () => {
    const user = userEvent.setup()
    renderImportFlow()

    await user.type(screen.getByLabelText(/نام دسته/), 'دسته بازار آزمایشی')
    await user.type(screen.getByLabelText(/توضیحات/), 'توضیحات تست')
    await user.click(screen.getByRole('button', { name: /ورودی جدید/ }))

    await waitFor(() => expect(screen.getByRole('heading', { name: 'دسته بازار آزمایشی' })).toBeInTheDocument(), { timeout: 6000 })
    expect(screen.getByText(/پیشرفت/)).toBeInTheDocument()
  })

  it('renders the dropzone and processing note', () => {
    renderImportFlow()
    expect(screen.getByText(/فایل‌ها را اینجا رها کنید/)).toBeInTheDocument()
    expect(screen.getByText(/پردازش به‌صورت ناهمگام/)).toBeInTheDocument()
  })
})
