import { describe, it, expect } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from './test-utils'
import { api } from '@/api'
import { ProjectDetailPage } from '@/pages/projects/ProjectDetailPage'

describe('project detail — attach existing leads', () => {
  it('shows related leads and does not offer a create-lead form', async () => {
    renderWithProviders(<ProjectDetailPage />, { path: '/projects/:id', url: '/projects/proj-1' })
    await waitFor(() => expect(screen.getByText('ERP Implementation')).toBeInTheDocument(), { timeout: 5000 })
    expect(screen.queryByRole('button', { name: /سرنخ جدید/ })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /اتصال سرنخ موجود/ })).toBeInTheDocument()
  })

  it('attaches an existing lead through the picker dialog', async () => {
    const user = userEvent.setup()
    renderWithProviders(<ProjectDetailPage />, { path: '/projects/:id', url: '/projects/proj-1' })
    await waitFor(() => expect(screen.getByRole('button', { name: /اتصال سرنخ موجود/ })).toBeInTheDocument(), { timeout: 5000 })

    await user.click(screen.getByRole('button', { name: /اتصال سرنخ موجود/ }))
    await waitFor(() => expect(screen.getByRole('dialog', { name: /انتخاب/ })).toBeInTheDocument(), { timeout: 5000 })

    let enabled: HTMLElement | undefined
    await waitFor(() => {
      const boxes = screen.getAllByRole('checkbox')
      enabled = boxes.find((c) => !(c as HTMLInputElement).disabled)
      expect(enabled).toBeTruthy()
    }, { timeout: 5000 })
    await user.click(enabled as HTMLElement)

    await user.click(screen.getByRole('button', { name: /انتخاب/ }))
    await waitFor(() => expect(screen.getByText(/متصل شدند/)).toBeInTheDocument(), { timeout: 5000 })
  })

  it('removes a lead connection without deleting the lead', async () => {
    const user = userEvent.setup()
    renderWithProviders(<ProjectDetailPage />, { path: '/projects/:id', url: '/projects/proj-1' })
    await waitFor(() => expect(screen.getByText('ERP Implementation')).toBeInTheDocument(), { timeout: 5000 })

    const row = screen.getByText('آریا پلیمر صنعت 1').closest('tr') as HTMLElement
    const removeButton = row.querySelector('button[aria-label*="حذف اتصال"]') as HTMLElement
    expect(removeButton).toBeTruthy()
    await user.click(removeButton)

    await waitFor(() => expect(screen.getByText(/جدا شود/)).toBeInTheDocument(), { timeout: 5000 })
    await user.click(screen.getByRole('button', { name: /تأیید/ }))
    await waitFor(() => expect(screen.getByText(/جدا شد/)).toBeInTheDocument(), { timeout: 5000 })

    // The global lead must still exist after detaching it from the project.
    const list = await api.listLeads({ search: 'آریا پلیمر صنعت 1', page_size: 10 })
    expect(list.items.some((l) => l.company_name === 'آریا پلیمر صنعت 1')).toBe(true)
  })
})
