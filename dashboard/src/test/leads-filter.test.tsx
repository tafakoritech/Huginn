import { describe, it, expect } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from './test-utils'
import { LeadsPage } from '@/pages/leads/LeadsPage'

describe('lead filters', () => {
  it('renders the leads table with company rows and total count', async () => {
    renderWithProviders(<LeadsPage />, { path: '/leads' })
    await waitFor(() => expect(screen.getAllByText(/آریا پلیمر صنعت/).length).toBeGreaterThan(0), { timeout: 5000 })
    expect(screen.getByText(/۶۴۲/)).toBeInTheDocument()
  })

  it('filters leads by search term', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LeadsPage />, { path: '/leads' })
    await waitFor(() => expect(screen.getAllByText(/آریا پلیمر صنعت/).length).toBeGreaterThan(0), { timeout: 5000 })

    const search = screen.getByRole('searchbox')
    await user.type(search, 'عروسک')

    await waitFor(
      () => {
        expect(screen.queryAllByText(/آریا پلیمر صنعت/)).toHaveLength(0)
        expect(screen.getAllByText(/تولیدی عروسک باران/).length).toBeGreaterThan(0)
      },
      { timeout: 5000 },
    )
  })

  it('applies a status filter through the filter panel', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LeadsPage />, { path: '/leads' })
    await waitFor(() => expect(screen.getAllByText(/آریا پلیمر صنعت/).length).toBeGreaterThan(0), { timeout: 5000 })

    await user.click(screen.getByRole('button', { name: /فیلتر سرنخ/ }))
    const statusSelect = screen.getAllByRole('combobox')[0]
    await user.selectOptions(statusSelect, 'QUALIFIED')
    await user.click(screen.getByRole('button', { name: /اعمال/ }))

    await waitFor(() => expect(screen.getByRole('button', { name: /واجد شرایط/ })).toBeInTheDocument(), { timeout: 5000 })
  })

  it('reads shareable filter state from the URL', async () => {
    renderWithProviders(<LeadsPage />, { path: '/leads', url: '/leads?status=QUALIFIED&score_min=70' })
    await waitFor(() => expect(screen.getByRole('button', { name: /واجد شرایط/ })).toBeInTheDocument(), { timeout: 5000 })
  })
})
