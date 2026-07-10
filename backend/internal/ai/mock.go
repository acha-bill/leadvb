package ai

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
)

// Mock is a deterministic provider for local development without API keys.
type Mock struct{}

var emailRe = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
var moneyRe = regexp.MustCompile(`\$?\s?(\d[\d,]{2,})`)

func (m *Mock) Complete(_ context.Context, _ string, turns []Turn) (string, error) {
	assistantTurns := 0
	var allUser []string
	lastUser := ""
	for _, t := range turns {
		if t.Role == "assistant" {
			assistantTurns++
		} else {
			allUser = append(allUser, t.Content)
			lastUser = t.Content
		}
	}
	full := strings.ToLower(strings.Join(allUser, "\n"))

	out := map[string]any{
		"reply":                 "",
		"quick_replies":         []string{},
		"contact":               map[string]any{},
		"bant":                  map[string]any{},
		"confidence":            60,
		"conversation_complete": false,
		"recommendation":        "continue",
		"summary":               "",
		"language":              "en",
	}
	if email := emailRe.FindString(strings.Join(allUser, " ")); email != "" {
		out["contact"] = map[string]any{"email": email}
	}
	if strings.Contains(strings.ToLower(lastUser), "human") || strings.Contains(strings.ToLower(lastUser), "agent") {
		out["reply"] = "Of course — I'll flag this for the team and someone will jump in shortly."
		out["recommendation"] = "handoff"
		b, _ := json.Marshal(out)
		return string(b), nil
	}

	switch assistantTurns {
	case 0, 1:
		out["reply"] = "Great to meet you! What kind of business do you run, and what are you looking for help with?"
	case 2:
		out["reply"] = "That makes sense. Do you have a rough budget range in mind for this?"
		out["quick_replies"] = []string{"Under $1,000", "$1,000–$5,000", "$5,000+"}
	case 3:
		out["reply"] = "Got it. How soon are you hoping to get started?"
		out["quick_replies"] = []string{"ASAP", "Within 30 days", "Just researching"}
	case 4:
		out["reply"] = "Perfect. Are you the person who'd make the final decision on this?"
		out["quick_replies"] = []string{"Yes", "No, someone else"}
	case 5:
		out["reply"] = "Thanks! What's the best email to reach you, and your name?"
	default:
		hasEmail := emailRe.MatchString(full)
		hasBudget := moneyRe.MatchString(full) || strings.Contains(full, "budget") || strings.Contains(full, "$")
		urgent := strings.Contains(full, "asap") || strings.Contains(full, "30 day") || strings.Contains(full, "week")
		decision := strings.Contains(full, "yes")
		score := 40
		if hasBudget {
			score += 20
		}
		if urgent {
			score += 15
		}
		if decision {
			score += 15
		}
		if hasEmail {
			score += 10
		}
		bant := map[string]any{"budget": boolScore(hasBudget), "authority": boolScore(decision), "need": 70, "timeline": boolScore(urgent), "fit": score}
		out["bant"] = bant
		out["confidence"] = 75
		out["conversation_complete"] = true
		if score >= 70 {
			out["recommendation"] = "qualified"
			out["reply"] = "Thanks so much! You sound like a great fit — the team will reach out shortly to set up a quick call. 🎉"
		} else {
			out["recommendation"] = "disqualified"
			out["reply"] = "Thanks for chatting! It doesn't look like we're the right fit just now, but feel free to reach back out anytime."
		}
		out["summary"] = "Mock-qualified conversation for local development. Visitor discussed needs, budget, timeline and decision authority."
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

func boolScore(b bool) int {
	if b {
		return 80
	}
	return 35
}
