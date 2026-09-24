package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/IAPro-Community/nexus/internal/voice"
)

const maxVoiceUploadBytes = (25 << 20) + (1 << 20)

func (h *NexusHandler) handleVoiceSettings(w http.ResponseWriter, r *http.Request) {
	if h.voice == nil {
		writeError(w, http.StatusServiceUnavailable, "voice service unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		view, err := h.voice.Settings()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load voice settings")
			return
		}
		writeJSON(w, http.StatusOK, view)
	case http.MethodPut:
		var update voice.SettingsUpdate
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&update); err != nil {
			writeError(w, http.StatusBadRequest, "invalid voice settings")
			return
		}
		view, err := h.voice.UpdateSettings(update)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, view)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *NexusHandler) handleVoiceTranscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.voice == nil {
		writeError(w, http.StatusServiceUnavailable, "voice service unavailable")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxVoiceUploadBytes)
	if err := r.ParseMultipartForm(maxVoiceUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid or oversized voice upload")
		return
	}
	file, header, err := r.FormFile("audio")
	if err != nil {
		writeError(w, http.StatusBadRequest, "audio file is required")
		return
	}
	defer file.Close()
	audio, err := io.ReadAll(io.LimitReader(file, (25<<20)+1))
	if err != nil || len(audio) == 0 {
		writeError(w, http.StatusBadRequest, "failed to read audio")
		return
	}
	if len(audio) > 25<<20 {
		writeError(w, http.StatusRequestEntityTooLarge, "audio exceeds 25 MiB safe transcription limit")
		return
	}
	durationMS, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("duration_ms")), 10, 64)
	result, err := h.voice.Transcribe(r.Context(), voice.TranscribeRequest{
		Audio:      audio,
		Filename:   header.Filename,
		MIMEType:   header.Header.Get("Content-Type"),
		Language:   r.FormValue("language"),
		DurationMS: durationMS,
	})
	if err != nil {
		if txErr, ok := err.(*voice.TranscriptionError); ok {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error":    txErr.Message,
				"code":     "VOICE_TRANSCRIPTION_FAILED",
				"attempts": txErr.Attempts,
			})
			return
		}
		writeError(w, http.StatusInternalServerError, "voice transcription failed")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
