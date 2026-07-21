import { useEffect, useState } from 'react'
import { api } from '../api'
import { useAuth } from '../App'
import type { Metrics } from '../types'
import { StatCard, Shell, fmtDuration } from '../components/ui'
import { TrendChart, ConfidenceHistogram, OutcomeBars } from '../components/charts'

export default function Overview() {
  const [metrics, setMetrics] = useState<Metrics | null>(null)
  const [days, setDays] = useState(30)
  const { account } = useAuth()

  useEffect(() => {
    api.get<Metrics>(`/api/metrics?days=${days}`).then(setMetrics).catch(() => {})
  }, [days])

  if (!metrics) return null

  const hours = metrics.time_saved_minutes / 60
  const timeSaved = hours >= 1 ? `${hours.toFixed(1)}h` : `${Math.round(metrics.time_saved_minutes)}m`

  return (
    <>
      <div className="page-head">
        <div>
          <span className="eyebrow">Overview</span>
          <h1>Good to see you, {account?.name?.split(' ')[0]}</h1>
          <p className="sub">A clear view of who visited, who qualified, and how much screening work the assistant handled.</p>
        </div>
        <div className="seg">
          {[7, 30, 90].map((d) => (
            <button key={d} className={days === d ? 'on' : ''} onClick={() => setDays(d)}>
              {d} days
            </button>
          ))}
        </div>
      </div>

      <div className="grid cols-4">
        <StatCard label="Conversations" value={metrics.conversations} hint={`${metrics.active} active right now`} />
        <StatCard label="Qualified leads" value={metrics.qualified} hint={<span className="stat-delta-good">{metrics.qualification_rate.toFixed(0)}% of completed</span>} delay={1} />
        <StatCard label="Avg. time to qualify" value={fmtDuration(metrics.avg_qualify_seconds)} hint="From first message to verdict" delay={2} />
        <StatCard label="Screening time saved" value={timeSaved} hint="vs. manual replies to every lead" delay={3} />
      </div>

      <div className="grid cols-3" style={{ marginTop: 20 }}>
        <div className="shell span-2 fade-up d1">
          <div className="card">
            <h3>Lead flow</h3>
            <p className="card-sub">Daily conversations compared with qualified leads.</p>
            <TrendChart data={metrics.daily} />
          </div>
        </div>
        <div className="shell fade-up d2">
          <div className="card">
            <h3>AI confidence</h3>
            <p className="card-sub">How confident the assistant was in each verdict.</p>
            <ConfidenceHistogram buckets={metrics.confidence_buckets} />
          </div>
        </div>
      </div>

      <div className="grid cols-3" style={{ marginTop: 20 }}>
        <div className="shell fade-up d1">
          <div className="card">
            <h3>Outcomes</h3>
            <p className="card-sub">Where conversations ended up.</p>
            <OutcomeBars
              rows={[
                { label: 'Qualified', value: metrics.qualified },
                { label: 'Not a fit', value: metrics.disqualified },
                { label: 'Abandoned', value: metrics.abandoned },
                { label: 'Handoff', value: metrics.handoff },
              ]}
            />
          </div>
        </div>
        <div className="shell fade-up d2">
          <div className="card">
            <h3>Widget engagement</h3>
            <p className="card-sub">Page loads that turned into chats.</p>
            <div className="kv" style={{ marginTop: 8 }}>
              <dt>Widget loads</dt>
              <dd className="tnum">{metrics.widget_loads}</dd>
              <dt>Chats opened</dt>
              <dd className="tnum">{metrics.chat_opens}</dd>
              <dt>Open rate</dt>
              <dd className="tnum">{metrics.open_rate.toFixed(1)}%</dd>
            </div>
          </div>
        </div>
        <Shell className="fade-up d3">
          <h3>Plan usage</h3>
          <p className="card-sub">Conversations this month.</p>
          <div className="stat-value tnum" style={{ fontSize: 24 }}>
            {account?.plan_limits.used_this_month} <span style={{ color: 'var(--muted)', fontSize: 15 }}>/ {account?.plan_limits.conversations_per_month}</span>
          </div>
          <div className="bant-track" style={{ marginTop: 10 }}>
            <div
              className="bant-fill"
              style={{
                width: `${Math.min(100, ((account?.plan_limits.used_this_month || 0) / (account?.plan_limits.conversations_per_month || 1)) * 100)}%`,
              }}
            />
          </div>
        </Shell>
      </div>
    </>
  )
}
