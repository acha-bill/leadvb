package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"leadqualifier/internal/config"
	"leadqualifier/internal/delivery"
	"leadqualifier/internal/models"
	"leadqualifier/internal/store"
	"leadqualifier/internal/wire"
)

type Broadcaster interface {
	ToConversation(convID int64, v any)
	ToAccount(accountID int64, v any)
}

type TurnResult struct {
	Reply        string            `json:"reply"`
	QuickReplies []string          `json:"quick_replies"`
	Contact      map[string]string `json:"contact"`
	Bant         map[string]*int   `json:"bant"`
	Confidence   *int              `json:"confidence"`
	Complete     bool              `json:"conversation_complete"`
	Recommend    string            `json:"recommendation"`
	Summary      string            `json:"summary"`
	Language     string            `json:"language"`
}

type Engine struct {
	Store    *store.Store
	Cfg      *config.Config
	Provider Provider
	Hub      Broadcaster

	locks sync.Map
}

func (e *Engine) lock(convID int64) func() {
	v, _ := e.locks.LoadOrStore(convID, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return func() {
		mu.Unlock()
		e.locks.Delete(convID)
	}
}

func (e *Engine) ProcessVisitorMessage(ctx context.Context, conv *models.Conversation, content string) (*models.Message, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, nil
	}
	if len(content) > 4000 {
		content = content[:4000]
	}

	visitorMsg, err := e.Store.InsertMessage(conv.ID, "visitor", content, "")
	if err != nil {
		return nil, err
	}
	e.Hub.ToConversation(conv.ID, map[string]any{"type": "message", "message": wire.Message(visitorMsg)})
	e.broadcastConvUpdate(conv.AccountID, conv.ID)

	if conv.Status != "active" {
		return nil, nil
	}

	unlock := e.lock(conv.ID)
	defer unlock()

	conv, err = e.Store.GetConversationByID(conv.AccountID, conv.ID)
	if err != nil {
		return nil, err
	}
	if conv.Status != "active" {
		return nil, nil
	}

	acct, err := e.Store.GetAccountByID(conv.AccountID)
	if err != nil {
		return nil, err
	}
	icp, err := e.Store.GetICP(conv.AccountID)
	if err != nil {
		return nil, err
	}
	widgetCfg, err := e.Store.GetWidgetConfig(conv.AccountID, acct.Company)
	if err != nil {
		return nil, err
	}
	routing, _ := e.Store.GetRoutingConfig(conv.AccountID)
	history, err := e.Store.ListMessages(conv.ID, 0)
	if err != nil {
		return nil, err
	}

	if conv.MessageCount >= e.Cfg.MaxMessagesPerConv {
		return e.finishOverLimit(acct, conv)
	}

	e.Hub.ToConversation(conv.ID, map[string]any{"type": "typing"})

	system := e.buildSystemPrompt(acct, icp, widgetCfg, routing, conv)
	turns := buildTurns(history)

	callCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	raw, err := e.Provider.Complete(callCtx, system, turns)
	if err != nil {
		log.Printf("ai: provider error conv=%d: %v", conv.ID, err)
		return e.sendFallback(acct, conv)
	}
	result, err := parseTurnResult(raw)
	if err != nil {
		retryTurns := append(turns, Turn{Role: "user", Content: "[system] Your previous output was not valid JSON. Respond again with ONLY the JSON object."})
		raw, err2 := e.Provider.Complete(callCtx, system, retryTurns)
		if err2 != nil {
			return e.sendFallback(acct, conv)
		}
		result, err = parseTurnResult(raw)
		if err != nil {
			log.Printf("ai: unparseable output conv=%d: %v", conv.ID, err)
			return e.sendFallback(acct, conv)
		}
	}

	return e.applyTurn(acct, conv, widgetCfg, routing, icp, result)
}

