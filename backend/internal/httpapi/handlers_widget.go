package httpapi

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"leadqualifier/internal/models"
	"leadqualifier/internal/wire"
)

// widgetAccount resolves the tenant for a widget request. Preferred path is the
// browser Origin/Referer header matched against registered widget domains, so
// customer pages never need to embed a key. A widget key (public identifier)
// is the fallback for previews, local files and multi-tenant embeds.
func (s *Server) widgetAccount(w http.ResponseWriter, r *http.Request, key string) *models.Account {
	if key != "" {
		acct, err := s.Store.GetAccountByPublicKey(key)
		if err != nil {
			errJSON(w, http.StatusUnauthorized, "invalid widget key")
			return nil
		}
		return acct
	}
	if host := requestSourceHost(r); host != "" {
		acct, err := s.Store.GetAccountByDomain(host)
		if err == nil {
			return acct
		}
	}
	errJSON(w, http.StatusUnauthorized, "unrecognized website — register your domain in the dashboard (Install & API) or pass data-widget-key")
	return nil
}

func requestSourceHost(r *http.Request) string {
	for _, raw := range []string{r.Header.Get("Origin"), r.Header.Get("Referer")} {
		if raw == "" || raw == "null" {
			continue
		}
		if u, err := url.Parse(raw); err == nil && u.Hostname() != "" {
			return u.Hostname()
		}
	}
	return ""
}

func (s *Server) quotaExceeded(acct *models.Account) bool {
	if !s.Cfg.EnforceQuotas {
		return false
	}
	used, err := s.Store.CountConversationsThisMonth(acct.ID)
	if err != nil {
		return false
	}
	return used >= s.Cfg.Plan(acct.Plan).ConversationsPerMonth
}

func (s *Server) handleWidgetBoot(w http.ResponseWriter, r *http.Request) {
	acct := s.widgetAccount(w, r, r.URL.Query().Get("key"))
	if acct == nil {
		return
	}
	cfg, err := s.Store.GetWidgetConfig(acct.ID, acct.Company)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "config error")
		return
	}
	if !s.Cfg.Features.ProactiveTriggers {
		cfg.Proactive.Enabled = false
	}
	if !s.Cfg.Features.ExitIntent {
		cfg.ExitIntent = false
	}
	if !s.Cfg.Features.QuickReplies {
		cfg.QuickReplies = false
	}
	if !s.Cfg.Plan(acct.Plan).WhiteLabel {
		cfg.Branding = true
	}
	wsBase := strings.Replace(s.Cfg.PublicBaseURL, "https://", "wss://", 1)
	wsBase = strings.Replace(wsBase, "http://", "ws://", 1)
	writeJSON(w, http.StatusOK, map[string]any{
		"config":   cfg,
		"base_url": s.Cfg.PublicBaseURL,
		"ws_url":   wsBase + "/ws/widget",
		"disabled": s.quotaExceeded(acct),
	})
}

