import { api } from '../api'
import { Empty, ErrorBox, Loading } from '../components/Status'
import { STATUS_LABEL, dateTime, money } from '../format'
import { useFetch } from '../hooks'
import { Link } from '../router'

export default function Orders() {
  const { data, loading, error, reload } = useFetch(() => api.myOrders(), [])

  return (
    <section>
      <h1>คำสั่งซื้อของฉัน</h1>
      {loading && <Loading />}
      {error && <ErrorBox error={error} onRetry={reload} />}
      {data && data.length === 0 && (
        <Empty>
          ยังไม่มีคำสั่งซื้อ — <Link to="/">เริ่มเลือกซื้อสินค้า</Link>
        </Empty>
      )}
      {data &&
        data.map((o) => (
          <div className="panel" key={o.id}>
            <div className="panel-head">
              <strong>คำสั่งซื้อ #{o.id}</strong>
              <span className={`pill ${o.status}`}>{STATUS_LABEL[o.status] || o.status}</span>
              <span className="muted">{dateTime(o.created_at)}</span>
            </div>
            <ul>
              {o.order_items.map((i) => (
                <li key={i.id}>
                  {i.title} × {i.qty} — {money(i.price * i.qty)}
                </li>
              ))}
            </ul>
            <p>
              <strong>รวม {money(o.total)}</strong> · จัดส่ง: {o.shipping_address}
            </p>
          </div>
        ))}
    </section>
  )
}
