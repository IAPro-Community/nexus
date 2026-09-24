package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/handoff"
	"github.com/IAPro-Community/nexus/internal/control/host"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	controlterminal "github.com/IAPro-Community/nexus/internal/control/terminal"
	controltui "github.com/IAPro-Community/nexus/internal/control/tui"
	"github.com/IAPro-Community/nexus/internal/control/web"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/cooldown"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	providerspec "github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/core/scheduler"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/promotion"
	"github.com/IAPro-Community/nexus/internal/profile"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

func controlCmd(args []string) error {
	if len(args) == 0 {
		target, err := controltui.RunControlTUI(context.Background())
		if err != nil {
			return err
		}
		if target != "" {
			return attachRuntime(target)
		}
		return nil
	}

	switch args[0] {
	case "ui", "tui":
		target, err := controltui.RunControlTUI(context.Background())
		if err != nil {
			return err
		}
		if target != "" {
			return attachRuntime(target)
		}
		return nil
	case "monitor":
		runtimeID := ""
		if len(args) > 1 {
			runtimeID = args[1]
		}
		return controltui.RunControlMonitor(context.Background(), runtimeID)
	case "web":
		return controlWebCmd(args[1:])
	case "help", "-h", "--help":
		controlHelp()
		return nil
	case "start":
		return controlStartCmd(args[1:])
	case "running", "list", "ls":
		return controlRunningCmd(args[1:])
	case "status":
		return controlStatusCmd(args[1:])
	case "attach":
		return controlAttachCmd(args[1:])
	case "stop":
		return controlStopCmd(args[1:])
	case "handoff":
		return controlHandoffCmd(args[1:])
	case "continue":
		return controlContinueCmd(args[1:])
	case "cleanup":
		return controlCleanupCmd(args[1:])
	case "doctor":
		return controlDoctorCmd(args[1:])
	default:
		return fmt.Errorf("unknown control command %q. Run '%s control --help' for usage", args[0], progName())
	}
}

func controlHelp() {
	p := progName()
	fmt.Printf(`Nexus Control Center — Supervised Agent Runtimes

USAGE:
  %s control [subcommand] [flags]
  %s ui

SUBCOMMANDS:
  (no args)                     Open interactive Bubble Tea Control Center
  monitor [runtime-id]          Open read-only sidecar monitor (never attaches)
  start <provider> [--profile <name>] [args...]
                                Start a supervised runtime session
  running [--json]              List active managed runtimes (alias: %s ps)
  status <runtime-id> [--json]  Display runtime details
  attach <runtime-id>           Connect terminal to running session
  stop <runtime-id>             Gracefully stop a runtime
  handoff <runtime-id> <provider:profile>
                                Same-provider account handoff
  continue <runtime-id> --with <provider[:profile]>
                                Cross-provider context handoff
  cleanup                       Clean up stale runtime records and dead sockets
  doctor [--json] [--repair]     Audit control runtime environment and drivers
  web [--port <port>] [--no-open] [--listen <ip>] [--remote]
                                Open browser-based Web Workspace OS
                                Default port: 13000 (override with NEXUS_WEB_PORT env var)
                                Registers nexus.dev in /etc/hosts for local name resolution
  web open                      Reopen the running Web UI from any terminal
  web url                       Print the current Web UI URL (with auth token if needed)

FLAGS:
  --json                        Output in machine-readable JSON format
  -h, --help                    Show this help message
`, p, p, p)
}

