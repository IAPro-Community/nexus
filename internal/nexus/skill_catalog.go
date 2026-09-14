package nexus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
)

// maestroSkillSource adapts Maestro's optional capability response to the
// source-agnostic Nexus Skill contract. Consumers never need to know that a
// selected skill originated in Maestro.
type maestroSkillSource struct {
	client *MaestroClient
}

func NewMaestroSkillSource(client *MaestroClient) nexusskills.Source {
	return maestroSkillSource{client: client}
}

func (s maestroSkillSource) ID() string { return string(nexusskills.SourceMaestro) }

func (s maestroSkillSource) Available(context.Context) bool {
	return s.client != nil && s.client.Status().Available && s.client.Status().Capabilities != nil
}

func (s maestroSkillSource) Discover(ctx context.Context) ([]nexusskills.Skill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !s.Available(ctx) {
		return nil, nexusskills.ErrSourceUnavailable
	}
	status := s.client.Status()
	result := make([]nexusskills.Skill, 0, len(status.Capabilities.Skills))
	for _, item := range status.Capabilities.Skills {
		result = append(result, nexusskills.Skill{
			ID: item.ID, Name: item.Name, Description: item.Description,
			Source: nexusskills.SourceMaestro, Version: item.Version,
			Triggers: append([]string(nil), item.Triggers...), Risk: item.Risk,
			Instructions: item.Prompt,
			Provenance:   nexusskills.Provenance{SourceID: string(nexusskills.SourceMaestro), Locator: "maestro:" + item.ID},
		})
	}
	return result, nil
}

func (s maestroSkillSource) Resolve(ctx context.Context, id string) (nexusskills.Skill, bool, error) {
	items, err := s.Discover(ctx)
	if err != nil {
		return nexusskills.Skill{}, false, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, true, nil
		}
	}
	return nexusskills.Skill{}, false, nil
}

func (n *Nexus) skillCatalogForProject(projectID string) (nexusskills.Catalog, error) {
	sources := []nexusskills.Source{nexusskills.NewBuiltinSource()}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		// Only the Nexus-owned user skill root. Do not scan ~/.codex/skills:
		// that tree belongs to other tools and would pollute catalog resolution.
		userRoots := []string{filepath.Join(home, ".nexus", "skills")}
		sources = append(sources, nexusskills.NewDirectorySource("user", nexusskills.SourceUser, userRoots))
	}
	if projectID != "" {
		st, err := n.OpenProject()
		if err != nil {
			return nexusskills.Catalog{}, err
		}
		project, err := st.GetProject(projectID)
		if err != nil {
			return nexusskills.Catalog{}, err
		}
		// Project skills are explicit roots, keeping discovery bounded and
		// preventing a catalog request from walking the whole repository.
		roots := []string{
			filepath.Join(project.CanonicalPath, ".nexus", "skills"),
			filepath.Join(project.CanonicalPath, ".codex", "skills"),
		}
		sources = append(sources, nexusskills.NewDirectorySource("project", nexusskills.SourceProject, roots))
	}
	maestro := NewMaestroClient()
	if n != nil && n.maestroStatus != nil {
		// The injected status is used by tests and embedded callers; create a
		// lightweight client only when the normal integration is not available.
		status := n.currentMaestroStatus()
		maestro.status = status
	}
	sources = append(sources, NewMaestroSkillSource(maestro))
	catalog, err := nexusskills.NewCatalog(sources...)
	if err != nil {
		return nexusskills.Catalog{}, fmt.Errorf("resolve Nexus skill catalog: %w", err)
	}
	return catalog, nil
}
