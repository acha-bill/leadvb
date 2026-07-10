package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAI struct {
	APIKey  string
	Model   string
	BaseURL string
}

type oaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (o *OpenAI) Complete(ctx context.Context, system string, turns []Turn) (string, error) {
	msgs := []oaMessage{{Role: "system", Content: system}}
	for _, t := range turns {
		msgs = append(msgs, oaMessage{Role: t.Role, Content: t.Content})
	}
	body := map[string]any{
		"model":           o.Model,
		"messages":        msgs,
		"temperature":     0.4,
		"max_tokens":      700,
		"response_format": map[string]string{"type": "json_object"},
	}
	b, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", o.BaseURL+"/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+o.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("openai status %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}
