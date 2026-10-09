package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/originpolicy"
)

func TestTunnelStopCancelsPendingStartBeforePublication(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	startCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tm := &tunnelManager{starting: true, startCancel: cancel}
	srv := &Server{auth: auth, tunnel: tm}

	statusRec := httptest.NewRecorder()
	srv.handleTunnelStatus(statusRec, httptest.NewRequest(http.MethodGet, "/api/v1/tunnel/status", nil))
	var status TunnelStatusResponse
	if err := json.Unmarshal(statusRec.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if !status.Starting || status.Active {
		t.Fatalf("pending tunnel must be reported as starting, not active: %+v", status)
	}

	rec := httptest.NewRecorder()
	srv.handleTunnelStop(rec, httptest.NewRequest(http.MethodPost, "/api/v1/tunnel/stop", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("stopping pending tunnel returned %d: %s", rec.Code, rec.Body.String())
	}
	if startCtx.Err() != context.Canceled {
		t.Fatal("Stop must cancel a startup before cloudflared is published")
	}
	tm.mu.Lock()
	stillStarting := tm.starting
	activeTunnel := tm.tunnel
	tm.mu.Unlock()
	if !stillStarting || activeTunnel != nil {
		t.Fatal("pending start must remain serialized until its handler unwinds")
	}
	if auth.IsTunnelActive() {
		t.Fatal("canceled pre-publication startup must not enable remote access")
	}
}

func TestTunnelStopCancelsReadinessAndRevokesExposure(t *testing.T) {
	const host = "readiness.trycloudflare.com"
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	auth.SetTunnelActive(true)
	originpolicy.RegisterTunnelHost(host)
	defer originpolicy.UnregisterTunnelHost(host)

	startCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tunnel := &Tunnel{URL: "https://" + host}
	tm := &tunnelManager{starting: true, startCancel: cancel, tunnel: tunnel}
	srv := &Server{auth: auth, tunnel: tm}

	rec := httptest.NewRecorder()
	srv.handleTunnelStop(rec, httptest.NewRequest(http.MethodPost, "/api/v1/tunnel/stop", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("stopping readiness tunnel returned %d: %s", rec.Code, rec.Body.String())
	}
	if startCtx.Err() != context.Canceled {
		t.Fatal("Stop must also interrupt the readiness probe")
	}
	if tunnelIsActive(tunnel) || auth.IsTunnelActive() {
		t.Fatal("stopping readiness must close cloudflared and revoke remote authentication")
	}
	tm.mu.Lock()
	owned := tm.tunnel
	tm.mu.Unlock()
	if owned != nil {
		t.Fatal("stopped tunnel must not remain registered as active")
	}
}
