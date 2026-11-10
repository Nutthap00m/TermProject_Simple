import { useState } from 'react'
import { api } from '../api'
import { Empty, ErrorBox, Loading } from '../components/Status'
import { STATUS_LABEL, dateTime, money } from '../format'
import { useFetch } from '../hooks'
import { Link } from '../router'

const TABS = [
  ['/admin', 'ภาพรวม'],
  ['/admin/products', 'สินค้า'],
  ['/admin/orders', 'คำสั่งซื้อ'],
  ['/admin/users', 'สมาชิก'],
]

export function AdminShell({ active, children }) {
  return (
    <section>
      <h1>จัดการร้าน</h1>
      <nav className="tabs" aria-label="เมนูแอดมิน">
        {TABS.map(([to, label]) => (
          <Link key={to} to={to} className={active === to ? 'tab active' : 'tab'}>
            {label}
          </Link>
        ))}
      </nav>
      {children}
    </section>
  )
}

export function AdminDashboard() {
  const { data, loading, error, reload } = useFetch(() => api.admin.stats(), [])
  const [syncing, setSyncing] = useState(false)
  const [msg, setMsg] = useState({ type: '', text: '' })

  const sync = async () => {
    setSyncing(true)
    setMsg({ type: '', text: '' })
    try {
      const r = await api.admin.sync()
      setMsg({ type: 'ok', text: `นำเข้า/อัปเดตสินค้าจาก fakestoreapi.com แล้ว ${r.imported} รายการ` })
      reload()
    } catch (err) {
      setMsg({ type: 'error', text: err.message })
    } finally {
      setSyncing(false)
    }
  }

  return (
    <AdminShell active="/admin">
      {loading && <Loading />}
      {error && <ErrorBox error={error} onRetry={reload} />}
      {data && (
        <div className="stats">
          <div className="stat">
            <span>สินค้า</span>
            <strong>{data.products}</strong>
          </div>
          <div className="stat">
            <span>สมาชิก</span>
            <strong>{data.users}</strong>
          </div>
          <div className="stat">
            <span>คำสั่งซื้อ</span>
            <strong>{data.orders}</strong>
          </div>
          <div className="stat">
            <span>ยอดขาย (ไม่รวมที่ยกเลิก)</span>
            <strong>{money(data.revenue)}</strong>
          </div>
        </div>
      )}
      <div className="panel">
        <h2>นำเข้าสินค้า</h2>
        <p className="muted">ดึงสินค้าจาก fakestoreapi.com มาใส่ฐานข้อมูล (รันซ้ำได้ ไม่เกิดสินค้าซ้ำ)</p>
        <button type="button" onClick={sync} disabled={syncing}>
          {syncing ? 'กำลังนำเข้า...' : 'นำเข้าสินค้า'}
        </button>
        {msg.text && (
          <p className={msg.type === 'ok' ? 'ok' : 'error-text'} role={msg.type === 'ok' ? 'status' : 'alert'}>
            {msg.text}
          </p>
        )}
      </div>
    </AdminShell>
  )
}

const EMPTY_FORM = { title: '', price: '', category: '', image: '', description: '' }

function ProductForm({ initial, onSaved, onCancel }) {
  const [f, setF] = useState(initial ? { ...initial, price: String(initial.price) } : EMPTY_FORM)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const set = (k) => (e) => setF({ ...f, [k]: e.target.value })

  const submit = async (e) => {
    e.preventDefault()
    setBusy(true)
    setError('')
    const body = { title: f.title, price: Number(f.price), category: f.category, image: f.image, description: f.description }
    try {
      if (initial) await api.admin.updateProduct(initial.id, body)
      else await api.admin.createProduct(body)
      onSaved()
    } catch (err) {
      setError(err.message)
      setBusy(false)
    }
  }

  return (
    <form className="form panel" onSubmit={submit}>
      <h2>{initial ? `แก้ไขสินค้า #${initial.id}` : 'เพิ่มสินค้าใหม่'}</h2>
      <label>
        ชื่อสินค้า
        <input value={f.title} onChange={set('title')} required maxLength={200} />
      </label>
      <div className="row">
        <label>
          ราคา (USD)
          <input type="number" min="0" step="0.01" value={f.price} onChange={set('price')} required />
        </label>
        <label>
          หมวดหมู่
          <input value={f.category} onChange={set('category')} required maxLength={60} />
        </label>
      </div>
      <label>
        URL รูปภาพ
        <input type="url" value={f.image} onChange={set('image')} placeholder="https://..." />
      </label>
      <label>
        รายละเอียด
        <textarea value={f.description} onChange={set('description')} rows="3" maxLength={5000} />
      </label>
      {error && (
        <p className="error-text" role="alert">
          {error}
        </p>
      )}
      <div className="row">
        <button type="submit" disabled={busy}>
          {busy ? 'กำลังบันทึก...' : 'บันทึก'}
        </button>
        <button type="button" className="secondary" onClick={onCancel}>
          ยกเลิก
        </button>
      </div>
    </form>
  )
}

