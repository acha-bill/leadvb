package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"leadqualifier/internal/config"
	"leadqualifier/internal/models"
	"leadqualifier/internal/store"
)

const maxAttempts = 5

type Worker struct {
	Store *store.Store
	Cfg   *config.Config
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick()
		}
	}
}

func (w *Worker) tick() {
	due, err := w.Store.FetchDueDeliveries(20)
	if err != nil {
		log.Printf("delivery: fetch error: %v", err)
		return
	}
	for _, d := range due {
		claimed, err := w.Store.ClaimDelivery(d.ID)
		if err != nil || !claimed {
			continue
		}
		if err := w.deliver(d); err != nil {
			log.Printf("delivery %d (%s/%s) attempt %d failed: %v", d.ID, d.Channel, d.Kind, d.Attempts+1, err)
			w.Store.MarkDeliveryFailed(d.ID, d.Attempts+1, err.Error(), maxAttempts)
		} else {
			w.Store.MarkDeliverySent(d.ID)
		}
	}
}

func (w *Worker) deliver(d *models.Delivery) error {
	switch d.Channel {
	case "email":
		var e struct {
			To      string `json:"to"`
			Subject string `json:"subject"`
			HTML    string `json:"html"`
		}
		if err := json.Unmarshal([]byte(d.Payload), &e); err != nil {
			return fmt.Errorf("bad payload: %w", err)
		}
		if e.To == "" {
			return fmt.Errorf("no recipient in payload")
		}
		return SendEmail(w.Cfg, e.To, e.Subject, e.HTML)
	case "slack":
		var wrap struct {
			WebhookURL string      `json:"webhook_url"`
			Lead       LeadPayload `json:"lead"`
		}
		if err := json.Unmarshal([]byte(d.Payload), &wrap); err != nil {
			return fmt.Errorf("bad payload: %w", err)
		}
		return SendSlack(wrap.WebhookURL, wrap.Lead)
	case "webhook":
		var wrap struct {
			URL    string          `json:"url"`
			Secret string          `json:"secret"`
			Lead   json.RawMessage `json:"lead"`
		}
		if err := json.Unmarshal([]byte(d.Payload), &wrap); err != nil {
			return fmt.Errorf("bad payload: %w", err)
		}
		return SendWebhook(wrap.URL, wrap.Secret, wrap.Lead, "lead."+d.Kind)
	default:
		return fmt.Errorf("unknown channel %s", d.Channel)
	}
}

func EnqueueLead(s *store.Store, cfg *config.Config, acct *models.Account, routing models.RoutingConfig, p LeadPayload, kind string) {
	convID := p.ConversationID
	plan := cfg.Plan(acct.Plan)

	if cfg.Features.EmailDelivery && routing.EmailEnabled && routing.EmailTo != "" {
		subject, body := LeadEmailHTML(p)
		payload := models.JSONString(map[string]any{"to": routing.EmailTo, "subject": subject, "html": body})
		s.EnqueueDelivery(acct.ID, &convID, "email", kind, payload)
	}
	if cfg.Features.SlackDelivery && routing.SlackEnabled && routing.SlackWebhookURL != "" {
		payload := models.JSONString(map[string]any{"webhook_url": routing.SlackWebhookURL, "lead": p})
		s.EnqueueDelivery(acct.ID, &convID, "slack", kind, payload)
	}
	if cfg.Features.WebhookDelivery && routing.WebhookEnabled && routing.WebhookURL != "" && plan.Webhooks {
		payload := models.JSONString(map[string]any{"url": routing.WebhookURL, "secret": routing.WebhookSecret, "lead": p})
		s.EnqueueDelivery(acct.ID, &convID, "webhook", kind, payload)
	}
}

func EnqueueSystemEmail(s *store.Store, accountID int64, to, subject, htmlBody, kind string) {
	payload := models.JSONString(map[string]any{"to": to, "subject": subject, "html": htmlBody})
	s.EnqueueDelivery(accountID, nil, "email", kind, payload)
}
