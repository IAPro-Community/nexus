package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/browser"
	"github.com/IAPro-Community/nexus/internal/control/originpolicy"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

// DefaultPort is the default TCP port for the Web Control Center when no
// --port flag or NEXUS_WEB_PORT environment variable is provided.
const DefaultPort = 13000

type ServerOptions struct {
	Host       string
	Port       int
	NoOpen     bool
	Remote     bool
	TunnelHost string // public hostname of Cloudflare Quick Tunnel (e.g., xxx.trycloudflare.com)
}

type Server struct {
	httpServer *http.Server
	listener   net.Listener
	auth       *AuthManager
	api        *APIHandler
	terminal   *TerminalHub
	bootstrap  string
	url        string
	pid        int
	hostname   string // registered /etc/hosts entry (empty if none)
	loopback   bool   // true when bound to localhost/127.0.0.1
	tunnelHost string // Cloudflare Quick Tunnel public hostname
	tunnel     *tunnelManager
	tunnelMu   sync.Mutex
}

func (s *Server) tunnelManager() *tunnelManager {
	s.tunnelMu.Lock()
	defer s.tunnelMu.Unlock()
	if s.tunnel == nil {
		s.tunnel = &tunnelManager{}
	}
	return s.tunnel
}

// cookieSecure returns the Secure flag for session cookies. Non-loopback
// binds (e.g. --remote on a private IP) set Secure to protect cookies
// traversing the network in plaintext. When a Cloudflare Quick Tunnel is
// active, cookies must also be Secure even on loopback because traffic
// traverses the internet.
func (s *Server) cookieSecure() bool {
	if s.auth == nil {
		return !s.loopback
	}
	return s.auth.RequiresSecureCookie()
}

func NewServer(opts ServerOptions) (*Server, error) {
	if opts.Host == "" {
		opts.Host = "127.0.0.1"
	}
	if opts.Port < 0 || opts.Port > 65535 {
		opts.Port = DefaultPort
	}

	// Enforce the loopback-default binding policy before opening any socket.
	if err := ValidateBind(opts.Host, opts.Remote); err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to %s: %w", addr, err)
	}

	tcpAddr := l.Addr().(*net.TCPAddr)
	portStr := strconv.Itoa(tcpAddr.Port)

	storeDir := ""
	loopback := isLoopbackHost(opts.Host)
	if loopback {
		storeDir = loopbackAuthStoreDir()
	}

	auth, bootstrapToken, err := NewAuthManagerWithStore(opts.Host, portStr, storeDir)
	if err != nil {
		_ = l.Close()
		return nil, fmt.Errorf("failed to initialize auth manager: %w", err)
	}

	api := NewAPIHandler(auth)
	nexusHandler := NewNexusHandler(auth)
	nexusHandler.setHostFilesystemEnabled(hostFilesystemEnabled(opts.Host))
	terminalHub := NewTerminalHub(auth, nexus.Default().RuntimeRegistry())
	terminalHub.SetNexus(nexus.Default())

	s := &Server{
		listener:  l,
		auth:      auth,
		api:       api,
		terminal:  terminalHub,
		bootstrap: bootstrapToken,
		url:       fmt.Sprintf("http://%s:%d", opts.Host, tcpAddr.Port),
		pid:       os.Getpid(),
		loopback:  loopback,
		tunnel:    &tunnelManager{},
	}

	mux := http.NewServeMux()
	registerRoutes(mux, routeDependencies{server: s, api: api, nexusHandler: nexusHandler})

	// Static Files & SPA Routing
	distFS, distErr := DistFileSystem()
	var fileServer http.Handler
	if distErr == nil {
		fileServer = http.FileServer(distFS)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1. Serve static SPA assets
		if fileServer != nil {
			// Disable client caching for index.html, bundle.js, bundle.css during development/live use
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")

			// If file exists serve it, otherwise serve index.html (SPA routing)
			f, err := distFS.Open(strings.TrimPrefix(r.URL.Path, "/"))
			if err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
			// Fallback to index.html
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<h1>IAPro Nexus Workspace OS</h1><p>Web frontend initializing...</p>"))
	})

	s.httpServer = &http.Server{
		Handler:     s.withSecurityHeaders(mux),
		ReadTimeout: 15 * time.Second,
		// Composer auto_plan may take up to ~90s (CLI oneshot); keep headroom for JSON write.
		WriteTimeout: 120 * time.Second,
	}

	if err := writeListenState(ListenState{
		URL:            s.url,
		BootstrapURL:   s.BootstrapURL(),
		BootstrapToken: s.bootstrap,
		PID:            s.pid,
		Loopback:       loopback,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "nexus web: failed to write listen state: %v\n", err)
	}

	// Register nexus.dev in /etc/hosts for local name resolution.
	// Failure is non-fatal: the server remains usable via IP.
	if err := EnsureNexusHostsEntry(); err != nil {
		fmt.Fprintf(os.Stderr, "nexus web: could not register nexus.dev in /etc/hosts: %v\n", err)
	} else {
		s.hostname = nexusHostname
	}

	// Register Cloudflare tunnel host for origin validation and arm tunnel
	// auth mode so CLI `nexus tunnel` matches API StartTunnel (one-time
	// bootstrap, no query-token WS, Secure cookies).
	if opts.TunnelHost != "" {
		s.tunnelHost = opts.TunnelHost
		originpolicy.RegisterTunnelHost(opts.TunnelHost)
		s.auth.SetTunnelActive(true)
	}

	return s, nil
}

