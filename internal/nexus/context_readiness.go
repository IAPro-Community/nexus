package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
)

// Context readiness is deliberately a small contract. Maestro remains the
// authority for semantic hydration; Nexus only records whether the inputs to
// a context are present and whether they still match the hydrated snapshot.
const (
	ContextMissing   = "MISSING"
	ContextHydrating = "HYDRATING"
	ContextReady     = "READY"
	ContextStale     = "STALE"
	ContextFailed    = "FAILED"
)

// ContextFingerprint identifies the repository state and Maestro authority
// used to produce a context. DirtyFingerprint is intentionally opaque: the
// caller may use a git status hash without exposing repository contents.
type ContextFingerprint struct {
	ProjectID        string `json:"project_id"`
	Branch           string `json:"branch"`
	Head             string `json:"head"`
	DirtyFingerprint string `json:"dirty_fingerprint"`
	MaestroVersion   string `json:"maestro_version"`
	MaestroState     string `json:"maestro_state"`
}

// ObserveContextFingerprint reads only repository metadata; semantic
// hydration remains owned by Maestro.
func ObserveContextFingerprint(ctx context.Context, projectID, projectPath, maestroVersion, maestroState string) (ContextFingerprint, error) {
	if strings.TrimSpace(projectID) == "" || strings.TrimSpace(projectPath) == "" {
		return ContextFingerprint{}, fmt.Errorf("project and path are required")
	}
	git := func(args ...string) (string, error) {
		out, err := exec.CommandContext(ctx, "git", append([]string{"-C", projectPath}, args...)...).Output()
		return strings.TrimSpace(string(out)), err
	}
	branch, err := git("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ContextFingerprint{}, fmt.Errorf("read branch: %w", err)
	}
	head, err := git("rev-parse", "HEAD")
	if err != nil {
		return ContextFingerprint{}, fmt.Errorf("read HEAD: %w", err)
	}
	status, err := git("status", "--porcelain=v1")
	if err != nil {
		return ContextFingerprint{}, fmt.Errorf("read dirty state: %w", err)
	}
	h := sha256.Sum256([]byte(status))
	return ContextFingerprint{ProjectID: projectID, Branch: branch, Head: head, DirtyFingerprint: hex.EncodeToString(h[:]), MaestroVersion: maestroVersion, MaestroState: maestroState}, nil
}

// ContextReadiness is the transport-neutral state consumed by Composer and
// other surfaces. Direct agents and project shells must not require it.
type ContextReadiness struct {
	State       string              `json:"state"`
	Fingerprint *ContextFingerprint `json:"fingerprint,omitempty"`
	Reason      string              `json:"reason,omitempty"`
}

// EvaluateContextReadiness compares the currently observed inputs with the
// snapshot used by hydration. It has no side effects and never performs
// semantic hydration itself.
func EvaluateContextReadiness(previous, current *ContextFingerprint, hydrating bool, failure string) ContextReadiness {
	if failure != "" {
		return ContextReadiness{State: ContextFailed, Fingerprint: current, Reason: failure}
	}
	if hydrating {
		return ContextReadiness{State: ContextHydrating, Fingerprint: current}
	}
	if previous == nil || current == nil {
		return ContextReadiness{State: ContextMissing, Fingerprint: current, Reason: "context has not been hydrated"}
	}
	if *previous != *current {
		return ContextReadiness{State: ContextStale, Fingerprint: current, Reason: "project inputs changed since hydration"}
	}
	return ContextReadiness{State: ContextReady, Fingerprint: current}
}
