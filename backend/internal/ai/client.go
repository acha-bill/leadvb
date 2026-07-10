package ai

import (
	"context"
	"fmt"

	"leadqualifier/internal/config"
)

type Turn struct {
	Role    string
	Content string
}

type Provider interface {
	Complete(ctx context.Context, system string, turns []Turn) (string, error)
}

func NewProvider(cfg *config.Config) (Provider, error) {
	switch cfg.AIProvider {
	case "openai":
		if cfg.OpenAIKey == "" {
			return nil, fmt.Errorf("AI_PROVIDER=openai but OPENAI_API_KEY is empty")
		}
		return &OpenAI{APIKey: cfg.OpenAIKey, Model: cfg.OpenAIModel, BaseURL: cfg.OpenAIBaseURL}, nil
	case "anthropic":
		if cfg.AnthropicKey == "" {
			return nil, fmt.Errorf("AI_PROVIDER=anthropic but ANTHROPIC_API_KEY is empty")
		}
		return &Anthropic{APIKey: cfg.AnthropicKey, Model: cfg.AnthropicModel}, nil
	case "mock":
		return &Mock{}, nil
	default:
		return nil, fmt.Errorf("unknown AI_PROVIDER %q (use openai, anthropic or mock)", cfg.AIProvider)
	}
}
