import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'

const RouterCtx = createContext(null)

export function Router({ children }) {
  const [path, setPath] = useState(() => window.location.pathname + window.location.search)

  useEffect(() => {
    const onPop = () => setPath(window.location.pathname + window.location.search)
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [])

  const navigate = useCallback((to, { replace = false } = {}) => {
    window.history[replace ? 'replaceState' : 'pushState'](null, '', to)
    setPath(to)
    window.scrollTo?.(0, 0)
  }, [])

  const value = useMemo(() => {
    const [pathname, search = ''] = path.split('?')
    return { path, pathname, query: new URLSearchParams(search), navigate }
  }, [path, navigate])

  return <RouterCtx.Provider value={value}>{children}</RouterCtx.Provider>
}

export const useRouter = () => useContext(RouterCtx)

export function Link({ to, children, ...rest }) {
  const { navigate } = useRouter()
  const onClick = (e) => {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return
    e.preventDefault()
    navigate(to)
  }
  return (
    <a href={to} onClick={onClick} {...rest}>
      {children}
    </a>
  )
}

export function matchPath(pattern, pathname) {
  const pp = pattern.split('/').filter(Boolean)
  const ap = pathname.split('/').filter(Boolean)
  if (pp.length !== ap.length) return null
  const params = {}
  for (let i = 0; i < pp.length; i++) {
    if (pp[i].startsWith(':')) params[pp[i].slice(1)] = decodeURIComponent(ap[i])
    else if (pp[i] !== ap[i]) return null
  }
  return params
}

export function Routes({ routes, fallback = null }) {
  const { pathname } = useRouter()
  for (const r of routes) {
    const params = matchPath(r.path, pathname)
    if (params) return r.render(params)
  }
  return fallback
}

// ป้องกัน open redirect: อนุญาตเฉพาะ path ภายในเว็บ
export function safeNext(next) {
  return next && next.startsWith('/') && !next.startsWith('//') ? next : '/'
}
