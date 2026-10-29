import { describe, it, expect } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from './test-utils'
import { ReportsPage } from '@/pages/reports/ReportsPage'
import { ReportDetailPage } from '@/pages/reports/ReportDetailPage'

describe('mandatory reports', () => {
  it('renders report rows with outcomes and statuses', async () => {
    renderWithProviders(<ReportsPage />, { path: '/reports' })
    await waitFor(() => expect(screen.getAllByText(/تماس موفق/).length).toBeGreaterThan(0), { timeout: 5000 })
    expect(screen.getByText(/گزارش‌های تماس/)).toBeInTheDocument()
  })

  it('highlights manager-overridden reports', async () => {
    renderWithProviders(<ReportsPage />, { path: '/reports' })
    await waitFor(() => expect(screen.getAllByText(/اصلاح مدیر/).length).toBeGreaterThan(0), { timeout: 5000 })
  })

  it('renders the report detail with interaction and override sections', async () => {
    renderWithProviders(<ReportDetailPage />, { path: '/reports/:id', url: '/reports/report-1' })
    await waitFor(() => expect(screen.getByText(/گزارش توسط مدیر بازبینی/)).toBeInTheDocument(), { timeout: 5000 })
    expect(screen.getByText(/اصلاح‌کننده/)).toBeInTheDocument()
    expect(screen.getByText(/دلیل اصلاح/)).toBeInTheDocument()
  })
})
