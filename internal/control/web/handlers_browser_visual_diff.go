package web

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/browsercontrol"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

// maxVisualDiffBody caps the request: two base64 captures plus metadata.
const maxVisualDiffBody = 6 * 1024 * 1024

// maxVisualDiffDelay bounds the server-side recheck window.
const maxVisualDiffDelay = 5 * time.Second

type visualDiffRequest struct {
	// Mode A: explicit before/after captures (base64 JPEG), fully traceable.
	BeforeB64 string `json:"before_b64,omitempty"`
	AfterB64  string `json:"after_b64,omitempty"`
	// Mode B: the server captures now, waits, captures again.
	RecheckDelayMS int `json:"recheck_delay_ms,omitempty"`

	Region *nexus.VisualDiffRegion `json:"region,omitempty"`
	// Region and thresholds must be declared; the endpoint never invents them.
	PixelThreshold  *uint8   `json:"pixel_threshold"`
	MaxDiffRatio    *float64 `json:"max_diff_ratio"`
	IncludeArtifact bool     `json:"include_artifact"`
}

// handleBrowserVisualDiff POST /api/v1/browser/{id}/visual-diff
//
// Deterministic screenshot comparison (capability visual-diff). Region and
// thresholds are declared by the caller and echoed back in the result; the
// diff artifact (grayscale + red highlights) is optional and returned as
// base64 PNG for review. The session is required by the dispatcher; the
// explicit-pair mode itself never touches it.
func (s *Server) handleBrowserVisualDiff(w http.ResponseWriter, r *http.Request, session *browsercontrol.Session) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxVisualDiffBody)
	var req visualDiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid visual-diff request: "+err.Error())
		return
	}
	if req.PixelThreshold == nil || req.MaxDiffRatio == nil {
		writeError(w, http.StatusBadRequest, "pixel_threshold and max_diff_ratio must be declared")
		return
	}
	input := nexus.VisualDiffInput{
		Region:         req.Region,
		PixelThreshold: *req.PixelThreshold,
		MaxDiffRatio:   *req.MaxDiffRatio,
	}

	var before, after []byte
	switch {
	case req.BeforeB64 != "" || req.AfterB64 != "":
		if req.BeforeB64 == "" || req.AfterB64 == "" {
			writeError(w, http.StatusBadRequest, "visual diff requires both before_b64 and after_b64")
			return
		}
		var err error
		if before, err = decodeBase64Capture(req.BeforeB64); err != nil {
			writeError(w, http.StatusBadRequest, "invalid before_b64: "+err.Error())
			return
		}
		if after, err = decodeBase64Capture(req.AfterB64); err != nil {
			writeError(w, http.StatusBadRequest, "invalid after_b64: "+err.Error())
			return
		}
	case req.RecheckDelayMS > 0:
		if req.RecheckDelayMS > int(maxVisualDiffDelay.Milliseconds()) {
			writeError(w, http.StatusBadRequest, "recheck_delay_ms exceeds the 5s window")
			return
		}
		if session == nil {
			writeError(w, http.StatusNotFound, "browser session not found")
			return
		}
		var err error
		if before, err = session.Screenshot(r.Context()); err != nil {
			writeError(w, http.StatusBadGateway, "browser screenshot unavailable")
			return
		}
		time.Sleep(time.Duration(req.RecheckDelayMS) * time.Millisecond)
		if after, err = session.Screenshot(r.Context()); err != nil {
			writeError(w, http.StatusBadGateway, "browser screenshot unavailable")
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "provide before_b64/after_b64 or recheck_delay_ms")
		return
	}

	result, err := nexus.CompareScreenshots(before, after, input)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response := map[string]any{"result": result}
	if req.IncludeArtifact {
		artifact, _, err := nexus.RenderDiffArtifact(before, after, input)
		if err == nil {
			response["artifact_b64"] = base64.StdEncoding.EncodeToString(artifact)
			response["artifact_format"] = "png"
		}
		// Artifact failure does not fail the comparison: the metrics are
		// the source of truth and the artifact is advisory for review.
	}
	writeJSON(w, http.StatusOK, response)
}

// decodeBase64Capture decodes a standard-base64 capture and rejects
// data-URL prefixes and padding tricks; the decode itself is format-checked
// downstream by image/jpeg/image/png.
func decodeBase64Capture(value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.Contains(trimmed, "data:") {
		return nil, base64.CorruptInputError(0)
	}
	return base64.StdEncoding.DecodeString(trimmed)
}
