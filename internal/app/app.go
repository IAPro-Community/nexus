package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/buildinfo"
	"github.com/IAPro-Community/nexus/internal/control/flags"
	"github.com/IAPro-Community/nexus/internal/conversation"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/cooldown"
	"github.com/IAPro-Community/nexus/internal/core/exitcode"
	"github.com/IAPro-Community/nexus/internal/core/fallback"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/agy"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/claude"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/cursor"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/gemini"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/opencode"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/core/scheduler"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/core/session"
	"github.com/IAPro-Community/nexus/internal/core/telemetry"
	diagnostic "github.com/IAPro-Community/nexus/internal/doctor"
	"github.com/IAPro-Community/nexus/internal/localization"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/profile"
	"github.com/IAPro-Community/nexus/internal/release"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
	"github.com/IAPro-Community/nexus/internal/tui"
)

var globalRegistry *provider.Registry

func initRegistry() *provider.Registry {
	if globalRegistry != nil {
		return globalRegistry
	}
	reg := provider.NewRegistry()
	_ = reg.Register(codex.New())
	_ = reg.Register(agy.New())
	_ = reg.Register(claude.New())
	_ = reg.Register(opencode.New())
	_ = reg.Register(gemini.New())
	_ = reg.Register(cursor.New())
	globalRegistry = reg
	return reg
}

func progName() string {
	if len(os.Args) > 0 && os.Args[0] != "" {
		base := filepath.Base(os.Args[0])
		if base != "" && base != "." && base != "/" {
			return base
		}
	}
	return "nexus"
}

func Run(args []string) error {
	initRegistry()
	cfg, _ := config.LoadConfig()
	localization.Set(localization.Resolve("", cfg.Language))
	flagLanguage, remaining, err := localization.ExtractGlobalFlag(args)
	if err != nil {
		return err
	}
	args = remaining
	localization.Set(localization.Resolve(flagLanguage, cfg.Language))

	if len(args) == 0 {
		return interactive(nil)
	}

	if len(args) == 1 && args[0] == "--tui" {
		return interactiveTUI()
	}

	// Short form: nexus codex:work -- --model ... or nexus codex:auto
	if strings.Contains(args[0], ":") {
		parts := strings.SplitN(args[0], ":", 2)
		prov := parts[0]
		prof := parts[1]
		if isSupportedProvider(prov) {
			if prof == "auto" {
				prof = ""
			}
			return executeProviderWithSmartSelection(prov, prof, trimDashDash(args[1:]))
		}
	}

	switch args[0] {
	case "help", "-h", "--help":
		if len(args) > 1 && isSupportedProvider(args[1]) {
			return flags.ShowProviderHelp(args[1])
		}
		usage()
		return nil
	case "version", "--version", "-v":
		return versionCmd(args[1:])
	case "release":
		return release.Run(".")
	case "web", "open":
		return controlWebCmd(args[1:])
	case "plan", "plans":
		return planCmd(args[1:])
	case "agents", "agent":
		return agentsCmd(args[1:])
	case "projects", "project":
		return projectsCmd(args[1:])
	case "conductor":
		return conductorCmd(args[1:])
	case "start":
		return controlStartCmd(args[1:])
	case "stop":
		return controlStopCmd(args[1:])
	case "ps", "running":
		return controlRunningCmd(args[1:])
	case "attach":
		return controlAttachCmd(args[1:])
	case "handoff":
		return controlHandoffCmd(args[1:])
	case "continue":
		for _, a := range args[1:] {
			if a == "--with" || strings.HasPrefix(a, "--with=") {
				return controlContinueCmd(args[1:])
			}
		}
		return resumeCmd(args[1:])
	case "providers":
		return providersCmd(args[1:])
	case "profiles", "list", "ls":
		return profilesCmd(args[1:])
	case "paths":
		return paths()
	case "doctor":
		return doctorCmd(args[1:])
	case "security":
		return securityCmd(args[1:])
	case "add":
		return addCmd(args[1:])
	case "remove", "rm":
		return removeCmd(args[1:])
	case "rename":
		return renameCmd(args[1:])
	case "login":
		return loginCmd(args[1:])
	case "logout":
		return logoutCmd(args[1:])
	case "inspect":
		return inspectCmd(args[1:])
	case "completion":
		return completionCmd(args[1:])
	case "run":
		return runCmd(args[1:])
	case "resume":
		return resumeCmd(args[1:])
	case "switch", "swap", "use":
		return useCmd(args[1:])
	case "current":
		return currentCmd(args[1:])
	case "status":
		return statusCmd(args[1:])
	case "usage", "quota":
		return usageCmd(args[1:])
	case "sessions":
		return sessionsCmd(args[1:])
	case "workspaces":
		return workspacesCmd(args[1:])
	case "bind":
		return bindCmd(args[1:])
	case "unbind":
		return unbindCmd(args[1:])
	case "bindings":
		return bindingsCmd(args[1:])
	case "explain":
		return explainCmd(args[1:])
	case "history":
		return historyCmd(args[1:])
	case "stats":
		return statsCmd(args[1:])
	case "export":
		return exportCmd(args[1:])
	case "issue-report":
		return issueReportCmd(args[1:])
	case "update", "upgrade":
		return updateCmd(args[1:])
	case "maestro":
		return maestroCmd(args[1:])
	case "config":
		return configCmd(args[1:])
	case "control", "ui":
		return controlCmd(args[1:])
	case "tunnel":
		return tunnelCmd(args[1:])
	case "__control-host":
		return controlHostCmd(args[1:])
	case "codex", "agy", "claude", "opencode", "gemini", "cursor":
		prov := args[0]
		if flags.IsHelpFlag(args[1:]) {
			return flags.ShowProviderHelp(prov)
		}
		targetProfile := ""
		rest := args[1:]
		if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
			targetProfile = rest[0]
			rest = rest[1:]
		}
		if targetProfile == "auto" {
			targetProfile = ""
		}
		return executeProviderWithSmartSelection(prov, targetProfile, trimDashDash(rest))
	default:
		if strings.HasPrefix(args[0], "-") {
			// Auto selection for configured default provider
			cfg, _ := config.LoadConfig()
			for prov, prof := range cfg.Defaults {
				if prof != "" {
					return executeProviderWithSmartSelection(prov, prof, trimDashDash(args))
				}
			}
			return fmt.Errorf("no default profile configured; specify provider, e.g.: %s codex", progName())
		}
		return fmt.Errorf("unknown command %q; run '%s help'", args[0], progName())
	}
}

