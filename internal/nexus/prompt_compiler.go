package nexus

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
)

type CompiledAgentPrompt struct {
	CompiledPrompt  string   `json:"compiled_prompt"`
	PromptHash      string   `json:"prompt_hash"`
	ValidatedSkills []string `json:"validated_skills"`
	SkillContracts  []string `json:"skill_contracts,omitempty"`
}

// CompileAgentPromptWithValidatedSkills constructs the compiled prompt envelope with already validated skills.
func CompileAgentPromptWithValidatedSkills(userPrompt string, validatedSkills []string) *CompiledAgentPrompt {
	return CompileAgentPromptWithContracts(userPrompt, validatedSkills, nil)
}

// CompileAgentPromptWithContracts embeds the complete selected generic Skill
// contracts. IDs alone are insufficient because the receiving Agent may not
// share the same local catalog.
func CompileAgentPromptWithContracts(userPrompt string, validatedSkills []string, contracts []nexusskills.Skill) *CompiledAgentPrompt {
	var compiled string
	if len(validatedSkills) == 0 {
		compiled = userPrompt
	} else {
		var b strings.Builder
		b.WriteString("Nexus execution context\n")
		b.WriteString("Scope: next prompt only\n")
		b.WriteString("Validated skills:\n")
		for _, s := range validatedSkills {
			b.WriteString("- " + s + "\n")
		}
		for _, skill := range contracts {
			if strings.TrimSpace(skill.Instructions) == "" {
				continue
			}
			b.WriteString("\n--- Skill contract: " + skill.ID + " (" + string(skill.Source) + ") ---\n")
			b.WriteString(skill.Instructions)
			b.WriteString("\n--- End skill contract ---\n")
		}
		b.WriteString("\nUser request:\n")
		b.WriteString(userPrompt)
		compiled = b.String()
	}

	h := sha256.Sum256([]byte(compiled))
	hashStr := hex.EncodeToString(h[:])

	return &CompiledAgentPrompt{
		CompiledPrompt:  compiled,
		PromptHash:      hashStr,
		ValidatedSkills: validatedSkills,
		SkillContracts:  contractIDs(contracts),
	}
}

// CompileAgentPromptWithSkillContracts embeds source-agnostic selected Skills.
// This is the native path used by Nexus execution; Maestro is only one
// optional source and is not required for validation or prompt compilation.
func CompileAgentPromptWithSkillContracts(userPrompt string, selected []nexusskills.Skill) *CompiledAgentPrompt {
	ids := make([]string, 0, len(selected))
	var b strings.Builder
	b.WriteString("Nexus execution context\n")
	b.WriteString("Scope: next prompt only\n")
	b.WriteString("Validated Nexus skills:\n")
	for _, skill := range selected {
		ids = append(ids, skill.ID)
		b.WriteString("- ")
		b.WriteString(skill.ID)
		b.WriteString(" (source: ")
		b.WriteString(string(skill.Source))
		b.WriteString(", version: ")
		b.WriteString(firstNonEmpty(skill.Version, "unknown"))
		b.WriteString(")\n")
		if strings.TrimSpace(skill.Instructions) == "" {
			continue
		}
		b.WriteString("\n--- Skill contract: ")
		b.WriteString(skill.ID)
		b.WriteString(" ---\n")
		b.WriteString(skill.Instructions)
		b.WriteString("\n--- End skill contract ---\n")
	}
	b.WriteString("\nUser request:\n")
	b.WriteString(strings.TrimSpace(userPrompt))
	compiled := b.String()
	h := sha256.Sum256([]byte(compiled))
	return &CompiledAgentPrompt{
		CompiledPrompt: compiled, PromptHash: hex.EncodeToString(h[:]),
		ValidatedSkills: ids, SkillContracts: ids,
	}
}

// CompileAgentPromptForProject resolves skills through the source-agnostic
// Nexus catalog. It is the canonical path for Agent execution; Maestro is
// only an optional catalog source and is never required for builtin/project
// skills.
func (n *Nexus) CompileAgentPromptForProject(projectID, userPrompt string, requestedSkills []string) (*CompiledAgentPrompt, error) {
	userPrompt = strings.TrimSpace(userPrompt)
	if userPrompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	if len(requestedSkills) == 0 {
		return CompileAgentPromptWithValidatedSkills(userPrompt, nil), nil
	}
	catalog, err := n.skillCatalogForProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve Nexus skill catalog: %w", err)
	}
	selected := make([]nexusskills.Skill, 0, len(requestedSkills))
	seen := make(map[string]struct{}, len(requestedSkills))
	for _, requested := range requestedSkills {
		id := strings.TrimSpace(requested)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		skill, ok := catalog.Resolve(id)
		if !ok || skill.Availability != nexusskills.AvailabilityAvailable {
			return nil, fmt.Errorf("skill validation failed: skill %q is unavailable", id)
		}
		seen[id] = struct{}{}
		selected = append(selected, skill)
	}
	return CompileAgentPromptWithSkillContracts(userPrompt, selected), nil
}

func contractIDs(skills []nexusskills.Skill) []string {
	ids := make([]string, 0, len(skills))
	for _, skill := range skills {
		ids = append(ids, skill.ID)
	}
	return ids
}
