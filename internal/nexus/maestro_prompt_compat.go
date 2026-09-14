package nexus

import (
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/maestrogates"
	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
)

// CompileAgentPrompt is a compatibility adapter for callers that still ask
// Maestro to validate skill IDs. The canonical compiler consumes generic
// Skills; this adapter only translates the optional legacy source at the
// boundary.
func CompileAgentPrompt(userPrompt string, requestedSkills []string, maestroClient *MaestroClient) (*CompiledAgentPrompt, error) {
	userPrompt = strings.TrimSpace(userPrompt)
	if userPrompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	var validatedSkills []string
	if len(requestedSkills) > 0 {
		if maestroClient == nil {
			maestroClient = NewMaestroClient()
		}
		status := maestroClient.Status()
		var cause error
		if status.Error != "" {
			cause = fmt.Errorf("%s", status.Error)
		}
		var catalog []string
		if status.Capabilities != nil {
			catalog = status.Capabilities.SkillIDs()
		}
		var err error
		validatedSkills, err = maestrogates.ValidateStrict(requestedSkills, status.Available, catalog, cause)
		if err != nil {
			return nil, fmt.Errorf("skill validation failed: %w", err)
		}
	}

	contracts := make([]nexusskills.Skill, 0, len(validatedSkills))
	for _, id := range validatedSkills {
		if skill, ok := maestroClient.CatalogSkill(id); ok {
			contracts = append(contracts, nexusskills.Skill{
				ID: skill.ID, Name: skill.Name, Description: skill.Description,
				Source: nexusskills.SourceMaestro, Version: skill.Version,
				Triggers: append([]string(nil), skill.Triggers...), Risk: skill.Risk,
				Instructions: skill.Contract,
			})
		}
	}
	return CompileAgentPromptWithContracts(userPrompt, validatedSkills, contracts), nil
}