func (s *Server) handleWidgetSession(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Key          string `json:"key"`
		SessionToken string `json:"session_token"`
		VisitorID    string `json:"visitor_id"`
		PageURL      string `json:"page_url"`
		Trigger      string `json:"trigger"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	acct := s.widgetAccount(w, r, in.Key)
	if acct == nil {
		return
	}

	if in.SessionToken != "" {
		conv, err := s.Store.GetConversationByToken(in.SessionToken)
		if err == nil && conv.AccountID == acct.ID {
			msgs, _ := s.Store.ListMessages(conv.ID, 0)
			writeJSON(w, http.StatusOK, map[string]any{
				"session_token": conv.Token,
				"status":        conv.Status,
				"messages":      wire.Messages(msgs),
				"resumed":       true,
			})
			return
		}
	}

	if !s.Limiter.Allow("session:"+clientIP(r), 0.5, 10) {
		errJSON(w, http.StatusTooManyRequests, "rate limited")
		return
	}
	if s.quotaExceeded(acct) {
		writeJSON(w, http.StatusOK, map[string]any{"disabled": true})
		return
	}

	if len(in.VisitorID) > 64 {
		in.VisitorID = in.VisitorID[:64]
	}
	conv, err := s.Store.CreateConversation(acct.ID, in.VisitorID, in.PageURL)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "could not create session")
		return
	}
	cfg, _ := s.Store.GetWidgetConfig(acct.ID, acct.Company)
	greeting := cfg.Greeting
	if greeting == "" {
		greeting = "Hi there! How can we help you today?"
	}
	s.Store.InsertMessage(conv.ID, "assistant", greeting, "")
	if in.Trigger == "proactive" && cfg.Proactive.Message != "" && cfg.Proactive.Message != greeting {
		s.Store.InsertMessage(conv.ID, "assistant", cfg.Proactive.Message, "")
	}
	s.Store.InsertEvent(acct.ID, in.VisitorID, "conversation_started", in.PageURL)

	msgs, _ := s.Store.ListMessages(conv.ID, 0)
	fresh, _ := s.Store.GetConversationByID(acct.ID, conv.ID)
	if fresh != nil {
		s.Hub.ToAccount(acct.ID, map[string]any{"type": "conversation_updated", "conversation": wire.Conversation(fresh)})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"session_token": conv.Token,
		"status":        "active",
		"messages":      wire.Messages(msgs),
		"resumed":       false,
	})
}

func (s *Server) handleWidgetMessage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SessionToken string `json:"session_token"`
		Content      string `json:"content"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.SessionToken == "" || strings.TrimSpace(in.Content) == "" {
		errJSON(w, http.StatusBadRequest, "session_token and content required")
		return
	}
	if !s.Limiter.Allow("msg:"+in.SessionToken, 1, 5) {
		errJSON(w, http.StatusTooManyRequests, "slow down a little")
		return
	}
	conv, err := s.Store.GetConversationByToken(in.SessionToken)
	if err != nil {
		errJSON(w, http.StatusUnauthorized, "invalid session")
		return
	}
	go func(content string) {
		if _, err := s.Engine.ProcessVisitorMessage(context.Background(), conv, content); err != nil {
			log.Printf("widget: process error conv=%d: %v", conv.ID, err)
		}
	}(in.Content)
	writeJSON(w, http.StatusAccepted, map[string]bool{"ok": true})
}

func (s *Server) handleWidgetPoll(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("session_token")
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	if token == "" {
		errJSON(w, http.StatusBadRequest, "missing session_token")
		return
	}
	if !s.Limiter.Allow("poll:"+token, 2, 10) {
		errJSON(w, http.StatusTooManyRequests, "rate limited")
		return
	}
	conv, err := s.Store.GetConversationByToken(token)
	if err != nil {
		errJSON(w, http.StatusUnauthorized, "invalid session")
		return
	}
	msgs, err := s.Store.ListMessages(conv.ID, after)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "poll error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": wire.Messages(msgs), "status": conv.Status})
}

var allowedEvents = map[string]bool{"loaded": true, "opened": true, "closed": true, "proactive_shown": true, "exit_shown": true}

func (s *Server) handleWidgetEvent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Key       string `json:"key"`
		Type      string `json:"type"`
		VisitorID string `json:"visitor_id"`
		PageURL   string `json:"page_url"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if !allowedEvents[in.Type] {
		errJSON(w, http.StatusBadRequest, "unknown event type")
		return
	}
	if !s.Limiter.Allow("evt:"+clientIP(r), 2, 20) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	acct := s.widgetAccount(w, r, in.Key)
	if acct == nil {
		return
	}
	s.Store.InsertEvent(acct.ID, in.VisitorID, in.Type, in.PageURL)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleWidgetJS(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.Cfg.WidgetDir, "chat.js")
	b, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "widget script not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Write(b)
}

func (s *Server) handleWidgetDemo(w http.ResponseWriter, r *http.Request) {
	keyAttr := ""
	if key := r.URL.Query().Get("key"); key != "" {
		keyAttr = ` data-widget-key="` + html.EscapeString(key) + `"`
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Widget preview</title>
<style>body{font-family:system-ui,sans-serif;margin:0;background:#f4f4f7;color:#333}
.hero{max-width:720px;margin:60px auto;padding:0 24px}
h1{font-size:28px}p{line-height:1.6;color:#555}</style></head>
<body>
<div class="hero">
<h1>Your website (preview)</h1>
<p>This is a sample page with your chat widget installed. Click the bubble in the corner to try a conversation. Proactive and exit-intent triggers behave exactly as configured.</p>
<p>Messages here create real conversations in your dashboard — handy for testing routing too.</p>
</div>
<script src="%s/widget/chat.js"%s async></script>
</body></html>`, s.Cfg.PublicBaseURL, keyAttr)
}
