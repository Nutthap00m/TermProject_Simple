import { useState } from 'react'
import { api } from '../api'
import { useCart } from '../cart'
import { ErrorBox, Loading } from '../components/Status'
import { money } from '../format'
import { useFetch } from '../hooks'
import { Link } from '../router'

export default function ProductDetail({ id }) {
  const { data: p, loading, error, reload } = useFetch(() => api.product(id), [id])
  const { add } = useCart()
  const [qty, setQty] = useState(1)
  const [added, setAdded] = useState(false)

  if (loading) return <Loading />
  if (error) return <ErrorBox error={error} onRetry={error.status === 404 ? undefined : reload} />

  const onAdd = () => {
    add(p, qty)
    setAdded(true)
  }

  return (
    <article className="detail">
      <Link to="/">← กลับไปหน้าสินค้า</Link>
      <div className="detail-grid">
        <div className="detail-img">{p.image && <img src={p.image} alt={p.title} />}</div>
        <div>
          <span className="chip small">{p.category}</span>
          <h1>{p.title}</h1>
          <p className="price">{money(p.price)}</p>
          {p.rating_count > 0 && (
            <p className="muted">
              ★ {Number(p.rating_rate).toFixed(1)} ({p.rating_count} รีวิว)
            </p>
          )}
          <p>{p.description}</p>
          <div className="row">
            <label>
              จำนวน
              <input
                type="number"
                min="1"
                max="99"
                value={qty}
                onChange={(e) => setQty(Math.max(1, Math.min(99, Number(e.target.value) || 1)))}
              />
            </label>
            <button type="button" onClick={onAdd}>
              เพิ่มลงตะกร้า
            </button>
          </div>
          {added && (
            <p className="ok" role="status">
              เพิ่มลงตะกร้าแล้ว — <Link to="/cart">ไปที่ตะกร้า</Link>
            </p>
          )}
        </div>
      </div>
    </article>
  )
}