func isSupportedProvider(p string) bool {
	switch strings.ToLower(p) {
	case "codex", "agy", "claude", "opencode", "gemini", "cursor":
		return true
	default:
		return false
	}
}

func trimDashDash(args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		return args[1:]
	}
	return args
}

func interactive(args []string) error {
	for _, a := range args {
		if a == "--tui" {
			return interactiveTUI()
		}
	}
	// Default primary product experience: Launch Web Workspace OS (§Phase I)
	return controlWebCmd(args)
}

func interactiveTUI() error {
	return usageCmd(nil)
}

func executeProviderWithSmartSelection(provName, explicitProfile string, args []string) error {
	resolver := NewLaunchModeResolver(LaunchModeInput{
		Args:       args,
		StdinIsTTY: IsTerminal(),
		Provider:   provName,
	})
	result := resolver.ResolveWithReason()

	switch result.Mode {
	case LaunchModeSupervised:
		return executeProviderSupervised(provName, explicitProfile, resolver.StripControlFlags(), resolver.WantInteractiveLead(), resolver.RequireInteractiveLead())
	case LaunchModeDirect:
		args = resolver.StripControlFlags()
	}

	fmt.Fprintf(os.Stderr, "⚡ [nexus] iniciando %s…\n", provName)

	// Keep quota monitoring alive for the lifetime of direct provider commands,
	// including nexus agy/codex/cursor invocations.
	nexus.Default().StartQuotaMonitor(context.Background())
	reg := initRegistry()
	pAdapter, ok := reg.Get(provName)
	if !ok {
		return fmt.Errorf("unknown provider %q (exit code: %d)", provName, exitcode.ProviderNotFound)
	}

	cfg, _ := config.LoadConfig()
	args = flags.Normalize(provName, args, cfg.FlagAliases)
	qEng := quota.NewEngine(5 * time.Minute)
	cdTracker := cooldown.NewTracker()
	sel := scheduler.NewSelector(cfg, qEng, cdTracker)
	exec := fallback.NewExecutor(sel, cdTracker)

	allProfiles, err := profile.List()
	if err != nil {
		return err
	}

	var candidates []model.Profile
	accounts := make(map[string]model.AccountInfo)
	for _, p := range allProfiles {
		if p.Provider != provName || p.Disabled {
			continue
		}
		candidates = append(candidates, p)
		accounts[p.Name] = profile.GetAccountInfo(provName, p.Name)
		snapshot := profile.PeekUsageSnapshot(provName, p.Name)
		account := accounts[p.Name]
		account.Usage = snapshot
		accounts[p.Name] = account
		// Peek is launch-only evidence: never persist ESTIMATED last-known
		// into usage.json (that would poison Refresh/scheduler cache).
	}

	if len(candidates) == 0 {
		return fmt.Errorf("no profiles configured for provider %s. Run: %s add %s <name>", provName, progName(), provName)
	}

	candidates, err = launchableProfiles(provName, explicitProfile, candidates)
	if err != nil {
		return err
	}

	// Direct/scripted launches share the same bounded Codex quota freshness
	// rules as the default supervised TTY route. Explicit profiles skip probe.
	refreshCodexLaunchQuota(context.Background(), provName, explicitProfile, candidates, accounts)

	// Pre-check: warn if ALL accounts are unavailable before launching.
	allUnavailable := true
	for _, p := range candidates {
		acc := accounts[p.Name]
		var snap model.UsageSnapshot
		if accounts[p.Name].AccountScope.Verifiable() {
			snap, _ = qEng.GetCachedUsageForScope(accounts[p.Name].AccountScope)
		}
		qv := quota.BuildQuotaView(snap, acc.Email, acc.Plan)
		if qv.IsAvailable() {
			allUnavailable = false
			break
		}
	}
	if allUnavailable {
		fmt.Fprintf(os.Stderr, "\n⚠  NENHUMA QUOTA ATUAL CONFIRMADA para %s:\n\n", strings.ToUpper(provName))
		for _, p := range candidates {
			acc := accounts[p.Name]
			var snap model.UsageSnapshot
			if accounts[p.Name].AccountScope.Verifiable() {
				snap, _ = qEng.GetCachedUsageForScope(accounts[p.Name].AccountScope)
			}
			qv := quota.BuildQuotaView(snap, acc.Email, acc.Plan)
			reason := qv.AvailabilityLabel()
			detail := ""
			if len(qv.AvailReasons.ExhaustedWindows) > 0 {
				detail = fmt.Sprintf(" (janelas esgotadas: %s)", strings.Join(qv.AvailReasons.ExhaustedWindows, ", "))
			}
			fmt.Fprintf(os.Stderr, "   ✗ %s: %s%s\n", p.Name, reason, detail)
		}
		fmt.Fprintf(os.Stderr, "\n   Verifique a quota em 'nexus usage' (atualização em segundo plano).\n")
		fmt.Fprintf(os.Stderr, "   Com quota desconhecida, o Nexus pode usar somente um perfil já fixado ou padrão,\n")
		fmt.Fprintf(os.Stderr, "   sem afirmar que ele possui mais capacidade. Para escolha explícita: nexus %s:<perfil>\n\n", provName)
	}

	cwd, _ := os.Getwd()
	ctx := context.Background()

	return exec.RunWithFallback(ctx, provName, cwd, explicitProfile, candidates, accounts, true, func(p model.Profile) (model.Failure, error) {
		accInfo := accounts[p.Name]
		accEmail := accInfo.Email
		if accEmail == "" {
			accEmail = p.Name
		}
		var snap model.UsageSnapshot
		if accInfo.AccountScope.Verifiable() {
			snap, _ = qEng.GetCachedUsageForScope(accInfo.AccountScope)
		}
		var capInfo string
		qvCap := quota.BuildQuotaView(snap, accEmail, accInfo.Plan)
		if summary := qvCap.CompactGroupSummary(); summary != "" {
			capInfo = fmt.Sprintf(" [Cap: %s]", summary)
		}
		fmt.Fprintf(os.Stderr, "⚡ [%s] Usando perfil: %s:%s (%s)%s\n", progName(), provName, p.Name, accEmail, capInfo)

		start := time.Now()
		_ = telemetry.LogEvent(telemetry.Event{
			Type:         telemetry.EventSessionStarted,
			ProviderID:   provName,
			ProfileID:    p.Name,
			Workspace:    cwd,
			AccountScope: accInfo.AccountScope,
		})

		fail, runErr := pAdapter.Run(ctx, p, args)
		dur := time.Since(start)

		if errors.Is(runErr, codex.ErrTUIBusy) {
			busyErr := fmt.Errorf("%s profile %s is temporarily unavailable for maintenance; retry shortly", provName, p.Name)
			return model.Failure{Kind: model.FailureCommand, Message: busyErr.Error()}, busyErr
		}

		if fail.Kind != model.FailureNone && fail.Kind != "" {
			_ = telemetry.LogEvent(telemetry.Event{
				Type:         telemetry.EventRateLimitDetected,
				ProviderID:   provName,
				ProfileID:    p.Name,
				Workspace:    cwd,
				DurationMs:   dur.Milliseconds(),
				FailureKind:  fail.Kind,
				AccountScope: accInfo.AccountScope,
			})
		}
		return fail, runErr
	})
}

