import { useState } from 'react'
import { api } from '../api'
import { useAuth, useToast } from '../App'
import type { Account } from '../types'
import { Toggle } from '../components/ui'

const PLANS = [
  { id: 'starter', name: 'Starter', price: '$99/mo', blurb: '200 conversations with email and Slack routing' },
  { id: 'professional', name: 'Professional', price: '$199/mo', blurb: '500 conversations, CRM webhooks, API access, and white label' },
  { id: 'agency', name: 'Agency', price: '$399/mo', blurb: '10 client accounts with Professional features' },
  { id: 'enterprise', name: 'Enterprise', price: 'Custom', blurb: 'Unlimited conversations, dedicated support, and self-hosting' },
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
          {busy ? 'Saving...' : 'Save changes'}
        </button>
      </div>

      <div className="shell fade-up">
        <div className="card">
          <h3>Plan</h3>
          <p className="card-sub">This self-hosted build does not include billing. Changing the plan updates account limits immediately.</p>
          <div className="plan-options" role="radiogroup" aria-label="Account plan">
            {PLANS.map((p) => (
              <button
                key={p.id}
                onClick={() => setPlan(p.id)}
                className={`plan-option${plan === p.id ? ' selected' : ''}`}
                role="radio"
                aria-checked={plan === p.id}
              >
                <div className="plan-option-name">{p.name}</div>
                <div className="plan-option-price">{p.price}</div>
                <div className="plan-option-copy">{p.blurb}</div>
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="shell fade-up d1" style={{ marginTop: 20 }}>
        <div className="card">
          <h3>Reports</h3>
          <Toggle checked={weekly} onChange={setWeekly} label="Weekly lead report" sub="Send conversation totals, qualified leads, and estimated time saved to your routing email." />
        </div>
      </div>

      <div className="shell fade-up d2" style={{ marginTop: 20 }}>
        <div className="card">
          <h3>Platform features</h3>
          <p className="card-sub">Your host controls these features through environment settings.</p>
          <div className="feature-badges">
            {Object.entries(account.features).map(([k, v]) => (
              <span key={k} className={`badge ${v ? 'qualified' : 'abandoned'}`}>{k.replace(/_/g, ' ')}</span>
            ))}
          </div>
        </div>
      </div>
    </>
  )
}
