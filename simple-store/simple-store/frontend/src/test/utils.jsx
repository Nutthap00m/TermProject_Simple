import { render } from '@testing-library/react'
import { AuthProvider } from '../auth'
import { CartProvider } from '../cart'
import { Router } from '../router'

// ครอบ Provider เหมือนใน main.jsx — ตั้ง URL เริ่มต้นได้
export function renderApp(ui, { url = '/' } = {}) {
  window.history.replaceState(null, '', url)
  return render(
    <Router>
      <AuthProvider>
        <CartProvider>{ui}</CartProvider>
      </AuthProvider>
    </Router>,
  )
}
