package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	nexusbrowser "github.com/IAPro-Community/nexus/internal/browser"
	"github.com/IAPro-Community/nexus/internal/core/flags"
)

type projectDesignBrowser struct {
	browser     *nexusbrowser.DesignBrowser
	url         string
	startedAt   time.Time
	screenshots map[string]string
}

type designModeManager struct {
	mu       sync.Mutex
	sessions map[string]*projectDesignBrowser
}

func newDesignModeManager() *designModeManager {
	return &designModeManager{sessions: make(map[string]*projectDesignBrowser)}
}

func (m *designModeManager) status(projectID string) (bool, string, time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session := m.sessions[projectID]
	if session == nil {
		return false, "", time.Time{}
	}
	return true, session.url, session.startedAt
}

func (m *designModeManager) replace(projectID, rawURL string, designBrowser *nexusbrowser.DesignBrowser) {
	m.mu.Lock()
	previous := m.sessions[projectID]
	m.sessions[projectID] = &projectDesignBrowser{
		browser: designBrowser, url: rawURL, startedAt: time.Now().UTC(), screenshots: make(map[string]string),
	}
	m.mu.Unlock()
	closeProjectDesignBrowser(previous)
}

func (m *designModeManager) stop(projectID string) bool {
	m.mu.Lock()
	session := m.sessions[projectID]
	delete(m.sessions, projectID)
	m.mu.Unlock()
	if session == nil {
		return false
	}
	closeProjectDesignBrowser(session)
	return true
}

func (m *designModeManager) closeAll() {
	m.mu.Lock()
	sessions := m.sessions
	m.sessions = make(map[string]*projectDesignBrowser)
	m.mu.Unlock()
	for _, session := range sessions {
		closeProjectDesignBrowser(session)
	}
}

func closeProjectDesignBrowser(session *projectDesignBrowser) {
	if session == nil {
		return
	}
	if session.browser != nil {
		_ = session.browser.Close()
	}
	for _, path := range session.screenshots {
		_ = os.Remove(path)
	}
}

func (m *designModeManager) withBrowser(projectID string, fn func(*projectDesignBrowser) error) error {
	m.mu.Lock()
	session := m.sessions[projectID]
	m.mu.Unlock()
	if session == nil || session.browser == nil {
		return errors.New("Design Mode browser is not active for this project")
	}
	return fn(session)
}

func (m *designModeManager) registerScreenshot(projectID, path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	ref := filepath.Base(path)
	m.mu.Lock()
	defer m.mu.Unlock()
	if session := m.sessions[projectID]; session != nil {
		session.screenshots[ref] = path
		return ref
	}
	return ""
}

func (m *designModeManager) screenshotPath(projectID, ref string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session := m.sessions[projectID]
	if session == nil {
		return "", false
	}
	path, ok := session.screenshots[ref]
	return path, ok
}

type designModeArtifactResponse struct {
	Version        int                         \`json:"version"\`
	URL            string                      \`json:"url"\`
	Title          string                      \`json:"title,omitempty"\`
	Selector       string                      \`json:"selector"\`
	HTML           string                      \`json:"html"\`
	Neighborhood   string                      \`json:"neighborhood,omitempty"\`
	Text           string                      \`json:"text,omitempty"\`
	ComputedStyles map[string]string           \`json:"computed_styles,omitempty"\`
	Bounds         nexusbrowser.UIElementBounds \`json:"bounds"\`
	SourceHints    map[string]string           \`json:"source_hints,omitempty"\`
	ScreenshotRef  string                      \`json:"screenshot_ref,omitempty"\`
	CapturedAt     time.Time                   \`json:"captured_at"\`
}

func (h *NexusHandler) handleProjectDesignMode(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	if _, _, err := h.projects.Get(r.Context(), projectID); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	enabled := flags.Enabled(flags.BrowserDesignMode)
	tail := strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"+projectID+"/design-mode")
	tail = strings.Trim(tail, "/")

	if tail == "status" && r.Method == http.MethodGet {
		active, rawURL, startedAt := h.designMode.status(projectID)
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": enabled,
			"active": active,
			"url": rawURL,
			"started_at": startedAt,
		})
		return
	}
	if !enabled {
		writeError(w, http.StatusNotFound, "browser Design Mode is disabled")
		return
	}

	switch {
	case tail == "start" && r.Method == http.MethodPost:
		var body struct {
			URL string \`json:"url"\`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.URL) == "" {
			writeError(w, http.StatusBadRequest, "url is required")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		designBrowser, err := nexusbrowser.StartDesignBrowser(ctx, strings.TrimSpace(body.URL))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.designMode.replace(projectID, strings.TrimSpace(body.URL), designBrowser)
		writeJSON(w, http.StatusCreated, map[string]any{"active": true, "url": strings.TrimSpace(body.URL)})
		return

	case tail == "stop" && r.Method == http.MethodPost:
		writeJSON(w, http.StatusOK, map[string]any{"stopped": h.designMode.stop(projectID)})
		return

	case tail == "picker/start" && r.Method == http.MethodPost:
		var status *nexusbrowser.DesignPickerStatus
		err := h.designMode.withBrowser(projectID, func(session *projectDesignBrowser) error {
			var inner error
			status, inner = session.browser.BeginElementPicker(r.Context())
			return inner
		})
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, status)
		return

	case tail == "picker" && r.Method == http.MethodGet:
		var status *nexusbrowser.DesignPickerStatus
		err := h.designMode.withBrowser(projectID, func(session *projectDesignBrowser) error {
			var inner error
			status, inner = session.browser.PickerStatus(r.Context())
			return inner
		})
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, status)
		return

	case tail == "inspect" && r.Method == http.MethodPost:
		var body struct {
			Selector string \`json:"selector"\`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.Selector) == "" {
			writeError(w, http.StatusBadRequest, "selector is required")
			return
		}
		var artifact *nexusbrowser.UIContextArtifact
		err := h.designMode.withBrowser(projectID, func(session *projectDesignBrowser) error {
			var inner error
			artifact, inner = session.browser.InspectSelector(r.Context(), body.Selector)
			if inner == nil {
				_ = session.browser.ClearPicker(r.Context())
			}
			return inner
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ref := h.designMode.registerScreenshot(projectID, artifact.ScreenshotPath)
		writeJSON(w, http.StatusOK, designModeArtifactResponse{
			Version: artifact.Version,
			URL: artifact.URL,
			Title: artifact.Title,
			Selector: artifact.Selector,
			HTML: artifact.HTML,
			Neighborhood: artifact.Neighborhood,
			Text: artifact.Text,
			ComputedStyles: artifact.ComputedStyles,
			Bounds: artifact.Bounds,
			SourceHints: artifact.SourceHints,
			ScreenshotRef: ref,
			CapturedAt: artifact.CapturedAt,
		})
		return

	case strings.HasPrefix(tail, "screenshot/") && r.Method == http.MethodGet:
		ref := strings.TrimSpace(strings.TrimPrefix(tail, "screenshot/"))
		if ref == "" || filepath.Base(ref) != ref {
			writeError(w, http.StatusBadRequest, "invalid screenshot reference")
			return
		}
		path, ok := h.designMode.screenshotPath(projectID, ref)
		if !ok {
			writeError(w, http.StatusNotFound, "screenshot not found")
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, path)
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "unsupported Design Mode operation")
}
