package web

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/buildinfo"
	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/control/workspace"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	coreprovider "github.com/IAPro-Community/nexus/internal/core/provider"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/profile"
)

type APIHandler struct {
	auth     *AuthManager
	reg      *registry.Registry
	runtimes *nexus.RuntimeApplicationService
	drivers  *driver.Registry
	quotaEng *quota.Engine
}

func (h *APIHandler) findProvider(id string) (driver.ControlDriver, bool) {
	for _, candidate := range h.drivers.List() {
		if strings.EqualFold(candidate.ProviderID(), id) {
			return candidate, true
		}
	}
	return nil, false
}

func NewAPIHandler(auth *AuthManager) *APIHandler {
	reg := registry.DefaultRegistry()
	drivers := driver.DefaultRegistry()
	launch := launcher.Default()
	return &APIHandler{
		auth:     auth,
		reg:      reg,
		runtimes: nexus.NewRuntimeApplicationService(reg, launch, drivers),
		drivers:  drivers,
		quotaEng: quota.NewEngine(5 * time.Minute),
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, APIError{
		Error: security.Redact(msg),
		Code:  stableErrorCode(status, msg),
	})
}

func stableErrorCode(status int, message string) string {
	lower := strings.ToLower(message)
	known := []struct {
		fragment string
		code     string
	}{
		{"intervention_already_resolved", "INTERVENTION_ALREADY_RESOLVED"},
		{"stale_intervention", "STALE_INTERVENTION"},
		{"unknown_external_outcome", "UNKNOWN_EXTERNAL_OUTCOME"},
		{"dispatch outcome is unknown", "UNKNOWN_EXTERNAL_OUTCOME"},
		{"invalid_intervention_option", "INVALID_INTERVENTION_OPTION"},
		{"intervention_policy_denied", "INTERVENTION_POLICY_DENIED"},
		{"quota", "QUOTA_UNKNOWN"},
		{"rate limit", "RATE_LIMITED"},
	}
	for _, candidate := range known {
		if strings.Contains(lower, candidate.fragment) {
			return candidate.code
		}
	}
	if strings.Contains(lower, "not found") {
		for _, candidate := range []struct {
			fragment string
			code     string
		}{
			{"project", "PROJECT_NOT_FOUND"},
			{"session", "SESSION_NOT_FOUND"},
			{"runtime", "RUNTIME_NOT_RUNNING"},
			{"mission", "MISSION_NOT_FOUND"},
		} {
			if strings.Contains(lower, candidate.fragment) {
				return candidate.code
			}
		}
		return "NOT_FOUND"
	}
	if strings.Contains(lower, "provider") && strings.Contains(lower, "unavailable") {
		return "PROVIDER_UNAVAILABLE"
	}
	if strings.Contains(lower, "workspace") && strings.Contains(lower, "invalid") {
		return "WORKSPACE_INVALID"
	}
	switch status {
	case http.StatusBadRequest:
		return "INVALID_REQUEST"
	case http.StatusUnauthorized:
		return "AUTH_REQUIRED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusTooManyRequests:
		return "RATE_LIMITED"
	case http.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	case http.StatusInternalServerError:
		return "INTERNAL_ERROR"
	default:
		return "HTTP_ERROR"
	}
}

func sanitizeSession(s registry.RuntimeSession) registry.RuntimeSession {
	s.Env = nil
	s.Args = nil
	s.Binary = ""
	return s
}

// Health Handler
func (h *APIHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"version":   buildinfo.Version,
	})
}

// handleSystemInfo exposes the stable API metadata needed by clients to
// negotiate capabilities without duplicating provider knowledge. It is
// intentionally read-only and derives capabilities from the existing driver
// registry rather than adding another provider switch.
func (h *APIHandler) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	providerIDs := make([]string, 0)
	capabilities := make(map[string]driver.ControlCapabilities)
	for _, registered := range h.drivers.List() {
		providerID := registered.ProviderID()
		providerIDs = append(providerIDs, providerID)
		capabilities[providerID] = registered.Capabilities(r.Context(), model.Profile{Provider: providerID})
	}
	sort.Strings(providerIDs)

	writeJSON(w, http.StatusOK, newSystemInfoResponse(capabilities, providerIDs))
}

