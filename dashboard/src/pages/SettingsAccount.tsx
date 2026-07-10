import { useState } from 'react'
import { api } from '../api'
import { useAuth, useToast } from '../App'
import type { Account } from '../types'
import { Toggle } from '../components/ui'

const PLANS = [
  { id: 'starter', name: 'Starter', price: '$99/mo', blurb: '200 conversations · email & Slack routing' },
  { id: 'professional', name: 'Professional', price: '$199/mo', blurb: '500 conversations · webhooks/CRM · API · white-label' },
  { id: 'agency', name: 'Agency', price: '$399/mo', blurb: '10 client accounts · everything in Professional' },
  { id: 'enterprise', name: 'Enterprise', price: 'Custom', blurb: 'Unlimited · dedicated support · self-hosted option' },
]

export default function SettingsAccount() {
  const { account, refresh } = useAuth()
  const { toast } = useToast()
  const [weekly, setWeekly] = useState(account?.settings.weekly_report ?? true)
  const [plan, setPlan] = useState(account?.plan || 'starter')
  const [busy, setBusy] = useState(false)

  if (!account) return null

  async function save() {
    setBusy(true)
    try {
      await api.put<Account>('/api/settings', { settings: { weekly_report: weekly }, plan })
      await refresh()
      toast('Account updated')
    } catch (ex) {
      toast(ex instanceof Error ? ex.message : 'Save failed', true)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <div className="page-head">
        <div>
          <span className="eyebrow">Account</span>
          <h1>Plan &amp; preferences</h1>
          <p className="sub">Signed in as {account.email}</p>
        </div>
        <button className="btn" onClick={save} disabled={busy}>
          Save changes <span className="btn-orb">✓</span>
        </button>
      </div>

      <div className="shell fade-up">
        <div className="card">
          <h3>Plan</h3>
          <p className="card-sub">Billing isn't wired in this self-hosted build — switching plans changes limits immediately.</p>
          <div className="grid cols-4" style={{ gap: 12 }}>
            {PLANS.map((p) => (
              <button
                key={p.id}
                onClick={() => setPlan(p.id)}
                style={{
                  textAlign: 'left',
                  font: 'inherit',
                  cursor: 'pointer',
                  border: plan === p.id ? '2px solid var(--accent)' : '1px solid var(--hairline)',
                  background: plan === p.id ? 'var(--accent-soft)' : 'var(--surface)',
                  borderRadius: 16,
                  padding: 16,
                  transition: 'all 0.3s var(--ease)',
                }}
              >
                <div style={{ fontWeight: 800 }}>{p.name}</div>
                <div style={{ fontWeight: 700, color: 'var(--accent)', margin: '4px 0' }}>{p.price}</div>
                <div style={{ fontSize: 12, color: 'var(--ink-2)' }}>{p.blurb}</div>
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="shell fade-up d1" style={{ marginTop: 20 }}>
        <div className="card">
          <h3>Reports</h3>
          <Toggle checked={weekly} onChange={setWeekly} label="Weekly lead report" sub="Every week: conversations, qualified leads, and time saved — straight to your routing email." />
        </div>
      </div>

      <div className="shell fade-up d2" style={{ marginTop: 20 }}>
        <div className="card">
          <h3>Platform features</h3>
          <p className="card-sub">Enabled by your host via environment flags.</p>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {Object.entries(account.features).map(([k, v]) => (
              <span key={k} className={`badge ${v ? 'qualified' : 'abandoned'}`}>{k.replace(/_/g, ' ')}</span>
            ))}
          </div>
        </div>
      </div>
    </>
  )
}
