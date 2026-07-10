# Lead Qualifier — REST API

Base URL: your `PUBLIC_BASE_URL` (e.g. `https://api.yourdomain.com`).

Authentication: every `/api/v1/*` request needs your **secret key**:

```
Authorization: Bearer sk_live_...
```

Keys live in the dashboard under **Install & API**. API access requires the Professional plan or higher (configurable). Rate limit: 10 req/s sustained, burst 30, per key.

Errors are JSON: `{"error": "message"}` with a matching HTTP status.

---

## Sessions & messages

### Create a session

```
POST /api/v1/sessions
{"page_url": "https://example.com/pricing", "visitor_id": "crm-user-42"}
```

201 →

```json
{
  "session_token": "3fa8...",
  "conversation_id": 17,
  "messages": [{"id": 41, "role": "assistant", "content": "Hi there! ...", "quick_replies": [], "created_at": "..."}]
}
```

### Send a visitor message (synchronous)

Blocks until the AI replies (up to ~45s worst case; typically 1–3s).

```
POST /api/v1/sessions/{session_token}/messages
{"content": "We're a 12-person firm looking for IT support"}
```

200 →

```json
{
  "status": "active",
  "reply": {"id": 43, "role": "assistant", "content": "...", "quick_replies": ["Under $1,000", "$1,000–$5,000"], "created_at": "..."}
}
```

`status` becomes `qualified` / `disqualified` / `handoff` when the conversation completes.

## Conversations

### List

```
GET /api/v1/conversations?status=qualified&page=1
```

200 → `{"conversations": [...], "total": 132, "page": 1}` (50 per page, newest activity first).

Conversation object:

```json
{
  "id": 17,
  "status": "qualified",
  "score": 84,
  "confidence": 90,
  "contact": {"name": "Dana", "email": "dana@firm.com", "phone": "", "company": "Firm LLP"},
  "bant": {"budget": 85, "authority": 90, "need": 80, "timeline": 75, "fit": 88},
  "summary": "12-person accounting firm needs managed IT within 30 days...",
  "page_url": "https://example.com/pricing",
  "language": "en",
  "override_status": null,
  "message_count": 12,
  "started_at": "...", "ended_at": "...", "last_activity_at": "..."
}
```

### Get one (with transcript)

```
GET /api/v1/conversations/{id}
```

Adds `"messages": [...]`.

## ICP configuration

```
GET /api/v1/icp
PUT /api/v1/icp
{"description": "We only work with ...", "threshold": 70,
 "weights": {"budget": 25, "authority": 20, "need": 25, "timeline": 20, "fit": 10}}
```

## Agency: client sub-accounts

Requires an Agency plan.

```
POST /api/v1/accounts
{"name": "Client Owner", "company": "Client Co", "email": "owner@client.com"}
```

201 → `{"account_id": 9, "temporary_password": "...", "public_key": "pk_live_...", "secret_key": "sk_live_..."}`

The client can sign in to the dashboard with the temporary password; you can drive their config with the returned secret key (all `/api/v1` endpoints operate on the account owning the key).

```
GET /api/v1/accounts        → {"accounts": [...]}
```

---

## Lead webhooks (outbound)

When a lead qualifies (or a visitor requests a human, if enabled), the platform POSTs to your configured webhook URL:

Headers:

```
Content-Type: application/json
X-LeadQualifier-Event: lead.qualified | lead.handoff | lead.test
X-LeadQualifier-Signature: sha256=<hex hmac>
```

Body (the lead payload):

```json
{
  "event": "lead.qualified",
  "account_id": 3,
  "company": "Northshore IT",
  "conversation_id": 17,
  "status": "qualified",
  "score": 84,
  "confidence": 90,
  "contact": {"name": "...", "email": "...", "phone": "...", "company": "..."},
  "bant": {"budget": 85, "authority": 90, "need": 80, "timeline": 75, "fit": 88},
  "summary": "...",
  "page_url": "...",
  "language": "en",
  "started_at": "...", "ended_at": "...",
  "dashboard_url": "https://app.yourdomain.com/conversations/17",
  "transcript": [{"role": "assistant", "content": "...", "at": "..."}, ...]
}
```

Retries: up to 5 attempts with quadratic backoff on any non-2xx response.

### Verifying the signature (Node)

```js
import crypto from 'node:crypto'

function verify(rawBody, header, secret) {
  const expected = 'sha256=' + crypto.createHmac('sha256', secret).update(rawBody).digest('hex')
  return crypto.timingSafeEqual(Buffer.from(header), Buffer.from(expected))
}
```

Compute over the **raw request body** before any JSON parsing.

---

## Widget endpoints

Used by `chat.js`; also available to custom widget builds. Tenant resolution, in order:

1. **Registered domain** (preferred) — the request's `Origin`/`Referer` host is matched against domains saved in the dashboard (subdomain-aware). No key needed anywhere in the page.
2. **Widget key** (`key` field / `data-widget-key`) — a public *identifier*, not a credential: it can start chats but cannot read conversations, contacts, or configuration. Used for previews, local files and multi-tenant embeds.

Reads of an ongoing conversation always require its unguessable `session_token`.

```
GET  /api/widget/boot?url=...[&key=pk_live_...]  → widget config + ws_url + disabled flag
POST /api/widget/session                         → create/resume {session_token?, visitor_id, page_url, trigger, key?}
POST /api/widget/message                         → {session_token, content} (reply arrives via WS/poll)
GET  /api/widget/poll?session_token=&after=      → {messages, status}
POST /api/widget/event                           → {type: loaded|opened|closed|proactive_shown|exit_shown, visitor_id, page_url, key?}
WS   /ws/widget?session_token=                   → {type: message|typing|status} frames; send {type:"message", content}
```

Manage registered domains from the dashboard, or via the authenticated dashboard API: `GET/PUT /api/widget-domains` with `{"domains": ["example.com"]}` (cookie auth).
