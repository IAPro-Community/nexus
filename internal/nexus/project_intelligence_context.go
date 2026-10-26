package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
)

const (
	maxGroundedProjectFacts            = 64
	maxGroundedProjectContextBytes     = 16 * 1024
	maxGroundedProjectFactFieldBytes   = 512
	maxGroundedProjectFactListItems    = 32
	maxGroundedProjectFactListItemSize = 128
)

func boundedGroundedString(value string, limit int) (string, bool) {
	value = security.Redact(value)
	if len(value) <= limit {
		return value, false
	}
	var bounded strings.Builder
	bounded.Grow(limit)
	for _, r := range value {
		if bounded.Len()+utf8.RuneLen(r) > limit {
			return bounded.String(), true
		}
		bounded.WriteRune(r)
	}
	return bounded.String(), false
}

func boundedGroundedFactValue(value any) (any, bool, bool) {
	switch typed := value.(type) {
	case string:
		bounded, truncated := boundedGroundedString(typed, maxGroundedProjectFactFieldBytes)
		return bounded, truncated, true
	case bool:
		return typed, false, true
	case []string:
		return boundedGroundedStringList(typed)
	case []any:
		values := make([]string, 0, len(typed))
		for _, raw := range typed {
			value, ok := raw.(string)
			if !ok {
				return nil, false, false
			}
			values = append(values, value)
		}
		return boundedGroundedStringList(values)
	default:
		return nil, false, false
	}
}

func boundedGroundedStringList(values []string) ([]string, bool, bool) {
	bounded := make([]string, 0, minInt(len(values), maxGroundedProjectFactListItems))
	truncated := len(values) > maxGroundedProjectFactListItems
	for _, value := range values {
		if len(bounded) >= maxGroundedProjectFactListItems {
			break
		}
		item, itemTruncated := boundedGroundedString(value, maxGroundedProjectFactListItemSize)
		bounded = append(bounded, item)
		truncated = truncated || itemTruncated
	}
	return bounded, truncated, true
}

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
	var inspectedIdentity *contextsnapshot.CodeIdentity
	if view, viewErr := n.GetProjectIntelligence(ctx, projectID); viewErr == nil {
		snapshot = view.CurrentSnapshot
		inspectedIdentity = &view.Identity
	}
	if snapshot == nil {
		var discovered contextsnapshot.ProjectContextSnapshot
		var discoverErr error
		if inspectedIdentity != nil {
			discovered, discoverErr = contextsnapshot.DiscoverWithIdentity(project.CanonicalPath, contextsnapshot.Metadata{ProjectID: projectID}, *inspectedIdentity)
		} else {
			discovered, discoverErr = contextsnapshot.Discover(project.CanonicalPath, contextsnapshot.Metadata{ProjectID: projectID})
		}
		if discoverErr != nil {
			return nil, fmt.Errorf("discover bounded project intelligence: %w", discoverErr)
		}
		snapshot = &discovered
	}

	identity := snapshot.Identity
	if inspectedIdentity != nil {
		identity = *inspectedIdentity
	}
	facts := make([]map[string]any, 0, minInt(len(snapshot.Facts), maxGroundedProjectFacts))
	envelope := map[string]any{
		"project_id":        projectID,
		"identity_digest":   identity.IdentityDigest,
		"identity_state":    identity.State,
		"identity_complete": identity.Complete,
		"scanner_version":   snapshot.ScannerVersion,
		"completeness":      snapshot.Completeness,
		"observed_at":       snapshot.ObservedAt.UTC().Format(time.RFC3339Nano),
		"facts":             facts,
		"truncated":         false,
	}
	for _, fact := range snapshot.Facts {
		if len(facts) >= maxGroundedProjectFacts {
			envelope["truncated"] = true
			break
		}
		value, valueTruncated, ok := boundedGroundedFactValue(fact.Value)
		if !ok {
			envelope["truncated"] = true
			continue
		}
		observedAt := fact.ObservedAt
		if observedAt.IsZero() {
			observedAt = snapshot.ObservedAt
		}
		item := map[string]any{
			"category":    fact.Category,
			"key":         fact.Key,
			"value":       value,
			"basis":       fact.Basis,
			"confidence":  fact.Confidence,
			"observed_at": observedAt.UTC().Format(time.RFC3339Nano),
		}
		if len(fact.Provenance) > 0 {
			source, sourceTruncated := boundedGroundedString(fact.Provenance[0].SourcePath, maxGroundedProjectFactFieldBytes)
			locator, locatorTruncated := boundedGroundedString(fact.Provenance[0].Locator, maxGroundedProjectFactFieldBytes)
			item["source"] = source
			item["locator"] = locator
			valueTruncated = valueTruncated || sourceTruncated || locatorTruncated
		}
		if valueTruncated {
			item["value_truncated"] = true
			envelope["truncated"] = true
		}
		facts = append(facts, item)
		envelope["facts"] = facts
		raw, marshalErr := json.Marshal(envelope)
		if marshalErr != nil {
			return nil, fmt.Errorf("encode bounded project intelligence: %w", marshalErr)
		}
		if len(raw) > maxGroundedProjectContextBytes {
			facts = facts[:len(facts)-1]
			envelope["facts"] = facts
			envelope["truncated"] = true
		}
	}
	for {
		raw, marshalErr := json.Marshal(envelope)
		if marshalErr != nil {
			return nil, fmt.Errorf("encode bounded project intelligence: %w", marshalErr)
		}
		if len(raw) <= maxGroundedProjectContextBytes || len(facts) == 0 {
			return envelope, nil
		}
		facts = facts[:len(facts)-1]
		envelope["facts"] = facts
		envelope["truncated"] = true
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
