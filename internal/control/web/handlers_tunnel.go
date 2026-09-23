package web

import (
	"context"
	"encoding/json"
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
	bURL := ""
	if active {
		url = tm.tunnel.URL
		bURL = s.remoteBootstrapURL(url)
	}
	tm.mu.Unlock()

	writeJSON(w, http.StatusOK, TunnelStatusResponse{
		Active:               active,
		URL:                  url,
		BootstrapURL:         bURL,
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
		if host := extractTunnelHost(tunnel.URL); host != "" {
			originpolicy.UnregisterTunnelHost(host)
		}
		s.auth.SetTunnelActive(false)
		tm := s.tunnelManager()
		tm.mu.Lock()
		if tm.tunnel == tunnel {
			tm.tunnel = nil
			tm.cancel = nil
		}
		tm.mu.Unlock()
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

	bURL := s.remoteBootstrapURL(tunnel.URL)

	writeJSON(w, http.StatusOK, TunnelStatusResponse{
		Active:               true,
		URL:                  tunnel.URL,
		BootstrapURL:         bURL,
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

// remoteBootstrapURL rewrites the local bootstrap URL to use the tunnel public URL.
func (s *Server) remoteBootstrapURL(tunnelURL string) string {
	if tunnelURL == "" {
		return ""
	}
	localURL := s.url + "/#nexus_bootstrap=" + s.bootstrap
	return strings.Replace(localURL, s.url, tunnelURL, 1)
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

// handleTunnelQR returns a JSON payload with the tunnel bootstrap URL for
// QR code generation on the frontend.
func (s *Server) handleTunnelQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tm := s.tunnelManager()
	tm.mu.Lock()
	active := tm.tunnel != nil && !tm.tunnel.stopped
	url := ""
	bURL := ""
	if active {
		url = tm.tunnel.URL
		bURL = s.remoteBootstrapURL(url)
	}
	tm.mu.Unlock()

	if !active || bURL == "" {
		writeError(w, http.StatusNotFound, "no active tunnel")
		return
	}

	// Return the URL as JSON — frontend generates QR code client-side
	data, _ := json.Marshal(map[string]string{
		"url":           url,
		"bootstrap_url": bURL,
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
