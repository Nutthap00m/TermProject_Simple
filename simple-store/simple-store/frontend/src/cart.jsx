import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'

const KEY = 'store.cart'
const MAX_QTY = 99

function load() {
  try {
    const v = JSON.parse(localStorage.getItem(KEY) || '[]')
    return Array.isArray(v) ? v : []
  } catch {
    return []
  }
}

const CartCtx = createContext(null)

export function CartProvider({ children }) {
  const [items, setItems] = useState(load)

  useEffect(() => {
    try {
      localStorage.setItem(KEY, JSON.stringify(items))
    } catch {
      // ignore
    }
  }, [items])

  const add = useCallback((product, qty = 1) => {
    setItems((cur) => {
      const found = cur.find((i) => i.id === product.id)
      if (found) {
        return cur.map((i) => (i.id === product.id ? { ...i, qty: Math.min(MAX_QTY, i.qty + qty) } : i))
      }
      const { id, title, price, image } = product
      return [...cur, { id, title, price, image, qty: Math.min(MAX_QTY, qty) }]
    })
  }, [])

  const setQty = useCallback((id, qty) => {
    setItems((cur) =>
      qty < 1 ? cur.filter((i) => i.id !== id) : cur.map((i) => (i.id === id ? { ...i, qty: Math.min(MAX_QTY, qty) } : i)),
    )
  }, [])

  const remove = useCallback((id) => setItems((cur) => cur.filter((i) => i.id !== id)), [])
  const clear = useCallback(() => setItems([]), [])

  const value = useMemo(
    () => ({
      items,
      add,
      setQty,
      remove,
      clear,
      count: items.reduce((n, i) => n + i.qty, 0),
      total: items.reduce((n, i) => n + i.price * i.qty, 0),
    }),
    [items, add, setQty, remove, clear],
  )

  return <CartCtx.Provider value={value}>{children}</CartCtx.Provider>
}

export const useCart = () => useContext(CartCtx)
