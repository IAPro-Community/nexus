package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const missionEvidenceStreamName = "mission-validation"

// ValidationEvidenceReport is a read-only projection of the canonical
// append-only Mission evidence stream. It never reconstructs claims from
// provider output or documentation.
type ValidationEvidenceReport struct {
	RunID         string                          `json:"run_id"`
	PlanID        string                          `json:"plan_id,omitempty"`
	PlanRevision  int                             `json:"plan_revision,omitempty"`
	Stream        *store.ValidationEvidenceStream `json:"stream,omitempty"`
	Entries       []store.ValidationEvidenceEntry `json:"entries"`
	ChainVerified bool                            `json:"chain_verified"`
}

// ValidationEvidence projects the durable stream for one MissionRun. The full
// stream is verified first, then entries are selected by their persisted
// run_id envelope so two Missions in one project cannot see each other's
// evidence. Missing evidence is represented explicitly rather than synthesized
// as a PASS.
func (s *RunApplicationService) ValidationEvidence(ctx context.Context, runID string) (ValidationEvidenceReport, error) {
	if err := s.ready(ctx); err != nil {
		return ValidationEvidenceReport{}, err
	}
	run, err := s.nexus.Runner().GetRun(ctx, runID)
	if err != nil {
		return ValidationEvidenceReport{}, err
	}
	report := ValidationEvidenceReport{
		RunID: run.ID, PlanID: run.PlanID, PlanRevision: run.PlanRevision,
		Entries: []store.ValidationEvidenceEntry{}, ChainVerified: false,
	}
	st, err := s.nexus.OpenProject()
	if err != nil {
		return ValidationEvidenceReport{}, err
	}
	stream, err := st.GetValidationEvidenceStreamByName(run.ProjectID, missionEvidenceStreamName)
	if errors.Is(err, store.ErrValidationEvidenceNotFound) {
		return report, nil
	}
	if err != nil {
		return ValidationEvidenceReport{}, err
	}
	if err := st.VerifyValidationEvidenceChain(stream.ID); err != nil {
		return ValidationEvidenceReport{}, err
	}
	allEntries, err := st.ListAllValidationEvidenceEntries(stream.ID)
	if err != nil {
		return ValidationEvidenceReport{}, err
	}
	entries := make([]store.ValidationEvidenceEntry, 0, len(allEntries))
	for _, entry := range allEntries {
		var metadata struct {
			RunID string `json:"run_id"`
		}
		if err := json.Unmarshal([]byte(entry.EvidenceJSON), &metadata); err != nil {
			return ValidationEvidenceReport{}, fmt.Errorf("decode mission evidence %s: %w", entry.ID, err)
		}
		if strings.TrimSpace(metadata.RunID) == "" {
			return ValidationEvidenceReport{}, fmt.Errorf("mission evidence %s has no run id", entry.ID)
		}
		if metadata.RunID == run.ID {
			entries = append(entries, entry)
		}
	}
	report.Stream = &stream
	report.Entries = entries
	report.ChainVerified = true
	return report, nil
}

