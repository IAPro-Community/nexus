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
		if feature := featureForProjectMutation(r.URL.Path); feature != "" {
            if optionalFeatureMutationBlocked(w, r, feature) {
                return
            }
        }
        // Collection endpoint (no project id): /api/v1/projects/summaries
		if strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"), "/") == "summaries" {
			h.handleProjectSummaries(w, r)
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/outputs"):
			h.handleProjectOutputs(w, r)
		case strings.HasSuffix(r.URL.Path, "/files"):
			h.handleProjectFiles(w, r)
		case strings.HasSuffix(r.URL.Path, "/events"):
			h.handleProjectEvents(w, r)
		case strings.HasSuffix(r.URL.Path, "/layout"):
			h.handleProjectLayout(w, r)
		case strings.Contains(r.URL.Path, "/agent-groups"):
			h.handleProjectAgentGroups(w, r)
		case strings.HasSuffix(r.URL.Path, "/director"):
			h.handleProjectDirector(w, r)
		case strings.HasSuffix(r.URL.Path, "/agents"):
			if r.Method == http.MethodGet {
				h.handleAgentsList(w, r)
			} else {
				h.handleAgentCreate(w, r)
			}
		case strings.HasSuffix(r.URL.Path, "/maestro/submissions/cancel"):
			h.handleProjectMaestroSubmissionCancel(w, r)
		// Must stay ahead of the generic "/maestro/submissions" suffix case.
		case strings.HasSuffix(r.URL.Path, "/maestro/submissions/import-usage"):
			h.handleProjectMaestroSubmissionUsageImport(w, r)
		case strings.HasSuffix(r.URL.Path, "/maestro/submissions"):
			h.handleProjectMaestroSubmissions(w, r)
		case strings.HasSuffix(r.URL.Path, "/maestro/runs"):
			h.handleProjectMaestroRunSubmit(w, r)
		case strings.HasSuffix(r.URL.Path, "/maestro/route-preview"):
			h.handleProjectMaestroRoutePreview(w, r)
		case strings.HasSuffix(r.URL.Path, "/maestro/compose-preview"):
			h.handleProjectMaestroComposePreview(w, r)
		case strings.HasSuffix(r.URL.Path, "/plans"):
			h.handleProjectPlans(w, r)
		case strings.HasSuffix(r.URL.Path, "/skills"):
			h.handleProjectSkillCatalog(w, r)
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
		case strings.Contains(r.URL.Path, "/shares/"):
			h.handleProjectShares(w, r)
		case strings.HasSuffix(r.URL.Path, "/shares"):
			h.handleProjectShares(w, r)
		case strings.HasSuffix(r.URL.Path, "/change-collisions"):
			h.handleProjectChangeCollisions(w, r)
		case strings.HasSuffix(r.URL.Path, "/release-desk"):
			h.handleProjectReleaseDesk(w, r)
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
			// An interactive terminal executes arbitrary host commands, so a
			// read-only (viewer) session may never attach.
			if !sess.CanControl() {
				writeError(w, http.StatusForbidden, "read-only session: terminal access is not allowed")
				return
			}
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
				} else if !s.terminalRuntimeOwnedByAgent(h, agentID, runtimeID) {
					// A live runtime owned by another agent must never be
					// attached through this agent-scoped route.
					writeError(w, http.StatusNotFound, "runtime not found: "+runtimeID)
					return
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
			case "director":
				h.handleAgentDirector(w, r)
				return
			case "explain", "wait":
				h.handleAgentObservation(w, r)
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

// terminalRuntimeOwnedByAgent proves that runtimeID belongs to agentID before
// an interactive attach. The live registry row is authoritative; legacy rows
// without an agent fall back to the durable runtime generation mapping and,
// as a last proof, to the agent's own current live generation.
func (s *Server) terminalRuntimeOwnedByAgent(h *NexusHandler, agentID, runtimeID string) bool {
	if agentID == "" || runtimeID == "" {
		return false
	}
	if sess, ok := s.terminal.registry.Get(runtimeID); ok && sess.AgentID != "" {
		return sess.AgentID == agentID
	}
	if h == nil || h.nexus == nil {
		return false
	}
	if owner, err := h.nexus.ResolveAgentByRuntimeID(runtimeID); err == nil && owner != "" {
		return owner == agentID
	}
	freshID, err := h.resolveAgentRuntimeID(agentID)
	return err == nil && freshID != "" && freshID == runtimeID
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
