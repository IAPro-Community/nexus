package launcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/host"
	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/control/launchenv"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

// LaunchOptions defines parameters for launching a supervised AI runtime.
type LaunchOptions struct {
	RuntimeID         string
	AgentID           string
	ProjectID         string
	ProjectName       string
	Title             string
	ProviderID        string
	ProfileID         string
	AccountScope      model.AccountScope `json:"account_scope,omitempty"`
	ProviderSessionID string
	Workspace         string
	Args              []string
	Standalone        bool // If true, runs host in-process instead of spawning detached daemon
	Timeout           time.Duration
	Model             string            `json:"model,omitempty"`
	Environment       map[string]string `json:"environment,omitempty"`
	Isolation         string            `json:"isolation,omitempty"`
	Options           map[string]any    `json:"options,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	PathPrepend       []string          `json:"path_prepend,omitempty"`
}

// Launcher unifies supervised SessionHost spawning and handshake verification across all commands.
type Launcher struct {
	reg     *registry.Registry
	drivers *driver.Registry
}

var defaultLauncher = NewLauncher()

func Default() *Launcher {
	return defaultLauncher
}

func NewLauncher() *Launcher {
	return &Launcher{
		reg:     registry.DefaultRegistry(),
		drivers: driver.DefaultRegistry(),
	}
}

func traceCaptureShellLaunchStage(stage string) {
	if os.Getenv("NEXUS_DOCS_CAPTURE") == "1" {
		slog.Info("nexus_docs_capture_launcher_shell_stage=" + stage)
	}
}

// Launch allocates, starts, and verifies a supervised runtime.
func (l *Launcher) Launch(ctx context.Context, opts LaunchOptions) (*registry.RuntimeSession, error) {
	captureShell := strings.EqualFold(opts.ProviderID, "shell")
	if captureShell {
		traceCaptureShellLaunchStage("start")
	}
	if opts.RuntimeID == "" {
		opts.RuntimeID = fmt.Sprintf("%s-%s", opts.ProviderID, ids.NewRuntimeID())
	}
	if opts.Workspace == "" {
		if strings.TrimSpace(opts.ProjectID) != "" {
			return nil, fmt.Errorf("project runtime requires an explicit workspace (refusing server CWD fallback)")
		}
		cwd, _ := os.Getwd()
		opts.Workspace = cwd
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}
	cleanupOrphanLaunchEnvelopes(10 * time.Minute)
	if captureShell {
		traceCaptureShellLaunchStage("orphan-cleanup-done")
	}

	if captureShell {
		traceCaptureShellLaunchStage("driver-lookup-start")
	}
	d, err := l.drivers.Get(opts.ProviderID)
	if captureShell {
		traceCaptureShellLaunchStage("driver-lookup-finished")
	}
	if err != nil {
		return nil, fmt.Errorf("unknown driver for provider %q: %w", opts.ProviderID, err)
	}

	prof := model.Profile{
		Provider: opts.ProviderID,
		Name:     opts.ProfileID,
	}

	launchModel := opts.Model
	if strings.EqualFold(strings.TrimSpace(opts.ProviderID), "agy") && strings.TrimSpace(launchModel) == "" {
		quotaEngine := quota.NewEngine(quota.DefaultTTL)
		var cachedUsage model.UsageSnapshot
		if opts.AccountScope.Verifiable() {
			cachedUsage, _ = quotaEngine.GetCachedUsageForScope(opts.AccountScope)
		}
		if !quotaEngine.Trustworthy(cachedUsage) {
			cachedUsage.Windows = nil
		}
		launchModel = driver.ResolveLaunchModel(opts.ProviderID, launchModel, cachedUsage)
	}
	if captureShell {
		traceCaptureShellLaunchStage("launch-config-start")
	}
	configuredArgs, err := driver.ApplyLaunchConfiguration(opts.ProviderID, launchModel, opts.Options, opts.Args)
	if captureShell {
		traceCaptureShellLaunchStage("launch-config-finished")
	}
	if err != nil {
		return nil, fmt.Errorf("invalid launch configuration for %s:%s: %w", opts.ProviderID, opts.ProfileID, err)
	}
	var bin string
	var extraArgs []string
	var env []string
	customCommand := false

	if template, ok := opts.Options["command_template"].(string); ok && strings.TrimSpace(template) != "" {
		// A custom target owns provider installation/isolation on the far side
		// (Docker/VPN, WSL, SSH, wrapper, etc.). Do not require the logical
		// provider binary to exist on the Nexus host before resolving the
		// wrapper command.
		profileHome := ""
		if value, ok := opts.Options["target_profile_home"].(string); ok {
			profileHome = strings.TrimSpace(value)
		}
		targetWorkspace := ""
		if value, ok := opts.Options["target_workspace"].(string); ok {
			targetWorkspace = strings.TrimSpace(value)
		}
		bin, extraArgs, err = ResolveCommandTemplateContext(template, CommandTemplateContext{
			Cwd:         opts.Workspace,
			TargetCwd:   targetWorkspace,
			Args:        configuredArgs,
			Provider:    opts.ProviderID,
			Profile:     opts.ProfileID,
			ProfileHome: profileHome,
		})
		if err != nil {
			return nil, fmt.Errorf("invalid command template: %w", err)
		}
		customBaseEnv := append([]string(nil), os.Environ()...)
		customBaseEnv = append(customBaseEnv,
			"AI_PROVIDER="+opts.ProviderID,
			"AI_PROFILE="+opts.ProfileID,
		)
		env = launchenv.Merge(customBaseEnv, opts.Environment, opts.PathPrepend)
		customCommand = true
	} else {
		if captureShell {
			traceCaptureShellLaunchStage("build-command-start")
		}
		bin, extraArgs, env, err = d.BuildCommand(ctx, prof, configuredArgs)
		if captureShell {
			traceCaptureShellLaunchStage("build-command-finished")
		}
		if err != nil {
			return nil, fmt.Errorf("failed to build command for %s:%s: %w", opts.ProviderID, opts.ProfileID, err)
		}
		env = launchenv.Merge(env, opts.Environment, opts.PathPrepend)
	}
	if captureShell {
		traceCaptureShellLaunchStage("command-built")
	}
	if opts.Workspace != "" && !customCommand {
		info, statErr := os.Stat(opts.Workspace)
		if statErr != nil {
			return nil, fmt.Errorf("workspace directory %q does not exist: %w", opts.Workspace, statErr)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("workspace path %q is not a directory", opts.Workspace)
		}
	}

	resolved, err := runtime.ResolveCommand(bin)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve executable for %s:%s: %w", opts.ProviderID, opts.ProfileID, err)
	}
	bin = resolved.LauncherPath
	extraArgs = append(append([]string(nil), resolved.PrefixArgs...), extraArgs...)
	if captureShell {
		traceCaptureShellLaunchStage("command-resolved")
	}

	if captureShell {
		traceCaptureShellLaunchStage("runtime-metadata-start")
	}
	title := opts.Title
	if strings.TrimSpace(title) == "" {
		title = fmt.Sprintf("%s (%s)", strings.ToUpper(opts.ProviderID), opts.ProfileID)
	}
	hostBinaryPath, _ := os.Executable()
	hostBinaryFingerprint, _ := registry.BinaryFingerprint(hostBinaryPath)
	providerBinaryFingerprint, _ := registry.BinaryFingerprint(bin)
	if captureShell {
		traceCaptureShellLaunchStage("runtime-metadata-finished")
	}

	sessionLabels := cloneLabels(opts.Labels)
	if customCommand {
		if sessionLabels == nil {
			sessionLabels = map[string]string{}
		}
		sessionLabels["nexus.execution_adapter"] = "command_template"
		if targetKind, ok := opts.Options["target_kind"].(string); ok && strings.TrimSpace(targetKind) != "" {
			sessionLabels["nexus.target_kind"] = strings.ToLower(strings.TrimSpace(targetKind))
		} else {
			sessionLabels["nexus.target_kind"] = "custom"
		}
	}

	// Detached hosts consume arguments through a one-time private envelope;
	// standalone hosts can keep them in memory without registry persistence.
	registryArgs := append([]string(nil), extraArgs...)
	if !opts.Standalone {
		registryArgs = nil
	}
	sess := registry.RuntimeSession{
		RuntimeID:                 opts.RuntimeID,
		AgentID:                   opts.AgentID,
		ProjectID:                 opts.ProjectID,
		ProjectName:               opts.ProjectName,
		Title:                     title,
		ProviderID:                opts.ProviderID,
		ProfileID:                 opts.ProfileID,
		AccountScope:              opts.AccountScope,
		ProviderSessionID:         opts.ProviderSessionID,
		Model:                     launchModel,
		Workspace:                 opts.Workspace,
		Binary:                    bin,
		HostBinaryPath:            hostBinaryPath,
		HostBinaryFingerprint:     hostBinaryFingerprint,
		ProviderBinaryPath:        bin,
		ProviderBinaryFingerprint: providerBinaryFingerprint,
		Args:                      registryArgs,
		Env:                       env,
		State:                     registry.StateStarting,
		ControlLevel:              d.EffectiveCaps(ctx, prof).ControlLevel,
		ControlEndpoint:           protocol.EndpointPath(opts.RuntimeID),
		StartedAt:                 time.Now(),
		MachineID:                 registry.LocalMachineID(),
		Location:                  "local",
		Transport:                 "ipc",
		Labels:                    sessionLabels,
	}

	if captureShell {
		traceCaptureShellLaunchStage("session-register-start")
	}
	if err := l.reg.Register(sess); err != nil {
		return nil, fmt.Errorf("failed to register runtime session: %w", err)
	}
	if captureShell {
		traceCaptureShellLaunchStage("session-register-finished")
		traceCaptureShellLaunchStage("session-registered")
	}
	var supervisedHost *host.SessionHost
	var detachedProcess *os.Process
	cleanupFailedLaunch := func() {
		if captureShell {
			traceCaptureShellLaunchStage("cleanup-start")
			traceCaptureShellLaunchStage("cleanup-envelope-remove-start")
		}
		removeLaunchEnvelope(opts.RuntimeID)
		if captureShell {
			traceCaptureShellLaunchStage("cleanup-envelope-remove-finished")
		}
		if supervisedHost != nil {
			if captureShell {
				traceCaptureShellLaunchStage("cleanup-supervised-terminate-start")
			}
			_ = supervisedHost.Terminate()
			if captureShell {
				traceCaptureShellLaunchStage("cleanup-supervised-terminate-finished")
			}
		}
		if detachedProcess != nil {
			// Ask the host to terminate its provider first when its endpoint is
			// already reachable; killing only the detached host can orphan the
			// provider process on some platforms.
			cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			if captureShell {
				traceCaptureShellLaunchStage("cleanup-protocol-dial-start")
			}
			client, clientErr := protocol.NewClient(opts.RuntimeID)
			if captureShell {
				if clientErr == nil {
					traceCaptureShellLaunchStage("cleanup-protocol-dial-connected")
				} else {
					traceCaptureShellLaunchStage("cleanup-protocol-dial-failed")
				}
			}
			if clientErr == nil {
				if captureShell {
					traceCaptureShellLaunchStage("cleanup-protocol-terminate-start")
				}
				_, _ = client.SendContext(cleanupCtx, protocol.CmdTerminate, nil)
				if captureShell {
					traceCaptureShellLaunchStage("cleanup-protocol-terminate-finished")
					traceCaptureShellLaunchStage("cleanup-protocol-close-start")
				}
				_ = client.Close()
				if captureShell {
					traceCaptureShellLaunchStage("cleanup-protocol-close-finished")
				}
			}
			cancel()
			if captureShell {
				traceCaptureShellLaunchStage("cleanup-protocol-finished")
				traceCaptureShellLaunchStage("cleanup-host-kill-start")
			}
			_ = detachedProcess.Kill()
			if captureShell {
				traceCaptureShellLaunchStage("cleanup-host-kill-finished")
				traceCaptureShellLaunchStage("cleanup-host-wait-start")
			}
			_, _ = detachedProcess.Wait()
			if captureShell {
				traceCaptureShellLaunchStage("cleanup-host-wait-finished")
			}
		}
		if captureShell {
			traceCaptureShellLaunchStage("cleanup-endpoint-remove-start")
		}
		_ = os.Remove(protocol.EndpointPath(opts.RuntimeID))
		if captureShell {
			traceCaptureShellLaunchStage("cleanup-endpoint-remove-finished")
			traceCaptureShellLaunchStage("cleanup-registry-delete-start")
		}
		_ = l.reg.Delete(opts.RuntimeID)
		if captureShell {
			traceCaptureShellLaunchStage("cleanup-registry-delete-finished")
		}
	}

	if opts.Standalone {
		sh, err := host.NewSessionHost(host.Config{
			Session:     sess,
			Registry:    l.reg,
			Binary:      bin,
			Args:        extraArgs,
			Env:         env,
			Cwd:         opts.Workspace,
			InitialRows: 24,
			InitialCols: 80,
		})
		if err != nil {
			cleanupFailedLaunch()
			return nil, fmt.Errorf("failed to initialize SessionHost: %w", err)
		}
		supervisedHost = sh
		if err := sh.Start(); err != nil {
			cleanupFailedLaunch()
			return nil, fmt.Errorf("failed to start SessionHost: %w", err)
		}
	} else {
		if err := createLaunchEnvelope(opts.RuntimeID, extraArgs, customCommand); err != nil {
			_ = l.reg.UpdateState(opts.RuntimeID, registry.StateFailed)
			return nil, fmt.Errorf("failed to create private launch envelope: %w", err)
		}
		selfExe, err := os.Executable()
		if err != nil {
			selfExe = "ai"
		}
		detachedProcess, err = SpawnDetachedHost(selfExe, opts.RuntimeID)
		if err != nil {
			cleanupFailedLaunch()
			return nil, fmt.Errorf("failed to spawn detached SessionHost daemon: %w", err)
		}
		sess.HostPID = detachedProcess.Pid
		_ = l.reg.Register(sess)
		if captureShell {
			traceCaptureShellLaunchStage("host-spawned")
		}
	}

	// Handshake: wait for IPC endpoint to be active and responsive.
	// Detached hosts that fail during Start (e.g. Codex TUI lease busy) exit
	// immediately; fail fast from the host log instead of waiting the full timeout.
	if captureShell {
		traceCaptureShellLaunchStage("handshake-start")
	}
	status, err := waitForRuntimeHandshake(ctx, opts.RuntimeID, detachedProcess, opts.Timeout)
	if err != nil {
		if captureShell {
			traceCaptureShellLaunchStage("handshake-failed")
		}
		fault := registry.StartupFaultIPCTimeout
		if isEarlyHostExit(err) {
			fault = registry.StartupFaultProcessSupervision
		}
		_ = l.reg.UpdateStartupStage(opts.RuntimeID, registry.StartupProviderStarting, fault)
		cleanupFailedLaunch()
		if captureShell {
			traceCaptureShellLaunchStage("handshake-cleanup-finished")
		}
		return nil, fmt.Errorf("runtime handshake failed for %s: %w", opts.RuntimeID, err)
	}

	if captureShell {
		traceCaptureShellLaunchStage("handshake-ready")
		traceCaptureShellLaunchStage("status-ready")
	}

	// SessionHost assigns the authoritative process identity during Start and
	// exposes it through the status handshake. Copy it back before the final
	// registry write; otherwise this launcher-owned snapshot can replace the
	// host's non-zero generation with the zero value from launch time.
	sess.PID = status.PID
	sess.ProviderGeneration = status.ProviderGeneration
	sess.HostPID = status.HostPID
	sess.HostGeneration = status.HostGeneration
	if status.HostBinaryPath != "" {
		sess.HostBinaryPath = status.HostBinaryPath
	}
	if status.HostBinaryFingerprint != "" {
		sess.HostBinaryFingerprint = status.HostBinaryFingerprint
	}
	if status.ProviderBinaryPath != "" {
		sess.ProviderBinaryPath = status.ProviderBinaryPath
	}
	if status.ProviderBinaryFingerprint != "" {
		sess.ProviderBinaryFingerprint = status.ProviderBinaryFingerprint
	}
	sess.State = registry.StateRunning
	_ = l.reg.Register(sess)
	if captureShell {
		traceCaptureShellLaunchStage("status-ready")
	}

	return &sess, nil
}

// Stop requests a graceful protocol stop, then force-kills only process
// identities that can be proven to belong to this runtime generation.
func (l *Launcher) Stop(runtimeID string) error {
	sess, ok := l.reg.Get(runtimeID)
	var protocolErr error
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		protocolErr = err
	} else {
		protocolErr = client.Stop()
		_ = client.Close()
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !sessionProcessAlive(sess, ok) {
			if protocolErr != nil && ok {
				_ = os.Remove(protocol.EndpointPath(runtimeID))
				_ = l.reg.UpdateState(runtimeID, registry.StateStale)
			}
			return protocolErr
		}
		time.Sleep(50 * time.Millisecond)
		sess, ok = l.reg.Get(runtimeID)
	}

	killErr := forceKillSession(runtimeID, sess, ok)
	_ = l.reg.UpdateState(runtimeID, registry.StateFailed)
	if killErr != nil {
		if protocolErr != nil {
			return fmt.Errorf("runtime %s stop failed (%v) and force-stop was rejected: %w", runtimeID, protocolErr, killErr)
		}
		return fmt.Errorf("runtime %s force-stop was rejected: %w", runtimeID, killErr)
	}
	if protocolErr != nil {
		return fmt.Errorf("force-stopped runtime %s after stop failure: %w", runtimeID, protocolErr)
	}
	return fmt.Errorf("force-stopped runtime %s after stop timeout", runtimeID)
}

func validatedRuntimeProcessPID(pid int, generation int64, label string) (int, error) {
	currentPID := os.Getpid()
	if pid <= 0 || pid == currentPID {
		return 0, nil
	}
	if generation <= 0 {
		if registry.IsProcessAlive(pid) {
			return 0, fmt.Errorf("%s PID %d is live but has no generation token", label, pid)
		}
		return 0, nil
	}
	if registry.IsProcessAliveWithGeneration(pid, generation) {
		return pid, nil
	}
	if registry.IsProcessAlive(pid) {
		return 0, fmt.Errorf("%s PID %d is live but does not match the recorded generation", label, pid)
	}
	return 0, nil
}

func forceKillSession(runtimeID string, sess registry.RuntimeSession, ok bool) error {
	if !ok {
		_ = os.Remove(protocol.EndpointPath(runtimeID))
		return nil
	}

	providerPID, providerErr := validatedRuntimeProcessPID(sess.PID, sess.ProviderGeneration, "provider")
	hostPID, hostErr := validatedRuntimeProcessPID(sess.HostPID, sess.HostGeneration, "SessionHost")
	if providerErr != nil || hostErr != nil {
		if providerErr != nil && hostErr != nil {
			return fmt.Errorf("%v; %v", providerErr, hostErr)
		}
		if providerErr != nil {
			return providerErr
		}
		return hostErr
	}

	// Kill the provider first while both identities are still verified. The
	// SessionHost is then terminated; on Windows its Job Object is an additional
	// child-cleanup barrier.
	if providerPID > 0 {
		KillProcess(providerPID)
	}
	if hostPID > 0 {
		KillProcess(hostPID)
	}
	_ = os.Remove(protocol.EndpointPath(runtimeID))
	return nil
}

func sessionProcessAlive(sess registry.RuntimeSession, ok bool) bool {
	if !ok {
		return false
	}
	currentPID := os.Getpid()
	// Raw liveness is sufficient for the wait barrier only. Destructive actions
	// are guarded by validatedRuntimeProcessPID before any signal is sent.
	if sess.PID > 0 && sess.PID != currentPID && registry.IsProcessAlive(sess.PID) {
		return true
	}
	if sess.HostPID > 0 && sess.HostPID != currentPID && registry.IsProcessAlive(sess.HostPID) {
		return true
	}
	return false
}

func cloneLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	out := make(map[string]string, len(labels))
	for key, value := range labels {
		out[key] = value
	}
	return out
}
