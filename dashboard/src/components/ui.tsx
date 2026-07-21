import type { ReactNode } from 'react'

export function Shell({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <section className={`card ${className}`}>{children}</section>
}

export function StatCard({ label, value, hint, delay = 0 }: { label: string; value: ReactNode; hint?: ReactNode; delay?: number }) {
  return (
    <section className={`card stat-card fade-up d${delay}`}>
      <div className="stat-label">{label}</div>
      <div className="stat-value tnum">{value}</div>
      {hint && <div className="stat-hint">{hint}</div>}
    </section>
  )
}

export function StatusBadge({ status }: { status: string }) {
  const labels: Record<string, string> = {
    active: 'Active',
    qualified: 'Qualified',
    disqualified: 'Not a fit',
    abandoned: 'Abandoned',
    handoff: 'Needs human',
    closed: 'Closed',
    hot: 'Hot',
    cold: 'Cold',
  }
  return <span className={`badge ${status}`}>{labels[status] || status}</span>
}

export function ScorePill({ score }: { score: number | null }) {
  if (score === null) return <span className="not-set">Not scored</span>
  const cls = score >= 70 ? 'score-high' : score >= 45 ? 'score-mid' : 'score-low'
  return <span className={`score-pill tnum ${cls}`}>{score}</span>
}

export function Toggle({ checked, onChange, label, sub }: { checked: boolean; onChange: (v: boolean) => void; label: string; sub?: string }) {
  return (
    <label className="switch">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      <span className="track" />
      <span className="switch-label">
        {label}
        {sub && <span className="switch-sub">{sub}</span>}
      </span>
    </label>
  )
}

export function timeAgo(iso: string): string {
  const t = new Date(iso).getTime()
  const s = Math.max(0, (Date.now() - t) / 1000)
  if (s < 60) return 'just now'
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`
  return `${Math.floor(s / 86400)}d ago`
}

export function fmtDuration(seconds: number): string {
  if (!seconds || seconds <= 0) return 'Not available'
  if (seconds < 60) return `${Math.round(seconds)}s`
  const m = Math.floor(seconds / 60)
  const s = Math.round(seconds % 60)
  return `${m}m ${s}s`
}
