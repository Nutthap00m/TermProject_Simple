import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../api'
import Login from '../pages/Login'
import { renderApp } from './utils'

const mocks = vi.hoisted(() => ({ login: vi.fn() }))
vi.mock('../api', async (importOriginal) => {
  const actual = await importOriginal()
  return { ...actual, api: { ...actual.api, login: mocks.login } }
})

const fill = async (email, password) => {
  await userEvent.type(screen.getByLabelText('อีเมล'), email)
  await userEvent.type(screen.getByLabelText('รหัสผ่าน'), password)
  await userEvent.click(screen.getByRole('button', { name: 'เข้าสู่ระบบ' }))
}

describe('Login', () => {
  beforeEach(() => mocks.login.mockReset())

  it('sends the credentials, stores the token and redirects to ?next', async () => {
    mocks.login.mockResolvedValue({ token: 'jwt-123', user: { id: 'u1', name: 'Mali', email: 'm@x.com', role: 'member' } })
    renderApp(<Login />, { url: '/login?next=/cart' })

    await fill('m@x.com', 'password123')

    expect(mocks.login).toHaveBeenCalledWith({ email: 'm@x.com', password: 'password123' })
    await waitFor(() => expect(window.location.pathname).toBe('/cart'))
    expect(localStorage.getItem('store.token')).toBe('jwt-123')
  })

  it('does not redirect to an external site given in ?next (open redirect)', async () => {
    mocks.login.mockResolvedValue({ token: 't', user: { id: 'u1', name: 'A', email: 'a@x.com', role: 'member' } })
    renderApp(<Login />, { url: '/login?next=//evil.example' })

    await fill('a@x.com', 'password123')

    await waitFor(() => expect(window.location.pathname).toBe('/'))
  })

  it('shows the server error and keeps the user on the page when login fails', async () => {
    mocks.login.mockRejectedValue(new ApiError(401, 'invalid email or password'))
    renderApp(<Login />, { url: '/login' })

    await fill('m@x.com', 'wrong-password')

    expect(await screen.findByRole('alert')).toHaveTextContent('invalid email or password')
    expect(window.location.pathname).toBe('/login')
    expect(localStorage.getItem('store.token')).toBeNull()
    expect(screen.getByRole('button', { name: 'เข้าสู่ระบบ' })).toBeEnabled()
  })
})
