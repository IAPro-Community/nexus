package doctor

import (
	"os"
	stdruntime "runtime"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	controlterminal "github.com/IAPro-Community/nexus/internal/control/terminal"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
	"golang.org/x/term"
)

type TerminalRuntimeReport struct {
	RuntimeID        string `json:"runtime_id"`
	State            string `json:"state"`
	HostPID          int    `json:"host_pid,omitempty"`
	HostGeneration   int64  `json:"host_generation,omitempty"`
	HostLive         bool   `json:"host_live"`
	BinaryGeneration string `json:"binary_generation,omitempty"`
	BinaryStatus     string `json:"binary_status"`
	Attachable       bool   `json:"attachable"`
	RecoveryAction   string `json:"recovery_action"`
}

type TerminalReport struct {
	Schema                   string                  `json:"schema"`
	Platform                 string                  `json:"platform"`
	IsTTY                    bool                    `json:"is_tty"`
	TermiosStateReadable     bool                    `json:"termios_state_readable"`
	KeyboardModes            map[string]string       `json:"keyboard_modes"`
	CodexKeyboardEnhancement string                  `json:"codex_tui_disable_keyboard_enhancement"`
	AttachedRuntimeID        string                  `json:"attached_runtime_id,omitempty"`
	Runtimes                 []TerminalRuntimeReport `json:"runtimes"`
	RecoveryAction           string                  `json:"recovery_action"`
}

// BuildTerminalReport reports the state Nexus can verify without claiming
// that private terminal modes are queryable. The effective private-mode state
// is the idempotent reset emitted on entry/recovery; termios readability is
// reported separately so users can distinguish an unverified TTY from a safe
// one.
func BuildTerminalReport(fd int, sessions []registry.RuntimeSession) TerminalReport {
	isTTY := term.IsTerminal(fd)
	termiosReadable := false
	if isTTY {
		_, err := term.GetState(fd)
		termiosReadable = err == nil
	}
	value := os.Getenv(nexusruntime.EnvCodexDisableKeyboardEnhancement)
	if value == "" {
		value = "unset (Nexus injects 1 for Codex runtimes)"
	}

	report := TerminalReport{
		Schema:                   "nexus.doctor.terminal/v1",
		Platform:                 stdruntime.GOOS + "/" + stdruntime.GOARCH,
		IsTTY:                    isTTY,
		TermiosStateReadable:     termiosReadable,
		CodexKeyboardEnhancement: value,
		AttachedRuntimeID:        strings.TrimSpace(os.Getenv("NEXUS_RUNTIME_ID")),
		KeyboardModes: map[string]string{
			"application_cursor": "disabled by Nexus reset",
			"kitty_csi_u":        "disabled by Nexus reset",
			"modify_other_keys":  "disabled by Nexus reset",
			"focus_reporting":    "disabled by Nexus reset",
			"bracketed_paste":    "disabled by Nexus reset",
		},
		RecoveryAction: "nexus doctor terminal --recover (or reattach; recovery is safe to repeat)",
		Runtimes:       make([]TerminalRuntimeReport, 0, len(sessions)),
	}
	for _, session := range sessions {
		if !session.IsActive() && session.State != registry.StateStale {
			continue
		}
		item := TerminalRuntimeReport{
			RuntimeID:      session.RuntimeID,
			State:          string(session.State),
			HostPID:        session.HostPID,
			HostGeneration: session.HostGeneration,
			HostLive:       session.HostLive(),
			BinaryStatus:   "incompatible",
			RecoveryAction: "recreate with nexus codex --yolo; old process is not terminated automatically",
		}
		if session.HostBinaryFingerprint != "" {
			fingerprint := session.HostBinaryFingerprint
			if len(fingerprint) > 12 {
				fingerprint = fingerprint[:12]
			}
			item.BinaryGeneration = fingerprint
		}
		if err := registry.ValidateAttachCompatibility(session); err == nil {
			item.BinaryStatus = "current-and-compatible"
			item.Attachable = true
			item.RecoveryAction = "none; attach is safe"
		} else if !item.HostLive {
			item.BinaryStatus = "stale-host-or-generation"
		}
		report.Runtimes = append(report.Runtimes, item)
	}
	return report
}

func RecoverTerminal(w *os.File) error {
	if w == nil || !term.IsTerminal(int(w.Fd())) {
		return nil
	}
	return controlterminal.ResetKeyboardModes(w)
}
