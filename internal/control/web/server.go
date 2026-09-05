package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/browser"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/nexus"
)

// DefaultPort is the default TCP port for the Web Control Center when no
// --port flag or NEXUS_WEB_PORT environment variable is provided.
const DefaultPort = 3000

type ServerOptions struct {
	Host   string
	Port   int
	NoOpen bool
	Remote bool
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
	terminalHub := NewTerminalHub(auth)

	s := &Server{
		listener:  l,
		auth:      auth,
		api:       api,
		terminal:  terminalHub,
		bootstrap: bootstrapToken,
		url:       fmt.Sprintf("http://%s:%d", opts.Host, tcpAddr.Port),
		pid:       os.Getpid(),
	}

	mux := http.NewServeMux()

	// REST API Routes
	mux.HandleFunc("/api/v1/health", api.handleHealth)
	mux.HandleFunc("/api/v1/session", api.handleSession)
	mux.HandleFunc("/api/v1/session/rotate", s.authMiddleware(s.handleSessionRotate))
	mux.HandleFunc("/api/v1/session/logout", s.authMiddleware(s.handleSessionLogout))
	mux.HandleFunc("/api/v1/workspaces", s.authMiddleware(api.handleWorkspaces))
	mux.HandleFunc("/api/v1/runtimes", s.authMiddleware(api.handleRuntimes))
	mux.HandleFunc("/api/v1/runtimes/", s.routeRuntime)
	mux.HandleFunc("/api/v1/providers", s.authMiddleware(api.handleProviders))
	mux.HandleFunc("/api/v1/profiles", s.authMiddleware(api.handleProfiles))
	mux.HandleFunc("/api/v1/events", s.authMiddleware(api.handleEvents))

	// Nexus Product API Routes (Project-first / Agent-first)
	mux.HandleFunc("/api/v1/projects", s.authMiddleware(nexusHandler.handleProjectsList))
	mux.HandleFunc("/api/v1/projects/", s.routeProject(nexusHandler))
	mux.HandleFunc("/api/v1/agents/", s.routeAgent(nexusHandler))

	// Resource Scheduler (Gate 5)
	mux.HandleFunc("/api/v1/resources", s.authMiddleware(nexusHandler.handleResourcesList))
	mux.HandleFunc("/api/v1/resources/select", s.authMiddleware(nexusHandler.handleResourceSelect))
	mux.HandleFunc("/api/v1/resources/recommend", s.authMiddleware(nexusHandler.handleResourceRecommend))

	// Maestro Assist (Gate 6)
	mux.HandleFunc("/api/v1/maestro", s.authMiddleware(nexusHandler.handleMaestroStatus))
	mux.HandleFunc("/api/v1/maestro/advice", s.authMiddleware(nexusHandler.handleMaestroAdvice))

	// WorkPlans & Intelligence (Phase C & D)
	mux.HandleFunc("/api/v1/intelligence", s.authMiddleware(nexusHandler.handleIntelligence))
	mux.HandleFunc("/api/v1/intelligence/probe", s.authMiddleware(nexusHandler.handleIntelligenceProbe))
	mux.HandleFunc("/api/v1/clarifications/", s.authMiddleware(nexusHandler.handleClarification))
	mux.HandleFunc("/api/v1/composer-sessions/", s.authMiddleware(nexusHandler.handleComposerSession))
	mux.HandleFunc("/api/v1/prompt-artifacts/", s.authMiddleware(nexusHandler.handlePromptArtifact))
	mux.HandleFunc("/api/v1/flows/decompose", s.authMiddleware(nexusHandler.handleFlowDecompose))
	mux.HandleFunc("/api/v1/plans/", s.routePlan(nexusHandler))

	// Autonomous Mission Runs (Phase F & H)
	mux.HandleFunc("/api/v1/runs", s.authMiddleware(nexusHandler.handleRunsList))
	mux.HandleFunc("/api/v1/runs/", s.routeRun(nexusHandler))
	mux.HandleFunc("/api/v1/schedules", s.authMiddleware(nexusHandler.handleMissionSchedules))

	// Missions (Gate 7 Beta)
	mux.HandleFunc("/api/v1/missions/", s.routeMission(nexusHandler))

	// System Updates (Auto-update for Nexus & Maestro)
	mux.HandleFunc("/api/v1/system/doctor", s.authMiddleware(nexusHandler.handleSystemDoctor))
	mux.HandleFunc("/api/v1/system/updates", s.authMiddleware(nexusHandler.handleSystemUpdates))
	mux.HandleFunc("/api/v1/system/update", s.authMiddleware(nexusHandler.handleSystemUpdate))

	// OS Filesystem & Discovery Routes
	mux.HandleFunc("/api/v1/fs/browse", s.authMiddleware(nexusHandler.handleFSBrowse))
	mux.HandleFunc("/api/v1/fs/scan", s.authMiddleware(nexusHandler.handleFSScan))
	mux.HandleFunc("/api/v1/fs/inspect", s.authMiddleware(nexusHandler.handleFSInspect))
	mux.HandleFunc("/api/v1/fs/mkdir", s.authMiddleware(nexusHandler.handleFSMkdir))

	// Static Files & SPA Routing
	distFS, distErr := DistFileSystem()
	var fileServer http.Handler
	if distErr == nil {
		fileServer = http.FileServer(distFS)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1. Check for one-time bootstrap token query parameter: ?token=...
		if token := r.URL.Query().Get("token"); token != "" {
			sess, ok := s.auth.ExchangeBootstrapToken(token)
			if ok && sess != nil {
				http.SetCookie(w, &http.Cookie{
					Name:     sessionCookieName,
					Value:    sess.ID,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteStrictMode,
				})
				// Redirect to clean URL without the token
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
		}

		// 2. Serve static SPA assets
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
		URL:          s.url,
		BootstrapURL: s.BootstrapURL(),
		PID:          s.pid,
		Loopback:     loopback,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "nexus web: failed to write listen state: %v\n", err)
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
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: next.ID, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
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
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	w.WriteHeader(http.StatusNoContent)
}

func projectSubroute(path string) string {
	switch {
	case strings.HasSuffix(path, "/layout"):
		return "layout"
	case strings.HasSuffix(path, "/agents"):
		return "agents"
	case strings.HasSuffix(path, "/missions"):
		return "missions"
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
		h := w.Header()
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; "+
				"font-src 'self' data:; "+
				"connect-src 'self' ws: wss:; "+
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

// routeProject dispatches project detail, layout, and agents sub-routes.
func (s *Server) routeProject(h *NexusHandler) http.HandlerFunc {
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
		// CSRF enforcement for all mutating methods (P0-2).
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			csrf := r.Header.Get(csrfHeaderName)
			if csrf == "" || csrf != sess.CSRFToken {
				writeError(w, http.StatusForbidden, "invalid CSRF token")
				return
			}
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/events"):
			h.handleProjectEvents(w, r)
		case strings.HasSuffix(r.URL.Path, "/layout"):
			h.handleProjectLayout(w, r)
		case strings.HasSuffix(r.URL.Path, "/agents"):
			if r.Method == http.MethodGet {
				h.handleAgentsList(w, r)
			} else {
				h.handleAgentCreate(w, r)
			}
		case strings.HasSuffix(r.URL.Path, "/missions"):
			if r.Method == http.MethodGet {
				h.handleMissionsList(w, r)
			} else {
				h.handleMissionCreate(w, r)
			}
		case strings.HasSuffix(r.URL.Path, "/plans"):
			h.handleProjectPlans(w, r)
		case strings.HasSuffix(r.URL.Path, "/composer-sessions"):
			h.handleProjectComposerSessions(w, r)
		case strings.HasSuffix(r.URL.Path, "/context/prepare"):
			h.handleProjectContextPrepare(w, r)
		case strings.HasSuffix(r.URL.Path, "/context"):
			h.handleProjectContext(w, r)
		case strings.HasSuffix(r.URL.Path, "/shell"):
			h.handleProjectShell(w, r)
		case strings.HasSuffix(r.URL.Path, "/open-os"):
			h.handleProjectOpenOS(w, r)
		case strings.HasSuffix(r.URL.Path, "/git/branches"):
			h.handleProjectGitBranches(w, r)
		case strings.HasSuffix(r.URL.Path, "/git/checkout"):
			h.handleProjectGitCheckout(w, r)
		default:
			h.handleProjectDetail(w, r)
		}
	}
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
		h.handleRunDetail(w, r)
	}
}

