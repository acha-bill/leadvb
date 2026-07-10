package delivery

import (
	"encoding/json"
	"time"

	"leadqualifier/internal/models"
)

type TranscriptLine struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	At      string `json:"at"`
}

type LeadPayload struct {
	Event          string            `json:"event"`
	AccountID      int64             `json:"account_id"`
	Company        string            `json:"company"`
	ConversationID int64             `json:"conversation_id"`
	Status         string            `json:"status"`
	Score          *int64            `json:"score"`
	Confidence     *int64            `json:"confidence"`
	Contact        map[string]string `json:"contact"`
	Bant           map[string]*int   `json:"bant"`
	Summary        string            `json:"summary"`
	PageURL        string            `json:"page_url"`
	Language       string            `json:"language"`
	StartedAt      string            `json:"started_at"`
	EndedAt        string            `json:"ended_at"`
	DashboardURL   string            `json:"dashboard_url"`
	Transcript     []TranscriptLine  `json:"transcript"`
}

func BuildLeadPayload(event string, acct *models.Account, conv *models.Conversation, msgs []*models.Message, dashboardOrigin string) LeadPayload {
	p := LeadPayload{
		Event:          event,
		AccountID:      acct.ID,
		Company:        acct.Company,
		ConversationID: conv.ID,
		Status:         conv.Status,
		Contact:        map[string]string{},
		Bant:           map[string]*int{},
		PageURL:        conv.PageURL,
		Language:       conv.Language,
		StartedAt:      conv.StartedAt.UTC().Format(time.RFC3339),
		DashboardURL:   dashboardOrigin + "/conversations/" + itoa(conv.ID),
	}
	if p.Company == "" {
		p.Company = acct.Name
	}
	if conv.Score.Valid {
		v := conv.Score.Int64
		p.Score = &v
	}
	if conv.Confidence.Valid {
		v := conv.Confidence.Int64
		p.Confidence = &v
	}
	if conv.Summary.Valid {
		p.Summary = conv.Summary.String
	}
	if conv.EndedAt.Valid {
		p.EndedAt = conv.EndedAt.Time.UTC().Format(time.RFC3339)
	}
	if conv.Contact.Valid {
		json.Unmarshal([]byte(conv.Contact.String), &p.Contact)
	}
	if conv.Bant.Valid {
		json.Unmarshal([]byte(conv.Bant.String), &p.Bant)
	}
	for _, m := range msgs {
		if m.Role == "system" {
			continue
		}
		p.Transcript = append(p.Transcript, TranscriptLine{Role: m.Role, Content: m.Content, At: m.CreatedAt.UTC().Format(time.RFC3339)})
	}
	return p
}

func itoa(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
