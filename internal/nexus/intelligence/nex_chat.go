package intelligence

import (
	"context"
	"errors"
	"strings"
)

// NexChatProvider is a narrow conversation capability independent of planning.
// It grants no execution or tool-use permissions to the model.
type NexChatProvider interface {
	NexReply(ctx context.Context, system, user string) (string, error)
}

// NexReply uses the configured OpenAI-compatible endpoint already available to
// Nexus Intelligence. No agent tools or function calls are sent to the model.
func (p *OpenAIIntelligenceProvider) NexReply(ctx context.Context, system, user string) (string, error) {
	if !p.Available(ctx) {
		return "", ErrIntelligenceUnavailable
	}
	body := map[string]any{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	}
	raw, err := p.doRequest(ctx, body)
	if err != nil {
		return "", err
	}
	reply := strings.TrimSpace(string(raw))
	if reply == "" {
		return "", errors.New("empty Nex answer")
	}
	if len(reply) > 6000 {
		reply = strings.ToValidUTF8(reply[:6000], "")
	}
	return reply, nil
}

// CLI intelligence currently uses coding-agent harnesses with their own tool
// permissions. Do not offer general Nex chat through a CLI until a verified
// read-only execution profile exists for each supported harness.
