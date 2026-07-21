import { useEffect, useState } from 'react'
import { api } from '../api'
import { useAuth, useToast } from '../App'

interface Keys {
  public_key: string
  secret_key: string
  domains: string[]
  snippet: string
  key_snippet: string
}

export default function SettingsInstall() {
  const [keys, setKeys] = useState<Keys | null>(null)
  const [domainsText, setDomainsText] = useState('')
  const [revealed, setRevealed] = useState(false)
  const [showKeySnippet, setShowKeySnippet] = useState(false)
  const [busy, setBusy] = useState(false)
  const { account } = useAuth()
  const { toast } = useToast()

  useEffect(() => {
    api.get<Keys>('/api/apikeys').then((k) => {
      setKeys(k)
      setDomainsText((k.domains || []).join('\n'))
    }).catch(() => {})
  }, [])

  if (!keys || !account) return null

  const domainsSaved = (keys.domains || []).length > 0

  function copy(text: string, what: string) {
    navigator.clipboard.writeText(text).then(() => toast(`${what} copied`))
  }

  async function saveDomains() {
    setBusy(true)
    try {
      const res = await api.put<{ domains: string[] }>('/api/widget-domains', {
        domains: domainsText.split('\n').map((s) => s.trim()).filter(Boolean),
      })
      setKeys({ ...keys!, domains: res.domains })
      setDomainsText(res.domains.join('\n'))
      toast('Domains saved. The standard snippet is ready to use.')
    } catch (ex) {
      toast(ex instanceof Error ? ex.message : 'Save failed', true)
    } finally {
      setBusy(false)
    }
  }

  async function rotate() {
    if (!window.confirm('Rotate keys? Anything using the current widget key or secret API key will stop working.')) return
    const fresh = await api.post<{ public_key: string; secret_key: string }>('/api/apikeys/rotate')
    setKeys({
      ...keys!,
      public_key: fresh.public_key,
      secret_key: fresh.secret_key,
      key_snippet: keys!.key_snippet.replace(keys!.public_key, fresh.public_key),
    })
    toast('Keys rotated')
  }

  return (
    <>
      <div className="page-head">
        <div>
          <span className="eyebrow">Go live</span>
          <h1>Install &amp; API</h1>
          <p className="sub">Register your website, then paste one script tag. The standard setup keeps keys and credentials out of your page source.</p>
        </div>
      </div>

      <div className="grid cols-2">
        <div className="shell fade-up">
          <div className="card">
            <h3>1 · Your website domains</h3>
            <p className="card-sub">The domain tells us which widget settings to load. Registering example.com also covers its subdomains, including www.</p>
            <label className="field">
              <span>Domains (one per line)</span>
              <textarea
                rows={3}
                value={domainsText}
                onChange={(e) => setDomainsText(e.target.value)}
                placeholder={'example.com\nlocalhost'}
              />
            </label>
            <button className="btn" onClick={saveDomains} disabled={busy}>
              {busy ? 'Saving...' : 'Save domains'}
            </button>
          </div>
        </div>

        <div className="shell fade-up d1">
          <div className="card">
            <h3>2 · Website snippet</h3>
            <p className="card-sub">Paste this before the closing &lt;/body&gt; tag or into the custom HTML area in WordPress, Webflow, Squarespace, or Wix.</p>
            {!domainsSaved && (
              <p className="inline-note" style={{ marginTop: 0 }}>
                Save your domain first. Until then, use the keyed snippet below.
              </p>
            )}
            <div className="snippet">{keys.snippet}</div>
            <div className="action-row" style={{ marginTop: 14 }}>
              <button className="btn" onClick={() => copy(keys.snippet, 'Snippet')} disabled={!domainsSaved}>
                Copy snippet
              </button>
              <button className="btn subtle" onClick={() => setShowKeySnippet(!showKeySnippet)}>
                {showKeySnippet ? 'Hide' : 'Show'} keyed variant
              </button>
            </div>
            {showKeySnippet && (
              <>
                <p className="field-hint" style={{ margin: '14px 0 8px' }}>
                  Use this for pages we cannot match by domain, such as local files, staging hosts, or agency embeds. The widget key can start chats but cannot read conversations or settings.
                </p>
                <div className="snippet">{keys.key_snippet}</div>
                <button className="btn subtle" style={{ marginTop: 10 }} onClick={() => copy(keys.key_snippet, 'Keyed snippet')}>
                  Copy keyed snippet
                </button>
              </>
            )}
          </div>
        </div>

        <div className="shell fade-up d2">
          <div className="card">
            <h3>Keys</h3>
            <p className="card-sub">The widget key is a public identifier that can only start chats. The secret key unlocks the REST API and belongs on your server, never in a web page.</p>
            <label className="field">
              <span>Widget key, public identifier</span>
              <div className="key-row">
                <input type="text" readOnly value={keys.public_key} className="mono" />
                <button className="btn subtle" onClick={() => copy(keys.public_key, 'Widget key')}>Copy</button>
              </div>
            </label>
            <label className="field">
              <span>Secret API key, keep private {!account.plan_limits.api && <span className="badge handoff">Professional plan</span>}</span>
              <div className="key-row">
                <input type="text" readOnly value={revealed ? keys.secret_key : '••••••••••••••••••••••••'} className="mono" />
                <button className="btn subtle" onClick={() => setRevealed(!revealed)}>{revealed ? 'Hide' : 'Show'}</button>
                <button className="btn subtle" onClick={() => copy(keys.secret_key, 'Secret key')}>Copy</button>
              </div>
            </label>
            <button className="btn danger" onClick={rotate}>Rotate keys</button>
          </div>
        </div>

        <div className="shell fade-up d3">
          <div className="card">
            <h3>REST API</h3>
            <p className="card-sub">Developers and agencies can find the full reference in docs/API.md.</p>
            <div className="snippet">{`# Create a session
curl -X POST ${location.origin}/api/v1/sessions \\
  -H "Authorization: Bearer ${revealed ? keys.secret_key : 'sk_live_...'}" \\
  -H "Content-Type: application/json" \\
  -d '{"page_url": "https://example.com/pricing"}'

# Send a visitor message (waits for the AI reply)
curl -X POST ${location.origin}/api/v1/sessions/SESSION_TOKEN/messages \\
  -H "Authorization: Bearer ${revealed ? keys.secret_key : 'sk_live_...'}" \\
  -H "Content-Type: application/json" \\
  -d '{"content": "We need IT support for a 12-person office"}'

# List qualified conversations
curl "${location.origin}/api/v1/conversations?status=qualified" \\
  -H "Authorization: Bearer ${revealed ? keys.secret_key : 'sk_live_...'}"`}</div>
          </div>
        </div>
      </div>
    </>
  )
}
