// Command nexus-e2e-local performs a real local Direct Work smoke test against
// a running Nexus instance and an already-authenticated provider/profile.
//
// Usage:
//
//	NEXUS_BOOTSTRAP_URL='http://127.0.0.1:PORT/?token=...' \
//	NEXUS_E2E_PROJECT_PATH=/path/to/git/project \
//	NEXUS_E2E_PROVIDER=claude \
//	NEXUS_E2E_PROFILE=default \
//	go run ./scripts/nexus-e2e-local.go
//
// Set NEXUS_E2E_ROOT to an explicitly provisioned empty non-temporary
// directory when a provider refuses temporary credential homes. The harness
// never removes an explicit root; remove it only after reviewing diagnostics.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/gorilla/websocket"
)

type apiClient struct {
	base string
	csrf string
	http *http.Client
}

func main() {
	start := flag.Bool("start", false, "start a local Nexus process on --port and clean it up")
	port := flag.Int("port", 13000, "Nexus HTTP port when --start is used")
	keep := flag.Bool("keep", false, "keep generated project/data artifacts (diagnostics only)")
	browser := flag.Bool("browser", false, "run the token-safe Chromium smoke after startup")
	safeApply := flag.Bool("safe-apply", false, "verify a real restart-style Safe Apply and terminal reconnect")
	mission := flag.Bool("mission", false, "run an authenticated autonomous Mission smoke instead of Direct Work")
	flag.Parse()

	if *mission && *safeApply {
		fmt.Fprintln(os.Stderr, "NEXUS_E2E_FAIL: -mission and -safe-apply are mutually exclusive")
		os.Exit(1)
	}
	timeout := 2 * time.Minute
	if *mission {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cleanup := func() {}
	if *start {
		var err error
		cleanup, err = startLocalNexus(ctx, *port, *keep, *browser)
		if err != nil {
			fmt.Fprintln(os.Stderr, "NEXUS_E2E_FAIL:", err)
			os.Exit(1)
		}
	}
	if err := run(ctx, *safeApply, *mission); err != nil {
		cleanup()
		if errors.Is(err, errNotAuthenticated) {
			fmt.Fprintln(os.Stderr, "NEXUS_E2E_NOT_AUTHENTICATED:", err)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "NEXUS_E2E_FAIL:", err)
		os.Exit(1)
	}
	cleanup()
	if *mission {
		fmt.Println("NEXUS_E2E_PASS: autonomous Mission reached COMPLETED_VERIFIED with persisted evidence")
		return
	}
	fmt.Println("NEXUS_E2E_PASS: Direct Work reached a real provider runtime and produced terminal output")
}

var errNotAuthenticated = errors.New("provider authentication is unavailable")

const providerAnswerMarker = "NEXUS_E2E_OK"

var ansiEscape = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
var secretAssignment = regexp.MustCompile(`(?i)(OPENAI_API_KEY|ANTHROPIC_API_KEY|GOOGLE_API_KEY|TOKEN|PASSWORD)=\S+`)
var profileCopyExcludedDirs = map[string]bool{
	".cache": true, ".local": true, ".omega": true, "go": true, "node_modules": true,
	"log": true, "logs": true, "shell_snapshots": true, "thread-writer-locks": true,
}

func providerMarkerSeen(transcript string) bool {
	for _, line := range strings.Split(sanitizeTranscript(transcript), "\n") {
		if strings.TrimSpace(line) == providerAnswerMarker {
			return true
		}
	}
	return false
}

func sanitizeTranscript(transcript string) string {
	clean := ansiEscape.ReplaceAllString(transcript, "")
	return secretAssignment.ReplaceAllString(clean, "$1=[REDACTED]")
}

func run(ctx context.Context, safeApply, mission bool) error {
	bootstrap := strings.TrimSpace(os.Getenv("NEXUS_BOOTSTRAP_URL"))
	if bootstrap == "" {
		return fmt.Errorf("NEXUS_BOOTSTRAP_URL is required; use the one-time URL printed by the running Nexus")
	}
	u, err := url.Parse(bootstrap)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid NEXUS_BOOTSTRAP_URL")
	}
	bootstrapToken, err := bootstrapTokenFromURL(u)
	if err != nil {
		return err
	}
	base := u.Scheme + "://" + u.Host
	projectPath := strings.TrimSpace(os.Getenv("NEXUS_E2E_PROJECT_PATH"))
	if mission && projectPath == "" {
		return fmt.Errorf("NEXUS_E2E_PROJECT_PATH is required in -mission mode; use an explicit disposable clean Git repository")
	}
	if projectPath == "" {
		projectPath, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	projectPath, err = filepath.Abs(projectPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(projectPath); err != nil {
		return fmt.Errorf("project path: %w", err)
	}
	if mission {
		if err := validateMissionE2EProject(ctx, projectPath); err != nil {
			return err
		}
	}

	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Jar: jar, Timeout: 30 * time.Second}
	api := &apiClient{base: base, http: hc}
	if err := api.do(ctx, http.MethodPost, "/api/v1/auth/bootstrap", map[string]string{"token": bootstrapToken}, nil); err != nil {
		return fmt.Errorf("bootstrap exchange: %w", err)
	}
	var session struct {
		Authenticated bool   `json:"authenticated"`
		CSRF          string `json:"csrf_token"`
	}
	if err := api.do(ctx, http.MethodGet, "/api/v1/session", nil, &session); err != nil {
		return err
	}
	if !session.Authenticated || session.CSRF == "" {
		return fmt.Errorf("bootstrap did not establish an authenticated session")
	}
	api.csrf = session.CSRF

	stamp := time.Now().UTC().Format("20060102-150405")
	var project struct {
		ID string `json:"id"`
	}
	if err := api.do(ctx, http.MethodPost, "/api/v1/projects", map[string]any{
		"name": "Nexus E2E " + stamp,
		"path": projectPath,
	}, &project); err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	if project.ID == "" {
		return fmt.Errorf("create project returned empty id")
	}

	var resources struct {
		Accounts []struct {
			ID            string `json:"id"`
			Provider      string `json:"provider"`
			Profile       string `json:"profile"`
			Authenticated bool   `json:"authenticated"`
			Available     bool   `json:"available"`
			RateLimited   bool   `json:"rate_limited"`
		} `json:"accounts"`
	}
	if err := api.do(ctx, http.MethodGet, "/api/v1/resources", nil, &resources); err != nil {
		return err
	}
	wantProvider := strings.TrimSpace(os.Getenv("NEXUS_E2E_PROVIDER"))
	wantProfile := strings.TrimSpace(os.Getenv("NEXUS_E2E_PROFILE"))
	var provider, profile string
	for _, account := range resources.Accounts {
		if !account.Authenticated || !account.Available || account.RateLimited {
			continue
		}
		if wantProvider != "" && account.Provider != wantProvider {
			continue
		}
		if wantProfile != "" && account.Profile != wantProfile {
			continue
		}
		provider, profile = account.Provider, account.Profile
		break
	}
	if provider == "" {
		return fmt.Errorf("%w: no eligible authenticated provider/profile matches provider=%q profile=%q", errNotAuthenticated, wantProvider, wantProfile)
	}
	if mission {
		return runMissionE2E(ctx, api, project.ID, provider, profile, projectPath)
	}

	var agent struct {
		ID string `json:"id"`
	}
	if err := api.do(ctx, http.MethodPost, "/api/v1/projects/"+url.PathEscape(project.ID)+"/agents", map[string]any{
		"name": "E2E Direct " + provider,
		"role": "developer",
	}, &agent); err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	if agent.ID == "" {
		return fmt.Errorf("create agent returned empty id")
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = api.do(cleanupCtx, http.MethodPost, "/api/v1/agents/"+url.PathEscape(agent.ID)+"/stop", map[string]any{}, nil)
		if os.Getenv("NEXUS_E2E_KEEP") != "1" {
			_ = api.do(cleanupCtx, http.MethodDelete, "/api/v1/agents/"+url.PathEscape(agent.ID), nil, nil)
			_ = api.do(cleanupCtx, http.MethodDelete, "/api/v1/projects/"+url.PathEscape(project.ID), nil, nil)
		}
	}()

	if err := api.do(ctx, http.MethodPost, "/api/v1/resources/select", map[string]any{
		"provider": provider,
		"profile":  profile,
		"policy":   "MANUAL",
		"agent_id": agent.ID,
	}, nil); err != nil {
		return fmt.Errorf("persist resource: %w", err)
	}

	var started struct {
		Runtime struct {
			RuntimeID string `json:"runtime_id"`
			AgentID   string `json:"agent_id"`
			State     string `json:"state"`
		} `json:"runtime"`
	}
	if err := api.do(ctx, http.MethodPost, "/api/v1/agents/"+url.PathEscape(agent.ID)+"/start", map[string]any{}, &started); err != nil {
		return fmt.Errorf("start agent: %w", err)
	}
	if started.Runtime.RuntimeID == "" || started.Runtime.AgentID != agent.ID {
		return fmt.Errorf("runtime identity mismatch runtime=%q runtime.agent_id=%q agent=%q", started.Runtime.RuntimeID, started.Runtime.AgentID, agent.ID)
	}

	wsScheme := "ws"
	if u.Scheme == "https" {
		wsScheme = "wss"
	}
	wsURL := wsScheme + "://" + u.Host + "/api/v1/agents/" + url.PathEscape(agent.ID) + "/terminal"
	header := http.Header{}
	header.Set("Origin", base)
	baseURL, _ := url.Parse(base)
	if cookies := jar.Cookies(baseURL); len(cookies) > 0 {
		parts := make([]string, 0, len(cookies))
		for _, c := range cookies {
			parts = append(parts, c.Name+"="+c.Value)
		}
		header.Set("Cookie", strings.Join(parts, "; "))
	}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	ws, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return fmt.Errorf("agent terminal websocket: %w", err)
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(45 * time.Second))
	control := false
	for !control {
		var msg map[string]any
		if err := ws.ReadJSON(&msg); err != nil {
			return fmt.Errorf("wait for terminal CONTROL lease: %w", err)
		}
		if msg["type"] == "lease" && msg["role"] == "CONTROL" {
			control = true
		}
	}

	prompt := strings.TrimSpace(os.Getenv("NEXUS_E2E_PROMPT"))
	if prompt == "" {
		prompt = "Reply with exactly NEXUS_E2E_OK and nothing else."
	}
	if err := ws.WriteJSON(map[string]any{"type": "input", "data": prompt + "\r"}); err != nil {
		return fmt.Errorf("send prompt: %w", err)
	}

	var providerOutput strings.Builder
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		_ = ws.SetReadDeadline(deadline)
		var msg map[string]any
		if err := ws.ReadJSON(&msg); err != nil {
			return fmt.Errorf("wait for provider output after prompt: %w; sanitized transcript: %q", err, transcriptExcerpt(sanitizeTranscript(providerOutput.String())))
		}
		if msg["type"] == "output" {
			if s, ok := msg["data"].(string); ok && s != "" {
				providerOutput.WriteString(s)
				if providerMarkerSeen(providerOutput.String()) {
					break
				}
			}
		}
	}
	transcript := sanitizeTranscript(providerOutput.String())
	if !providerMarkerSeen(transcript) {
		return fmt.Errorf("provider answer marker %q was not received; sanitized transcript: %q", providerAnswerMarker, transcriptExcerpt(transcript))
	}

	var detail struct {
		Generations []struct {
			RuntimeID string `json:"runtime_id"`
			AgentID   string `json:"agent_id"`
		} `json:"generations"`
	}
	if err := api.do(ctx, http.MethodGet, "/api/v1/agents/"+url.PathEscape(agent.ID), nil, &detail); err != nil {
		return err
	}
	found := false
	for _, generation := range detail.Generations {
		if generation.RuntimeID == started.Runtime.RuntimeID && generation.AgentID == agent.ID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("runtime generation was not durably linked back to the Agent")
	}

	// A reconnect must be agent-scoped. Verify that the same agent still points
	// at the same generation after the terminal exchange; this is the HTTP-side
	// invariant consumed by the browser's reconnect logic.
	var afterReconnect struct {
		Agent struct {
			ID string `json:"id"`
		} `json:"agent"`
		Generations []struct {
			RuntimeID string `json:"runtime_id"`
			AgentID   string `json:"agent_id"`
		} `json:"generations"`
	}
	if err := api.do(ctx, http.MethodGet, "/api/v1/agents/"+url.PathEscape(agent.ID), nil, &afterReconnect); err != nil {
		return fmt.Errorf("agent refresh after websocket exchange: %w", err)
	}
	if afterReconnect.Agent.ID != agent.ID || len(afterReconnect.Generations) == 0 {
		return fmt.Errorf("agent identity was not preserved after reconnect")
	}
	if safeApply {
		if err := verifySafeApply(ctx, api, u, jar, base, agent.ID, started.Runtime.RuntimeID); err != nil {
			return err
		}
	}
	return nil
}

