import { describe, it, expect, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from './test-utils'
import { LeadDetailPage } from '@/pages/leads/LeadDetailPage'

describe('lead detail localization', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('renders history/timeline and score labels in Persian', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LeadDetailPage />, { path: '/leads/:id', url: '/leads/lead-1' })

    // History / timeline
    await waitFor(() => expect(screen.getByRole('tab', { name: 'تاریخچه' })).toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('tab', { name: 'تاریخچه' }))
    await waitFor(() => expect(screen.getByText(/ایجاد شد/)).toBeInTheDocument(), { timeout: 5000 })

    // No raw machine keys or i18n paths may leak into the timeline.
    expect(screen.queryByText(/lead\.created/)).toBeNull()
    expect(screen.queryByText(/leads\.timeline/)).toBeNull()

    // Score
    await user.click(screen.getByRole('tab', { name: 'امتیاز' }))
    await waitFor(() => expect(screen.getByText(/تناسب کسب‌وکار/)).toBeInTheDocument(), { timeout: 5000 })
    expect(screen.queryByText(/business_fit/)).toBeNull()
    expect(screen.queryByText(/contactability/)).toBeNull()
  })

  it('keeps English translations when English is active', async () => {
    localStorage.setItem('huginn.locale', 'en')
    const user = userEvent.setup()
    renderWithProviders(<LeadDetailPage />, { path: '/leads/:id', url: '/leads/lead-1' })

    await waitFor(() => expect(screen.getByRole('tab', { name: 'History' })).toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('tab', { name: 'History' }))
    await waitFor(() => expect(screen.getByText(/Created/)).toBeInTheDocument(), { timeout: 5000 })

    await user.click(screen.getByRole('tab', { name: 'Score' }))
    await waitFor(() => expect(screen.getByText(/Business fit/)).toBeInTheDocument(), { timeout: 5000 })
  })
})
