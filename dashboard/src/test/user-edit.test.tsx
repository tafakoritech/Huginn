import { describe, it, expect } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from './test-utils'
import { UsersPage } from '@/pages/settings/UsersPage'

describe('user edit', () => {
  it('saves edited user fields through the dialog', async () => {
    const user = userEvent.setup()
    renderWithProviders(<UsersPage />, { path: '/settings/users' })

    await waitFor(() => expect(screen.getByText('علی محمدی')).toBeInTheDocument(), { timeout: 5000 })

    // Open the edit dialog for the first user (edit icon buttons carry the edit label).
    const editButtons = await screen.findAllByRole('button', { name: /ویرایش/ })
    await user.click(editButtons[0])

    // The dialog must be pre-filled with the target user's values.
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument(), { timeout: 5000 })
    const nameInput = screen.getByLabelText(/نام/)
    expect((nameInput as HTMLInputElement).value).not.toBe('')

    await user.clear(nameInput)
    await user.type(nameInput, 'علی رحیمی')

    await user.click(screen.getByRole('button', { name: /ذخیره/ }))

    await waitFor(() => expect(screen.getByText('علی رحیمی')).toBeInTheDocument(), { timeout: 5000 })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('saves even when the username field contains non-ASCII input', async () => {
    const user = userEvent.setup()
    renderWithProviders(<UsersPage />, { path: '/settings/users' })

    await waitFor(() => expect(screen.getByText('علی محمدی')).toBeInTheDocument(), { timeout: 5000 })
    const editButtons = await screen.findAllByRole('button', { name: /ویرایش/ })
    await user.click(editButtons[0])

    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument(), { timeout: 5000 })
    const usernameInput = screen.getByLabelText(/Username/)
    await user.type(usernameInput, 'ییییی')

    await user.click(screen.getByRole('button', { name: /ذخیره/ }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument(), { timeout: 5000 })
    // No validation banner should block the save.
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