const missionE2EMarkerFile = "nexus-e2e-mission.txt"
const missionE2EMarker = "NEXUS_MISSION_E2E_OK"

func validateMissionE2EProject(ctx context.Context, projectPath string) error {
	for _, args := range [][]string{
		{"-C", projectPath, "rev-parse", "--is-inside-work-tree"},
		{"-C", projectPath, "rev-parse", "HEAD"},
	} {
		cmd := exec.CommandContext(ctx, "git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("mission E2E requires an initialized Git repository with HEAD: git %v: %w: %s", args, err, strings.TrimSpace(string(out)))
		}
	}
	status, err := exec.CommandContext(ctx, "git", "-C", projectPath, "status", "--porcelain").Output()
	if err != nil {
		return fmt.Errorf("inspect Mission E2E project status: %w", err)
	}
	if strings.TrimSpace(string(status)) != "" {
		return fmt.Errorf("mission E2E requires a clean disposable repository; working tree has local changes")
	}
	return nil
}

type missionE2EPreview struct {
	Revision                 int      `json:"revision"`
	Ready                    bool     `json:"ready"`
	Blockers                 []string `json:"blockers"`
	ApprovalToken            string   `json:"approval_token"`
	CostConfirmationRequired bool     `json:"cost_confirmation_required"`
	CostConfirmationToken    string   `json:"cost_confirmation_token"`
}

