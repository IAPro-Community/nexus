package nexus

import (
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// packageSkillIDs is the compatibility boundary for persisted skill
// contracts. New plans use SkillIDs; the legacy MaestroSkills field remains
// readable when the canonical field is absent. MaestroGates are process
// requirements, not Skills, and must never be sent to the SkillCatalog.
func packageSkillIDs(pkg store.WorkPackage) []string {
	if len(pkg.SkillIDs) > 0 {
		return uniqueStrings(pkg.SkillIDs)
	}
	return uniqueStrings(pkg.MaestroSkills)
}

func packageRunSkillIDs(pkg *runner.PackageRun) []string {
	if pkg == nil {
		return nil
	}
	if len(pkg.SkillIDs) > 0 {
		return uniqueStrings(pkg.SkillIDs)
	}
	return uniqueStrings(pkg.MaestroSkills)
}

func validateGenericSkillIDs(n *Nexus, projectID string, requested []string) ([]string, error) {
	requested = uniqueStrings(requested)
	if len(requested) == 0 {
		return nil, nil
	}
	catalog, err := n.skillCatalogForProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve Nexus skill catalog: %w", err)
	}
	validated := make([]string, 0, len(requested))
	for _, id := range requested {
		skill, ok := catalog.Resolve(id)
		if !ok {
			return nil, fmt.Errorf("skill %q is not available in the resolved catalog", id)
		}
		if skill.Availability != nexusskills.AvailabilityAvailable {
			return nil, fmt.Errorf("skill %q is not operational (availability=%s)", id, skill.Availability)
		}
		validated = append(validated, skill.ID)
	}
	return validated, nil
}

// resolveSkillReferences returns the immutable provenance that explains how
// the canonical skill IDs on a WorkPackage were selected. It is intentionally
// derived from the same catalog boundary used for admission and prompt
// compilation; consumers never need to inspect a source-specific type.
func resolveSkillReferences(n *Nexus, projectID string, requested []string) ([]nexusskills.Resolution, error) {
	requested = uniqueStrings(requested)
	if len(requested) == 0 {
		return nil, nil
	}
	catalog, err := n.skillCatalogForProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve Nexus skill catalog: %w", err)
	}
	resolutions := make([]nexusskills.Resolution, 0, len(requested))
	for _, id := range requested {
		skill, ok := catalog.Resolve(id)
		if !ok {
			return nil, fmt.Errorf("skill %q is not available in the resolved catalog", id)
		}
		if skill.Availability != nexusskills.AvailabilityAvailable {
			return nil, fmt.Errorf("skill %q is not operational (availability=%s)", id, skill.Availability)
		}
		resolutions = append(resolutions, nexusskills.Resolution{
			RequestedCapability: id,
			Candidates:          []string{skill.ID},
			Selected:            skill.ID,
			Source:              skill.Source,
			Version:             skill.Version,
			Hash:                skill.Hash,
			Reason:              "deterministic canonical skill selection",
		})
	}
	return resolutions, nil
}

func canonicalSkillIDs(ids []string) []string {
	return uniqueStrings(ids)
}

func skillIDsErrorContext(ids []string) string {
	return strings.Join(canonicalSkillIDs(ids), ", ")
}
