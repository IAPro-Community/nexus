package nexus

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"

	"github.com/IAPro-Community/nexus/internal/profile"
)

// registeredProviderIDs returns the provider universe owned by Nexus profile
// registration. This is the same registry that feeds Usage/ListResources; it
// never discovers provider names from PATH, model output, prompts, or the web.
func registeredProviderIDs() ([]string, error) {
	profiles, err := profile.List()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, item := range profiles {
		if item.Disabled {
			continue
		}
		id := strings.ToLower(strings.TrimSpace(item.Provider))
		if normalizeRegisteredProviderID(id) == "" {
			continue
		}
		seen[id] = struct{}{}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

// ResolveRegisteredProviderPreference recognizes an explicitly named provider
// only when Nexus already owns at least one enabled profile for it. Actual
// authentication, health, quota and capability eligibility are intentionally
// left to ListResources/ResourceScheduler, which is the execution authority.
func (n *Nexus) ResolveRegisteredProviderPreference(prompt string) (string, bool, error) {
	ids, err := registeredProviderIDs()
	if err != nil {
		return "", false, err
	}
	provider, ok := resolveRegisteredProviderPreference(prompt, ids)
	return provider, ok, nil
}

func resolveRegisteredProviderPreference(prompt string, registered []string) (string, bool) {
	// A provider name is routing authority only when the user couples it to an
	// execution/delegation verb. Incidental prose such as "the Codex docs" must
	// never move the turn away from the current Lead.
	if !providerRoutingRequested(prompt) {
		return "", false
	}
	terms := providerPromptTerms(prompt)
	if len(terms) == 0 || len(registered) == 0 {
		return "", false
	}
	matched := map[string]struct{}{}
	for _, provider := range registered {
		canonical := normalizeRegisteredProviderID(provider)
		if canonical == "" {
			continue
		}
		if _, ok := terms[canonical]; ok {
			matched[strings.ToLower(strings.TrimSpace(provider))] = struct{}{}
		}
	}
	if len(matched) != 1 {
		// More than one named registered provider is an explicit comparison or
		// ambiguity, not permission for Nexus to invent a winner.
		return "", false
	}
	for provider := range matched {
		return provider, true
	}
	return "", false
}

func providerPromptTerms(prompt string) map[string]struct{} {
	raw := strings.FieldsFunc(strings.ToLower(prompt), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_'
	})
	terms := make(map[string]struct{}, len(raw)*2)
	normalized := make([]string, 0, len(raw))
	for _, token := range raw {
		token = normalizeRegisteredProviderID(token)
		if token == "" {
			continue
		}
		normalized = append(normalized, token)
		terms[token] = struct{}{}
	}
	// Accept common spaced spellings such as "open code" without maintaining
	// a second alias registry. Candidate existence is still registration-owned.
	for i := 0; i+1 < len(normalized); i++ {
		terms[normalized[i]+normalized[i+1]] = struct{}{}
	}
	return terms
}

func providerRoutingRequested(prompt string) bool {
	terms := providerPromptTerms(prompt)
	for _, verb := range []string{
		"chame", "chamar", "chama", "use", "usar", "usando", "rode", "rodar",
		"execute", "executar", "delegue", "delegar", "encaminhe", "encaminhar",
		"prefira", "preferir", "peça", "peca", "pedir", "mande", "mandar",
		"call", "ask", "run", "delegate", "route", "using", "via",
	} {
		if _, ok := terms[normalizeRegisteredProviderID(verb)]; ok {
			return true
		}
	}
	return false
}

func normalizeRegisteredProviderID(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// applyRegisteredProviderPreferenceToFlow projects an explicit user-owned
// provider request as a provider PIN. The scheduler may still balance between
// eligible profiles/accounts of that provider, but it must not silently switch
// to another provider. Existing explicit per-step resource decisions win.
func applyRegisteredProviderPreferenceToFlow(flow *FlowDefinition, provider string) {
	if flow == nil || strings.TrimSpace(provider) == "" {
		return
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	for i := range flow.Steps {
		step := &flow.Steps[i]
		canApply := strings.TrimSpace(step.Provider) == "" &&
			strings.TrimSpace(step.Profile) == "" &&
			strings.TrimSpace(step.DesiredProvider) == "" &&
			strings.TrimSpace(step.DesiredProfile) == ""
		if !canApply {
			continue
		}
		step.DesiredProvider = provider
		// MANUAL + DesiredProvider means provider PIN with profile/account
		// balancing still available inside that provider.
		step.ResourcePolicy = string(PolicyManual)
		var req TaskRequirements
		if strings.TrimSpace(step.TaskRequirements) != "" {
			_ = json.Unmarshal([]byte(step.TaskRequirements), &req)
		}
		if strings.TrimSpace(req.PreferProvider) == "" {
			req.PreferProvider = provider
		}
		if raw, err := json.Marshal(req); err == nil {
			step.TaskRequirements = string(raw)
		}
	}
}
