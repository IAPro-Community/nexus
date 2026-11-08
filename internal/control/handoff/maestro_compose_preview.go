package handoff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/security"
)

var maestroComposeModelID = regexp.MustCompile("^[A-Za-z0-9._:/+-]{1,128}$")

// MaestroComposePreviewRequest uses a persisted, finalized PromptArtifact.
// ApprovedModelUse is explicit because semantic composition invokes an external
// Codex model and can incur real provider usage.
type MaestroComposePreviewRequest struct {
	ArtifactID       string
	Version          int
	Content          string
	SHA256           string
	Model            string
	ApprovedModelUse bool
}

type MaestroComposedTask struct {
	ID                   string   `json:"id"`
	Title                string   `json:"title"`
	Objective            string   `json:"objective"`
	Type                 string   `json:"type,omitempty"`
	DependsOn            []string `json:"dependsOn"`
	AcceptanceCriteria   []string `json:"acceptanceCriteria"`
	RequiredCapabilities []string `json:"requiredCapabilities,omitempty"`
}

type MaestroComposePreview struct {
	SchemaVersion    int    `json:"schemaVersion"`
	Authority        string `json:"authority"`
	Source           string `json:"source"`
	ProviderID       string `json:"providerId"`
	ReviewOnly       bool   `json:"reviewOnly"`
	Executable       bool   `json:"executable"`
	RequiresApproval bool   `json:"requiresApproval"`
	Artifact         struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
		SHA256  string `json:"sha256"`
	} `json:"artifact"`
	Context struct {
		ContentDigest string `json:"contentDigest"`
	} `json:"context"`
	Validation struct {
		Valid    bool            `json:"valid"`
		Blockers json.RawMessage `json:"blockers"`
		Warnings json.RawMessage `json:"warnings"`
	} `json:"validation"`
	Tasks []MaestroComposedTask `json:"tasks"`
}

func maestroComposeMethod(init maestroBridgeInit) (string, error) {
	if init.ProtocolVersion != maestroBridgeProtocolVersion || !init.Capabilities.ProjectCompose {
		return "", errors.New("external Maestro does not support canonical project.compose")
	}
	return "project.compose", nil
}

