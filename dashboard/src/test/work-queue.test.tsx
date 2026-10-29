import { describe, it, expect } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from './test-utils'
import { WorkQueuePage } from '@/pages/WorkQueuePage'

describe('work queue page', () => {
  it('renders tabs and lists next actions', async () => {
    renderWithProviders(<WorkQueuePage />)
    await waitFor(() => expect(screen.getAllByRole('tab').length).toBeGreaterThanOrEqual(8), { timeout: 5000 })
    await waitFor(() => expect(screen.getByRole('table')).toBeInTheDocument(), { timeout: 5000 })
    // The mock seeds a pending-report item that must surface with its CTA.
    await waitFor(() => expect(screen.getAllByText(/ثبت گزارش/i).length).toBeGreaterThan(0), { timeout: 5000 })
  })

  it('can switch to the urgent filter', async () => {
    const user = userEvent.setup()
    renderWithProviders(<WorkQueuePage />)
    await waitFor(() => expect(screen.getByRole('tab', { name: /فوری/ })).toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('tab', { name: /فوری/ }))
    await waitFor(() => expect(screen.getAllByRole('row').length).toBeGreaterThan(0), { timeout: 5000 })
  })
})
