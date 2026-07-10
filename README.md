# Lead Qualifier — AI Chatbot for Lead Qualification

An always-on AI assistant for small B2B service businesses. It chats with website visitors, qualifies them against a plain-English ideal customer profile (BANT + custom criteria), and delivers only qualified leads — with the full transcript — to email, Slack, or any CRM via webhook.

## What's in the box

| Folder | Application | Stack |
|---|---|---|
| `backend/` | API, conversation engine, scoring, delivery workers, WebSockets | Go + MySQL |
| `widget/` | Embeddable chat widget (one script tag, Shadow DOM, no dependencies) | Vanilla JS |
| `dashboard/` | Owner dashboard: ICP, live conversations, overrides, routing, metrics | React + Vite + TypeScript |
| `site/` | Marketing landing page | Static HTML |
| `docs/` | REST API reference | — |

### Feature checklist

- **Widget**: floating button, mobile-responsive chat window, WebSocket with polling fallback, session persistence across reloads (localStorage), quick-reply buttons, typing indicator, brand colors/logo/greeting, left/right position, unread badge, "Powered by" branding (removable on Professional+).
- **Triggers**: visitor-initiated, time-delay proactive open, exit-intent (desktop), page targeting with include/exclude URL patterns and `*` wildcards.
- **AI engine**: OpenAI (`gpt-4o-mini` default) or Anthropic, plus a `mock` provider for key-free local testing. One question at a time, contact collection, multi-language (auto-detect or fixed), anti-hallucination instructions, JSON-structured verdicts with retry on malformed output.
- **Scoring**: 0–100 across budget / authority / need / timeline / fit with per-account weights and threshold; verdicts: qualified, disqualified, handoff.
- **Feedback loop**: manual hot/cold overrides (with notes) are injected into future prompts as owner corrections.
- **Human handoff**: visitor asks for a human → owner alerted → owner replies live from the dashboard into the widget.
- **Routing**: email (SMTP), Slack (incoming webhook), generic webhook with HMAC-SHA256 signature; per-channel test buttons; DB-backed outbox queue with retries and exponential backoff.
- **Analytics**: conversations, qualification rate, average time-to-verdict, estimated screening time saved, AI confidence distribution, widget load→open rate, daily trend.
- **Emails**: welcome email on signup, weekly lead report (toggleable per account).
- **Plans & quotas**: starter/professional/agency/enterprise with monthly conversation quotas and gated features (webhooks, API, white-label, sub-accounts). Quota enforcement is off by default (`ENFORCE_QUOTAS`).
- **Agency/white-label**: create and manage client sub-accounts via API; white-label widget.
- **Public REST API**: sessions, messages (synchronous AI reply), conversations, ICP config — see `docs/API.md`.
- **Configurability**: 10 platform feature flags via env vars + per-account toggles in the dashboard.

## Quick start (local)

Prereqs: Docker with the compose plugin.

```bash
cp .env.example .env        # defaults work out of the box (mock AI, no email)
docker compose up --build
```

| URL | What |
|---|---|
| http://localhost:8082 | Marketing site |
| http://localhost:8081 | Dashboard — create your account here |
| http://localhost:8080 | API + widget script (`/widget/chat.js`) |

Then:

1. Sign up at http://localhost:8081/signup.
2. Write your ICP under **Ideal customer** (or click "Use example").
3. Open **Install & API**, copy the snippet — or just open **Widget design** and use the live preview to have a full conversation with the mock AI.
4. Watch the conversation appear under **Conversations** in real time; the mock provider qualifies after ~6 exchanges.

The default `AI_PROVIDER=mock` needs no API key and runs a deterministic qualification script — ideal for testing routing, the dashboard, and the widget end-to-end.

### Using a real AI provider

In `.env`:

```bash
AI_PROVIDER=openai
OPENAI_API_KEY=sk-...          # https://platform.openai.com/api-keys
OPENAI_MODEL=gpt-4o-mini
```

or

```bash
AI_PROVIDER=anthropic
ANTHROPIC_API_KEY=sk-ant-...   # https://console.anthropic.com
ANTHROPIC_MODEL=claude-haiku-4-5-20251001
```

Restart: `docker compose up -d --build backend`.

### Enabling email (welcome, leads, weekly reports)

