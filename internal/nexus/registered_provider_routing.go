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
	tokens := providerPromptTokens(prompt)
	if len(tokens) == 0 || len(registered) == 0 {
		return "", false
	}
	matched := map[string]struct{}{}
	for _, provider := range registered {
		canonical := normalizeRegisteredProviderID(provider)
		if canonical == "" || !hasExplicitProviderRoutingIntent(tokens, canonical) {
			continue
		}
		matched[strings.ToLower(strings.TrimSpace(provider))] = struct{}{}
	}
	if len(matched) != 1 {
		// Multiple explicitly requested registered providers are ambiguous: Nexus
		// must not invent a winner. Incidental provider-name mentions do not count.
		return "", false
	}
	for provider := range matched {
		return provider, true
	}
	return "", false
}

func providerPromptTokens(prompt string) []string {
	raw := strings.FieldsFunc(strings.ToLower(prompt), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_'
	})
	tokens := make([]string, 0, len(raw))
	for _, token := range raw {
		token = normalizeRegisteredProviderID(token)
		if token != "" {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

func hasExplicitProviderRoutingIntent(tokens []string, provider string) bool {
	provider = normalizeRegisteredProviderID(provider)
	if provider == "" {
		return false
	}
	requestVerb := func(token string) bool {
		switch token {
		case "use", "usar", "usando", "utilize", "utilizar",
			"chame", "chamar", "chama", "call", "ask", "pergunte", "perguntar",
			"delegue", "delegar", "delegate", "route", "roteie", "rotear",
			"rode", "rodar", "execute", "executar":
			return true
		default:
			return false
		}
	}
	taskVerb := func(token string) bool {
		switch token {
		case "revise", "revisar", "review", "valide", "validar", "verify",
			"analise", "analisar", "investigue", "investigar", "audite", "auditar":
			return true
		default:
			return false
		}
	}

	for i := 0; i < len(tokens); i++ {
		spanEnd := i
		matched := tokens[i] == provider
		if !matched && i+1 < len(tokens) && tokens[i]+tokens[i+1] == provider {
			matched = true
			spanEnd = i + 1
		}
		if !matched {
			continue
		}

		start := i - 5
		if start < 0 {
			start = 0
		}
		end := spanEnd + 5
		if end >= len(tokens) {
			end = len(tokens) - 1
		}
		for j := start; j <= end; j++ {
			if requestVerb(tokens[j]) {
				return true
			}
		}

		// Natural forms such as "revise com codex" and "codex revise isso" are
		// explicit enough while a phrase such as "bug no cursor" is not.
		if i > 0 && (tokens[i-1] == "via" || tokens[i-1] == "com") {
			for j := start; j < i; j++ {
				if taskVerb(tokens[j]) {
					return true
				}
			}
		}
		if spanEnd+1 < len(tokens) && taskVerb(tokens[spanEnd+1]) {
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
