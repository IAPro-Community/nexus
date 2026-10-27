package artifacts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestOutputHistoryVersionsAndAgentProvenance(t *testing.T) {
	root, dir := makeTestRoot(t)
	dataDir := t.TempDir()
	item, err := Describe(root, "images/a.png")
	if err != nil {
		t.Fatal(err)
	}
	published, err := Observe(root, dataDir, "project-1", item, "agent", "agent-1", "runtime-1")
	if err != nil || published.Revision != 1 || published.RuntimeID != "runtime-1" {
		t.Fatalf("first revision: %+v %v", published, err)
	}
	first, err := os.ReadFile(filepath.Join(dir, "images", "a.png"))
	if err != nil {
		t.Fatal(err)
	}
	prev, err := Observe(root, dataDir, "project-1", item, "filesystem", "", "")
	if err != nil || prev.Revision != 1 || prev.Origin != "agent" || prev.RuntimeID != "runtime-1" {
		t.Fatalf("repeat observation changed provenance: %+v %v", prev, err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	img.Set(3, 3, color.RGBA{G: 200, A: 255})
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "images", "a.png"), buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	item2, err := Describe(root, "images/a.png")
	if err != nil {
		t.Fatal(err)
	}
	v2, err := Observe(root, dataDir, "project-1", item2, "agent", "agent-2", "runtime-2")
	if err != nil || v2.Revision != 2 || v2.Revisions != 2 {
		t.Fatalf("changed file didn't create v2: %+v %v", v2, err)
	}
	rec, err := History(dataDir, "project-1", "images/a.png")
	if err != nil || len(rec.Versions) != 2 || rec.AgentID != "agent-2" {
		t.Fatalf("history: %+v %v", rec, err)
	}
	r := httptest.NewRequest(http.MethodGet, "/revision", nil)
	w := httptest.NewRecorder()
	if err := ServeRevision(w, r, dataDir, "project-1", "images/a.png", 1); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(first)
	if w.Code != 200 || hex.EncodeToString(sum[:]) != rec.Versions[0].SHA256 || !bytes.Equal(w.Body.Bytes(), first) {
		t.Fatal("original bytes not preserved in revision 1")
	}
	if err := ServeRevision(httptest.NewRecorder(), r, dataDir, "project-2", "images/a.png", 1); err == nil {
		t.Fatal("cross-project revision access accepted")
	}
	if err := ServeRevision(httptest.NewRecorder(), r, dataDir, "project-1", "../../secret", 1); err == nil {
		t.Fatal("unsafe path accepted")
	}
}