func controlWebCmd(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "open":
			return controlWebReopenCmd(false)
		case "url":
			return controlWebReopenCmd(true)
		}
	}

	var port int
	var host = "127.0.0.1"
	var noOpen bool
	var remote bool

	// Resolve port: CLI flag > env var > default (13000)
	port = web.DefaultPort
	if envPort := os.Getenv("NEXUS_WEB_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 && p <= 65535 {
			port = p
		}
	}

	for i := 0; i < len(args); i++ {
		if (args[i] == "--port" || args[i] == "-p") && i+1 < len(args) {
			p, err := strconv.Atoi(args[i+1])
			if err == nil && p >= 0 && p <= 65535 {
				port = p
			}
			i++
		} else if (args[i] == "--listen" || args[i] == "-l") && i+1 < len(args) {
			host = args[i+1]
			i++
		} else if args[i] == "--no-open" {
			noOpen = true
		} else if args[i] == "--remote" {
			remote = true
		}
	}

	core, err := NewCore(CoreConfig{
		Host:   host,
		Port:   port,
		NoOpen: noOpen,
		Remote: remote,
	})
	if err != nil {
		return err
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down Nexus Control Center...")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = core.Stop(ctx)
		os.Exit(0)
	}()

	go func() {
		<-core.Ready()
		bootstrapURL := core.BootstrapURL()
		fmt.Println("=== Nexus Control Center (Web UI) ===")
		fmt.Printf("URL:       %s\n", core.URL())
		fmt.Println("Press Ctrl+C to stop the Web Control Center.")
		fmt.Printf("Reopen from any terminal: %s web open\n", progName())

		if !noOpen {
			time.Sleep(200 * time.Millisecond)
			_ = web.OpenBrowser(bootstrapURL)
		}
	}()

	return core.Start(context.Background())
}

func controlWebReopenCmd(printOnly bool) error {
	state, err := web.ReadListenState()
	if err != nil {
		return fmt.Errorf("nexus web não está em execução neste usuário. Inicie com `%s web` e, se o browser não abrir, rode `%s web open`", progName(), progName())
	}
	target := state.BootstrapURL
	if target == "" {
		target = state.URL
	}
	if state.BootstrapToken != "" && !strings.Contains(target, "#nexus_bootstrap=") {
		target = strings.TrimRight(state.URL, "/") + "/#nexus_bootstrap=" + state.BootstrapToken
	}
	if printOnly {
		fmt.Println(target)
		return nil
	}
	if err := web.OpenBrowser(target); err != nil {
		fmt.Println(target)
		return fmt.Errorf("não foi possível abrir o browser: %w", err)
	}
	fmt.Printf("Opened %s\n", target)
	return nil
}