func executeResume(provName, profName, sessionID string, args []string) error {
	reg := initRegistry()
	pAdapter, ok := reg.Get(provName)
	if !ok {
		return fmt.Errorf("provider %q not found", provName)
	}

	if profName == "" {
		cfg, _ := config.LoadConfig()
		profName = cfg.Defaults[provName]
		if profName == "" {
			ps, _ := profile.List()
			for _, p := range ps {
				if p.Provider == provName {
					profName = p.Name
					break
				}
			}
		}
	}

	p := model.Profile{Provider: provName, Name: profName}
	if convProv, ok := pAdapter.(provider.ConversationProvider); ok {
		fail, err := convProv.Resume(context.Background(), p, sessionID, args)
		if err != nil {
			return err
		}
		if fail.Kind != model.FailureNone && fail.Kind != "" {
			return fmt.Errorf("%s", fail.Message)
		}
		return nil
	}

	return fmt.Errorf("provider %s does not support session resume (exit code: %d)", provName, exitcode.ResumeUnsupported)
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
  %s <provider>:auto [flags]      Explicit auto-selection
  %s resume [id] [provider:name]  Resume previous session using provider-native syntax

  %s providers [--json]           List installed providers, versions & capabilities
  %s providers status [--json]    Show installation and registration lifecycle
  %s providers register <provider> [--profile <name>] [--no-login]
  %s profiles [--json]            List configured profiles, auth status & priorities
  %s paths                        Show Nexus data and configuration paths
  %s add <provider> [name]        Add a new provider authentication profile
  %s remove <provider> <name>     Remove an existing profile
  %s rename <provider> <old> <new> Rename a configured provider profile
  %s login <provider> [name]      Run provider official login flow; defaults to the provider profile
  %s logout <provider> <name>     Run provider official logout flow
  %s use <provider> <name>        Set default active profile for provider
  %s status [provider[:profile]]  Display profile health, plan and account status
  %s usage [--json] [--refresh]   Searchable quota table; --json for integrations; --refresh for live CLI fetch
  %s inspect <provider> <name>    Inspect profile configuration details

  %s projects [list|add]          Register and list project workspaces
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
  %s issue-report                 Create a diagnostic issue report
  %s config <show|validate>       Manage control plane settings
  %s update                       Update Nexus only (use 'maestro update' explicitly for Maestro)
  %s maestro <status|doctor|update> Manage optional Maestro integration explicitly
  nexus maestro memory-recovery-status [--json]
                                  Read-only check for incomplete restoration journals
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
		p, p, p, p, p, p, p, p, p, p, p, p, p,
		p, p, p, p, p, p, p,
		p, p, p, p, p, p, p, p, p, p, p,
		p, p, p,
		p, p, p, p, p, p, p, p, p, p, p, p, p, p, p, p, p, p, p, p, p,
		p, p,
	)
	fmt.Print(localization.HumanizeHelp(body))
	fmt.Println(localization.T("help.language"))
}

