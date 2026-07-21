import { useMemo, useRef, useState } from 'react'
import type { DailyPoint } from '../types'

const S1 = '#2455d6'
const S2 = '#16734a'
const GRID = '#deded7'
const MUTED = '#6f7c7f'
const ORDINAL = ['#cad7f8', '#92acef', '#5f82e3', '#2455d6', '#173ba0']

function niceMax(v: number): number {
  if (v <= 5) return 5
  const pow = Math.pow(10, Math.floor(Math.log10(v)))
  for (const m of [1, 2, 2.5, 5, 10]) {
    if (v <= m * pow) return m * pow
  }
  return 10 * pow
}

function topRoundedBar(x: number, y: number, w: number, h: number, r: number): string {
  const rr = Math.min(r, w / 2, h)
  return `M${x},${y + h} L${x},${y + rr} Q${x},${y} ${x + rr},${y} L${x + w - rr},${y} Q${x + w},${y} ${x + w},${y + rr} L${x + w},${y + h} Z`
}

export function TrendChart({ data }: { data: DailyPoint[] }) {
  const [hover, setHover] = useState<number | null>(null)
  const wrapRef = useRef<HTMLDivElement>(null)

  const W = 640
  const H = 220
  const pad = { l: 34, r: 14, t: 12, b: 26 }
  const iw = W - pad.l - pad.r
  const ih = H - pad.t - pad.b

  const maxV = useMemo(() => niceMax(Math.max(1, ...data.map((d) => d.conversations))), [data])
  const x = (i: number) => pad.l + (data.length <= 1 ? iw / 2 : (i / (data.length - 1)) * iw)
  const y = (v: number) => pad.t + ih - (v / maxV) * ih

  const path = (get: (d: DailyPoint) => number) =>
    data.map((d, i) => `${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(get(d)).toFixed(1)}`).join(' ')

  const ticks = [0, maxV / 2, maxV]
  const xtickEvery = Math.max(1, Math.ceil(data.length / 6))

  function onMove(e: React.MouseEvent<SVGSVGElement>) {
    const rect = e.currentTarget.getBoundingClientRect()
    const px = ((e.clientX - rect.left) / rect.width) * W
    if (data.length === 0) return
    const idx = Math.round(((px - pad.l) / iw) * (data.length - 1))
    setHover(Math.max(0, Math.min(data.length - 1, idx)))
  }

  const h = hover !== null ? data[hover] : null
  const last = data[data.length - 1]

  return (
    <div ref={wrapRef} style={{ position: 'relative' }}>
      <div className="legend">
        <span className="lg"><i style={{ background: S1 }} />Conversations</span>
        <span className="lg"><i style={{ background: S2 }} />Qualified</span>
      </div>
      <svg
        viewBox={`0 0 ${W} ${H}`}
        style={{ width: '100%', height: 'auto', display: 'block' }}
        onMouseMove={onMove}
        onMouseLeave={() => setHover(null)}
        role="img"
        aria-label="Daily conversations and qualified leads"
      >
        {ticks.map((t) => (
          <g key={t}>
            <line x1={pad.l} x2={W - pad.r} y1={y(t)} y2={y(t)} stroke={GRID} strokeWidth={1} />
            <text x={pad.l - 8} y={y(t) + 4} textAnchor="end" fontSize={10} fill={MUTED} style={{ fontVariantNumeric: 'tabular-nums' }}>
              {t}
            </text>
          </g>
        ))}
        {data.map((d, i) =>
          i % xtickEvery === 0 ? (
            <text key={d.date} x={x(i)} y={H - 8} textAnchor="middle" fontSize={10} fill={MUTED}>
              {d.date.slice(5)}
            </text>
          ) : null,
        )}
        {hover !== null && (
          <line x1={x(hover)} x2={x(hover)} y1={pad.t} y2={pad.t + ih} stroke={MUTED} strokeWidth={1} strokeDasharray="3 3" />
        )}
        <path d={path((d) => d.conversations)} fill="none" stroke={S1} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" />
        <path d={path((d) => d.qualified)} fill="none" stroke={S2} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" />
        {hover !== null && h && (
          <g>
            <circle cx={x(hover)} cy={y(h.conversations)} r={4.5} fill={S1} stroke="#ffffff" strokeWidth={2} />
            <circle cx={x(hover)} cy={y(h.qualified)} r={4.5} fill={S2} stroke="#ffffff" strokeWidth={2} />
          </g>
        )}
        {last && hover === null && (
          <g>
            <circle cx={x(data.length - 1)} cy={y(last.conversations)} r={3.5} fill={S1} />
            <circle cx={x(data.length - 1)} cy={y(last.qualified)} r={3.5} fill={S2} />
          </g>
        )}
      </svg>
      {hover !== null && h && wrapRef.current && (
        <div
          className="viz-tooltip"
          style={{ left: `${((x(hover) / W) * 100).toFixed(2)}%`, top: 30 }}
        >
          <div className="tt-title">{h.date}</div>
          <div className="tt-row"><span className="tt-dot" style={{ background: S1 }} />Conversations: <b>{h.conversations}</b></div>
          <div className="tt-row"><span className="tt-dot" style={{ background: S2 }} />Qualified: <b>{h.qualified}</b></div>
        </div>
      )}
    </div>
  )
}

