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
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/buildinfo"
	"github.com/kivervinicius/ai-cli/internal/control/flags"
	"github.com/kivervinicius/ai-cli/internal/conversation"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/cooldown"
	"github.com/kivervinicius/ai-cli/internal/core/exitcode"
	"github.com/kivervinicius/ai-cli/internal/core/fallback"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/provider"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/agy"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/claude"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/codex"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/cursor"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/gemini"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/opencode"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/core/scheduler"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/core/session"
	"github.com/kivervinicius/ai-cli/internal/core/telemetry"
	"github.com/kivervinicius/ai-cli/internal/localization"
	"github.com/kivervinicius/ai-cli/internal/nexus"
	"github.com/kivervinicius/ai-cli/internal/profile"
	"github.com/kivervinicius/ai-cli/internal/release"
	"github.com/kivervinicius/ai-cli/internal/tui"
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
			return executeProviderWithSmartSelection(prov, prof, trimDashDash(args[1:]), true)
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
	case "capacity-monitor":
		return capacityMonitorCmd(args[1:])
	case "web", "open":
		return controlWebCmd(args[1:])
	case "plan", "plans":
		return planCmd(args[1:])
	case "agents", "agent":
		return agentsCmd(args[1:])
	case "projects", "project":
		return projectsCmd(args[1:])
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
	case "config":
		return configCmd(args[1:])
	case "control", "ui":
		return controlCmd(args[1:])
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
		return executeProviderWithSmartSelection(prov, targetProfile, trimDashDash(rest), true)
	default:
		if strings.HasPrefix(args[0], "-") {
			// Auto selection for configured default provider
			cfg, _ := config.LoadConfig()
			for prov, prof := range cfg.Defaults {
				if prof != "" {
					return executeProviderWithSmartSelection(prov, prof, trimDashDash(args), true)
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
	res, err := tui.ShowMenu()
	if err != nil {
		return err
	}
	if res == nil || res.Action == tui.ActionQuit || res.Action == tui.ActionNone {
		return nil
	}
	switch res.Action {
	case tui.ActionRunProfile:
		return executeProviderWithSmartSelection(res.Provider, res.ProfileName, res.Args, true)
	case tui.ActionResumeConversation:
		return executeResume(res.Provider, res.ProfileName, res.ConversationID, res.Args)
	case tui.ActionLogin:
		return loginCmd([]string{res.Provider, res.ProfileName})
	default:
		return nil
	}
}

func executeProviderWithSmartSelection(provName, explicitProfile string, args []string, allowFallback bool) error {
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
		if p.Provider == provName {
			candidates = append(candidates, p)
			accounts[p.Name] = profile.GetAccountInfo(provName, p.Name)
			_ = profile.GetUsageSnapshot(provName, p.Name)
		}
	}

	if len(candidates) == 0 {
		return fmt.Errorf("no profiles configured for provider %s. Run: %s add %s <name>", provName, progName(), provName)
	}

	// Pre-check: warn if ALL accounts are unavailable before launching.
	allUnavailable := true
	for _, p := range candidates {
		acc := accounts[p.Name]
		snap, _ := qEng.GetCachedUsage(provName, p.Name)
		qv := quota.BuildQuotaView(snap, acc.Email, acc.Plan)
		if qv.IsAvailable() {
			allUnavailable = false
			break
		}
	}
	if allUnavailable {
		fmt.Fprintf(os.Stderr, "\n⚠  NENHUMA CONTA DISPONÍVEL para %s:\n\n", strings.ToUpper(provName))
		for _, p := range candidates {
			acc := accounts[p.Name]
			snap, _ := qEng.GetCachedUsage(provName, p.Name)
			qv := quota.BuildQuotaView(snap, acc.Email, acc.Plan)
			reason := qv.AvailabilityLabel()
			detail := ""
			if len(qv.AvailReasons.ExhaustedWindows) > 0 {
				detail = fmt.Sprintf(" (janelas esgotadas: %s)", strings.Join(qv.AvailReasons.ExhaustedWindows, ", "))
			}
			fmt.Fprintf(os.Stderr, "   ✗ %s: %s%s\n", p.Name, reason, detail)
		}
		fmt.Fprintf(os.Stderr, "\n   Aguarde o reset da quota ou use outro provider.\n\n")
	}

	cwd, _ := os.Getwd()
	ctx := context.Background()

	return exec.RunWithFallback(ctx, provName, cwd, explicitProfile, candidates, accounts, allowFallback, func(p model.Profile) (model.Failure, error) {
		accInfo := accounts[p.Name]
		accEmail := accInfo.Email
		if accEmail == "" {
			accEmail = p.Name
		}
		snap, _ := qEng.GetCachedUsage(provName, p.Name)
		var capInfo string
		qvCap := quota.BuildQuotaView(snap, accEmail, accInfo.Plan)
		if summary := qvCap.CompactGroupSummary(); summary != "" {
			capInfo = fmt.Sprintf(" [Cap: %s]", summary)
		}
		fmt.Fprintf(os.Stderr, "⚡ [%s] Usando perfil: %s:%s (%s)%s\n", progName(), provName, p.Name, accEmail, capInfo)

		start := time.Now()
		_ = telemetry.LogEvent(telemetry.Event{
			Type:       telemetry.EventSessionStarted,
			ProviderID: provName,
			ProfileID:  p.Name,
			Workspace:  cwd,
		})

		fail, runErr := pAdapter.Run(ctx, p, args)
		dur := time.Since(start)

		if fail.Kind != model.FailureNone && fail.Kind != "" {
			_ = telemetry.LogEvent(telemetry.Event{
				Type:        telemetry.EventRateLimitDetected,
				ProviderID:  provName,
				ProfileID:   p.Name,
				Workspace:   cwd,
				DurationMs:  dur.Milliseconds(),
				FailureKind: fail.Kind,
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
  %s control [subcmd]             Nexus Supervised Agent Runtimes (alias: %s ui)
  %s start <provider> [flags]     Start supervised agent runtime & attach
  %s stop <runtime-id>            Stop running supervised runtime
  %s ps / %s running              List active running supervised runtimes
  %s attach <runtime-id>          Attach terminal to running runtime
  %s handoff <id> <target>        Same-provider account handoff
  %s continue <id> --with <prov>  Cross-provider context handoff

  %s <provider> [flags]           Launch provider with intelligent account selection
  %s <provider>:<profile> [flags] Launch specific profile (e.g. %s codex:work)
  %s <provider>:auto [flags]      Explicit auto-selection
  %s resume [id] [provider:name]  Resume previous session using provider-native syntax

  %s providers [--json]           List installed providers, versions & capabilities
  %s profiles [--json]            List configured profiles, auth status & priorities
  %s add <provider> [name]        Add a new provider authentication profile
  %s remove <provider> <name>     Remove an existing profile
  %s login / logout <p> <name>    Run provider official login/logout flow
  %s use <provider> <name>        Set default active profile for provider
  %s status [provider[:profile]]  Display profile health, plan and account status
  %s usage [--json] [--refresh]   Searchable quota table; --json for integrations; --refresh for live CLI fetch
  %s inspect <provider> <name>    Inspect profile configuration details

  %s sessions [search] [--json]   Universal session index across all providers
  %s workspaces [--json]          View workspaces, session history & bindings
  %s bind <provider>:<profile>    Bind current workspace to a preferred profile
  %s unbind <provider>            Unbind current workspace
  %s bindings [--json]            List all active workspace bindings
  %s explain <provider>           Explain account selection decision and scores
  %s doctor [--json]              Deep diagnostics of runtime, keyrings & CLIs
  %s security [profile] [--json]  Audit file sharing and isolation boundary
  %s history [--json]             View local session execution log
  %s stats [--json]               Aggregated statistics (sessions, fallbacks, rate limits)
  %s config <show|validate>       Manage control plane settings
  %s update                       Update Nexus and Orquestrador Maestro to latest
  %s completion <bash|zsh|fish>   Generate shell completion scripts
  %s version [--json]             Display build and platform information
  %s release                      Interactively bump, build, install and validate Nexus

Universal Canonical Aliases (translated to native options for all providers):
  --yolo / -y                     Bypass approval prompts and permissions
  --continue / -c                 Continue most recent session
  --resume <id> / -r <id>         Resume session by ID
  --print / -p                    Run non-interactively
  --effort <low|med|high>         Set model reasoning effort
  --plan                          Start session in planning mode

Merged Help:
  %s <provider> --help            Show Nexus canonical aliases merged with official CLI help
`,
		p, p, p, p, p, p, p, p, p, p, p, p, p,
		p, p, p, p, p,
		p, p, p, p, p, p, p, p, p,
		p, p, p, p, p, p, p, p, p, p, p, p, p, p,
		p,
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
			email = "(unauthenticated)"
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
			fmt.Println("Cancelled.")
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

func loginCmd(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: %s login <provider> <name>", progName())
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
		return authProv.Login(context.Background(), p)
	}
	return fmt.Errorf("provider %s does not support login", prov)
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

func usageCmd(args []string) error {
	ps, err := profile.List()
	if err != nil {
		return err
	}
	if len(ps) == 0 {
		fmt.Println("No profiles configured.")
		return nil
	}

	isJSON := false
	refresh := false
	for _, arg := range args {
		switch arg {
		case "--json":
			isJSON = true
		case "--refresh":
			refresh = true
		}
	}
	if refresh {
		fmt.Fprintln(os.Stderr, "Atualizando quotas ao vivo…")
		for _, p := range ps {
			fmt.Fprintf(os.Stderr, "  • %s:%s\n", p.Provider, p.Name)
			_ = profile.RefreshUsageSnapshot(p.Provider, p.Name)
		}
	}
	if isJSON {
		var list []model.UsageSnapshot
		for _, p := range ps {
			list = append(list, profile.GetUsageSnapshot(p.Provider, p.Name))
		}
		b, _ := json.MarshalIndent(list, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	return usageTableCmd(ps)
}

func usageTableCmd(ps []model.Profile) error {
	rows := make([]tui.UsageTableRow, 0, len(ps)*2)
	for _, p := range ps {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		qv := profile.GetQuotaView(p.Provider, p.Name, acc.Plan, acc.Email)
		for _, group := range qv.ModelGroups {
			fiveHour, weekly := quotaWindowDisplay(group.Windows, "5h"), quotaWindowDisplay(group.Windows, "weekly")
			if fiveHour == "-" && weekly == "-" {
				continue
			}
			rows = append(rows, tui.UsageTableRow{Provider: p.Provider, Profile: p.Name, Account: acc.Email, Plan: acc.Plan, Group: group.Name, FiveHour: fiveHour, Weekly: weekly, Status: quotaGroupStatus(group, qv.Status)})
		}
	}
	if len(rows) == 0 {
		fmt.Println("Nenhuma quota disponível para exibir.")
		return nil
	}
	return tui.RunUsageTable(rows)
}

func quotaWindowDisplay(windows []quota.Window, kind string) string {
	for _, window := range windows {
		matches := kind == "5h" && (window.Kind == "5h" || window.Kind == "daily" || window.Kind == "claude_5h" || window.Kind == "claude_five_hour")
		if kind == "weekly" {
			matches = window.Kind == "weekly" || window.Kind == "claude_weekly"
		}
		if matches {
			reset := strings.TrimSpace(window.ResetDesc)
			if reset == "" {
				reset = "desconhecido"
			}
			return fmt.Sprintf("%2.0f%% / %s", window.Remaining, reset)
		}
	}
	return "-"
}

func quotaGroupStatus(group quota.ModelGroup, snapshotStatus string) string {
	if snapshotStatus == string(model.UsageUnknown) || snapshotStatus == string(model.UsageError) {
		return "UNKNOWN"
	}
	for _, window := range group.Windows {
		if window.Kind != "unknown" && window.Remaining <= 0 {
			return "INDISPONIVEL"
		}
	}
	return "DISPONIVEL"
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
			_ = profile.GetUsageSnapshot(prov, p.Name)
		}
	}

	cwd, _ := os.Getwd()
	fmt.Print(sel.ExplainSelection(context.Background(), prov, cwd, candidates, accounts))
	return nil
}

func doctorCmd(args []string) error {
	reg := initRegistry()
	ctx := context.Background()
	detections := reg.DetectAll(ctx)
	ps, _ := profile.List()

	if len(args) > 0 && args[0] == "--json" {
		out := map[string]interface{}{
			"detections": detections,
			"profiles":   ps,
		}
		b, _ := json.MarshalIndent(out, "", "  ")
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

func exportCmd(args []string) error {
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

func issueReportCmd(args []string) error {
	reg := initRegistry()
	ctx := context.Background()
	detections := reg.DetectAll(ctx)

	var sb strings.Builder
	sb.WriteString("### Environment Diagnostics\n\n")
	sb.WriteString(fmt.Sprintf("- IAPro Nexus Version: `%s`\n", buildinfo.Version))
	sb.WriteString(fmt.Sprintf("- OS: `%s`\n\n", runtime.GOOS))
	sb.WriteString("### Installed Providers\n\n")
	for id, det := range detections {
		sb.WriteString(fmt.Sprintf("- **%s**: Installed=%v, Version=`%s`\n", id, det.Installed, det.Version))
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
	return executeProviderWithSmartSelection(prov, prof, trimDashDash(rest), true)
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
    opts="web start stop ps running attach handoff continue resume control ui providers profiles add remove login logout use status usage inspect sessions workspaces bind unbind bindings explain doctor security history stats config update completion version release codex agy claude opencode gemini cursor"
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
        'update:Update Nexus and Orquestrador Maestro'
    )
    _describe 'command' commands
}
_nexus "$@"
`)
	case "fish":
		fmt.Print(`complete -c nexus -f -a "web start stop ps running attach handoff continue resume control ui providers profiles add remove login logout use status usage sessions workspaces bind unbind explain doctor security history stats update config version release"
complete -c ai -f -a "web start stop ps running attach handoff continue resume control ui providers profiles add remove login logout use status usage sessions workspaces bind unbind explain doctor security history stats update config version release"
`)
	case "powershell", "pwsh":
		fmt.Print(`Register-ArgumentCompleter -Native -CommandName @('nexus', 'ai') -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $commands = @(
        'web', 'start', 'stop', 'ps', 'running', 'attach', 'handoff', 'continue', 'resume',
        'control', 'ui', 'providers', 'profiles', 'add', 'remove', 'login', 'logout', 'use',
        'status', 'usage', 'inspect', 'sessions', 'workspaces', 'bind', 'unbind', 'bindings',
        'explain', 'doctor', 'security', 'history', 'stats', 'config', 'update', 'completion', 'version', 'release',
        'codex', 'agy', 'claude', 'opencode', 'gemini', 'cursor'
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
