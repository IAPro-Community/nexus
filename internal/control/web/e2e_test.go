package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/gorilla/websocket"
)

func TestWeb_FullE2E(t *testing.T) {
	testDir := t.TempDir()
	t.Setenv("AI_MANAGER_DATA_DIR", testDir)
	t.Setenv("AI_CLI_DATA_DIR", testDir)

	// Register a fake runtime in the isolated registry
	reg := registry.DefaultRegistry()
	fakeSession := registry.RuntimeSession{
		RuntimeID:       "e2e-fake-1",
		ProviderID:      "fake",
		ProfileID:       "default",
		Workspace:       testDir,
		PID:             99999,
		HostPID:         99999,
		HostGeneration:  time.Now().UnixNano(),
		Binary:          "/usr/bin/secret-binary",
		Args:            []string{"--token=ghp_secrettoken1234567890abcdef"},
		Env:             []string{"DATABASE_PASSWORD=s3cr3tP@ssword", "SECRET_KEY=supersecretkey"},
		State:           registry.StateRunning,
		ControlLevel:    registry.ControlLevelTerminal,
		ControlEndpoint: filepath.Join(testDir, "fake.sock"),
		StartedAt:       time.Now(),
	}
	if err := reg.Register(fakeSession); err != nil {
		t.Fatalf("failed to register fake runtime: %v", err)
	}
	defer func() {
		_ = reg.UpdateState(fakeSession.RuntimeID, registry.StateStopped)
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

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// 1. Bootstrap via POST
	body := strings.NewReader(`{"token":"` + srv.bootstrap + `"}`)
	resp, err := client.Post(srv.URL()+"/api/v1/auth/bootstrap", "application/json", body)
	if err != nil {
		t.Fatalf("failed to POST bootstrap token: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bootstrap POST returned %d", resp.StatusCode)
	}

	// 2. Fetch session details (CSRF token)
	sessResp, err := client.Get(srv.URL() + "/api/v1/session")
	if err != nil {
		t.Fatalf("failed to fetch session: %v", err)
	}
	defer sessResp.Body.Close()
	var sessData struct {
		Authenticated bool   `json:"authenticated"`
		CSRFToken     string `json:"csrf_token"`
	}
	_ = json.NewDecoder(sessResp.Body).Decode(&sessData)
	if !sessData.Authenticated {
		t.Fatal("expected session to be authenticated")
	}

	// 3. Fetch runtimes list: should include our fake runtime and NOT expose secrets/env
	rtsResp, err := client.Get(srv.URL() + "/api/v1/runtimes")
	if err != nil {
		t.Fatalf("failed to fetch runtimes: %v", err)
	}
	defer rtsResp.Body.Close()
	rtsRaw, err := io.ReadAll(rtsResp.Body)
	if err != nil {
		t.Fatalf("failed to read runtimes body: %v", err)
	}
	rtsStr := string(rtsRaw)
	if strings.Contains(rtsStr, "supersecretkey") || strings.Contains(rtsStr, "s3cr3tP@ssword") || strings.Contains(rtsStr, "secret-binary") {
		t.Errorf("runtimes API response leaked secrets/environment: %s", rtsStr)
	}

	var runtimes []*registry.RuntimeSession
	_ = json.Unmarshal(rtsRaw, &runtimes)

	found := false
	for _, r := range runtimes {
		if r.RuntimeID == "e2e-fake-1" {
			found = true
			if len(r.Env) != 0 {
				t.Errorf("expected r.Env to be empty, got %+v", r.Env)
			}
			if r.Binary != "" {
				t.Errorf("expected r.Binary to be empty, got %q", r.Binary)
			}
			if len(r.Args) != 0 {
				t.Errorf("expected r.Args to be empty, got %+v", r.Args)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected to find e2e-fake-1 in runtimes, got %d runtimes", len(runtimes))
	}

	wsURL := "ws://" + srv.listener.Addr().String() + "/api/v1/runtimes/e2e-fake-1/terminal"
	parsedURL, _ := url.Parse(srv.URL())
	cookies := jar.Cookies(parsedURL)

	// 4a. Unauthenticated WebSocket dial (no cookie): MUST return 401 Unauthorized
	unauthHeader := http.Header{}
	unauthHeader.Set("Origin", parsedURL.Scheme+"://"+parsedURL.Host)
	_, badResp, err := websocket.DefaultDialer.Dial(wsURL, unauthHeader)
	if err == nil {
		t.Errorf("expected unauthenticated WebSocket dial to fail")
	} else if badResp != nil && badResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for unauthenticated WebSocket, got %d", badResp.StatusCode)
	}

	// 4b. Untrusted / spoofed Origin (http://localhost.evil.com): MUST return 403 Forbidden
	evilHeader := http.Header{}
	evilHeader.Set("Origin", "http://localhost.evil.com")
	for _, c := range cookies {
		evilHeader.Add("Cookie", c.String())
	}
	_, evilResp, err := websocket.DefaultDialer.Dial(wsURL, evilHeader)
	if err == nil {
		t.Errorf("expected WebSocket dial with untrusted origin to fail")
	} else if evilResp != nil && evilResp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for untrusted origin, got %d", evilResp.StatusCode)
	}

	// 4c. Missing Origin header on WebSocket upgrade: MUST return 403 Forbidden
	noOriginHeader := http.Header{}
	for _, c := range cookies {
		noOriginHeader.Add("Cookie", c.String())
	}
	_, noOriginResp, err := websocket.DefaultDialer.Dial(wsURL, noOriginHeader)
	if err == nil {
		t.Errorf("expected WebSocket dial without Origin to fail")
	} else if noOriginResp != nil && noOriginResp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for missing Origin on WebSocket, got %d", noOriginResp.StatusCode)
	}

	// 4d. Authenticated dial with valid Origin: auth/origin pass, but a registry
	// row whose PID is dead must not upgrade (black xterm after reboot).
	authHeader := http.Header{}
	authHeader.Set("Origin", parsedURL.Scheme+"://"+parsedURL.Host)
	for _, c := range cookies {
		authHeader.Add("Cookie", c.String())
	}
	_, deadResp, err := websocket.DefaultDialer.Dial(wsURL, authHeader)
	if err == nil {
		t.Fatal("expected WebSocket dial to a dead runtime host to fail")
	}
	if deadResp == nil || deadResp.StatusCode != http.StatusNotFound {
		got := 0
		if deadResp != nil {
			got = deadResp.StatusCode
		}
		t.Errorf("expected 404 for dead runtime host, got %d (%v)", got, err)
	}
}
