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
      <span className="eyebrow">Get started</span>
      <h1>Create your account</h1>
      <p className="sub">Set your lead criteria next, then add the assistant to your site.</p>
      <form onSubmit={submit}>
        <label className="field">
          <span>Your name</span>
          <input type="text" value={name} onChange={(e) => setName(e.target.value)} required autoFocus autoComplete="name" />
        </label>
        <label className="field">
          <span>Company</span>
          <input type="text" value={company} onChange={(e) => setCompany(e.target.value)} placeholder="Acme IT Services" autoComplete="organization" />
        </label>
        <label className="field">
          <span>Email</span>
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoComplete="email" />
        </label>
        <label className="field">
          <span>Password</span>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} autoComplete="new-password" />
          <div className="field-hint">At least 8 characters.</div>
        </label>
        {err && <p className="form-error" role="alert">{err}</p>}
        <button className="btn" disabled={busy} style={{ width: '100%', justifyContent: 'center' }}>
          {busy ? 'Creating account...' : 'Create account'}
        </button>
      </form>
      <div className="auth-alt">
        Already have an account? <Link to="/login">Sign in</Link>
      </div>
    </AuthShell>
  )
}
