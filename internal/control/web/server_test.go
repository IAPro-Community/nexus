package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

func TestMain(m *testing.M) {
	testDir, err := os.MkdirTemp("", "ai-control-web-test-*")
	if err == nil {
		_ = os.Setenv("NEXUS_DATA_DIR", testDir)
		_ = os.Setenv("AI_MANAGER_DATA_DIR", testDir)
		_ = os.Setenv("AI_CLI_DATA_DIR", testDir)
	}
	_ = os.Setenv("NEXUS_DISABLE_HOSTS_FILE", "1")

	// The web package uses the process-wide Nexus singleton. Reset it before and
	// after the suite so SQLite/background workers cannot retain nexus.db handles
	// past the test data lifetime (especially on Windows).
	nexus.ResetDefaultForTest()
	registry.ResetDefaultRegistryForTest()
	code := m.Run()
	nexus.ResetDefaultForTest()
	registry.ResetDefaultRegistryForTest()
	if testDir != "" {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(code)
}

func TestTunnelHostArmsTunnelAuthMode(t *testing.T) {
	srv, err := NewServer(ServerOptions{
		Host:       "127.0.0.1",
		Port:       0,
		NoOpen:     true,
		TunnelHost: "example.trycloudflare.com",
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })
	if !srv.auth.IsTunnelActive() {
		t.Fatal("TunnelHost must call SetTunnelActive(true) so CLI tunnel matches API auth policy")
	}
	if !srv.cookieSecure() {
		t.Fatal("tunnel-active loopback cookies must be Secure")
	}
}

func TestServer_BootstrapAndAuth(t *testing.T) {
	srv, err := NewServer(ServerOptions{
		Host: "127.0.0.1",
		Port: 0, // Random port
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	go func() {
		_ = srv.Start()
	}()
	defer srv.Shutdown(context.Background())

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// 1. Bootstrap via POST with token: should exchange and set cookie
	body := strings.NewReader(`{"token":"` + srv.bootstrap + `"}`)
	resp, err := client.Post(srv.URL()+"/api/v1/auth/bootstrap", "application/json", body)
	if err != nil {
		t.Fatalf("failed to POST bootstrap token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bootstrap POST returned %d", resp.StatusCode)
	}

	// 2. Check session endpoint
	sessResp, err := client.Get(srv.URL() + "/api/v1/session")
	if err != nil {
		t.Fatalf("failed to query session endpoint: %v", err)
	}
	defer sessResp.Body.Close()

	var sessData struct {
		Authenticated bool   `json:"authenticated"`
		CSRFToken     string `json:"csrf_token"`
	}
	if err := json.NewDecoder(sessResp.Body).Decode(&sessData); err != nil {
		t.Fatalf("failed to decode session response: %v", err)
	}

	if !sessData.Authenticated || sessData.CSRFToken == "" {
		t.Errorf("expected session to be authenticated with CSRF token, got %+v", sessData)
	}

	// 3. Query health endpoint
	healthResp, err := client.Get(srv.URL() + "/api/v1/health")
	if err != nil || healthResp.StatusCode != http.StatusOK {
		t.Errorf("health endpoint failed: %v, status: %d", err, healthResp.StatusCode)
	}
	healthResp.Body.Close()

	// 4. Query workspaces endpoint
	wsResp, err := client.Get(srv.URL() + "/api/v1/workspaces")
	if err != nil || wsResp.StatusCode != http.StatusOK {
		t.Errorf("workspaces endpoint failed: %v, status: %d", err, wsResp.StatusCode)
	}
	wsResp.Body.Close()

	// 5. Query providers endpoint
	provResp, err := client.Get(srv.URL() + "/api/v1/providers")
	if err != nil {
		t.Errorf("providers endpoint request failed: %v", err)
	} else {
		defer provResp.Body.Close()
		if provResp.StatusCode != http.StatusOK {
			t.Errorf("providers endpoint failed: status: %d", provResp.StatusCode)
		}
	}

	// 6. Test Origin enforcement
	badReq, _ := http.NewRequest(http.MethodGet, srv.URL()+"/api/v1/workspaces", nil)
	badReq.Header.Set("Origin", "http://malicious-site.com")
	badResp, err := client.Do(badReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer badResp.Body.Close()
	if badResp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for untrusted Origin, got %d", badResp.StatusCode)
	}

	// 7. Test prefix spoofed Origin enforcement
	evilReq, _ := http.NewRequest(http.MethodGet, srv.URL()+"/api/v1/workspaces", nil)
	evilReq.Header.Set("Origin", "http://localhost.evil.com")
	evilResp, err := client.Do(evilReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer evilResp.Body.Close()
	if evilResp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for prefix spoofed Origin http://localhost.evil.com, got %d", evilResp.StatusCode)
	}

	evilReq2, _ := http.NewRequest(http.MethodGet, srv.URL()+"/api/v1/workspaces", nil)
	evilReq2.Header.Set("Origin", "http://127.0.0.1.attacker.com")
	evilResp2, err := client.Do(evilReq2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer evilResp2.Body.Close()
	if evilResp2.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for prefix spoofed Origin http://127.0.0.1.attacker.com, got %d", evilResp2.StatusCode)
	}

	// 8. Test unauthenticated client: GET on API routes MUST return 401 Unauthorized
	unauthClient := &http.Client{}
	unauthEndpoints := []string{
		"/api/v1/workspaces",
		"/api/v1/runtimes",
		"/api/v1/providers",
		"/api/v1/profiles",
		"/api/v1/events",
	}
	for _, ep := range unauthEndpoints {
		resp, err := unauthClient.Get(srv.URL() + ep)
		if err != nil {
			t.Fatalf("GET %s failed: %v", ep, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for unauthenticated GET %s, got %d", ep, resp.StatusCode)
		}
	}

	// 9. Test public endpoints without authentication: health and session must succeed (200)
	pubHealthResp, err := unauthClient.Get(srv.URL() + "/api/v1/health")
	if err != nil || pubHealthResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for public health endpoint, got %d", pubHealthResp.StatusCode)
	}
	pubHealthResp.Body.Close()

	pubSessResp, err := unauthClient.Get(srv.URL() + "/api/v1/session")
	if err != nil || pubSessResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for public session endpoint, got %d", pubSessResp.StatusCode)
	}
	var unauthSessData struct {
		Authenticated bool `json:"authenticated"`
	}
	_ = json.NewDecoder(pubSessResp.Body).Decode(&unauthSessData)
	pubSessResp.Body.Close()
	if unauthSessData.Authenticated {
		t.Errorf("expected unauthenticated session for unauth client, got true")
	}
}

func TestProvidersDocsCaptureUsesSyntheticInventory(t *testing.T) {
	t.Setenv("NEXUS_DOCS_CAPTURE", "1")
	client, srv := newTestClient(t)
	resp, err := client.Get(srv.URL() + "/api/v1/providers")
	if err != nil {
		t.Fatalf("providers request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("providers returned %d", resp.StatusCode)
	}
	var providers []struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
		t.Fatalf("decode providers: %v", err)
	}
	if len(providers) != 1 || providers[0].ID != "demo-provider" || providers[0].Version != "synthetic-fixture" {
		t.Fatalf("documentation capture exposed non-synthetic providers: %+v", providers)
	}
}

func TestServerBootstrapURLKeepsTokenInFragment(t *testing.T) {
	srv, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: 0})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	defer srv.Shutdown(context.Background())
	bootstrapURL := srv.BootstrapURL()
	if !strings.Contains(bootstrapURL, "/#nexus_bootstrap=") {
		t.Fatalf("bootstrap URL must use browser fragment: %s", bootstrapURL)
	}
	if strings.Contains(bootstrapURL, "?") {
		t.Fatalf("bootstrap token must not be placed in query string: %s", bootstrapURL)
	}
}