type missionE2EPackageRun struct {
	PackageID       string `json:"package_id"`
	State           string `json:"state"`
	AssignedAgent   string `json:"assigned_agent"`
	AssignedRuntime string `json:"assigned_runtime"`
	Workspace       string `json:"workspace"`
}

type missionE2ERun struct {
	ID          string                 `json:"id"`
	State       string                 `json:"state"`
	PackageRuns []missionE2EPackageRun `json:"package_runs"`
}

type missionE2EReceipt struct {
	StepID       string   `json:"step_id"`
	Status       string   `json:"status"`
	ChangedFiles []string `json:"changed_files"`
}

type missionE2EEvidence struct {
	Capsules []json.RawMessage   `json:"capsules"`
	Receipts []missionE2EReceipt `json:"receipts"`
}

func runMissionE2E(ctx context.Context, api *apiClient, projectID, provider, profile, projectPath string) error {
	if api == nil {
		return fmt.Errorf("mission E2E API client is required")
	}
	if err := api.do(ctx, http.MethodPatch, "/api/v1/projects/"+url.PathEscape(projectID), map[string]any{
		"default_isolation": "worktree",
	}, nil); err != nil {
		return fmt.Errorf("configure Mission E2E worktree isolation: %w", err)
	}

	agentIDs := map[string]struct{}{}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if os.Getenv("NEXUS_E2E_KEEP") == "1" {
			return
		}
		for agentID := range agentIDs {
			_ = api.do(cleanupCtx, http.MethodPost, "/api/v1/agents/"+url.PathEscape(agentID)+"/stop", map[string]any{}, nil)
			_ = api.do(cleanupCtx, http.MethodDelete, "/api/v1/agents/"+url.PathEscape(agentID), nil, nil)
		}
		_ = api.do(cleanupCtx, http.MethodDelete, "/api/v1/projects/"+url.PathEscape(projectID), nil, nil)
	}()

	intentRaw, _ := json.Marshal(map[string]any{
		"strategy":                "PLAN",
		"confidence":              "HIGH",
		"recommended_next_action": "create_work_plan",
		"task_requirements": map[string]any{
			"task_kind": "coding",
			"role":      "implementer",
		},
	})
	goal := "Create a UTF-8 text file named " + missionE2EMarkerFile +
		" in the repository root containing exactly " + missionE2EMarker +
		" followed by one newline. Do not modify any other file. Do not commit, push, deploy, access secrets, or use external network."

	var plan struct {
		ID              string `json:"id"`
		CurrentRevision int    `json:"current_revision"`
	}
	if err := api.do(ctx, http.MethodPost, "/api/v1/projects/"+url.PathEscape(projectID)+"/plans", map[string]any{
		"title":       "Authenticated Mission E2E",
		"description": goal,
		"facts": map[string]string{
			"nexus.intent_decision":   string(intentRaw),
			"nexus.intent_strategy":   "PLAN",
			"nexus.intent_confidence": "HIGH",
			"nexus.delegation_mode":   "OFF",
		},
		"phases": []map[string]any{{
			"id":    "phase-e2e",
			"title": "Mission E2E implementation",
			"order": 1,
			"type":  "IMPLEMENTATION",
			"packages": []map[string]any{{
				"id":                        "mission-e2e",
				"title":                     "Create Mission E2E marker",
				"goal":                      goal,
				"priority":                  "HIGH",
				"status":                    "READY",
				"dependencies":              []string{},
				"role":                      "implementer",
				"phase_type":                "IMPLEMENTATION",
				"review_required":           false,
				"assignment_strategy":       "CREATE",
				"resource_policy":           "MANUAL",
				"provider":                  provider,
				"profile":                   profile,
				"relevant_paths":            []string{missionE2EMarkerFile},
				"acceptance_criteria":       []string{"marker file exists with the exact requested content and no unrelated file changed"},
				"verification_requirements": []string{"git diff --check"},
				"required_evidence":         []string{"VERIFICATION_PASSED"},
			}},
		}},
	}, &plan); err != nil {
		return fmt.Errorf("create Mission E2E plan: %w", err)
	}
	if plan.ID == "" || plan.CurrentRevision <= 0 {
		return fmt.Errorf("mission E2E plan returned invalid identity/revision")
	}

	var preview missionE2EPreview
	if err := api.do(ctx, http.MethodPost, "/api/v1/plans/"+url.PathEscape(plan.ID)+"/execution-preview", map[string]any{}, &preview); err != nil {
		return fmt.Errorf("mission E2E execution preview: %w", err)
	}
	if !preview.Ready || preview.ApprovalToken == "" || preview.Revision != plan.CurrentRevision {
		return fmt.Errorf("mission E2E preview not ready: revision=%d ready=%t blockers=%v", preview.Revision, preview.Ready, preview.Blockers)
	}

	var preflight struct {
		Ready    bool     `json:"ready"`
		Blockers []string `json:"blockers"`
	}
	if err := api.do(ctx, http.MethodPost, "/api/v1/plans/"+url.PathEscape(plan.ID)+"/preflight", map[string]any{}, &preflight); err != nil {
		return fmt.Errorf("mission E2E preflight: %w", err)
	}
	if !preflight.Ready {
		return fmt.Errorf("mission E2E preflight blocked: %v", preflight.Blockers)
	}

	runBody := map[string]any{
		"plan_revision":           plan.CurrentRevision,
		"execution_preview_token": preview.ApprovalToken,
		"autonomous":              true,
	}
	if preview.CostConfirmationRequired {
		if strings.TrimSpace(preview.CostConfirmationToken) == "" {
			return fmt.Errorf("mission E2E cost confirmation is required but preview returned no token")
		}
		runBody["cost_confirmation_token"] = preview.CostConfirmationToken
	}

	var run missionE2ERun
	if err := api.do(ctx, http.MethodPost, "/api/v1/plans/"+url.PathEscape(plan.ID)+"/run", runBody, &run); err != nil {
		return fmt.Errorf("start Mission E2E run: %w", err)
	}
	if run.ID == "" {
		return fmt.Errorf("mission E2E run returned empty id")
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := api.do(ctx, http.MethodGet, "/api/v1/runs/"+url.PathEscape(run.ID), nil, &run); err != nil {
			return fmt.Errorf("poll Mission E2E run: %w", err)
		}
		switch run.State {
		case "COMPLETED_VERIFIED":
			goto completed
		case "FAILED", "FAILED_NO_PROGRESS", "FAILED_BUDGET_EXCEEDED", "FAILED_VERIFICATION", "CANCELED_BY_USER":
			return fmt.Errorf("Mission E2E terminated in %s", run.State)
		case "BLOCKED_NEEDS_USER", "PAUSED":
			return fmt.Errorf("Mission E2E requires intervention in state %s", run.State)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("Mission E2E timed out in state %s: %w", run.State, ctx.Err())
		case <-ticker.C:
		}
	}

