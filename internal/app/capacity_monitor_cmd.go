package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/nexus"
	"github.com/kivervinicius/ai-cli/internal/nexus/capacitymonitor"
	"github.com/kivervinicius/ai-cli/internal/profile"
)

func capacityMonitorCmd(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Println("Usage: nexus capacity-monitor [run|status|enable-autostart|disable-autostart] [--json]")
		return nil
	}

	autostartMgr := capacitymonitor.NewAutostartManager()

	switch args[0] {
	case "status":
		asStatus, _ := autostartMgr.Status(context.Background())
		for _, a := range args[1:] {
			if a == "--json" {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(asStatus)
			}
		}
		fmt.Printf("Autostart Supported: %v\n", asStatus.Supported)
		fmt.Printf("Autostart Installed: %v\n", asStatus.Installed)
		fmt.Printf("Autostart Enabled:   %v\n", asStatus.Enabled)
		fmt.Printf("Autostart Active:    %v\n", asStatus.Active)
		if asStatus.UnitPath != "" {
			fmt.Printf("Unit Path:           %s\n", asStatus.UnitPath)
		}
		return nil

	case "enable-autostart":
		execPath, err := os.Executable()
		if err != nil {
			execPath = "nexus"
		}
		if err := autostartMgr.InstallAndEnable(context.Background(), execPath); err != nil {
			return fmt.Errorf("failed to enable autostart: %w", err)
		}
		fmt.Println("Capacity monitor autostart enabled successfully.")
		return nil

	case "disable-autostart":
		if err := autostartMgr.DisableAndRemove(context.Background()); err != nil {
			return fmt.Errorf("failed to disable autostart: %w", err)
		}
		fmt.Println("Capacity monitor autostart disabled and removed.")
		return nil

	case "run":
		n := nexus.Default()
		st, err := n.OpenProject()
		if err != nil {
			return fmt.Errorf("failed to open store: %w", err)
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		probeReg := profile.NewProbeRegistry()
		deps := capacitymonitor.Dependencies{
			Profiles: func() ([]model.Profile, error) {
				return profile.List()
			},
			ProbeRegistry:    probeReg,
			ActivityDetector: capacitymonitor.NewActivityDetector(),
			PowerState:       capacitymonitor.NewPowerState(),
			Store:            st,
			OwnerID:          fmt.Sprintf("daemon:%d", os.Getpid()),
		}

		mon := capacitymonitor.New(deps)
		return mon.Run(ctx)

	default:
		return fmt.Errorf("unknown capacity-monitor subcommand: %s", args[0])
	}
}