func (e *Engine) applyTurn(acct *models.Account, conv *models.Conversation, widgetCfg models.WidgetConfig, routing models.RoutingConfig, icp *models.ICP, r *TurnResult) (*models.Message, error) {
	contact := mergeContact(conv, r.Contact)
	bant := mergeBant(conv, r.Bant)
	score := computeScore(bant, icp.Weights)

	quickJSON := ""
	if e.Cfg.Features.QuickReplies && widgetCfg.QuickReplies && len(r.QuickReplies) > 0 {
		if len(r.QuickReplies) > 3 {
			r.QuickReplies = r.QuickReplies[:3]
		}
		quickJSON = models.JSONString(r.QuickReplies)
	}

	reply := strings.TrimSpace(r.Reply)
	if reply == "" {
		reply = "Could you tell me a bit more about that?"
	}

	status := "active"
	ended := false
	if r.Recommend == "handoff" && e.Cfg.Features.Handoff {
		status = "handoff"
	} else if r.Complete {
		qualified := false
		if score != nil {
			qualified = *score >= icp.Threshold
		} else {
			qualified = r.Recommend == "qualified"
		}
		if qualified {
			status = "qualified"
		} else {
			status = "disqualified"
		}
		ended = true
	}

	lang := conv.Language
	if r.Language != "" && len(r.Language) <= 12 {
		lang = r.Language
	}
	summary := r.Summary
	if summary == "" && conv.Summary.Valid {
		summary = conv.Summary.String
	}

	msg, err := e.Store.InsertMessage(conv.ID, "assistant", reply, quickJSON)
	if err != nil {
		return nil, err
	}
	if err := e.Store.UpdateConversationAI(conv.ID, status, score, models.JSONString(bant), models.JSONString(contact), summary, r.Confidence, lang, ended); err != nil {
		return nil, err
	}

	e.Hub.ToConversation(conv.ID, map[string]any{"type": "message", "message": wire.Message(msg)})
	if status != "active" {
		e.Hub.ToConversation(conv.ID, map[string]any{"type": "status", "status": status})
	}
	e.broadcastConvUpdate(conv.AccountID, conv.ID)

	if status == "qualified" || status == "handoff" {
		fresh, err := e.Store.GetConversationByID(conv.AccountID, conv.ID)
		if err == nil {
			msgs, _ := e.Store.ListMessages(conv.ID, 0)
			kind := "qualified"
			event := "lead.qualified"
			if status == "handoff" {
				kind = "handoff"
				event = "handoff"
				if !routing.NotifyHandoff {
					return msg, nil
				}
			}
			payload := delivery.BuildLeadPayload(event, acct, fresh, msgs, e.Cfg.DashboardOrigin)
			delivery.EnqueueLead(e.Store, e.Cfg, acct, routing, payload, kind)
		}
	}
	return msg, nil
}

func (e *Engine) sendFallback(acct *models.Account, conv *models.Conversation) (*models.Message, error) {
	msg, err := e.Store.InsertMessage(conv.ID, "assistant",
		"Sorry — I'm having a little trouble right now. Could you leave your name and email and the team will follow up shortly?", "")
	if err != nil {
		return nil, err
	}
	e.Hub.ToConversation(conv.ID, map[string]any{"type": "message", "message": wire.Message(msg)})
	e.broadcastConvUpdate(conv.AccountID, conv.ID)
	return msg, nil
}

func (e *Engine) finishOverLimit(acct *models.Account, conv *models.Conversation) (*models.Message, error) {
	msg, err := e.Store.InsertMessage(conv.ID, "assistant",
		"Thanks for the chat! I'll pass everything along to the team — they'll follow up if it's a fit.", "")
	if err != nil {
		return nil, err
	}
	e.Store.SetConversationStatus(conv.ID, "disqualified", true)
	e.Hub.ToConversation(conv.ID, map[string]any{"type": "message", "message": wire.Message(msg)})
	e.Hub.ToConversation(conv.ID, map[string]any{"type": "status", "status": "disqualified"})
	e.broadcastConvUpdate(conv.AccountID, conv.ID)
	return msg, nil
}

func (e *Engine) broadcastConvUpdate(accountID, convID int64) {
	c, err := e.Store.GetConversationByID(accountID, convID)
	if err != nil {
		return
	}
	e.Hub.ToAccount(accountID, map[string]any{"type": "conversation_updated", "conversation": wire.Conversation(c)})
}

