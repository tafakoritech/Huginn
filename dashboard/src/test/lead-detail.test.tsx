import { describe, it, expect } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from './test-utils'
import { LeadDetailPage } from '@/pages/leads/LeadDetailPage'

describe('lead detail', () => {
  it('renders the lead hero with company, status and score', async () => {
    renderWithProviders(<LeadDetailPage />, { path: '/leads/:id', url: '/leads/lead-641' })
    await waitFor(() => expect(screen.getByRole('heading', { name: /تولیدی عروسک باران/ })).toBeInTheDocument(), { timeout: 5000 })
    expect(screen.getByText(/واجد شرایط/)).toBeInTheDocument()
  })

  it('shows the evidence tab with confidence bars and excerpts', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LeadDetailPage />, { path: '/leads/:id', url: '/leads/lead-1' })
    await waitFor(() => expect(screen.getByRole('tab', { name: /شواهد/ })).toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('tab', { name: /شواهد/ }))

    await waitFor(() => expect(screen.getAllByText(/تولیدکننده/).length).toBeGreaterThan(0), { timeout: 5000 })
    expect(screen.getAllByText(/٪/).length).toBeGreaterThan(0)
  })

  it('shows the score breakdown tab with dimensions', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LeadDetailPage />, { path: '/leads/:id', url: '/leads/lead-1' })
    await waitFor(() => expect(screen.getByRole('tab', { name: /امتیاز/ })).toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('tab', { name: /امتیاز/ }))

    await waitFor(() => expect(screen.getByText(/تناسب کسب‌وکار/)).toBeInTheDocument(), { timeout: 5000 })
    expect(screen.getByText(/امتیاز کل/)).toBeInTheDocument()
  })

  it('shows project memberships with stage and next action', async () => {
    renderWithProviders(<LeadDetailPage />, { path: '/leads/:id', url: '/leads/lead-1' })
    await waitFor(() => expect(screen.getByText(/پروژه‌ها/)).toBeInTheDocument(), { timeout: 5000 })
    // The seeded membership for proj-1 carries MEETING_COMPLETED and an action.
    expect(screen.getAllByText('ERP Implementation').length).toBeGreaterThan(0)
    expect(screen.getByText('MEETING_COMPLETED')).toBeInTheDocument()
    expect(screen.getByText(/Call to follow up/)).toBeInTheDocument()
  })
})
