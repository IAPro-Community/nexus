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
	channel := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			asJSON = true
		case strings.HasPrefix(a, "--channel="):
			channel = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(a, "--channel=")))
		case a == "--channel":
			if i+1 >= len(args) {
				return fmt.Errorf("--channel requires beta or stable")
			}
			channel = strings.ToLower(strings.TrimSpace(args[i+1]))
			i++
		case strings.HasPrefix(a, "-"):
			return fmt.Errorf("unknown update option %q", a)
		case subcmd == "":
			subcmd = strings.ToLower(strings.TrimSpace(a))
		default:
			return fmt.Errorf("unexpected update argument %q", a)
		}
	}
	if channel != "" && channel != "beta" && channel != "stable" {
		return fmt.Errorf("invalid update channel %q; use beta or stable", channel)
	}
	if subcmd != "" && subcmd != "check" {
		return fmt.Errorf("unknown update subcommand %q; use check or omit it to apply", subcmd)
	}

	execP, _ := os.Executable()
	svc := update.NewService(update.ServiceConfig{
		CurrentVer: buildinfo.Version,
		Channel:    channel,
		ExecPath:   execP,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if subcmd == "check" {
		res, err := svc.Check(ctx)
		if err != nil {
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
