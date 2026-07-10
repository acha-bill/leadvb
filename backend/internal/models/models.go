package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Account struct {
	ID           int64
	Name         string
	Company      string
	Email        string
	PasswordHash string
	Plan         string
	ParentID     sql.NullInt64
	WhiteLabel   bool
	Settings     sql.NullString
	CreatedAt    time.Time
}

type AccountSettings struct {
	WeeklyReport bool `json:"weekly_report"`
}

func (a *Account) ParsedSettings() AccountSettings {
	s := AccountSettings{WeeklyReport: true}
	if a.Settings.Valid && a.Settings.String != "" {
		json.Unmarshal([]byte(a.Settings.String), &s)
	}
	return s
}

type APIKey struct {
	ID        int64  `json:"id"`
	AccountID int64  `json:"-"`
	PublicKey string `json:"public_key"`
	SecretKey string `json:"secret_key"`
	Active    bool   `json:"active"`
}

type ICP struct {
	AccountID   int64   `json:"-"`
	Description string  `json:"description"`
	Threshold   int     `json:"threshold"`
	Weights     Weights `json:"weights"`
}

type Weights struct {
	Budget    int `json:"budget"`
	Authority int `json:"authority"`
	Need      int `json:"need"`
	Timeline  int `json:"timeline"`
	Fit       int `json:"fit"`
}

func DefaultWeights() Weights {
	return Weights{Budget: 25, Authority: 20, Need: 25, Timeline: 20, Fit: 10}
}

type ProactiveConfig struct {
	Enabled      bool   `json:"enabled"`
	DelaySeconds int    `json:"delay_seconds"`
	Message      string `json:"message"`
}

type PageRules struct {
	Mode     string   `json:"mode"`
	Patterns []string `json:"patterns"`
}

type WidgetConfig struct {
	CompanyName  string          `json:"company_name"`
	PrimaryColor string          `json:"primary_color"`
	Position     string          `json:"position"`
	Greeting     string          `json:"greeting"`
	LogoURL      string          `json:"logo_url"`
	Branding     bool            `json:"branding"`
	QuickReplies bool            `json:"quick_replies"`
	Language     string          `json:"language"`
	Proactive    ProactiveConfig `json:"proactive"`
	ExitIntent   bool            `json:"exit_intent"`
	Pages        PageRules       `json:"pages"`
}

func DefaultWidgetConfig(company string) WidgetConfig {
	return WidgetConfig{
		CompanyName:  company,
		PrimaryColor: "#4F46E5",
		Position:     "right",
		Greeting:     "Hi there! 👋 How can we help you today?",
		Branding:     true,
		QuickReplies: true,
		Language:     "auto",
		Proactive:    ProactiveConfig{Enabled: false, DelaySeconds: 8, Message: "Have a question? I can help you find out if we're a good fit."},
		ExitIntent:   false,
		Pages:        PageRules{Mode: "all"},
	}
}

type DisqualifiedAction struct {
	Mode          string `json:"mode"`
	NewsletterURL string `json:"newsletter_url"`
}

type RoutingConfig struct {
	EmailEnabled    bool               `json:"email_enabled"`
	EmailTo         string             `json:"email_to"`
	SlackEnabled    bool               `json:"slack_enabled"`
	SlackWebhookURL string             `json:"slack_webhook_url"`
	WebhookEnabled  bool               `json:"webhook_enabled"`
	WebhookURL      string             `json:"webhook_url"`
	WebhookSecret   string             `json:"webhook_secret"`
	NotifyHandoff   bool               `json:"notify_handoff"`
	Disqualified    DisqualifiedAction `json:"disqualified"`
}

func DefaultRoutingConfig(email string) RoutingConfig {
	return RoutingConfig{
		EmailEnabled:  true,
		EmailTo:       email,
		NotifyHandoff: true,
		Disqualified:  DisqualifiedAction{Mode: "polite"},
	}
}

type Conversation struct {
	ID             int64
	AccountID      int64
	VisitorID      string
	Token          string
	PageURL        string
	Status         string
	Score          sql.NullInt64
	Bant           sql.NullString
	Contact        sql.NullString
	Summary        sql.NullString
	Confidence     sql.NullInt64
	Language       string
	OverrideStatus sql.NullString
	OverrideNote   sql.NullString
	MessageCount   int
	StartedAt      time.Time
	EndedAt        sql.NullTime
	LastActivityAt time.Time
}

type Message struct {
	ID             int64
	ConversationID int64
	Role           string
	Content        string
	QuickReplies   sql.NullString
	CreatedAt      time.Time
}

type Delivery struct {
	ID             int64
	AccountID      int64
	ConversationID sql.NullInt64
	Channel        string
	Kind           string
	Status         string
	Attempts       int
	LastError      sql.NullString
	Payload        string
	NextAttemptAt  time.Time
	CreatedAt      time.Time
	SentAt         sql.NullTime
}

type Feedback struct {
	ID              int64
	AccountID       int64
	ConversationID  int64
	OriginalStatus  string
	CorrectedStatus string
	Note            sql.NullString
	CreatedAt       time.Time
}

func JSONString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
