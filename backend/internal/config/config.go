package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Features struct {
	ProactiveTriggers bool
	ExitIntent        bool
	QuickReplies      bool
	EmailDelivery     bool
	SlackDelivery     bool
	WebhookDelivery   bool
	WeeklyReports     bool
	PublicAPI         bool
	Handoff           bool
	MultiLanguage     bool
}

type Config struct {
	Port            string
	DBDSN           string
	JWTSecret       string
	PublicBaseURL   string
	DashboardOrigin string
	CookieSecure    bool

	AIProvider     string
	OpenAIKey      string
	OpenAIModel    string
	OpenAIBaseURL  string
	AnthropicKey   string
	AnthropicModel string

	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	WidgetDir           string
	DefaultPlan         string
	EnforceQuotas       bool
	ManualScreenMinutes int
	IdleAbandonMinutes  int
	MaxMessagesPerConv  int

	Features Features
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := strings.ToLower(os.Getenv(key))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func Load() *Config {
	c := &Config{
		Port:            env("PORT", "8080"),
		DBDSN:           env("DB_DSN", "app:app@tcp(127.0.0.1:3306)/leadqualifier?parseTime=true&charset=utf8mb4"),
		JWTSecret:       env("JWT_SECRET", ""),
		PublicBaseURL:   strings.TrimRight(env("PUBLIC_BASE_URL", "http://localhost:8080"), "/"),
		DashboardOrigin: strings.TrimRight(env("DASHBOARD_ORIGIN", "http://localhost:8081"), "/"),
		CookieSecure:    envBool("COOKIE_SECURE", false),

		AIProvider:     strings.ToLower(env("AI_PROVIDER", "mock")),
		OpenAIKey:      env("OPENAI_API_KEY", ""),
		OpenAIModel:    env("OPENAI_MODEL", "gpt-4o-mini"),
		OpenAIBaseURL:  strings.TrimRight(env("OPENAI_BASE_URL", "https://api.openai.com"), "/"),
		AnthropicKey:   env("ANTHROPIC_API_KEY", ""),
		AnthropicModel: env("ANTHROPIC_MODEL", "claude-haiku-4-5-20251001"),

		SMTPHost: env("SMTP_HOST", ""),
		SMTPPort: envInt("SMTP_PORT", 587),
		SMTPUser: env("SMTP_USER", ""),
		SMTPPass: env("SMTP_PASS", ""),
		SMTPFrom: env("SMTP_FROM", "Lead Qualifier <no-reply@localhost>"),

		WidgetDir:           env("WIDGET_DIR", "../widget"),
		DefaultPlan:         env("DEFAULT_PLAN", "professional"),
		EnforceQuotas:       envBool("ENFORCE_QUOTAS", false),
		ManualScreenMinutes: envInt("MANUAL_SCREEN_MINUTES", 9),
		IdleAbandonMinutes:  envInt("IDLE_ABANDON_MINUTES", 30),
		MaxMessagesPerConv:  envInt("MAX_MESSAGES_PER_CONV", 60),

		Features: Features{
			ProactiveTriggers: envBool("FEATURE_PROACTIVE_TRIGGERS", true),
			ExitIntent:        envBool("FEATURE_EXIT_INTENT", true),
			QuickReplies:      envBool("FEATURE_QUICK_REPLIES", true),
			EmailDelivery:     envBool("FEATURE_EMAIL_DELIVERY", true),
			SlackDelivery:     envBool("FEATURE_SLACK_DELIVERY", true),
			WebhookDelivery:   envBool("FEATURE_WEBHOOK_DELIVERY", true),
			WeeklyReports:     envBool("FEATURE_WEEKLY_REPORTS", true),
			PublicAPI:         envBool("FEATURE_PUBLIC_API", true),
			Handoff:           envBool("FEATURE_HANDOFF", true),
			MultiLanguage:     envBool("FEATURE_MULTILANG", true),
		},
	}
	if c.JWTSecret == "" {
		c.JWTSecret = "dev-insecure-secret-change-me"
		log.Println("WARNING: JWT_SECRET not set, using insecure development secret")
	}
	return c
}

type PlanLimits struct {
	ConversationsPerMonth int
	Webhooks              bool
	API                   bool
	WhiteLabel            bool
	SubAccounts           int
}

func (c *Config) Plan(name string) PlanLimits {
	switch name {
	case "professional":
		return PlanLimits{ConversationsPerMonth: 500, Webhooks: true, API: true, WhiteLabel: true}
	case "agency":
		return PlanLimits{ConversationsPerMonth: 100000, Webhooks: true, API: true, WhiteLabel: true, SubAccounts: 10}
	case "enterprise":
		return PlanLimits{ConversationsPerMonth: 1000000, Webhooks: true, API: true, WhiteLabel: true, SubAccounts: 1000}
	default:
		return PlanLimits{ConversationsPerMonth: 200}
	}
}