Any SMTP provider works. Suggested: [Resend](https://resend.com) (SMTP mode), Mailgun, Postmark, or Amazon SES — all have free tiers that cover a small business.

```bash
SMTP_HOST=smtp.resend.com
SMTP_PORT=587                  # 465 (implicit TLS) also supported
SMTP_USER=resend
SMTP_PASS=re_xxxxxxxx
SMTP_FROM=Lead Qualifier <leads@yourdomain.com>
```

Use the **Send test email** button in Settings → Lead routing to verify.

### Slack routing

1. In Slack: **Apps → Incoming Webhooks → Add to a channel** → copy the `https://hooks.slack.com/services/...` URL.
2. Paste it in Settings → Lead routing → Slack, enable, **Send test message**.

### CRM / webhook routing

Point the webhook URL at HubSpot/Pipedrive/Zapier/Make or your own endpoint. If you set a signing secret, verify `X-LeadQualifier-Signature` (HMAC-SHA256 of the raw body, hex, prefixed `sha256=`) — sample code in `docs/API.md`.

## Embedding the widget on a customer website

Register the website's domain in the dashboard (Install & API), then paste one credential-free line anywhere before `</body>`:

```html
<script src="https://api.yourdomain.com/widget/chat.js" async></script>
```

The backend matches the browser's `Origin`/`Referer` against registered domains (subdomains included: `example.com` covers `www.example.com`), so nothing secret — or even account-identifying — appears in the customer's page.

For pages that can't be matched by domain (local HTML files, one-off staging hosts, agency multi-tenant embeds) there's a fallback attribute carrying the **widget key**, a public identifier that can only start chats — it can never read conversations, contacts, or settings:

```html
<script src="https://api.yourdomain.com/widget/chat.js" data-widget-key="pk_live_..." async></script>
```

The **secret API key** (`sk_live_...`) is different: it unlocks the REST API and must only be used server-side.

- WordPress: paste into any "insert headers and footers" plugin.
- The optional `data-base-url` attribute overrides the API origin if you serve `chat.js` from a CDN.

## Configuration reference

All backend configuration is via environment variables (see `.env.example`):

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | API listen port |
| `DB_DSN` | local MySQL | Go MySQL DSN, needs `parseTime=true` |
| `JWT_SECRET` | insecure dev value | **Set a long random value in production** |
| `PUBLIC_BASE_URL` | `http://localhost:8080` | Public URL of the API (used in snippets, emails, WS URL) |
| `DASHBOARD_ORIGIN` | `http://localhost:8081` | Dashboard URL (CORS, links in emails) |
| `COOKIE_SECURE` | `false` | Set `true` when serving over HTTPS |
| `AI_PROVIDER` | `mock` | `openai` \| `anthropic` \| `mock` |
| `DEFAULT_PLAN` | `professional` | Plan assigned on signup (no billing integration is included) |
| `ENFORCE_QUOTAS` | `false` | Enforce monthly conversation quotas per plan |
| `MANUAL_SCREEN_MINUTES` | `9` | Minutes of manual screening assumed per lead (time-saved metric) |
| `IDLE_ABANDON_MINUTES` | `30` | Active chats idle longer than this are marked abandoned |
| `MAX_MESSAGES_PER_CONV` | `60` | Hard cap per conversation (cost guard) |

Feature flags (all `true` by default) — set to `false` to switch off platform-wide:

`FEATURE_PROACTIVE_TRIGGERS`, `FEATURE_EXIT_INTENT`, `FEATURE_QUICK_REPLIES`, `FEATURE_EMAIL_DELIVERY`, `FEATURE_SLACK_DELIVERY`, `FEATURE_WEBHOOK_DELIVERY`, `FEATURE_WEEKLY_REPORTS`, `FEATURE_PUBLIC_API`, `FEATURE_HANDOFF`, `FEATURE_MULTILANG`

Per-account toggles (dashboard): proactive triggers, exit intent, quick replies, page targeting, language, branding, weekly report, routing channels, disqualified-visitor behaviour (polite vs newsletter), qualification threshold and BANT weights.

### Plans

Defined in `backend/internal/config/config.go`:

| Plan | Conversations/mo | Webhooks/CRM | API | White-label | Sub-accounts |
|---|---|---|---|---|---|
| starter | 200 | — | — | — | — |
| professional | 500 | ✓ | ✓ | ✓ | — |
| agency | ~unlimited | ✓ | ✓ | ✓ | 10 |
| enterprise | unlimited | ✓ | ✓ | ✓ | 1000 |

Plan switching is self-serve in the dashboard (no billing is wired in — integrate Stripe or similar before charging customers).

## Hosting in production

The stack is a single `docker compose up` on any VPS (DigitalOcean, Hetzner, Linode, EC2 — 1 vCPU / 2GB is plenty to start).

### 1. DNS

Point three records at your server:

```
api.yourdomain.com        → A <server-ip>     (backend: API + widget + WS)
app.yourdomain.com        → A <server-ip>     (dashboard)
www.yourdomain.com        → A <server-ip>     (marketing site)
```

### 2. Reverse proxy + TLS (Caddy — easiest)

Install [Caddy](https://caddyserver.com) on the host; it provisions Let's Encrypt certificates automatically:

```
# /etc/caddy/Caddyfile
api.yourdomain.com {
    reverse_proxy localhost:8080
}
app.yourdomain.com {
    reverse_proxy localhost:8081
}
www.yourdomain.com {
    reverse_proxy localhost:8082
}
```

(nginx + certbot works identically — proxy the same three ports; remember `proxy_set_header Upgrade/Connection` for `/ws/` on the API host.)

### 3. Production .env

```bash
PUBLIC_BASE_URL=https://api.yourdomain.com
DASHBOARD_ORIGIN=https://app.yourdomain.com
COOKIE_SECURE=true
JWT_SECRET=$(openssl rand -hex 32)
MYSQL_PASSWORD=$(openssl rand -hex 16)
MYSQL_ROOT_PASSWORD=$(openssl rand -hex 16)
AI_PROVIDER=openai
OPENAI_API_KEY=sk-...
SMTP_HOST=...   # etc.
ENFORCE_QUOTAS=true
DEFAULT_PLAN=starter
```

Then `docker compose up -d --build`. The dashboard container proxies `/api`, `/ws` and `/widget` to the backend internally, so the app works same-origin; the widget snippet uses `PUBLIC_BASE_URL`.

Also update the hardcoded `http://localhost:8081` links in `site/index.html` to your `app.` domain before going live.

### 4. Backups

All state lives in the `mysql_data` volume:

```bash
docker compose exec mysql sh -c 'mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" leadqualifier' > backup-$(date +%F).sql
```

Cron that daily and ship it off-box.

### 5. Updating

```bash
git pull && docker compose up -d --build
```

Migrations run automatically at backend startup.

## Local development (without Docker)

```bash
# MySQL
docker run -d -p 3306:3306 -e MYSQL_DATABASE=leadqualifier -e MYSQL_USER=app \
  -e MYSQL_PASSWORD=app -e MYSQL_ROOT_PASSWORD=root mysql:8.0

# Backend (serves ../widget/chat.js automatically)
cd backend && go run .

# Dashboard with hot reload (proxies /api and /ws to :8080)
cd dashboard && npm install && npm run dev   # http://localhost:5173
```

## Architecture

```
customer website ──(chat.js snippet)──► backend :8080
                                         ├─ HTTP API (widget, dashboard, REST v1)
                                         ├─ WebSocket hub (visitor chats + dashboard live feed)
                                         ├─ conversation engine (OpenAI/Anthropic/mock, BANT scoring)
                                         ├─ delivery worker (outbox: email/Slack/webhook, retries)
                                         ├─ weekly reporter + abandon sweeper
                                         └─ MySQL (accounts, conversations, messages,
                                                   deliveries, feedback, events)
dashboard :8081 (nginx) ── proxies /api, /ws, /widget ──► backend
marketing site :8082 (nginx, static)
```

## Troubleshooting

- **Widget bubble doesn't appear** — check the browser console; verify the `data-api-key` is the *public* key and `PUBLIC_BASE_URL` is reachable from the page. Page-targeting rules can also hide it.
- **No AI replies** — `docker compose logs backend`; with `AI_PROVIDER=openai` a 401 means a bad key. The widget falls back to an apology message when the provider errors.
- **Emails not arriving** — use the routing test button; the response surfaces the SMTP error verbatim. Port 465 uses implicit TLS, 587 STARTTLS.
- **WebSocket drops behind a proxy** — ensure `Upgrade`/`Connection` headers are forwarded (see Caddy/nginx notes). The widget silently falls back to polling.
- **Reset everything** — `docker compose down -v` (deletes the database).

## Not included (by design)

Billing/payments (plans switch freely — wire Stripe before charging), file uploads in chat, WordPress plugin packaging (the snippet works in any "custom HTML" box), and automated tests (per project scope).