// routeAgent dispatches agent detail, actions, and the agent-scoped terminal WS.
func (s *Server) routeAgent(h *NexusHandler) http.HandlerFunc {
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
		// CSRF enforcement for all mutating methods (P0-2).
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			csrf := r.Header.Get(csrfHeaderName)
			if csrf == "" || csrf != sess.CSRFToken {
				writeError(w, http.StatusForbidden, "invalid CSRF token")
				return
			}
		}

		path := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
		parts := strings.Split(path, "/")

		// WebSocket terminal: /api/v1/agents/:id/terminal[?runtime_id=…]
		if len(parts) == 2 && parts[1] == "terminal" && r.Method == http.MethodGet {
			agentID := parts[0]
			runtimeID := strings.TrimSpace(r.URL.Query().Get("runtime_id"))
			if runtimeID != "" {
				rtSess, ok := registry.DefaultRegistry().Get(runtimeID)
				if !ok || !rtSess.HostLive() {
					// Fallback: check if the agent has a fresher live runtime generation
					if freshID, err := h.resolveAgentRuntimeID(agentID); err == nil && freshID != "" {
						runtimeID = freshID
					} else {
						writeError(w, http.StatusNotFound, "runtime not found: "+runtimeID)
						return
					}
				}
			} else {
				var err error
				runtimeID, err = h.resolveAgentRuntimeID(agentID)
				if err != nil {
					writeError(w, http.StatusNotFound, "agent has no active runtime: "+err.Error())
					return
				}
			}
			s.terminal.HandleWebSocket(w, r, agentID, runtimeID)
			return
		}

		if len(parts) >= 2 {
			switch parts[1] {
			case "start":
				h.handleAgentStart(w, r)
				return
			case "stop":
				h.handleAgentStop(w, r)
				return
			case "recover":
				h.handleAgentRecover(w, r)
				return
			case "ask":
				h.handleAgentAsk(w, r)
				return
			case "config":
				if len(parts) >= 3 {
					switch parts[2] {
					case "apply":
						h.handleAgentConfigApply(w, r)
						return
					case "impact":
						h.handleAgentConfigImpact(w, r)
						return
					}
				}
				h.handleAgentConfigGet(w, r)
				return
			}
		}
		h.handleAgentDetail(w, r)
	}
}

// routeMission dispatches mission tasks and assignments sub-routes.
func (s *Server) routeMission(h *NexusHandler) http.HandlerFunc {
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
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/missions/")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			switch parts[1] {
			case "tasks":
				h.handleMissionTaskCreate(w, r)
				return
			case "assign":
				h.handleMissionAssign(w, r)
				return
			}
		}
		h.handleMissionDetail(w, r)
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
	return fmt.Sprintf("%s/?token=%s", s.url, s.bootstrap)
}

func (s *Server) Start() error {
	return s.httpServer.Serve(s.listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
	removeListenState(s.pid)
	return s.httpServer.Shutdown(ctx)
}

func OpenBrowser(url string) error {
	return browser.OpenAppWindow(url)
}
