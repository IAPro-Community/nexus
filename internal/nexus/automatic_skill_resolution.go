package nexus

import (
	"context"
	"strings"

	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
)

const maxAutomaticSkills = 4

// ResolveAutomaticSkillIDs maps a task to a bounded set of already-known Nexus
// skills. It never executes skill scripts and never creates a skill identity:
// every selected ID must resolve through the canonical project/user/builtin/
// optional-Maestro catalog.
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
	capabilities := automaticSkillCapabilities(prompt, ClassifyTaskRequirements(prompt))
	selected := make([]string, 0, len(capabilities))
	seen := map[string]struct{}{}
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
		if skill, ok := catalog.Resolve(capability); ok && skill.Availability == nexusskills.AvailabilityAvailable {
			if _, duplicate := seen[skill.ID]; !duplicate {
				seen[skill.ID] = struct{}{}
				selected = append(selected, skill.ID)
			}
			continue
		}
		resolution, ok := catalog.ResolveCapability(capability)
		if !ok || strings.TrimSpace(resolution.Selected) == "" {
			continue
		}
		skill, ok := catalog.Resolve(resolution.Selected)
		if !ok || skill.Availability != nexusskills.AvailabilityAvailable {
			continue
		}
		if _, duplicate := seen[skill.ID]; duplicate {
			continue
		}
		seen[skill.ID] = struct{}{}
		selected = append(selected, skill.ID)
	}
	return selected, nil
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
		case "code-review":
			add("review")
		case "root-cause-analysis":
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
	if containsAny(text, "teste", "testes", "testing", "test ", "tests") {
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
