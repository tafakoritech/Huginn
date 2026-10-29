import { describe, it, expect } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from './test-utils'
import { LeadsPage } from '@/pages/leads/LeadsPage'

describe('lead management', () => {
  it('creates an individual lead with a source', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LeadsPage />, { path: '/leads' })

    await waitFor(() => expect(screen.getAllByText(/آریا پلیمر صنعت/).length).toBeGreaterThan(0), { timeout: 5000 })

    await user.click(screen.getByRole('button', { name: /سرنخ جدید/ }))
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument(), { timeout: 5000 })

    await user.type(screen.getByLabelText(/نام مجموعه/), 'سرنخ دستی تست')
    await user.type(screen.getByLabelText(/منبع/), 'دستی')
    await user.click(screen.getByRole('button', { name: /ذخیره/ }))

    await waitFor(() => expect(screen.getByText('سرنخ دستی تست')).toBeInTheDocument(), { timeout: 5000 })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
