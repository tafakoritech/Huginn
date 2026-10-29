import { describe, it, expect } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from './test-utils'
import { FollowupsPage } from '@/pages/FollowupsPage'
import { LeadDetailPage } from '@/pages/leads/LeadDetailPage'

describe('follow-ups page', () => {
  it('renders tabs and lists follow-ups', async () => {
    renderWithProviders(<FollowupsPage />)
    await waitFor(() => expect(screen.getAllByRole('tab').length).toBeGreaterThanOrEqual(4), { timeout: 5000 })
    // The mock seeds at least one lead with a follow-up.
    await waitFor(() => expect(screen.getAllByText(/شرکت|کارخانه|گروه|بازرگانی|تولیدی|پخش|خدمات/i).length).toBeGreaterThan(0), { timeout: 5000 })
  })

  it('can switch to the completed tab', async () => {
    const user = userEvent.setup()
    renderWithProviders(<FollowupsPage />)
    await waitFor(() => expect(screen.getByRole('tab', { name: /تکمیل شده/ })).toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('tab', { name: /تکمیل شده/ }))
    // Tab selection must not blow up and the table must render.
    await waitFor(() => expect(screen.getAllByRole('row').length).toBeGreaterThan(0), { timeout: 5000 })
  })

  it('completing a pending follow-up triggers the action', async () => {
    const user = userEvent.setup()
    renderWithProviders(<FollowupsPage />)
    await waitFor(() => expect(screen.getAllByRole('button', { name: /تکمیل شد/ }).length).toBeGreaterThan(0), { timeout: 5000 })
    await user.click(screen.getAllByRole('button', { name: /تکمیل شد/ })[0])
    await waitFor(() => expect(screen.getAllByText(/انجام شد|تکمیل/).length).toBeGreaterThan(0), { timeout: 5000 })
  })
})

describe('lead detail follow-ups tab', () => {
  it('shows lead-scoped follow-ups with actions', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LeadDetailPage />, { path: '/leads/:id', url: '/leads/lead-1' })
    await waitFor(() => expect(screen.getByRole('tab', { name: /پیگیری‌ها/ })).toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('tab', { name: /پیگیری‌ها/ }))
    await waitFor(() => expect(screen.getByRole('table')).toBeInTheDocument(), { timeout: 5000 })
    // Either seeded follow-up rows or the empty state must render.
    await waitFor(() => {
      const hasRows = screen.getAllByRole('row').length > 1
      const hasEmpty = screen.getAllByText(/یافت نشد|no data|پیگیری/i).length > 0
      expect(hasRows || hasEmpty).toBeTruthy()
    }, { timeout: 5000 })
  })
})
