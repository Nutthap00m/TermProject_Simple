import { useAuth } from '../auth'
import { useCart } from '../cart'
import { Link, useRouter } from '../router'

export default function Layout({ children }) {
  const { user, isAdmin, logout } = useAuth()
  const { count } = useCart()
  const { navigate } = useRouter()

  const onLogout = () => {
    logout()
    navigate('/')
  }

  return (
    <>
      <header className="site-header">
        <Link to="/" className="brand">
          Simple Store
        </Link>
        <nav aria-label="เมนูหลัก">
          <Link to="/">สินค้า</Link>
          <Link to="/cart">ตะกร้า{count > 0 && <span className="badge">{count}</span>}</Link>
          {user ? (
            <>
              <Link to="/orders">คำสั่งซื้อ</Link>
              <Link to="/profile">โปรไฟล์</Link>
              {isAdmin && <Link to="/admin">จัดการร้าน</Link>}
              <span className="who">สวัสดี {user.name}</span>
              <button type="button" className="link-btn" onClick={onLogout}>
                ออกจากระบบ
              </button>
            </>
          ) : (
            <>
              <Link to="/login">เข้าสู่ระบบ</Link>
              <Link to="/register" className="cta">
                สมัครสมาชิก
              </Link>
            </>
          )}
        </nav>
      </header>
      <main className="container">{children}</main>
      <footer className="site-footer">ข้อมูลสินค้าตัวอย่างจาก fakestoreapi.com</footer>
    </>
  )
}
