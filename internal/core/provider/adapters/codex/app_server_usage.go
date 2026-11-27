package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/executil"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

// appServerTimeout bounds the whole quota probe: spawn, handshake and read.
const appServerTimeout = 8 * time.Second

// appServerClientName identifies Nexus to the Codex app-server.
const appServerClientName = "nexus_workspace_os"

// appServerBucketID is the metered bucket used for Codex CLI usage.
const appServerBucketID = "codex"

// appServerRateLimitsMethod is the official quota read method. It belongs to the
// normal protocol surface and does not require experimental capabilities.
const appServerRateLimitsMethod = "account/rateLimits/read"

// rateLimitWindowRPC mirrors the official camelCase RateLimitWindow.
type rateLimitWindowRPC struct {
	UsedPercent        float64 `json:"usedPercent"`
	WindowDurationMins *int    `json:"windowDurationMins"`
	ResetsAt           *int64  `json:"resetsAt"`
}

// creditsSnapshotRPC mirrors the official CreditsSnapshot.
type creditsSnapshotRPC struct {
	HasCredits bool    `json:"hasCredits"`
	Unlimited  bool    `json:"unlimited"`
	Balance    *string `json:"balance"`
}

// rateLimitSnapshotRPC mirrors the official RateLimitSnapshot.
type rateLimitSnapshotRPC struct {
	LimitID              *string             `json:"limitId"`
	LimitName            *string             `json:"limitName"`
	NormalModelSlug      *string             `json:"normalModelSlug"`
	Primary              *rateLimitWindowRPC `json:"primary"`
	Secondary            *rateLimitWindowRPC `json:"secondary"`
	Credits              *creditsSnapshotRPC `json:"credits"`
	SpendControlReached  *bool               `json:"spendControlReached"`
	PlanType             *string             `json:"planType"`
	RateLimitReachedType *string             `json:"rateLimitReachedType"`
}

// getAccountRateLimitsResponse mirrors the official response envelope.
type getAccountRateLimitsResponse struct {
	OrdinaryUsageAllowed *bool                           `json:"ordinaryUsageAllowed"`
	RateLimits           rateLimitSnapshotRPC            `json:"rateLimits"`
	RateLimitsByLimitID  map[string]rateLimitSnapshotRPC `json:"rateLimitsByLimitId"`
	AccountID            *string                         `json:"accountId"`
}

// initializeResult mirrors the fields of InitializeResponse that Nexus verifies.
type initializeResult struct {
	CodexHome string `json:"codexHome"`
}

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

	resp, err := runAppServerRateLimits(ctx, home)
	if err != nil {
		// Only report coarse error categories. RPC errors may contain server
		// detail that must not be copied into logs, even at debug level.
		if errors.Is(err, errAppServerAuthRequired) {
			slog.Debug("codex app-server quota rejected: auth expired", "provider", string(a.ID()), "profile", p.Name, "reason", "auth_required")
			return model.UsageSnapshot{
				ProviderID:     string(a.ID()),
				ProfileID:      p.Name,
				Account:        expected,
				Status:         model.UsageError,
				Source:         model.SourceOfficialAPI,
				Confidence:     model.UsageConfidenceHigh,
				FetchedAt:      time.Now(),
				DiagnosticCode: "AUTH_REQUIRED",
				Error:          "sessão expirada — faça login novamente",
			}, true
		}
		reason := "probe_failure"
		if errors.Is(err, context.DeadlineExceeded) {
			reason = "timeout"
		} else if errors.Is(err, context.Canceled) {
			reason = "canceled"
		}
		slog.Debug("codex app-server quota unavailable", "provider", string(a.ID()), "profile", p.Name, "reason", reason)
		return model.UsageSnapshot{}, false
	}

	// Identity gate: an app-server bound to this CODEX_HOME must report this
	// profile's account. A missing or mismatched accountId means attribution
	// would rely on trust alone and is refused.
	if resp.AccountID == nil || strings.TrimSpace(*resp.AccountID) == "" {
		slog.Debug("codex app-server quota rejected: account identity missing", "profile", p.Name)
		return model.UsageSnapshot{}, false
	}
	got := strings.TrimSpace(*resp.AccountID)
	if !strings.EqualFold(got, expected) {
		// Never include either account identifier in the log line.
		slog.Debug("codex app-server quota rejected: foreign account", "profile", p.Name)
		return model.UsageSnapshot{}, false
	}

	bucket, ok := selectRateLimitBucket(*resp)
	if !ok {
		return model.UsageSnapshot{}, false
	}
	windows := buildWindowsFromRPC(bucket)
	if len(windows) == 0 {
		return model.UsageSnapshot{}, false
	}

	snap := model.UsageSnapshot{
		ProviderID: string(a.ID()),
		ProfileID:  p.Name,
		Account:    expected,
		Status:     model.UsageLive,
		Source:     model.SourceOfficialAPI,
		Confidence: model.UsageConfidenceHigh,
		FetchedAt:  time.Now(),
		Windows:    windows,
		Plan:       planFromRPC(bucket.PlanType),
	}
	if bucket.NormalModelSlug != nil && strings.TrimSpace(*bucket.NormalModelSlug) != "" {
		snap.ModelName = strings.TrimSpace(*bucket.NormalModelSlug)
	}
	if blocked, reason := blockedByBackend(*resp, bucket); blocked {
		snap.Status = model.UsageRateLimited
		snap.Error = reason
	}
	return snap, true
}

