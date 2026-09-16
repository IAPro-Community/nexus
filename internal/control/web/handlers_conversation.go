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
	"time"

	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func wantsComposerProgressStream(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "application/x-ndjson") || strings.Contains(accept, "text/event-stream")
}

type ndjsonComposerProgress struct {
	mu      sync.Mutex
	w       http.ResponseWriter
	flusher http.Flusher
}

func (p *ndjsonComposerProgress) Step(id, status string) {
	p.write(map[string]any{"type": "step", "id": id, "status": status})
}

func (p *ndjsonComposerProgress) write(payload map[string]any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	enc := json.NewEncoder(p.w)
	_ = enc.Encode(payload)
	if p.flusher != nil {
		p.flusher.Flush()
	}
}

func (h *NexusHandler) streamComposerCreate(w http.ResponseWriter, r *http.Request, projectID, goal, sourcePrompt, inputMode string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusCreated)
	flusher.Flush()

	sink := &ndjsonComposerProgress{w: w, flusher: flusher}
	started := time.Now()
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-r.Context().Done():
				return
			case <-ticker.C:
				sink.write(map[string]any{
					"type":       "heartbeat",
					"elapsed_ms": time.Since(started).Milliseconds(),
				})
			}
		}
	}()

	view, err := h.composer.CreateWithProgress(r.Context(), projectID, goal, sourcePrompt, inputMode, sink)
	close(done)
	if err != nil {
		sink.write(map[string]any{"type": "error", "message": err.Error()})
		return
	}
	sink.write(map[string]any{"type": "complete", "view": view})
}

var allowedAttachmentMIME = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
}

func (h *NexusHandler) handleComposerAttachments(w http.ResponseWriter, r *http.Request, sessionID string) {
	switch r.Method {
	case http.MethodGet:
		st, err := h.nexus.OpenProject()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		items, err := st.ListConversationAttachments(sessionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if items == nil {
			items = []store.ConversationAttachment{}
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		h.uploadComposerAttachment(w, r, sessionID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *NexusHandler) uploadComposerAttachment(w http.ResponseWriter, r *http.Request, sessionID string) {
	const maxBytes = 8 << 20
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart body")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if !allowedAttachmentMIME[strings.ToLower(mimeType)] {
		writeError(w, http.StatusBadRequest, "only PNG, JPEG and WEBP images are allowed")
		return
	}
	dataDir, err := coreconfig.DataDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := st.GetComposerSession(sessionID); err != nil {
		writeError(w, http.StatusNotFound, "composer session not found")
		return
	}
	dir := filepath.Join(dataDir, "attachments", sessionID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	pending := store.ConversationAttachment{
		SessionID: sessionID,
		Name:      filepath.Base(header.Filename),
		MimeType:  mimeType,
		Type:      "image",
	}
	created, err := st.CreateConversationAttachment(pending)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	finalPath := filepath.Join(dir, created.ID)
	out, err := os.OpenFile(finalPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		_ = st.DeleteConversationAttachment(created.ID)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	written, copyErr := io.Copy(out, io.LimitReader(file, maxBytes+1))
	_ = out.Close()
	if copyErr != nil || written > maxBytes {
		_ = os.Remove(finalPath)
		_ = st.DeleteConversationAttachment(created.ID)
		if written > maxBytes {
			writeError(w, http.StatusBadRequest, "file too large")
			return
		}
		writeError(w, http.StatusInternalServerError, copyErr.Error())
		return
	}
	created.Size = written
	created.StorageRef = finalPath
	if err := st.UpdateConversationAttachment(*created); err != nil {
		_ = os.Remove(finalPath)
		_ = st.DeleteConversationAttachment(created.ID)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type uploadResponse struct {
		*store.ConversationAttachment
		URL string `json:"url"`
	}
	writeJSON(w, http.StatusCreated, uploadResponse{created, attachmentURL(created.ID)})
}

func (h *NexusHandler) handleAttachmentByID(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/attachments/"), "/")
	if id == "" {
		writeError(w, http.StatusNotFound, "missing attachment id")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	att, err := st.GetConversationAttachment(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		http.ServeFile(w, r, att.StorageRef)
	case http.MethodDelete:
		_ = os.Remove(att.StorageRef)
		if err := st.DeleteConversationAttachment(id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// Ensure nexus.ComposerProgressSink is referenced for compile clarity.
var _ nexus.ComposerProgressSink = (*ndjsonComposerProgress)(nil)

func attachmentURL(id string) string {
	return fmt.Sprintf("/api/v1/attachments/%s", id)
}
