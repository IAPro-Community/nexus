package nexus

import (
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

// MissionStageProjection is a read-only projection over the canonical
// MissionRunner state machine. It never creates a scheduler, router,
// memory engine or mission engine: it only reads runner.MissionRun and
// projects current stage/agent/task/resource/fallback-retry/verification.
// Unknown or empty states are reported as UNKNOWN, never invented.
type MissionStageProjection struct {
	RunID           string                     `json:"run_id"`
	CurrentStage    string                     `json:"current_stage"`
	AgentID         string                     `json:"agent_id,omitempty"`
	TaskID          string                     `json:"task_id,omitempty"`
	TaskTitle       string                     `json:"task_title,omitempty"`
	Provider        string                     `json:"provider,omitempty"`
	Profile         string                     `json:"profile,omitempty"`
	Resource        string                     `json:"resource,omitempty"`
	IsFallbackRetry bool                       `json:"is_fallback_retry"`
	Attempt         int                        `json:"attempt,omitempty"`
	RetryFrom       string                     `json:"retry_from,omitempty"`
	FailureKind     string                     `json:"failure_kind,omitempty"`
	Verification    MissionVerificationSummary `json:"verification"`
	RunState        string                     `json:"run_state,omitempty"`
}

// MissionVerificationSummary summarizes durable verification evidence without
// inventing PASS when no evidence exists.
type MissionVerificationSummary struct {
	Count       int  `json:"count"`
	Passed      int  `json:"passed"`
	Failed      int  `json:"failed"`
	HasEvidence bool `json:"has_evidence"`
}

// CanonicalMissionStage maps the runner state machine to the presentation
// pipeline READY→ALLOCATE→COMPILE_PROMPT→EXECUTE→TEST→REVIEW→VERIFY.
// Terminal runner states project to their honest outcome (VERIFIED/FAILED/...),
// never to a guessed pipeline stage. Empty or unrecognized states are UNKNOWN.
func CanonicalMissionStage(state runner.State) string {
	switch state {
	case runner.StateReady:
		return "READY"
	case runner.StateAllocating:
		return "ALLOCATE"
	case runner.StateCompiling:
		return "COMPILE_PROMPT"
	case runner.StateExecuting:
		return "EXECUTE"
	case runner.StateTesting:
		return "TEST"
	case runner.StateReviewing:
		return "REVIEW"
	case runner.StateVerifying:
		return "VERIFY"
	case runner.StateRemediating:
		return "VERIFY"
	case runner.StateVerified, runner.StateCompletedVerified:
		return "VERIFY"
	case runner.StatePending:
		return "PENDING"
	case runner.StateFailed, runner.StateFailedNoProgress,
		runner.StateFailedBudgetExceeded, runner.StateFailedVerification,
		runner.StateBlockedNeedsUser, runner.StateEscalated,
		runner.StatePaused, runner.StateCanceledByUser:
		return string(state)
	default:
		if strings.TrimSpace(string(state)) == "" {
			return "UNKNOWN"
		}
		return "UNKNOWN"
	}
}

// BuildMissionStageProjection projects the current package of a MissionRun.
// It uses CurrentPkgIndex when valid, otherwise the first actionable package,
// otherwise UNKNOWN without inventing agent/task/resource evidence.
func BuildMissionStageProjection(run *runner.MissionRun) MissionStageProjection {
	if run == nil || strings.TrimSpace(run.ID) == "" {
		return MissionStageProjection{CurrentStage: "UNKNOWN"}
	}
	out := MissionStageProjection{RunID: run.ID, RunState: string(run.State)}
	pkg := currentProjectionPackage(run)
	if pkg == nil {
		out.CurrentStage = "UNKNOWN"
		return out
	}
	out.CurrentStage = CanonicalMissionStage(pkg.State)
	out.AgentID = pkg.AssignedAgent
	out.TaskID = pkg.PackageID
	out.TaskTitle = pkg.Title
	out.Provider = pkg.Provider
	out.Profile = pkg.Profile
	if pkg.Provider != "" || pkg.Profile != "" {
		out.Resource = strings.TrimSpace(pkg.Provider + ":" + pkg.Profile)
		out.Resource = strings.Trim(out.Resource, ":")
	}
	out.Attempt = pkg.Attempt
	if pkg.RetryFrom != "" {
		out.RetryFrom = string(pkg.RetryFrom)
	}
	if pkg.FailureKind != "" {
		out.FailureKind = string(pkg.FailureKind)
	}
	out.IsFallbackRetry = pkg.Attempt > 1
	summary := MissionVerificationSummary{}
	for _, v := range pkg.Verifications {
		summary.Count++
		if v.Passed {
			summary.Passed++
		} else {
			summary.Failed++
		}
	}
	summary.HasEvidence = summary.Count > 0
	out.Verification = summary
	return out
}

func currentProjectionPackage(run *runner.MissionRun) *runner.PackageRun {
	if run == nil || len(run.PackageRuns) == 0 {
		return nil
	}
	if run.CurrentPkgIndex >= 0 && run.CurrentPkgIndex < len(run.PackageRuns) {
		return &run.PackageRuns[run.CurrentPkgIndex]
	}
	// Fall back to the first non-terminal package; if all terminal, use first.
	for i := range run.PackageRuns {
		switch run.PackageRuns[i].State {
		case runner.StateVerified, runner.StateCompletedVerified,
			runner.StateFailed, runner.StateFailedNoProgress,
			runner.StateFailedBudgetExceeded, runner.StateFailedVerification,
			runner.StateCanceledByUser:
			continue
		default:
			return &run.PackageRuns[i]
		}
	}
	return &run.PackageRuns[0]
}
