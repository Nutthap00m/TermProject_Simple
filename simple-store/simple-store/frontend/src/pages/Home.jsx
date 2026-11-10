import { useState } from 'react'
import { api } from '../api'
import ProductCard from '../components/ProductCard'
import { Empty, ErrorBox, Loading } from '../components/Status'
import { useFetch } from '../hooks'

export default function Home() {
  const [category, setCategory] = useState('')
  const [text, setText] = useState('')
  const [q, setQ] = useState('')

  const cats = useFetch(() => api.categories(), [])
  const products = useFetch(() => api.products({ category, q }), [category, q])

  const onSearch = (e) => {
    e.preventDefault()
    setQ(text.trim())
  }

  return (
    <section>
      <h1>สินค้าทั้งหมด</h1>

      <form className="toolbar" onSubmit={onSearch} role="search">
        <input
          type="search"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="ค้นหาสินค้า..."
          aria-label="ค้นหาสินค้า"
        />
        <button type="submit">ค้นหา</button>
      </form>

      <div className="chips" aria-label="หมวดหมู่">
        <button type="button" className={`chip ${category === '' ? 'active' : ''}`} onClick={() => setCategory('')}>
          ทั้งหมด
        </button>
        {(cats.data || []).map((c) => (
          <button
            key={c}
            type="button"
            className={`chip ${category === c ? 'active' : ''}`}
            onClick={() => setCategory(c)}
          >
            {c}
          </button>
        ))}
      </div>

      {products.loading && <Loading label="กำลังโหลดสินค้า..." />}
      {products.error && <ErrorBox error={products.error} onRetry={products.reload} />}
      {!products.loading && !products.error && products.data.length === 0 && (
        <Empty>ไม่พบสินค้า{q || category ? 'ที่ตรงกับเงื่อนไข' : ' ยังไม่มีสินค้าในร้าน'}</Empty>
      )}
      {!products.loading && !products.error && products.data.length > 0 && (
        <div className="grid">
          {products.data.map((p) => (
            <ProductCard key={p.id} product={p} />
          ))}
        </div>
      )}
    </section>
  )
}
