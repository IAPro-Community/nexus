package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/memoryrecovery"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

// maestroMemoryMigrateCmd exposes the Maestro canonical memory migration:
// plan (read-only dry-run), apply (confirmed export + read-back + gate) and
// recover (resume an interrupted operation).
func maestroMemoryMigrateCmd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: nexus maestro memory-migrate <plan|apply|recover> --workspace <path> [--reason <text>] [--confirm]")
	}
	mode := args[0]
	workspace := ""
	reason := ""
	confirmed := false
	asJSON := false
	for index := 1; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--json":
			asJSON = true
		case arg == "--confirm":
			confirmed = true
		case arg == "--workspace" || arg == "--reason":
			if index+1 >= len(args) {
				return fmt.Errorf("memory-migrate option %q requires a value", arg)
			}
			index++
			if arg == "--workspace" {
				workspace = args[index]
			} else {
				reason = args[index]
			}
		case strings.HasPrefix(arg, "--workspace="):
			workspace = strings.TrimPrefix(arg, "--workspace=")
		case strings.HasPrefix(arg, "--reason="):
			reason = strings.TrimPrefix(arg, "--reason=")
		default:
			return fmt.Errorf("unknown memory-migrate option %q", arg)
		}
	}
	if strings.TrimSpace(workspace) == "" {
		return errors.New("memory-migrate requires --workspace <path>")
	}
	n := nexus.Default()
	switch mode {
	case "plan":
		plan, err := n.PlanMemoryMigration(context.Background(), workspace)
		if err != nil {
			return err
		}
		return printMemoryMigrationJSON(plan, asJSON)
	case "apply", "recover":
		if !confirmed {
			return errors.New("memory-migrate apply/recover requires --confirm")
		}
		if strings.TrimSpace(reason) == "" {
			return errors.New("memory-migrate apply/recover requires --reason <text>")
		}
		var result *nexus.MemoryMigrationResult
		var err error
		if mode == "apply" {
			result, err = n.MigrateIntelligenceMemoryToMaestro(context.Background(), workspace, confirmed, reason)
		} else {
			result, err = n.RecoverMemoryMigration(context.Background(), workspace, confirmed, reason)
		}
		if err != nil {
			return err
		}
		return printMemoryMigrationJSON(result, asJSON)
	default:
		return fmt.Errorf("unknown memory-migrate mode %q; use plan, apply, or recover", mode)
	}
}

func printMemoryMigrationJSON(value any, asJSON bool) error {
	if asJSON {
		encoded, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

// Only status inspection is exposed. Starting, advancing, unlocking and
// applying a restore are deliberately NOT CLI commands in this phase.
func maestroMemoryRecoveryStatusCmd(args []string) error {
	asJSON := false
	for _, arg := range args {
		switch arg {
		case "--json":
			if asJSON {
				return errors.New("duplicate --json")
			}
			asJSON = true
		default:
			return fmt.Errorf("unknown memory-recovery-status option %q", arg)
		}
	}
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}
	report, err := memoryrecovery.Inspect(dataDir)
	if err != nil {
		return err
	}
	if asJSON {
		bytes, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(bytes))
		if report.Blocked {
			return errors.New("recovery journal requires manual review")
		}
		return nil
	}
	fmt.Printf("Nexus recovery journal: %d operation(s)\n", len(report.Operations))
	if report.Blocked {
		fmt.Println("Recovery blocked: unfinished, locked or corrupted journal requires manual review")
	} else {
		fmt.Println("Recovery journal: no unresolved operations")
	}
	for _, op := range report.Operations {
		fmt.Printf("  %s: %s (%d events)\n", op.ID, op.Stage, op.Events)
	}
	for _, problem := range report.Problems {
		fmt.Printf("  integrity problem: %s\n", problem)
	}
	if report.Blocked {
		return errors.New("recovery journal requires manual review")
	}
	return nil
}
