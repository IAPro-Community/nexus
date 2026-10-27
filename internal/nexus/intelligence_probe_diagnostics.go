package nexus

import (
	"context"
	"errors"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/classifier"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
)

// IntelligenceProbeFailureReason intentionally returns a bounded classification,
// never provider stdout/stderr, paths, API response bodies or credentials.
// It is advisory: the real provider probe remains the authority.
func IntelligenceProbeFailureReason(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, intelligence.ErrIntelligenceUnavailable) {
		return "not_configured"
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "timeout"
	}
	failure := classifier.Classify(err, "")
	switch failure.Kind {
	case model.FailureRateLimit, model.FailureQuota:
		return "rate_limit"
	case model.FailureAuth:
		return "authentication"
	case model.FailureNetwork:
		return "network"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "401"), strings.Contains(msg, "403"),
		strings.Contains(msg, "unauthorized"), strings.Contains(msg, "forbidden"),
		strings.Contains(msg, "invalid api key"), strings.Contains(msg, "authentication"),
		strings.Contains(msg, "not logged in"), strings.Contains(msg, "login required"):
		return "authentication"
	case strings.Contains(msg, "model not found"), strings.Contains(msg, "model_not_found"),
		strings.Contains(msg, "model does not exist"), strings.Contains(msg, "unknown model"):
		return "model"
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "timed out"),
		strings.Contains(msg, "deadline exceeded"):
		return "timeout"
	case strings.Contains(msg, "connection refused"), strings.Contains(msg, "no such host"),
		strings.Contains(msg, "network is unreachable"), strings.Contains(msg, "could not resolve"):
		return "network"
	case strings.Contains(msg, "decode "), strings.Contains(msg, "invalid json"),
		strings.Contains(msg, "malformed chat response"), strings.Contains(msg, "extract json"):
		return "response_format"
	default:
		return "provider_failure"
	}
}

// IntelligenceProbeFailureMessage gives an actionable error without leaking
// untrusted or potentially secret-containing provider diagnostics.
func IntelligenceProbeFailureMessage(err error) string {
	switch IntelligenceProbeFailureReason(err) {
	case "not_configured":
		return "Nenhuma IA compatível está disponível neste projeto. Configure um provedor e perfil em Nexus Intelligence."
	case "authentication":
		return "O provedor de IA recusou a autenticação. Verifique login do CLI, perfil ou credenciais em Nexus Intelligence."
	case "rate_limit":
		return "O provedor de IA atingiu seu limite de uso ou cota. Verifique o limite ou escolha outro modelo."
	case "model":
		return "O modelo selecionado não está disponível. Revise o modelo configurado em Nexus Intelligence."
	case "timeout":
		return "O provedor não respondeu dentro do limite. Verifique o CLI ou servidor do modelo e tente novamente."
	case "network":
		return "Não foi possível conectar ao provedor de IA. Verifique rede, servidor local ou URL do serviço."
	case "response_format":
		return "O provedor respondeu, mas não entregou o JSON esperado pelo Composer. Verifique o modelo ou compatibilidade do CLI."
	default:
		return "Não foi possível concluir o teste real da IA. Verifique o provedor, perfil, modelo e conectividade em Nexus Intelligence."
	}
}
