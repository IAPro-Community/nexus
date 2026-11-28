package codex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	sdkproviders "github.com/IAPro-Community/IAPro-SDK/providers"
	sdkcodex "github.com/IAPro-Community/IAPro-SDK/providers/codex"
	sdkparser "github.com/IAPro-Community/IAPro-SDK/providers/parsers/codex"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/executil"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

// appServerTimeout bounds the whole quota probe: spawn, handshake and read.
const appServerTimeout = 8 * time.Second

// appServerRateLimitsMethod is the official quota read method. It belongs to the
// normal protocol surface and does not require experimental capabilities.
const appServerRateLimitsMethod = "account/rateLimits/read"

// fetchUsageFromAppServer reads official quota through the Codex app-server for
// a single profile. It never sends a prompt, never opens a thread and never
// consumes quota.
//
// The probe is deliberately quota-only: it performs the mandatory handshake,
// issues account/rateLimits/read, validates that the server bound itself to the
// profile's isolated CODEX_HOME and that the returned account matches the
// profile's authenticated identity, then terminates the subprocess.
func (a *Adapter) fetchUsageFromAppServer(ctx context.Context, p model.Profile) (model.UsageSnapshot, bool) {
	if !appServerEnabled() {
		return model.UsageSnapshot{}, false
	}
	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil || strings.TrimSpace(home) == "" {
		return model.UsageSnapshot{}, false
	}

	info := a.InspectAuth(ctx, p)
	if !info.Authenticated {
		return model.UsageSnapshot{}, false
	}
	// A verifiable chatgpt_account_id is required: without it the response could
	// not be checked against this profile's identity, and quota must never be
	// attributed on trust alone.
	expected := strings.TrimSpace(info.ExternalAccountID)
	if expected == "" {
		return model.UsageSnapshot{}, false
	}

	localScope, err := readProfileScope(home, p.Name)
	if err != nil || localScope.AccountID != expected {
		return model.UsageSnapshot{}, false
	}
	snap, err := runAppServerRateLimits(ctx, home, localScope)
	if err != nil {
		var diagnostic *sdkproviders.ProbeError
		if errors.As(err, &diagnostic) && sdkproviders.ClassifyProbeError(err) == sdkproviders.ProbeAuthExpired {
			return model.UsageSnapshot{
				ProviderID:     string(a.ID()),
				ProfileID:      p.Name,
				Account:        expected,
				AccountScope:   localScope,
				Status:         model.UsageError,
				Source:         model.SourceOfficialAPI,
				Confidence:     model.UsageConfidenceHigh,
				FetchedAt:      time.Now(),
				DiagnosticCode: "AUTH_REQUIRED",
				Error:          "sessão expirada — faça login novamente",
			}, true
		}
		slog.Debug("codex app-server quota unavailable", "provider", string(a.ID()), "profile", p.Name)
		return model.UsageSnapshot{}, false
	}
	if len(snap.Windows) == 0 {
		return model.UsageSnapshot{}, false
	}
	// The SDK verifies exact local credential generation and remote account.
	// Nexus still binds that observation to its durable host account scope.
	snap.Account = expected
	if snap.Plan != "" && snap.Plan != "unknown" {
		snap.Plan = "ChatGPT " + titlePlan(snap.Plan)
	} else {
		snap.Plan = ""
	}
	return snap, true
}

// readProfileScope reads only the existing host-authorized profile. The SDK
// owns the attribution and generation algorithm; the host owns file access.
func readProfileScope(home, profile string) (model.AccountScope, error) {
	raw, err := readQuotaAuth(home)
	if err != nil {
		return model.AccountScope{}, err
	}
	defer clear(raw)
	return sdkcodex.ScopeFromAuth(profile, raw)
}

func readQuotaAuth(home string) ([]byte, error) {
	for _, path := range []string{filepath.Join(home, "auth.json"), filepath.Join(home, ".codex", "auth.json")} {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(f, 4*1024*1024+1))
		_ = f.Close()
		if err == nil && len(raw) > 0 && len(raw) <= 4*1024*1024 {
			return raw, nil
		}
		clear(raw)
	}
	return nil, sdkproviders.ErrIdentityMismatch
}

