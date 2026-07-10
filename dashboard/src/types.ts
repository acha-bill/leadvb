export interface PlanLimits {
  conversations_per_month: number
  webhooks: boolean
  api: boolean
  white_label: boolean
  used_this_month: number
}

export interface Account {
  id: number
  name: string
  company: string
  email: string
  plan: string
  settings: { weekly_report: boolean }
  plan_limits: PlanLimits
  features: Record<string, boolean>
}

export interface Message {
  id: number
  role: 'visitor' | 'assistant' | 'owner' | 'system'
  content: string
  quick_replies: string[] | null
  created_at: string
}

export interface Conversation {
  id: number
  token: string
  visitor_id: string
  page_url: string
  status: string
  score: number | null
  confidence: number | null
  summary: string
  language: string
  override_status: string | null
  override_note: string
  message_count: number
  started_at: string
  last_activity_at: string
  ended_at: string | null
  contact: Record<string, string>
  bant: Record<string, number | null>
  messages?: Message[]
}

export interface DailyPoint {
  date: string
  conversations: number
  qualified: number
}

export interface Metrics {
  conversations: number
  qualified: number
  disqualified: number
  abandoned: number
  handoff: number
  active: number
  qualification_rate: number
  avg_qualify_seconds: number
  time_saved_minutes: number
  confidence_buckets: number[]
  daily: DailyPoint[]
  widget_loads: number
  chat_opens: number
  open_rate: number
}

export interface Weights {
  budget: number
  authority: number
  need: number
  timeline: number
  fit: number
}

export interface ICP {
  description: string
  threshold: number
  weights: Weights
}

export interface RoutingConfig {
  email_enabled: boolean
  email_to: string
  slack_enabled: boolean
  slack_webhook_url: string
  webhook_enabled: boolean
  webhook_url: string
  webhook_secret: string
  notify_handoff: boolean
  disqualified: { mode: string; newsletter_url: string }
}

export interface WidgetConfig {
  company_name: string
  primary_color: string
  position: string
  greeting: string
  logo_url: string
  branding: boolean
  quick_replies: boolean
  language: string
  proactive: { enabled: boolean; delay_seconds: number; message: string }
  exit_intent: boolean
  pages: { mode: string; patterns: string[] }
}
