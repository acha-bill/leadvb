package delivery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func SendSlack(webhookURL string, p LeadPayload) error {
	title := ":tada: New qualified lead"
	if p.Event == "handoff" {
		title = ":raising_hand: Visitor requested a human"
	}
	score := "—"
	if p.Score != nil {
		score = fmt.Sprintf("%d/100", *p.Score)
	}
	fields := []map[string]any{
		{"type": "mrkdwn", "text": "*Name:*\n" + orDash(p.Contact["name"])},
		{"type": "mrkdwn", "text": "*Email:*\n" + orDash(p.Contact["email"])},
		{"type": "mrkdwn", "text": "*Phone:*\n" + orDash(p.Contact["phone"])},
		{"type": "mrkdwn", "text": "*Score:*\n" + score},
	}
	transcript := ""
	max := len(p.Transcript)
	if max > 12 {
		max = 12
	}
	for _, t := range p.Transcript[:max] {
		who := "Visitor"
		if t.Role == "assistant" {
			who = "Bot"
		} else if t.Role == "owner" {
			who = "Owner"
		}
		transcript += fmt.Sprintf("*%s:* %s\n", who, t.Content)
	}
	if len(p.Transcript) > max {
		transcript += "_…transcript truncated, see dashboard_\n"
	}

	blocks := []map[string]any{
		{"type": "header", "text": map[string]any{"type": "plain_text", "text": title, "emoji": true}},
		{"type": "section", "fields": fields},
	}
	if p.Summary != "" {
		blocks = append(blocks, map[string]any{"type": "section", "text": map[string]any{"type": "mrkdwn", "text": "*Summary:* " + p.Summary}})
	}
	if transcript != "" {
		blocks = append(blocks, map[string]any{"type": "section", "text": map[string]any{"type": "mrkdwn", "text": transcript}})
	}
	blocks = append(blocks, map[string]any{"type": "section", "text": map[string]any{"type": "mrkdwn", "text": "<" + p.DashboardURL + "|Open in dashboard →>"}})

	body, _ := json.Marshal(map[string]any{"blocks": blocks})
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("slack status %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