completed:
	if len(run.PackageRuns) != 1 {
		return fmt.Errorf("Mission E2E expected exactly one package run, got %d", len(run.PackageRuns))
	}
	pkg := run.PackageRuns[0]
	if pkg.PackageID != "mission-e2e" || pkg.State != "VERIFIED" ||
		pkg.AssignedAgent == "" || pkg.AssignedRuntime == "" || strings.TrimSpace(pkg.Workspace) == "" {
		return fmt.Errorf("Mission E2E package lacks verified execution identity: %+v", pkg)
	}
	agentIDs[pkg.AssignedAgent] = struct{}{}

	markerPath := filepath.Join(pkg.Workspace, missionE2EMarkerFile)
	markerBytes, err := os.ReadFile(markerPath)
	if err != nil {
		return fmt.Errorf("read Mission E2E marker from isolated worktree: %w", err)
	}
	if string(markerBytes) != missionE2EMarker+"\n" {
		return fmt.Errorf("Mission E2E marker content mismatch")
	}
	if filepath.Clean(pkg.Workspace) == filepath.Clean(projectPath) {
		return fmt.Errorf("Mission E2E autonomous package executed in the source checkout instead of an isolated worktree")
	}

	var evidence missionE2EEvidence
	if err := api.do(ctx, http.MethodGet, "/api/v1/runs/"+url.PathEscape(run.ID)+"/evidence", nil, &evidence); err != nil {
		return fmt.Errorf("read Mission E2E evidence: %w", err)
	}
	if len(evidence.Capsules) == 0 {
		return fmt.Errorf("Mission E2E produced no persisted ContextCapsule")
	}
	if len(evidence.Receipts) != 1 {
		return fmt.Errorf("Mission E2E expected one WorkReceipt, got %d", len(evidence.Receipts))
	}
	receipt := evidence.Receipts[0]
	if receipt.StepID != "mission-e2e" || receipt.Status != "VERIFIED" ||
		len(receipt.ChangedFiles) != 1 || filepath.ToSlash(receipt.ChangedFiles[0]) != missionE2EMarkerFile {
		return fmt.Errorf("Mission E2E WorkReceipt does not prove the expected isolated change: %+v", receipt)
	}
	return nil
}