// controlHostCmd is the internal background daemon worker running as: ai __control-host --runtime <runtime-id>
func controlHostCmd(args []string) error {
	var runtimeID string
	for i := 0; i < len(args); i++ {
		if args[i] == "--runtime" && i+1 < len(args) {
			runtimeID = args[i+1]
			i++
		}
	}
	if runtimeID == "" {
		return errors.New("missing --runtime <id> parameter")
	}

	reg := registry.DefaultRegistry()
	sess, ok := reg.Get(runtimeID)
	if !ok {
		return fmt.Errorf("runtime %q not found in registry", runtimeID)
	}

	d, err := driver.DefaultRegistry().Get(sess.ProviderID)
	if err != nil {
		return fmt.Errorf("driver not found for provider %s: %w", sess.ProviderID, err)
	}

	p := model.Profile{Name: sess.ProfileID, Provider: sess.ProviderID}
	launchArgs, customCommand, envelopeErr := launcher.ConsumeLaunchEnvelope(runtimeID)
	if envelopeErr != nil {
		// Compatibility for runtimes created before private launch envelopes.
		launchArgs = sess.Args
		if len(launchArgs) == 0 {
			return fmt.Errorf("failed to consume private launch envelope: %w", envelopeErr)
		}
	}
	bin, cmdArgs, env := sess.Binary, launchArgs, sess.Env
	if !customCommand {
		bin, cmdArgs, env, err = d.BuildCommand(context.Background(), p, launchArgs)
		if err != nil {
			return fmt.Errorf("failed to build runtime command: %w", err)
		}
	}

	sh, err := host.NewSessionHost(host.Config{
		Session:  sess,
		Registry: reg,
		Binary:   bin,
		Args:     cmdArgs,
		Env:      env,
		Cwd:      sess.Workspace,
	})
	if err != nil {
		return fmt.Errorf("failed to create SessionHost: %w", err)
	}

	if err := sh.Start(); err != nil {
		return fmt.Errorf("failed to start SessionHost: %w", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		sh.Stop()
	}()

	sh.Wait()
	return nil
}

func controlStartCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s start <provider> [--profile <name>] [args...]", progName())
	}

	providerID := args[0]
	var profileName string
	var extraArgs []string

	for i := 1; i < len(args); i++ {
		if (args[i] == "--profile" || args[i] == "-p") && i+1 < len(args) {
			profileName = args[i+1]
			i++
		} else {
			extraArgs = append(extraArgs, args[i])
		}
	}

	ctx := context.Background()

	fmt.Fprintf(os.Stderr, "⚡ [nexus] iniciando %s…\n", providerID)

	// 1. Resolve Profile using the shared explicit/best/default fallback policy.
	cwd, _ := os.Getwd()
	cfg, _ := config.LoadConfig()
	qEng := quota.NewEngine(5 * time.Minute)
	sel := scheduler.NewSelector(cfg, qEng, cooldown.NewTracker())
	allProfiles, err := profile.List()
	if err != nil {
		return err
	}
	var candidates []model.Profile
	accounts := make(map[string]model.AccountInfo)
	for _, candidate := range allProfiles {
		if candidate.Provider != providerID || candidate.Disabled {
			continue
		}
		candidates = append(candidates, candidate)
		account := profile.GetAccountInfo(providerID, candidate.Name)
		snapshot := profile.PeekUsageSnapshot(providerID, candidate.Name)
		account.Usage = snapshot
		accounts[candidate.Name] = account
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no enabled profiles configured for provider %s", providerID)
	}
	candidates, err = launchableProfiles(providerID, profileName, candidates)
	if err != nil {
		return err
	}

	// 2. Launch via unified RuntimeLauncher. Auto-selected profiles fall back
	// when Codex exclusive maintenance holds the TUI lock.
	sess, profileName, err := launchWithBusyFailover(
		ctx,
		launcher.Default(),
		sel,
		providerID,
		profileName,
		cwd,
		cfg.Defaults[providerID],
		candidates,
		accounts,
		launcher.LaunchOptions{Args: extraArgs},
		os.Stderr,
	)
	if err != nil {
		if errors.Is(err, codex.ErrTUIBusy) {
			return busyProfileError(providerID, profileName)
		}
		return err
	}

	fmt.Printf("✓ Started supervised %s runtime (ID: %s, Profile: %s, Control: %s)\n",
		strings.ToUpper(providerID), sess.RuntimeID, profileName, sess.ControlLevel)
	fmt.Printf("  Connecting interactive terminal...\n\n")

	time.Sleep(50 * time.Millisecond)
	return attachRuntime(sess.RuntimeID)
}

func controlRunningCmd(args []string) error {
	jsonMode := hasFlag(args, "--json")
	reg := registry.DefaultRegistry()
	_, _ = reg.CleanupStale()
	active := reg.ListActive()

	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(active)
	}

	if len(active) == 0 {
		fmt.Printf("No active supervised runtimes. Start one with: %s start <provider>\n", progName())
		return nil
	}

	fmt.Printf("%-16s %-10s %-15s %-12s %-12s %s\n", "ID", "PROVIDER", "PROFILE", "STATE", "CONTROL", "WORKSPACE")
	fmt.Println(strings.Repeat("─", 80))
	for _, s := range active {
		fmt.Printf("%-16s %-10s %-15s %-12s %-12s %s\n",
			s.RuntimeID, strings.ToUpper(s.ProviderID), s.ProfileID, s.State, s.ControlLevel, s.Workspace,
		)
	}
	return nil
}

func controlStatusCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s control status <runtime-id> [--json]", progName())
	}
	runtimeID := args[0]
	jsonMode := hasFlag(args, "--json")

	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		// Fallback to registry
		reg := registry.DefaultRegistry()
		s, ok := reg.Get(runtimeID)
		if !ok {
			return fmt.Errorf("runtime %q not found or endpoint unreachable: %w", runtimeID, err)
		}
		if jsonMode {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(s)
		}
		fmt.Printf("Runtime ID:  %s\nProvider:    %s\nProfile:     %s\nState:       %s (Host unreachable)\n", s.RuntimeID, s.ProviderID, s.ProfileID, s.State)
		return nil
	}
	defer client.Close()

	st, err := client.Status()
	if err != nil {
		return err
	}

	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(st)
	}

	fmt.Printf("Runtime ID:         %s\n", st.RuntimeID)
	fmt.Printf("Provider:           %s\n", strings.ToUpper(st.ProviderID))
	fmt.Printf("Profile:            %s\n", st.ProfileID)
	if st.ProviderSessionID != "" {
		fmt.Printf("Provider Session:   %s\n", st.ProviderSessionID)
	}
	fmt.Printf("PID:                %d\n", st.PID)
	fmt.Printf("State:              %s\n", st.State)
	fmt.Printf("Control Level:      %s\n", st.ControlLevel)
	fmt.Printf("Workspace:          %s\n", st.Workspace)
	fmt.Printf("Started At:         %s\n", st.StartedAt.Format(time.RFC3339))
	if st.DroppedOutputChunks > 0 {
		fmt.Printf("Dropped Output:     %d chunks (slow consumer)\n", st.DroppedOutputChunks)
	}
	return nil
}

func controlAttachCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s attach <runtime-id>", progName())
	}
	return attachRuntime(args[0])
}

func attachRuntime(runtimeID string) error {
	return attachRuntimeWithInput(runtimeID, nil)
}

