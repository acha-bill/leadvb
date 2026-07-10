import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useAuth } from '../App'
import type { Account } from '../types'
import AuthShell from '../components/AuthShell'

export default function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)
  const { setAccount } = useAuth()
  const nav = useNavigate()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setErr('')
    try {
      const acct = await api.post<Account>('/api/auth/login', { email, password })
      setAccount(acct)
      nav('/')
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : 'Login failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthShell>
      <span className="eyebrow">Welcome back</span>
      <h1>Sign in</h1>
      <p className="sub">Your conversations and leads are waiting.</p>
      <form onSubmit={submit}>
        <label className="field">
          <span>Email</span>
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoFocus />
        </label>
        <label className="field">
          <span>Password</span>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        </label>
        {err && <p style={{ color: 'var(--bad)', fontSize: 13 }}>{err}</p>}
        <button className="btn" disabled={busy} style={{ width: '100%', justifyContent: 'center' }}>
          Sign in <span className="btn-orb">→</span>
        </button>
      </form>
      <div className="auth-alt">
        New here? <Link to="/signup">Create an account</Link>
      </div>
    </AuthShell>
  )
}