func versionCmd(args []string) error {
	if len(args) > 0 && args[0] == "--json" {
		out := buildinfo.JSON()
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	fmt.Println(buildinfo.String())
	return nil
}

func providersCmd(args []string) error {
	if len(args) > 0 && (args[0] == "status" || args[0] == "register") {
		if args[0] == "status" {
			return providerStatusCmd(args[1:])
		}
		return providerRegisterCmd(args[1:])
	}
	reg := initRegistry()
	ctx := context.Background()
	detections := reg.DetectAll(ctx)
	profiles, _ := profile.List()

	profCount := make(map[string]int)
	for _, p := range profiles {
		profCount[p.Provider]++
	}

	if len(args) > 0 && args[0] == "--json" {
		out := make(map[string]interface{})
		for _, p := range reg.List() {
			id := string(p.ID())
			det := detections[id]
			out[id] = map[string]interface{}{
				"name":         p.Name(),
				"installed":    det.Installed,
				"version":      det.Version,
				"binary_path":  det.BinaryPath,
				"profiles":     profCount[id],
				"capabilities": p.Capabilities(),
			}
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("%-14s %-12s %-24s %-10s %s\n", "PROVIDER", "INSTALLED", "VERSION", "PROFILES", "CAPABILITIES")
	for _, p := range reg.List() {
		id := string(p.ID())
		det := detections[id]
		instStr := "no"
		if det.Installed {
			instStr = "yes"
		}
		verStr := det.Version
		if verStr == "" {
			verStr = "—"
		}
		if len(verStr) > 22 {
			verStr = verStr[:20] + ".."
		}
		caps := p.Capabilities()
		capSummary := fmt.Sprintf("usage:%v resume:%v isolate:%v", caps.Usage, caps.Resume, caps.IsolatedRuntime)
		fmt.Printf("%-14s %-12s %-24s %-10d %s\n", p.Name(), instStr, verStr, profCount[id], capSummary)
	}
	return nil
}

func providerStatusCmd(args []string) error {
	jsonOutput := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
		}
	}
	reg := initRegistry()
	installations := provider.NewInstallationRegistry()
	records, err := installations.Refresh(reg, context.Background())
	if err != nil {
		return err
	}
	profiles, err := profile.List()
	if err != nil {
		return err
	}
	for i := range records {
		hasProfile := false
		hasAuthenticated := false
		for _, p := range profiles {
			if p.Provider != records[i].Provider {
				continue
			}
			hasProfile = true
			info := profile.GetAccountInfo(p.Provider, p.Name)
			if info.Authenticated {
				hasAuthenticated = true
			}
		}
		switch {
		case hasAuthenticated:
			records[i].State = provider.RegisteredAuthenticated
		case hasProfile:
			records[i].State = provider.PendingAuth
		case !records[i].Installed:
			// Preserve the installation record while making the current
			// availability explicit in JSON/CLI output.
			records[i].State = provider.InstalledUnregistered
		default:
			records[i].State = provider.InstalledUnregistered
		}
	}
	if jsonOutput {
		b, _ := json.MarshalIndent(records, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	fmt.Printf("%-12s %-24s %-18s %s\n", "PROVIDER", "BINARY", "STATE", "VERSION")
	for _, record := range records {
		fmt.Printf("%-12s %-24s %-18s %s\n", record.Provider, record.Binary, record.State, record.Version)
	}
	return nil
}

func providerRegisterCmd(args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: %s providers register <provider> [--profile <name>] [--no-login]", progName())
	}
	providerID := strings.ToLower(args[0])
	profileName := providerID
	noLogin := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--profile":
			if i+1 >= len(args) {
				return fmt.Errorf("--profile requires a name")
			}
			profileName = args[i+1]
			i++
		case "--no-login":
			noLogin = true
		default:
			return fmt.Errorf("unknown option %q", args[i])
		}
	}
	reg := initRegistry()
	if _, ok := reg.Get(providerID); !ok {
		return fmt.Errorf("unknown provider %s", providerID)
	}
	if profile.Exists(providerID, profileName) {
		return fmt.Errorf("profile %s:%s already exists", providerID, profileName)
	}
	if _, err := profile.Create(providerID, profileName); err != nil {
		return err
	}
	fmt.Printf("Registered %s:%s with an isolated profile.\n", providerID, profileName)
	if noLogin {
		fmt.Println("Authentication deferred; state is PENDING_AUTH.")
		return nil
	}
	if err := loginCmd([]string{providerID, profileName}); err != nil {
		fmt.Printf("Authentication pending for %s:%s: %v\n", providerID, profileName, err)
		return err
	}
	return nil
}

func profilesCmd(args []string) error {
	ps, err := profile.List()
	if err != nil {
		return err
	}
	cfg, _ := config.LoadConfig()

	if len(args) > 0 && args[0] == "--json" {
		type profileJSON struct {
			model.Profile
			Account model.AccountInfo `json:"account"`
			Default bool              `json:"is_default"`
		}
		var list []profileJSON
		for _, p := range ps {
			acc := profile.GetAccountInfo(p.Provider, p.Name)
			list = append(list, profileJSON{
				Profile: p,
				Account: acc,
				Default: cfg.Defaults[p.Provider] == p.Name,
			})
		}
		b, _ := json.MarshalIndent(list, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(ps) == 0 {
		fmt.Printf("No profiles configured. Example: %s add codex work\n", progName())
		return nil
	}

	fmt.Printf("%-10s %-18s %-28s %-16s %-12s %s\n", "PROVIDER", "PROFILE", "ACCOUNT / EMAIL", "PLAN", "STATUS", "DEFAULT")
	for _, p := range ps {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		star := ""
		if cfg.Defaults[p.Provider] == p.Name {
			star = "★ (default)"
		}
		email := acc.Email
		if email == "" {
			if acc.Authenticated {
				email = "—"
			} else {
				email = "(unauthenticated)"
			}
		}
		if len(email) > 26 {
			email = email[:24] + ".."
		}
		fmt.Printf("%-10s %-18s %-28s %-16s %-12s %s\n", p.Provider, p.Name, email, acc.Plan, acc.Status, star)
	}
	return nil
}

func addCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: %s add <provider> [name] [--no-login]", progName())
	}
	providerName := args[0]
	if err := profile.ValidateProvider(providerName); err != nil {
		return err
	}

	noLogin := false
	name := ""
	for _, a := range args[1:] {
		if a == "--no-login" {
			noLogin = true
		} else if name == "" {
			name = a
		} else {
			return fmt.Errorf("unexpected argument %q", a)
		}
	}
	if name == "" {
		fmt.Printf("Profile name for %s: ", providerName)
		s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		name = strings.TrimSpace(s)
	}

	p, err := profile.Create(providerName, name)
	if err != nil {
		return err
	}

	reg := initRegistry()
	pAdapter, ok := reg.Get(providerName)
	if ok {
		if err := pAdapter.Prepare(context.Background(), p); err != nil {
			_ = profile.Delete(providerName, name)
			return err
		}
	}

	fmt.Printf("✓ Created %s profile %q.\n", p.Provider, p.Name)
	if d, _ := config.GetDefaultProfile(providerName); d == "" {
		_ = config.SetDefaultProfile(providerName, name)
	}

	if noLogin {
		return nil
	}
	if providerName == "opencode" {
		fmt.Println("OpenCode does not have a Nexus-managed login. Authenticate its chosen provider in the same runtime, for example:")
		fmt.Println("  opencode auth login <provider>")
		return nil
	}
	fmt.Printf("Starting official authentication flow for %s:%s...\n", providerName, name)
	if authProv, ok := pAdapter.(provider.AuthProvider); ok {
		return authProv.Login(context.Background(), p)
	}
	return nil
}