func attachRuntimeWithInput(runtimeID string, input func(io.Writer) error) error {
	// No compatibility gate here (retired with the maximus sync): the runtime
	// handshake below (generation/PID check) is the fail-closed attach
	// authority. Unknown IDs still fail fast with a recreate hint.
	session, ok := registry.DefaultRegistry().Get(runtimeID)
	if !ok {
		return fmt.Errorf("runtime %q not found; start a new session with: %s codex --yolo", runtimeID, progName())
	}
	previousRuntime, hadPreviousRuntime := os.LookupEnv("NEXUS_RUNTIME_ID")
	_ = os.Setenv("NEXUS_RUNTIME_ID", runtimeID)
	defer func() {
		if hadPreviousRuntime {
			_ = os.Setenv("NEXUS_RUNTIME_ID", previousRuntime)
		} else {
			_ = os.Unsetenv("NEXUS_RUNTIME_ID")
		}
	}()

	tty := controlterminal.NewTTYController(os.Stdin, os.Stdout)
	if err := tty.Enter(); err != nil {
		return err
	}
	defer func() { _ = tty.Restore() }()

	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return fmt.Errorf("failed to attach to runtime %q: %w", runtimeID, err)
	}
	defer client.Close()

	// 1. Put user terminal into Raw Mode if stdin is a terminal
	isTerm := term.IsTerminal(int(os.Stdin.Fd()))
	defer nexusruntime.GuardInteractiveTerminalKeyboardModes(os.Stdin, os.Stdout)()
	if isTerm {
		// 2. Register window resize signal listener
		sigChan := make(chan os.Signal, 1)
		protocol.NotifyWinSizeChange(sigChan)
		defer signal.Stop(sigChan)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		interruptChan := make(chan os.Signal, 1)
		signal.Notify(interruptChan, os.Interrupt)
		defer signal.Stop(interruptChan)

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-interruptChan:
					_ = tty.Reassert()
				case <-sigChan:
					if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil && w > 0 && h > 0 {
						go func(rows, cols int) {
							rpcClient, err := protocol.NewClient(runtimeID)
							if err == nil {
								_ = rpcClient.Resize(rows, cols)
								_ = rpcClient.Close()
							}
						}(h, w)
					}
				}
			}
		}()

		// Send initial terminal window size
		if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil && w > 0 && h > 0 {
			_ = client.Resize(h, w)
		}
	}

	status, err := client.Status()
	if err != nil {
		_ = registry.DefaultRegistry().UpdateState(runtimeID, registry.StateStale)
		return fmt.Errorf("runtime %q could not prove its host generation before attach: %v; recreate the session with %s codex --yolo", runtimeID, err, progName())
	}
	if status.RuntimeID != runtimeID || status.PID != session.PID ||
		(status.HostPID > 0 && status.HostPID != session.HostPID) ||
		(status.HostGeneration > 0 && status.HostGeneration != session.HostGeneration) {
		_ = registry.DefaultRegistry().UpdateState(runtimeID, registry.StateStale)
		return fmt.Errorf("runtime %q changed generation/PID before attach and was rejected; recreate the session with %s codex --yolo (the old process was not terminated)", runtimeID, progName())
	}

	// 3. Send Attach command to switch connection to raw streaming
	resp, err := client.Send(protocol.CmdAttach, nil)
	if err != nil {
		return fmt.Errorf("attach failed: %w", err)
	}

	// Disable deadline after Attach RPC
	_ = client.ClearDeadline()

	// Print initial ring buffer history
	var history string
	if json.Unmarshal(resp.Data, &history) == nil && history != "" {
		os.Stdout.WriteString(history)
	}

	rawConn := client.RawConn()
	output := nexusruntime.NewTerminalOutputScrubber(os.Stdout)
	inputStream := nexusruntime.NewTerminalInputScrubber(rawConn)
	defer func() { _ = output.Flush() }()
	defer func() { _ = inputStream.Flush() }()

	// 4. Stream stdout from runtime host to user terminal
	errChan := make(chan error, 2)
	go func() {
		defer rawConn.Close()
		r := client.Reader()
		if r != nil && r.Buffered() > 0 {
			buf := make([]byte, r.Buffered())
			n, _ := r.Read(buf)
			if n > 0 {
				_, _ = output.Write(buf[:n])
			}
		}
		_, copyErr := io.Copy(output, rawConn)
		errChan <- copyErr
	}()

	// 5. Stream user stdin to runtime host
	go func() {
		defer rawConn.Close()
		if input != nil {
			errChan <- input(inputStream)
			return
		}
		_, copyErr := io.Copy(inputStream, os.Stdin)
		errChan <- copyErr
	}()

	<-errChan
	// The stream can end without Ctrl+C (provider crash, endpoint loss, or
	// disconnect). Recover before deferred restoration on every exit.
	_ = tty.Reassert()
	return nil
}