func (s *Server) handleSessionRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	old := s.auth.AuthenticateRequest(r)
	if old == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	next, err := s.auth.RotateSession(old.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to rotate session")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: next.ID, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: s.cookieSecure()})
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "csrf_token": next.CSRFToken, "expires_at": next.ExpiresAt, "idle_timeout": int(sessionIdleTTL.Seconds())})
}

func (s *Server) handleSessionLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sess := s.auth.AuthenticateRequest(r)
	if sess == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	s.auth.RevokeSession(sess.ID)
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1, Expires: time.Unix(1, 0), Secure: s.cookieSecure()})
	s.clearLoopbackDeviceCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func projectSubroute(path string) string {
	switch {
	case strings.HasSuffix(path, "/layout"):
		return "layout"
	case strings.HasSuffix(path, "/agents"):
		return "agents"
	case strings.HasSuffix(path, "/plans"):
		return "plans"
	case strings.HasSuffix(path, "/open-os"):
		return "open-os"
	case strings.HasSuffix(path, "/git/branches"):
		return "git-branches"
	case strings.HasSuffix(path, "/git/checkout"):
		return "git-checkout"
	default:
		return "detail"
	}
}

// withSecurityHeaders applies defense-in-depth HTTP security headers to every
// response, including CSP, MIME sniffing prevention and framing protection.
func (s *Server) withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && originpolicy.Validate(r.Host, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-CSRF-Token, X-Nexus-Session, Accept")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		h := w.Header()
		h.Set("Content-Security-Policy",
			"default-src 'self' wails:; "+
				"script-src 'self' wails:; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: wails:; "+
				"font-src 'self' data:; "+
				"connect-src 'self' ws: wss: wails:; "+
				"base-uri 'self'; "+
				"form-action 'self'; "+
				"frame-ancestors 'none'; "+
				"object-src 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) routeRuntime(w http.ResponseWriter, r *http.Request) {
	// Check for WebSocket terminal route: /api/v1/runtimes/:id/terminal
	if strings.HasSuffix(r.URL.Path, "/terminal") {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/runtimes/"), "/")
		if len(parts) == 2 && parts[1] == "terminal" {
			if !s.auth.ValidateOrigin(r) {
				writeError(w, http.StatusForbidden, "invalid origin")
				return
			}
			if s.auth.AuthenticateRequest(r) == nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			runtimeID := parts[0]
			agentID, err := nexus.Default().ResolveAgentByRuntimeID(runtimeID)
			if err != nil || agentID == "" {
				agentID = runtimeID
			}
			s.terminal.HandleWebSocket(w, r, agentID, runtimeID)
			return
		}
	}

	// Normal REST API runtime action
	s.authMiddleware(s.api.handleRuntimeDetail)(w, r)
}

// routePlan dispatches plan detail, compile, and run routes.
func (s *Server) routePlan(h *NexusHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.ValidateOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid origin")
			return
		}
		sess := s.auth.AuthenticateRequest(r)
		if sess == nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			csrf := r.Header.Get(csrfHeaderName)
			if csrf == "" || csrf != sess.CSRFToken {
				writeError(w, http.StatusForbidden, "invalid CSRF token")
				return
			}
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/delegation/approve"):
			h.handlePlanDelegationApprove(w, r)
		case strings.HasSuffix(r.URL.Path, "/compile"):
			h.handlePlanCompile(w, r)
		case strings.HasSuffix(r.URL.Path, "/run"):
			h.handlePlanRun(w, r)
		case strings.HasSuffix(r.URL.Path, "/restore"):
			h.handlePlanRestore(w, r)
		case strings.HasSuffix(r.URL.Path, "/diff"):
			h.handlePlanDiff(w, r)
		case strings.HasSuffix(r.URL.Path, "/leader"):
			h.handleFlowLeader(w, r)
		case strings.HasSuffix(r.URL.Path, "/clone"):
			h.handleFlowClone(w, r)
		case strings.HasSuffix(r.URL.Path, "/preflight"):
			h.handleFlowPreflight(w, r)
		default:
			h.handlePlanDetail(w, r)
		}
	}
}