// Session Handler (checks authentication status & returns CSRF token)
func (h *APIHandler) handleSession(w http.ResponseWriter, r *http.Request) {
	sess := h.auth.AuthenticateRequest(r)
	if sess == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": false,
		})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.auth != nil && (h.auth.IsTunnelActive() || !isLoopbackHost(h.auth.listenHost)),
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"csrf_token":    sess.CSRFToken,
		"expires_at":    sess.ExpiresAt,
		"idle_timeout":  int(sessionIdleTTL.Seconds()),
	})
}

// Workspaces / Projects Handler
func (h *APIHandler) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	wsStore := workspace.DefaultStore()

	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, wsStore.List())
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Path string `json:"path"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		proj, err := wsStore.Add(req.Path, req.Name)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, proj)
		return
	}

	if r.Method == http.MethodDelete {
		idOrPath := r.URL.Query().Get("path")
		if idOrPath == "" {
			idOrPath = r.URL.Query().Get("id")
		}
		if idOrPath == "" {
			writeError(w, http.StatusBadRequest, "missing path or id query param")
			return
		}
		if err := wsStore.Remove(idOrPath); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// Runtimes List & Start Handler
func (h *APIHandler) handleRuntimes(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		all, err := h.runtimes.List(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		sanitized := make([]registry.RuntimeSession, len(all))
		for i, s := range all {
			clean := sanitizeSession(s)
			if agentID, err := nexus.Default().ResolveAgentByRuntimeID(s.RuntimeID); err == nil {
				clean.AgentID = agentID
			}
			sanitized[i] = clean
		}
		writeJSON(w, http.StatusOK, sanitized)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Title      string   `json:"title"`
			ProviderID string   `json:"provider"`
			ProfileID  string   `json:"profile"`
			Workspace  string   `json:"workspace"`
			Args       []string `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Workspace == "" {
			req.Workspace, _ = os.Getwd()
		}
		workspace.DefaultStore().Touch(req.Workspace)

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		sess, err := h.runtimes.Start(ctx, launcher.LaunchOptions{
			Title:      req.Title,
			ProviderID: req.ProviderID,
			ProfileID:  req.ProfileID,
			Workspace:  req.Workspace,
			Args:       req.Args,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		cleanSess := sanitizeSession(*sess)
		writeJSON(w, http.StatusCreated, cleanSess)
		return
	}

	if r.Method == http.MethodDelete {
		cleaned, purged, err := h.runtimes.Cleanup(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"cleaned": cleaned,
			"purged":  purged,
		})
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// Runtime Detail, Stop, Handoff, Continue Handlers
func (h *APIHandler) handleRuntimeDetail(w http.ResponseWriter, r *http.Request) {
	// Path: /api/v1/runtimes/<id>[/<action>]
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/runtimes/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "missing runtime ID")
		return
	}

	runtimeID := parts[0]
	sess, err := h.runtimes.Get(r.Context(), runtimeID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// GET detail
	if len(parts) == 1 && r.Method == http.MethodGet {
		_, caps, detailErr := h.runtimes.Detail(r.Context(), runtimeID)
		if detailErr != nil {
			writeError(w, http.StatusNotFound, detailErr.Error())
			return
		}
		var effCaps *driver.EffectiveCapabilities
		if caps != nil {
			effCaps = caps
		}
		cleanSess := sanitizeSession(sess)
		writeJSON(w, http.StatusOK, RuntimeDetailResponse{
			Session:      cleanSess,
			Capabilities: effCaps,
		})
		return
	}

	// DELETE runtime record
	if len(parts) == 1 && r.Method == http.MethodDelete {
		_ = h.runtimes.Delete(r.Context(), runtimeID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}

	// Actions: stop, handoff, continue
	if len(parts) == 2 && r.Method == http.MethodPost {
		action := parts[1]
		switch action {
		case "stop":
			_ = h.runtimes.Stop(r.Context(), runtimeID)
			// Wait briefly for the host to reap the child (shells SIGKILL after 250ms).
			if err := h.runtimes.WaitForExit(r.Context(), runtimeID, 1500*time.Millisecond); err != nil {
				writeError(w, http.StatusRequestTimeout, err.Error())
				return
			}
			_ = h.runtimes.MarkStopped(r.Context(), runtimeID)
			writeJSON(w, http.StatusOK, map[string]any{
				"ok": true, "code": "STOP_REQUESTED", "runtime_id": runtimeID,
				"action": "stop", "state": string(registry.StateStopped),
				"message": "runtime stopped", "correlation_id": sess.LineageID,
				"status": "stopped",
			})
			return

		case "handoff":
			var payload struct {
				Target string `json:"target"` // provider:profile
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Target == "" {
				writeError(w, http.StatusBadRequest, "missing target profile")
				return
			}
			newSess, err := h.runtimes.AccountHandoff(r.Context(), runtimeID, payload.Target)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			cleanSess := sanitizeSession(*newSess)
			writeJSON(w, http.StatusOK, cleanSess)
			return

		case "continue":
			var payload struct {
				TargetProvider string `json:"provider"`
				TargetProfile  string `json:"profile"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.TargetProvider == "" {
				writeError(w, http.StatusBadRequest, "missing target provider")
				return
			}
			newSess, err := h.runtimes.ContextHandoff(r.Context(), runtimeID, payload.TargetProvider, payload.TargetProfile)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			cleanSess := sanitizeSession(*newSess)
			writeJSON(w, http.StatusOK, cleanSess)
			return

		case "title":
			var payload struct {
				Title string `json:"title"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeError(w, http.StatusBadRequest, "invalid payload")
				return
			}
			if err := h.runtimes.UpdateTitle(r.Context(), runtimeID, payload.Title); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "title": payload.Title})
			return

		case "respond":
			var payload struct {
				Input string `json:"input"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeError(w, http.StatusBadRequest, "invalid payload")
				return
			}
			err := h.runtimes.Respond(r.Context(), runtimeID, payload.Input)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to send response: "+err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "sent": payload.Input})
			return
		}
	}

	writeError(w, http.StatusBadRequest, "unknown runtime endpoint")
}

// Providers Handler
func (h *APIHandler) handleProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Provider string `json:"provider"`
			Profile  string `json:"profile"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Provider) == "" {
			writeError(w, http.StatusBadRequest, "provider is required")
			return
		}
		req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
		if req.Profile == "" {
			req.Profile = req.Provider
		}
		if _, ok := h.findProvider(req.Provider); !ok {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		if profile.Exists(req.Provider, req.Profile) {
			writeError(w, http.StatusConflict, "profile already exists")
			return
		}
		created, err := profile.Create(req.Provider, req.Profile)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"profile": created, "state": coreprovider.PendingAuth})
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	type ProviderView struct {
		ID                string                       `json:"id"`
		Installed         bool                         `json:"installed"`
		Version           string                       `json:"version"`
		ControlLevel      registry.ControlLevel        `json:"control_level"`
		Capabilities      driver.EffectiveCapabilities `json:"capabilities"`
		RegistrationState string                       `json:"registration_state,omitempty"`
		BinaryPath        string                       `json:"binary_path,omitempty"`
	}
	if os.Getenv("NEXUS_DOCS_CAPTURE") == "1" {
		// Documentation captures must never disclose which provider binaries or
		// versions happen to be installed on the capture host.
		demo := driver.NewShellDriver().EffectiveCaps(r.Context(), model.Profile{
			Name: "docs-fixture",
		})
		writeJSON(w, http.StatusOK, []ProviderView{{
			ID:           "demo-provider",
			Installed:    true,
			Version:      "synthetic-fixture",
			ControlLevel: demo.ControlLevel,
			Capabilities: demo,
		}})
		return
	}

	drivers := h.drivers.List()

	showInternal := r.URL.Query().Get("internal") == "true"
	var selectedDrivers []driver.ControlDriver
	for _, d := range drivers {
		if !showInternal && (d.ProviderID() == "fake" || d.ProviderID() == "shell") {
			continue
		}
		selectedDrivers = append(selectedDrivers, d)
	}

	res := make([]ProviderView, len(selectedDrivers))
	var wg sync.WaitGroup
	for i, d := range selectedDrivers {
		wg.Add(1)
		go func(idx int, drv driver.ControlDriver) {
			defer wg.Done()
			pctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			det, _ := drv.Detect(pctx)
			caps := drv.EffectiveCaps(pctx, model.Profile{Name: "default", Provider: drv.ProviderID()})
			res[idx] = ProviderView{
				ID:           drv.ProviderID(),
				Installed:    det.Installed,
				Version:      det.Version,
				ControlLevel: caps.ControlLevel,
				Capabilities: caps,
				BinaryPath:   det.BinaryPath,
			}
			if det.Installed {
				res[idx].RegistrationState = string(coreprovider.InstalledUnregistered)
			}
			if ps, listErr := profile.List(); listErr == nil {
				for _, registered := range ps {
					if registered.Provider == drv.ProviderID() {
						if profile.GetAccountInfo(registered.Provider, registered.Name).Authenticated {
							res[idx].RegistrationState = string(coreprovider.RegisteredAuthenticated)
							continue
						}
						res[idx].RegistrationState = string(coreprovider.PendingAuth)
					}
				}
			}
		}(i, d)
	}
	wg.Wait()
	writeJSON(w, http.StatusOK, res)
}

// Profiles Handler
func (h *APIHandler) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		profiles, _ := profile.List()
		writeJSON(w, http.StatusOK, profiles)
		return
	}
	if r.Method == http.MethodDelete {
		var body struct {
			Provider string `json:"provider"`
			Profile  string `json:"profile"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		prov := strings.TrimSpace(body.Provider)
		if prov == "" {
			prov = strings.TrimSpace(r.URL.Query().Get("provider"))
		}
		name := strings.TrimSpace(body.Profile)
		if name == "" {
			name = strings.TrimSpace(r.URL.Query().Get("profile"))
		}
		if name == "" {
			name = strings.TrimSpace(r.URL.Query().Get("name"))
		}
		if prov == "" || name == "" {
			writeError(w, http.StatusBadRequest, "provider and profile are required")
			return
		}
		if err := profile.Delete(prov, name); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": true})
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (h *APIHandler) handleProfileDefault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Provider string `json:"provider"`
		Profile  string `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Provider) == "" || strings.TrimSpace(req.Profile) == "" {
		writeError(w, http.StatusBadRequest, "provider and profile are required")
		return
	}
	if err := config.SetDefaultProfile(strings.TrimSpace(req.Provider), strings.TrimSpace(req.Profile)); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "provider": req.Provider, "profile": req.Profile})
}

