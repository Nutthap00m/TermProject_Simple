import { useState } from 'react'
import { api } from '../api'
import { useAuth } from '../auth'
import { useCart } from '../cart'
import { Empty } from '../components/Status'
import { money } from '../format'
import { Link, useRouter } from '../router'

export default function Cart() {
  const { items, setQty, remove, clear, total } = useCart()
  const { user } = useAuth()
  const { navigate } = useRouter()
  const [address, setAddress] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  if (items.length === 0) {
    return (
      <Empty>
        ตะกร้าว่างเปล่า — <Link to="/">เลือกซื้อสินค้า</Link>
      </Empty>
    )
  }

  const checkout = async (e) => {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await api.createOrder({
        shipping_address: address,
        items: items.map((i) => ({ product_id: i.id, qty: i.qty })),
      })
      clear()
      navigate('/orders')
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <section>
      <h1>ตะกร้าสินค้า</h1>
      <table className="table">
        <thead>
          <tr>
            <th>สินค้า</th>
            <th>ราคา</th>
            <th>จำนวน</th>
            <th>รวม</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {items.map((i) => (
            <tr key={i.id}>
              <td>{i.title}</td>
              <td>{money(i.price)}</td>
              <td>
                <input
                  type="number"
                  min="1"
                  max="99"
                  value={i.qty}
                  aria-label={`จำนวน ${i.title}`}
                  onChange={(e) => setQty(i.id, Number(e.target.value) || 1)}
                />
              </td>
              <td>{money(i.price * i.qty)}</td>
              <td>
                <button type="button" className="link-btn" onClick={() => remove(i.id)}>
                  ลบ
                </button>
              </td>
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr>
            <td colSpan="3">รวมทั้งหมด</td>
            <td colSpan="2">
              <strong>{money(total)}</strong>
            </td>
          </tr>
        </tfoot>
      </table>

      {user ? (
        <form className="form" onSubmit={checkout}>
          <h2>ที่อยู่จัดส่ง</h2>
          <label>
            ที่อยู่
            <textarea value={address} onChange={(e) => setAddress(e.target.value)} rows="3" required maxLength={500} />
          </label>
          {error && (
            <p className="error-text" role="alert">
              {error}
            </p>
          )}
          <button type="submit" disabled={busy}>
            {busy ? 'กำลังสั่งซื้อ...' : 'ยืนยันสั่งซื้อ'}
          </button>
        </form>
      ) : (
        <div className="status">
          ต้องเป็นสมาชิกจึงจะสั่งซื้อได้ — <Link to="/login?next=/cart">เข้าสู่ระบบ</Link> หรือ{' '}
          <Link to="/register">สมัครสมาชิก</Link> (สินค้าในตะกร้าจะยังอยู่)
        </div>
      )}
    </section>
  )
}
