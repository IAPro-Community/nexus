package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/originpolicy"
	"github.com/IAPro-Community/nexus/internal/core/entitlement"
	nexuscore "github.com/IAPro-Community/nexus/internal/nexus"
)

// TunnelStatusResponse is the contract returned by tunnel status/start/stop.
type TunnelStatusResponse struct {
	Active               bool   `json:"active"`
	URL                  string `json:"url,omitempty"`
	BootstrapURL         string `json:"bootstrap_url,omitempty"`
	CloudflaredInstalled bool   `json:"cloudflared_installed"`
	CloudflaredVersion   string `json:"cloudflared_version,omitempty"`
	Error                string `json:"error,omitempty"`
}

// tunnelManager holds the runtime state for the Cloudflare Quick Tunnel.
type tunnelManager struct {
	mu          sync.Mutex
	tunnel      *Tunnel
	cancel      context.CancelFunc
	startCancel context.CancelFunc
	starting    bool
}

func (s *Server) handleTunnelStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	installed := IsCloudflaredInstalled()
	version := ""
	if installed {
		p, _ := cloudflaredPath()
		version = CloudflaredVersion(p)
	}

	tm := s.tunnelManager()
	tm.mu.Lock()
	active := tm.tunnel != nil && !tm.tunnel.stopped
	url := ""
	if active {
		url = tm.tunnel.URL
	}
	tm.mu.Unlock()

	writeJSON(w, http.StatusOK, TunnelStatusResponse{
		Active:               active,
		URL:                  url,
		CloudflaredInstalled: installed,
		CloudflaredVersion:   version,
	})
}

func (s *Server) handleTunnelStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := nexuscore.Default().RequireCapability(r.Context(), entitlement.CapRemoteAccess); err != nil {
		writeError(w, http.StatusForbidden, "remote access entitlement denied: "+err.Error())
		return
	}

	tm := s.tunnelManager()
	tm.mu.Lock()
	if tm.tunnel != nil && !tm.tunnel.stopped || tm.starting {
		tm.mu.Unlock()
		writeError(w, http.StatusConflict, "tunnel already active")
		return
	}
	tm.starting = true
	tm.mu.Unlock()

	installed := IsCloudflaredInstalled()
	version := ""
	if installed {
		p, _ := cloudflaredPath()
		version = CloudflaredVersion(p)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	tm.mu.Lock()
	tm.startCancel = cancel
	tm.mu.Unlock()
	defer func() {
		tm.mu.Lock()
		tm.startCancel = nil
		tm.mu.Unlock()
	}()

	tunnel, err := StartTunnel(ctx, s.port())
	if err != nil {
		tm.mu.Lock()
		tm.starting = false
		tm.mu.Unlock()
		writeJSON(w, http.StatusBadGateway, TunnelStatusResponse{
			CloudflaredInstalled: installed,
			CloudflaredVersion:   version,
			Error:                err.Error(),
		})
		return
	}

	// Arm tunnel auth fail-closed as soon as the public URL exists — before
	// WaitForTunnel — so bootstrap/query-token cannot stay reusable while the
	// hostname is already world-reachable.
	if tunnelHost := extractTunnelHost(tunnel.URL); tunnelHost != "" {
		originpolicy.RegisterTunnelHost(tunnelHost)
		s.tunnelHost = tunnelHost
	}
	s.auth.SetTunnelActive(true)

	tunnel.OnExit = func() {
		tm := s.tunnelManager()
		tm.mu.Lock()
		// An old cloudflared process may exit after a newer tunnel has
		// started. It must not disarm that new tunnel's authentication.
		ownsActiveTunnel := tm.tunnel == tunnel
		if ownsActiveTunnel {
			tm.tunnel = nil
			tm.cancel = nil
		}
		tm.mu.Unlock()
		if ownsActiveTunnel {
			if host := extractTunnelHost(tunnel.URL); host != "" {
				originpolicy.UnregisterTunnelHost(host)
			}
			s.auth.SetTunnelActive(false)
		}
	}

	readyCtx, readyCancel := context.WithTimeout(ctx, 60*time.Second)
	defer readyCancel()
	if err := tunnel.WaitForTunnel(readyCtx); err != nil {
		if tunnelHost := extractTunnelHost(tunnel.URL); tunnelHost != "" {
			originpolicy.UnregisterTunnelHost(tunnelHost)
		}
		s.auth.SetTunnelActive(false)
		_ = tunnel.Stop()
		tm.mu.Lock()
		tm.starting = false
		tm.mu.Unlock()
		writeJSON(w, http.StatusBadGateway, TunnelStatusResponse{
			CloudflaredInstalled: installed,
			CloudflaredVersion:   version,
			Error:                "tunnel readiness failed: " + err.Error(),
		})
		return
	}

	tm.mu.Lock()
	tm.tunnel = tunnel
	tm.cancel = tunnel.cancel
	tm.starting = false
	tm.mu.Unlock()

	writeJSON(w, http.StatusOK, TunnelStatusResponse{
		Active:               true,
		URL:                  tunnel.URL,
		CloudflaredInstalled: installed,
		CloudflaredVersion:   version,
	})
}

func (s *Server) handleTunnelStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tm := s.tunnelManager()
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.tunnel == nil || tm.tunnel.stopped {
		writeJSON(w, http.StatusOK, TunnelStatusResponse{Active: false})
		return
	}

	// Unregister tunnel host
	if tm.tunnel.URL != "" {
		if tunnelHost := extractTunnelHost(tm.tunnel.URL); tunnelHost != "" {
			originpolicy.UnregisterTunnelHost(tunnelHost)
		}
	}

	_ = tm.tunnel.Stop()
	tm.tunnel = nil
	if tm.cancel != nil {
		tm.cancel()
		tm.cancel = nil
	}

	// Inform auth manager that tunnel is no longer active.
	s.auth.SetTunnelActive(false)

	writeJSON(w, http.StatusOK, TunnelStatusResponse{Active: false})
}

// extractTunnelHost extracts the hostname from a tunnel URL.
func extractTunnelHost(tunnelURL string) string {
	if strings.HasPrefix(tunnelURL, "https://") {
		url := strings.TrimPrefix(tunnelURL, "https://")
		if idx := strings.Index(url, "/"); idx >= 0 {
			return url[:idx]
		}
		return url
	}
	return ""
}

func (s *Server) port() int {
	if s.httpServer != nil && s.httpServer.Addr != "" {
		parts := strings.Split(s.httpServer.Addr, ":")
		if len(parts) > 0 {
			var p int
			fmt.Sscanf(parts[len(parts)-1], "%d", &p)
			if p > 0 {
				return p
			}
		}
	}
	return DefaultPort
}

// handleTunnelQR mints a fresh invitation on an authenticated POST.
// GET requests must never create or reveal credentials.
func (s *Server) handleTunnelQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tm := s.tunnelManager()
	tm.mu.Lock()
	active := tm.tunnel != nil && !tm.tunnel.stopped
	tunnelURL := ""
	if active {
		tunnelURL = tm.tunnel.URL
	}
	tm.mu.Unlock()
	if !active || tunnelURL == "" {
		writeError(w, http.StatusNotFound, "no active tunnel")
		return
	}
	token, err := s.auth.IssueTunnelInvitation(extractTunnelHost(tunnelURL))
	if err != nil {
		writeError(w, http.StatusConflict, "unable to generate invitation: "+err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"url": tunnelURL,
		"bootstrap_url": tunnelURL + "/#nexus_bootstrap=" + token,
		"expires_in_seconds": int(remoteBootstrapTTL.Seconds()),
	})
}
