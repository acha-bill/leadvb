import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useAuth } from '../App'
import type { Account } from '../types'
import AuthShell from '../components/AuthShell'

export default function Signup() {
  const [name, setName] = useState('')
  const [company, setCompany] = useState('')
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
      const acct = await api.post<Account>('/api/auth/signup', { name, company, email, password })
      setAccount(acct)
      nav('/settings/icp')
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : 'Signup failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthShell>
      <span className="eyebrow">14 minutes to live</span>
      <h1>Create your account</h1>
      <p className="sub">Your AI assistant starts qualifying leads today.</p>
      <form onSubmit={submit}>
        <label className="field">
          <span>Your name</span>
          <input type="text" value={name} onChange={(e) => setName(e.target.value)} required autoFocus />
        </label>
        <label className="field">
          <span>Company</span>
          <input type="text" value={company} onChange={(e) => setCompany(e.target.value)} placeholder="Acme IT Services" />
        </label>
        <label className="field">
          <span>Email</span>
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        </label>
        <label className="field">
          <span>Password</span>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} />
          <div className="field-hint">At least 8 characters.</div>
        </label>
        {err && <p style={{ color: 'var(--bad)', fontSize: 13 }}>{err}</p>}
        <button className="btn" disabled={busy} style={{ width: '100%', justifyContent: 'center' }}>
          Create account <span className="btn-orb">→</span>
        </button>
      </form>
      <div className="auth-alt">
        Already have an account? <Link to="/login">Sign in</Link>
      </div>
    </AuthShell>
  )
}
