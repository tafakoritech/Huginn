import { describe, it, expect } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders, testUser } from './test-utils'
import { QualifiedPage } from '@/pages/leads/QualifiedPage'

describe('muninn export state', () => {
  it('shows exported leads with Muninn ID badges', async () => {
    renderWithProviders(<QualifiedPage />, { path: '/leads/qualified' })
    await waitFor(() => expect(screen.getByText(/سرنخ‌های واجد شرایط/)).toBeInTheDocument(), { timeout: 5000 })
    await waitFor(() => expect(screen.getByText(/MN-/)).toBeInTheDocument(), { timeout: 5000 })
  })

  it('displays failed export state with attempt count and error', async () => {
    renderWithProviders(<QualifiedPage />, { path: '/leads/qualified' })
    await waitFor(() => expect(screen.getAllByText(/connection timeout to muninn/).length).toBeGreaterThan(0), { timeout: 5000 })
  })

  it('queues an export through the confirm dialog', async () => {
    const user = userEvent.setup()
    renderWithProviders(<QualifiedPage />, { path: '/leads/qualified', user: testUser('super_admin') })
    await waitFor(() => expect(screen.getAllByRole('button', { name: /انتقال به Muninn/ }).length).toBeGreaterThan(0), { timeout: 5000 })

    await user.click(screen.getAllByRole('button', { name: /انتقال به Muninn/ })[0])
    await waitFor(() => expect(screen.getByText(/برای انتقال این سرنخ به Muninn مطمئن هستید/)).toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('button', { name: /تأیید/ }))

    await waitFor(() => expect(screen.getByText(/انتقال به Muninn در صف قرار گرفت/)).toBeInTheDocument(), { timeout: 5000 })
  })
})
