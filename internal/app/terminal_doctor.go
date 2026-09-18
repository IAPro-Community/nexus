package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	diagnostic "github.com/IAPro-Community/nexus/internal/doctor"
)

func terminalDoctorCmd(args []string) error {
	jsonOutput := false
	recoverTerminal := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		case "--recover":
			recoverTerminal = true
		default:
			return fmt.Errorf("unknown terminal doctor flag %q; use --json or --recover", arg)
		}
	}
	if recoverTerminal {
		if err := diagnostic.RecoverTerminal(os.Stdout); err != nil {
			return err
		}
	}

	report := diagnostic.BuildTerminalReport(int(os.Stdin.Fd()), registry.DefaultRegistry().List())
	if jsonOutput {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("=== Nexus Terminal Diagnostics ===")
	fmt.Printf("Platform:                    %s\n", report.Platform)
	fmt.Printf("TTY:                        %t (termios readable: %t)\n", report.IsTTY, report.TermiosStateReadable)
	fmt.Printf("CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT: %s\n", report.CodexKeyboardEnhancement)
	if strings.TrimSpace(report.AttachedRuntimeID) == "" {
		fmt.Println("Attached runtime:            none reported by this shell")
	} else {
		fmt.Printf("Attached runtime:            %s\n", report.AttachedRuntimeID)
	}
	fmt.Println("Keyboard modes after reset:")
	for _, mode := range []string{"application_cursor", "kitty_csi_u", "modify_other_keys", "focus_reporting", "bracketed_paste"} {
		fmt.Printf("  %-24s %s\n", mode, report.KeyboardModes[mode])
	}
	if len(report.Runtimes) == 0 {
		fmt.Println("Runtime generations:         no active supervised runtimes")
	} else {
		fmt.Println("Runtime generations:")
		for _, runtime := range report.Runtimes {
			fmt.Printf("  %-22s state=%-10s host_pid=%d generation=%d binary=%s attachable=%t\n",
				runtime.RuntimeID, runtime.State, runtime.HostPID, runtime.HostGeneration,
				runtime.BinaryStatus, runtime.Attachable)
			if !runtime.Attachable {
				fmt.Printf("    action: %s\n", runtime.RecoveryAction)
			}
		}
	}
	if recoverTerminal {
		if report.IsTTY {
			fmt.Println("Recovery:                    keyboard reset emitted")
		} else {
			fmt.Println("Recovery:                    not emitted (stdout is not a TTY)")
		}
	} else {
		fmt.Printf("Recovery action:             %s\n", report.RecoveryAction)
	}
	return nil
}
