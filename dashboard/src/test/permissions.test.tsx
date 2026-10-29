import { describe, it, expect } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders, testUser } from './test-utils'
import { LeadsPage } from '@/pages/leads/LeadsPage'

describe('permissions', () => {
  it('hides manager actions for viewer role', async () => {
    renderWithProviders(<LeadsPage />, { path: '/leads', user: testUser('viewer') })
    await waitFor(() => expect(screen.getAllByText(/آریا پلیمر صنعت/).length).toBeGreaterThan(0), { timeout: 5000 })
    expect(screen.queryByRole('button', { name: /تخصیص انتخاب‌شده/ })).not.toBeInTheDocument()
  })

  it('shows assignment action for super_admin', async () => {
    renderWithProviders(<LeadsPage />, { path: '/leads', user: testUser('super_admin') })
    await waitFor(() => expect(screen.getAllByText(/آریا پلیمر صنعت/).length).toBeGreaterThan(0), { timeout: 5000 })
    expect(screen.getByRole('button', { name: /فیلتر سرنخ/ })).toBeInTheDocument()
  })
})
