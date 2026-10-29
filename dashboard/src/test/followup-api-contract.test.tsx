import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from './test-utils'
import { api } from '@/api'
import { LeadDetailPage } from '@/pages/leads/LeadDetailPage'
import { FollowupsPage } from '@/pages/FollowupsPage'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('follow-up API contract', () => {
  it('LeadDetailPage creates via the nested endpoint WITHOUT lead_id in the body', async () => {
    const spy = vi.spyOn(api, 'addFollowUp')
    const user = userEvent.setup()
    renderWithProviders(<LeadDetailPage />, { path: '/leads/:id', url: '/leads/lead-1' })

    await waitFor(() => expect(screen.getByRole('heading', { name: /آریا پلیمر صنعت/ })).toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('button', { name: 'پیگیری' }))
    await waitFor(() => expect(screen.getByRole('dialog', { name: /ایجاد پیگیری/ })).toBeInTheDocument(), { timeout: 5000 })

    await user.click(screen.getByRole('radio', { name: /ایمیل/ }))
    await user.type(screen.getByPlaceholderText(/اقدام بعدی/), 'تماس مجدد با مدیر مجموعه')
    await user.click(screen.getByRole('button', { name: /ذخیره/ }))

    await waitFor(() => expect(spy).toHaveBeenCalled(), { timeout: 5000 })
    const [leadId, body] = spy.mock.calls[0] as [string, Record<string, unknown>]
    expect(leadId).toBe('lead-1')
    expect(body.type).toBe('EMAIL')
    expect(body.description).toBe('تماس مجدد با مدیر مجموعه')
    expect(body).not.toHaveProperty('lead_id')
    expect(Object.keys(body)).not.toContain('lead_id')
  })

  it('FollowupsPage creates via the global endpoint WITH lead_id in the body', async () => {
    const spy = vi.spyOn(api, 'createFollowUp')
    const user = userEvent.setup()
    renderWithProviders(<FollowupsPage />)

    await waitFor(() => expect(screen.getByRole('button', { name: /ایجاد پیگیری/ })).toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('button', { name: /ایجاد پیگیری/ }))
    await waitFor(() => expect(screen.getByRole('dialog', { name: /ایجاد پیگیری/ })).toBeInTheDocument(), { timeout: 5000 })

    const dialog = screen.getByRole('dialog', { name: /ایجاد پیگیری/ })
    const searchInput = within(dialog).getByRole('textbox', { name: /سرنخ/ })
    await user.click(searchInput)
    await user.type(searchInput, 'آریا')
    const leadOption = await within(dialog).findByText('آریا پلیمر صنعت 1', {}, { timeout: 5000 })
    await user.click(leadOption)
    await user.click(screen.getByRole('radio', { name: /تماس/ }))
    // Pick a clearly-future day so the new item lands in the upcoming tab.
    await user.click(screen.getByRole('button', { name: /تاریخ سررسید/ }))
    await user.click(screen.getByRole('button', { name: /^۱۵ / }))
    await user.type(screen.getByPlaceholderText(/اقدام بعدی/), 'هماهنگی جلسه معرفی')
    await user.click(screen.getByRole('button', { name: /ذخیره/ }))

    await waitFor(() => expect(spy).toHaveBeenCalled(), { timeout: 5000 })
    const [body] = spy.mock.calls[0] as [Record<string, unknown>]
    expect(body.lead_id).toBe('lead-1')
    expect(body.type).toBe('CALL')
    expect(body.description).toBe('هماهنگی جلسه معرفی')
    expect(typeof body.due_at).toBe('string')

    // Dialog closes on success and the list query is refreshed.
    await waitFor(() => expect(screen.queryByRole('dialog', { name: /ایجاد پیگیری/ })).not.toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('tab', { name: /آینده/ }))
    await waitFor(() => expect(screen.getByText('هماهنگی جلسه معرفی')).toBeInTheDocument(), { timeout: 5000 })
  })
})