// routeRun dispatches autonomous mission run detail and step actions.
func (s *Server) routeRun(h *NexusHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.ValidateOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid origin")
			return
		}
		sess := s.auth.AuthenticateRequest(r)
		if sess == nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			csrf := r.Header.Get(csrfHeaderName)
			if csrf == "" || csrf != sess.CSRFToken {
				writeError(w, http.StatusForbidden, "invalid CSRF token")
				return
			}
		}
		path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/runs/"), "/")
		parts := strings.Split(path, "/")
		if len(parts) == 2 && strings.EqualFold(parts[1], "routing") {
			h.handleRunRouting(w, r, parts[0])
			return
		}
		if len(parts) == 2 && strings.EqualFold(parts[1], "validation-evidence") {
			h.handleRunValidationEvidence(w, r, parts[0])
			return
		}
		h.handleRunDetail(w, r)
	}
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.ValidateOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid origin")
			return
		}

		// Enforce authentication on all API routes, including GET requests
		sess := s.auth.AuthenticateRequest(r)
		if sess == nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		// Non-GET requests require CSRF token validation
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			csrf := r.Header.Get(csrfHeaderName)
			if csrf == "" || csrf != sess.CSRFToken {
				writeError(w, http.StatusForbidden, "invalid CSRF token")
				return
			}
		}

		next(w, r)
	}
}

func (s *Server) URL() string {
	return s.url
}

func (s *Server) BootstrapURL() string {
	// The token lives in the browser fragment. The SPA exchanges it through
	// POST, so it never travels in an HTTP request URL or server access log.
	return s.url + "/#nexus_bootstrap=" + s.bootstrap
}

// BootstrapToken returns the bootstrap token for trusted local programmatic
// clients. Browser launches carry it only in a fragment, which is removed by
// the SPA before normal navigation and is never sent in an HTTP request.
func (s *Server) BootstrapToken() string {
	return s.bootstrap
}

func (s *Server) Start() error {
	return s.httpServer.Serve(s.listener)
}

func (s *Server) Handler() http.Handler {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Handler
}

func (s *Server) CreateDesktopSession() (*Session, error) {
	sess, err := s.auth.CreateSession()
	if err != nil {
		return nil, err
	}
	s.auth.SetDesktopSession(sess)
	return sess, nil
}

// handleAuthBootstrap exchanges a one-time bootstrap token sent via POST body
// for a session cookie. The token must never appear in URLs, logs, or browser
// history.
func (s *Server) handleAuthBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		if !originpolicy.Validate(r.Host, origin) {
			writeError(w, http.StatusForbidden, "invalid origin")
			return
		}
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || body.Token == "" {
		writeError(w, http.StatusBadRequest, "missing bootstrap token")
		return
	}
	sess, ok := s.auth.ExchangeBootstrapToken(body.Token)
	if !ok || sess == nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired bootstrap token")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.cookieSecure(),
	})
	s.setLoopbackDeviceCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"csrf_token":    sess.CSRFToken,
		"expires_at":    sess.ExpiresAt,
	})
}

