package app

// `nexus doctor` (environment diagnostics with --json/--bundle) and
// `nexus security` (profile isolation audit with --json). Split out of app.go
// without changing flags, JSON schemas or bundle contents.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/buildinfo"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/security"
	diagnostic "github.com/IAPro-Community/nexus/internal/doctor"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/profile"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

func doctorCmd(args []string) error {
	if len(args) > 0 && args[0] == "terminal" {
		return terminalDoctorCmd(args[1:])
	}
	jsonOutput := false
	bundleOutput := false
	storageOnly := false
	pruneCaches := false
	applyPrune := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		case "--bundle":
			bundleOutput = true
		case "--storage":
			storageOnly = true
		case "--prune-caches":
			pruneCaches = true
		case "--apply":
			applyPrune = true
		}
	}

	if storageOnly || pruneCaches {
		dataDir, err := config.DataDir()
		if err != nil {
			return fmt.Errorf("resolve data dir: %w", err)
		}
		var rep diagnostic.StorageReport
		if pruneCaches {
			rep, err = diagnostic.PruneProfileStorage(dataDir, diagnostic.PruneOptions{
				DryRun:               !applyPrune,
				IncludeGhostProfiles: true,
				MinTempAge:           time.Hour,
			})
		} else {
			rep, err = diagnostic.InspectProfileStorage(dataDir)
		}
		if err != nil {
			return err
		}
		if jsonOutput {
			b, _ := json.MarshalIndent(rep, "", "  ")
			fmt.Println(string(b))
			return nil
		}
		mode := "AUDIT (dry-run)"
		if !rep.DryRun {
			mode = "APPLIED"
		}
		fmt.Printf("=== Nexus Profile Storage [%s] ===\n", mode)
		fmt.Printf("Data dir:    %s\n", rep.DataDir)
		fmt.Printf("Total:       %.2f MiB (Reclaimable: %.2f MiB, Freed: %.2f MiB)\n",
			float64(rep.TotalBytes)/(1024*1024),
			float64(rep.ReclaimableBytes)/(1024*1024),
			float64(rep.FreedBytes)/(1024*1024),
		)
		for _, p := range rep.Profiles {
			flag := ""
			if !p.ValidName {
				flag = " [INVALID/GHOST]"
			}
			fmt.Printf("  - %-10s %-26s %8.2f MiB (reclaimable: %7.2f MiB, symlinks: %d)%s\n",
				p.Provider, p.Profile,
				float64(p.TotalBytes)/(1024*1024),
				float64(p.Reclaimable)/(1024*1024),
				p.SymlinksFound, flag)
		}
		fmt.Printf("Candidates (%d):\n", len(rep.Candidates))
		for _, c := range rep.Candidates {
			status := "would-remove"
			if c.Removed {
				status = "removed"
			} else if c.Skipped != "" {
				status = "skipped: " + c.Skipped
			}
			fmt.Printf("  - [%s] %-14s %8.2f MiB %s\n", status, c.Kind, float64(c.Bytes)/(1024*1024), c.Path)
		}
		if pruneCaches && !applyPrune && len(rep.Candidates) > 0 {
			fmt.Println("\nDry-run only. Re-run with --prune-caches --apply to reclaim disk space safely.")
		}
		return nil
	}

	reg := initRegistry()
	ctx := context.Background()
	detections := reg.DetectAll(ctx)
	report := diagnostic.BuildReport(buildinfo.Version, detections, nexusruntime.DefaultCredentialIsolator().Capability())
	if bundleOutput {
		path := filepath.Join(os.TempDir(), fmt.Sprintf("nexus-doctor-%d.zip", time.Now().UnixNano()))
		if err := report.WriteBundle(path); err != nil {
			return fmt.Errorf("write doctor bundle: %w", err)
		}
		fmt.Println(path)
		return nil
	}

	if jsonOutput {
		b, _ := report.JSON()
		fmt.Println(string(b))
		return nil
	}

	fmt.Println("=== IAPro Nexus Diagnostics ===")
	for id, det := range detections {
		if det.Installed {
			fmt.Printf("✓ %-12s installed (%s) at %s\n", id, det.Version, det.BinaryPath)
		} else {
			fmt.Printf("✗ %-12s not installed (%s)\n", id, det.Error)
		}
	}
	fmt.Println("\n=== Platform Checks ===")
	for _, check := range report.Checks {
		fmt.Printf("%s %-28s %s\n", check.Status, check.ID, check.Summary)
	}

	ps, _ := profile.List()

	fmt.Println("\n=== Profile & Auth Status ===")
	for _, p := range ps {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		if acc.Authenticated {
			fmt.Printf("✓ %s:%s authenticated (%s)\n", p.Provider, p.Name, acc.Email)
		} else {
			fmt.Printf("⚠ %s:%s unauthenticated (%s)\n", p.Provider, p.Name, acc.Status)
		}
	}

	fmt.Println("\n=== Orquestrador Maestro ===")
	maestroClient := nexus.NewMaestroClient()
	mStatus := maestroClient.Status()
	if mStatus.Available {
		ver := "unknown"
		if mStatus.Capabilities != nil {
			ver = mStatus.Capabilities.Version
		}
		fmt.Printf("✓ Maestro status: AVAILABLE (v%s, mode: %s)\n", ver, mStatus.Mode)
		if mStatus.Capabilities != nil && len(mStatus.Capabilities.Skills) > 0 {
			fmt.Printf("  • Integrated skills: %s\n", strings.Join(mStatus.Capabilities.SkillIDs(), ", "))
		}
	} else {
		fmt.Printf("⚠ Maestro status: DEGRADED (%s)\n", mStatus.Error)
		fmt.Println("  Run: npm install -g @iapro/orquestrador-maestro-cli")
	}

	return nil
}

func securityCmd(args []string) error {
	cfg, _ := config.LoadConfig()
	ps, _ := profile.List()
	hostHome := security.FindHostHome()

	if len(args) > 0 && args[0] == "--json" {
		var audits []security.SecurityAudit
		for _, p := range ps {
			home, _ := config.ProfileHome(p.Provider, p.Name)
			audits = append(audits, security.AuditProfile(p.Provider, p.Name, home, hostHome, cfg.IsolationPreset))
		}
		b, _ := json.MarshalIndent(audits, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Isolation Preset: %s\n\n", cfg.IsolationPreset)
	for _, p := range ps {
		home, _ := config.ProfileHome(p.Provider, p.Name)
		audit := security.AuditProfile(p.Provider, p.Name, home, hostHome, cfg.IsolationPreset)
		fmt.Printf("Profile: %s\n", audit.Profile)
		fmt.Println("  Shared:")
		for _, s := range audit.Shared {
			fmt.Printf("    ✓ %s\n", s)
		}
		fmt.Println("  Protected / Not shared:")
		for _, pt := range audit.Protected {
			fmt.Printf("    ✓ %s\n", pt)
		}
		if len(audit.Warnings) > 0 {
			fmt.Println("  Warnings:")
			for _, w := range audit.Warnings {
				fmt.Printf("    ⚠ %s\n", w)
			}
		}
		fmt.Println()
	}
	return nil
}
