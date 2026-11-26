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
func (n *Nexus) ResolveRegisteredProviderPreference(prompt string, currentProvider ...string) (string, bool, error) {
	ids, err := registeredProviderIDs()
	if err != nil {
		return "", false, err
	}
	if len(currentProvider) > 0 && strings.TrimSpace(currentProvider[0]) != "" {
		filtered := ids[:0]
		for _, id := range ids {
			if !strings.EqualFold(id, strings.TrimSpace(currentProvider[0])) {
				filtered = append(filtered, id)
			}
		}
		ids = filtered
	}
	provider, found := resolveRegisteredProviderPreference(prompt, ids)
	return provider, found, nil
}

// ResolveExplicitProviderDelegate recognizes a provider named in a concrete
// routing instruction. It does not authorize Mission creation; registration,
// runtime permission and resource eligibility are checked by the delegate.
func (n *Nexus) ResolveExplicitProviderDelegate(prompt string, currentProvider ...string) (string, bool, error) {
	if n == nil || !providerRoutingRequested(prompt) {
		return "", false, nil
	}
	provider, found, err := n.ResolveRegisteredProviderPreference(prompt, currentProvider...)
	if err != nil || !found {
		return "", false, err
	}
	return provider, true, nil
}

func resolveRegisteredProviderPreference(prompt string, registered []string) (string, bool) {
	tokens := providerPromptTokens(prompt)
	if len(tokens) == 0 || len(registered) == 0 {
		return "", false
	}
	matched := map[string]struct{}{}
	for _, provider := range registered {
		canonical := normalizeRegisteredProviderID(provider)
		if canonical == "" {
			continue
		}
		if providerBoundToRoutingAction(tokens, canonical) {
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

func providerBoundToRoutingAction(tokens []string, provider string) bool {
	for providerStart := range tokens {
		providerEnd := providerStart
		if tokens[providerStart] != provider {
			if providerStart+1 >= len(tokens) || tokens[providerStart]+tokens[providerStart+1] != provider {
				continue
			}
			providerEnd++
		}
		for action := range tokens {
			if action >= providerStart || !isProviderRoutingVerb(tokens[action]) || providerRequestNegated(tokens, action) || providerRequestIsDescriptive(tokens, action) {
				continue
			}
			// A provider mention only routes when the request names it as the
			// destination ("delegate ... to Cursor") or gives it a task
			// ("use Cursor to review"). General proximity between an action
			// elsewhere in the sentence and a provider name is not intent.
			if providerRoutingDestination(tokens, providerStart) && providerStart-action <= 7 {
				return true
			}
			if providerRoutingPurpose(tokens, providerEnd) && providerStart-action <= 3 {
				return true
			}
		}
	}
	return false
}

func providerRequestIsDescriptive(tokens []string, action int) bool {
	if action == 0 {
		return false
	}
	switch tokens[action-1] {
	case "i", "we", "eu", "nos", "nós", "vou", "vamos", "irei", "iremos":
		return true
	}
	if tokens[action] != "use" && tokens[action] != "usar" {
		return false
	}
	// First-person tool-choice statements such as “I want to use Cursor” are
	// descriptive. A second-person subject keeps explicit instructions such as
	// “I want you to use Cursor” routable.
	start := action - 5
	if start < 0 {
		start = 0
	}
	for subject := start; subject < action; subject++ {
		if !isFirstPersonProviderSubject(tokens[subject]) {
			continue
		}
		selfIntent, addressedToAgent := false, false
		for i := subject + 1; i < action; i++ {
			switch tokens[i] {
			case "you", "your", "tu", "voce", "você", "voces", "vocês":
				addressedToAgent = true
			case "want", "need", "plan", "intend", "hope", "like", "prefer", "will", "would", "could", "can", "should", "going", "m", "am",
				"quero", "preciso", "planejo", "pretendo", "gostaria", "posso", "poderia", "devo", "vou":
				selfIntent = true
			}
		}
		if selfIntent && !addressedToAgent {
			return true
		}
	}
	return false
}

func isFirstPersonProviderSubject(token string) bool {
	switch token {
	case "i", "we", "eu", "nos", "nós":
		return true
	default:
		return false
	}
}

func providerRoutingDestination(tokens []string, providerStart int) bool {
	if providerStart > 0 && isProviderDestinationPreposition(tokens[providerStart-1]) {
		return true
	}
	return providerStart > 1 && isProviderArticle(tokens[providerStart-1]) && isProviderDestinationPreposition(tokens[providerStart-2])
}

func providerRoutingPurpose(tokens []string, providerEnd int) bool {
	return providerEnd+1 < len(tokens) && isProviderDestinationPreposition(tokens[providerEnd+1])
}

func isProviderDestinationPreposition(token string) bool {
	switch token {
	case "a", "ao", "para", "pra", "to", "for", "via":
		return true
	default:
		return false
	}
}

func isProviderArticle(token string) bool {
	switch token {
	case "o", "a", "the":
		return true
	default:
		return false
	}
}

func isProviderRoutingVerb(token string) bool {
	switch token {
	case "chame", "chamar", "chama", "use", "usar", "rode", "rodar",
		"execute", "executar", "delegue", "delegar", "encaminhe", "encaminhar",
		"prefira", "preferir", "peca", "peça", "pedir", "mande", "mandar",
		"call", "ask", "run", "delegate", "route":
		return true
	default:
		return false
	}
}

func providerRequestNegated(tokens []string, action int) bool {
	start := action - 2
	if start < 0 {
		start = 0
	}
	for i := start; i < action; i++ {
		token := tokens[i]
		switch token {
		case "not", "never", "dont", "não", "nao", "nunca", "jamais":
			return true
		case "don":
			if i+1 < action && tokens[i+1] == "t" {
				return true
			}
		}
	}
	return false
}

func providerPromptTerms(prompt string) map[string]struct{} {
	raw := providerPromptTokens(prompt)
	terms := make(map[string]struct{}, len(raw)*2)
	for _, token := range raw {
		terms[token] = struct{}{}
	}
	// Accept common spaced spellings such as "open code" without maintaining
	// a second alias registry. Candidate existence is still registration-owned.
	for i := 0; i+1 < len(raw); i++ {
		terms[raw[i]+raw[i+1]] = struct{}{}
	}
	return terms
}

func providerPromptTokens(prompt string) []string {
	raw := strings.FieldsFunc(strings.ToLower(prompt), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_'
	})
	normalized := make([]string, 0, len(raw))
	for _, token := range raw {
		token = normalizeRegisteredProviderID(token)
		if token == "" {
			continue
		}
		normalized = append(normalized, token)
	}
	return normalized
}

func providerRoutingRequested(prompt string) bool {
	terms := providerPromptTerms(prompt)
	for _, verb := range []string{
		"chame", "chamar", "chama", "use", "usar", "rode", "rodar",
		"execute", "executar", "delegue", "delegar", "encaminhe", "encaminhar",
		"prefira", "preferir", "peça", "peca", "pedir", "mande", "mandar",
		"call", "ask", "run", "delegate", "route",
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