// attachInteractiveLeadRuntime keeps the Lead Agent lifecycle bound to the
// supervised session while preserving byte-transparent provider I/O. Complete
// prompt lines pass through the promotion policy; control sequences and
// bracketed paste are never interpreted as prompt text.
func attachInteractiveLeadRuntime(runtimeID string, n *nexus.Nexus, session registry.RuntimeSession, agentID string) error {
	if n != nil {
		defer func() { _ = n.ReleaseInteractiveLeadRuntime(context.Background(), agentID, runtimeID) }()
	}
	if n == nil || session.ProjectID == "" || agentID == "" {
		return attachRuntime(runtimeID)
	}
	mode := interactiveDelegationMode(session)
	return attachRuntimeWithInput(runtimeID, func(conn io.Writer) error {
		shadow := promotion.NewLineShadow()
		if mode == nexus.DelegationOff {
			shadow.SetEnabled(false)
		}
		promoter := promotion.NewPromoter(n)
		buf := make([]byte, 1024)
		pendingASK := ""
		startMission := func(prompt string, fallbackToLead bool) {
			accepted, startErr := startInteractiveDelegatedMission(n, session, agentID, prompt, mode)
			if startErr != nil {
				safeAdmissionFallback := fallbackToLead && accepted != nil && accepted.AdmissionBlocked
				if safeAdmissionFallback {
					// A heuristic promotion is disposable only when the Core
					// proved that Mission admission failed before a run existed.
					// Remove its temporary WorkPlan before returning ownership
					// to the interactive Lead.
					if strings.TrimSpace(accepted.PlanID) != "" {
						_ = n.DeleteWorkPlan(context.Background(), accepted.PlanID)
					}
					// The prompt is still present in the provider TUI because
					// Enter was held. Submit it unchanged to the Lead exactly once.
					_, _ = conn.Write([]byte("\r"))
					_, _ = fmt.Fprintf(os.Stderr, "\r\n[nexus] Mission não admitida (%v); continuando com o Lead.\r\n", startErr)
					return
				}

				// Explicit Mission requests never silently degrade to Direct.
				// The same conservative rule applies to untyped/internal start
				// failures: without a proven admission boundary, do not risk
				// duplicate side effects by submitting the prompt to the Lead.
				_, _ = conn.Write([]byte(clearLineSequence(session.ProviderID)))
				if accepted != nil && accepted.AdmissionBlocked {
					if strings.TrimSpace(accepted.PlanID) != "" {
						_, _ = fmt.Fprintf(os.Stderr, "\r\n[nexus] Mission solicitada, mas a admissão foi bloqueada (plan %s, etapa %s): %v\r\n", accepted.PlanID, accepted.AdmissionStage, startErr)
					} else {
						_, _ = fmt.Fprintf(os.Stderr, "\r\n[nexus] Mission solicitada, mas a admissão foi bloqueada (etapa %s): %v\r\n", accepted.AdmissionStage, startErr)
					}
				} else {
					_, _ = fmt.Fprintf(os.Stderr, "\r\n[nexus] Promoção para Mission falhou antes da confirmação de admissão: %v. Fallback Direct desativado por segurança.\r\n", startErr)
				}
				return
			}
			_, _ = conn.Write([]byte(clearLineSequence(session.ProviderID)))
			_, _ = fmt.Fprintf(os.Stderr, "\r\n[nexus] Mission %s iniciada; o Lead receberá a síntese quando ela estabilizar.\r\n", accepted.RunID)
			go waitInteractiveDelegatedMission(n, session, agentID, prompt, accepted)
		}
		for {
			nread, err := os.Stdin.Read(buf)
			if nread > 0 {
				input := buf[:nread]
				if pendingASK != "" {
					answer := input[0]
					input = input[1:]
					switch answer {
					case 'y', 'Y':
						// Confirmation makes this a user-owned Mission request:
						// admission failure must be shown, not executed as Direct.
						startMission(pendingASK, false)
					case 'n', 'N':
						_, _ = conn.Write([]byte("\r"))
					default:
						_, _ = conn.Write([]byte("\r"))
					}
					pendingASK = ""
				}
				if len(input) == 0 {
					if err != nil {
						return err
					}
					continue
				}
				event := shadow.Consume(input)
				if _, writeErr := conn.Write(event.Forward); writeErr != nil {
					return writeErr
				}
				if event.Submit {
					for _, line := range event.Submissions {
						// Hold Enter through both routing and Mission admission.
						// The provider receives the original line only when the
						// Lead owns it or a heuristic Mission cannot be admitted.
						decision := decideInteractiveSubmitDecision(context.Background(), promoter, session.ProjectID, line, mode)
						if decision.Action == promotion.KeepWithLead {
							_, _ = conn.Write([]byte("\r"))
							continue
						}
						if decision.Action == promotion.Ask {
							_, _ = fmt.Fprint(os.Stderr, "\r\n[nexus] Promover esta solicitação para uma Mission? [y/n] ")
							pendingASK = line
							continue
						}
						startMission(line, shouldFallbackPromotionToLead(decision))
					}
				}
			}
			if err != nil {
				return err
			}
		}
	})
}

