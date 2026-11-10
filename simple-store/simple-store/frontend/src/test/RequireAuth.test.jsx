import { screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { RequireAuth } from '../auth'
import { renderApp } from './utils'

const mocks = vi.hoisted(() => ({ me: vi.fn() }))
vi.mock('../api', async (importOriginal) => {
  const actual = await importOriginal()
  return { ...actual, api: { ...actual.api, me: mocks.me } }
})

const guarded = (role) => (
  <RequireAuth role={role}>
    <p>secret area</p>
  </RequireAuth>
)

describe('RequireAuth (route guard)', () => {
  beforeEach(() => mocks.me.mockReset())

  it('sends a guest to /login and remembers where they wanted to go', async () => {
    renderApp(guarded(), { url: '/orders' })

    await waitFor(() => expect(window.location.pathname).toBe('/login'))
    expect(window.location.search).toBe('?next=%2Forders')
    expect(screen.queryByText('secret area')).not.toBeInTheDocument()
  })

  it('lets a logged-in member see member pages', async () => {
    localStorage.setItem('store.token', 'tok')
    mocks.me.mockResolvedValue({ id: 'u1', name: 'Mali', email: 'm@x.com', role: 'member' })
    renderApp(guarded(), { url: '/orders' })

    expect(await screen.findByText('secret area')).toBeInTheDocument()
  })

  it('blocks a member from admin pages', async () => {
    localStorage.setItem('store.token', 'tok')
    mocks.me.mockResolvedValue({ id: 'u1', name: 'Mali', email: 'm@x.com', role: 'member' })
    renderApp(guarded('admin'), { url: '/admin' })

    expect(await screen.findByRole('alert')).toHaveTextContent('ไม่มีสิทธิ์')
    expect(screen.queryByText('secret area')).not.toBeInTheDocument()
  })

  it('lets an admin into admin pages', async () => {
    localStorage.setItem('store.token', 'tok')
    mocks.me.mockResolvedValue({ id: 'a1', name: 'Boss', email: 'a@x.com', role: 'admin' })
    renderApp(guarded('admin'), { url: '/admin' })

    expect(await screen.findByText('secret area')).toBeInTheDocument()
  })

  it('treats an invalid stored token as logged out', async () => {
    localStorage.setItem('store.token', 'expired')
    mocks.me.mockRejectedValue(new Error('401'))
    renderApp(guarded(), { url: '/profile' })

    await waitFor(() => expect(window.location.pathname).toBe('/login'))
    expect(localStorage.getItem('store.token')).toBeNull()
  })
})
