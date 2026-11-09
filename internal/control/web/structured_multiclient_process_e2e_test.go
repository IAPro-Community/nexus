package web

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/host"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

// A real child process speaking ACP JSON-RPC, not an in-memory registry state
// fixture. It requests an identical permission on each of two independent
// provider sessions and completes only after the Nexus response is received.
func TestACPProcessPermissionFixture(t *testing.T) {
	if os.Getenv("NEXUS_ACP_E2E_CHILD") != "1" {
		return
	}
	sessionID := os.Getenv("NEXUS_ACP_E2E_SESSION")
	in := bufio.NewScanner(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	send := func(message any) {
		if err := json.NewEncoder(out).Encode(message); err != nil {
			os.Exit(2)
		}
		if err := out.Flush(); err != nil {
			os.Exit(2)
		}
	}
	var promptID json.RawMessage
	for in.Scan() {
		var message struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(in.Bytes(), &message); err != nil {
			os.Exit(3)
		}
		switch message.Method {
		case "initialize":
			send(map[string]any{"jsonrpc": "2.0", "id": message.ID,
				"result": map[string]any{"protocolVersion": 1}})
		case "session/new":
			send(map[string]any{"jsonrpc": "2.0", "id": message.ID,
				"result": map[string]string{"sessionId": sessionID}})
		case "session/prompt":
			promptID = append([]byte(nil), message.ID...)
			send(map[string]any{"jsonrpc": "2.0", "id": "request-permission",
				"method": "session/request_permission",
				"params": map[string]any{
					"sessionId": sessionID, "title": "Allow the same harmless action?",
					"options": []map[string]string{
						{"optionId": "deny", "name": "Deny"},
						{"optionId": "allow_once", "name": "Allow once"},
					},
				},
			})
		default:
			if string(message.ID) != `"request-permission"` || len(promptID) == 0 {
				continue
			}
			var result struct {
				Outcome struct {
					Outcome  string `json:"outcome"`
					OptionID string `json:"optionId"`
				} `json:"outcome"`
			}
			if json.Unmarshal(message.Result, &result) != nil || result.Outcome.Outcome != "selected" {
				os.Exit(4)
			}
			send(map[string]any{"jsonrpc": "2.0", "id": promptID,
				"result": map[string]string{"stopReason": "end_turn"}})
			promptID = nil
		}
	}
	os.Exit(0)
}

func TestTwoSupervisedACPProcessesAndTwoAuthenticatedHTTPClients(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AI_MANAGER_DATA_DIR", dir)
	t.Setenv("AI_CLI_DATA_DIR", dir)
	reg := registry.DefaultRegistry()

	launch := func(runtimeID, projectID, sessionID string) *host.SessionHost {
		t.Helper()
		workspace := filepath.Join(dir, projectID)
		if err := os.MkdirAll(workspace, 0700); err != nil {
			t.Fatal(err)
		}
		s := registry.RuntimeSession{
			RuntimeID: runtimeID, ProviderID: "opencode", ProfileID: "default",
			Workspace: workspace, ProjectID: projectID, ProjectName: projectID,
			State: registry.StateStarting, StartedAt: time.Now(),
			ControlLevel: registry.ControlLevelEvents,
			Labels:       map[string]string{"nexus.agent_protocol": "acp"},
		}
		h, err := host.NewSessionHost(host.Config{
			Session: s, Registry: reg, Binary: os.Args[0],
			Args: []string{"-test.run=^TestACPProcessPermissionFixture$"},
			Env:  append(os.Environ(), "NEXUS_ACP_E2E_CHILD=1", "NEXUS_ACP_E2E_SESSION="+sessionID),
			Cwd:  workspace,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := h.Start(); err != nil {
			t.Fatalf("start %s: %v", runtimeID, err)
		}
		return h
	}
	alpha := launch("acp-http-agent-alpha", "project-alpha", "provider-alpha")
	defer func() { _ = alpha.Terminate(); alpha.Wait() }()
	beta := launch("acp-http-agent-beta", "project-beta", "provider-beta")
	defer func() { _ = beta.Terminate(); beta.Wait() }()

	srv, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: 0, NoOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Start() }()
	defer srv.Shutdown(context.Background())

	client := func() *http.Client {
		t.Helper()
		jar, err := cookiejar.New(nil)
		if err != nil {
			t.Fatal(err)
		}
		c := &http.Client{Jar: jar, Timeout: 3 * time.Second}
		res, err := c.Post(srv.URL()+"/api/v1/auth/bootstrap", "application/json",
			strings.NewReader(fmt.Sprintf(`{"token":%q}`, srv.bootstrap)))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("bootstrap status %d", res.StatusCode)
		}
		return c
	}
	desktop := client()
	mobile := client()
	read := func(c *http.Client) map[string]registry.RuntimeSession {
		t.Helper()
		response, err := c.Get(srv.URL() + "/api/v1/runtimes")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("runtime API status %d", response.StatusCode)
		}
		var sessions []registry.RuntimeSession
		if err := json.NewDecoder(response.Body).Decode(&sessions); err != nil {
			t.Fatal(err)
		}
		result := make(map[string]registry.RuntimeSession)
		for _, s := range sessions {
			result[s.RuntimeID] = s
		}
		return result
	}
	submit := func(id string) {
		t.Helper()
		c, err := protocol.NewClient(id)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if err := c.Ping(); err != nil {
			t.Fatal(err)
		}
		if _, err := c.RawConn().Write([]byte("ask safely\r\n")); err != nil {
			t.Fatal(err)
		}
	}
	submit("acp-http-agent-alpha")
	submit("acp-http-agent-beta")

	wait := func(condition func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if condition() {
				return
			}
			time.Sleep(25 * time.Millisecond)
		}
		t.Fatal("timed out waiting for process-backed attention transition")
	}
	wait(func() bool {
		a := read(desktop)
		b := read(mobile)
		alpha1, okA := a["acp-http-agent-alpha"]
		beta1, okB := b["acp-http-agent-beta"]
		if !okA || !okB {
			return false
		}
		return alpha1.AttentionKind == "needs_user" && beta1.AttentionKind == "needs_user"
	})
	first := read(desktop)
	a, b := first["acp-http-agent-alpha"], first["acp-http-agent-beta"]
	if a.AttentionContext != b.AttentionContext || a.AttentionFingerprint == b.AttentionFingerprint {
		t.Fatalf("same question merged across live process boundary: %q %q", a.AttentionFingerprint, b.AttentionFingerprint)
	}

	// Respond through the actual runtime IPC, not Registry.UpdateAttentionMeta.
	c, err := protocol.NewClient("acp-http-agent-alpha")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Ping(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.RawConn().Write([]byte("/permit req-1 deny\r\n")); err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
	wait(func() bool {
		state := read(mobile)
		return state["acp-http-agent-alpha"].AttentionKind == "completed" &&
			state["acp-http-agent-beta"].AttentionKind == "needs_user"
	})
	if err := beta.Terminate(); err != nil {
		t.Fatal(err)
	}
	beta.Wait()
	wait(func() bool {
		state := read(desktop)
		return state["acp-http-agent-beta"].AttentionKind == "idle"
	})
}

