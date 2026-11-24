package web

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestWriteErrorPreservesMessageAndAddsStableCode(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeError(recorder, http.StatusNotFound, "project not found")

	var body APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "project not found" || body.Code != "PROJECT_NOT_FOUND" {
		t.Fatalf("unexpected error response: %#v", body)
	}
}

func TestRuntimeDetailResponsePreservesContractFields(t *testing.T) {
	response := RuntimeDetailResponse{}
	if response.Session.RuntimeID != "" || response.Capabilities != nil {
		t.Fatalf("zero value unexpectedly populated: %#v", response)
	}
}

func TestStableErrorCodeDoesNotTreatUnknownAsZero(t *testing.T) {
	if got := stableErrorCode(http.StatusServiceUnavailable, "quota unavailable"); got != "QUOTA_UNKNOWN" {
		t.Fatalf("code = %q, want QUOTA_UNKNOWN", got)
	}
	if got := stableErrorCode(http.StatusInternalServerError, "something failed"); got != "INTERNAL_ERROR" {
		t.Fatalf("code = %q, want INTERNAL_ERROR", got)
	}
	for _, tc := range []struct {
		message string
		want    string
	}{
		{message: "INTERVENTION_ALREADY_RESOLVED: existing resolution", want: "INTERVENTION_ALREADY_RESOLVED"},
		{message: "STALE_INTERVENTION: version mismatch", want: "STALE_INTERVENTION"},
		{message: "provider dispatch outcome is unknown", want: "UNKNOWN_EXTERNAL_OUTCOME"},
		{message: "INTERVENTION_POLICY_DENIED: unsafe retry", want: "INTERVENTION_POLICY_DENIED"},
	} {
		if got := stableErrorCode(http.StatusConflict, tc.message); got != tc.want {
			t.Errorf("message %q: code = %q, want %q", tc.message, got, tc.want)
		}
	}
}

// Project outputs are readable only by an authenticated identity and only
// inside the project that owns the artifact. The MIME of the original bytes
// is preserved so a downloaded file is not re-interpreted by the browser.
func TestProjectFilesEnforceIdentityAndProjectScope(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("AI_MANAGER_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	nexus.ResetDefaultForTest()
	t.Cleanup(nexus.ResetDefaultForTest)

	writePNG := func(dir, name string, shade color.RGBA) []byte {
		t.Helper()
		buf := new(bytes.Buffer)
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		img.Set(0, 0, shade)
		if err := png.Encode(buf, img); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}

	dirA := t.TempDir()
	dirB := t.TempDir()
	bytesA := writePNG(dirA, "a.png", color.RGBA{R: 12, G: 34, B: 56, A: 255})
	bytesB := writePNG(dirB, "a.png", color.RGBA{R: 200, G: 100, B: 20, A: 255})

	n := nexus.Default()
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectA, err := st.CreateProject(store.Project{Name: "Outputs A", CanonicalPath: dirA})
	if err != nil {
		t.Fatal(err)
	}
	projectB, err := st.CreateProject(store.Project{Name: "Outputs B", CanonicalPath: dirB})
	if err != nil {
		t.Fatal(err)
	}

	srv, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: 0, NoOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = srv.Shutdown(context.Background()) }()

	handler := srv.Handler()
	host := strings.TrimPrefix(srv.URL(), "http://")
	request := func(path, sessionID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = host
		if sessionID != "" {
			req.Header.Set("Authorization", "Bearer "+sessionID)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	files := "/api/v1/projects/" + projectA.ID + "/files?download&path=a.png"
	if rec := request(files, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated project files status = %d, want 401", rec.Code)
	}

	sess, err := srv.auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	rec := request(files, sess.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("authenticated project files status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("served MIME = %q, want image/png", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("served nosniff = %q", got)
	}
	if !bytes.Equal(rec.Body.Bytes(), bytesA) {
		t.Fatal("project A served different bytes than the artifact it owns")
	}

	other := request("/api/v1/projects/"+projectB.ID+"/files?download&path=a.png", sess.ID)
	if other.Code != http.StatusOK {
		t.Fatalf("project B files status = %d, want 200", other.Code)
	}
	if !bytes.Equal(other.Body.Bytes(), bytesB) {
		t.Fatal("project B served bytes owned by project A")
	}
	if bytes.Equal(rec.Body.Bytes(), other.Body.Bytes()) {
		t.Fatal("project scoped outputs are not isolated")
	}

	traversal := "/api/v1/projects/" + projectB.ID + "/files?download&path=..%2F" + filepath.Base(dirA) + "%2Fa.png"
	if rec := request(traversal, sess.ID); rec.Code == http.StatusOK {
		t.Fatalf("project B served another project artifact through traversal")
	}
	if rec := request("/api/v1/projects/unknown-project-id/files?download&path=a.png", sess.ID); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown project files status = %d, want 404", rec.Code)
	}
}