func (h *APIHandler) handleProfileLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Provider string `json:"provider"`
		Profile  string `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Provider) == "" || strings.TrimSpace(req.Profile) == "" {
		writeError(w, http.StatusBadRequest, "provider and profile are required")
		return
	}
	prov := strings.TrimSpace(req.Provider)
	name := strings.TrimSpace(req.Profile)

	p := model.Profile{Provider: prov, Name: name}
	if drv, ok := h.findProvider(prov); ok {
		if authProv, ok := any(drv).(coreprovider.AuthProvider); ok {
			_ = authProv.Logout(r.Context(), p)
		}
	}
	if home, err := config.ProfileHome(prov, name); err == nil {
		_ = os.Remove(filepath.Join(home, "auth.json"))
		_ = os.Remove(filepath.Join(home, "."+prov, "auth.json"))
	}
	_ = profile.RefreshUsageSnapshot(prov, name)
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "logged_out": true})
}

// Events Handler
func (h *APIHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	runtimeID := r.URL.Query().Get("runtime_id")
	limit := 50
	evs := events.DefaultBus().GetHistory(runtimeID, limit)
	if evs == nil {
		evs = []events.Event{}
	}
	if accountID := strings.TrimSpace(r.URL.Query().Get("account_id")); accountID != "" {
		providerID := strings.TrimSpace(r.URL.Query().Get("provider_id"))
		identityVersion := strings.TrimSpace(r.URL.Query().Get("identity_version"))
		filtered := evs[:0]
		for _, ev := range evs {
			scope := ev.AccountScope
			if scope.AccountID != accountID || (providerID != "" && scope.ProviderID != providerID) || (identityVersion != "" && scope.IdentityVersion != identityVersion) {
				continue
			}
			filtered = append(filtered, ev)
		}
		evs = filtered
	}
	for i := range evs {
		evs[i].Summary = security.Redact(evs[i].Summary)
	}
	writeJSON(w, http.StatusOK, evs)
}
