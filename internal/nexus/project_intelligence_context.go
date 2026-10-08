package nexus

import (
	"context"
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
)

const maxGroundedProjectFacts = 64

// BoundedProjectIntelligenceContext returns a compact, evidence-bearing
// envelope for reasoning providers. It prefers the persisted snapshot bound to
// the current code identity and performs only the existing static bounded scan
// when no persisted snapshot is available. It never executes project commands.
func (n *Nexus) BoundedProjectIntelligenceContext(ctx context.Context, projectID string) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	var snapshot *contextsnapshot.ProjectContextSnapshot
	view, viewErr := n.GetProjectIntelligence(ctx, projectID)
	if viewErr == nil && view != nil {
		snapshot = view.CurrentSnapshot
	}
	if snapshot == nil {
		var discovered contextsnapshot.ProjectContextSnapshot
		var discoverErr error
		if viewErr == nil && view != nil {
			discovered, discoverErr = contextsnapshot.DiscoverWithIdentity(
				project.CanonicalPath,
				contextsnapshot.Metadata{ProjectID: projectID},
				view.Identity,
			)
		} else {
			discovered, discoverErr = contextsnapshot.Discover(
				project.CanonicalPath,
				contextsnapshot.Metadata{ProjectID: projectID},
			)
		}
		if discoverErr != nil {
			return nil, fmt.Errorf("discover bounded project intelligence: %w", discoverErr)
		}
		snapshot = &discovered
	}

	facts := make([]map[string]any, 0, minInt(len(snapshot.Facts), maxGroundedProjectFacts))
	for _, fact := range snapshot.Facts {
		if len(facts) >= maxGroundedProjectFacts {
			break
		}
		item := map[string]any{
			"category":   fact.Category,
			"key":        fact.Key,
			"value":      fact.Value,
			"basis":      fact.Basis,
			"confidence": fact.Confidence,
		}
		if len(fact.Provenance) > 0 {
			item["source"] = fact.Provenance[0].SourcePath
			item["locator"] = fact.Provenance[0].Locator
		}
		facts = append(facts, item)
	}
	return map[string]any{
		"project_id":      projectID,
		"canonical_path":  project.CanonicalPath,
		"scanner_version": snapshot.ScannerVersion,
		"identity":        snapshot.Identity,
		"completeness":    snapshot.Completeness,
		"observed_at":     snapshot.ObservedAt.Format(time.RFC3339Nano),
		"facts":           facts,
		"warnings":        snapshot.Warnings,
	}, nil
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
