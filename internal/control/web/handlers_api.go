package web

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/buildinfo"
	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/control/events"
	"github.com/kivervinicius/ai-cli/internal/control/handoff"
	"github.com/kivervinicius/ai-cli/internal/control/launcher"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/control/workspace"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/nexus"
	"github.com/kivervinicius/ai-cli/internal/profile"
)

type APIHandler struct {
	auth     *AuthManager
	reg      *registry.Registry
	launcher *launcher.Launcher
	drivers  *driver.Registry
	quotaEng *quota.Engine
}

func NewAPIHandler(auth *AuthManager) *APIHandler {
	return &APIHandler{
		auth:     auth,
		reg:      registry.DefaultRegistry(),
		launcher: launcher.Default(),
		drivers:  driver.DefaultRegistry(),
		quotaEng: quota.NewEngine(5 * time.Minute),
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": security.Redact(msg)})
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

// Session Handler (checks authentication status & returns CSRF token)
func (h *APIHandler) handleSession(w http.ResponseWriter, r *http.Request) {
	sess := h.auth.AuthenticateRequest(r)
	if sess == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": false,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"csrf_token":    sess.CSRFToken,
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
		_, _ = h.reg.CleanupStale()
		all := h.reg.List()
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

		sess, err := h.launcher.Launch(ctx, launcher.LaunchOptions{
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
		cleaned, _ := h.reg.CleanupStale()
		purged, _ := h.reg.PurgeInactive()
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
	sess, exists := h.reg.Get(runtimeID)
	if !exists {
		writeError(w, http.StatusNotFound, "runtime not found")
		return
	}

	// GET detail
	if len(parts) == 1 && r.Method == http.MethodGet {
		d, _ := h.drivers.Get(sess.ProviderID)
		var effCaps any
		if d != nil {
			effCaps = d.EffectiveCaps(r.Context(), model.Profile{Name: sess.ProfileID, Provider: sess.ProviderID})
		}
		cleanSess := sanitizeSession(sess)
		writeJSON(w, http.StatusOK, map[string]any{
			"session":      cleanSess,
			"capabilities": effCaps,
		})
		return
	}

	// DELETE runtime record
	if len(parts) == 1 && r.Method == http.MethodDelete {
		_ = h.reg.Delete(runtimeID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}

	// Actions: stop, handoff, continue
	if len(parts) == 2 && r.Method == http.MethodPost {
		action := parts[1]
		switch action {
		case "stop":
			client, err := protocol.NewClient(runtimeID)
			if err == nil {
				_ = client.Stop()
				_ = client.Close()
			}
			// Wait for the process to actually exit before reporting STOPPED.
			if s, ok := h.reg.Get(runtimeID); ok {
				deadline := time.Now().Add(5 * time.Second)
				for time.Now().Before(deadline) && s.PID > 0 && registry.IsProcessAlive(s.PID) {
					time.Sleep(100 * time.Millisecond)
					s, _ = h.reg.Get(runtimeID)
				}
			}
			_ = h.reg.UpdateState(runtimeID, registry.StateStopped)
			writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
			return

		case "handoff":
			var payload struct {
				Target string `json:"target"` // provider:profile
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Target == "" {
				writeError(w, http.StatusBadRequest, "missing target profile")
				return
			}
			newSess, err := handoff.PerformAccountHandoff(r.Context(), runtimeID, payload.Target)
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
			newSess, err := handoff.PerformContextHandoff(r.Context(), runtimeID, payload.TargetProvider, payload.TargetProfile)
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
			if err := h.reg.UpdateTitle(runtimeID, payload.Title); err != nil {
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
			client, err := protocol.NewClient(runtimeID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to connect to runtime: "+err.Error())
				return
			}
			defer client.Close()

			inputStr := payload.Input
			if !strings.HasSuffix(inputStr, "\n") {
				inputStr += "\n"
			}
			inputBytes, _ := json.Marshal(protocol.InputPayload{Data: inputStr})
			_, err = client.Send(protocol.CmdInput, inputBytes)
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
	drivers := h.drivers.List()
	type ProviderView struct {
		ID           string                       `json:"id"`
		Installed    bool                         `json:"installed"`
		Version      string                       `json:"version"`
		ControlLevel registry.ControlLevel        `json:"control_level"`
		Capabilities driver.EffectiveCapabilities `json:"capabilities"`
	}

	showInternal := r.URL.Query().Get("internal") == "true"
	var filtered []driver.ControlDriver
	for _, d := range drivers {
		if !showInternal && (d.ProviderID() == "fake" || d.ProviderID() == "shell") {
			continue
		}
		filtered = append(filtered, d)
	}

	res := make([]ProviderView, len(filtered))
	var wg sync.WaitGroup
	wg.Add(len(filtered))

	for i, d := range filtered {
		go func(idx int, drv driver.ControlDriver) {
			defer wg.Done()
			pctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
			defer cancel()
			det, _ := drv.Detect(pctx)
			caps := drv.EffectiveCaps(pctx, model.Profile{Name: "default", Provider: drv.ProviderID()})
			res[idx] = ProviderView{
				ID:           drv.ProviderID(),
				Installed:    det.Installed,
				Version:      det.Version,
				ControlLevel: caps.ControlLevel,
				Capabilities: caps,
			}
		}(i, d)
	}

	wg.Wait()
	writeJSON(w, http.StatusOK, res)
}

// Profiles Handler
func (h *APIHandler) handleProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, _ := profile.List()
	writeJSON(w, http.StatusOK, profiles)
}

// Events Handler
func (h *APIHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	runtimeID := r.URL.Query().Get("runtime_id")
	evs := events.DefaultBus().GetHistory(runtimeID, 50)
	if evs == nil {
		evs = []events.Event{}
	}
	for i := range evs {
		evs[i].Summary = security.Redact(evs[i].Summary)
	}
	writeJSON(w, http.StatusOK, evs)
}
