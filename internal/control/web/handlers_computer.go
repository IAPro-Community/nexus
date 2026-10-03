package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/IAPro-Community/nexus/internal/control/computer"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const computerUseRequestBodyLimit = 1 << 20


const computerScreenshotRetentionPerProject = 20

type computerScreenshotManager struct {
	mu        sync.Mutex
	root      string
	paths     map[string]map[string]string
	order     map[string][]string
}

func newComputerScreenshotManager() *computerScreenshotManager {
	root := ""
	if dataDir, err := config.DataDir(); err == nil {
		root = filepath.Join(dataDir, "computer-use")
	}
	return &computerScreenshotManager{
		root:  root,
		paths: make(map[string]map[string]string),
		order: make(map[string][]string),
	}
}

func (m *computerScreenshotManager) register(projectID, path string) (string, error) {
	projectID = strings.TrimSpace(projectID)
	path = strings.TrimSpace(path)
	if projectID == "" || path == "" {
		return "", fmt.Errorf("computer screenshot project and path are required")
	}
	if err := m.validatePath(path); err != nil {
		return "", err
	}
	ref := filepath.Base(path)
	if ref == "" || ref == "." || ref != filepath.Base(ref) {
		return "", fmt.Errorf("computer screenshot reference is invalid")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.paths[projectID] == nil {
		m.paths[projectID] = make(map[string]string)
	}
	if _, exists := m.paths[projectID][ref]; !exists {
		m.order[projectID] = append(m.order[projectID], ref)
	}
	m.paths[projectID][ref] = path

	for len(m.order[projectID]) > computerScreenshotRetentionPerProject {
		oldest := m.order[projectID][0]
		m.order[projectID] = m.order[projectID][1:]
		oldPath := m.paths[projectID][oldest]
		delete(m.paths[projectID], oldest)
		if oldPath != "" && oldPath != path {
			_ = os.Remove(oldPath)
		}
	}
	return ref, nil
}

func (m *computerScreenshotManager) validatePath(path string) error {
	if m == nil || strings.TrimSpace(m.root) == "" {
		return fmt.Errorf("computer screenshot root is unavailable")
	}
	rootAbs, err := filepath.Abs(m.root)
	if err != nil {
		return fmt.Errorf("resolve computer screenshot root: %w", err)
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve computer screenshot path: %w", err)
	}
	rootResolved := rootAbs
	if resolved, resolveErr := filepath.EvalSymlinks(rootAbs); resolveErr == nil {
		rootResolved = resolved
	}
	pathResolved, err := filepath.EvalSymlinks(pathAbs)
	if err != nil {
		return fmt.Errorf("resolve computer screenshot artifact: %w", err)
	}
	rel, err := filepath.Rel(rootResolved, pathResolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("computer screenshot is outside the Nexus artifact directory")
	}
	info, err := os.Stat(pathResolved)
	if err != nil {
		return fmt.Errorf("stat computer screenshot artifact: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 32<<20 {
		return fmt.Errorf("computer screenshot artifact is not a bounded regular file")
	}
	if !strings.EqualFold(filepath.Ext(pathResolved), ".png") {
		return fmt.Errorf("computer screenshot artifact must be PNG")
	}
	return nil
}

func (m *computerScreenshotManager) path(projectID, ref string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	byProject := m.paths[strings.TrimSpace(projectID)]
	if byProject == nil {
		return "", false
	}
	path, ok := byProject[ref]
	return path, ok
}

func (m *computerScreenshotManager) closeAll() {
	m.mu.Lock()
	paths := m.paths
	m.paths = make(map[string]map[string]string)
	m.order = make(map[string][]string)
	m.mu.Unlock()
	for _, byProject := range paths {
		for _, path := range byProject {
			_ = os.Remove(path)
		}
	}
}

type computerSnapshotResponse struct {
	computer.Snapshot
	ScreenshotRef string `json:"screenshot_ref,omitempty"`
}

type computerActionResponse struct {
	Snapshot     computerSnapshotResponse      `json:"snapshot"`
	Verification computer.ActionVerification   `json:"verification"`
}

func (h *NexusHandler) computerSnapshotResponse(projectID string, snapshot computer.Snapshot) (computerSnapshotResponse, error) {
	ref := ""
	if strings.TrimSpace(snapshot.Screenshot) != "" {
		if h.computerScreenshots == nil {
			return computerSnapshotResponse{}, fmt.Errorf("computer screenshot manager is unavailable")
		}
		var err error
		ref, err = h.computerScreenshots.register(projectID, snapshot.Screenshot)
		if err != nil {
			return computerSnapshotResponse{}, err
		}
	}
	snapshot.Screenshot = ""
	return computerSnapshotResponse{Snapshot: snapshot, ScreenshotRef: ref}, nil
}

func (h *NexusHandler) handleProjectComputerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if _, err := st.GetProject(projectID); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status, err := h.nexus.ComputerUseStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *NexusHandler) handleProjectComputerApps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	if err := h.requireComputerProject(projectID); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	var body struct {
		Contract runner.AutonomyContract `json:"contract"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, computerUseRequestBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid computer apps body")
		return
	}
	apps, err := h.nexus.ComputerListApps(r.Context(), body.Contract)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"apps": apps})
}

func (h *NexusHandler) handleProjectComputerWindows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	if err := h.requireComputerProject(projectID); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	var body struct {
		Contract runner.AutonomyContract `json:"contract"`
		App      string                  `json:"app"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, computerUseRequestBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || strings.TrimSpace(body.App) == "" {
		writeError(w, http.StatusBadRequest, "app is required")
		return
	}
	windows, err := h.nexus.ComputerListWindows(r.Context(), body.Contract, body.App)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"windows": windows})
}

func (h *NexusHandler) handleProjectComputerSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	if err := h.requireComputerProject(projectID); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		Contract runner.AutonomyContract  `json:"contract"`
		Request  computer.SnapshotRequest `json:"request"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, computerUseRequestBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid computer snapshot body")
		return
	}
	body.Request.App = strings.TrimSpace(body.Request.App)
	snapshot, err := h.nexus.ComputerSnapshot(r.Context(), body.Contract, body.Request)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	response, err := h.computerSnapshotResponse(projectID, *snapshot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *NexusHandler) handleProjectComputerAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	if err := h.requireComputerProject(projectID); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		Contract runner.AutonomyContract `json:"contract"`
		Request  computer.ActionRequest  `json:"request"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, computerUseRequestBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid computer action body")
		return
	}
	body.Request.App = strings.TrimSpace(body.Request.App)
	result, err := h.nexus.ComputerAct(r.Context(), body.Contract, body.Request)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	responseSnapshot, err := h.computerSnapshotResponse(projectID, result.Snapshot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, computerActionResponse{
		Snapshot:     responseSnapshot,
		Verification: result.Verification,
	})
}

func (h *NexusHandler) handleProjectComputerScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	if err := h.requireComputerProject(projectID); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	prefix := "/api/v1/projects/" + projectID + "/computer/screenshot/"
	ref := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, prefix))
	if ref == "" || filepath.Base(ref) != ref || h.computerScreenshots == nil {
		writeError(w, http.StatusBadRequest, "invalid screenshot reference")
		return
	}
	path, ok := h.computerScreenshots.path(projectID, ref)
	if !ok {
		writeError(w, http.StatusNotFound, "screenshot not found")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, path)
}

func (h *NexusHandler) requireComputerProject(projectID string) error {
	st, err := h.nexus.OpenProject()
	if err != nil {
		return err
	}
	_, err = st.GetProject(strings.TrimSpace(projectID))
	return err
}