// selectRateLimitBucket prefers the metered "codex" bucket and falls back to the
// backward-compatible single-bucket view.
func selectRateLimitBucket(resp getAccountRateLimitsResponse) (rateLimitSnapshotRPC, bool) {
	if b, ok := resp.RateLimitsByLimitID[appServerBucketID]; ok && b.Primary != nil {
		return b, true
	}
	if resp.RateLimits.Primary != nil {
		return resp.RateLimits, true
	}
	// Some plans expose only a weekly window; accept it rather than reporting
	// no data at all.
	if b, ok := resp.RateLimitsByLimitID[appServerBucketID]; ok && b.Secondary != nil {
		return b, true
	}
	if resp.RateLimits.Secondary != nil {
		return resp.RateLimits, true
	}
	return rateLimitSnapshotRPC{}, false
}

// blockedByBackend reports a backend-declared block. Percentages and reset times
// must never be used to infer recovery, so these signals win over any window
// that looks positive.
func blockedByBackend(resp getAccountRateLimitsResponse, bucket rateLimitSnapshotRPC) (bool, string) {
	if resp.OrdinaryUsageAllowed != nil && !*resp.OrdinaryUsageAllowed {
		return true, "provedor bloqueou uso incluso (ordinaryUsageAllowed=false)"
	}
	if bucket.SpendControlReached != nil && *bucket.SpendControlReached {
		return true, "limite de gasto atingido"
	}
	if bucket.RateLimitReachedType != nil && strings.TrimSpace(*bucket.RateLimitReachedType) != "" {
		return true, "limite atingido: " + strings.TrimSpace(*bucket.RateLimitReachedType)
	}
	return false, ""
}

func planFromRPC(planType *string) string {
	if planType == nil {
		return ""
	}
	v := strings.TrimSpace(*planType)
	if v == "" || v == "unknown" {
		return ""
	}
	if v == "pro" {
		return "ChatGPT Pro"
	}
	return "ChatGPT " + titlePlan(v)
}

// buildWindowsFromRPC maps official camelCase windows onto the Nexus model,
// reusing the same kind/group semantics as the rollout fallback so consumers do
// not need to know which source produced the snapshot.
func buildWindowsFromRPC(bucket rateLimitSnapshotRPC) []model.UsageWindow {
	limits := rolloutRateLimits{}
	if bucket.Primary != nil {
		limits.Primary = toRolloutWindow(*bucket.Primary)
	}
	if bucket.Secondary != nil {
		limits.Secondary = toRolloutWindow(*bucket.Secondary)
	}
	return buildWindowsFromRateLimits(limits)
}

