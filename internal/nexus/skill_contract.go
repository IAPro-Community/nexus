package nexus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const sourceArtifactFactKey = "nexus.source_artifact_id"

// FrozenSkillContract is the minimal immutable Skill knowledge bound to one
// PromptArtifact/Mission snapshot. It deliberately excludes scripts, local
// provenance paths and operational permissions: Skills remain knowledge, not
// executable plugins or authority grants.
type FrozenSkillContract struct {
	ID           string                `json:"id"`
	Source       nexusskills.SourceID  `json:"source"`
	Version      string                `json:"version"`
	SkillHash    string                `json:"skill_hash"`
	Instructions string                `json:"instructions"`
	ContractHash string                `json:"contract_hash"`
}

func freezeSkillContract(skill nexusskills.Skill) FrozenSkillContract {
	contract := FrozenSkillContract{
		ID: strings.TrimSpace(skill.ID), Source: skill.Source,
		Version: firstNonEmpty(strings.TrimSpace(skill.Version), "unknown"),
		SkillHash: strings.TrimSpace(skill.Hash), Instructions: strings.TrimSpace(skill.Instructions),
	}
	contract.ContractHash = frozenSkillContractHash(contract)
	return contract
}

func frozenSkillContractHash(contract FrozenSkillContract) string {
	payload := struct {
		ID           string               `json:"id"`
		Source       nexusskills.SourceID `json:"source"`
		Version      string               `json:"version"`
		SkillHash    string               `json:"skill_hash"`
		Instructions string               `json:"instructions"`
	}{
		ID: contract.ID, Source: contract.Source, Version: contract.Version,
		SkillHash: contract.SkillHash, Instructions: contract.Instructions,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func validateFrozenSkillContract(contract FrozenSkillContract) error {
	if strings.TrimSpace(contract.ID) == "" {
		return fmt.Errorf("frozen skill contract id is required")
	}
	if strings.TrimSpace(contract.ContractHash) == "" || contract.ContractHash != frozenSkillContractHash(contract) {
		return fmt.Errorf("frozen skill contract %s failed integrity validation", contract.ID)
	}
	return nil
}

func frozenSkillContractsFromCatalog(catalog nexusskills.Catalog, requested []string) ([]FrozenSkillContract, error) {
	requested = uniqueStrings(requested)
	out := make([]FrozenSkillContract, 0, len(requested))
	for _, id := range requested {
		skill, ok := catalog.Resolve(id)
		if !ok || skill.Availability != nexusskills.AvailabilityAvailable {
			return nil, fmt.Errorf("skill %q is not available in the resolved catalog", id)
		}
		out = append(out, freezeSkillContract(skill))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func frozenSkillContractsForIDs(contracts []FrozenSkillContract, requested []string) ([]FrozenSkillContract, error) {
	byID := make(map[string]FrozenSkillContract, len(contracts))
	for _, contract := range contracts {
		if err := validateFrozenSkillContract(contract); err != nil {
			return nil, err
		}
		byID[contract.ID] = contract
	}
	out := make([]FrozenSkillContract, 0, len(requested))
	for _, id := range uniqueStrings(requested) {
		contract, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("frozen skill contract %q is missing from execution snapshot", id)
		}
		out = append(out, contract)
	}
	return out, nil
}

func frozenSkillContractsFromArtifact(artifact store.PromptArtifact) ([]FrozenSkillContract, error) {
	if strings.TrimSpace(artifact.ContextJSON) == "" {
		return nil, nil
	}
	var envelope struct {
		SkillContracts []FrozenSkillContract `json:"skill_contracts"`
	}
	if err := json.Unmarshal([]byte(artifact.ContextJSON), &envelope); err != nil {
		return nil, fmt.Errorf("decode frozen Composer skill contracts: %w", err)
	}
	for _, contract := range envelope.SkillContracts {
		if err := validateFrozenSkillContract(contract); err != nil {
			return nil, err
		}
	}
	return envelope.SkillContracts, nil
}

func resolveFrozenSkillContractsForPlan(n *Nexus, plan *store.WorkPlan) ([]FrozenSkillContract, error) {
	if n == nil || plan == nil {
		return nil, fmt.Errorf("freeze skill contracts requires Nexus and WorkPlan")
	}
	requested := []string{}
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			requested = append(requested, packageSkillIDs(pkg)...)
		}
	}
	requested = uniqueStrings(requested)
	if len(requested) == 0 {
		return nil, nil
	}

	existing := map[string]FrozenSkillContract{}
	if artifactID := strings.TrimSpace(plan.StructuredFacts[sourceArtifactFactKey]); artifactID != "" {
		st, err := n.OpenProject()
		if err != nil {
			return nil, err
		}
		if artifact, artifactErr := st.GetPromptArtifact(artifactID); artifactErr == nil {
			contracts, decodeErr := frozenSkillContractsFromArtifact(*artifact)
			if decodeErr != nil {
				return nil, decodeErr
			}
			for _, contract := range contracts {
				existing[contract.ID] = contract
			}
		}
	}

	missing := []string{}
	for _, id := range requested {
		if _, ok := existing[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		catalog, err := n.skillCatalogForProject(plan.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("resolve Nexus skill catalog: %w", err)
		}
		resolved, err := frozenSkillContractsFromCatalog(catalog, missing)
		if err != nil {
			return nil, err
		}
		for _, contract := range resolved {
			existing[contract.ID] = contract
		}
	}

	out := make([]FrozenSkillContract, 0, len(requested))
	for _, id := range requested {
		contract, ok := existing[id]
		if !ok {
			return nil, fmt.Errorf("skill %q has no frozen execution contract", id)
		}
		out = append(out, contract)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

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
