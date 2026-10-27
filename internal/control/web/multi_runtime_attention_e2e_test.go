package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestTwoAgentsTwoClientsAttentionScenario(t *testing.T) {
	testDir := t.TempDir()
	t.Setenv("AI_MANAGER_DATA_DIR", testDir)
	t.Setenv("AI_CLI_DATA_DIR", testDir)

	reg := registry.DefaultRegistry()

	// 1. Setup two runtimes in two different projects
	runtime1 := registry.RuntimeSession{
		RuntimeID:        "rt-agent-alpha",
		ProviderID:       "opencode",
		ProfileID:        "default",
		Workspace:        filepath.Join(testDir, "proj-a"),
		ProjectID:        "project-alpha",
		ProjectName:      "Project Alpha",
		PID:              10001,
		HostPID:          10001,
		HostGeneration:   time.Now().UnixNano(),
		Binary:           "/usr/bin/opencode",
		State:            registry.StateRunning,
		ControlLevel:     registry.ControlLevelEvents,
		ControlEndpoint:  filepath.Join(testDir, "alpha.sock"),
		StartedAt:        time.Now(),
		AttentionReason:  "NEEDS_USER",
		AttentionKind:    "needs_user",
		PromptKind:       "ask_question",
		AttentionContext: "Do you want to apply the database migration?",
	}

	runtime2 := registry.RuntimeSession{
		RuntimeID:       "rt-agent-beta",
		ProviderID:      "omp",
		ProfileID:       "default",
		Workspace:       filepath.Join(testDir, "proj-b"),
		ProjectID:       "project-beta",
		ProjectName:     "Project Beta",
		PID:             10002,
		HostPID:         10002,
		HostGeneration:  time.Now().UnixNano(),
		Binary:          "/usr/bin/omp",
		State:           registry.StateRunning,
		ControlLevel:    registry.ControlLevelEvents,
		ControlEndpoint: filepath.Join(testDir, "beta.sock"),
		StartedAt:       time.Now(),
		AttentionReason: "NEEDS_USER",
		AttentionKind:   "needs_user",
		PromptKind:      "ask_question",
		// Identical question text to test deduplication resilience
		AttentionContext: "Do you want to apply the database migration?",
	}

	if err := reg.Register(runtime1); err != nil {
		t.Fatalf("failed to register runtime 1: %v", err)
	}
	if err := reg.Register(runtime2); err != nil {
		t.Fatalf("failed to register runtime 2: %v", err)
	}
	defer func() {
		_ = reg.UpdateState(runtime1.RuntimeID, registry.StateStopped)
		_ = reg.UpdateState(runtime2.RuntimeID, registry.StateStopped)
	}()

	srv, err := NewServer(ServerOptions{
		Host:   "127.0.0.1",
		Port:   0,
		NoOpen: true,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	go func() {
		_ = srv.Start()
	}()
	defer srv.Shutdown(context.Background())

	// Client 1 login
	jar1, _ := cookiejar.New(nil)
	client1 := &http.Client{Jar: jar1}
	resp1, err := client1.Post(srv.URL()+"/api/v1/auth/bootstrap", "application/json", strings.NewReader(`{"token":"`+srv.bootstrap+`"}`))
	if err != nil || resp1.StatusCode != http.StatusOK {
		t.Fatalf("client 1 bootstrap failed: %v", err)
	}
	_ = resp1.Body.Close()

	// Client 2 login (separate session / cookie jar)
	jar2, _ := cookiejar.New(nil)
	client2 := &http.Client{Jar: jar2}
	resp2, err := client2.Post(srv.URL()+"/api/v1/auth/bootstrap", "application/json", strings.NewReader(`{"token":"`+srv.bootstrap+`"}`))
	if err != nil || resp2.StatusCode != http.StatusOK {
		t.Fatalf("client 2 bootstrap failed: %v", err)
	}
	_ = resp2.Body.Close()

	fetchRuntimes := func(c *http.Client) []*registry.RuntimeSession {
		r, err := c.Get(srv.URL() + "/api/v1/runtimes")
		if err != nil {
			t.Fatalf("failed to fetch runtimes: %v", err)
		}
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		var list []*registry.RuntimeSession
		_ = json.Unmarshal(body, &list)
		return list
	}

	// Step 1: Verify Client 1 sees both runtimes independently with identical questions
	list1 := fetchRuntimes(client1)
	if len(list1) < 2 {
		t.Fatalf("expected at least 2 runtimes, got %d", len(list1))
	}

	var foundAlpha, foundBeta *registry.RuntimeSession
	for _, r := range list1 {
		if r.RuntimeID == "rt-agent-alpha" {
			foundAlpha = r
		}
		if r.RuntimeID == "rt-agent-beta" {
			foundBeta = r
		}
	}

	if foundAlpha == nil || foundBeta == nil {
		t.Fatalf("did not find both runtimes: alpha=%v, beta=%v", foundAlpha, foundBeta)
	}
	if foundAlpha.ProjectID != "project-alpha" || foundBeta.ProjectID != "project-beta" {
		t.Fatalf("project IDs incorrect: alpha=%s, beta=%s", foundAlpha.ProjectID, foundBeta.ProjectID)
	}
	if foundAlpha.AttentionContext != foundBeta.AttentionContext {
		t.Fatalf("questions should be identical fixture: %s vs %s", foundAlpha.AttentionContext, foundBeta.AttentionContext)
	}
	if foundAlpha.AttentionKind != "needs_user" || foundBeta.AttentionKind != "needs_user" {
		t.Fatalf("both runtimes should be needs_user: alpha=%s, beta=%s", foundAlpha.AttentionKind, foundBeta.AttentionKind)
	}

	// Step 2: Verify Client 2 (reconnected/independent client) receives reconstructed state
	list2 := fetchRuntimes(client2)
	var c2Alpha, c2Beta *registry.RuntimeSession
	for _, r := range list2 {
		if r.RuntimeID == "rt-agent-alpha" {
			c2Alpha = r
		}
		if r.RuntimeID == "rt-agent-beta" {
			c2Beta = r
		}
	}
	if c2Alpha == nil || c2Beta == nil {
		t.Fatalf("client 2 failed to reconstruct runtimes: alpha=%v, beta=%v", c2Alpha, c2Beta)
	}

	// Step 3: Respond ONLY to Project Alpha agent (simulate user approval)
	_ = reg.UpdateAttentionMeta("rt-agent-alpha", registry.AttentionUpdate{
		State:        registry.StateRunning,
		Reason:       "USER_RESPONDED",
		Kind:         "working",
		PromptKind:   "none",
		Context:      "",
		DynamicTitle: "Project Alpha · OPENCODE",
	})

	// Step 4: Verify Project Alpha is working and Project Beta remains waiting (NEEDS_USER)
	updatedList := fetchRuntimes(client1)
	for _, r := range updatedList {
		if r.RuntimeID == "rt-agent-alpha" {
			if r.AttentionKind != "working" {
				t.Fatalf("expected alpha to be working, got %s", r.AttentionKind)
			}
		}
		if r.RuntimeID == "rt-agent-beta" {
			if r.AttentionKind != "needs_user" {
				t.Fatalf("expected beta to remain needs_user, got %s", r.AttentionKind)
			}
		}
	}

	// Step 5: Terminate Project Beta agent
	_ = reg.UpdateAttentionMeta("rt-agent-beta", registry.AttentionUpdate{
		State:      registry.StateStopped,
		Reason:     "",
		Kind:       "idle",
		PromptKind: "none",
		Context:    "",
	})
	_ = reg.UpdateState("rt-agent-beta", registry.StateStopped)

	// Step 6: Verify Client 2 sees Beta's attention cleared
	finalList := fetchRuntimes(client2)
	for _, r := range finalList {
		if r.RuntimeID == "rt-agent-beta" {
			if r.AttentionKind != "idle" || r.AttentionReason != "" {
				t.Fatalf("stale attention remained for terminated beta: kind=%s reason=%s", r.AttentionKind, r.AttentionReason)
			}
		}
	}
}
