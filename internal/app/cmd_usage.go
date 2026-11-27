package app

// Quota, history and statistics commands plus the usage help screen. Split out
// of app.go without changing flags, JSON shapes or localization keys.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	sdkproviders "github.com/IAPro-Community/IAPro-SDK/providers"
	"github.com/IAPro-Community/nexus/internal/buildinfo"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/core/telemetry"
	"github.com/IAPro-Community/nexus/internal/localization"
	"github.com/IAPro-Community/nexus/internal/profile"
	"github.com/IAPro-Community/nexus/internal/tui"
)

func usageCmd(args []string) error {
	ps, err := profile.List()
	if err != nil {
		return err
	}
	// Asking for quota is an explicit intent to use the account. Profiles that
	// were never authenticated can inherit the host CLI login once; the notice
	// keeps the link from being silent, and an explicit logout opts out.
	for _, p := range ps {
		if src, linked := profile.EnsureHostCredential(p.Provider, p.Name); linked {
			fmt.Fprintf(os.Stderr,
				"Conta %s vinculada ao perfil %s:%s a partir do login do terminal (%s).\n",
				src.Plan, p.Provider, p.Name, p.Provider)
		}
	}
	if cleaned, _ := profile.DeduplicateAccounts(context.Background()); len(cleaned) > 0 {
		for _, item := range cleaned {
			if len(item.RemovedProfiles) == 0 {
				continue
			}
			label := item.AccountLabel
			if label == "" {
				label = item.Identity
			}
			fmt.Fprintf(os.Stderr,
				"Contas duplicadas removidas em %s (%s): %s (mantido %s).\n",
				item.Provider, label, strings.Join(item.RemovedProfiles, ", "), item.KeptProfile)
		}
		if refreshed, listErr := profile.List(); listErr == nil {
			ps = refreshed
		}
	}

	isJSON := false
	refresh := false
	initialMode := tui.ModeSafe
	initialContinue := false

	for _, arg := range args {
		switch arg {
		case "--json":
			isJSON = true
		case "--refresh":
			refresh = true
		case "--yolo", "-y":
			initialMode = tui.ModeYOLO
		case "--plan":
			initialMode = tui.ModePlan
		case "--continue", "-c":
			initialContinue = true
		}
	}
	if refresh && isJSON {
		fmt.Fprintln(os.Stderr, "Atualizando quotas ao vivo em paralelo…")
		var quotaTargets []model.Profile
		for _, candidate := range ps {
			if s, ok := spec.Get(candidate.Provider); ok && s.Quota == spec.QuotaUnsupported {
				continue
			}
			quotaTargets = append(quotaTargets, candidate)
		}
		_ = profile.RefreshUsageBatch(context.Background(), quotaTargets, sdkproviders.BatchOptions{
			MaxConcurrency:  6,
			PerProbeTimeout: 10 * time.Second,
			GlobalTimeout:   25 * time.Second,
		})
	}

	// Detect installed CLIs across the system so both the TUI and the
	// machine-readable surface can show configured profiles plus installed
	// tools that are awaiting configuration (e.g. Claude Code, Cursor CLI, OpenCode).
	reg := initRegistry()
	ctx := context.Background()
	detections := reg.DetectAll(ctx)

	profCount := make(map[string]int)
	for _, p := range ps {
		profCount[p.Provider]++
	}

	var unconfiguredCLIs []tui.InstalledProviderInfo
	for _, p := range reg.List() {
		id := string(p.ID())
		det := detections[id]
		if det.Installed && profCount[id] == 0 {
			unconfiguredCLIs = append(unconfiguredCLIs, tui.InstalledProviderInfo{
				ID:        id,
				Name:      p.Name(),
				Version:   det.Version,
				Installed: true,
				Profiles:  0,
			})
		}
	}

	if isJSON {
		now := time.Now()
		list := make([]usageJSONRow, 0, len(ps))
		for _, p := range ps {
			snap := profile.GetUsageSnapshot(p.Provider, p.Name)
			acc := profile.GetAccountInfo(p.Provider, p.Name)
			qv := quota.BuildQuotaView(snap, acc.Email, acc.Plan)
			proj := qv.ProjectionAt(now, quota.DefaultTTL, acc.Authenticated, string(acc.Health))
			elig := quota.EvaluateEligibility(quota.EligibilityInput{
				View:          qv,
				Authenticated: acc.Authenticated,
				Health:        string(acc.Health),
			})
			list = append(list, usageJSONRow{
				UsageSnapshot:   snap,
				QuotaView:       qv,
				QuotaProjection: &proj,
				Eligibility:     elig,
				Health:          acc.Health,
				Authenticated:   acc.Authenticated,
			})
		}
		list = append(list, unconfiguredUsageRows(unconfiguredCLIs)...)
		b, _ := json.MarshalIndent(list, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	cfg, _ := config.LoadConfig()
	accs := make(map[string]model.AccountInfo)
	rows := make([]tui.UsageTableRow, 0, len(ps))
	profilesToLoad := make([]model.Profile, 0, len(ps))

	for _, p := range ps {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		accs[p.Provider+":"+p.Name] = acc
		profilesToLoad = append(profilesToLoad, p)
		// Placeholder rows so the TUI opens immediately; async load fills quotas.
		rows = append(rows, tui.UsageTableRow{
			Provider:  p.Provider,
			Profile:   p.Name,
			Account:   acc.Email,
			Plan:      acc.Plan,
			FiveHour:  "…",
			Weekly:    "…",
			Status:    "carregando",
			ModelName: "—",
			IsDefault: cfg.Defaults[p.Provider] == p.Name,
		})
	}

	cwd, _ := os.Getwd()

	opts := tui.UnifiedUsageOptions{
		Rows:              rows,
		UnconfiguredCLIs:  unconfiguredCLIs,
		Sessions:          nil,
		Accounts:          accs,
		Defaults:          cfg.Defaults,
		Workspace:         cwd,
		InitialMode:       initialMode,
		InitialContinue:   initialContinue,
		ProfilesToLoad:    profilesToLoad,
		LoadSessionsAsync: true,
		ForceLiveRefresh:  refresh,
	}

	sel, err := tui.RunUnifiedUsage(opts)
	if err != nil {
		return err
	}
	if sel == nil || sel.Action == tui.ActionQuit || sel.Action == tui.ActionNone {
		return nil
	}

	switch sel.Action {
	case tui.ActionRunProfile:
		return executeProviderWithSmartSelection(sel.Provider, sel.ProfileName, sel.Args)
	case tui.ActionResumeConversation:
		return executeResume(sel.Provider, sel.ProfileName, sel.ConversationID, sel.Args)
	case tui.ActionLogin:
		return loginCmd([]string{sel.Provider, sel.ProfileName})
	case tui.ActionAdd:
		args := []string{sel.Provider}
		if sel.ProfileName != "" {
			args = append(args, sel.ProfileName)
		}
		return addCmd(args)
	default:
		return nil
	}
}

// unconfiguredUsageRows renders installed-but-unconfigured providers for
// `nexus usage --json`, mirroring the TUI rows so every surface shows an
// installed provider (e.g. OpenCode) before its first profile exists.
func unconfiguredUsageRows(clis []tui.InstalledProviderInfo) []usageJSONRow {
	rows := make([]usageJSONRow, 0, len(clis))
	for _, cli := range clis {
		snap := model.UsageSnapshot{
			ProviderID: cli.ID,
			Status:     model.UsageUnknown,
			Source:     model.SourceNone,
			Windows:    []model.UsageWindow{},
			Account:    cli.Name,
			Error:      "sem perfil configurado",
		}
		qv := quota.BuildQuotaView(snap, "", "")
		elig := quota.EvaluateEligibility(quota.EligibilityInput{
			View:          qv,
			Authenticated: false,
			Health:        string(model.HealthUnknown),
		})
		rows = append(rows, usageJSONRow{
			UsageSnapshot: snap,
			QuotaView:     qv,
			Eligibility:   elig,
			Health:        model.HealthUnknown,
			Authenticated: false,
			Unconfigured:  true,
		})
	}
	return rows
}

func quotaGroupStatus(group quota.ModelGroup, snapshotStatus string) string {
	switch snapshotStatus {
	case string(model.UsageUnknown), string(model.UsageError), string(model.UsageUnsupported), "":
		return "SEM EVIDÊNCIA"
	case string(model.UsageEstimated):
		return "ESTIMADA"
	case string(model.UsageRateLimited):
		return "RATE LIMITED"
	}
	knownWindows := 0
	allExhausted := true
	for _, window := range group.Windows {
		if window.Kind == "unknown" {
			continue
		}
		if window.ResetTime != nil && window.ResetTime.Before(time.Now()) {
			continue
		}
		knownWindows++
		if window.Remaining > 0 {
			allExhausted = false
		}
	}
	if knownWindows == 0 {
		return "SEM EVIDÊNCIA"
	}
	if allExhausted {
		return "SEM QUOTA"
	}
	return "COM QUOTA"
}

// quotaUnknownLabel returns a human-readable placeholder for providers
// whose quota data is not available (UNKNOWN, ERROR, etc).
func quotaUnknownLabel(status string) string {
	switch status {
	case string(model.UsageUnknown):
		return "— desconhecido"
	case string(model.UsageError):
		return "— erro"
	case string(model.UsageRateLimited):
		return "— rate limited"
	default:
		return "—"
	}
}

func historyCmd(args []string) error {
	events, err := telemetry.ReadRecentEvents(50)
	if err != nil {
		return err
	}
	if len(args) > 0 && args[0] == "--json" {
		b, _ := json.MarshalIndent(events, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	if len(events) == 0 {
		fmt.Println("No recent history recorded.")
		return nil
	}
	for _, ev := range events {
		fmt.Printf("%s  %-22s  %s:%s  %s\n",
			ev.Timestamp.Format("15:04:05"), ev.Type, ev.ProviderID, ev.ProfileID, ev.Reason)
	}
	return nil
}

func statsCmd(args []string) error {
	s, err := telemetry.ComputeStats(7 * 24 * time.Hour)
	if err != nil {
		return err
	}
	if len(args) > 0 && args[0] == "--json" {
		b, _ := json.MarshalIndent(s, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	fmt.Printf("Local Statistics (Last 7 Days):\n\n")
	fmt.Printf("Total Sessions:   %d\n", s.TotalSessions)
	fmt.Printf("Total Fallbacks:  %d\n", s.TotalFallbacks)
	fmt.Printf("Rate Limits:      %d\n\n", s.TotalRateLimits)
	fmt.Println("Sessions by Provider:")
	for p, count := range s.ByProvider {
		fmt.Printf("  • %-12s %d\n", p, count)
	}
	return nil
}

func usage() {
	p := progName()
	fmt.Println(localization.T("help.header", map[string]any{"Version": buildinfo.Version}))
	fmt.Println(localization.T("help.usage"))
	body := fmt.Sprintf(`
  %s                              Open Web Workspace OS (default)
  %s web [flags]                  Launch IAPro Nexus Workspace OS (Web UI)
  %s web open                     Reopen the running Web UI from any terminal
  %s tunnel                       Remote access via Cloudflare Quick Tunnel
  %s control [subcmd]             Nexus Supervised Agent Runtimes (alias: %s ui)
  %s start <provider> [flags]     Start supervised agent runtime & attach
  %s stop <runtime-id>            Stop running supervised runtime
  %s ps / %s running              List active running supervised runtimes
  %s attach <runtime-id>          Attach terminal to running runtime
  %s handoff <id> <target>        Same-provider account handoff
  %s continue <id> --with <prov>  Cross-provider context handoff
  run "<goal>"                    Classify, route and execute a Nexus task

  %s <provider> [flags]           Launch provider with intelligent account selection
  %s <provider> --supervised      Launch through SessionHost; enables /nexus control
  %s <provider> --lead            Require persistent Lead identity (delegation stays OFF)
  %s <provider>:<profile> [flags] Launch specific profile (e.g. %s codex:work)
  %s <provider>:auto [flags]      Rotate to the next eligible account (alias: :balance)
  %s <provider>:balance [flags]   Persisted round-robin across eligible accounts
  %s resume [id] [provider:name]  Resume previous session using provider-native syntax

  %s providers [--json]           List installed providers, versions & capabilities
  %s providers status [--json]    Show installation and registration lifecycle
  %s providers register <provider> [--profile <name>] [--no-login]
  %s profiles [--json]            List configured profiles, auth status & priorities
  %s paths                        Show Nexus data and configuration paths
  %s add <provider> [name]        Add a new provider authentication profile
  %s import                       Discover & import active CLI sessions from this computer
  %s remove <provider> <name>     Remove an existing profile
  %s rename <provider> <old> <new> Rename a configured provider profile
  %s login <provider> [name]      Run provider official login flow; defaults to the provider profile
  %s logout <provider> <name>     Run provider official logout flow
  %s use <provider> <name>        Set default active profile for provider
  %s status [provider[:profile]]  Display profile health, plan and account status
  %s usage [--json] [--refresh]   Searchable quota table; --json for integrations; --refresh for live CLI fetch
  %s inspect <provider> <name>    Inspect profile configuration details

  %s projects [list|add|prune]    Register, list and prune project workspaces (prune is dry-run unless --apply)
  %s agents [project-id]          List persistent agents for a project
  %s plan [list|create|show|compile|run]
                                  Manage, compile and run WorkPlans

  %s sessions [search] [--json]   Universal session index across all providers
  %s workspaces [--json]          View workspaces, session history & bindings
  %s current                      Show the current active provider/profile
  %s bind <provider>:<profile>    Bind current workspace to a preferred profile
  %s unbind <provider>            Unbind current workspace
  %s bindings [--json]            List all active workspace bindings
  %s explain <provider>           Explain account selection decision and scores
  %s doctor [terminal] [--json|--bundle|--recover]
                                Deep diagnostics; terminal audits TTY modes, generations & recovery
  %s security [profile] [--json]  Audit file sharing and isolation boundary
  %s history [--json]             View local session execution log
  %s stats [--json]               Aggregated statistics (sessions, fallbacks, rate limits)
  %s export [--json]              Export local session data
  nexus cost summary [--days N] [--project-id <id>] [--json]
                                Local cost analytics summary (read-only; same schema as /api/v1/cost/analytics)
  nexus cost analytics [--days N] [--project-id <id>] [--json]
                                Full cost analytics report (read-only; identical schema to /api/v1/cost/analytics)
  %s issue-report                 Create a diagnostic issue report
  %s config <show|validate>       Manage control plane settings
  %s update                       Update Nexus only (use 'maestro update' explicitly for Maestro)
  %s maestro <status|doctor|update> Manage optional Maestro integration explicitly
  nexus maestro memory-recovery-status [--json]
                                  Read-only check for incomplete restoration journals
  nexus maestro memory-migrate <plan|apply|recover> --workspace <path> [--reason <text>] [--confirm] [--json]
                                  Migrate legacy intelligence memory to Maestro canonical memory
  %s completion <bash|zsh|fish>   Generate shell completion scripts
  %s version [--json]             Display build and platform information
  %s release                      Interactively bump, build, install and validate Nexus

Canonical Aliases (translated to native options; Codex keeps short -c/-p native):
  --yolo / -y                     Bypass approval prompts and permissions
  --lead                          Bind Interactive Lead Agent (Mission promotion on submit)
  --continue                      Continue most recent session (AGY/Claude also accept -c)
  --resume <id> / -r <id>         Resume session by ID
  --print                         Run non-interactively (AGY/Claude also accept -p)
  --effort <low|med|high>         Set model reasoning effort
  --plan                          Start session in planning mode
  Codex note: -c is --config, -p is --profile; use --continue / --print for Nexus aliases

Merged Help:
  %s <provider> --help            Show Nexus canonical aliases merged with official CLI help
`,
		p, p, p, p, p, p, p, p, p, p,
		p, p, p, p, p, p, p, p, p, p,
		p, p, p, p, p, p, p, p, p, p,
		p, p, p, p, p, p, p, p, p, p,
		p, p, p, p, p, p, p, p, p, p,
		p, p, p, p, p, p, p, p, p,
	)
	fmt.Print(localization.HumanizeHelp(body))
	fmt.Println(localization.T("help.language"))
}
