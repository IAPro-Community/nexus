package browsercontrol

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestValidURL(t *testing.T) {
	for _, raw := range []string{"http://localhost:3000", "https://example.org/path?q=1", "http://127.0.0.1:8080"} {
		if err := ValidURL(raw); err != nil {
			t.Errorf("%q should be valid: %v", raw, err)
		}
	}
	for _, raw := range []string{"", "file:///etc/passwd", "javascript:alert(1)", "http://", "https://user:pass@example.org", "about:blank", "ftp://example.org"} {
		if err := ValidURL(raw); err == nil {
			t.Errorf("%q should be rejected", raw)
		}
	}
}
func TestManagerEmptyAndClose(t *testing.T) {
	m := NewManager()
	if got := m.List(""); len(got) != 0 {
		t.Fatalf("new manager has sessions: %d", len(got))
	}
	if m.Close("not-exists") {
		t.Fatal("closing unknown session returned true")
	}
	m.Shutdown()
}

// A remote session cannot mutate the viewport while the desktop holds a lease.
func TestExclusiveControlLease(t *testing.T) {
	s := &Session{}
	if ok, _ := s.AcquireControl("desktop"); !ok {
		t.Fatal("desktop did not get lease")
	}
	if !s.CanControl("desktop") || s.CanControl("phone") {
		t.Fatal("lease is not exclusive")
	}
	if ok, _ := s.AcquireControl("phone"); ok {
		t.Fatal("phone stole active desktop lease")
	}
	s.ReleaseControl("phone")
	if !s.CanControl("desktop") {
		t.Fatal("different device released owner")
	}
	s.ReleaseControl("desktop")
	if ok, _ := s.AcquireControl("phone"); !ok {
		t.Fatal("phone cannot take released lease")
	}
	s.mu.Lock()
	s.controllerUntil = time.Now().Add(-time.Second)
	s.mu.Unlock()
	if s.CanControl("phone") {
		t.Fatal("expired lease remains active")
	}
	if ok, _ := s.AcquireControl("desktop"); !ok {
		t.Fatal("expired lease not recoverable")
	}
}

func TestChromiumSmokeOptional(t *testing.T) {
	if os.Getenv("NEXUS_BROWSER_INTEGRATION") != "1" {
		t.Skip("set NEXUS_BROWSER_INTEGRATION=1 for real Chromium smoke test")
	}
	if _, err := chromeExecutable(); err != nil {
		t.Skipf("no Chromium available: %v", err)
	}
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body><button id="action" onclick="this.innerText='Clicked'">Initial</button><h1>Nexus Browser Smoke</h1><input id="login" type="password"><button id="signIn" onclick="document.getElementById('logged').textContent='Authenticated'">Sign in</button><p id="logged"></p></body></html>`))
	}))
	defer local.Close()
	m := NewManager()
	defer m.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	info, err := m.Create(ctx, "p-test", local.URL)
	if err != nil {
		t.Fatalf("create Chromium: %v", err)
	}
	s, ok := m.Get(info.ID)
	if !ok {
		t.Fatal("created session missing")
	}
	if ok, _ := s.AcquireControl("actor"); !ok {
		t.Fatal("no control")
	}
	deadline := time.Now().Add(8 * time.Second)
	var snapshot string
	for time.Now().Before(deadline) {
		data, readErr := s.Snapshot(ctx)
		if readErr == nil {
			snapshot = string(data)
		}
		if strings.Contains(snapshot, "Nexus Browser Smoke") {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !strings.Contains(snapshot, "Nexus Browser Smoke") {
		t.Fatalf("page not loaded: %s", snapshot)
	}
	if _, err := s.Screenshot(ctx); err != nil {
		t.Fatalf("screenshot: %v", err)
	}
	frames, unsubscribe, err := s.SubscribeFrames()
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer unsubscribe()
	time.Sleep(300 * time.Millisecond)
	if _, err := s.Do(ctx, Action{Type: "reload"}); err != nil {
		t.Fatalf("reload: %v", err)
	}
	select {
	case data := <-frames:
		if len(data) < 100 {
			t.Fatal("empty or invalid CDP screencast frame")
		}
	case <-time.After(7 * time.Second):
		t.Fatal("Chromium screencast did not deliver a live frame")
	}
	if _, err := s.Do(ctx, Action{Type: "wait_selector", Selector: "#action", TimeoutMS: 4000}); err != nil {
		t.Fatalf("wait for action button: %v", err)
	}
	if _, err := s.Do(ctx, Action{Type: "viewport", Width: 390, Height: 844}); err != nil {
		t.Fatalf("mobile viewport: %v", err)
	}
	if state := s.Info(); state.Width != 390 || state.Height != 844 {
		t.Fatalf("viewport state not updated: %+v", state)
	}
	if _, err := s.Do(ctx, Action{Type: "click_selector", Selector: "#action"}); err != nil {
		t.Fatalf("semantic click: %v", err)
	}
	after, err := s.Snapshot(ctx)
	if err != nil || !strings.Contains(string(after), "Clicked") {
		t.Fatalf("semantic click did not update page: %v (%s)", err, after)
	}
	// Exercise the complete human-in-the-loop flow against real Chromium:
	// an agent requests help, cannot see the private step, and can only
	// resume after the human explicitly completes it.
	s.ReleaseControl("actor")
	grant, token, err := m.IssueAgentGrant(info.ID, "Codex smoke")
	if err != nil { t.Fatalf("grant: %v", err) }
	if _, resolved, valid := m.ResolveAgentGrant(token); !valid || resolved != s {
		t.Fatal("agent grant did not resolve to the existing browser")
	}
	if ok, reason := s.AcquireAgentControl(grant.Actor); !ok {
		t.Fatalf("agent control: %s", reason)
	}
	if _, err := s.RequestHuman(grant.Actor, "Complete test login"); err != nil {
		t.Fatalf("request human: %v", err)
	}
	if _, err := s.ScreenshotAs(ctx, grant.Actor); err == nil {
		t.Fatal("agent screenshot allowed during private human login")
	}
	if _, err := s.BeginHuman("human:integration"); err != nil {
		t.Fatalf("human takeover: %v", err)
	}
	if _, err := s.DoAs(ctx,"human:integration",Action{Type:"fill_selector", Selector:"#login", Text:"secret-123"}); err != nil {
		t.Fatalf("human password entry: %v",err)
	}
	if _, err := s.DoAs(ctx,"human:integration",Action{Type:"click_selector", Selector:"#signIn"}); err != nil {
		t.Fatalf("human sign in: %v",err)
	}
	if _, err := s.FinishHuman("human:integration"); err != nil {
		t.Fatalf("finish human: %v",err)
	}
	if ok, reason := s.AcquireAgentControl(grant.Actor); !ok {
		t.Fatalf("agent resume: %s",reason)
	}
	verified, err := s.SnapshotAs(ctx, grant.Actor)
	if err != nil || !strings.Contains(string(verified),"Authenticated") || strings.Contains(string(verified),"secret-123") {
		t.Fatalf("resume did not preserve authenticated state or leaked credentials: %v %s",err,verified)
	}
	if !m.RevokeAgentGrant(info.ID,grant.ID) { t.Fatal("revoke failed") }
	if _,_,valid:=m.ResolveAgentGrant(token);valid {t.Fatal("revoked token accepted")}
	if !m.Close(info.ID) {
		t.Fatal("session did not close")
	}
	if _, ok := m.Get(info.ID); ok {
		t.Fatal("closed session retained")
	}
}
