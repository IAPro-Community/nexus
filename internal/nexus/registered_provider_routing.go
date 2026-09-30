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
		if !explicitProviderRoutingRequested(prompt, provider) {
			// Mentioning a provider while discussing its adapter/configuration is
			// not the same as asking Nexus to execute that provider.
			return "", false
		}
		return provider, true
	}
	return "", false
}

func explicitProviderRoutingRequested(prompt, provider string) bool {
	words := strings.FieldsFunc(strings.ToLower(prompt), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	normalized := make([]string, 0, len(words))
	for _, word := range words {
		if token := normalizeRegisteredProviderID(word); token != "" {
			normalized = append(normalized, token)
		}
	}
	wanted := normalizeRegisteredProviderID(provider)
	if wanted == "" {
		return false
	}
	isVerb := func(token string) bool {
		switch token {
		case "use", "usar", "usa", "chame", "chamar", "call", "ask", "pergunte",
			"delegate", "delegue", "delegar", "route", "roteie", "rotear",
			"execute", "executar", "run", "rode", "rodar", "via":
			return true
		default:
			return false
		}
	}
	for i := range normalized {
		span := 1
		if normalized[i] != wanted {
			if i+1 >= len(normalized) || normalized[i]+normalized[i+1] != wanted {
				continue
			}
			span = 2
		}
		start := i - 3
		if start < 0 {
			start = 0
		}
		for j := start; j < i; j++ {
			if isVerb(normalized[j]) {
				return true
			}
		}
		if i > 0 && (normalized[i-1] == "com" || normalized[i-1] == "with") {
			return true
		}
		after := i + span
		if after < len(normalized) && (normalized[after] == "para" || normalized[after] == "to") {
			return true
		}
	}
	return false
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

func normalizeRegisteredProviderID(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// applyRegisteredProviderPreferenceToFlow projects an explicit provider request
// as PREFER, never PIN. ResourceScheduler may therefore fail over to another
// registered Usage resource when the requested provider is unavailable or out
// of quota. Explicit per-step provider/profile decisions remain authoritative.
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
		switch SchedulerPolicy(strings.ToUpper(strings.TrimSpace(step.ResourcePolicy))) {
		case PolicyManual:
			// MANUAL is a stronger explicit contract and must not be
			// silently weakened into fallback-capable preference.
		default:
			step.ResourcePolicy = string(PolicyPreferProvider)
		}
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
