import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, connectDashboardWS } from '../api'
import type { Conversation } from '../types'
import { StatusBadge, ScorePill, timeAgo } from '../components/ui'

const FILTERS = [
  { key: '', label: 'All' },
  { key: 'active', label: 'Active' },
  { key: 'qualified', label: 'Qualified' },
  { key: 'disqualified', label: 'Not a fit' },
  { key: 'handoff', label: 'Needs human' },
  { key: 'abandoned', label: 'Abandoned' },
]

export default function Conversations() {
  const [items, setItems] = useState<Conversation[]>([])
  const [total, setTotal] = useState(0)
  const [status, setStatus] = useState('')
  const [q, setQ] = useState('')
  const [page, setPage] = useState(1)
  const nav = useNavigate()
  const stateRef = useRef({ status, page, q })
  stateRef.current = { status, page, q }

  function load() {
    const { status, page, q } = stateRef.current
    const params = new URLSearchParams()
    if (status) params.set('status', status)
    if (q) params.set('q', q)
    params.set('page', String(page))
    api
      .get<{ conversations: Conversation[]; total: number }>(`/api/conversations?${params}`)
      .then((res) => {
        setItems(res.conversations || [])
        setTotal(res.total)
      })
      .catch(() => {})
  }

  useEffect(load, [status, page])

  useEffect(() => {
    const t = setTimeout(load, 300)
    return () => clearTimeout(t)
  }, [q])

  useEffect(() => {
    return connectDashboardWS((ev) => {
      if (ev.type === 'conversation_updated') load()
    })
  }, [])

  const pages = Math.max(1, Math.ceil(total / 25))

  return (
    <>
      <div className="page-head">
        <div>
          <span className="eyebrow">Live feed</span>
          <h1>Conversations</h1>
          <p className="sub">Every chat your assistant has had — updated in real time.</p>
        </div>
      </div>

      <div className="filters">
        <div className="seg">
          {FILTERS.map((f) => (
            <button
              key={f.key}
              className={status === f.key ? 'on' : ''}
              onClick={() => {
                setStatus(f.key)
                setPage(1)
              }}
            >
              {f.label}
            </button>
          ))}
        </div>
        <input type="text" placeholder="Search name, email, page…" value={q} onChange={(e) => { setQ(e.target.value); setPage(1) }} />
      </div>

      <div className="shell">
        <div className="card" style={{ padding: 6 }}>
          {items.length === 0 ? (
            <div className="empty">
              <b>No conversations yet</b>
              Install the widget on your site and chats will appear here instantly.
            </div>
          ) : (
            <table className="list">
              <thead>
                <tr>
                  <th>Visitor</th>
                  <th>Status</th>
                  <th>Score</th>
                  <th>Summary</th>
                  <th>Page</th>
                  <th>Last activity</th>
                </tr>
              </thead>
              <tbody>
                {items.map((c) => (
                  <tr key={c.id} className="row-link" onClick={() => nav(`/conversations/${c.id}`)}>
                    <td>
                      <b>{c.contact.name || c.contact.email || 'Anonymous'}</b>
                      {c.contact.name && c.contact.email && (
                        <div style={{ color: 'var(--muted)', fontSize: 12 }}>{c.contact.email}</div>
                      )}
                    </td>
                    <td>
                      <StatusBadge status={c.status} />{' '}
                      {c.override_status && <StatusBadge status={c.override_status} />}
                    </td>
                    <td><ScorePill score={c.score} /></td>
                    <td style={{ maxWidth: 320, color: 'var(--ink-2)' }}>
                      {(c.summary || '').slice(0, 90) || <span style={{ color: 'var(--muted)' }}>{c.message_count} messages</span>}
                    </td>
                    <td style={{ maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: 'var(--muted)', fontSize: 12 }}>
                      {pathOf(c.page_url)}
                    </td>
                    <td style={{ whiteSpace: 'nowrap', color: 'var(--muted)' }}>{timeAgo(c.last_activity_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {pages > 1 && (
        <div style={{ display: 'flex', gap: 8, justifyContent: 'center', marginTop: 16 }}>
          <button className="btn subtle" disabled={page <= 1} onClick={() => setPage(page - 1)}>← Prev</button>
          <span style={{ alignSelf: 'center', color: 'var(--muted)', fontSize: 13 }}>
            Page {page} of {pages}
          </span>
          <button className="btn subtle" disabled={page >= pages} onClick={() => setPage(page + 1)}>Next →</button>
        </div>
      )}
    </>
  )
}

function pathOf(url: string): string {
  try {
    const u = new URL(url)
    return u.pathname + u.search
  } catch {
    return url
  }
}
