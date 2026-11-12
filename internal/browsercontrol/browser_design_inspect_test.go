package browsercontrol

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestInspectRejectsEmptyOrOversizedSelectorBeforeCDP(t *testing.T) {
	s := &Session{}
	for _, selector := range []string{"", "   ", string(bytes.Repeat([]byte{'a'}, 2049))} {
		if _, err := s.Inspect(context.Background(), selector); err == nil {
			t.Fatalf("Inspect(%d bytes) accepted invalid selector", len(selector))
		}
	}
}

func TestChromiumDesignInspectOptional(t *testing.T) {
	if os.Getenv("NEXUS_BROWSER_INTEGRATION") != "1" {
		t.Skip("set NEXUS_BROWSER_INTEGRATION=1 for real Chromium design-inspect test")
	}
	if _, err := chromeExecutable(); err != nil {
		t.Skipf("no Chromium available: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body><section id="card" style="width:240px;height:120px;padding:8px;background:#eee"><h1>Welcome person@example.test</h1><input type="email" value="secret@example.test"><button id="target">Inspect me</button></section></body></html>`))
	}))
	defer server.Close()
	m := NewManager()
	defer m.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	info, err := m.Create(ctx, "design-inspect-test", server.URL)
	if err != nil {
		t.Fatalf("create Chromium: %v", err)
	}
	session, ok := m.Get(info.ID)
	if !ok {
		t.Fatal("created browser session missing")
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if _, err = session.Snapshot(ctx); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	inspection, err := session.Inspect(ctx, "#card")
	if err != nil {
		t.Fatalf("inspect element: %v", err)
	}
	if !inspection.MaskingApplied || inspection.Tag != "section" || inspection.ComputedStyles["width"] != "240px" {
		t.Fatalf("unexpected design inspection: %#v", inspection)
	}
	if !strings.Contains(inspection.Text, "[REDACTED_EMAIL]") || strings.Contains(inspection.Text, "person@example.test") {
		t.Fatalf("text was not sanitized: %q", inspection.Text)
	}
	if len(inspection.CropJPEG) < 100 || !bytes.HasPrefix(inspection.CropJPEG, []byte{0xff, 0xd8}) {
		t.Fatalf("expected a JPEG element crop, got %d bytes", len(inspection.CropJPEG))
	}
}

func TestMaskDesignCropOpaqueMasksSensitiveRect(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 20, 20))
	draw.Draw(base, base.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, base, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	clean, err := maskDesignCrop(encoded.Bytes(), viewportRect{X: 10, Y: 10, Width: 20, Height: 20}, []viewportRect{{X: 15, Y: 15, Width: 8, Height: 8}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(clean))
	if err != nil {
		t.Fatal(err)
	}
	masked := color.RGBAModel.Convert(decoded.At(8, 8)).(color.RGBA)
	if masked.R > 90 || masked.G > 100 || masked.B > 110 {
		t.Fatalf("sensitive pixel not obscured: %+v", masked)
	}
	clear := color.RGBAModel.Convert(decoded.At(1, 1)).(color.RGBA)
	if clear.R < 220 || clear.G < 220 || clear.B < 220 {
		t.Fatalf("unmasked pixels were changed: %+v", clear)
	}
}
