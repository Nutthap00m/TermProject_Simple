import { useState } from 'react'
import { useAuth } from '../auth'

export default function Profile() {
  const { user, updateProfile, changePassword } = useAuth()

  const [name, setName] = useState(user.name)
  const [profileMsg, setProfileMsg] = useState({ type: '', text: '' })
  const [savingProfile, setSavingProfile] = useState(false)

  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [pwMsg, setPwMsg] = useState({ type: '', text: '' })
  const [savingPw, setSavingPw] = useState(false)

  const saveProfile = async (e) => {
    e.preventDefault()
    setSavingProfile(true)
    setProfileMsg({ type: '', text: '' })
    try {
      await updateProfile(name)
      setProfileMsg({ type: 'ok', text: 'บันทึกโปรไฟล์แล้ว' })
    } catch (err) {
      setProfileMsg({ type: 'error', text: err.message })
    } finally {
      setSavingProfile(false)
    }
  }

  const savePassword = async (e) => {
    e.preventDefault()
    if (next !== confirm) {
      setPwMsg({ type: 'error', text: 'รหัสผ่านใหม่และการยืนยันไม่ตรงกัน' })
      return
    }
    setSavingPw(true)
    setPwMsg({ type: '', text: '' })
    try {
      await changePassword(current, next)
      setCurrent('')
      setNext('')
      setConfirm('')
      setPwMsg({ type: 'ok', text: 'เปลี่ยนรหัสผ่านแล้ว' })
    } catch (err) {
      setPwMsg({ type: 'error', text: err.message })
    } finally {
      setSavingPw(false)
    }
  }

  return (
    <section className="two-col">
      <form className="form" onSubmit={saveProfile}>
        <h1>โปรไฟล์</h1>
        <label>
          อีเมล
          <input value={user.email} disabled />
        </label>
        <label>
          ชื่อ
          <input value={name} onChange={(e) => setName(e.target.value)} required maxLength={100} />
        </label>
        {profileMsg.text && (
          <p className={profileMsg.type === 'ok' ? 'ok' : 'error-text'} role={profileMsg.type === 'ok' ? 'status' : 'alert'}>
            {profileMsg.text}
          </p>
        )}
        <button type="submit" disabled={savingProfile}>
          {savingProfile ? 'กำลังบันทึก...' : 'บันทึก'}
        </button>
      </form>

      <form className="form" onSubmit={savePassword}>
        <h2>เปลี่ยนรหัสผ่าน</h2>
        <label>
          รหัสผ่านปัจจุบัน
          <input
            type="password"
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
            required
            autoComplete="current-password"
          />
        </label>
        <label>
          รหัสผ่านใหม่ (อย่างน้อย 8 ตัวอักษร)
          <input
            type="password"
            value={next}
            onChange={(e) => setNext(e.target.value)}
            required
            minLength={8}
            autoComplete="new-password"
          />
        </label>
        <label>
          ยืนยันรหัสผ่านใหม่
          <input
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            required
            autoComplete="new-password"
          />
        </label>
        {pwMsg.text && (
          <p className={pwMsg.type === 'ok' ? 'ok' : 'error-text'} role={pwMsg.type === 'ok' ? 'status' : 'alert'}>
            {pwMsg.text}
          </p>
        )}
        <button type="submit" disabled={savingPw}>
          {savingPw ? 'กำลังบันทึก...' : 'เปลี่ยนรหัสผ่าน'}
        </button>
      </form>
    </section>
  )
}