// decodeMaestroComposePreview enforces provider, source and artifact identity.
// A response marked invalid is returned only for operator review, never apply.
func decodeMaestroComposePreview(raw json.RawMessage, req MaestroComposePreviewRequest) (MaestroComposePreview, error) {
	var out MaestroComposePreview
	if err := json.Unmarshal(raw, &out); err != nil {
		return MaestroComposePreview{}, errors.New("invalid Maestro compose response")
	}
	if out.SchemaVersion != 1 || out.Authority != "MAESTRO" ||
		out.Source != "maestro-semantic-planner" || out.ProviderID != "codex" ||
		!out.ReviewOnly || out.Executable || !out.RequiresApproval ||
		out.Artifact.ID != req.ArtifactID || out.Artifact.Version != req.Version ||
		!strings.EqualFold(out.Artifact.SHA256, req.SHA256) ||
		len(out.Context.ContentDigest) != 64 {
		return MaestroComposePreview{}, errors.New("maestro compose source or approval provenance mismatch")
	}
	if _, err := hex.DecodeString(out.Context.ContentDigest); err != nil {
		return MaestroComposePreview{}, errors.New("invalid Maestro context digest")
	}
	if !out.Validation.Valid {
		if len(out.Tasks) != 0 {
			return MaestroComposePreview{}, errors.New("invalid Maestro plan cannot supply executable tasks")
		}
		return out, nil
	}
	// A model or stale Bridge must not mark a plan as valid while still
	// returning blockers. The Nexus UI would otherwise offer an unsafe GO.
	var blockers []json.RawMessage
	if len(out.Validation.Blockers) == 0 ||
		strings.TrimSpace(string(out.Validation.Blockers)) == "null" ||
		json.Unmarshal(out.Validation.Blockers, &blockers) != nil || len(blockers) != 0 {
		return MaestroComposePreview{}, errors.New("maestro compose marked a blocked plan as valid")
	}
	if len(out.Tasks) == 0 || len(out.Tasks) > 32 {
		return MaestroComposePreview{}, errors.New("maestro generated an invalid task count")
	}
	dependencies := make(map[string][]string, len(out.Tasks))
	for i := range out.Tasks {
		task := &out.Tasks[i]
		if _, exists := dependencies[task.ID]; exists {
			return MaestroComposePreview{}, errors.New("maestro compose has duplicate task IDs")
		}
		if !maestroSafeAssignmentID.MatchString(task.ID) ||
			strings.TrimSpace(task.Title) == "" || len(task.Title) > 240 ||
			strings.TrimSpace(task.Objective) == "" || len(task.Objective) > 4000 ||
			len(task.AcceptanceCriteria) < 1 || len(task.AcceptanceCriteria) > 20 ||
			len(task.DependsOn) > 32 {
			return MaestroComposePreview{}, errors.New("maestro compose task is malformed")
		}
		for _, criterion := range task.AcceptanceCriteria {
			if strings.TrimSpace(criterion) == "" || len(criterion) < 3 || len(criterion) > 500 {
				return MaestroComposePreview{}, errors.New("maestro compose acceptance criterion is malformed")
			}
		}
		seenDependencies := make(map[string]bool, len(task.DependsOn))
		for _, dependency := range task.DependsOn {
			if seenDependencies[dependency] {
				return MaestroComposePreview{}, errors.New("maestro compose has duplicate dependency edges")
			}
			seenDependencies[dependency] = true
		}
		task.Title = security.Redact(task.Title)
		task.Objective = security.Redact(task.Objective)
		for j := range task.AcceptanceCriteria {
			task.AcceptanceCriteria[j] = security.Redact(task.AcceptanceCriteria[j])
		}
		dependencies[task.ID] = task.DependsOn
	}
	// Even a valid=true response from a provider must not introduce a cycle,
	// missing dependency or implicit duplicate writer in the local draft.
	visiting, visited := make(map[string]bool), make(map[string]bool)
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return false
		}
		if visited[id] {
			return true
		}
		deps, exists := dependencies[id]
		if !exists {
			return false
		}
		visiting[id] = true
		for _, dep := range deps {
			if dep == id || !visit(dep) {
				return false
			}
		}
		delete(visiting, id)
		visited[id] = true
		return true
	}
	for id := range dependencies {
		if !visit(id) {
			return MaestroComposePreview{}, errors.New("maestro compose DAG is invalid")
		}
	}
	return out, nil
}

// FetchMaestroComposePreview calls only the external canonical Maestro Bridge.
// It does not persist a plan, create missions, dispatch runs or fall back to a
// Nexus-owned planner. A finalized artifact and fresh cost consent are required.
func FetchMaestroComposePreview(parent context.Context, workspace string, req MaestroComposePreviewRequest) (MaestroComposePreview, error) {
	if !req.ApprovedModelUse {
		return MaestroComposePreview{}, errors.New("explicit model-use approval required")
	}
	if !maestroSafeAssignmentID.MatchString(req.ArtifactID) || req.Version < 1 ||
		req.Version > 1000000 || len(req.Content) < 8 || len(req.Content) > 64000 {
		return MaestroComposePreview{}, errors.New("invalid or unbounded finalized PromptArtifact")
	}
	hash := sha256.Sum256([]byte(req.Content))
	if !strings.EqualFold(hex.EncodeToString(hash[:]), req.SHA256) {
		return MaestroComposePreview{}, errors.New("PromptArtifact integrity mismatch")
	}
	model := req.Model
	if model == "" {
		model = "default"
	}
	if !maestroComposeModelID.MatchString(model) || strings.Contains(model, "..") {
		return MaestroComposePreview{}, errors.New("invalid Codex model selection")
	}
	params := map[string]any{
		"projectPath": workspace,
		"artifact": map[string]any{
			"id": req.ArtifactID, "version": req.Version,
			"content": req.Content, "sha256": req.SHA256, "approved": true,
		},
		"providerId": "codex", "model": model, "approvedModelUse": true,
	}
	raw, method, err := maestroBridgeCall(parent, workspace, 135*time.Second, maestroComposeMethod, params)
	if err != nil {
		return MaestroComposePreview{}, fmt.Errorf("maestro composition unavailable: %w", err)
	}
	if method != "project.compose" {
		return MaestroComposePreview{}, errors.New("unexpected Maestro composition method")
	}
	return decodeMaestroComposePreview(raw, req)
}