// TestACPAttentionRadarBrowserIntegration is the combined no-overlay contract:
// supervised ACP subprocesses publish attention through Registry and the real
// HTTP endpoint, then Playwright consumes that response and focuses each
// project-owned runtime terminal. Run it via the Web E2E npm script.
func TestACPAttentionRadarBrowserIntegration(t *testing.T) {
	if os.Getenv("NEXUS_E2E_BROWSER") != "1" {
		t.Skip("browser integration is run by test:e2e:attention-radar-process-api-focus")
	}
	dir := t.TempDir()
	t.Setenv("AI_MANAGER_DATA_DIR", dir)
	t.Setenv("AI_CLI_DATA_DIR", dir)
	t.Setenv("NEXUS_DATA_DIR", dir)
	t.Setenv("NEXUS_CONFIG_DIR", filepath.Join(dir, "config"))
	registry.ResetDefaultRegistryForTest()
	nexus.ResetDefaultForTest()
	t.Cleanup(registry.ResetDefaultRegistryForTest)
	t.Cleanup(nexus.ResetDefaultForTest)
	reg := registry.DefaultRegistry()

	srv, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: 0, NoOpen: true})
	if err != nil {
		t.Fatalf("create web server: %v", err)
	}
	go func() { _ = srv.Start() }()
	defer srv.Shutdown(context.Background())

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, Timeout: 3 * time.Second}
	bootstrap, err := client.Post(srv.URL()+"/api/v1/auth/bootstrap", "application/json",
		strings.NewReader(fmt.Sprintf(`{"token":%q}`, srv.bootstrap)))
	if err != nil {
		t.Fatalf("bootstrap HTTP client: %v", err)
	}
	if bootstrap.StatusCode != http.StatusOK {
		bootstrap.Body.Close()
		t.Fatalf("bootstrap HTTP status %d", bootstrap.StatusCode)
	}
	bootstrap.Body.Close()
	var session struct {
		CSRFToken string `json:"csrf_token"`
	}
	sessionResponse, err := client.Get(srv.URL() + "/api/v1/session")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sessionResponse.StatusCode != http.StatusOK {
		sessionResponse.Body.Close()
		t.Fatalf("session HTTP status %d", sessionResponse.StatusCode)
	}
	if err := json.NewDecoder(sessionResponse.Body).Decode(&session); err != nil {
		sessionResponse.Body.Close()
		t.Fatalf("decode session: %v", err)
	}
	sessionResponse.Body.Close()
	if session.CSRFToken == "" {
		t.Fatal("authenticated session did not return CSRF token")
	}

	type browserProject struct {
		ID   string
		Name string
		Path string
	}
	createProject := func(name string) browserProject {
		t.Helper()
		projectPath := filepath.Join(dir, strings.ToLower(strings.ReplaceAll(name, " ", "-")))
		if err := os.MkdirAll(projectPath, 0700); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.Command("git", "init", "-q", projectPath).CombinedOutput(); err != nil {
			t.Fatalf("git init %s: %v %s", name, err, output)
		}
		payload, _ := json.Marshal(map[string]string{"name": name, "path": projectPath})
		req, err := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/projects", strings.NewReader(string(payload)))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", session.CSRFToken)
		response, err := client.Do(req)
		if err != nil {
			t.Fatalf("create project %s: %v", name, err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(response.Body)
			t.Fatalf("create project %s status %d: %s", name, response.StatusCode, body)
		}
		var created struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
			t.Fatalf("decode project %s: %v", name, err)
		}
		if created.ID == "" {
			t.Fatalf("project %s has empty ID", name)
		}
		return browserProject{ID: created.ID, Name: created.Name, Path: projectPath}
	}
	alphaProject := createProject("Attention Radar Alpha")
	betaProject := createProject("Attention Radar Beta")
	providerBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve ACP fixture binary: %v", err)
	}
	providerFingerprint, err := registry.BinaryFingerprint(providerBinary)
	if err != nil {
		t.Fatalf("fingerprint ACP fixture binary: %v", err)
	}

	launch := func(runtimeID string, project browserProject, sessionID string) *host.SessionHost {
		t.Helper()
		runtimeSession := registry.RuntimeSession{
			RuntimeID: runtimeID, ProviderID: "opencode", ProfileID: "default",
			Workspace: project.Path, ProjectID: project.ID, ProjectName: project.Name,
			ProviderBinaryPath: providerBinary, ProviderBinaryFingerprint: providerFingerprint,
			State: registry.StateStarting, StartedAt: time.Now(),
			ControlLevel: registry.ControlLevelEvents,
			Labels:       map[string]string{"nexus.agent_protocol": "acp"},
		}
		processHost, err := host.NewSessionHost(host.Config{
			Session: runtimeSession, Registry: reg, Binary: os.Args[0],
			Args: []string{"-test.run=^TestACPProcessPermissionFixture$"},
			Env:  append(os.Environ(), "NEXUS_ACP_E2E_CHILD=1", "NEXUS_ACP_E2E_SESSION="+sessionID),
			Cwd:  project.Path,
		})
		if err != nil {
			t.Fatalf("create SessionHost %s: %v", runtimeID, err)
		}
		if err := processHost.Start(); err != nil {
			t.Fatalf("start SessionHost %s: %v", runtimeID, err)
		}
		return processHost
	}
	alpha := launch("radar-browser-alpha", alphaProject, "radar-provider-alpha")
	defer func() { _ = alpha.Terminate(); alpha.Wait() }()
	beta := launch("radar-browser-beta", betaProject, "radar-provider-beta")
	defer func() { _ = beta.Terminate(); beta.Wait() }()

	for _, runtimeID := range []string{"radar-browser-alpha", "radar-browser-beta"} {
		ipc, err := protocol.NewClient(runtimeID)
		if err != nil {
			t.Fatalf("connect to %s: %v", runtimeID, err)
		}
		if err := ipc.Ping(); err != nil {
			ipc.Close()
			t.Fatalf("ping %s: %v", runtimeID, err)
		}
		if _, err := ipc.RawConn().Write([]byte("ask safely\r\n")); err != nil {
			ipc.Close()
			t.Fatalf("submit ACP prompt to %s: %v", runtimeID, err)
		}
		ipc.Close()
	}

	readRuntimes := func() map[string]registry.RuntimeSession {
		t.Helper()
		response, err := client.Get(srv.URL() + "/api/v1/runtimes")
		if err != nil {
			t.Fatalf("read runtime API: %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("runtime API status %d", response.StatusCode)
		}
		var list []registry.RuntimeSession
		if err := json.NewDecoder(response.Body).Decode(&list); err != nil {
			t.Fatalf("decode runtime API: %v", err)
		}
		indexed := make(map[string]registry.RuntimeSession, len(list))
		for _, runtimeSession := range list {
			indexed[runtimeSession.RuntimeID] = runtimeSession
		}
		return indexed
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		indexed := readRuntimes()
		a, okA := indexed["radar-browser-alpha"]
		b, okB := indexed["radar-browser-beta"]
		if okA && okB && a.AttentionKind == "needs_user" && b.AttentionKind == "needs_user" {
			if a.AttentionContext != b.AttentionContext || a.AttentionFingerprint == b.AttentionFingerprint {
				t.Fatalf("same questions must remain separate: alpha=%+v beta=%+v", a, b)
			}
			if a.AttentionSource != "AUTHORITATIVE" || b.AttentionSource != "AUTHORITATIVE" {
				t.Fatalf("ACP provenance must be authoritative: alpha=%s beta=%s", a.AttentionSource, b.AttentionSource)
			}
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	indexed := readRuntimes()
	if indexed["radar-browser-alpha"].AttentionKind != "needs_user" || indexed["radar-browser-beta"].AttentionKind != "needs_user" {
		t.Fatal("ACP permission requests did not reach the real runtime API")
	}

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	browserCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	browser := exec.CommandContext(browserCtx, "node", "web/scripts/e2e-attention-radar-process-api-browser.mjs")
	browser.Dir = repoRoot
	browser.Env = append(os.Environ(),
		"NEXUS_E2E_BROWSER_ORIGIN="+srv.URL(),
		"NEXUS_E2E_BROWSER_BOOTSTRAP="+srv.bootstrap,
		"NEXUS_E2E_BROWSER_ALPHA_PROJECT="+alphaProject.ID,
		"NEXUS_E2E_BROWSER_BETA_PROJECT="+betaProject.ID,
		"NEXUS_E2E_BROWSER_ALPHA_RUNTIME=radar-browser-alpha",
		"NEXUS_E2E_BROWSER_BETA_RUNTIME=radar-browser-beta",
	)
	output, err := browser.CombinedOutput()
	if err != nil {
		t.Fatalf("Playwright continuation failed: %v\n%s", err, output)
	}
	t.Logf("Playwright continuation output:\n%s", output)
	_ = beta.Terminate()
	beta.Wait()
}