// verifySafeApply changes only a harmless runtime environment marker. That is
// deliberately a restart-required config change: it proves a fresh runtime
// generation and reconnect without changing provider/profile or credentials.
func verifySafeApply(ctx context.Context, api *apiClient, bootstrap *url.URL, jar http.CookieJar, base, agentID, previousRuntimeID string) error {
	var current struct {
		Config map[string]any `json:"config"`
	}
	if err := api.do(ctx, http.MethodGet, "/api/v1/agents/"+url.PathEscape(agentID)+"/config", nil, &current); err != nil {
		return fmt.Errorf("read config for Safe Apply: %w", err)
	}
	if current.Config == nil {
		return fmt.Errorf("safe apply current config is empty")
	}
	environment, _ := current.Config["environment"].(map[string]any)
	if environment == nil {
		environment = map[string]any{}
	}
	environment["NEXUS_E2E_SAFE_APPLY"] = "1"
	current.Config["environment"] = environment

	var preview struct {
		Impact struct {
			Mode            string `json:"mode"`
			RequiresRestart bool   `json:"requires_restart"`
		} `json:"impact"`
	}
	if err := api.do(ctx, http.MethodPost, "/api/v1/agents/"+url.PathEscape(agentID)+"/config/impact", current.Config, &preview); err != nil {
		return fmt.Errorf("preview Safe Apply: %w", err)
	}
	if preview.Impact.Mode != "RESTART_RUNTIME" || !preview.Impact.RequiresRestart {
		return fmt.Errorf("safe apply preview did not require restart: mode=%q restart=%t", preview.Impact.Mode, preview.Impact.RequiresRestart)
	}
	if err := api.do(ctx, http.MethodPost, "/api/v1/agents/"+url.PathEscape(agentID)+"/config/apply", current.Config, nil); err != nil {
		return fmt.Errorf("apply Safe Apply: %w", err)
	}

	var detail struct {
		Agent struct {
			ID string `json:"id"`
		} `json:"agent"`
		Generations []struct {
			RuntimeID string `json:"runtime_id"`
			AgentID   string `json:"agent_id"`
		} `json:"generations"`
	}
	if err := api.do(ctx, http.MethodGet, "/api/v1/agents/"+url.PathEscape(agentID), nil, &detail); err != nil {
		return fmt.Errorf("read Safe Apply generation: %w", err)
	}
	if detail.Agent.ID != agentID || len(detail.Generations) < 2 || detail.Generations[0].RuntimeID == "" || detail.Generations[0].RuntimeID == previousRuntimeID || detail.Generations[0].AgentID != agentID {
		return fmt.Errorf("safe apply did not create a new runtime generation for the same agent")
	}

	wsScheme := "ws"
	if bootstrap.Scheme == "https" {
		wsScheme = "wss"
	}
	wsURL := wsScheme + "://" + bootstrap.Host + "/api/v1/agents/" + url.PathEscape(agentID) + "/terminal"
	header := http.Header{}
	header.Set("Origin", base)
	baseURL, _ := url.Parse(base)
	if cookies := jar.Cookies(baseURL); len(cookies) > 0 {
		parts := make([]string, 0, len(cookies))
		for _, c := range cookies {
			parts = append(parts, c.Name+"="+c.Value)
		}
		header.Set("Cookie", strings.Join(parts, "; "))
	}
	ws, _, err := (&websocket.Dialer{HandshakeTimeout: 10 * time.Second}).DialContext(ctx, wsURL, header)
	if err != nil {
		return fmt.Errorf("safe apply terminal reconnect: %w", err)
	}
	defer ws.Close()
	deadline := time.Now().Add(45 * time.Second)
	for {
		_ = ws.SetReadDeadline(deadline)
		var msg map[string]any
		if err := ws.ReadJSON(&msg); err != nil {
			return fmt.Errorf("wait for Safe Apply CONTROL: %w", err)
		}
		if msg["type"] == "lease" && msg["role"] == "CONTROL" {
			break
		}
	}
	if err := ws.WriteJSON(map[string]any{"type": "input", "data": "Reply with exactly NEXUS_E2E_OK and nothing else.\r"}); err != nil {
		return fmt.Errorf("send Safe Apply marker prompt: %w", err)
	}
	var transcript strings.Builder
	for time.Now().Before(deadline) {
		_ = ws.SetReadDeadline(deadline)
		var msg map[string]any
		if err := ws.ReadJSON(&msg); err != nil {
			return fmt.Errorf("wait for Safe Apply provider output: %w", err)
		}
		if msg["type"] == "output" {
			if text, ok := msg["data"].(string); ok {
				transcript.WriteString(text)
				if providerMarkerSeen(transcript.String()) {
					return nil
				}
			}
		}
	}
	return fmt.Errorf("safe apply provider marker %q was not received; sanitized transcript: %q", providerAnswerMarker, transcriptExcerpt(sanitizeTranscript(transcript.String())))
}

