import { useEffect, useState } from 'react'
import { api } from '../api'
import { useAuth, useToast } from '../App'
import type { RoutingConfig } from '../types'
import { Toggle } from '../components/ui'

export default function SettingsRouting() {
  const [cfg, setCfg] = useState<RoutingConfig | null>(null)
  const [busy, setBusy] = useState(false)
  const [testing, setTesting] = useState('')
  const { account } = useAuth()
  const { toast } = useToast()

  useEffect(() => {
    api.get<RoutingConfig>('/api/routing').then(setCfg).catch(() => {})
  }, [])

  if (!cfg || !account) return null

  async function save() {
    if (!cfg) return
    setBusy(true)
    try {
      setCfg(await api.put<RoutingConfig>('/api/routing', cfg))
      toast('Routing saved')
    } catch (ex) {
      toast(ex instanceof Error ? ex.message : 'Save failed', true)
    } finally {
      setBusy(false)
    }
  }

  async function test(channel: string) {
    setTesting(channel)
    try {
      await api.post('/api/routing/test', { channel })
      toast(`Test ${channel} sent — check the destination`)
    } catch (ex) {
      toast(ex instanceof Error ? ex.message : 'Test failed', true)
    } finally {
      setTesting('')
    }
  }

  const webhookAllowed = account.plan_limits.webhooks

  return (
    <>
      <div className="page-head">
        <div>
          <span className="eyebrow">Delivery</span>
          <h1>Lead routing</h1>
          <p className="sub">Where qualified leads (and human-handoff alerts) get sent, with the full transcript attached.</p>
        </div>
        <button className="btn" onClick={save} disabled={busy}>
          Save routing <span className="btn-orb">✓</span>
        </button>
      </div>

      <div className="grid cols-2">
        <div className="shell fade-up">
          <div className="card">
            <h3>Email</h3>
            <p className="card-sub">Lead summary + transcript straight to your inbox.</p>
            <Toggle checked={cfg.email_enabled} onChange={(v) => setCfg({ ...cfg, email_enabled: v })} label="Send qualified leads by email" />
            <label className="field">
              <span>Destination email</span>
              <input type="email" value={cfg.email_to} onChange={(e) => setCfg({ ...cfg, email_to: e.target.value })} placeholder="you@company.com" />
            </label>
            <button className="btn ghost" onClick={() => test('email')} disabled={testing !== ''}>
              {testing === 'email' ? 'Sending…' : 'Send test email'}
            </button>
          </div>
        </div>

        <div className="shell fade-up d1">
          <div className="card">
            <h3>Slack</h3>
            <p className="card-sub">Post each qualified lead into a channel.</p>
            <Toggle checked={cfg.slack_enabled} onChange={(v) => setCfg({ ...cfg, slack_enabled: v })} label="Send qualified leads to Slack" />
            <label className="field">
              <span>Incoming webhook URL</span>
              <input type="url" value={cfg.slack_webhook_url} onChange={(e) => setCfg({ ...cfg, slack_webhook_url: e.target.value })} placeholder="https://hooks.slack.com/services/…" />
              <div className="field-hint">Slack → Apps → Incoming Webhooks → Add to channel.</div>
            </label>
            <button className="btn ghost" onClick={() => test('slack')} disabled={testing !== ''}>
              {testing === 'slack' ? 'Sending…' : 'Send test message'}
            </button>
          </div>
        </div>

        <div className="shell fade-up d2">
          <div className="card">
            <h3>Webhook / CRM {!webhookAllowed && <span className="badge handoff">Professional plan</span>}</h3>
            <p className="card-sub">POST every qualified lead as JSON to HubSpot, Pipedrive, Zapier or your own endpoint.</p>
            <Toggle
              checked={cfg.webhook_enabled}
              onChange={(v) => setCfg({ ...cfg, webhook_enabled: v && webhookAllowed })}
              label="Send qualified leads to a webhook"
              sub={webhookAllowed ? undefined : 'Upgrade to Professional to enable'}
            />
            <label className="field">
              <span>Webhook URL</span>
              <input type="url" value={cfg.webhook_url} onChange={(e) => setCfg({ ...cfg, webhook_url: e.target.value })} placeholder="https://your-crm.com/hooks/leads" />
            </label>
            <label className="field">
              <span>Signing secret (optional)</span>
              <input type="text" value={cfg.webhook_secret} onChange={(e) => setCfg({ ...cfg, webhook_secret: e.target.value })} placeholder="whsec_…" />
              <div className="field-hint">We sign payloads with HMAC-SHA256 in the X-LeadQualifier-Signature header.</div>
            </label>
            <button className="btn ghost" onClick={() => test('webhook')} disabled={testing !== '' || !webhookAllowed}>
              {testing === 'webhook' ? 'Sending…' : 'Send test payload'}
            </button>
          </div>
        </div>

        <div className="shell fade-up d3">
          <div className="card">
            <h3>Behaviour</h3>
            <p className="card-sub">What happens around the edges.</p>
            <Toggle checked={cfg.notify_handoff} onChange={(v) => setCfg({ ...cfg, notify_handoff: v })} label="Alert me when a visitor asks for a human" sub="Uses the channels enabled above" />
            <label className="field" style={{ marginTop: 10 }}>
              <span>Unqualified visitors</span>
              <select
                value={cfg.disqualified.mode}
                onChange={(e) => setCfg({ ...cfg, disqualified: { ...cfg.disqualified, mode: e.target.value } })}
              >
                <option value="polite">Politely wrap up</option>
                <option value="newsletter">Wrap up + invite to newsletter</option>
              </select>
            </label>
            {cfg.disqualified.mode === 'newsletter' && (
              <label className="field">
                <span>Newsletter link</span>
                <input type="url" value={cfg.disqualified.newsletter_url} onChange={(e) => setCfg({ ...cfg, disqualified: { ...cfg.disqualified, newsletter_url: e.target.value } })} placeholder="https://yoursite.com/newsletter" />
              </label>
            )}
          </div>
        </div>
      </div>
    </>
  )
}
