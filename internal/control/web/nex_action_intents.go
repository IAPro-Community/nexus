package web

import (
	"strings"
)

// NexActionProposal is not a tool call. It is a typed, user-confirmable
// navigation intent into an existing Nexus screen. The model has no authority
// to synthesize IDs or execute commands based on its generated text.
type NexActionProposal struct {
	ID                   string `json:"id"`
	Title                string `json:"title"`
	Description          string `json:"description"`
	Target               string `json:"target"`
	Risk                 string `json:"risk"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
}

var nexSafeNavigation = map[string]NexActionProposal{
	"agentes": {
		ID: "nexus.open.agents", Title: "Abrir agentes",
		Description: "Navegar para a lista de agentes; nenhuma execução será iniciada.",
		Target: "agents", Risk: "navigation", RequiresConfirmation: true,
	},
	"missoes": {
		ID: "nexus.open.missions", Title: "Abrir missões",
		Description: "Navegar para as missões existentes; não inicia nem cancela trabalho.",
		Target: "missions", Risk: "navigation", RequiresConfirmation: true,
	},
	"alertas": {
		ID: "nexus.open.usage", Title: "Examinar alertas",
		Description: "Abrir as informações de recursos e alertas existentes.",
		Target: "usage", Risk: "navigation", RequiresConfirmation: true,
	},
	"configuracoes": {
		ID: "nexus.open.settings", Title: "Abrir configurações",
		Description: "Abrir configurações para revisão; não modifica preferências.",
		Target: "settings", Risk: "navigation", RequiresConfirmation: true,
	},
	"novo-agente": {
		ID: "nexus.open.new-agent", Title: "Preparar novo agente",
		Description: "Abrir o cadastro de agente, sem executar nem salvar automaticamente.",
		Target: "new-agent", Risk: "navigation", RequiresConfirmation: true,
	},
	"nexus": {
		ID: "nexus.open.home", Title: "Abrir Nexus",
		Description: "Navegar para o Nexus sem iniciar qualquer operação.",
		Target: "nexus", Risk: "navigation", RequiresConfirmation: true,
	},
}

func normalizeNexIntent(input string) string {
	n := strings.ToLower(strings.TrimSpace(input))
	n = strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a",
		"é", "e", "ê", "e", "í", "i",
		"ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c",
		"?", " ", "!", " ", ".", " ", ",", " ",
	).Replace(n)
	return strings.Join(strings.Fields(n), " ")
}

// resolveNexActionIntent deliberately recognizes only exact, reviewed
// navigation phrases. Arbitrary shell code, prompts or longer natural text
// are never interpreted as an execution request.
func resolveNexActionIntent(message string) (NexActionProposal, bool) {
	n := normalizeNexIntent(message)
	key := ""
	switch n {
	case "abrir agentes", "abra agentes", "mostrar agentes", "mostre agentes", "ver agentes", "meus agentes":
		key = "agentes"
	case "abrir missoes", "abra missoes", "mostrar missoes", "mostre missoes", "ver missoes",
		"abrir execucoes", "abra execucoes", "mostrar execucoes", "ver execucoes":
		key = "missoes"
	case "abrir alertas", "mostrar alertas", "mostre alertas", "ver alertas", "ver recursos":
		key = "alertas"
	case "abrir configuracoes", "abra configuracoes", "mostrar configuracoes", "ver configuracoes":
		key = "configuracoes"
	case "criar agente", "novo agente", "cadastrar agente", "adicionar agente":
		key = "novo-agente"
	case "abrir nexus", "abra nexus", "abrir inicio", "voltar ao nexus":
		key = "nexus"
	}
	proposal, exists := nexSafeNavigation[key]
	return proposal, exists
}