// recordMissionValidationEvidence binds a concrete verification result to the
// repository identity observed at the time of the check. It deliberately
// records failed checks too: a failed claim is still useful recovery evidence,
// while callers must never promote it to VERIFIED.
func (n *Nexus) recordMissionValidationEvidence(ctx context.Context, run *runner.MissionRun, scenario, command string, results []runner.VerificationResult, provider, profile, model string, attempt int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if run == nil {
		return fmt.Errorf("mission run is required")
	}
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	project, err := st.GetProject(run.ProjectID)
	if err != nil {
		return fmt.Errorf("load evidence project: %w", err)
	}
	identity, err := contextsnapshot.InspectCodeIdentity(project.CanonicalPath)
	if err != nil {
		return fmt.Errorf("inspect evidence identity: %w", err)
	}

	outcome, confidence, exitCode := store.EvidencePass, store.EvidenceVerified, (*int)(nil)
	passed := true
	for _, result := range results {
		if !result.Passed {
			passed = false
			value := result.ExitCode
			exitCode = &value
			break
		}
	}
	switch {
	case !passed:
		// Assertive failure wins over missing HEAD — a failed verification must
		// never be recorded as NOT_VERIFIED just because the workspace is non-git.
		outcome = store.EvidenceFail
		confidence = store.EvidenceObserved
	case len(results) == 0 || identity.HeadSHA == "":
		outcome = store.EvidenceNotVerified
		confidence = store.EvidenceObserved
	}
	environment, err := validationEnvironment(ctx, project.CanonicalPath)
	if err != nil {
		return fmt.Errorf("collect validation environment: %w", err)
	}
	evidence, err := json.Marshal(map[string]any{"run_id": run.ID, "plan_id": run.PlanID, "plan_revision": run.PlanRevision, "results": results})
	if err != nil {
		return fmt.Errorf("encode validation evidence: %w", err)
	}

	stream, err := st.GetValidationEvidenceStreamByName(run.ProjectID, missionEvidenceStreamName)
	if err != nil {
		if !strings.Contains(err.Error(), store.ErrValidationEvidenceNotFound.Error()) {
			return err
		}
		stream, err = st.CreateValidationEvidenceStream(run.ProjectID, missionEvidenceStreamName)
		if err != nil {
			// Another worker may have won stream creation. Re-read before failing.
			stream, err = st.GetValidationEvidenceStreamByName(run.ProjectID, missionEvidenceStreamName)
			if err != nil {
				return err
			}
		}
	}
	entry := store.ValidationEvidenceEntry{
		ID: stableEvidenceID(run.ID, scenario, attempt, len(results)), GitSHA: identity.HeadSHA,
		IdentityDigest: identity.IdentityDigest, RepositoryState: string(identity.State), EnvironmentJSON: environment,
		Provider: provider, Profile: profile, Model: model, Scenario: scenario, CommandDisplay: command, Outcome: outcome, Confidence: confidence,
		ExitCode: exitCode, EvidenceJSON: string(evidence), CreatedAt: time.Now().UTC(),
	}
	if _, err := st.AppendValidationEvidence(store.ValidationEvidenceAppendRequest{StreamID: stream.ID, Entry: entry}); err != nil {
		return err
	}
	if err := st.VerifyValidationEvidenceChain(stream.ID); err != nil {
		return err
	}
	return nil
}

// validationEnvironment records bounded toolchain metadata without invoking a
// shell or reading credential-bearing environment variables. Tool versions
// are optional when a tool is not installed or cannot answer quickly.
func validationEnvironment(ctx context.Context, projectRoot string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	environment := map[string]string{
		"go":   runtime.Version(),
		"os":   runtime.GOOS,
		"arch": runtime.GOARCH,
	}
	if version := boundedToolVersion(ctx, "node"); version != "" {
		environment["node"] = version
	}
	if packageManager := detectPackageManager(projectRoot); packageManager != "" {
		environment["package_manager"] = packageManager
		if version := boundedToolVersion(ctx, packageManager); version != "" {
			environment["package_manager_version"] = version
		}
	}
	raw, err := json.Marshal(environment)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func boundedToolVersion(ctx context.Context, name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	versionCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(versionCtx, path, "--version").Output()
	if err != nil {
		return ""
	}
	value := strings.TrimSpace(string(output))
	if len(value) > 128 {
		value = value[:128]
	}
	return value
}

func detectPackageManager(projectRoot string) string {
	candidates := []struct {
		manager string
		files   []string
	}{
		{manager: "pnpm", files: []string{"pnpm-lock.yaml"}},
		{manager: "yarn", files: []string{"yarn.lock"}},
		{manager: "bun", files: []string{"bun.lock", "bun.lockb"}},
		{manager: "npm", files: []string{"package-lock.json", "package.json"}},
	}
	for _, candidate := range candidates {
		for _, name := range candidate.files {
			path := filepath.Join(projectRoot, name)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return candidate.manager
			}
		}
	}
	return ""
}

func stableEvidenceID(runID, scenario string, iteration, resultCount int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d\x00%d", runID, scenario, iteration, resultCount)))
	return "mission_vee_" + hex.EncodeToString(sum[:])
}

func (e *nexusPackageExecutor) RecordValidationEvidence(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun, results []runner.VerificationResult) error {
	if pkg == nil {
		return fmt.Errorf("package is required")
	}
	return e.n.recordMissionValidationEvidence(ctx, run, "mission/package/"+pkg.PackageID, strings.Join(pkg.VerificationRequirements, " && "), results, pkg.Provider, pkg.Profile, selectedModelFromRoutingDecision(pkg.RoutingDecisionJSON), pkg.Attempt)
}

func (e *nexusPackageExecutor) RecordGlobalValidationEvidence(ctx context.Context, run *runner.MissionRun, results []runner.VerificationResult) error {
	return e.n.recordMissionValidationEvidence(ctx, run, "mission/global-definition", strings.Join(run.Contract.GlobalVerificationCommands, " && "), results, "", "", "", run.TotalIterations)
}

func selectedModelFromRoutingDecision(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var decision struct {
		SelectedModel string `json:"selected_model"`
	}
	if json.Unmarshal([]byte(raw), &decision) != nil {
		return ""
	}
	return strings.TrimSpace(decision.SelectedModel)
}