func transcriptExcerpt(transcript string) string {
	const max = 1000
	if len(transcript) > max {
		return transcript[:max] + "…"
	}
	return transcript
}

// resolveE2ERoot keeps the default harness isolated and disposable, while
// allowing providers that reject temporary homes (for example Codex's helper
// binary policy) to run from an explicitly provisioned, empty directory.
// Explicit roots are never removed by the harness.
func resolveE2ERoot() (string, bool, error) {
	configured := strings.TrimSpace(os.Getenv("NEXUS_E2E_ROOT"))
	if configured == "" {
		root, err := os.MkdirTemp("", "nexus-e2e-")
		return root, true, err
	}

	root, err := filepath.Abs(configured)
	if err != nil {
		return "", false, fmt.Errorf("resolve explicit E2E root: %w", err)
	}
	if root == filepath.VolumeName(root)+string(filepath.Separator) {
		return "", false, fmt.Errorf("refusing filesystem root as explicit E2E root")
	}
	info, err := os.Lstat(root)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := os.MkdirAll(root, 0700); err != nil {
			return "", false, fmt.Errorf("create explicit E2E root: %w", err)
		}
		info, err = os.Lstat(root)
		if err != nil {
			return "", false, fmt.Errorf("inspect created E2E root: %w", err)
		}
	case err != nil:
		return "", false, fmt.Errorf("inspect explicit E2E root: %w", err)
	case info.Mode()&os.ModeSymlink != 0:
		return "", false, fmt.Errorf("explicit E2E root must not be a symlink: %s", root)
	case !info.IsDir():
		return "", false, fmt.Errorf("explicit E2E root is not a directory: %s", root)
	}
	// Windows does not expose POSIX permission bits for this policy. The ACL
	// boundary is enforced by the OS there; owner-only mode is meaningful on
	// Unix only.
	if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return "", false, fmt.Errorf("explicit E2E root must be owner-only (0700): %s", root)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", false, fmt.Errorf("read explicit E2E root: %w", err)
	}
	if len(entries) != 0 {
		return "", false, fmt.Errorf("explicit E2E root must be empty: %s", root)
	}
	return root, false, nil
}

