package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestAttachLoopbackSessionReusesWebSession(t *testing.T) {
	srv := newStartedTestServer(t)

	sess, err := AttachLoopbackSession(ListenState{
		URL:            srv.URL(),
		BootstrapToken: srv.bootstrap,
		Loopback:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sess.ID == "" || sess.CSRFToken == "" {
		t.Fatalf("expected authenticated desktop session, got %+v", sess)
	}
}

func TestAttachLoopbackSessionRejectsDifferentBootstrapOrigin(t *testing.T) {
	sess, err := AttachLoopbackSession(ListenState{
		URL:            "http://127.0.0.1:13000",
		BootstrapToken: "local",
		Loopback:       true,
	})
	if err == nil || sess != nil {
		t.Fatal("expected bootstrap origin mismatch to be rejected")
	}
}

func TestLoopbackBackendProxyRewritesWailsOrigin(t *testing.T) {
	srv := newStartedTestServer(t)
	sess, err := AttachLoopbackSession(ListenState{
		URL:            srv.URL(),
		BootstrapToken: srv.bootstrap,
		Loopback:       true,
	})
	if err != nil {
		t.Fatal(err)
	}

	proxy, err := NewLoopbackBackendProxy(srv.URL())
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	req.Header.Set("Origin", "wails://wails")
	req.Header.Set("Authorization", "Bearer "+sess.ID)
	resp := httptest.NewRecorder()
	proxy.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected proxied session 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Authenticated {
		t.Fatal("expected proxied request to remain authenticated")
	}
}

func TestLoopbackBackendHandlerServesLocalSPAForNavigation(t *testing.T) {
	srv := newStartedTestServer(t)
	handler, err := NewLoopbackBackendHandler(srv.URL(), fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>desktop-shell</html>")},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/p/project-1/terminals", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected local SPA fallback 200, got %d", resp.Code)
	}
	if got := resp.Body.String(); got != "<html>desktop-shell</html>" {
		t.Fatalf("expected local index document, got %q", got)
	}
}
