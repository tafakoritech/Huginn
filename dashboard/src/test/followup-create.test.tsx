import { describe, it, expect } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from './test-utils'
import { LeadDetailPage } from '@/pages/leads/LeadDetailPage'
import { FollowupsPage } from '@/pages/FollowupsPage'

const COMPANY = /آریا پلیمر صنعت/

async function openFollowUpDrawer(user: ReturnType<typeof userEvent.setup>) {
  await waitFor(() => expect(screen.getByRole('heading', { name: COMPANY })).toBeInTheDocument(), { timeout: 5000 })
  await user.click(screen.getByRole('button', { name: 'پیگیری' }))
  await waitFor(() => expect(screen.getByRole('dialog', { name: /ایجاد پیگیری/ })).toBeInTheDocument(), { timeout: 5000 })
}

describe('create follow-up', () => {
  it('shows the lead name, all type options, jalali date picker and save/cancel buttons', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LeadDetailPage />, { path: '/leads/:id', url: '/leads/lead-1' })
    await openFollowUpDrawer(user)

    const dialog = screen.getByRole('dialog', { name: /ایجاد پیگیری/ })
    expect(within(dialog).getByText('آریا پلیمر صنعت 1')).toBeInTheDocument()

    expect(within(dialog).getByRole('radio', { name: /تماس/ })).toBeInTheDocument()
    expect(within(dialog).getByRole('radio', { name: /پیگیری/ })).toBeInTheDocument()
    expect(within(dialog).getByRole('radio', { name: /پیام/ })).toBeInTheDocument()
    expect(within(dialog).getByRole('radio', { name: /ایمیل/ })).toBeInTheDocument()

    expect(within(dialog).getByRole('button', { name: /تاریخ سررسید/ })).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: /انصراف/ })).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: /ذخیره/ })).toBeInTheDocument()
  })

  it('selects a follow-up type and a jalali date, saves and shows the new follow-up in the lead tab without a manual refresh', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LeadDetailPage />, { path: '/leads/:id', url: '/leads/lead-1' })
    await openFollowUpDrawer(user)

    await user.click(screen.getByRole('radio', { name: /ایمیل/ }))

    await user.click(screen.getByRole('button', { name: /تاریخ سررسید/ }))
    await user.click(screen.getByRole('button', { name: /امروز ·/ }))

    await user.type(screen.getByPlaceholderText(/اقدام بعدی/), 'ارسال ایمیل به مدیر مجموعه')

    await user.click(screen.getByRole('button', { name: /ذخیره/ }))

    await waitFor(() => expect(screen.queryByRole('dialog', { name: /ایجاد پیگیری/ })).not.toBeInTheDocument(), { timeout: 5000 })

    await user.click(screen.getByRole('tab', { name: /پیگیری‌ها/ }))
    await waitFor(() => expect(screen.getByText('ارسال ایمیل به مدیر مجموعه')).toBeInTheDocument(), { timeout: 5000 })
    expect(screen.getByText('ایمیل')).toBeInTheDocument()
  })

  it('renders the jalali date picker with Persian digits and a today shortcut', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LeadDetailPage />, { path: '/leads/:id', url: '/leads/lead-1' })
    await openFollowUpDrawer(user)

    await user.click(screen.getByRole('button', { name: /تاریخ سررسید/ }))
    await waitFor(() => expect(screen.getByRole('button', { name: /امروز ·/ })).toBeInTheDocument(), { timeout: 5000 })

    const dayCells = screen.getAllByText(/[۰-۹]/)
    expect(dayCells.length).toBeGreaterThan(0)
  })

  it('a saved follow-up appears in the global Follow-ups menu without a manual refresh', async () => {
    const user = userEvent.setup()
    const { unmount } = renderWithProviders(<LeadDetailPage />, { path: '/leads/:id', url: '/leads/lead-1' })
    await openFollowUpDrawer(user)

    await user.click(screen.getByRole('radio', { name: /ایمیل/ }))
    // Pick a clearly-future day (day 15 of the shown Jalali month).
    await user.click(screen.getByRole('button', { name: /تاریخ سررسید/ }))
    await user.click(screen.getByRole('button', { name: /^۱۵ / }))
    await user.type(screen.getByPlaceholderText(/اقدام بعدی/), 'پیگیری برای انتقال به Muninn')

    await user.click(screen.getByRole('button', { name: /ذخیره/ }))
    await waitFor(() => expect(screen.queryByRole('dialog', { name: /ایجاد پیگیری/ })).not.toBeInTheDocument(), { timeout: 5000 })

    unmount()

    renderWithProviders(<FollowupsPage />)
    await waitFor(() => expect(screen.getByRole('tab', { name: /آینده/ })).toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('tab', { name: /آینده/ }))
    await waitFor(() => expect(screen.getByText('پیگیری برای انتقال به Muninn')).toBeInTheDocument(), { timeout: 5000 })
  })
})