func clearLineSequence(providerID string) string {
	if provider, ok := providerspec.Get(providerID); ok && provider.ClearLine != "" {
		return provider.ClearLine
	}
	return "\x15"
}

func controlStopCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s stop <runtime-id>", progName())
	}
	runtimeID := args[0]
	reg := registry.DefaultRegistry()
	client, err := protocol.NewClient(runtimeID)
	if err == nil {
		defer client.Close()
		if stopErr := client.Stop(); stopErr == nil {
			fmt.Printf("✓ Sent stop signal to runtime %s\n", runtimeID)
			return nil
		}
	}

	// Fallback if socket unreachable: check registry and validate PID identity before kill
	if s, ok := reg.Get(runtimeID); ok {
		if s.PID > 0 {
			if registry.IsProcessAliveWithGeneration(s.PID, s.HostGeneration) {
				if p, pErr := os.FindProcess(s.PID); pErr == nil {
					_ = p.Kill()
				}
			} else if registry.IsProcessAlive(s.PID) {
				// Process is alive but host generation does not match: PID was recycled by the OS!
				_ = os.Remove(protocol.EndpointPath(runtimeID))
				_ = reg.UpdateState(runtimeID, registry.StateStale)
				return fmt.Errorf("safety abort: PID %d is active but does not match runtime %s host generation (PID recycled); marked as STALE without killing", s.PID, runtimeID)
			}
		}
		_ = os.Remove(protocol.EndpointPath(runtimeID))
		_ = reg.Delete(runtimeID)
		fmt.Printf("✓ Cleaned up dead runtime %s\n", runtimeID)
		return nil
	}

	return fmt.Errorf("runtime %q not found or already stopped", runtimeID)
}

func controlHandoffCmd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: %s handoff <runtime-id> <provider:profile>", progName())
	}
	runtimeID := args[0]
	target := args[1]

	newSess, err := handoff.PerformAccountHandoff(context.Background(), runtimeID, target)
	if err != nil {
		return fmt.Errorf("account handoff failed: %w", err)
	}

	fmt.Printf("✓ Account handoff completed: %s -> %s (New Runtime: %s)\n", runtimeID, target, newSess.RuntimeID)
	return attachRuntime(newSess.RuntimeID)
}

func controlContinueCmd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: %s continue <runtime-id> --with <provider[:profile]>", progName())
	}
	runtimeID := args[0]
	var targetProvider, targetProfile string

	for i := 1; i < len(args); i++ {
		if (args[i] == "--with" || args[i] == "-w") && i+1 < len(args) {
			target := args[i+1]
			if idx := strings.Index(target, ":"); idx != -1 {
				targetProvider = target[:idx]
				targetProfile = target[idx+1:]
			} else {
				targetProvider = target
			}
			i++
		}
	}

	if targetProvider == "" {
		return errors.New("missing --with <provider[:profile]> argument")
	}

	newSess, err := handoff.PerformContextHandoff(context.Background(), runtimeID, targetProvider, targetProfile)
	if err != nil {
		return fmt.Errorf("context handoff failed: %w", err)
	}

	fmt.Printf("✓ Context handoff completed: %s -> %s:%s (New Runtime: %s)\n", runtimeID, targetProvider, targetProfile, newSess.RuntimeID)
	return attachRuntime(newSess.RuntimeID)
}

