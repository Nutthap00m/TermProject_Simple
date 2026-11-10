import { RequireAuth } from './auth'
import Layout from './components/Layout'
import { Routes, Link } from './router'
import Cart from './pages/Cart'
import Home from './pages/Home'
import Login from './pages/Login'
import Orders from './pages/Orders'
import ProductDetail from './pages/ProductDetail'
import Profile from './pages/Profile'
import Register from './pages/Register'
import { AdminDashboard, AdminOrders, AdminProducts, AdminUsers } from './pages/Admin'

const admin = (el) => <RequireAuth role="admin">{el}</RequireAuth>
const member = (el) => <RequireAuth>{el}</RequireAuth>

const routes = [
  // Guest
  { path: '/', render: () => <Home /> },
  { path: '/products/:id', render: ({ id }) => <ProductDetail id={id} /> },
  { path: '/cart', render: () => <Cart /> },
  { path: '/login', render: () => <Login /> },
  { path: '/register', render: () => <Register /> },
  // Member
  { path: '/orders', render: () => member(<Orders />) },
  { path: '/profile', render: () => member(<Profile />) },
  // Admin
  { path: '/admin', render: () => admin(<AdminDashboard />) },
  { path: '/admin/products', render: () => admin(<AdminProducts />) },
  { path: '/admin/orders', render: () => admin(<AdminOrders />) },
  { path: '/admin/users', render: () => admin(<AdminUsers />) },
]

const NotFound = (
  <div className="status">
    ไม่พบหน้านี้ — <Link to="/">กลับหน้าแรก</Link>
  </div>
)

export default function App() {
  return (
    <Layout>
      <Routes routes={routes} fallback={NotFound} />
    </Layout>
  )
}
