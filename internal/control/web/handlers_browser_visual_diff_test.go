package web

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"net/http/httptest"
	"testing"

	"github.com/IAPro-Community/nexus/internal/browsercontrol"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

func jpegBase64(t *testing.T, width, height int, c color.RGBA) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func visualDiffHandler() (*Server, *NexusHandler) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		panic(err)
	}
	srv := &Server{auth: auth, browsers: browsercontrol.NewManager()}
	handler := NewNexusHandler(auth)
	return srv, handler
}

func TestBrowserVisualDiffExplicitPair(t *testing.T) {
	srv, _ := visualDiffHandler()
	threshold := uint8(4)
	maxRatio := 0.01
	body, err := json.Marshal(visualDiffRequest{
		BeforeB64:       jpegBase64(t, 8, 8, color.RGBA{R: 10, G: 20, B: 30, A: 255}),
		AfterB64:        jpegBase64(t, 8, 8, color.RGBA{R: 10, G: 20, B: 30, A: 255}),
		PixelThreshold:  &threshold,
		MaxDiffRatio:    &maxRatio,
		IncludeArtifact: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/browser/sessions/00000000000000000000000000000000/visual-diff", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.handleBrowserVisualDiff(rec, req, nil)
	if rec.Code != 200 {
		t.Fatalf("identical pair must pass: %d (%s)", rec.Code, rec.Body.String())
	}
	var payload struct {
		Result         nexus.VisualDiffResult `json:"result"`
		ArtifactFormat string                 `json:"artifact_format"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Result.Passed || payload.Result.DifferentPixels != 0 {
		t.Fatalf("identical captures must pass: %+v", payload.Result)
	}
	if payload.ArtifactFormat != "png" {
		t.Fatalf("artifact must be rendered: %+v", payload)
	}

	// Size mismatch is an explicit 422.
	body, err = json.Marshal(visualDiffRequest{
		BeforeB64:      jpegBase64(t, 8, 8, color.RGBA{R: 1, G: 2, B: 3, A: 255}),
		AfterB64:       jpegBase64(t, 4, 4, color.RGBA{R: 1, G: 2, B: 3, A: 255}),
		PixelThreshold: &threshold,
		MaxDiffRatio:   &maxRatio,
	})
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest("POST", "http://127.0.0.1/x", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	srv.handleBrowserVisualDiff(rec, req, nil)
	if rec.Code != 422 {
		t.Fatalf("size mismatch must be 422, got %d", rec.Code)
	}

	// Undeclared thresholds are rejected (region/threshold must be explicit).
	body, err = json.Marshal(visualDiffRequest{
		BeforeB64: jpegBase64(t, 4, 4, color.RGBA{R: 1, G: 1, B: 1, A: 255}),
		AfterB64:  jpegBase64(t, 4, 4, color.RGBA{R: 1, G: 1, B: 1, A: 255}),
	})
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest("POST", "http://127.0.0.1/x", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	srv.handleBrowserVisualDiff(rec, req, nil)
	if rec.Code != 400 {
		t.Fatalf("missing thresholds must be 400, got %d", rec.Code)
	}

	// Mode without captures and without delay is rejected.
	body, err = json.Marshal(visualDiffRequest{PixelThreshold: &threshold, MaxDiffRatio: &maxRatio})
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest("POST", "http://127.0.0.1/x", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	srv.handleBrowserVisualDiff(rec, req, nil)
	if rec.Code != 400 {
		t.Fatalf("mode-less request must be 400, got %d", rec.Code)
	}

	// GET is never allowed.
	req = httptest.NewRequest("GET", "http://127.0.0.1/x", nil)
	rec = httptest.NewRecorder()
	srv.handleBrowserVisualDiff(rec, req, nil)
	if rec.Code != 405 {
		t.Fatalf("GET must be 405, got %d", rec.Code)
	}
}