func (e *Engine) buildSystemPrompt(acct *models.Account, icp *models.ICP, w models.WidgetConfig, routing models.RoutingConfig, conv *models.Conversation) string {
	company := w.CompanyName
	if company == "" {
		company = acct.Company
	}
	if company == "" {
		company = acct.Name
	}

	icpText := strings.TrimSpace(icp.Description)
	if icpText == "" {
		icpText = "No ICP provided. Qualify generally on budget, authority, need and timeline for a small B2B service business."
	}

	langRule := "Reply in English."
	if e.Cfg.Features.MultiLanguage {
		if w.Language == "auto" || w.Language == "" {
			langRule = "Reply in the visitor's language (detect it from their messages)."
		} else {
			langRule = "Reply in language: " + w.Language + "."
		}
	}

	disq := "wrap up politely."
	if routing.Disqualified.Mode == "newsletter" && routing.Disqualified.NewsletterURL != "" {
		disq = "wrap up politely and suggest they join the newsletter at " + routing.Disqualified.NewsletterURL + "."
	}

	quickRule := "Offer quick_replies (max 3, under 25 chars each) when the question has obvious short answers, e.g. budget ranges or yes/no."
	if !e.Cfg.Features.QuickReplies || !w.QuickReplies {
		quickRule = "Always return an empty quick_replies array."
	}

	feedbackSection := ""
	if examples, err := e.Store.RecentFeedback(acct.ID, 5); err == nil && len(examples) > 0 {
		var sb strings.Builder
		sb.WriteString("## Owner corrections on past conversations (learn from these)\n")
		for _, ex := range examples {
			sb.WriteString(fmt.Sprintf("- AI said %s, owner corrected to %s.", ex.OriginalStatus, ex.CorrectedStatus))
			if ex.Note != "" {
				sb.WriteString(" Owner note: " + ex.Note)
			}
			if ex.Summary != "" {
				sb.WriteString(" (conversation: " + truncateStr(ex.Summary, 160) + ")")
			}
			sb.WriteString("\n")
		}
		feedbackSection = sb.String()
	}

	contactJSON := "{}"
	if conv.Contact.Valid && conv.Contact.String != "" {
		contactJSON = conv.Contact.String
	}
	bantJSON := "{}"
	if conv.Bant.Valid && conv.Bant.String != "" {
		bantJSON = conv.Bant.String
	}

	return fmt.Sprintf(`You are a friendly, concise lead-qualification assistant chatting with a website visitor on behalf of %s.

## Ideal customer profile (written by the business owner)
%s

## Your job
1. Engage naturally like a helpful human rep. Keep replies under 60 words and ask ONE question at a time.
2. Qualify the visitor against the profile: budget, authority (are they the decision maker?), need, timeline, plus any profile-specific criteria.
3. Collect contact details naturally during the conversation: name and email (email is required before completing a qualified lead), phone/company if offered.
4. Answer basic questions about the business honestly. If you don't know something, say the team will follow up — NEVER invent facts, prices or promises.
5. %s
6. Once you have enough signal (typically 4-8 exchanges) set conversation_complete=true and give a recommendation.
7. If the visitor asks for a human, is frustrated, or you cannot make progress, set recommendation="handoff".
8. If the visitor is clearly not a fit, %s Set recommendation="disqualified".

## Scoring
Rate each 0-100, or null when still unknown: budget, authority, need, timeline, fit (overall profile match). confidence = how certain you are overall (0-100).

%s
## State so far
Known contact: %s
Known scores: %s

## Output format — respond with STRICT JSON only, no prose or fences:
{"reply": "your next message", "quick_replies": ["..."], "contact": {"name": null, "email": null, "phone": null, "company": null}, "bant": {"budget": null, "authority": null, "need": null, "timeline": null, "fit": null}, "confidence": 0, "conversation_complete": false, "recommendation": "continue", "summary": "", "language": "en"}
recommendation must be one of: continue, qualified, disqualified, handoff.
summary: 2-3 sentence recap of who they are and what they need (fill when conversation_complete=true).
%s`, company, icpText, langRule, disq, feedbackSection, contactJSON, bantJSON, quickRule)
}

func buildTurns(history []*models.Message) []Turn {
	turns := make([]Turn, 0, len(history))
	for _, m := range history {
		switch m.Role {
		case "visitor":
			turns = append(turns, Turn{Role: "user", Content: m.Content})
		case "assistant":
			turns = append(turns, Turn{Role: "assistant", Content: m.Content})
		case "owner":
			turns = append(turns, Turn{Role: "assistant", Content: "[Human agent] " + m.Content})
		}
	}
	if len(turns) > 40 {
		turns = turns[len(turns)-40:]
	}
	merged := make([]Turn, 0, len(turns))
	for _, t := range turns {
		if n := len(merged); n > 0 && merged[n-1].Role == t.Role {
			merged[n-1].Content += "\n" + t.Content
			continue
		}
		merged = append(merged, t)
	}
	if len(merged) == 0 || merged[0].Role != "user" {
		merged = append([]Turn{{Role: "user", Content: "[The visitor just opened the chat widget]"}}, merged...)
	}
	return merged
}

func parseTurnResult(raw string) (*TurnResult, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON object in output")
	}
	var r TurnResult
	if err := json.Unmarshal([]byte(raw[start:end+1]), &r); err != nil {
		return nil, err
	}
	switch r.Recommend {
	case "continue", "qualified", "disqualified", "handoff":
	default:
		r.Recommend = "continue"
	}
	return &r, nil
}

func mergeContact(conv *models.Conversation, update map[string]string) map[string]string {
	out := map[string]string{}
	if conv.Contact.Valid && conv.Contact.String != "" {
		json.Unmarshal([]byte(conv.Contact.String), &out)
	}
	for k, v := range update {
		v = strings.TrimSpace(v)
		if v != "" && strings.ToLower(v) != "null" {
			out[k] = v
		}
	}
	return out
}

func mergeBant(conv *models.Conversation, update map[string]*int) map[string]*int {
	out := map[string]*int{}
	if conv.Bant.Valid && conv.Bant.String != "" {
		json.Unmarshal([]byte(conv.Bant.String), &out)
	}
	for k, v := range update {
		if v != nil {
			out[k] = v
		}
	}
	return out
}

func computeScore(bant map[string]*int, w models.Weights) *int {
	weights := map[string]int{"budget": w.Budget, "authority": w.Authority, "need": w.Need, "timeline": w.Timeline, "fit": w.Fit}
	sum, wsum := 0, 0
	for k, weight := range weights {
		if v, ok := bant[k]; ok && v != nil {
			val := *v
			if val < 0 {
				val = 0
			}
			if val > 100 {
				val = 100
			}
			sum += val * weight
			wsum += weight
		}
	}
	if wsum == 0 {
		return nil
	}
	score := sum / wsum
	return &score
}

func truncateStr(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