export function ConfidenceHistogram({ buckets }: { buckets: number[] }) {
  const [hover, setHover] = useState<number | null>(null)
  const labels = ['0-20', '21-40', '41-60', '61-80', '81-100']
  const W = 300
  const H = 170
  const pad = { l: 8, r: 8, t: 10, b: 24 }
  const iw = W - pad.l - pad.r
  const ih = H - pad.t - pad.b
  const max = Math.max(1, ...buckets)
  const bw = iw / buckets.length

  return (
    <div style={{ position: 'relative' }}>
      <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: 'auto', display: 'block' }} role="img" aria-label="AI confidence distribution">
        <line x1={pad.l} x2={W - pad.r} y1={pad.t + ih} y2={pad.t + ih} stroke="#c3c2b7" strokeWidth={1} />
        {buckets.map((v, i) => {
          const bh = Math.max(v > 0 ? 3 : 0, (v / max) * ih)
          const bx = pad.l + i * bw + 6
          const bwid = bw - 12
          return (
            <g key={i} onMouseEnter={() => setHover(i)} onMouseLeave={() => setHover(null)}>
              <rect x={pad.l + i * bw} y={pad.t} width={bw} height={ih} fill="transparent" />
              {v > 0 && <path d={topRoundedBar(bx, pad.t + ih - bh, bwid, bh, 4)} fill={ORDINAL[i]} opacity={hover === null || hover === i ? 1 : 0.45} />}
              <text x={pad.l + i * bw + bw / 2} y={H - 8} textAnchor="middle" fontSize={9.5} fill={MUTED}>
                {labels[i]}
              </text>
              {v > 0 && (
                <text x={pad.l + i * bw + bw / 2} y={pad.t + ih - bh - 5} textAnchor="middle" fontSize={10.5} fontWeight={700} fill="#526064" style={{ fontVariantNumeric: 'tabular-nums' }}>
                  {v}
                </text>
              )}
            </g>
          )
        })}
      </svg>
    </div>
  )
}

export function OutcomeBars({ rows }: { rows: { label: string; value: number }[] }) {
  const max = Math.max(1, ...rows.map((r) => r.value))
  return (
    <div className="bant-bars" role="img" aria-label="Conversation outcomes">
      {rows.map((r) => (
        <div className="bant-row" key={r.label}>
          <span className="bl">{r.label}</span>
          <div className="bant-track">
            <div className="bant-fill" style={{ width: `${(r.value / max) * 100}%`, background: S1 }} />
          </div>
          <span className="bant-val tnum">{r.value}</span>
        </div>
      ))}
    </div>
  )
}
