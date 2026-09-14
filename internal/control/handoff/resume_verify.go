package handoff

import (
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

// VerifyResumeContinuity performs runtime-level verification that a target
// runtime genuinely resumed the intended provider session.
//
// It proves:
//  1. The target runtime process is alive.
//  2. The target runtime reached RUNNING state.
//  3. The source session ID is present.
//  4. The resume arguments actually referenced the source session ID.
//
// It does NOT claim provider-level confirmation (e.g. the provider answering
// "yes, session X is active"): that is only possible through a CONTROL_API
// adapter and is reported separately in the capability matrix.
func VerifyResumeContinuity(sess *registry.RuntimeSession, resumeArgs []string, sessionID string) (bool, string) {
	if sess == nil || strings.TrimSpace(sess.RuntimeID) == "" {
		return false, "target runtime is missing"
	}
	if strings.TrimSpace(sessionID) == "" {
		return false, "source session ID is empty"
	}
	if sess.PID <= 0 {
		return false, "target process PID is unknown"
	}
	if !registry.IsProcessAlive(sess.PID) {
		return false, "target process is not alive"
	}
	if sess.State != registry.StateRunning {
		return false, fmt.Sprintf("target runtime state is %s, want %s", sess.State, registry.StateRunning)
	}
	if !resumeArgsReferenceSession(resumeArgs, sessionID) {
		return false, "resume arguments do not reference the source session ID"
	}
	return true, ""
}

// resumeArgsReferenceSession reports whether any resume argument references the
// given session ID either as an exact token or as a value-bearing substring.
func resumeArgsReferenceSession(resumeArgs []string, sessionID string) bool {
	for _, a := range resumeArgs {
		if a == sessionID {
			return true
		}
		if strings.Contains(a, sessionID) {
			return true
		}
	}
	return false
}
