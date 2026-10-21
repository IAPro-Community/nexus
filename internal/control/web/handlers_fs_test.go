package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFSBrowse(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "my-repo")
	_ = os.MkdirAll(filepath.Join(subDir, ".git"), 0755)
	_ = os.WriteFile(filepath.Join(subDir, "go.mod"), []byte("module my-repo\n"), 0644)

	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fs/browse?path="+tempDir+"&details=full", nil)
	w := httptest.NewRecorder()

	handler.handleFSBrowse(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp FSBrowseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.CurrentPath != tempDir {
		t.Errorf("expected current path %s, got %s", tempDir, resp.CurrentPath)
	}

	if len(resp.Entries) == 0 {
		t.Fatalf("expected entries in %s, got 0", tempDir)
	}

	found := false
	for _, entry := range resp.Entries {
		if entry.Name == "my-repo" {
			found = true
			if !entry.IsDir {
				t.Errorf("expected my-repo to be dir")
			}
			if !entry.IsGit {
				t.Errorf("expected my-repo to be git repo")
			}
		}
	}
	if !found {
		t.Errorf("my-repo not found in browse results")
	}
}

func TestFSBrowseDefaultsToLightweightDirectoryEntries(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "my-repo")
	if err := os.MkdirAll(filepath.Join(subDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "go.mod"), []byte("module my-repo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fs/browse?path="+tempDir, nil)
	w := httptest.NewRecorder()

	handler.handleFSBrowse(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp FSBrowseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Entries) != 1 || resp.Entries[0].Name != "my-repo" {
		t.Fatalf("unexpected entries: %+v", resp.Entries)
	}
	if resp.Entries[0].IsGit || len(resp.Entries[0].Tech) != 0 || resp.Entries[0].ModTime != "" {
		t.Fatalf("lightweight browse unexpectedly computed child metadata: %+v", resp.Entries[0])
	}
}

func TestDetectTechReturnsEmptyArrayWhenNothingIsDetected(t *testing.T) {
	tech := detectTech(t.TempDir())

	if tech == nil {
		t.Fatal("expected an empty JSON-safe array, got nil")
	}
	if len(tech) != 0 {
		t.Fatalf("expected no detected technologies, got %v", tech)
	}
}

func TestFSInspect(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir, ".git"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, ".git", "HEAD"), []byte("ref: refs/heads/feature-xyz\n"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(`{"name":"test-app"}`), 0644)

	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fs/inspect?path="+tempDir, nil)
	w := httptest.NewRecorder()

	handler.handleFSInspect(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp FSInspectResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Exists || !resp.IsDir || !resp.IsGit {
		t.Errorf("expected exists=true, isDir=true, isGit=true; got %+v", resp)
	}

	if resp.GitBranch != "feature-xyz" {
		t.Errorf("expected branch feature-xyz, got %s", resp.GitBranch)
	}
}

func TestFSMkdir(t *testing.T) {
	tempDir := t.TempDir()
	newFolder := filepath.Join(tempDir, "created-folder")

	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)

	bodyBytes, err := json.Marshal(map[string]string{"path": newFolder})
	if err != nil {
		t.Fatalf("failed to encode mkdir request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/mkdir", strings.NewReader(string(bodyBytes)))
	w := httptest.NewRecorder()

	handler.handleFSMkdir(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	if _, err := os.Stat(newFolder); err != nil {
		t.Fatalf("expected folder to exist on disk: %v", err)
	}
}

func TestIsWithinAllowedRoots(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		path   string
		allow  bool
		reason string
	}{
		{filepath.Join(home, "projects", "myapp"), true, "under home"},
		{filepath.Join(os.TempDir(), "something"), true, "under OS temp"},
		{"/etc/cron.d/evil", false, "system directory"},
		{"/proc/self/environ", false, "proc filesystem"},
	}
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			path   string
			allow  bool
			reason string
		}{"/tmp/something", true, "under /tmp"})
	}
	for _, tc := range tests {
		got := isWithinAllowedRoots(tc.path)
		if got != tc.allow {
			t.Errorf("isWithinAllowedRoots(%q) = %v, want %v (%s)", tc.path, got, tc.allow, tc.reason)
		}
	}
}

func TestIsWithinAllowedRootsProjectWorkspace(t *testing.T) {
	tempDir := t.TempDir()
	// Create a .git marker so tempDir looks like a project workspace
	_ = os.MkdirAll(filepath.Join(tempDir, ".git"), 0755)

	child := filepath.Join(tempDir, "subdir", "newdir")
	if !isWithinAllowedRoots(child) {
		t.Errorf("expected child of project workspace to be allowed")
	}
}

