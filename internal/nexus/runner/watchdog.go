package runner

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

const defaultStallTimeout = 15 * time.Minute

var inFlightWatchStates = map[State]struct{}{
	StateAllocating:  {},
	StateExecuting:   {},
	StateCompiling:   {},
	StateTesting:     {},
	StateReviewing:   {},
	StateVerifying:   {},
	StateRemediating: {},
}

func StallTimeout(contract AutonomyContract) time.Duration {
	if contract.StallTimeoutSeconds < 0 {
		return 0
	}
	if contract.StallTimeoutSeconds == 0 {
		return defaultStallTimeout
	}
	return time.Duration(contract.StallTimeoutSeconds) * time.Second
}

func progressAnchor(run *MissionRun) time.Time {
	if run == nil {
		return time.Time{}
	}
	if !run.LastProgressAt.IsZero() {
		return run.LastProgressAt
	}
	return run.StartedAt
}

func runIsInFlight(run *MissionRun) bool {
	if run == nil {
		return false
	}
	if _, ok := inFlightWatchStates[run.State]; ok {
		return true
	}
	for i := range run.PackageRuns {
		if _, ok := inFlightWatchStates[run.PackageRuns[i].State]; ok {
			return true
		}
	}
	return false
}

// ApplyProgressWatchdog fails closed when an in-flight Mission has produced
// no durable progress within StallTimeout. Heartbeats and UpdatedAt are not
// progress. FAILED_NO_PROGRESS is not converted into NEEDS_YOU.
func ApplyProgressWatchdog(run *MissionRun, now time.Time) bool {
	if run == nil || isTerminalRunState(run.State) || run.State == StatePaused || run.State == StateBlockedNeedsUser {
		return false
	}
	timeout := StallTimeout(run.Contract)
	if timeout <= 0 {
		return false
	}
	if !runIsInFlight(run) {
		return false
	}
	anchor := progressAnchor(run)
	if anchor.IsZero() {
		anchor = now
		run.LastProgressAt = now
		return false
	}
	if now.Sub(anchor) < timeout {
		return false
	}
	run.State = StateFailedNoProgress
	run.PausedReason = "watchdog: no durable progress within stall timeout"
	run.UpdatedAt = now
	for i := range run.PackageRuns {
		if _, ok := inFlightWatchStates[run.PackageRuns[i].State]; ok {
			run.PackageRuns[i].State = StateFailed
			run.PackageRuns[i].ErrorMessage = run.PausedReason
			run.PackageRuns[i].FinishedAt = &now
		}
	}
	return true
}

// armStallWatchdog cancels the in-flight operation context when the stall
// deadline elapses during a blocked provider/allocate/verify call — not only
// on the next ExecuteNextStep entry.
func armStallWatchdog(parent context.Context, run *MissionRun) (context.Context, context.CancelFunc) {
	timeout := StallTimeout(run.Contract)
	if timeout <= 0 || parent == nil {
		return parent, func() {}
	}
	anchor := progressAnchor(run)
	if anchor.IsZero() {
		anchor = time.Now().UTC()
	}
	deadline := anchor.Add(timeout)
	ctx, cancel := context.WithCancel(parent)
	go func() {
		delay := time.Until(deadline)
		if delay < 0 {
			delay = 0
		}
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			cancel()
		}
	}()
	return ctx, cancel
}

func (r *MissionRunner) finalizeStallIfNeeded(ctx context.Context, run *MissionRun, err error) (bool, error) {
	if err == nil || run == nil {
		return false, nil
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return false, nil
	}
	now := time.Now().UTC()
	if !ApplyProgressWatchdog(run, now) {
		// Context canceled for another reason (lease loss / package timeout).
		return false, nil
	}
	if saveErr := r.saveRun(ctx, run); saveErr != nil {
		return true, saveErr
	}
	return true, fmt.Errorf("mission stalled: no durable progress within stall timeout")
}

func progressFingerprint(run *MissionRun) string {
	if run == nil {
		return ""
	}
	buf := string(run.State)
	for i := range run.PackageRuns {
		pkg := &run.PackageRuns[i]
		buf += "|" + pkg.PackageID + ":" + string(pkg.State) + ":" + string(pkg.DispatchState)
		buf += ":" + pkg.StrategyVariant
		if pkg.WorkReceipt != nil {
			buf += ":receipt"
		}
		buf += ":" + strconv.Itoa(pkg.NoProgressCount) + ":" + strconv.Itoa(len(pkg.Verifications))
	}
	if run.NeedsHuman != nil {
		buf += ":h" + run.NeedsHuman.ID
	}
	return buf
}
