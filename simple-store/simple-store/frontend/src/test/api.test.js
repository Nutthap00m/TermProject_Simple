import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api, setToken, setUnauthorizedHandler } from '../api'

const jsonResponse = (status, body) =>
  new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })

afterEach(() => {
  vi.unstubAllGlobals()
  setToken(null)
  setUnauthorizedHandler(null)
})

describe('api client (fetch)', () => {
  it('sends the Bearer token and JSON body', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(201, { id: 5 }))
    vi.stubGlobal('fetch', fetchMock)
    setToken('abc')

    await api.createOrder({ shipping_address: 'x', items: [{ product_id: 1, qty: 2 }] })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toMatch(/\/api\/orders$/)
    expect(init.method).toBe('POST')
    expect(init.headers.Authorization).toBe('Bearer abc')
    expect(JSON.parse(init.body)).toEqual({ shipping_address: 'x', items: [{ product_id: 1, qty: 2 }] })
  })

  it('does not send Authorization when logged out, and builds the query string', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, []))
    vi.stubGlobal('fetch', fetchMock)

    await api.products({ category: 'bags', q: '', limit: 10 })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toMatch(/\/api\/products\?category=bags&limit=10$/)
    expect(init.headers.Authorization).toBeUndefined()
  })

  it('throws ApiError carrying the server message and status', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(409, { error: 'email already registered' })))

    await expect(api.register({})).rejects.toMatchObject({
      name: 'ApiError',
      status: 409,
      message: 'email already registered',
    })
  })

  it('throws ApiError(0) when the network is down', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))

    const err = await api.products().catch((e) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err.status).toBe(0)
  })

  it('calls the unauthorized handler on 401 only when a token was sent', async () => {
    const handler = vi.fn()
    setUnauthorizedHandler(handler)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(401, { error: 'invalid or expired token' })))

    await api.me().catch(() => {}) // ไม่มี token (เช่น login ผิด) → ไม่ต้อง logout
    expect(handler).not.toHaveBeenCalled()

    setToken('stale')
    await api.me().catch(() => {})
    expect(handler).toHaveBeenCalledTimes(1)
  })

  it('returns null for 204 No Content', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))
    setToken('t')
    await expect(api.admin.deleteProduct(1)).resolves.toBeNull()
  })
})
