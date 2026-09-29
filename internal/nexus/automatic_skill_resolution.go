package nexus

import (
	"context"
	"sort"
	"strings"
	"unicode"

	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
)

const maxAutomaticSkills = 4

type automaticSkillCandidate struct {
	ID    string
	Score int
}

// ResolveAutomaticSkillIDs maps a task to a bounded set of already-known Nexus
// skills. It never executes skill scripts and never creates a skill identity:
// every selected ID must resolve through the canonical project/user/builtin/
// optional-Maestro catalog.
//
// The order mirrors the v1 contract: classify -> metadata search -> candidates
// -> bounded load. Metadata-ranked skills are preferred, while canonical
// capability IDs provide a deterministic fallback when a catalog has sparse
// metadata.
func (n *Nexus) ResolveAutomaticSkillIDs(ctx context.Context, projectID, prompt string) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	catalog, err := n.skillCatalogForProject(projectID)
	if err != nil {
		return nil, err
	}
	req := ClassifyTaskRequirements(prompt)
	capabilities := automaticSkillCapabilities(prompt, req)
	ranked := rankAutomaticSkillCandidates(catalog, prompt, capabilities)

	selected := make([]string, 0, maxAutomaticSkills)
	seen := map[string]struct{}{}
	add := func(skill nexusskills.Skill) {
		if len(selected) >= maxAutomaticSkills || skill.Availability != nexusskills.AvailabilityAvailable {
			return
		}
		if _, duplicate := seen[skill.ID]; duplicate {
			return
		}
		seen[skill.ID] = struct{}{}
		selected = append(selected, skill.ID)
	}

	for _, candidate := range ranked {
		if len(selected) >= maxAutomaticSkills {
			break
		}
		if skill, ok := catalog.Resolve(candidate.ID); ok {
			add(skill)
		}
	}

	for _, capability := range capabilities {
		if len(selected) >= maxAutomaticSkills {
			break
		}
		capability = strings.ToLower(strings.TrimSpace(capability))
		if capability == "" {
			continue
		}
		// Prefer the canonical ID. A project-local skill with the same ID
		// already wins catalog source precedence over builtin/user copies.
		if skill, ok := catalog.Resolve(capability); ok {
			add(skill)
			continue
		}
		resolution, ok := catalog.ResolveCapability(capability)
		if !ok || strings.TrimSpace(resolution.Selected) == "" {
			continue
		}
		if skill, ok := catalog.Resolve(resolution.Selected); ok {
			add(skill)
		}
	}
	return selected, nil
}

