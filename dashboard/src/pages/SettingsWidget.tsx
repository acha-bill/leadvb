import { useEffect, useRef, useState } from 'react'
import { api } from '../api'
import { useAuth, useToast } from '../App'
import type { WidgetConfig } from '../types'
import { Toggle } from '../components/ui'

interface Resp {
  config: WidgetConfig
  public_key: string
  base_url: string
}

export default function SettingsWidget() {
  const [cfg, setCfg] = useState<WidgetConfig | null>(null)
  const [pk, setPk] = useState('')
  const [baseURL, setBaseURL] = useState('')
  const [busy, setBusy] = useState(false)
  const [previewNonce, setPreviewNonce] = useState(0)
  const { account } = useAuth()
  const { toast } = useToast()
  const fileRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    api.get<Resp>('/api/widget-config').then((r) => {
      setCfg(r.config)
      setPk(r.public_key)
      setBaseURL(r.base_url)
    }).catch(() => {})
  }, [])

  if (!cfg || !account) return null

  async function save() {
    if (!cfg) return
    setBusy(true)
    try {
      setCfg(await api.put<WidgetConfig>('/api/widget-config', cfg))
      toast('Widget saved')
      setPreviewNonce((n) => n + 1)
    } catch (ex) {
      toast(ex instanceof Error ? ex.message : 'Save failed', true)
    } finally {
      setBusy(false)
    }
  }

  function onLogo(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0]
    if (!f || !cfg) return
    if (f.size > 200 * 1024) {
      toast('Logo must be under 200KB', true)
      return
    }
    const reader = new FileReader()
    reader.onload = () => setCfg({ ...cfg, logo_url: String(reader.result) })
    reader.readAsDataURL(f)
  }

  const whiteLabel = account.plan_limits.white_label
  const feats = account.features

  return (
    <>
      <div className="page-head">
        <div>
          <span className="eyebrow">Appearance &amp; triggers</span>
          <h1>Widget design</h1>
          <p className="sub">Brand the chat bubble and choose when it speaks up.</p>
        </div>
        <button className="btn" onClick={save} disabled={busy}>
          Save widget <span className="btn-orb">✓</span>
        </button>
      </div>

      <div className="grid cols-3">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }} className="span-2">
          <div className="shell fade-up">
            <div className="card">
              <h3>Branding</h3>
              <div className="grid cols-2" style={{ gap: 14 }}>
                <label className="field">
                  <span>Company name</span>
                  <input type="text" value={cfg.company_name} onChange={(e) => setCfg({ ...cfg, company_name: e.target.value })} />
                </label>
                <label className="field">
                  <span>Brand color</span>
                  <div style={{ display: 'flex', gap: 10 }}>
                    <input type="color" value={cfg.primary_color} onChange={(e) => setCfg({ ...cfg, primary_color: e.target.value })} style={{ width: 48, height: 42, padding: 4, border: '1px solid var(--hairline)', borderRadius: 12, background: 'var(--surface)' }} />
                    <input type="text" value={cfg.primary_color} onChange={(e) => setCfg({ ...cfg, primary_color: e.target.value })} />
                  </div>
                </label>
                <label className="field">
                  <span>Button position</span>
                  <select value={cfg.position} onChange={(e) => setCfg({ ...cfg, position: e.target.value })}>
                    <option value="right">Bottom right</option>
                    <option value="left">Bottom left</option>
                  </select>
                </label>
                <label className="field">
                  <span>Logo (max 200KB)</span>
                  <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
                    {cfg.logo_url && <img src={cfg.logo_url} alt="logo" style={{ width: 38, height: 38, borderRadius: 10, objectFit: 'cover' }} />}
                    <button type="button" className="btn subtle" onClick={() => fileRef.current?.click()}>Upload</button>
                    {cfg.logo_url && (
                      <button type="button" className="btn subtle" onClick={() => setCfg({ ...cfg, logo_url: '' })}>Remove</button>
                    )}
                    <input ref={fileRef} type="file" accept="image/*" style={{ display: 'none' }} onChange={onLogo} />
                  </div>
                </label>
              </div>
              <label className="field">
                <span>Welcome message</span>
                <input type="text" value={cfg.greeting} onChange={(e) => setCfg({ ...cfg, greeting: e.target.value })} />
              </label>
              <label className="field">
                <span>Conversation language</span>
                <select value={cfg.language} onChange={(e) => setCfg({ ...cfg, language: e.target.value })}>
                  <option value="auto">Auto — match the visitor</option>
                  <option value="en">English</option>
                  <option value="fr">Français</option>
                  <option value="es">Español</option>
                </select>
              </label>
              <Toggle checked={cfg.quick_replies} onChange={(v) => setCfg({ ...cfg, quick_replies: v })} label="Quick reply buttons" sub="One-tap answers for budget ranges, yes/no questions, etc." />
              <Toggle
                checked={cfg.branding && !whiteLabel ? true : cfg.branding}
                onChange={(v) => setCfg({ ...cfg, branding: whiteLabel ? v : true })}
                label='Show "Powered by Lead Qualifier"'
                sub={whiteLabel ? 'You can remove branding on your plan' : 'Upgrade to Professional to remove'}
              />
            </div>
          </div>

          <div className="shell fade-up d1">
            <div className="card">
              <h3>Proactive triggers</h3>
              <p className="card-sub">Don't wait for clicks — start the conversation.</p>
              {feats.proactive_triggers && (
                <>
                  <Toggle checked={cfg.proactive.enabled} onChange={(v) => setCfg({ ...cfg, proactive: { ...cfg.proactive, enabled: v } })} label="Open automatically after a delay" />
                  {cfg.proactive.enabled && (
                    <div className="grid cols-2" style={{ gap: 14 }}>
                      <label className="field">
                        <span>Delay (seconds)</span>
                        <input type="number" min={2} max={120} value={cfg.proactive.delay_seconds} onChange={(e) => setCfg({ ...cfg, proactive: { ...cfg.proactive, delay_seconds: Number(e.target.value) || 8 } })} />
                      </label>
                      <label className="field">
                        <span>Opening line</span>
                        <input type="text" value={cfg.proactive.message} onChange={(e) => setCfg({ ...cfg, proactive: { ...cfg.proactive, message: e.target.value } })} />
                      </label>
                    </div>
                  )}
                </>
              )}
              {feats.exit_intent && (
                <Toggle checked={cfg.exit_intent} onChange={(v) => setCfg({ ...cfg, exit_intent: v })} label="Exit intent" sub="Open when the cursor heads for the close button (desktop)" />
              )}
              <label className="field" style={{ marginTop: 10 }}>
                <span>Show widget on</span>
                <select value={cfg.pages.mode} onChange={(e) => setCfg({ ...cfg, pages: { ...cfg.pages, mode: e.target.value } })}>
                  <option value="all">Every page</option>
                  <option value="include">Only these pages</option>
                  <option value="exclude">Every page except these</option>
                </select>
              </label>
              {cfg.pages.mode !== 'all' && (
                <label className="field">
                  <span>URL patterns (one per line, * wildcard ok)</span>
                  <textarea
                    rows={3}
                    value={(cfg.pages.patterns || []).join('\n')}
                    onChange={(e) => setCfg({ ...cfg, pages: { ...cfg.pages, patterns: e.target.value.split('\n').map((s) => s.trim()).filter(Boolean) } })}
                    placeholder={'/pricing\n/contact\n/services/*'}
                  />
                </label>
              )}
            </div>
          </div>
        </div>

        <div className="shell fade-up d2">
          <div className="card" style={{ padding: 10 }}>
            <h3 style={{ padding: '10px 10px 0' }}>Live preview</h3>
            <p className="card-sub" style={{ padding: '0 10px' }}>Saved settings, real conversations.</p>
            <iframe
              key={previewNonce}
              title="Widget preview"
              src={`${baseURL}/widget/demo?key=${pk}`}
              style={{ width: '100%', height: 480, border: 'none', borderRadius: 12, background: '#f4f4f7' }}
            />
          </div>
        </div>
      </div>
    </>
  )
}
