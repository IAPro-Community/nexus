package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/webp"

	"github.com/IAPro-Community/nexus/internal/control/ids"
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

func (h *NexusHandler) streamComposerCreate(
	w http.ResponseWriter,
	r *http.Request,
	projectID, goal, sourcePrompt, inputMode, interviewMode string,
) {
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

	view, err := h.composer.CreateWithInterviewModeProgress(
		r.Context(),
		projectID,
		goal,
		sourcePrompt,
		inputMode,
		interviewMode,
		sink,
	)
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

// Attachment ingestion limits. The per-file cap is enforced on the decoded
// source and on the re-encoded output; the request cap covers the whole
// multipart body including framing overhead.
const (
	attachmentMaxFileBytes    = 8 << 20
	attachmentMaxRequestBytes = (8 << 20) + (1 << 20)
	attachmentMaxPerSession   = 50
	attachmentMaxSessionBytes = 64 << 20
	attachmentMaxDimension    = 8192
	attachmentMaxPixels       = int64(32_000_000)
)

// attachmentScopeChars restricts IDs used as storage path segments so a
// hostile session or attachment ID can never escape the attachments dir.
func validAttachmentScope(value string) bool {
	if value == "" || len(value) > 64 || value != filepath.Base(value) {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

// validAttachmentID accepts only server-minted att_<ULID> IDs, so path
// traversal through /api/v1/attachments/<id> is impossible by construction.
func validAttachmentID(id string) bool {
	const prefix = "att_"
	if !strings.HasPrefix(id, prefix) || len(id) != len(prefix)+26 {
		return false
	}
	for _, r := range id[len(prefix):] {
		if r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' {
			if r == 'I' || r == 'L' || r == 'O' || r == 'U' {
				return false
			}
			continue
		}
		return false
	}
	return true
}

// sniffImageContentType determines the image type from magic bytes only. The
// client-supplied Content-Type header is never trusted.
func sniffImageContentType(head []byte) string {
	if len(head) > 512 {
		head = head[:512]
	}
	detected := http.DetectContentType(head)
	if allowedAttachmentMIME[detected] {
		return detected
	}
	return ""
}

// looksLikePolyglot rejects text markup smuggled inside an image container.
func looksLikePolyglot(head []byte) bool {
	sample := strings.ToLower(string(head))
	if len(sample) > 2048 {
		sample = sample[:2048]
	}
	for _, marker := range []string{"<html", "<script", "<svg", "<?xml", "<!doctype", "javascript:"} {
		if strings.Contains(sample, marker) {
			return true
		}
	}
	return false
}

// sanitizeAttachmentName keeps a display name without ever using it as a path.
func sanitizeAttachmentName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	base = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, base)
	if base == "" || base == "." || base == "/" {
		return "attachment"
	}
	if len(base) > 128 {
		base = base[:128]
	}
	return base
}

// normalizeAttachmentImage fully decodes raw and re-encodes it, destroying
// embedded payloads (polyglots, trailing scripts, malicious metadata). It
// returns the normalized bytes and the server-determined MIME type. WebP is
// normalized to PNG because the server must only ever serve bytes produced
// by its own encoder.
func normalizeAttachmentImage(raw []byte) ([]byte, string, error) {
	mimeType := sniffImageContentType(raw)
	if mimeType == "" {
		return nil, "", fmt.Errorf("unsupported image content")
	}
	if looksLikePolyglot(raw) {
		return nil, "", fmt.Errorf("suspect embedded markup")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, "", fmt.Errorf("unreadable image: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > attachmentMaxDimension || cfg.Height > attachmentMaxDimension {
		return nil, "", fmt.Errorf("image dimensions out of range")
	}
	if int64(cfg.Width)*int64(cfg.Height) > attachmentMaxPixels {
		return nil, "", fmt.Errorf("image resolution too large")
	}
	var img image.Image
	switch mimeType {
	case "image/webp":
		img, err = webp.Decode(bytes.NewReader(raw))
	default:
		img, err = decodeStdImage(bytes.NewReader(raw))
	}
	if err != nil {
		return nil, "", fmt.Errorf("corrupt image: %w", err)
	}
	var out bytes.Buffer
	switch mimeType {
	case "image/jpeg":
		if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 90}); err != nil {
			return nil, "", err
		}
		return out.Bytes(), "image/jpeg", nil
	default:
		if err := png.Encode(&out, img); err != nil {
			return nil, "", err
		}
		return out.Bytes(), "image/png", nil
	}
}

// decodeStdImage decodes PNG/JPEG via the standard library. WebP is handled
// separately because std has no WebP codec.
func decodeStdImage(r io.Reader) (image.Image, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if img, err := png.Decode(bytes.NewReader(raw)); err == nil {
		return img, nil
	}
	return jpeg.Decode(bytes.NewReader(raw))
}