func rankAutomaticSkillCandidates(catalog nexusskills.Catalog, prompt string, capabilities []string) []automaticSkillCandidate {
	promptTokens := automaticSkillTokens(prompt, true)
	lowerPrompt := strings.ToLower(strings.TrimSpace(prompt))
	capabilitySet := map[string]struct{}{}
	for _, capability := range capabilities {
		capabilitySet[normalizeCapabilityName(capability)] = struct{}{}
	}

	out := make([]automaticSkillCandidate, 0)
	for _, skill := range catalog.Library {
		if skill.Availability != nexusskills.AvailabilityAvailable {
			continue
		}
		score := 0
		strongSignal := false

		id := strings.ToLower(strings.TrimSpace(skill.ID))
		if id != "" && strings.Contains(lowerPrompt, id) {
			score += 180
			strongSignal = true
		}
		for _, trigger := range skill.Triggers {
			trigger = strings.ToLower(strings.TrimSpace(trigger))
			if len([]rune(trigger)) < 3 {
				continue
			}
			if strings.Contains(lowerPrompt, trigger) {
				score += 100
				strongSignal = true
			}
		}

		// ID/name matches are strong because they describe what the skill is.
		for token := range automaticSkillTokens(skill.ID+" "+skill.Name, false) {
			if _, ok := promptTokens[token]; ok {
				score += 15
				strongSignal = true
			}
		}

		// Description/trigger overlap is useful only with at least two distinct
		// signals; this prevents generic prose such as "...and review..." from
		// pulling an unrelated skill into the bounded context.
		descriptionOverlap := 0
		metadata := skill.Description + " " + strings.Join(skill.Triggers, " ")
		for token := range automaticSkillTokens(metadata, true) {
			if _, ok := promptTokens[token]; ok {
				descriptionOverlap++
			}
		}
		if descriptionOverlap >= 2 {
			score += descriptionOverlap * 3
			strongSignal = true
		}

		for _, capability := range skill.Capabilities {
			if _, ok := capabilitySet[normalizeCapabilityName(capability)]; ok {
				score += 70
				strongSignal = true
			}
		}
		for _, trigger := range skill.Triggers {
			if _, ok := capabilitySet[normalizeCapabilityName(trigger)]; ok {
				score += 50
				strongSignal = true
			}
		}

		if !strongSignal {
			continue
		}
		score += automaticSkillSourceBonus(skill.Source)
		out = append(out, automaticSkillCandidate{ID: skill.ID, Score: score})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func automaticSkillSourceBonus(source nexusskills.SourceID) int {
	switch source {
	case nexusskills.SourceProject:
		return 12
	case nexusskills.SourceUser:
		return 10
	case nexusskills.SourceMaestro:
		return 8
	case nexusskills.SourceBuiltin:
		return 4
	case nexusskills.SourceExternal:
		return 2
	default:
		return 0
	}
}

func automaticSkillTokens(value string, allowGenericSkill bool) map[string]struct{} {
	out := map[string]struct{}{}
	for _, raw := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		token := canonicalAutomaticSkillToken(raw)
		if token == "" || (!allowGenericSkill && token == "skill") {
			continue
		}
		switch token {
		case "the", "and", "for", "with", "from", "para", "com", "uma", "que", "por", "this", "that":
			continue
		}
		out[token] = struct{}{}
	}
	return out
}

func canonicalAutomaticSkillToken(token string) string {
	token = strings.ToLower(strings.TrimSpace(token))
	switch {
	case token == "":
		return ""
	case strings.HasPrefix(token, "revis"), token == "review", strings.HasPrefix(token, "evaluat"),
		strings.HasPrefix(token, "avali"), strings.HasPrefix(token, "audit"), strings.HasPrefix(token, "assess"):
		return "review"
	case strings.HasPrefix(token, "valid"), strings.HasPrefix(token, "verif"):
		return "verify"
	case strings.HasPrefix(token, "skill"):
		return "skill"
	case strings.HasPrefix(token, "test"):
		return "test"
	case strings.HasPrefix(token, "seguran"), strings.HasPrefix(token, "security"):
		return "security"
	case strings.HasPrefix(token, "arquitet"), strings.HasPrefix(token, "architect"):
		return "architecture"
	case strings.HasPrefix(token, "refator"), strings.HasPrefix(token, "refactor"):
		return "refactor"
	default:
		return token
	}
}

func automaticSkillCapabilities(prompt string, req TaskRequirements) []string {
	text := strings.ToLower(strings.TrimSpace(prompt))
	out := make([]string, 0, maxAutomaticSkills+2)
	add := func(value string) {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			return
		}
		for _, existing := range out {
			if existing == value {
				return
			}
		}
		out = append(out, value)
	}

	switch strings.ToLower(strings.TrimSpace(req.TaskKind)) {
	case "review":
		add("review")
	case "security":
		add("security")
	case "architecture":
		add("architecture")
	case "investigation":
		add("debugging")
	case "coding":
		add("coding")
	}

	for _, capability := range req.RequiredCapabilities {
		switch normalizeCapabilityName(capability) {
		case "code_review":
			add("review")
		case "root_cause_analysis":
			add("debugging")
		case "security":
			add("security")
		case "architecture":
			add("architecture")
		}
	}

	if containsAny(text, "validar", "valide", "validação", "validacao", "verify", "verification", "verificar") {
		add("verification")
	}
	if containsAny(text, "teste", "testes", "testing", "tests", "test") {
		add("testing")
	}
	if containsAny(text, "refactor", "refatore", "refatorar", "refatoração", "refatoracao") {
		add("refactoring")
	}
	if containsAny(text, "frontend", "front-end", "react", "ui") {
		add("frontend")
	}
	if containsAny(text, "backend", "back-end", "api", "servidor") {
		add("backend")
	}
	if containsAny(text, "deploy", "devops", "kubernetes", "k8s", "infraestrutura") {
		add("devops")
	}
	if containsAny(text, "segurança", "seguranca", "security") {
		add("security")
	}
	if containsAny(text, "arquitetura", "architecture") {
		add("architecture")
	}
	return out
}
