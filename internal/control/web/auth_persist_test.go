package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoopbackSessionsSurviveRestart(t *testing.T) {
	dir := t.TempDir()
	first, token, err := NewAuthManagerWithStore("127.0.0.1", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	sess, ok := first.ExchangeBootstrapToken(token)
	if !ok || sess == nil {
		t.Fatal("bootstrap exchange failed")
	}
	if _, err := os.Stat(filepath.Join(dir, sessionsFileName)); err != nil {
		t.Fatalf("loopback exchange must write sessions.json: %v", err)
	}

	second, _, err := NewAuthManagerWithStore("127.0.0.1", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:13000/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	got := second.AuthenticateRequest(req)
	if got == nil {
		t.Fatal("loopback cookie must authenticate after process restart")
	}
	if got.CSRFToken != sess.CSRFToken {
		t.Fatalf("csrf mismatch after restore: got %s want %s", got.CSRFToken, sess.CSRFToken)
	}
}

func TestRemoteBindDoesNotRestoreSessions(t *testing.T) {
	dir := t.TempDir()
	first, token, err := NewAuthManagerWithStore("192.168.1.10", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	sess, ok := first.ExchangeBootstrapToken(token)
	if !ok || sess == nil {
		t.Fatal("bootstrap exchange failed")
	}
	if _, err := os.Stat(filepath.Join(dir, sessionsFileName)); !os.IsNotExist(err) {
		t.Fatal("remote bind must not write a session store")
	}

	second, _, err := NewAuthManagerWithStore("192.168.1.10", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, "http://192.168.1.10:13000/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	if second.AuthenticateRequest(req) != nil {
		t.Fatal("remote sessions must not survive restart")
	}
}

func TestIdlePersistedSessionIsDropped(t *testing.T) {
	dir := t.TempDir()
	auth, _, err := NewAuthManagerWithStore("127.0.0.1", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	auth.mu.Lock()
	auth.sessions["idle"] = &Session{
		ID:           "idle",
		CSRFToken:    "csrf",
		CreatedAt:    time.Now().Add(-5 * time.Hour),
		ExpiresAt:    time.Now().Add(20 * time.Hour),
		LastActiveAt: time.Now().Add(-5 * time.Hour),
	}
	auth.persistLocked()
	auth.mu.Unlock()

	restored, _, err := NewAuthManagerWithStore("127.0.0.1", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "idle"})
	if restored.AuthenticateRequest(req) != nil {
		t.Fatal("idle persisted session must not authenticate after restart")
	}
}

func TestExpiredPersistedSessionIsDropped(t *testing.T) {
	dir := t.TempDir()
	auth, _, err := NewAuthManagerWithStore("127.0.0.1", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	auth.mu.Lock()
	auth.sessions["dead"] = &Session{
		ID:           "dead",
		CSRFToken:    "csrf",
		CreatedAt:    time.Now().Add(-48 * time.Hour),
		ExpiresAt:    time.Now().Add(-time.Hour),
		LastActiveAt: time.Now().Add(-48 * time.Hour),
	}
	auth.persistLocked()
	auth.mu.Unlock()

	restored, _, err := NewAuthManagerWithStore("127.0.0.1", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "dead"})
	if restored.AuthenticateRequest(req) != nil {
		t.Fatal("expired persisted session must not authenticate")
	}
}

func TestListenStateRoundTripForLivePID(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(health.Close)
	state := ListenState{
		URL:            health.URL,
		BootstrapURL:   health.URL,
		BootstrapToken: "abc",
		PID:            os.Getpid(),
		Loopback:       true,
	}
	if err := writeListenState(state); err != nil {
		t.Fatal(err)
	}
	got, err := ReadListenState()
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != state.URL || got.BootstrapURL != state.BootstrapURL || got.BootstrapToken != state.BootstrapToken {
		t.Fatalf("listen state mismatch: %+v", got)
	}
	removeListenState(os.Getpid())
	if _, err := ReadListenState(); err == nil {
		t.Fatal("listen state should be gone after matching pid remove")
	}
}

func TestReadListenStateFallsBackToAnotherLiveServerAfterClose(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(health.Close)

	first := ListenState{
		URL:            health.URL,
		BootstrapURL:   health.URL + "/first",
		BootstrapToken: "first-token",
		PID:            os.Getpid(),
		Loopback:       true,
	}
	second := ListenState{
		URL:            health.URL,
		BootstrapURL:   health.URL + "/second",
		BootstrapToken: "second-token",
		PID:            os.Getpid() + 1,
		Loopback:       true,
	}
	if err := writeListenState(first); err != nil {
		t.Fatal(err)
	}
	if err := writeListenState(second); err != nil {
		t.Fatal(err)
	}

	removeListenState(second.PID)

	got, err := ReadListenState()
	if err != nil {
		t.Fatalf("ReadListenState() after closing newest server: %v", err)
	}
	if got.PID != first.PID || got.BootstrapURL != first.BootstrapURL || got.BootstrapToken != first.BootstrapToken {
		t.Fatalf("expected remaining live server state, got %+v", got)
	}
}

func TestReadListenStateSupportsLegacySingleFile(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(health.Close)
	state := ListenState{
		URL:            health.URL,
		BootstrapURL:   health.URL,
		BootstrapToken: "legacy-token",
		PID:            os.Getpid(),
		Loopback:       true,
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	dir := loopbackAuthStoreDir()
	if err := os.MkdirAll(dir, webAuthDirMode); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(filepath.Join(dir, listenFileName), data); err != nil {
		t.Fatal(err)
	}

	got, err := ReadListenState()
	if err != nil {
		t.Fatalf("ReadListenState() for legacy file: %v", err)
	}
	if got.PID != state.PID || got.BootstrapToken != state.BootstrapToken {
		t.Fatalf("legacy listen state mismatch: %+v", got)
	}
}

func TestReadListenStateRejectsDeadPID(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	if err := writeListenState(ListenState{
		URL:            "http://127.0.0.1:13000",
		BootstrapURL:   "http://127.0.0.1:13000",
		BootstrapToken: "abc",
		PID:            1 << 30,
		Loopback:       true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadListenState(); err == nil {
		t.Fatal("dead pid must not look like a running nexus web")
	}
}

// TestRemoteSessionsRemainEphemeralAfterTunnelStops covers a subtle transition:
// once tunnelActive becomes false, writing a local session cache must never
// accidentally serialize an earlier PUBLIC_REMOTE session.
func TestRemoteSessionsRemainEphemeralAfterTunnelStops(t *testing.T) {
	dir := t.TempDir()
	auth, _, err := NewAuthManagerWithStore("127.0.0.1", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	auth.SetTunnelActive(true)
	token, err := auth.IssueTunnelInvitation("example.trycloudflare.com")
	if err != nil {
		t.Fatal(err)
	}
	remote, ok := auth.ExchangeBootstrapTokenForHost(token, "example.trycloudflare.com")
	if !ok || remote == nil {
		t.Fatal("failed to create remote session")
	}
	auth.SetTunnelActive(false)
	local, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	restored, _, err := NewAuthManagerWithStore("127.0.0.1", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: remote.ID})
	if restored.AuthenticateRequest(req) != nil {
		t.Fatal("remote session must not persist after tunnel stops")
	}
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: local.ID})
	if restored.AuthenticateRequest(req) == nil {
		t.Fatal("valid local session should still be persisted")
	}
}

func TestRotatedRemoteSessionNeverBecomesPersistent(t *testing.T) {
	dir := t.TempDir()
	auth, _, err := NewAuthManagerWithStore("127.0.0.1", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	auth.SetTunnelActive(true)
	token, err := auth.IssueTunnelInvitation("example.trycloudflare.com")
	if err != nil {
		t.Fatal(err)
	}
	remote, ok := auth.ExchangeBootstrapTokenForHost(token, "example.trycloudflare.com")
	if !ok || remote == nil {
		t.Fatal("failed to create remote session")
	}
	auth.SetTunnelActive(false)
	rotated, err := auth.RotateSession(remote.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !rotated.NonPersistable {
		t.Fatal("rotation must preserve remote session provenance")
	}
	restored, _, err := NewAuthManagerWithStore("127.0.0.1", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: rotated.ID})
	if restored.AuthenticateRequest(req) != nil {
		t.Fatal("rotated remote session must not survive restart")
	}
}