// attachmentPublicView is the API shape for attachments. StorageRef (an
// absolute server path) is deliberately never exposed to clients.
type attachmentPublicView struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Name      string `json:"name"`
	MimeType  string `json:"mime_type"`
	Size      int64  `json:"size"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
	URL       string `json:"url"`
}

func publicAttachmentView(att *store.ConversationAttachment) attachmentPublicView {
	return attachmentPublicView{
		ID:        att.ID,
		SessionID: att.SessionID,
		Name:      att.Name,
		MimeType:  att.MimeType,
		Size:      att.Size,
		Type:      att.Type,
		CreatedAt: att.CreatedAt.UTC().Format(time.RFC3339Nano),
		URL:       attachmentURL(att.ID),
	}
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
		views := make([]attachmentPublicView, 0, len(items))
		for i := range items {
			views = append(views, publicAttachmentView(&items[i]))
		}
		writeJSON(w, http.StatusOK, views)
	case http.MethodPost:
		h.uploadComposerAttachment(w, r, sessionID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *NexusHandler) uploadComposerAttachment(w http.ResponseWriter, r *http.Request, sessionID string) {
	if !validAttachmentScope(sessionID) {
		writeError(w, http.StatusNotFound, "composer session not found")
		return
	}
	// Cap the whole request body: ParseMultipartForm alone cannot bound the
	// total bytes a hostile client sends.
	r.Body = http.MaxBytesReader(w, r.Body, attachmentMaxRequestBytes)
	if err := r.ParseMultipartForm(attachmentMaxRequestBytes); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, attachmentMaxFileBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "unreadable upload")
		return
	}
	if len(raw) == 0 {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	if len(raw) > attachmentMaxFileBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file too large")
		return
	}
	// Decode and re-encode before touching disk or DB: only pixels verified
	// by the server decoder are ever stored, and the MIME type below is the
	// server's verdict, never the client's claim.
	normalized, mimeType, err := normalizeAttachmentImage(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "only PNG, JPEG and WEBP images are allowed")
		return
	}
	if len(normalized) > attachmentMaxFileBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file too large")
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
	if existing, err := st.ListConversationAttachments(sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	} else {
		var total int64
		for i := range existing {
			total += existing[i].Size
		}
		if len(existing) >= attachmentMaxPerSession || total+int64(len(normalized)) > attachmentMaxSessionBytes {
			writeError(w, http.StatusTooManyRequests, "attachment quota for this session is exhausted")
			return
		}
	}
	dir := filepath.Join(dataDir, "attachments", sessionID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// The physical name is a fresh crypto-random ID, never the client name.
	id := "att_" + ids.NewULID()
	finalPath := filepath.Join(dir, id)
	tmpPath := finalPath + ".part"
	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := out.Write(normalized); err != nil {
		_ = out.Close()
		_ = os.Remove(tmpPath)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	created, err := st.CreateConversationAttachment(store.ConversationAttachment{
		ID:         id,
		SessionID:  sessionID,
		Name:       sanitizeAttachmentName(header.Filename),
		MimeType:   mimeType,
		Size:       int64(len(normalized)),
		Type:       "image",
		StorageRef: finalPath,
	})
	if err != nil {
		_ = os.Remove(tmpPath)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		_ = st.DeleteConversationAttachment(created.ID)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, publicAttachmentView(created))
}

func (h *NexusHandler) handleAttachmentByID(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/attachments/"), "/")
	if !validAttachmentID(id) {
		writeError(w, http.StatusNotFound, "attachment not found")
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
		h.serveAttachment(w, r, st, att)
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

// serveAttachment streams stored bytes with server-determined headers. The
// MIME type comes from the verified re-encode at upload time, never from the
// client, and the file is served by explicit content (not MIME-sniffed path
// dispatch) with nosniff plus an explicit cache policy.
func (h *NexusHandler) serveAttachment(w http.ResponseWriter, r *http.Request, st *store.Store, att *store.ConversationAttachment) {
	if _, err := st.GetComposerSession(att.SessionID); err != nil {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}
	dataDir, err := coreconfig.DataDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !attachmentPathInsideDataDir(dataDir, att.StorageRef) {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}
	content, err := os.ReadFile(att.StorageRef)
	if err != nil {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}
	mimeType := att.MimeType
	if !allowedAttachmentMIME[mimeType] {
		mimeType = "application/octet-stream"
	}
	ext := ".png"
	if mimeType == "image/jpeg" {
		ext = ".jpg"
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="attachment-%s%s"`, att.ID, ext))
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, att.ID+ext, att.CreatedAt, bytes.NewReader(content))
}

// attachmentPathInsideDataDir confines served files to the attachments tree
// even if a storage_ref value was tampered with outside the upload path.
func attachmentPathInsideDataDir(dataDir, storageRef string) bool {
	base, err := filepath.Abs(filepath.Join(dataDir, "attachments"))
	if err != nil {
		return false
	}
	target, err := filepath.Abs(storageRef)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

// Ensure nexus.ComposerProgressSink is referenced for compile clarity.
var _ nexus.ComposerProgressSink = (*ndjsonComposerProgress)(nil)

func attachmentURL(id string) string {
	return fmt.Sprintf("/api/v1/attachments/%s", id)
}
