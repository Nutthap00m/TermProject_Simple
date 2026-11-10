import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { api, setToken, setUnauthorizedHandler } from './api'
import { useRouter } from './router'
import { Loading } from './components/Status'

const KEY = 'store.token'

function readToken() {
  try {
    return localStorage.getItem(KEY)
  } catch {
    return null
  }
}

function writeToken(t) {
  try {
    if (t) localStorage.setItem(KEY, t)
    else localStorage.removeItem(KEY)
  } catch {
    // เบราว์เซอร์ปิดการเก็บข้อมูล — ใช้งานต่อได้ แต่จะไม่จำการ login
  }
}

const AuthCtx = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [ready, setReady] = useState(() => !readToken()) // false = กำลังตรวจ token เดิม

  const clear = useCallback(() => {
    setToken(null)
    writeToken(null)
    setUser(null)
  }, [])

  const persist = useCallback((token, u) => {
    setToken(token)
    writeToken(token)
    setUser(u)
  }, [])

  useEffect(() => {
    setUnauthorizedHandler(clear)
    const stored = readToken()
    if (!stored) return
    setToken(stored)
    let alive = true
    api
      .me()
      .then((u) => alive && setUser(u))
      .catch(() => alive && clear())
      .finally(() => alive && setReady(true))
    return () => {
      alive = false
    }
  }, [clear])

  const value = useMemo(
    () => ({
      user,
      ready,
      isAdmin: user?.role === 'admin',
      async login(email, password) {
        const r = await api.login({ email, password })
        persist(r.token, r.user)
        return r.user
      },
      async register(name, email, password) {
        const r = await api.register({ name, email, password })
        persist(r.token, r.user)
        return r.user
      },
      logout: clear,
      async updateProfile(name) {
        setUser(await api.updateMe({ name }))
      },
      changePassword: (current_password, new_password) =>
        api.changePassword({ current_password, new_password }),
    }),
    [user, ready, persist, clear],
  )

  return <AuthCtx.Provider value={value}>{children}</AuthCtx.Provider>
}

export const useAuth = () => useContext(AuthCtx)

// ใช้ครอบหน้าที่ต้อง login (และ role ถ้าระบุ) — ฝั่ง Backend ตรวจสิทธิ์ซ้ำทุก request
export function RequireAuth({ role, children }) {
  const { user, ready } = useAuth()
  const { path, navigate } = useRouter()

  useEffect(() => {
    if (ready && !user) navigate(`/login?next=${encodeURIComponent(path)}`, { replace: true })
  }, [ready, user, path, navigate])

  if (!ready) return <Loading label="กำลังตรวจสอบสิทธิ์..." />
  if (!user) return null
  if (role && user.role !== role) {
    return (
      <div className="status error" role="alert">
        คุณไม่มีสิทธิ์เข้าถึงหน้านี้
      </div>
    )
  }
  return children
}
