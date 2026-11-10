const BASE = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/+$/, '')

export class ApiError extends Error {
  constructor(status, message) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

let token = null
let onUnauthorized = null

export const setToken = (t) => {
  token = t
}
export const setUnauthorizedHandler = (fn) => {
  onUnauthorized = fn
}

async function request(method, path, body) {
  const headers = { Accept: 'application/json' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (token) headers.Authorization = `Bearer ${token}`

  let res
  try {
    res = await fetch(`${BASE}${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new ApiError(0, 'เชื่อมต่อเซิร์ฟเวอร์ไม่ได้ กรุณาลองใหม่อีกครั้ง')
  }

  if (res.status === 204) return null

  let data = null
  try {
    data = await res.json()
  } catch {
    // ไม่ใช่ JSON
  }

  if (!res.ok) {
    // token หมดอายุ/ไม่ถูกต้อง → ให้ AuthProvider ออกจากระบบ
    if (res.status === 401 && token && onUnauthorized) onUnauthorized()
    throw new ApiError(res.status, data?.error || `เกิดข้อผิดพลาด (${res.status})`)
  }
  return data
}

const qs = (obj = {}) => {
  const p = new URLSearchParams()
  Object.entries(obj).forEach(([k, v]) => {
    if (v !== '' && v != null) p.set(k, v)
  })
  const s = p.toString()
  return s ? `?${s}` : ''
}

export const api = {
  // Guest
  products: (params) => request('GET', `/api/products${qs(params)}`),
  product: (id) => request('GET', `/api/products/${id}`),
  categories: () => request('GET', '/api/categories'),
  register: (body) => request('POST', '/api/auth/register', body),
  login: (body) => request('POST', '/api/auth/login', body),

  // Member
  me: () => request('GET', '/api/me'),
  updateMe: (body) => request('PUT', '/api/me', body),
  changePassword: (body) => request('POST', '/api/me/password', body),
  createOrder: (body) => request('POST', '/api/orders', body),
  myOrders: () => request('GET', '/api/orders'),

  // Admin
  admin: {
    stats: () => request('GET', '/api/admin/stats'),
    sync: () => request('POST', '/api/admin/sync'),
    createProduct: (body) => request('POST', '/api/admin/products', body),
    updateProduct: (id, body) => request('PUT', `/api/admin/products/${id}`, body),
    deleteProduct: (id) => request('DELETE', `/api/admin/products/${id}`),
    orders: () => request('GET', '/api/admin/orders'),
    updateOrder: (id, status) => request('PATCH', `/api/admin/orders/${id}`, { status }),
    users: () => request('GET', '/api/admin/users'),
  },
}