func startLocalNexus(ctx context.Context, port int, keep bool, browser bool) (func(), error) {
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port %d", port)
	}
	root, removeRoot, err := resolveE2ERoot()
	if err != nil {
		return nil, err
	}
	cleanupRoot := func() {
		if removeRoot && !keep {
			_ = os.RemoveAll(root)
		}
	}
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(project, 0700); err != nil {
		cleanupRoot()
		return nil, err
	}
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "e2e@example.invalid"}, {"config", "user.name", "Nexus E2E"}} {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", project}, args...)...)
		if out, e := cmd.CombinedOutput(); e != nil {
			cleanupRoot()
			return nil, fmt.Errorf("temporary git project: %w (%s)", e, strings.TrimSpace(string(out)))
		}
	}

	bin := strings.TrimSpace(os.Getenv("NEXUS_E2E_NEXUS_BIN"))
	if bin == "" {
		bin = "./nexus"
	}
	cmd := exec.CommandContext(ctx, bin, "web", "--listen", "127.0.0.1", "--port", fmt.Sprint(port), "--no-open")
	dataDir := filepath.Join(root, "data")
	if os.Getenv("NEXUS_E2E_IMPORT_PROFILES") == "1" {
		source := resolveE2EProfileSource()
		if err := copyProfileTree(filepath.Join(source, "profiles"), filepath.Join(dataDir, "profiles")); err != nil {
			cleanupRoot()
			return nil, fmt.Errorf("copy authenticated profiles to isolated data: %w", err)
		}
	}
	cmd.Env = append(os.Environ(), "NEXUS_DATA_DIR="+dataDir, "AI_MANAGER_DATA_DIR="+dataDir, "NEXUS_E2E_PROJECT_PATH="+project)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = redactWriter{dst: os.Stderr}
	if err := cmd.Start(); err != nil {
		cleanupRoot()
		return nil, fmt.Errorf("start Nexus: %w (build ./nexus first or set NEXUS_E2E_NEXUS_BIN)", err)
	}

	ready := make(chan string, 1)
	go func() {
		buf := make([]byte, 4096)
		var all strings.Builder
		for {
			n, e := stdout.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				all.WriteString(chunk)
				if u := findBootstrap(all.String()); u != "" {
					ready <- u
					return
				}
				// The web command intentionally prints only the public URL. Resolve
				// the one-time token through the persisted `web url` command instead
				// of requiring the server to echo credentials to stdout.
				if strings.Contains(all.String(), "URL:") {
					probe := exec.CommandContext(ctx, bin, "web", "url")
					probe.Env = cmd.Env
					if out, probeErr := probe.Output(); probeErr == nil {
						if u := parseBootstrapURL(strings.TrimSpace(string(out))); u != "" {
							ready <- u
							return
						}
					}
				}
			}
			if e != nil {
				return
			}
		}
	}()
	select {
	case bootstrap := <-ready:
		os.Setenv("NEXUS_BOOTSTRAP_URL", bootstrap)
		if browser {
			artifactDir := filepath.Join(root, "artifacts")
			smoke := exec.CommandContext(ctx, "./scripts/nexus-browser-smoke.sh")
			smoke.Env = append(os.Environ(), "NEXUS_BOOTSTRAP_URL="+bootstrap, "NEXUS_E2E_ARTIFACT_DIR="+artifactDir)
			smoke.Stdout = os.Stdout
			smoke.Stderr = redactWriter{dst: os.Stderr}
			if err := smoke.Run(); err != nil {
				if strings.Contains(err.Error(), "exit status 2") {
					fmt.Fprintln(os.Stderr, "NEXUS_BROWSER_SMOKE_NOT_RUN: browser executable unavailable")
				} else {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
					cleanupRoot()
					return nil, fmt.Errorf("browser smoke: %w", err)
				}
			}
		}
		return func() {
			_ = cmd.Process.Signal(os.Interrupt)
			_ = cmd.Wait()
			cleanupRoot()
		}, nil
	case <-time.After(15 * time.Second):
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		cleanupRoot()
		return nil, fmt.Errorf("timed out waiting for Nexus bootstrap URL")
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		cleanupRoot()
		return nil, ctx.Err()
	}
}

