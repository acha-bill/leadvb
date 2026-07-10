import { useEffect, useState } from 'react'
import { api } from '../api'
import { useToast } from '../App'
import type { ICP, Weights } from '../types'

const EXAMPLE = `We only work with B2B service businesses in Canada or the US that have 5+ employees. The person chatting must be an owner or decision maker. Minimum budget is $2,000/month. They should need help within the next 30 days. We do NOT work with e-commerce stores or one-person startups.`

export default function SettingsICP() {
  const [icp, setIcp] = useState<ICP | null>(null)
  const [busy, setBusy] = useState(false)
  const { toast } = useToast()

  useEffect(() => {
    api.get<ICP>('/api/icp').then(setIcp).catch(() => {})
  }, [])

  if (!icp) return null

  async function save() {
    if (!icp) return
    setBusy(true)
    try {
      setIcp(await api.put<ICP>('/api/icp', icp))
      toast('Ideal customer profile saved')
    } catch (ex) {
      toast(ex instanceof Error ? ex.message : 'Save failed', true)
    } finally {
      setBusy(false)
    }
  }

  const weightKeys: (keyof Weights)[] = ['budget', 'authority', 'need', 'timeline', 'fit']

  return (
    <>
      <div className="page-head">
        <div>
          <span className="eyebrow">Qualification brain</span>
          <h1>Ideal customer profile</h1>
          <p className="sub">Describe your perfect customer in plain English. The AI turns this into qualifying questions and a score.</p>
        </div>
      </div>

      <div className="grid cols-3">
        <div className="shell span-2 fade-up">
          <div className="card">
            <h3>Who do you want to work with?</h3>
            <p className="card-sub">Budgets, company size, decision authority, timeline, industries to avoid — anything goes.</p>
            <textarea
              rows={9}
              value={icp.description}
              onChange={(e) => setIcp({ ...icp, description: e.target.value })}
              placeholder={EXAMPLE}
            />
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 14 }}>
              <button className="btn subtle" onClick={() => setIcp({ ...icp, description: EXAMPLE })}>
                Use example
              </button>
              <button className="btn" onClick={save} disabled={busy}>
                Save profile <span className="btn-orb">✓</span>
              </button>
            </div>
          </div>
        </div>

        <div className="shell fade-up d1">
          <div className="card">
            <h3>Scoring</h3>
            <p className="card-sub">A lead is qualified when its score clears the threshold.</p>
            <label className="field">
              <span>Qualification threshold — {icp.threshold}</span>
              <input
                type="number"
                min={1}
                max={100}
                value={icp.threshold}
                onChange={(e) => setIcp({ ...icp, threshold: Number(e.target.value) || 70 })}
              />
              <div className="field-hint">Default 70. Lower it to let more leads through.</div>
            </label>
            <div style={{ fontSize: 12, fontWeight: 700, color: 'var(--ink-2)', margin: '14px 0 8px' }}>Criteria weights</div>
            {weightKeys.map((k) => (
              <div key={k} style={{ display: 'grid', gridTemplateColumns: '90px 1fr 40px', gap: 10, alignItems: 'center', marginBottom: 8 }}>
                <span style={{ fontSize: 12.5, textTransform: 'capitalize', color: 'var(--ink-2)', fontWeight: 600 }}>{k}</span>
                <input
                  type="range"
                  min={0}
                  max={50}
                  value={icp.weights[k]}
                  onChange={(e) => setIcp({ ...icp, weights: { ...icp.weights, [k]: Number(e.target.value) } })}
                />
                <span className="tnum" style={{ fontSize: 12.5, fontWeight: 700, textAlign: 'right' }}>{icp.weights[k]}</span>
              </div>
            ))}
            <div className="field-hint">Relative importance of each BANT dimension.</div>
          </div>
        </div>
      </div>
    </>
  )
}
