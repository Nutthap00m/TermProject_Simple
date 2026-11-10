import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../api'
import { RequireAuth } from '../auth'
import Profile from '../pages/Profile'
import { renderApp } from './utils'

const mocks = vi.hoisted(() => ({ me: vi.fn(), updateMe: vi.fn(), changePassword: vi.fn() }))
vi.mock('../api', async (importOriginal) => {
  const actual = await importOriginal()
  return { ...actual, api: { ...actual.api, me: mocks.me, updateMe: mocks.updateMe, changePassword: mocks.changePassword } }
})

const open = async () => {
  localStorage.setItem('store.token', 'tok')
  mocks.me.mockResolvedValue({ id: 'u1', name: 'Mali', email: 'm@x.com', role: 'member' })
  renderApp(
    <RequireAuth>
      <Profile />
    </RequireAuth>,
    { url: '/profile' },
  )
  await screen.findByRole('heading', { name: 'โปรไฟล์' })
}

describe('Profile', () => {
  beforeEach(() => {
    mocks.me.mockReset()
    mocks.updateMe.mockReset()
    mocks.changePassword.mockReset()
  })

  it('updates the display name', async () => {
    await open()
    mocks.updateMe.mockResolvedValue({ id: 'u1', name: 'Mali K.', email: 'm@x.com', role: 'member' })

    const name = screen.getByLabelText('ชื่อ')
    await userEvent.clear(name)
    await userEvent.type(name, 'Mali K.')
    await userEvent.click(screen.getByRole('button', { name: 'บันทึก' }))

    expect(mocks.updateMe).toHaveBeenCalledWith({ name: 'Mali K.' })
    expect(await screen.findByText('บันทึกโปรไฟล์แล้ว')).toBeInTheDocument()
  })

  it('changes the password with current + new password', async () => {
    await open()
    mocks.changePassword.mockResolvedValue({ status: 'password updated' })

    await userEvent.type(screen.getByLabelText('รหัสผ่านปัจจุบัน'), 'old-password')
    await userEvent.type(screen.getByLabelText(/^รหัสผ่านใหม่/), 'new-password-1')
    await userEvent.type(screen.getByLabelText('ยืนยันรหัสผ่านใหม่'), 'new-password-1')
    await userEvent.click(screen.getByRole('button', { name: 'เปลี่ยนรหัสผ่าน' }))

    expect(mocks.changePassword).toHaveBeenCalledWith({ current_password: 'old-password', new_password: 'new-password-1' })
    expect(await screen.findByText('เปลี่ยนรหัสผ่านแล้ว')).toBeInTheDocument()
  })

  it('does not call the API when the confirmation does not match', async () => {
    await open()

    await userEvent.type(screen.getByLabelText('รหัสผ่านปัจจุบัน'), 'old-password')
    await userEvent.type(screen.getByLabelText(/^รหัสผ่านใหม่/), 'new-password-1')
    await userEvent.type(screen.getByLabelText('ยืนยันรหัสผ่านใหม่'), 'different-1')
    await userEvent.click(screen.getByRole('button', { name: 'เปลี่ยนรหัสผ่าน' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('ไม่ตรงกัน')
    expect(mocks.changePassword).not.toHaveBeenCalled()
  })

  it('shows the server error when the current password is wrong', async () => {
    await open()
    mocks.changePassword.mockRejectedValue(new ApiError(400, 'current password is incorrect'))

    await userEvent.type(screen.getByLabelText('รหัสผ่านปัจจุบัน'), 'nope-nope')
    await userEvent.type(screen.getByLabelText(/^รหัสผ่านใหม่/), 'new-password-1')
    await userEvent.type(screen.getByLabelText('ยืนยันรหัสผ่านใหม่'), 'new-password-1')
    await userEvent.click(screen.getByRole('button', { name: 'เปลี่ยนรหัสผ่าน' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('current password is incorrect')
  })
})
