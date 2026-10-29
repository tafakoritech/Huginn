import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders, testUser } from './test-utils'
import { api } from '@/api'
import { FollowupsPage } from '@/pages/FollowupsPage'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('global follow-up creation', () => {
  it('renders the «ایجاد پیگیری» primary button for authorized roles', async () => {
    renderWithProviders(<FollowupsPage />, { user: testUser('sales_agent') })
    await waitFor(() => expect(screen.getByRole('button', { name: /ایجاد پیگیری/ })).toBeInTheDocument(), { timeout: 5000 })
  })

  it('hides the create button for the viewer role (RBAC gate)', async () => {
    renderWithProviders(<FollowupsPage />, { user: testUser('viewer') })
    await waitFor(() => expect(screen.getByRole('heading', { name: /پیگیری‌ها/ })).toBeInTheDocument(), { timeout: 5000 })
    expect(screen.queryByRole('button', { name: /ایجاد پیگیری/ })).not.toBeInTheDocument()
  })

  it('lead selection is required and POST /followups is called with the correct payload', async () => {
    const spy = vi.spyOn(api, 'createFollowUp')
    const user = userEvent.setup()
    renderWithProviders(<FollowupsPage />, { user: testUser('sales_agent') })

    await user.click(await screen.findByRole('button', { name: /ایجاد پیگیری/ }))
    const dialog = await screen.findByRole('dialog', { name: /ایجاد پیگیری/ })

    // Save is disabled until a lead is selected.
    expect(within(dialog).getByRole('button', { name: /ذخیره/ })).toBeDisabled()

    // Searchable lead selection.
    const searchInput = within(dialog).getByRole('textbox', { name: /سرنخ/ })
    await user.click(searchInput)
    await user.type(searchInput, 'آریا')
    await within(dialog).findByText('آریا پلیمر صنعت 1', {}, { timeout: 5000 })
    await user.click(within(dialog).getByText('آریا پلیمر صنعت 1'))

    expect(within(dialog).getByRole('button', { name: /ذخیره/ })).toBeEnabled()

    // Follow-up type.
    await user.click(within(dialog).getByRole('radio', { name: /ایمیل/ }))

    // Jalali due date (day 15 → future → lands in the upcoming view).
    await user.click(within(dialog).getByRole('button', { name: /تاریخ سررسید/ }))
    await user.click(within(dialog).getByRole('button', { name: /^۱۵ / }))

    // Description.
    await user.type(within(dialog).getByPlaceholderText(/اقدام بعدی/), 'هماهنگی تماس معرفی')

    await user.click(within(dialog).getByRole('button', { name: /ذخیره/ }))

    await waitFor(() => expect(spy).toHaveBeenCalled(), { timeout: 5000 })
    const [body] = spy.mock.calls[0] as [Record<string, unknown>]
    expect(body.lead_id).toBe('lead-1')
    expect(body.type).toBe('EMAIL')
    expect(body.description).toBe('هماهنگی تماس معرفی')
    expect(typeof body.due_at).toBe('string')

    // Dialog closes and the created follow-up appears without a manual refresh.
    await waitFor(() => expect(screen.queryByRole('dialog', { name: /ایجاد پیگیری/ })).not.toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('tab', { name: /آینده/ }))
    await waitFor(() => expect(screen.getByText('هماهنگی تماس معرفی')).toBeInTheDocument(), { timeout: 5000 })
  })
})
