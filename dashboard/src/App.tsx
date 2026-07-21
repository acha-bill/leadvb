import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { api } from './api'
import type { Account } from './types'
import Layout from './components/Layout'
import Login from './pages/Login'
import Signup from './pages/Signup'
import Overview from './pages/Overview'
import Conversations from './pages/Conversations'
import ConversationDetail from './pages/ConversationDetail'
import SettingsICP from './pages/SettingsICP'
import SettingsRouting from './pages/SettingsRouting'
import SettingsWidget from './pages/SettingsWidget'
import SettingsInstall from './pages/SettingsInstall'
import SettingsAccount from './pages/SettingsAccount'

interface AuthCtx {
  account: Account | null
  refresh: () => Promise<void>
  setAccount: (a: Account | null) => void
}

const Ctx = createContext<AuthCtx>({ account: null, refresh: async () => {}, setAccount: () => {} })
export const useAuth = () => useContext(Ctx)

interface ToastCtx {
  toast: (msg: string, isError?: boolean) => void
}
const TCtx = createContext<ToastCtx>({ toast: () => {} })
export const useToast = () => useContext(TCtx)

export default function App() {
  const [account, setAccount] = useState<Account | null>(null)
  const [loading, setLoading] = useState(true)
  const [toastMsg, setToastMsg] = useState<{ msg: string; err: boolean } | null>(null)
  const location = useLocation()

  const refresh = useCallback(async () => {
    try {
      setAccount(await api.get<Account>('/api/auth/me'))
    } catch {
      setAccount(null)
    }
  }, [])

  useEffect(() => {
    refresh().finally(() => setLoading(false))
  }, [refresh])

  const toast = useCallback((msg: string, isError = false) => {
    setToastMsg({ msg, err: isError })
    window.setTimeout(() => setToastMsg(null), 3200)
  }, [])

  if (loading) {
    return <div className="app-loader" role="status"><span className="loader-mark" />Loading your dashboard</div>
  }

  const authed = account !== null
  const publicPage = location.pathname === '/login' || location.pathname === '/signup'

  return (
    <Ctx.Provider value={{ account, refresh, setAccount }}>
      <TCtx.Provider value={{ toast }}>
        {!authed && !publicPage && <Navigate to="/login" replace />}
        {authed && publicPage && <Navigate to="/" replace />}
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/signup" element={<Signup />} />
          {authed && (
            <Route element={<Layout />}>
              <Route path="/" element={<Overview />} />
              <Route path="/conversations" element={<Conversations />} />
              <Route path="/conversations/:id" element={<ConversationDetail />} />
              <Route path="/settings/icp" element={<SettingsICP />} />
              <Route path="/settings/routing" element={<SettingsRouting />} />
              <Route path="/settings/widget" element={<SettingsWidget />} />
              <Route path="/settings/install" element={<SettingsInstall />} />
              <Route path="/settings/account" element={<SettingsAccount />} />
            </Route>
          )}
          <Route path="*" element={<Navigate to={authed ? '/' : '/login'} replace />} />
        </Routes>
        {toastMsg && <div className={'toast' + (toastMsg.err ? ' err' : '')} role={toastMsg.err ? 'alert' : 'status'} aria-live="polite">{toastMsg.msg}</div>}
      </TCtx.Provider>
    </Ctx.Provider>
  )
}