export function AdminProducts() {
  const { data, loading, error, reload } = useFetch(() => api.products({ limit: 200 }), [])
  const [editing, setEditing] = useState(null) // null | 'new' | product
  const [actionError, setActionError] = useState('')

  const remove = async (p) => {
    if (!window.confirm(`ลบสินค้า "${p.title}" ?`)) return
    setActionError('')
    try {
      await api.admin.deleteProduct(p.id)
      reload()
    } catch (err) {
      setActionError(err.message)
    }
  }

  const saved = () => {
    setEditing(null)
    reload()
  }

  return (
    <AdminShell active="/admin/products">
      {editing ? (
        <ProductForm initial={editing === 'new' ? null : editing} onSaved={saved} onCancel={() => setEditing(null)} />
      ) : (
        <button type="button" onClick={() => setEditing('new')}>
          + เพิ่มสินค้า
        </button>
      )}
      {actionError && (
        <p className="error-text" role="alert">
          {actionError}
        </p>
      )}
      {loading && <Loading />}
      {error && <ErrorBox error={error} onRetry={reload} />}
      {data && data.length === 0 && <Empty>ยังไม่มีสินค้า — กด "นำเข้าสินค้า" ที่หน้าภาพรวม หรือเพิ่มเอง</Empty>}
      {data && data.length > 0 && (
        <table className="table">
          <thead>
            <tr>
              <th>#</th>
              <th>ชื่อ</th>
              <th>หมวดหมู่</th>
              <th>ราคา</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {data.map((p) => (
              <tr key={p.id}>
                <td>{p.id}</td>
                <td>{p.title}</td>
                <td>{p.category}</td>
                <td>{money(p.price)}</td>
                <td className="actions">
                  <button type="button" className="link-btn" onClick={() => setEditing(p)}>
                    แก้ไข
                  </button>
                  <button type="button" className="link-btn danger" onClick={() => remove(p)}>
                    ลบ
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </AdminShell>
  )
}

export function AdminOrders() {
  const { data, loading, error, reload } = useFetch(() => api.admin.orders(), [])
  const [actionError, setActionError] = useState('')

  const changeStatus = async (id, status) => {
    setActionError('')
    try {
      await api.admin.updateOrder(id, status)
      reload()
    } catch (err) {
      setActionError(err.message)
    }
  }

  return (
    <AdminShell active="/admin/orders">
      {actionError && (
        <p className="error-text" role="alert">
          {actionError}
        </p>
      )}
      {loading && <Loading />}
      {error && <ErrorBox error={error} onRetry={reload} />}
      {data && data.length === 0 && <Empty>ยังไม่มีคำสั่งซื้อ</Empty>}
      {data && data.length > 0 && (
        <table className="table">
          <thead>
            <tr>
              <th>#</th>
              <th>ลูกค้า</th>
              <th>รายการ</th>
              <th>รวม</th>
              <th>วันที่</th>
              <th>สถานะ</th>
            </tr>
          </thead>
          <tbody>
            {data.map((o) => (
              <tr key={o.id}>
                <td>{o.id}</td>
                <td>{o.users ? `${o.users.name} (${o.users.email})` : o.user_id}</td>
                <td>{o.order_items.map((i) => `${i.title} ×${i.qty}`).join(', ')}</td>
                <td>{money(o.total)}</td>
                <td>{dateTime(o.created_at)}</td>
                <td>
                  <select
                    value={o.status}
                    aria-label={`สถานะคำสั่งซื้อ #${o.id}`}
                    onChange={(e) => changeStatus(o.id, e.target.value)}
                  >
                    {Object.entries(STATUS_LABEL).map(([v, label]) => (
                      <option key={v} value={v}>
                        {label}
                      </option>
                    ))}
                  </select>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </AdminShell>
  )
}

export function AdminUsers() {
  const { data, loading, error, reload } = useFetch(() => api.admin.users(), [])
  return (
    <AdminShell active="/admin/users">
      {loading && <Loading />}
      {error && <ErrorBox error={error} onRetry={reload} />}
      {data && data.length === 0 && <Empty>ยังไม่มีสมาชิก</Empty>}
      {data && data.length > 0 && (
        <table className="table">
          <thead>
            <tr>
              <th>ชื่อ</th>
              <th>อีเมล</th>
              <th>บทบาท</th>
              <th>สมัครเมื่อ</th>
            </tr>
          </thead>
          <tbody>
            {data.map((u) => (
              <tr key={u.id}>
                <td>{u.name}</td>
                <td>{u.email}</td>
                <td>{u.role}</td>
                <td>{dateTime(u.created_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </AdminShell>
  )
}