func removeCmd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: %s remove <provider> <name> [--yes]", progName())
	}
	prov, name := args[0], args[1]
	if !profile.Exists(prov, name) {
		return fmt.Errorf("profile %s:%s does not exist", prov, name)
	}
	yes := len(args) >= 3 && args[2] == "--yes"
	if !yes {
		fmt.Printf("Delete profile %s:%s and its isolated local storage? [y/N] ", prov, name)
		s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "y" && s != "yes" {
			fmt.Println("Canceled.")
			return nil
		}
	}
	return profile.Delete(prov, name)
}

func renameCmd(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: %s rename <provider> <old_name> <new_name>", progName())
	}
	return profile.Rename(args[0], args[1], args[2])
}

func resolveLoginTarget(args []string) (string, string, error) {
	if len(args) == 1 {
		return args[0], args[0], nil
	}
	if len(args) == 2 {
		return args[0], args[1], nil
	}
	return "", "", fmt.Errorf("usage: %s login <provider> [name]", progName())
}

func loginCmd(args []string) error {
	prov, name, err := resolveProfileArgs(args, "login")
	if err != nil {
		return err
	}
	if !profile.Exists(prov, name) {
		return fmt.Errorf("profile %s:%s does not exist", prov, name)
	}
	reg := initRegistry()
	pAdapter, ok := reg.Get(prov)
	if !ok {
		return fmt.Errorf("unknown provider %s", prov)
	}
	p := model.Profile{Provider: prov, Name: name}
	if authProv, ok := pAdapter.(provider.AuthProvider); ok {
		return authProv.Login(context.Background(), p)
	}
	return fmt.Errorf("provider %s does not support login", prov)
}

// resolveProfileArgs accepts "<provider> <name>" and, when the provider has a
// single unambiguous profile, the shorter "<provider>" form. The short form
// keeps the copyable command from the UI from repeating the provider name
// (`nexus login codex` instead of `nexus login codex codex`).
func resolveProfileArgs(args []string, verb string) (string, string, error) {
	if len(args) == 2 {
		return args[0], args[1], nil
	}
	if len(args) != 1 {
		return "", "", fmt.Errorf("usage: %s %s <provider> [<name>]", progName(), verb)
	}
	prov := strings.TrimSpace(args[0])
	if prov == "" {
		return "", "", fmt.Errorf("usage: %s %s <provider> [<name>]", progName(), verb)
	}
	if profile.Exists(prov, prov) {
		return prov, prov, nil
	}
	all, err := profile.List()
	if err != nil {
		return "", "", err
	}
	var matches []string
	for _, p := range all {
		if p.Provider == prov && !p.Disabled {
			matches = append(matches, p.Name)
		}
	}
	switch len(matches) {
	case 1:
		return prov, matches[0], nil
	case 0:
		return "", "", fmt.Errorf("no profile found for provider %s; run %s providers register %s", prov, progName(), prov)
	default:
		sort.Strings(matches)
		return "", "", fmt.Errorf("provider %s has multiple profiles (%s); specify one: %s %s %s <name>", prov, strings.Join(matches, ", "), progName(), verb, prov)
	}
}

func logoutCmd(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: %s logout <provider> <name>", progName())
	}
	prov, name := args[0], args[1]
	if !profile.Exists(prov, name) {
		return fmt.Errorf("profile %s:%s does not exist", prov, name)
	}
	reg := initRegistry()
	pAdapter, ok := reg.Get(prov)
	if !ok {
		return fmt.Errorf("unknown provider %s", prov)
	}
	p := model.Profile{Provider: prov, Name: name}
	if authProv, ok := pAdapter.(provider.AuthProvider); ok {
		return authProv.Logout(context.Background(), p)
	}
	return nil
}

func useCmd(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: %s use <provider> <name>", progName())
	}
	if err := config.SetDefaultProfile(args[0], args[1]); err != nil {
		return err
	}
	fmt.Printf("Default %s profile: %s\n", args[0], args[1])
	return nil
}

