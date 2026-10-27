package web

import (
	"errors"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/artifacts"
)

// Project outputs/files reuse the Nexus authenticated project route. In
// particular they NEVER turn the host-local /api/v1/fs endpoints into remote
// filesystem APIs.
func (h *NexusHandler) projectArtifactRoot(r *http.Request) (*os.Root, error) {
	id := projectIDFromPath(r.URL.Path)
	if id == "" || strings.ContainsAny(id, "/\\.") {
		return nil, artifacts.ErrForbidden
	}
	project, _, err := h.projects.Get(r.Context(), id)
	if err != nil {
		return nil, err
	}
	return artifacts.Root(project.CanonicalPath)
}
func (h *NexusHandler) handleProjectOutputs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	root, err := h.projectArtifactRoot(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "project unavailable")
		return
	}
	defer root.Close()
	offset := 0
	limit := 48
	if str := r.URL.Query().Get("offset"); str != "" {
		offset, err = strconv.Atoi(str)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
	}
	if str := r.URL.Query().Get("limit"); str != "" {
		limit, err = strconv.Atoi(str)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}
	k := r.URL.Query().Get("kind")
	if k != "" && k != "all" && k != "image" && k != "document" && k != "video" && k != "audio" && k != "text" && k != "download" {
		writeError(w, http.StatusBadRequest, "invalid kind")
		return
	}
	items, err := artifacts.Scan(root, offset, limit, k)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if dataDir, dataErr := config.DataDir(); dataErr == nil {
		for i := range items.Items {
			if enhanced, noteErr := artifacts.Observe(root, dataDir, projectIDFromPath(r.URL.Path), items.Items[i], "filesystem", "", ""); noteErr == nil {
				items.Items[i] = enhanced
			}
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, items)
}
func (h *NexusHandler) handleProjectFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeError(w, http.StatusMethodNotAllowed, "read only")
		return
	}
	root, err := h.projectArtifactRoot(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "project unavailable")
		return
	}
	defer root.Close()
	rel := r.URL.Query().Get("path")
	if rel == "" {
		rel = "."
	}
	if r.URL.Query().Has("history") {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "GET required"); return
		}
		dataDir, err := config.DataDir()
		if err != nil { writeError(w, http.StatusServiceUnavailable, "output history unavailable");return }
		// Refuse history of paths that are no longer valid, authorized files.
		if _, err := artifacts.Describe(root, rel); err != nil { writeError(w, http.StatusNotFound, "output unavailable");return }
		versions, err := artifacts.ListRevisions(dataDir, projectIDFromPath(r.URL.Path), rel)
		if err != nil { writeError(w, http.StatusNotFound, "output history unavailable");return }
		writeJSON(w,http.StatusOK,map[string]any{"versions":versions})
		return
	}
	if numberRaw := r.URL.Query().Get("version"); numberRaw != "" {
		dataDir, err := config.DataDir()
		if err != nil { writeError(w, http.StatusServiceUnavailable, "output history unavailable");return }
		number, err := strconv.Atoi(numberRaw)
		if err != nil { writeError(w, http.StatusBadRequest, "invalid revision");return }
		// The current project root is resolved before serving any old versions.
		if _, err := artifacts.Describe(root, rel); err != nil { writeError(w, http.StatusNotFound, "output unavailable");return }
		if err := artifacts.ServeRevision(w,r,dataDir,projectIDFromPath(r.URL.Path),rel,number);err!=nil {
			writeError(w,http.StatusNotFound,"output revision unavailable")
		}
		return
	}
	if r.URL.Query().Has("thumbnail") {
		dataDir, err := config.DataDir()
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "thumbnail cache unavailable")
			return
		}
		if err := artifacts.Thumbnail(w, r, root, rel, dataDir, projectIDFromPath(r.URL.Path)); err != nil {
			writeError(w, http.StatusUnsupportedMediaType, "thumbnail unavailable")
		}
		return
	}
	if r.URL.Query().Has("download") || r.URL.Query().Has("view") {
		if err := artifacts.Serve(w, r, root, rel, r.URL.Query().Has("view")); err != nil {
			code := http.StatusForbidden
			if errors.Is(err, artifacts.ErrUnsupported) {
				code = http.StatusUnsupportedMediaType
			} else if errors.Is(err, http.ErrMissingFile) {
				code = http.StatusNotFound
			}
			writeError(w, code, "file unavailable or not permitted")
		}
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	result, err := artifacts.Directory(root, rel)
	if err != nil {
		writeError(w, http.StatusForbidden, "directory unavailable or not permitted")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, map[string]any{"path": rel, "entries": result})
}