func toRolloutWindow(w rateLimitWindowRPC) *rateLimitWindow {
	out := &rateLimitWindow{UsedPercent: w.UsedPercent}
	if w.WindowDurationMins != nil {
		out.WindowMinutes = *w.WindowDurationMins
	}
	if w.ResetsAt != nil {
		out.ResetsAt = *w.ResetsAt
	}
	return out
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

const (
	appServerMaxRPCLine  = 4 * 1024 * 1024
	appServerMaxMessages = 512
)

var (
	errAppServerAuthRequired = errors.New("codex app-server authentication required")
	errAppServerProtocol     = errors.New("codex rpc protocol error")
)

type codexRPCResponse struct {
	ID     *uint64         `json:"id"`
	Method string          `json:"method"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

type codexStdioRPC struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	in     io.WriteCloser
	out    io.ReadCloser
	reader *bufio.Reader
	mu     sync.Mutex
	once   sync.Once
	nextID uint64
}

func startCodexStdioRPC(ctx context.Context, probeHome string) (*codexStdioRPC, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	runCtx, cancel := context.WithCancel(ctx)
	cmd, err := appServerCommand(runCtx, probeHome)
	if err != nil || cmd == nil || cmd.Process != nil {
		cancel()
		if err != nil {
			return nil, err
		}
		return nil, errAppServerProtocol
	}
	cmd.Stderr = io.Discard
	cmd.WaitDelay = 2 * time.Second
	in, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		_ = in.Close()
		cancel()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		_ = in.Close()
		_ = out.Close()
		cancel()
		return nil, err
	}
	return &codexStdioRPC{
		cmd:    cmd,
		cancel: cancel,
		in:     in,
		out:    out,
		reader: bufio.NewReaderSize(out, appServerMaxRPCLine),
	}, nil
}

func (r *codexStdioRPC) Close() error {
	r.once.Do(func() {
		r.cancel()
		_ = r.in.Close()
		_ = r.out.Close()
		_ = r.cmd.Wait()
	})
	return nil
}

func (r *codexStdioRPC) write(method string, params any, id uint64) error {
	payload := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		payload["params"] = params
	}
	if id != 0 {
		payload["id"] = id
	}
	data, err := json.Marshal(payload)
	if err != nil || len(data) >= appServerMaxRPCLine {
		return errAppServerProtocol
	}
	if _, err := r.in.Write(append(data, '\n')); err != nil {
		return errAppServerProtocol
	}
	return nil
}

func (r *codexStdioRPC) exchange(ctx context.Context, method string, params any, notify bool) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	type rpcOutcome struct {
		raw json.RawMessage
		err error
	}
	result := make(chan rpcOutcome, 1)
	go func() {
		id := uint64(0)
		if !notify {
			r.nextID++
			id = r.nextID
		}
		if err := r.write(method, params, id); err != nil {
			result <- rpcOutcome{err: err}
			return
		}
		if notify {
			result <- rpcOutcome{}
			return
		}
		for i := 0; i < appServerMaxMessages; i++ {
			line, err := r.reader.ReadSlice('\n')
			if err != nil {
				result <- rpcOutcome{err: errAppServerProtocol}
				return
			}
			var response codexRPCResponse
			if json.Unmarshal(line, &response) != nil {
				result <- rpcOutcome{err: errAppServerProtocol}
				return
			}
			if response.Method != "" && response.ID != nil {
				result <- rpcOutcome{err: errors.New("unexpected app-server request")}
				return
			}
			if response.ID == nil || *response.ID != id {
				continue
			}
			if len(response.Error) > 0 && string(response.Error) != "null" {
				if isAppServerAuthErrorMessage(string(response.Error)) {
					result <- rpcOutcome{err: errAppServerAuthRequired}
					return
				}
				result <- rpcOutcome{err: errAppServerProtocol}
				return
			}
			if len(response.Result) == 0 {
				result <- rpcOutcome{err: errAppServerProtocol}
				return
			}
			result <- rpcOutcome{raw: response.Result}
			return
		}
		result <- rpcOutcome{err: errAppServerProtocol}
	}()
	select {
	case <-ctx.Done():
		_ = r.Close()
		<-result
		return nil, ctx.Err()
	case out := <-result:
		if out.err != nil {
			_ = r.Close()
		}
		return out.raw, out.err
	}
}

func (r *codexStdioRPC) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	return r.exchange(ctx, method, params, false)
}

func (r *codexStdioRPC) Notify(ctx context.Context, method string, params any) error {
	_, err := r.exchange(ctx, method, params, true)
	return err
}

// runAppServerRateLimits performs the full stdio JSON-RPC exchange against a
// transient `codex app-server` subprocess in an isolated ephemeral CODEX_HOME
// seeded read-only with the profile's auth.json so live TUIs never block reads.
func runAppServerRateLimits(ctx context.Context, profileHome string) (*getAccountRateLimitsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, appServerTimeout)
	defer cancel()

	probeHome, cleanup, err := prepareEphemeralProbeHome(profileHome)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	rpc, err := startCodexStdioRPC(ctx, probeHome)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rpc.Close() }()

	initRaw, err := rpc.Call(ctx, "initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    appServerClientName,
			"version": appServerClientVersion(),
		},
	})
	if err != nil {
		return nil, err
	}
	var initRes initializeResult
	if err := json.Unmarshal(initRaw, &initRes); err != nil {
		return nil, fmt.Errorf("app-server initialize decode: %w", err)
	}
	if strings.TrimSpace(initRes.CodexHome) == "" {
		return nil, errors.New("app-server initialize omitted codexHome")
	}
	// Isolation gate: refuse a server that resolved a different CODEX_HOME.
	if !config.FilesystemPathsEquivalent(initRes.CodexHome, probeHome) {
		return nil, fmt.Errorf("app-server bound to unexpected codex home")
	}

	// The handshake is only complete after the `initialized` notification.
	if err := rpc.Notify(ctx, "initialized", nil); err != nil {
		return nil, err
	}
	raw, err := rpc.Call(ctx, appServerRateLimitsMethod, map[string]any{
		"excludeResetCreditDetails": true,
	})
	if err != nil {
		return nil, err
	}
	var resp getAccountRateLimitsResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
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

func appServerClientVersion() string {
	if v := strings.TrimSpace(os.Getenv("NEXUS_VERSION")); v != "" {
		return v
	}
	return "1.0.0"
}

func isAppServerAuthErrorMessage(raw string) bool {
	msg := strings.ToLower(raw)
	return strings.Contains(msg, "invalid_jwt") ||
		strings.Contains(msg, "401") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "token_expired") ||
		strings.Contains(msg, "signing in again") ||
		strings.Contains(msg, "authentication token")
}

// prepareEphemeralProbeHome provisions an isolated directory seeded read-only
// with the profile's auth.json so app-server never touches a live TUI's stores.
func prepareEphemeralProbeHome(profileHome string) (string, func(), error) {
	trimmed := strings.TrimSpace(profileHome)
	if trimmed == "" {
		return "", nil, fmt.Errorf("empty CODEX_HOME for probe")
	}
	var authBytes []byte
	var readErr error
	for _, candidate := range []string{
		filepath.Join(trimmed, "auth.json"),
		filepath.Join(trimmed, ".codex", "auth.json"),
	} {
		data, err := os.ReadFile(candidate)
		if err == nil && len(data) > 0 {
			authBytes = data
			readErr = nil
			break
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			readErr = err
		}
	}
	if len(authBytes) == 0 {
		if readErr != nil {
			return "", nil, readErr
		}
		return "", nil, fmt.Errorf("codex auth.json not found in %s", trimmed)
	}
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
