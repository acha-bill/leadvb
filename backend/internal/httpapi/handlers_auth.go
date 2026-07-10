package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"leadqualifier/internal/auth"
	"leadqualifier/internal/delivery"
	"leadqualifier/internal/models"
)

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	if !s.Limiter.Allow("signup:"+clientIP(r), 0.1, 5) {
		errJSON(w, http.StatusTooManyRequests, "too many attempts, try later")
		return
	}
	var in struct {
		Name     string `json:"name"`
		Company  string `json:"company"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || !strings.Contains(in.Email, "@") || len(in.Password) < 8 {
		errJSON(w, http.StatusBadRequest, "name, valid email and password (8+ chars) required")
		return
	}
	if _, err := s.Store.GetAccountByEmail(in.Email); err == nil {
		errJSON(w, http.StatusConflict, "an account with this email already exists")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "could not create account")
		return
	}
	id, err := s.Store.CreateAccount(in.Name, in.Company, in.Email, hash, s.Cfg.DefaultPlan, nil)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "could not create account")
		return
	}
	s.bootstrapAccount(id, in.Company, in.Email)
	if err := s.setSessionCookie(w, id); err != nil {
		errJSON(w, http.StatusInternalServerError, "session error")
		return
	}
	acct, _ := s.Store.GetAccountByID(id)
	writeJSON(w, http.StatusCreated, s.accountJSON(acct))
}

func (s *Server) bootstrapAccount(id int64, company, email string) {
	s.Store.CreateAPIKeys(id)
	s.Store.UpsertWidgetConfig(id, models.DefaultWidgetConfig(company))
	s.Store.UpsertRoutingConfig(id, models.DefaultRoutingConfig(email))
	welcome := fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:640px;margin:0 auto">
<h2>Welcome to Lead Qualifier 🎉</h2>
<p>Your AI assistant is ready to qualify leads. Three steps to go live:</p>
<ol>
<li><b>Describe your ideal customer</b> in plain English (Settings → ICP).</li>
<li><b>Pick where leads go</b> — email, Slack or your CRM (Settings → Routing).</li>
<li><b>Paste the widget snippet</b> into your website (Settings → Install).</li>
</ol>
<p><a href="%s" style="color:#4F46E5">Open your dashboard →</a></p>
</div>`, s.Cfg.DashboardOrigin)
	delivery.EnqueueSystemEmail(s.Store, id, email, "Welcome to Lead Qualifier — 3 steps to your first qualified lead", welcome, "welcome")
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.Limiter.Allow("login:"+clientIP(r), 0.2, 8) {
		errJSON(w, http.StatusTooManyRequests, "too many attempts, try later")
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	acct, err := s.Store.GetAccountByEmail(strings.ToLower(strings.TrimSpace(in.Email)))
	if err != nil || !auth.CheckPassword(acct.PasswordHash, in.Password) {
		errJSON(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err := s.setSessionCookie(w, acct.ID); err != nil {
		errJSON(w, http.StatusInternalServerError, "session error")
		return
	}
	writeJSON(w, http.StatusOK, s.accountJSON(acct))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "lq_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request, acct *models.Account) {
	writeJSON(w, http.StatusOK, s.accountJSON(acct))
}

func (s *Server) accountJSON(acct *models.Account) map[string]any {
	plan := s.Cfg.Plan(acct.Plan)
	used, _ := s.Store.CountConversationsThisMonth(acct.ID)
	return map[string]any{
		"id":       acct.ID,
		"name":     acct.Name,
		"company":  acct.Company,
		"email":    acct.Email,
		"plan":     acct.Plan,
		"settings": acct.ParsedSettings(),
		"plan_limits": map[string]any{
			"conversations_per_month": plan.ConversationsPerMonth,
			"webhooks":                plan.Webhooks,
			"api":                     plan.API,
			"white_label":             plan.WhiteLabel,
			"used_this_month":         used,
		},
		"features": map[string]bool{
			"proactive_triggers": s.Cfg.Features.ProactiveTriggers,
			"exit_intent":        s.Cfg.Features.ExitIntent,
			"quick_replies":      s.Cfg.Features.QuickReplies,
			"email_delivery":     s.Cfg.Features.EmailDelivery,
			"slack_delivery":     s.Cfg.Features.SlackDelivery,
			"webhook_delivery":   s.Cfg.Features.WebhookDelivery,
			"weekly_reports":     s.Cfg.Features.WeeklyReports,
			"public_api":         s.Cfg.Features.PublicAPI,
			"handoff":            s.Cfg.Features.Handoff,
			"multi_language":     s.Cfg.Features.MultiLanguage,
		},
	}
}
