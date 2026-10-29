package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

// maestroCmd is deliberately separate from updateCmd. Nexus updates are
// handled by the shared Update Service; Maestro is optional and may only be
// inspected or changed through this explicit namespace.
func maestroCmd(args []string) error {
	if len(args) > 0 && (args[0] == "memory-preview" || args[0] == "memory-backup" || args[0] == "memory-restore-preview" || args[0] == "memory-restore-stage") {
		return maestroMemoryMigrationCmd(args)
	}
	subcmd := "status"
	asJSON := false
	for _, arg := range args {
		switch {
		case arg == "--json":
			asJSON = true
		case !strings.HasPrefix(arg, "-") && subcmd == "status":
			subcmd = arg
		}
	}

	if subcmd != "status" && subcmd != "doctor" && subcmd != "update" {
		return fmt.Errorf("unknown maestro command %q; use status, doctor, update, memory-preview, or memory-backup", subcmd)
	}

	if subcmd == "update" {
		result := PerformSystemUpdate()
		if asJSON {
			encoded, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(encoded))
		}
		return nil
	}

	status := nexus.NewMaestroClient().Status()
	if asJSON {
		encoded, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}

	if status.Available {
		version := "unknown"
		if status.Capabilities != nil && status.Capabilities.Version != "" {
			version = status.Capabilities.Version
		}
		fmt.Printf("Maestro: available (v%s)\n", version)
		fmt.Printf("Mode: %s\n", status.Mode)
		return nil
	}

	fmt.Println("Maestro: unavailable (Nexus remains operational in degraded mode)")
	if status.Error != "" {
		fmt.Printf("Reason: %s\n", status.Error)
	}
	return nil
}
