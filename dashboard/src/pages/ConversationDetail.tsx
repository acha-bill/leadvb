import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, connectDashboardWS } from '../api'
import { useToast } from '../App'
import type { Conversation, Message } from '../types'
import { StatusBadge, ScorePill, timeAgo } from '../components/ui'

export default function ConversationDetail() {
  const { id } = useParams()
  const [conv, setConv] = useState<Conversation | null>(null)
  const [note, setNote] = useState('')
  const [reply, setReply] = useState('')
  const [showOverride, setShowOverride] = useState(false)
  const { toast } = useToast()
  const logRef = useRef<HTMLDivElement>(null)

  function load() {
    api.get<Conversation>(`/api/conversations/${id}`).then((c) => {
      setConv(c)
      requestAnimationFrame(() => {
        logRef.current?.scrollTo({ top: logRef.current.scrollHeight })
      })
    }).catch(() => {})
  }

  useEffect(load, [id])

  useEffect(() => {
    return connectDashboardWS((ev) => {
      if (ev.type === 'conversation_updated' || ev.type === 'message') load()
    })
  }, [id])

  if (!conv) return null

  async function override(status: 'hot' | 'cold') {
    try {
      await api.post(`/api/conversations/${id}/override`, { status, note })
      toast(`Marked ${status}. The assistant will use your note next time.`)
      setShowOverride(false)
      setNote('')
      load()
    } catch (ex) {
      toast(ex instanceof Error ? ex.message : 'Failed', true)
    }
  }

  async function sendReply(e: React.FormEvent) {
    e.preventDefault()
    if (!reply.trim()) return
    try {
      await api.post(`/api/conversations/${id}/reply`, { content: reply })
      setReply('')
      load()
    } catch (ex) {
      toast(ex instanceof Error ? ex.message : 'Failed to send', true)
    }
  }

  async function closeConv() {
    try {
      await api.post(`/api/conversations/${id}/close`)
      toast('Conversation closed')
      load()
    } catch (ex) {
      toast(ex instanceof Error ? ex.message : 'Failed', true)
    }
  }

  const canReply = conv.status === 'active' || conv.status === 'handoff'
  const bantOrder = ['budget', 'authority', 'need', 'timeline', 'fit']

  return (
    <>
      <div className="page-head">
        <div>
          <Link to="/conversations" className="back-link">← All conversations</Link>
          <h1 style={{ marginTop: 8 }}>{conv.contact.name || conv.contact.email || 'Anonymous visitor'}</h1>
          <p className="sub">
            <StatusBadge status={conv.status} /> {conv.override_status && <StatusBadge status={conv.override_status} />}{' '}
            <span aria-hidden="true"> · </span>started {timeAgo(conv.started_at)}<span aria-hidden="true"> · </span>{conv.message_count} messages
          </p>
        </div>
        <div className="action-row">
          {!showOverride && (
            <button className="btn ghost" onClick={() => setShowOverride(true)}>Override verdict</button>
          )}
          {canReply && <button className="btn subtle" onClick={closeConv}>Close chat</button>}
        </div>
      </div>

      {showOverride && (
        <div className="shell" style={{ marginBottom: 20 }}>
          <div className="card">
            <h3>Manual override</h3>
            <p className="card-sub">Change the verdict and add enough context to explain what the assistant missed.</p>
            <label className="field">
              <span>Reason (optional)</span>
              <input type="text" value={note} onChange={(e) => setNote(e.target.value)} placeholder="For example: They meant $2,000 monthly, not total" />
            </label>
            <div className="action-row">
              <button className="btn" onClick={() => override('hot')}>Mark as hot</button>
              <button className="btn ghost" onClick={() => override('cold')}>Mark cold</button>
              <button className="btn subtle" onClick={() => setShowOverride(false)}>Cancel</button>
            </div>
          </div>
        </div>
      )}

      <div className="grid cols-3">
        <div className="shell span-2">
          <div className="card">
            <h3>Transcript</h3>
            <div className="chat-log" ref={logRef} style={{ maxHeight: 520, overflowY: 'auto' }}>
              {(conv.messages || []).map((m: Message) => (
                <div key={m.id} className={`chat-row ${m.role}`}>
                  <div>
                    {m.role === 'owner' && <div className="chat-meta">You</div>}
                    <div className="chat-bubble">{m.content}</div>
                    <div className="chat-meta">{new Date(m.created_at).toLocaleTimeString()}</div>
                  </div>
                </div>
              ))}
            </div>
            {canReply && (
              <form onSubmit={sendReply} className="chat-reply">
                <input
                  type="text"
                  value={reply}
                  onChange={(e) => setReply(e.target.value)}
                  placeholder={conv.status === 'handoff' ? 'The visitor asked for a person. Reply here.' : 'Reply as yourself'}
                  style={{ flex: 1 }}
                />
                <button className="btn">Send</button>
              </form>
            )}
          </div>
        </div>

        <div className="stack">
          <div className="shell">
            <div className="card">
              <h3>Verdict</h3>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12, margin: '10px 0 16px' }}>
                <ScorePill score={conv.score} />
                <span style={{ color: 'var(--muted)', fontSize: 12 }}>
                  {conv.confidence !== null ? `${conv.confidence}% confidence` : 'no confidence yet'}
                </span>
              </div>
              <div className="bant-bars">
                {bantOrder.map((k) => {
                  const v = conv.bant?.[k]
                  return (
                    <div className="bant-row" key={k}>
                      <span className="bl">{k}</span>
                      <div className="bant-track">
                        <div className="bant-fill" style={{ width: `${v ?? 0}%` }} />
                      </div>
                      <span className="bant-val tnum">{v ?? 'Not set'}</span>
                    </div>
                  )
                })}
              </div>
              {conv.summary && (
                <p style={{ fontSize: 13, color: 'var(--ink-2)', marginTop: 16, marginBottom: 0 }}>{conv.summary}</p>
              )}
            </div>
          </div>

          <div className="shell">
            <div className="card">
              <h3>Contact</h3>
              <dl className="kv" style={{ marginTop: 10 }}>
                <dt>Name</dt><dd>{conv.contact.name || 'Not provided'}</dd>
                <dt>Email</dt><dd>{conv.contact.email ? <a href={`mailto:${conv.contact.email}`}>{conv.contact.email}</a> : 'Not provided'}</dd>
                <dt>Phone</dt><dd>{conv.contact.phone || 'Not provided'}</dd>
                <dt>Company</dt><dd>{conv.contact.company || 'Not provided'}</dd>
                <dt>Page</dt><dd style={{ fontSize: 12 }}>{conv.page_url || 'Not recorded'}</dd>
                <dt>Language</dt><dd>{conv.language || 'Not recorded'}</dd>
              </dl>
              {conv.override_note && (
                <p style={{ fontSize: 12.5, color: 'var(--warn)', marginTop: 12 }}>
                  Override note: {conv.override_note}
                </p>
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