// resolveE2EProfileSource identifies the host data root used to import
// authenticated profiles. HOME may be rewritten by an isolated provider
// runtime, so it is not a reliable source of the real Nexus profile store.
func resolveE2EProfileSource() string {
	if source := strings.TrimSpace(os.Getenv("NEXUS_E2E_PROFILE_SOURCE")); source != "" {
		return source
	}
	if hostHome := security.FindHostHome(); hostHome != "" {
		return filepath.Join(hostHome, ".local", "share", "ai-manager")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "ai-manager")
}

func copyProfileTree(source, destination string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		from := filepath.Join(source, entry.Name())
		to := filepath.Join(destination, entry.Name())
		// DirEntry.Info may describe a symlink target on some filesystems;
		// reject the directory entry itself before doing any metadata lookup so
		// an isolated E2E profile can never follow a link loop or escape source.
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		info, err := os.Lstat(from)
		if err != nil {
			if os.IsNotExist(err) {
				// Ignore dangling provider-cache symlinks.
				continue
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || entryInfo.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if info.IsDir() {
			if profileCopyExcludedDirs[entry.Name()] {
				continue
			}
			if err := os.MkdirAll(to, 0700); err != nil {
				return err
			}
			if err := copyProfileTree(from, to); err != nil {
				return err
			}
			continue
		}
		// Provider homes may contain live IPC sockets and other transient
		// non-regular entries. Never copy those into the isolated E2E data.
		if !info.Mode().IsRegular() {
			continue
		}
		data, err := os.ReadFile(from)
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if err := os.MkdirAll(filepath.Dir(to), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(to, data, mode); err != nil {
			return err
		}
	}
	return nil
}

func findBootstrap(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Bootstrap:") {
			continue
		}
		u := strings.TrimSpace(strings.TrimPrefix(line, "Bootstrap:"))
		if parsed := parseBootstrapURL(u); parsed != "" {
			return parsed
		}
	}
	return ""
}

func parseBootstrapURL(u string) string {
	parsed, err := url.Parse(strings.TrimSpace(u))
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" {
		return ""
	}
	if parsed.Query().Get("token") == "" && !strings.HasPrefix(parsed.Fragment, "nexus_bootstrap=") {
		return ""
	}
	return parsed.String()
}

func bootstrapTokenFromURL(bootstrap *url.URL) (string, error) {
	if bootstrap == nil {
		return "", fmt.Errorf("bootstrap URL is required")
	}
	if token := strings.TrimSpace(bootstrap.Query().Get("token")); token != "" {
		return token, nil
	}
	const prefix = "nexus_bootstrap="
	fragment := strings.TrimSpace(bootstrap.Fragment)
	if !strings.HasPrefix(fragment, prefix) {
		return "", fmt.Errorf("bootstrap URL does not contain a token")
	}
	token, err := url.QueryUnescape(strings.TrimPrefix(fragment, prefix))
	if err != nil || strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("bootstrap URL contains an invalid token")
	}
	return strings.TrimSpace(token), nil
}

type redactWriter struct{ dst io.Writer }

func (w redactWriter) Write(p []byte) (int, error) {
	s := string(p)
	if i := strings.Index(s, "token="); i >= 0 {
		s = s[:i] + "token=[REDACTED]"
	}
	return w.dst.Write([]byte(s))
}

func (a *apiClient) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.base+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet && method != http.MethodHead && a.csrf != "" {
		req.Header.Set("X-CSRF-Token", a.csrf)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode %s %s: %w", method, path, err)
		}
	}
	return nil
}
