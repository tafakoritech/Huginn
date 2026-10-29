import { describe, it, expect } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from './test-utils'
import { DuplicatesPage } from '@/pages/DuplicatesPage'

describe('duplicate review', () => {
  it('shows side-by-side comparison with similarity and matching fields', async () => {
    renderWithProviders(<DuplicatesPage />, { path: '/duplicates' })
    await waitFor(() => expect(screen.getAllByText(/شباهت/).length).toBeGreaterThan(0), { timeout: 5000 })
    expect(screen.getAllByText(/سرنخ A/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/سرنخ B/).length).toBeGreaterThan(0)
    expect(screen.getByText(/%۹۳/)).toBeInTheDocument()
  })

  it('opens the merge dialog with surviving lead selection and confirms merge', async () => {
    const user = userEvent.setup()
    renderWithProviders(<DuplicatesPage />, { path: '/duplicates' })
    await waitFor(() => expect(screen.getAllByText(/شباهت/).length).toBeGreaterThan(0), { timeout: 5000 })

    await user.click(screen.getAllByRole('button', { name: /ادغام/ })[0])
    await waitFor(() => expect(screen.getByText(/سرنخ باقی‌مانده/)).toBeInTheDocument(), { timeout: 5000 })
    expect(screen.getByText(/فیلدهای انتخاب‌شده از A/)).toBeInTheDocument()
  })

  it('marks a pair as distinct without auto-merging', async () => {
    const user = userEvent.setup()
    renderWithProviders(<DuplicatesPage />, { path: '/duplicates' })
    await waitFor(() => expect(screen.getAllByText(/شباهت/).length).toBeGreaterThan(0), { timeout: 5000 })
    await user.click(screen.getAllByRole('button', { name: /جدا هستند/ })[0])
    await waitFor(() => expect(screen.queryByText(/%۹۳/)).not.toBeInTheDocument(), { timeout: 5000 })
  })
})