func controlCleanupCmd(_ []string) error {
	reg := registry.DefaultRegistry()
	cleaned, _ := reg.CleanupStale()
	reaped := launcher.ReapOrphanHosts(reg.List())
	purged, _ := reg.PurgeInactive()
	fmt.Printf("✓ Cleaned up %d stale records, reaped %d orphan hosts, and purged %d inactive runtimes.\n", cleaned, reaped, purged)
	return nil
}

func controlDoctorCmd(args []string) error {
	jsonMode := hasFlag(args, "--json")
	ctx := context.Background()

	reg := registry.DefaultRegistry()
	staleCount := 0
	for _, session := range reg.List() {
		if session.State == registry.StateStale {
			staleCount++
		}
	}
	if hasFlag(args, "--repair") {
		staleCount, _ = reg.CleanupStale()
	}

	var filterProvider string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			filterProvider = strings.ToLower(a)
			break
		}
	}

	allDrivers := driver.DefaultRegistry().List()
	var drivers []driver.ControlDriver
	for _, d := range allDrivers {
		if filterProvider == "" || d.ProviderID() == filterProvider {
			drivers = append(drivers, d)
		}
	}
	type driverReport struct {
		Provider     string                       `json:"provider"`
		Installed    bool                         `json:"installed"`
		Version      string                       `json:"version,omitempty"`
		BinaryPath   string                       `json:"binary_path,omitempty"`
		ControlLevel registry.ControlLevel        `json:"control_level"`
		Capabilities driver.EffectiveCapabilities `json:"capabilities"`
		Error        string                       `json:"error,omitempty"`
	}

	var dReports []driverReport
	showAll := hasFlag(args, "--all") || hasFlag(args, "--test-mode")
	for _, d := range drivers {
		if d.ProviderID() == "fake" && !showAll {
			continue
		}
		det, _ := d.Detect(ctx)
		effCaps := d.EffectiveCaps(ctx, model.Profile{Name: "default", Provider: d.ProviderID()})
		dReports = append(dReports, driverReport{
			Provider:     d.ProviderID(),
			Installed:    det.Installed,
			Version:      det.Version,
			BinaryPath:   det.BinaryPath,
			ControlLevel: effCaps.ControlLevel,
			Capabilities: effCaps,
			Error:        det.Error,
		})
	}

	ipcMechanism := "Unix Domain Socket"
	if runtime.GOOS == "windows" {
		ipcMechanism = "Windows Named Pipe"
	}

	termMechanism := "Unix PTY (creack/pty)"
	if runtime.GOOS == "windows" {
		termMechanism = "Windows ConPTY / Standard Pipes"
	}

	report := map[string]any{
		"platform": map[string]string{
			"os":       runtime.GOOS,
			"arch":     runtime.GOARCH,
			"ipc":      ipcMechanism,
			"terminal": termMechanism,
		},
		"runtimes_total":  len(reg.List()),
		"runtimes_active": len(reg.ListActive()),
		"stale_cleaned":   staleCount,
		"drivers":         dReports,
	}

	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	fmt.Println("=== Nexus Control Runtime Doctor ===")
	fmt.Printf("Platform:         %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("IPC Transport:    %s\n", ipcMechanism)
	fmt.Printf("Terminal Engine:  %s\n", termMechanism)
	fmt.Printf("Active Runtimes:  %d\n", len(reg.ListActive()))
	fmt.Printf("Total Runtimes:   %d\n", len(reg.List()))
	fmt.Printf("Stale Cleaned:    %d\n\n", staleCount)

	fmt.Println("Provider Drivers Truthful Status:")
	for _, dr := range dReports {
		mark := "✓"
		if !dr.Installed {
			mark = "✗"
		}
		verStr := dr.Version
		if verStr == "" {
			verStr = "not installed"
		}
		fmt.Printf(" %s %-10s (Version: %-18s | Control: %-8s | Resume: %s)\n",
			mark, strings.ToUpper(dr.Provider), verStr, dr.ControlLevel, dr.Capabilities.Resume.Status)
	}

	return nil
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}