// handleAuthLoopbackReauth mints a fresh session for a previously paired
// loopback browser (HttpOnly device cookie) when the fragment bootstrap was
// lost. The process bootstrap secret is never accepted as an anonymous
// capability. Remote / tunnel binds never expose this path.
func (s *Server) handleAuthLoopbackReauth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.auth == nil || !s.auth.isLoopbackListen() || s.auth.IsTunnelActive() {
		writeError(w, http.StatusForbidden, "loopback reauth unavailable")
		return
	}
	if !clientIsLoopback(r) {
		writeError(w, http.StatusForbidden, "loopback client required")
		return
	}
	if !requireExactOrigin(r) || !sameOriginFetch(r) {
		writeError(w, http.StatusForbidden, "invalid origin")
		return
	}
	deviceCookie, err := r.Cookie(deviceCookieName)
	if err != nil || deviceCookie == nil || !s.auth.MatchLoopbackDeviceToken(deviceCookie.Value) {
		writeError(w, http.StatusForbidden, "loopback device pairing required")
		return
	}
	sess, err := s.auth.CreateSession()
	if err != nil || sess == nil {
		writeError(w, http.StatusUnauthorized, "unable to mint loopback session")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.cookieSecure(),
	})
	s.setLoopbackDeviceCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"csrf_token":    sess.CSRFToken,
		"expires_at":    sess.ExpiresAt,
		"idle_timeout":  int(sessionIdleTTL.Seconds()),
	})
}

func (s *Server) setLoopbackDeviceCookie(w http.ResponseWriter) {
	if s.auth == nil {
		return
	}
	token, err := s.auth.IssueLoopbackDeviceToken()
	if err != nil || token == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     deviceCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.cookieSecure(),
		MaxAge:   int(deviceCookieTTL.Seconds()),
	})
}

func (s *Server) clearLoopbackDeviceCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     deviceCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.cookieSecure(),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
}

func clientIsLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) handleDesktopBootstrap(w http.ResponseWriter, r *http.Request) {
	if !originpolicy.IsTrustedDesktopRequest(r.Host, r.Header.Get("Origin"), r.Header.Get("Referer")) {
		writeError(w, http.StatusForbidden, "desktop origin required")
		return
	}

	desktopSess := s.auth.GetDesktopSession()
	if desktopSess == nil {
		writeError(w, http.StatusNotFound, "desktop session not available")
		return
	}
	// Origin/Referer identify the Wails document but are forgeable by any local
	// process talking to loopback. Require the pre-provisioned desktop bearer as
	// a second factor before returning session material.
	sess := s.auth.AuthenticateRequest(r)
	if sess == nil || sess.ID != desktopSess.ID {
		writeError(w, http.StatusUnauthorized, "desktop session authentication required")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.cookieSecure(),
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"serverUrl":     s.url,
		"sessionToken":  sess.ID,
		"csrfToken":     sess.CSRFToken,
		"authenticated": true,
	})
}

func (s *Server) Shutdown(ctx context.Context) error {
	removeListenState(s.pid)
	if tunnel := s.tunnelManager(); tunnel != nil {
		tunnel.mu.Lock()
		active := tunnel.tunnel
		tunnel.tunnel = nil
		tunnel.starting = false
		startCancel := tunnel.startCancel
		tunnel.startCancel = nil
		tunnel.mu.Unlock()
		if startCancel != nil {
			startCancel()
		}
		if active != nil {
			if host := extractTunnelHost(active.URL); host != "" {
				originpolicy.UnregisterTunnelHost(host)
			}
			_ = active.Stop()
		}
	}
	if s.hostname != "" {
		if err := RemoveHostsEntry(s.hostname, nexusIP); err != nil {
			fmt.Fprintf(os.Stderr, "nexus web: could not remove %s from /etc/hosts: %v\n", s.hostname, err)
		}
	}
	if s.tunnelHost != "" {
		originpolicy.UnregisterTunnelHost(s.tunnelHost)
	}
	if s.auth != nil {
		s.auth.SetTunnelActive(false)
		if sess := s.auth.GetDesktopSession(); sess != nil {
			s.auth.RevokeSession(sess.ID)
		}
	}
	return s.httpServer.Shutdown(ctx)
}

func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", "-n", "-a", "Google Chrome", "--args", "--app="+url).Start()
	default:
		// Chromium --app= reuses the existing app window by origin and drops the
		// URL fragment. Bootstrap tokens live in #nexus_bootstrap=…, so auth
		// launches must open a normal browser window/tab instead.
		if strings.Contains(url, "nexus_bootstrap=") {
			return browser.Open([]string{url})
		}
		return browser.OpenAppWindow(url)
	}
}
