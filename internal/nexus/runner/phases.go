package runner

import (
	"sort"
	"strings"
	"time"
)

const (
	phaseReview         = "REVIEW"
	phaseSecurityReview = "SECURITY_REVIEW"
	phaseVerification   = "VERIFICATION"
	phaseImplementation = "IMPLEMENTATION"
)

func reviewRequiredForPackage(spec PackageSpec) bool {
	if spec.ReviewRequired {
		return true
	}
	switch strings.ToUpper(strings.TrimSpace(spec.PhaseType)) {
	case "", phaseReview, phaseSecurityReview:
		return true
	default:
		return false
	}
}

func isReviewPhase(pkg *PackageRun) bool {
	if pkg == nil {
		return false
	}
	kind := strings.ToUpper(strings.TrimSpace(pkg.PhaseType))
	return kind == phaseReview || kind == phaseSecurityReview
}

func isVerificationPhase(pkg *PackageRun) bool {
	return pkg != nil && strings.EqualFold(strings.TrimSpace(pkg.PhaseType), phaseVerification)
}

func refreshPhaseRuns(run *MissionRun) {
	if run == nil {
		return
	}
	groups := make(map[string]*PhaseRun)
	phaseStates := make(map[string][]State)
	order := make([]string, 0)
	for _, pkg := range run.PackageRuns {
		phaseID := strings.TrimSpace(pkg.PhaseID)
		if phaseID == "" {
			phaseID = pkg.PackageID
		}
		phase, exists := groups[phaseID]
		if !exists {
			phase = &PhaseRun{ID: phaseID, Type: pkg.PhaseType, State: StatePending, UpdatedAt: time.Now().UTC()}
			groups[phaseID] = phase
			order = append(order, phaseID)
		}
		phase.PackageIDs = append(phase.PackageIDs, pkg.PackageID)
		if pkg.AssignedAgent != "" && !containsString(phase.AgentIDs, pkg.AssignedAgent) {
			phase.AgentIDs = append(phase.AgentIDs, pkg.AssignedAgent)
		}
		if pkg.RoutingDecisionJSON != "" {
			phase.RoutingReasons = append(phase.RoutingReasons, pkg.RoutingDecisionJSON)
		}
		if pkg.HandoffJSON != "" && phase.HandoffJSON == "" {
			phase.HandoffJSON = pkg.HandoffJSON
		}
		if pkg.Attempt > phase.Attempt {
			phase.Attempt = pkg.Attempt
		}
		phaseStates[phaseID] = append(phaseStates[phaseID], pkg.State)
		phase.UpdatedAt = time.Now().UTC()
	}
	for _, id := range order {
		phase := groups[id]
		phase.State = aggregatePhaseState(phaseStates[id])
		phase.AgentIDs = uniqueSorted(phase.AgentIDs)
		phase.RoutingReasons = uniqueSorted(phase.RoutingReasons)
	}
	run.PhaseRuns = make([]PhaseRun, 0, len(order))
	for _, id := range order {
		run.PhaseRuns = append(run.PhaseRuns, *groups[id])
	}
	run.CurrentPhaseID = ""
	for _, phase := range run.PhaseRuns {
		if phase.State != StateVerified {
			run.CurrentPhaseID = phase.ID
			break
		}
	}
	if run.State == StateCompletedVerified && len(run.PhaseRuns) > 0 {
		run.CurrentPhaseID = run.PhaseRuns[len(run.PhaseRuns)-1].ID
	}
}

func aggregatePhaseState(states []State) State {
	if len(states) == 0 {
		return StatePending
	}
	for _, state := range states {
		if strings.HasPrefix(string(state), "FAILED") || state == StateBlockedNeedsUser || state == StateEscalated || state == StateCanceledByUser {
			return state
		}
	}
	active := []State{StateExecuting, StateTesting, StateReviewing, StateVerifying, StateRemediating, StateAllocating, StateCompiling, StatePaused}
	for _, wanted := range active {
		for _, state := range states {
			if state == wanted {
				return wanted
			}
		}
	}
	for _, state := range states {
		if state == StateReady {
			return StateReady
		}
	}
	for _, state := range states {
		if state != StateVerified {
			return state
		}
	}
	return StateVerified
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
