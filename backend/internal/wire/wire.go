package wire

import (
	"encoding/json"
	"time"

	"leadqualifier/internal/models"
)

func Message(m *models.Message) map[string]any {
	var qr []string
	if m.QuickReplies.Valid && m.QuickReplies.String != "" {
		json.Unmarshal([]byte(m.QuickReplies.String), &qr)
	}
	return map[string]any{
		"id":            m.ID,
		"role":          m.Role,
		"content":       m.Content,
		"quick_replies": qr,
		"created_at":    m.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func Messages(ms []*models.Message) []map[string]any {
	out := make([]map[string]any, 0, len(ms))
	for _, m := range ms {
		out = append(out, Message(m))
	}
	return out
}

func Conversation(c *models.Conversation) map[string]any {
	out := map[string]any{
		"id":               c.ID,
		"token":            c.Token,
		"visitor_id":       c.VisitorID,
		"page_url":         c.PageURL,
		"status":           c.Status,
		"language":         c.Language,
		"message_count":    c.MessageCount,
		"started_at":       c.StartedAt.UTC().Format(time.RFC3339),
		"last_activity_at": c.LastActivityAt.UTC().Format(time.RFC3339),
	}
	if c.Score.Valid {
		out["score"] = c.Score.Int64
	} else {
		out["score"] = nil
	}
	if c.Confidence.Valid {
		out["confidence"] = c.Confidence.Int64
	} else {
		out["confidence"] = nil
	}
	if c.Summary.Valid {
		out["summary"] = c.Summary.String
	} else {
		out["summary"] = ""
	}
	if c.OverrideStatus.Valid {
		out["override_status"] = c.OverrideStatus.String
	} else {
		out["override_status"] = nil
	}
	if c.OverrideNote.Valid {
		out["override_note"] = c.OverrideNote.String
	} else {
		out["override_note"] = ""
	}
	if c.EndedAt.Valid {
		out["ended_at"] = c.EndedAt.Time.UTC().Format(time.RFC3339)
	} else {
		out["ended_at"] = nil
	}
	contact := map[string]string{}
	if c.Contact.Valid && c.Contact.String != "" {
		json.Unmarshal([]byte(c.Contact.String), &contact)
	}
	out["contact"] = contact
	bant := map[string]*int{}
	if c.Bant.Valid && c.Bant.String != "" {
		json.Unmarshal([]byte(c.Bant.String), &bant)
	}
	out["bant"] = bant
	return out
}