// appServerCommand builds the subprocess used for the quota probe. It is a
// package variable so protocol tests can substitute a fake server without
// introducing an env-driven binary override in production code.
var appServerCommand = func(ctx context.Context, codexHome string) (*exec.Cmd, error) {
	bin, err := spec.MustGet("codex").ResolveBinary()
	if err != nil {
		return nil, fmt.Errorf("codex binary not found")
	}
	// stdio transport only. The unix/ws transports are shared daemons whose
	// CODEX_HOME may belong to another profile.
	cmd := exec.CommandContext(ctx, bin, "app-server", "--stdio")
	executil.HideConsoleWindow(cmd)
	cmd.Env = appServerEnv(bin, codexHome)
	cmd.Dir = codexHome
	return cmd, nil
}

// appServerEnabled reports whether the official probe may run. The kill switch
// exists so an environment with a broken or absent app-server can fall back to
// rollout evidence without an 8s penalty per read.
func appServerEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("NEXUS_CODEX_APP_SERVER"))) {
	case "0", "false", "off", "no":
		return false
	}
	return true
}

// runAppServerRateLimits delegates the JSON-RPC transport, handshake and
// quota parsing to the SDK. Nexus retains its existing host isolation policy.
func runAppServerRateLimits(ctx context.Context, profileHome string, scope model.AccountScope) (model.UsageSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, appServerTimeout)
	defer cancel()

	probeHome, cleanup, err := prepareEphemeralProbeHome(profileHome)
	if err != nil {
		return model.UsageSnapshot{}, err
	}
	defer cleanup()
	probeScope, err := readProfileScope(probeHome, scope.ProfileID)
	if err != nil || probeScope != scope {
		return model.UsageSnapshot{}, sdkproviders.ErrIdentityMismatch
	}
	rpc, err := sdkcodex.StartStdioRPC(ctx, func(runCtx context.Context) (*exec.Cmd, error) {
		return appServerCommand(runCtx, probeHome)
	})
	if err != nil {
		return model.UsageSnapshot{}, err
	}
	snap, err := sdkparser.Probe(ctx, rpc, scope, scope.AccountID, probeHome, config.FilesystemPathsEquivalent, time.Now)
	current, identityErr := readProfileScope(profileHome, scope.ProfileID)
	if identityErr != nil || current != scope {
		return model.UsageSnapshot{}, sdkproviders.ErrIdentityMismatch
	}
	return snap, err
}

// appServerEnv builds the isolated environment. Every path the app-server may
// write to is confined to the profile home, including the local thread SQLite
// it may create or migrate on startup.
func appServerEnv(bin, codexHome string) []string {
	return runtime.EnvSet(os.Environ(), runtime.ApplyProfileIsolationEnv(codexHome, map[string]string{
		"HOME":             codexHome,
		"CODEX_HOME":       codexHome,
		"CODEX_CONFIG_DIR": codexHome,
		"XDG_CONFIG_HOME":  filepath.Join(codexHome, ".config"),
		"XDG_CACHE_HOME":   filepath.Join(codexHome, ".cache"),
		"XDG_DATA_HOME":    filepath.Join(codexHome, ".local", "share"),
		"XDG_STATE_HOME":   filepath.Join(codexHome, ".local", "state"),
		"PATH":             runtime.EnhancedPATH(filepath.Dir(bin)),
		"NO_COLOR":         "1",
	}), "OPENAI_API_KEY")
}

// prepareEphemeralProbeHome provisions an isolated directory seeded read-only
// with the profile's auth.json so app-server never touches a live TUI's stores.
func prepareEphemeralProbeHome(profileHome string) (string, func(), error) {
	trimmed := strings.TrimSpace(profileHome)
	if trimmed == "" {
		return "", nil, fmt.Errorf("empty CODEX_HOME for probe")
	}
	authBytes, err := readQuotaAuth(trimmed)
	if err != nil {
		return "", nil, err
	}
	defer clear(authBytes)
	parent := filepath.Dir(trimmed)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return "", nil, err
	}
	probeHome, err := os.MkdirTemp(parent, ".quota-probe-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(probeHome)
	}
	if err := os.Chmod(probeHome, 0o700); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := os.WriteFile(filepath.Join(probeHome, "auth.json"), authBytes, 0o600); err != nil {
		cleanup()
		return "", nil, err
	}
	return probeHome, cleanup, nil
}