func TestIsWithinAllowedRootsOSTempDirectory(t *testing.T) {
	tempRoot := os.TempDir()
	child := filepath.Join(tempRoot, "nexus-allowed-root", "nested")
	if !isWithinAllowedRoots(child) {
		t.Fatalf("expected OS temp directory child %q to be allowed", child)
	}
}

func TestFSMkdirRejectsOutsideAllowedRoots(t *testing.T) {
	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)

	bodyBytes, _ := json.Marshal(map[string]string{"path": "/etc/cron.d/evil"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/mkdir", strings.NewReader(string(bodyBytes)))
	w := httptest.NewRecorder()

	handler.handleFSMkdir(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for path outside allowed roots, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFSMkdirRejectsSymlinkEscape(t *testing.T) {
	link := filepath.Join(t.TempDir(), "workspace-link")
	if err := os.Symlink("/etc", link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)
	bodyBytes, _ := json.Marshal(map[string]string{"path": filepath.Join(link, "nexus-escape")})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/mkdir", strings.NewReader(string(bodyBytes)))
	w := httptest.NewRecorder()

	handler.handleFSMkdir(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for symlink escape, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetGitRemoteRedactsCredentials(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("[remote \"origin\"]\n\turl = https://secret-token@github.com/acme/private.git\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := getGitRemote(dir); got != "https://github.com/acme/private.git" {
		t.Fatalf("remote = %q, credentials must be redacted", got)
	}
}

func TestFSBrowseRejectsOutsideAllowedRoots(t *testing.T) {
	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)

	outside := "/etc"
	if runtime.GOOS == "windows" {
		outside = filepath.VolumeName(os.TempDir()) + string(filepath.Separator)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fs/browse?path="+outside, nil)
	w := httptest.NewRecorder()

	handler.handleFSBrowse(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for path outside allowed roots, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFSBrowseRejectsNonexistentOutsideAllowedRoots(t *testing.T) {
	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)

	var base string
	if runtime.GOOS == "windows" {
		base = filepath.VolumeName(os.TempDir()) + string(filepath.Separator)
	} else {
		base = "/etc"
	}
	outside := filepath.Join(base, "nexus-outside-nonexistent-9f3b")
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Skipf("precondition missing: %q exists", outside)
	}
	if isWithinAllowedRoots(outside) {
		t.Skipf("precondition missing: %q inside allowed roots", outside)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fs/browse?path="+outside, nil)
	w := httptest.NewRecorder()

	handler.handleFSBrowse(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for nonexistent outside path, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFSBrowseWithoutPathFallsBackHomeOutsideAllowedRoots(t *testing.T) {
	outside := "/etc"
	if runtime.GOOS == "windows" {
		outside = filepath.VolumeName(os.TempDir()) + string(filepath.Separator)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Skipf("precondition missing: %q unavailable: %v", outside, err)
	}

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to read working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
	if err := os.Chdir(outside); err != nil {
		t.Skipf("cannot change working directory to %q: %v", outside, err)
	}

	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fs/browse", nil)
	w := httptest.NewRecorder()

	handler.handleFSBrowse(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when browsing without a path away from allowed roots, got %d: %s", w.Code, w.Body.String())
	}

	var resp FSBrowseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !isWithinAllowedRoots(resp.CurrentPath) {
		t.Errorf("default browse path %q is outside allowed roots", resp.CurrentPath)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" && resp.CurrentPath != home {
		t.Errorf("default browse path = %q, want home directory %q", resp.CurrentPath, home)
	}
}

func TestFSBrowseWithoutPathPrefersAllowedWorkingDirectory(t *testing.T) {
	dir := t.TempDir() // under the OS temp dir, always an allowed root

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to read working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	auth, _, _ := NewAuthManager("127.0.0.1", "")
	handler := NewNexusHandler(auth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fs/browse", nil)
	w := httptest.NewRecorder()

	handler.handleFSBrowse(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp FSBrowseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	// os.Getwd reports the physical path on macOS (/private/var/...) while
	// t.TempDir keeps the logical one (/var/...), so compare canonical forms.
	want := dir
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		want = resolved
	}
	if resp.CurrentPath != want {
		t.Errorf("default browse path = %q, want working directory %q", resp.CurrentPath, want)
	}
}

func TestGetOSBookmarksOnlyExposeAllowedRoots(t *testing.T) {
	bookmarks := getOSBookmarks()
	if bookmarks == nil {
		t.Fatal("expected a JSON-safe array, got nil")
	}
	for _, bookmark := range bookmarks {
		if !isWithinAllowedRoots(bookmark.Path) {
			t.Errorf("bookmark %q points to %q, which is outside allowed roots", bookmark.Label, bookmark.Path)
		}
	}
}
