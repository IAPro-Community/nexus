package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/buildinfo"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/update"
)

// UpdateResult summarizes explicit Maestro maintenance; `nexus update` uses
// the shared Update Service and does not call this operation.
type UpdateResult = nexus.UpdateResult

func updateCmd(args []string) error {
	subcmd := ""
	asJSON := false
	for _, a := range args {
		if a == "--json" {
			asJSON = true
		} else if !strings.HasPrefix(a, "-") && subcmd == "" {
			subcmd = a
		}
	}

	execP, _ := os.Executable()
	svc := update.NewService(update.ServiceConfig{
		CurrentVer: buildinfo.Version,
		ExecPath:   execP,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if subcmd == "check" {
		res, err := svc.Check(ctx)
		if err != nil && res == nil {
			return fmt.Errorf("update check failed: %w", err)
		}
		if asJSON {
			b, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(b))
			return nil
		}
		fmt.Println("=== IAPro Nexus Update Check ===")
		fmt.Printf("Installed:           %s\n", res.CurrentVersion)
		fmt.Printf("Latest:              %s\n", res.LatestVersion)
		fmt.Printf("Channel:             %s\n", res.Channel)
		fmt.Printf("Installation Method: %s\n", res.InstallationMethod)
		if res.UpdateAvailable {
			fmt.Printf("Update Available:    YES\n")
			if !res.AllowsSelfUpdate {
				fmt.Printf("Upgrade Action:      %s\n", res.Instruction)
			} else {
				fmt.Println("Run 'nexus update' to download and apply.")
			}
		} else {
			fmt.Println("Status:              Up to date.")
		}
		return nil
	}

	check, err := svc.Check(ctx)
	if err == nil && !check.AllowsSelfUpdate {
		if asJSON {
			b, _ := json.MarshalIndent(check, "", "  ")
			fmt.Println(string(b))
			return nil
		}
		fmt.Println("=== IAPro Nexus Updater ===")
		fmt.Printf("Installed:           %s\n", check.CurrentVersion)
		fmt.Printf("Latest:              %s\n", check.LatestVersion)
		fmt.Printf("Installation Method: %s\n", check.InstallationMethod)
		fmt.Printf("\nDirect self-update is disabled for package-managed installations.\n")
		fmt.Printf("Please upgrade using your package manager:\n  %s\n", check.Instruction)
		return nil
	}

	fmt.Println("=== IAPro Nexus Updater ===")
	fmt.Println("Checking and applying updates...")

	receipt, err := svc.Apply(ctx)
	if err != nil {
		return fmt.Errorf("update could not be applied: %w", err)
	}

	if asJSON {
		b, _ := json.MarshalIndent(receipt, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("\n✓ Successfully updated to Nexus %s!\n", receipt.AppliedVersion)
	return nil
}

// PerformSystemUpdate updates the Maestro library. The Nexus binary is not replaced.
func PerformSystemUpdate() UpdateResult {
	return nexus.PerformSystemUpdate()
}
