package web

import (
	"net/http"
	"strings"
)

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
		// Collection endpoint (no project id): /api/v1/projects/summaries
		if strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"), "/") == "summaries" {
			h.handleProjectSummaries(w, r)
			return
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
		case strings.HasSuffix(r.URL.Path, "/plans"):
			h.handleProjectPlans(w, r)
		case strings.HasSuffix(r.URL.Path, "/composer-sessions"):
			h.handleProjectComposerSessions(w, r)
		case strings.HasSuffix(r.URL.Path, "/context/prepare"):
			h.handleProjectContextPrepare(w, r)
		case strings.HasSuffix(r.URL.Path, "/context"):
			h.handleProjectContext(w, r)
		case strings.HasSuffix(r.URL.Path, "/project-intelligence/scans"):
			h.handleProjectIntelligenceScan(w, r)
		case strings.HasSuffix(r.URL.Path, "/project-intelligence"):
			h.handleProjectIntelligence(w, r)
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
				rtSess, ok := s.terminal.registry.Get(runtimeID)
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

func projectIDFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/projects/"), "/")
	if len(parts) >= 1 && strings.HasSuffix(path, "/agents") {
		return parts[0]
	}
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func agentIDFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/agents/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