func currentCmd(args []string) error {
	providers := []string{"codex", "agy", "claude", "opencode", "gemini", "cursor"}
	if len(args) >= 1 && args[0] != "--json" {
		providers = []string{args[0]}
	}
	isJSON := len(args) >= 1 && (args[0] == "--json" || (len(args) >= 2 && args[1] == "--json"))

	cwd, _ := os.Getwd()
	cfg, _ := config.LoadConfig()

	if isJSON {
		out := make(map[string]map[string]string)
		for _, p := range providers {
			bound := config.GetBinding(cwd, p)
			def := cfg.Defaults[p]
			out[p] = map[string]string{
				"bound":   bound,
				"default": def,
			}
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Workspace: %s\n\n", cwd)
	for _, p := range providers {
		bound := config.GetBinding(cwd, p)
		def := cfg.Defaults[p]
		if def == "" {
			def = "(none)"
		}
		boundStr := ""
		if bound != "" {
			boundStr = fmt.Sprintf(" [bound: %s]", bound)
		}
		fmt.Printf("%-10s %s%s\n", p, def, boundStr)
	}
	return nil
}

func statusCmd(args []string) error {
	if len(args) == 0 {
		return profilesCmd(nil)
	}
	prov := args[0]
	name := ""
	if strings.Contains(prov, ":") {
		parts := strings.SplitN(prov, ":", 2)
		prov = parts[0]
		name = parts[1]
	} else if len(args) >= 2 {
		name = args[1]
	}

	if name == "" {
		return profilesCmd(nil)
	}

	acc := profile.GetAccountInfo(prov, name)
	fmt.Printf("=== %s:%s Status ===\n", strings.ToUpper(prov), name)
	fmt.Printf("Email:         %s\n", acc.Email)
	fmt.Printf("Plan:          %s\n", acc.Plan)
	fmt.Printf("Status:        %s\n", acc.Status)
	fmt.Printf("Health:        %s\n", acc.Health)
	if !acc.ExpiresAt.IsZero() {
		fmt.Printf("Expires At:    %s\n", acc.ExpiresAt.Format(time.RFC3339))
	}
	return nil
}

// usageJSONRow is the machine-readable row of `nexus usage --json`. It keeps
// the raw snapshot plus the derived quota view and eligibility so web, TUI and
// CLI surfaces can agree on the same fixture.
type usageJSONRow struct {
	model.UsageSnapshot
	QuotaView     quota.QuotaView      `json:"quota_view"`
	Eligibility   quota.Eligibility    `json:"eligibility"`
	Health        model.ProviderHealth `json:"health"`
	Authenticated bool                 `json:"authenticated"`
	// Unconfigured marks a provider that is installed but has no profile yet;
	// it stays visible on purpose so tooling never hides an installed CLI.
	Unconfigured bool `json:"unconfigured,omitempty"`
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
	if refresh {
		fmt.Fprintln(os.Stderr, "Atualizando quotas ao vivo…")
		for _, p := range ps {
			fmt.Fprintf(os.Stderr, "  • %s:%s\n", p.Provider, p.Name)
			_ = profile.RefreshUsageSnapshot(p.Provider, p.Name)
		}
	}

	// Detect installed CLIs across the system so both the TUI and the
	// machine-readable surface can show configured profiles plus installed
	// tools that are awaiting configuration (e.g. Claude Code, Gemini CLI,
	// Cursor CLI, OpenCode).
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
		list := make([]usageJSONRow, 0, len(ps))
		for _, p := range ps {
			snap := profile.GetUsageSnapshot(p.Provider, p.Name)
			acc := profile.GetAccountInfo(p.Provider, p.Name)
			qv := quota.BuildQuotaView(snap, acc.Email, acc.Plan)
			elig := quota.EvaluateEligibility(quota.EligibilityInput{
				View:          qv,
				Authenticated: acc.Authenticated,
				Health:        string(acc.Health),
			})
			list = append(list, usageJSONRow{
				UsageSnapshot: snap,
				QuotaView:     qv,
				Eligibility:   elig,
				Health:        acc.Health,
				Authenticated: acc.Authenticated,
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

func sessionsCmd(args []string) error {
	cwd, _ := os.Getwd()
	convs := conversation.ListRecent(50, cwd)

	if len(args) > 0 && args[0] == "search" && len(args) >= 2 {
		q := args[1]
		var filtered []conversation.Conversation
		for _, c := range convs {
			if strings.Contains(strings.ToLower(c.Title), strings.ToLower(q)) ||
				strings.Contains(strings.ToLower(c.ID), strings.ToLower(q)) ||
				strings.Contains(strings.ToLower(c.Workspace), strings.ToLower(q)) {
				filtered = append(filtered, c)
			}
		}
		convs = filtered
	}

	if len(args) > 0 && args[len(args)-1] == "--json" {
		b, _ := json.MarshalIndent(convs, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(convs) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}
	rows := make([]tui.DataTableRow, 0, len(convs))
	for _, c := range convs {
		values := []string{c.Provider, c.ID, c.Title, c.Workspace}
		rows = append(rows, tui.DataTableRow{Values: values, SearchText: strings.Join(values, " ")})
	}
	return tui.RunDataTable(tui.DataTableOptions{Title: "Nexus · Sessões recentes", FilterPlaceholder: "filtrar por provedor, sessão, título ou workspace", Columns: []tui.DataTableColumn{{Title: "PROVEDOR", Width: 12}, {Title: "SESSÃO", Width: 28}, {Title: "TÍTULO", Width: 30}, {Title: "WORKSPACE", Width: 38}}, Rows: rows})
}

func workspacesCmd(args []string) error {
	cwd, _ := os.Getwd()
	convs := conversation.ListRecent(100, cwd)
	cfg, _ := config.LoadConfig()

	var sessions []model.Session
	for _, c := range convs {
		sessions = append(sessions, model.Session{
			ProviderID: c.Provider,
			ID:         c.ID,
			Title:      c.Title,
			Workspace:  c.Workspace,
			UpdatedAt:  c.LastModified,
		})
	}

	store := session.NewStore()
	wsList := store.GroupByWorkspace(sessions, cfg)

	if len(args) > 0 && args[0] == "--json" {
		b, _ := json.MarshalIndent(wsList, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Active Workspaces (%d discovered):\n\n", len(wsList))
	for _, ws := range wsList {
		fmt.Printf("📁 %s\n", ws.Path)
		if len(ws.Bindings) > 0 {
			fmt.Printf("   Bindings: %+v\n", ws.Bindings)
		}
		fmt.Printf("   Sessions: %d\n\n", len(ws.Sessions))
	}
	return nil
}

func bindCmd(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: %s bind <provider>:<profile>", progName())
	}
	parts := strings.SplitN(args[0], ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("usage: %s bind <provider>:<profile>", progName())
	}
	prov, prof := parts[0], parts[1]
	cwd, _ := os.Getwd()
	if err := config.BindWorkspace(cwd, prov, prof); err != nil {
		return err
	}
	fmt.Printf("✓ Bound workspace %s to %s:%s\n", cwd, prov, prof)
	return nil
}

func unbindCmd(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: %s unbind <provider>", progName())
	}
	cwd, _ := os.Getwd()
	if err := config.UnbindWorkspace(cwd, args[0]); err != nil {
		return err
	}
	fmt.Printf("✓ Unbound provider %s from workspace %s\n", args[0], cwd)
	return nil
}

func bindingsCmd(args []string) error {
	cfg, _ := config.LoadConfig()
	if len(args) > 0 && args[0] == "--json" {
		b, _ := json.MarshalIndent(cfg.Bindings, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	if len(cfg.Bindings) == 0 {
		fmt.Println("No workspace bindings configured.")
		return nil
	}
	for ws, b := range cfg.Bindings {
		fmt.Printf("%s:\n", ws)
		for p, prof := range b {
			fmt.Printf("  • %-10s -> %s\n", p, prof)
		}
	}
	return nil
}

func explainCmd(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: %s explain <provider>", progName())
	}
	prov := args[0]
	cfg, _ := config.LoadConfig()
	qEng := quota.NewEngine(5 * time.Minute)
	cdTracker := cooldown.NewTracker()
	sel := scheduler.NewSelector(cfg, qEng, cdTracker)

	ps, _ := profile.List()
	var candidates []model.Profile
	accounts := make(map[string]model.AccountInfo)
	for _, p := range ps {
		if p.Provider == prov {
			candidates = append(candidates, p)
			accounts[p.Name] = profile.GetAccountInfo(prov, p.Name)
			snap := profile.GetUsageSnapshot(prov, p.Name)
			account := accounts[p.Name]
			account.Usage = snap
			accounts[p.Name] = account
		}
	}

	cwd, _ := os.Getwd()
	fmt.Print(sel.ExplainSelection(context.Background(), prov, cwd, candidates, accounts))
	return nil
}

func doctorCmd(args []string) error {
	if len(args) > 0 && args[0] == "terminal" {
		return terminalDoctorCmd(args[1:])
	}
	reg := initRegistry()
	ctx := context.Background()
	detections := reg.DetectAll(ctx)
	jsonOutput := false
	bundleOutput := false
	for _, arg := range args {
		jsonOutput = jsonOutput || arg == "--json"
		bundleOutput = bundleOutput || arg == "--bundle"
	}
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

func exportCmd(_ []string) error {
	reg := initRegistry()
	ctx := context.Background()
	detections := reg.DetectAll(ctx)
	ps, _ := profile.List()
	cfg, _ := config.LoadConfig()

	exportData := map[string]interface{}{
		"version":    buildinfo.Version,
		"os":         runtime.GOOS,
		"detections": detections,
		"profiles":   ps,
		"config":     cfg,
	}

	b, _ := json.MarshalIndent(exportData, "", "  ")
	sanitized := security.Redact(string(b))
	fmt.Println(sanitized)
	return nil
}

func issueReportCmd(_ []string) error {
	reg := initRegistry()
	ctx := context.Background()
	detections := reg.DetectAll(ctx)

	var sb strings.Builder
	sb.WriteString("### Environment Diagnostics\n\n")
	fmt.Fprintf(&sb, "- IAPro Nexus Version: `%s`\n", buildinfo.Version)
	fmt.Fprintf(&sb, "- OS: `%s`\n\n", runtime.GOOS)
	sb.WriteString("### Installed Providers\n\n")
	for id, det := range detections {
		fmt.Fprintf(&sb, "- **%s**: Installed=%v, Version=`%s`\n", id, det.Installed, det.Version)
	}
	fmt.Println(security.Redact(sb.String()))
	return nil
}

func configCmd(args []string) error {
	if len(args) == 0 || args[0] == "show" {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	if args[0] == "validate" {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		issues := cfg.Validate()
		if len(issues) == 0 {
			fmt.Println("✓ " + localization.T("config.valid"))
			return nil
		}
		fmt.Println(localization.T("config.issues"))
		for _, is := range issues {
			fmt.Printf("  ⚠ %s\n", is)
		}
		return nil
	}
	if args[0] == "language" {
		if len(args) != 2 || !localization.IsSupported(args[1]) {
			return errors.New(localization.T("error.config_usage"))
		}
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		value := args[1]
		if value != "auto" {
			value = localization.Normalize(value)
		}
		cfg.Language = value
		if err := config.SaveConfig(cfg); err != nil {
			return err
		}
		localization.Set(localization.Resolve("", value))
		fmt.Println(localization.T("config.language_saved", map[string]any{"Language": value}))
		return nil
	}
	return errors.New(localization.T("error.config_usage"))
}

func inspectCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s inspect <provider> <name>", progName())
	}
	prov := args[0]
	name := ""
	if strings.Contains(prov, ":") {
		parts := strings.SplitN(prov, ":", 2)
		prov = parts[0]
		name = parts[1]
	} else if len(args) >= 2 {
		name = args[1]
	}
	info, err := profile.Inspect(prov, name)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(info, "", "  ")
	fmt.Println(string(b))
	return nil
}

func runCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: %s run <provider> [profile] [--] [args...]", progName())
	}
	// Preserve the historical provider form while making the canonical Nexus
	// task form executable: nexus run "corrija a UI React". The task form
	// enters the same Flow -> AgentMatcher -> ResourceScheduler -> runtime
	// pipeline used by the control plane.
	if !isRunProviderInvocation(args) {
		if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
			fmt.Printf("usage: %s run <goal>\n       %s run <provider> [profile] [--] [args...]\n", progName(), progName())
			return nil
		}
		return runGoalCmd(args)
	}
	prov := args[0]
	prof := ""
	rest := args[1:]
	if strings.Contains(prov, ":") {
		parts := strings.SplitN(prov, ":", 2)
		prov = parts[0]
		prof = parts[1]
	} else if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		prof = rest[0]
		rest = rest[1:]
	}
	return executeProviderWithSmartSelection(prov, prof, trimDashDash(rest))
}

func isRunProviderInvocation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	if isSupportedProvider(args[0]) {
		return true
	}
	providerID, profileName, hasProfile := strings.Cut(args[0], ":")
	return hasProfile && isSupportedProvider(providerID) && profileName != "" &&
		strings.TrimSpace(profileName) == profileName && !strings.ContainsAny(profileName, " \t\r\n")
}

func runGoalCmd(args []string) error {
	goal := strings.TrimSpace(strings.Join(args, " "))
	if goal == "" {
		return fmt.Errorf("informe o objetivo da tarefa")
	}

	ctx := context.Background()
	n := nexus.Default()
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	projects, err := st.ListProjects()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		return fmt.Errorf("nenhum projeto ativo; registre um projeto primeiro")
	}
	project := projects[0]
	intentDecision, decisionErr := n.DecideIntentForProject(ctx, project.ID, goal)
	if decisionErr != nil {
		return fmt.Errorf("decidir intenção: %w", decisionErr)
	}
	if intentDecision.Strategy == nexus.IntentClarify {
		return fmt.Errorf("BLOCKED_NEEDS_USER: %s", strings.Join(intentDecision.BlockingQuestions, "; "))
	}
	proposal, err := n.DecomposePromptIntoFlowProposal(ctx, nexus.FlowDecompositionRequest{
		ProjectID:      project.ID,
		Goal:           goal,
		SourcePrompt:   goal,
		Simple:         intentDecision.Strategy == nexus.IntentDirect,
		DelegationMode: nexus.DelegationAuto,
	})
	if err != nil {
		return fmt.Errorf("classificar e decompor tarefa: %w", err)
	}
	flowPlan := nexus.WorkPlanFromFlow(proposal.Flow)
	structuredFacts, err := nexus.PersistIntentDecisionFacts(flowPlan.StructuredFacts, intentDecision)
	if err != nil {
		return fmt.Errorf("persistir decisão de intenção: %w", err)
	}
	plan, err := n.CreateWorkPlan(ctx, project.ID, proposal.Title, proposal.Description, flowPlan.Phases, structuredFacts)
	if err != nil {
		return fmt.Errorf("persistir plano da tarefa: %w", err)
	}
	run, err := n.StartMissionRun(ctx, plan.ID, "", runner.DefaultAutonomyContract(), true)
	if err != nil {
		return fmt.Errorf("iniciar tarefa: %w", err)
	}
	requirements := nexus.ClassifyTaskRequirements(goal)
	return waitAndPrintMission(ctx, n, run, map[string]any{
		"autonomous":             true,
		"plan_id":                plan.ID,
		"project_id":             project.ID,
		"intent_decision":        intentDecision,
		"task_requirements":      requirements,
		"requires_decomposition": requirements.RequiresDecomposition,
	})
}

func resumeCmd(args []string) error {
	if len(args) == 0 {
		cwd, _ := os.Getwd()
		convs := conversation.ListRecent(10, cwd)
		if len(convs) == 0 {
			fmt.Println("No recent sessions found.")
			return nil
		}
		fmt.Println("Recent Sessions:")
		for i, c := range convs {
			fmt.Printf("  %d) [%s] %-36s (%s)\n", i+1, strings.ToUpper(c.Provider), c.Title, c.ID[:8])
		}
		fmt.Print("Select session [1-9] (q to cancel): ")
		var input string
		fmt.Scanln(&input)
		input = strings.TrimSpace(input)
		if input == "q" || input == "Q" || input == "" {
			return nil
		}
		var idx int
		if _, err := fmt.Sscanf(input, "%d", &idx); err != nil || idx < 1 || idx > len(convs) {
			return errors.New("invalid selection")
		}
		chosen := convs[idx-1]
		return executeResume(chosen.Provider, "", chosen.ID, nil)
	}

	sessionID := args[0]
	prov := "codex"
	prof := ""
	if len(args) >= 2 {
		target := args[1]
		if strings.Contains(target, ":") {
			parts := strings.SplitN(target, ":", 2)
			prov = parts[0]
			prof = parts[1]
		} else {
			prof = target
		}
	}
	return executeResume(prov, prof, sessionID, nil)
}

func paths() error {
	d, err := config.DataDir()
	if err != nil {
		return err
	}
	c, err := config.ConfigDir()
	if err != nil {
		return err
	}
	s, err := config.StateDir()
	if err != nil {
		return err
	}
	fmt.Println("data:  ", d)
	fmt.Println("config:", c)
	fmt.Println("state: ", s)
	return nil
}

func completionCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s completion <bash|zsh|fish|powershell>", progName())
	}
	switch args[0] {
	case "bash":
		fmt.Print(`_nexus_completion() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="web start stop ps running attach handoff continue resume tunnel control ui providers profiles add remove login logout use status usage inspect sessions workspaces bind unbind bindings explain doctor security history stats config update maestro completion version release codex agy claude opencode gemini cursor --supervised --direct --lead --yolo --continue --resume"
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}
complete -F _nexus_completion nexus
complete -F _nexus_completion ai
`)
	case "zsh":
		fmt.Print(`#compdef nexus ai
_nexus() {
    local -a commands
    commands=(
        'web:Launch IAPro Nexus Workspace OS (Web UI)'
        'tunnel:Remote access via Cloudflare Quick Tunnel'
        'start:Start supervised agent runtime'
        'stop:Stop running supervised runtime'
        'ps:List active supervised runtimes'
        'running:List active supervised runtimes'
        'attach:Attach terminal to running runtime'
        'handoff:Same-provider account handoff'
        'continue:Cross-provider context handoff'
        'control:Nexus Supervised Agent Runtimes'
        'ui:Interactive Control Center'
        'providers:List AI providers and capabilities'
        'profiles:List configured profiles'
        'add:Add a provider authentication profile'
        'remove:Remove a profile'
        'login:Run provider official login flow'
        'logout:Run provider official logout flow'
        'use:Set default active profile for provider'
        'status:Display profile health, plan and account status'
        'usage:Display quota and rate limits'
        'sessions:Universal session index'
        'doctor:Diagnostics and health checks'
        'security:Security and isolation audit'
        'update:Update Nexus only'
        'maestro:Manage optional Maestro integration explicitly'
    )
    _describe 'command' commands
}
_nexus "$@"
`)
	case "fish":
		fmt.Print(`complete -c nexus -f -a "web start stop ps running attach handoff continue resume tunnel control ui providers profiles add remove login logout use status usage inspect sessions workspaces bind unbind bindings explain doctor security history stats config update maestro completion version release codex agy claude opencode gemini cursor --supervised --direct --lead --yolo --continue --resume"
complete -c ai -f -a "web start stop ps running attach handoff continue resume tunnel control ui providers profiles add remove login logout use status usage inspect sessions workspaces bind unbind bindings explain doctor security history stats config update maestro completion version release codex agy claude opencode gemini cursor --supervised --direct --lead --yolo --continue --resume"
`)
	case "powershell", "pwsh":
		fmt.Print(`Register-ArgumentCompleter -Native -CommandName @('nexus', 'ai') -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $commands = @(
        'web', 'start', 'stop', 'ps', 'running', 'attach', 'handoff', 'continue', 'resume',
        'tunnel', 'control', 'ui', 'providers', 'profiles', 'add', 'remove', 'login', 'logout', 'use',
        'status', 'usage', 'inspect', 'sessions', 'workspaces', 'bind', 'unbind', 'bindings',
        'explain', 'doctor', 'security', 'history', 'stats', 'config', 'update', 'completion', 'version', 'release',
        'codex', 'agy', 'claude', 'opencode', 'gemini', 'cursor', '--supervised', '--direct', '--lead', '--yolo', '--continue', '--resume'
    )
    $commands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
    }
}
`)
	default:
		return fmt.Errorf("unsupported shell %q. Supported: bash, zsh, fish, powershell", args[0])
	}
	return nil
}
