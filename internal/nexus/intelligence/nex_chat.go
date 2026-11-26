package intelligence

import (
	"context"
	"errors"
	"strings"
)

// ErrNexChatRequiresReadOnly is returned when a CLI provider does not have a
// verified read-only no-network execution contract for Nex conversations.
var ErrNexChatRequiresReadOnly = errors.New("cli provider lacks verified read-only sandbox for Nex chat")

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

// NexReply runs a narrow read-only conversation prompt when and only when the
// CLI provider was constructed with a verified read-only sandbox runner.
func (p *CLIProvider) NexReply(ctx context.Context, system, user string) (string, error) {
	if !p.Available(ctx) {
		return "", ErrIntelligenceUnavailable
	}
	if p.readOnlyRun == nil {
		return "", ErrNexChatRequiresReadOnly
	}
	prompt := strings.TrimSpace(system) + "\n\nContext & User Request:\n" + strings.TrimSpace(user)
	out, err := p.readOnlyRun(ctx, prompt)
	if err != nil {
		return "", err
	}
	reply := strings.TrimSpace(out)
	if reply == "" {
		return "", errors.New("empty Nex answer")
	}
	if len(reply) > 6000 {
		reply = strings.ToValidUTF8(reply[:6000], "")
	}
	return reply, nil
}
