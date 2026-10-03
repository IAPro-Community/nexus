package web

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/computer"
)

func writeComputerScreenshotFixture(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("\x89PNG\r\n\x1a\nfixture"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestComputerScreenshotManagerConfinesAndRetainsArtifacts(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)

	manager := newComputerScreenshotManager()
	root := filepath.Join(dataDir, "computer-use")
	if manager.root != root {
		t.Fatalf("screenshot root=%q want=%q", manager.root, root)
	}

	outside := filepath.Join(t.TempDir(), "outside.png")
	writeComputerScreenshotFixture(t, outside)
	if _, err := manager.register("project-1", outside); err == nil {
		t.Fatal("screenshot outside Nexus data dir was accepted")
	}

	refs := make([]string, 0, computerScreenshotRetentionPerProject+1)
	paths := make([]string, 0, computerScreenshotRetentionPerProject+1)
	for i := 0; i < computerScreenshotRetentionPerProject+1; i++ {
		path := filepath.Join(root, "snapshot-"+time.Now().Add(time.Duration(i)*time.Nanosecond).Format("150405.000000000")+".png")
		writeComputerScreenshotFixture(t, path)
		ref, err := manager.register("project-1", path)
		if err != nil {
			t.Fatalf("register screenshot %d: %v", i, err)
		}
		refs = append(refs, ref)
		paths = append(paths, path)
	}

	if _, ok := manager.path("project-1", refs[0]); ok {
		t.Fatal("oldest screenshot remained addressable after retention limit")
	}
	if _, err := os.Stat(paths[0]); !os.IsNotExist(err) {
		t.Fatalf("oldest screenshot was not removed, err=%v", err)
	}
	lastPath, ok := manager.path("project-1", refs[len(refs)-1])
	if !ok || lastPath == "" {
		t.Fatal("newest screenshot is not addressable")
	}

	manager.closeAll()
	if _, err := os.Stat(lastPath); !os.IsNotExist(err) {
		t.Fatalf("closeAll did not remove retained screenshot, err=%v", err)
	}
}

func TestComputerScreenshotManagerRejectsSymlinkEscapeWhenSupported(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	manager := newComputerScreenshotManager()
	root := filepath.Join(dataDir, "computer-use")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}

	outside := filepath.Join(t.TempDir(), "private.png")
	writeComputerScreenshotFixture(t, outside)
	link := filepath.Join(root, "linked.png")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink creation unavailable on this platform: %v", err)
	}
	if _, err := manager.register("project-1", link); err == nil {
		t.Fatal("symlink escaping screenshot root was accepted")
	}
}

func TestComputerSnapshotResponseUsesOpaqueReferenceOnly(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	root := filepath.Join(dataDir, "computer-use")
	path := filepath.Join(root, "snapshot-safe.png")
	writeComputerScreenshotFixture(t, path)

	handler := &NexusHandler{computerScreenshots: newComputerScreenshotManager()}
	response, err := handler.computerSnapshotResponse("project-1", computer.Snapshot{
		ID:         "snapshot-1",
		Screenshot: path,
		CapturedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.ScreenshotRef != filepath.Base(path) {
		t.Fatalf("screenshot ref=%q", response.ScreenshotRef)
	}
	if response.Snapshot.Screenshot != "" {
		t.Fatalf("local screenshot path leaked in response: %q", response.Snapshot.Screenshot)
	}
}


func TestComputerMutationRoutesRequireCSRFBeforeProviderAccess(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL() + "/api/v1/projects/missing-project/computer/"
	cases := []struct {
		path string
		body string
	}{
		{path: "apps", body: `{"contract":{"allow_computer_use":true}}`},
		{path: "windows", body: `{"contract":{"allow_computer_use":true},"app":"code"}`},
		{path: "snapshot", body: `{"contract":{"allow_computer_use":true},"request":{"app":"code"}}`},
		{path: "action", body: `{"contract":{"allow_computer_use":true,"allow_pointer_input":true},"request":{"kind":"CLICK","app":"code","x":10,"y":10}}`},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, base+tc.path, bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("without CSRF got %d want %d", resp.StatusCode, http.StatusForbidden)
			}

			req, err = http.NewRequest(http.MethodPost, base+tc.path, bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(csrfHeaderName, csrf)
			resp, err = client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("registered route with CSRF got %d want %d", resp.StatusCode, http.StatusNotFound)
			}
		})
	}
}
