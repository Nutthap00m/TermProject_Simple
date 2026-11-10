import { useCart } from '../cart'
import { money } from '../format'
import { Link } from '../router'

export default function ProductCard({ product }) {
  const { add } = useCart()
  return (
    <article className="card">
      <Link to={`/products/${product.id}`} className="card-img">
        {product.image ? <img src={product.image} alt={product.title} loading="lazy" /> : <span>ไม่มีรูป</span>}
      </Link>
      <div className="card-body">
        <span className="chip small">{product.category}</span>
        <h3>
          <Link to={`/products/${product.id}`}>{product.title}</Link>
        </h3>
        <div className="card-foot">
          <strong>{money(product.price)}</strong>
          <button type="button" onClick={() => add(product)} aria-label={`เพิ่ม ${product.title} ลงตะกร้า`}>
            เพิ่มลงตะกร้า
          </button>
        </div>
      </div>
    </article>
  )
}
