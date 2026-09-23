package nexus

import (
	"context"
	"fmt"
	"sort"
	"strings"

	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const composerSkillCatalogLimit = 100

// ComposerSkillCatalogItem is the browser-safe projection of a Nexus Skill.
// Prompt instructions, permissions and local provenance paths never cross this
// boundary; they are resolved again server-side only when a skill is applied.
type ComposerSkillCatalogItem struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	Description  string                   `json:"description,omitempty"`
	Source       nexusskills.SourceID     `json:"source"`
	Version      string                   `json:"version,omitempty"`
	Risk         string                   `json:"risk,omitempty"`
	Availability nexusskills.Availability `json:"availability"`
	Capabilities []string                 `json:"capabilities,omitempty"`
	Triggers     []string                 `json:"triggers,omitempty"`
}

type composerSkillCatalogMatch struct {
	Item  ComposerSkillCatalogItem
	Score int
}

func (n *Nexus) ListComposerSkillCatalog(ctx context.Context, projectID, query string) ([]ComposerSkillCatalogItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	catalog, err := n.skillCatalogForProject(projectID)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	matches := make([]composerSkillCatalogMatch, 0, len(catalog.Library))
	for _, skill := range catalog.Library {
		item := ComposerSkillCatalogItem{
			ID: skill.ID, Name: skill.Name, Description: skill.Description,
			Source: skill.Source, Version: skill.Version, Risk: skill.Risk,
			Availability: skill.Availability,
			Capabilities: append([]string(nil), skill.Capabilities...),
			Triggers:     append([]string(nil), skill.Triggers...),
		}
		score := composerSkillCatalogScore(item, query)
		if query != "" && score == 0 {
			continue
		}
		matches = append(matches, composerSkillCatalogMatch{Item: item, Score: score})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		if matches[i].Item.Availability != matches[j].Item.Availability {
			return matches[i].Item.Availability == nexusskills.AvailabilityAvailable
		}
		return matches[i].Item.ID < matches[j].Item.ID
	})
	if len(matches) > composerSkillCatalogLimit {
		matches = matches[:composerSkillCatalogLimit]
	}
	out := make([]ComposerSkillCatalogItem, 0, len(matches))
	for _, match := range matches {
		out = append(out, match.Item)
	}
	return out, nil
}

func composerSkillCatalogScore(item ComposerSkillCatalogItem, query string) int {
	if query == "" {
		if item.Availability == nexusskills.AvailabilityAvailable {
			return 1
		}
		return 0
	}
	haystack := strings.ToLower(strings.Join([]string{
		item.ID, item.Name, item.Description, string(item.Source),
		strings.Join(item.Capabilities, " "), strings.Join(item.Triggers, " "),
	}, " "))
	score := 0
	if strings.EqualFold(item.ID, query) || strings.EqualFold(item.Name, query) {
		score += 100
	}
	if strings.HasPrefix(strings.ToLower(item.ID), query) || strings.HasPrefix(strings.ToLower(item.Name), query) {
		score += 50
	}
	for _, term := range strings.Fields(query) {
		if len(term) < 2 {
			continue
		}
		if strings.Contains(haystack, term) {
			score += 10
		}
	}
	if item.Availability == nexusskills.AvailabilityAvailable {
		score += 2
	}
	return score
}

func (n *Nexus) SelectComposerSkill(ctx context.Context, sessionID, skillID string) (*ComposerSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	if session.State == store.ComposerFinalized {
		return nil, fmt.Errorf("composer session is finalized; create a new session to change skills")
	}
	catalog, err := n.skillCatalogForProject(session.ProjectID)
	if err != nil {
		return nil, err
	}
	skill, ok := catalog.Resolve(strings.TrimSpace(skillID))
	if !ok {
		return nil, fmt.Errorf("skill %q is not available in the resolved Nexus catalog", skillID)
	}
	if skill.Availability != nexusskills.AvailabilityAvailable {
		return nil, fmt.Errorf("skill %q is not operational (availability=%s)", skill.ID, skill.Availability)
	}

	state := store.ComposerSkillAccepted
	if proposals, listErr := st.ListComposerSkillProposals(sessionID); listErr == nil {
		for _, proposal := range proposals {
			if proposal.SkillID == skill.ID && proposal.State == store.ComposerSkillApplied {
				state = store.ComposerSkillApplied
				break
			}
		}
	}
	if _, err := st.UpsertComposerSkillProposal(store.ComposerSkillProposal{
		SessionID: sessionID, SkillID: skill.ID, State: state,
		Reason:        "Selected manually from the source-agnostic Nexus Skill Catalog.",
		Applicability: skill.Description, Risk: skill.Risk, Source: string(skill.Source),
		Version: skill.Version, Available: true,
	}); err != nil {
		return nil, err
	}
	return n.composeSessionView(st, *session, decodeComposerBrief(session.BriefJSON))
}
