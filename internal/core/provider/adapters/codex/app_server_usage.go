package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

// appServerTimeout bounds the whole quota probe: spawn, handshake and read.
const appServerTimeout = 8 * time.Second

// appServerMaxLine bounds a single JSON-RPC message so a malformed or hostile
// stream cannot exhaust memory.
const appServerMaxLine = 4 * 1024 * 1024

// appServerMaxStderr bounds how much diagnostic stderr we retain.
const appServerMaxStderr = 8 * 1024

// appServerMaxMessages bounds how many unrelated notifications we skip while
// waiting for our response.
const appServerMaxMessages = 512

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

type jsonrpcEnvelope struct {
	ID     *json.RawMessage `json:"id"`
	Method string           `json:"method"`
	Result json.RawMessage  `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
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
		return model.UsageSnapshot{}, false
	}

	// Identity gate: an app-server bound to this CODEX_HOME must report this
	// profile's account. A missing or mismatched accountId means attribution
	// would rely on trust alone and is refused.
	if resp.AccountID == nil || strings.TrimSpace(*resp.AccountID) == "" {
		return model.UsageSnapshot{}, false
	}
	got := strings.TrimSpace(*resp.AccountID)
	if !strings.EqualFold(got, expected) {
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

// errTUIBusy is returned when an interactive Codex TUI owns the profile home.
var errTUIBusy = errors.New("codex TUI holds CODEX_HOME lock")

// runAppServerRateLimits performs the full stdio JSON-RPC exchange against a
// transient `codex app-server` subprocess bound to codexHome.
func runAppServerRateLimits(ctx context.Context, codexHome string) (*getAccountRateLimitsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, appServerTimeout)
	defer cancel()

	// Never compete with a live interactive TUI for the same CODEX_HOME.
	probeLock, ok, err := TryAcquireMaintenanceLock(codexHome)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errTUIBusy
	}
	defer func() { _ = probeLock.Release() }()

	cmd, err := appServerCommand(ctx, codexHome)
	if err != nil {
		return nil, err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Always reap the subprocess: a leaked app-server would hold the profile's
	// SQLite locks and keep polling the backend.
	defer func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	// Drain stderr with a hard cap so the pipe never blocks the child. Content
	// is intentionally discarded: it may echo request context.
	go drainCapped(stderr, appServerMaxStderr)

	reader := bufio.NewReaderSize(stdout, 64*1024)

	if err := writeRPC(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"clientInfo": map[string]any{
				"name":    appServerClientName,
				"version": appServerClientVersion(),
			},
		},
	}); err != nil {
		return nil, err
	}

	initRaw, err := awaitResult(ctx, reader, 1)
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
	if !config.FilesystemPathsEquivalent(initRes.CodexHome, codexHome) {
		return nil, fmt.Errorf("app-server bound to unexpected codex home")
	}

	// The handshake is only complete after the `initialized` notification.
	if err := writeRPC(stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "initialized",
	}); err != nil {
		return nil, err
	}

	if err := writeRPC(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  appServerRateLimitsMethod,
		"params": map[string]any{
			"excludeResetCreditDetails": true,
		},
	}); err != nil {
		return nil, err
	}

	raw, err := awaitResult(ctx, reader, 2)
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
	return runtime.EnvSet(os.Environ(), map[string]string{
		"HOME":             codexHome,
		"CODEX_HOME":       codexHome,
		"CODEX_CONFIG_DIR": codexHome,
		"XDG_CONFIG_HOME":  filepath.Join(codexHome, ".config"),
		"XDG_CACHE_HOME":   filepath.Join(codexHome, ".cache"),
		"XDG_DATA_HOME":    filepath.Join(codexHome, ".local", "share"),
		"XDG_STATE_HOME":   filepath.Join(codexHome, ".local", "state"),
		"PATH":             runtime.EnhancedPATH(filepath.Dir(bin)),
		"NO_COLOR":         "1",
	})
}

func appServerClientVersion() string {
	if v := strings.TrimSpace(os.Getenv("NEXUS_VERSION")); v != "" {
		return v
	}
	return "1.0.0"
}

func writeRPC(w io.Writer, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

// awaitResult reads messages until the response matching wantID arrives,
// skipping unrelated notifications and server requests.
func awaitResult(ctx context.Context, reader *bufio.Reader, wantID int64) (json.RawMessage, error) {
	for i := 0; i < appServerMaxMessages; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line, err := readLimitedLine(reader, appServerMaxLine)
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var env jsonrpcEnvelope
		if json.Unmarshal([]byte(line), &env) != nil {
			continue
		}
		// Notifications and server-initiated requests carry a method; only a
		// response to our request has an id without a method.
		if env.Method != "" || env.ID == nil {
			continue
		}
		id, ok := numericRequestID(*env.ID)
		if !ok || id != wantID {
			continue
		}
		if env.Error != nil {
			return nil, fmt.Errorf("app-server rpc error %d", env.Error.Code)
		}
		if len(env.Result) == 0 {
			return nil, errors.New("app-server returned empty result")
		}
		return env.Result, nil
	}
	return nil, errors.New("app-server response not found")
}

func numericRequestID(raw json.RawMessage) (int64, bool) {
	var n int64
	if json.Unmarshal(raw, &n) == nil {
		return n, true
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		var parsed int64
		if _, err := fmt.Sscanf(s, "%d", &parsed); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

// readLimitedLine reads one newline-terminated message, failing instead of
// buffering without bound.
func readLimitedLine(reader *bufio.Reader, limit int) (string, error) {
	var sb strings.Builder
	for {
		chunk, err := reader.ReadString('\n')
		sb.WriteString(chunk)
		if sb.Len() > limit {
			return "", errors.New("app-server message exceeds limit")
		}
		if err != nil {
			if err == io.EOF && sb.Len() > 0 {
				return sb.String(), nil
			}
			return "", err
		}
		if strings.HasSuffix(chunk, "\n") {
			return sb.String(), nil
		}
	}
}

func drainCapped(r io.Reader, limit int) {
	buf := make([]byte, 4096)
	total := 0
	for {
		n, err := r.Read(buf)
		total += n
		if err != nil {
			return
		}
		if total > limit {
			// Keep reading to avoid blocking the child, but stop accounting.
			_, _ = io.Copy(io.Discard, r)
			return
		}
	}
}
