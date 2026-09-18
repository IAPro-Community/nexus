package web

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVerifyCloudflaredChecksumUsesPlatformPin(t *testing.T) {
	key := cloudflaredChecksumKey()
	if key == "" {
		t.Skipf("no cloudflared checksum key for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	path := filepath.Join(t.TempDir(), cloudflaredBinaryName())
	data := []byte("verified cloudflared fixture")
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])

	if runtime.GOOS == "darwin" {
		// Darwin verifies the extracted binary via install sidecar, not the .tgz pin.
		if err := os.WriteFile(path, data, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cloudflaredBinarySidecar(path), []byte(digest+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := verifyCloudflaredChecksum(path); err != nil {
			t.Fatalf("verified fixture rejected: %v", err)
		}
		if err := os.WriteFile(path, []byte("tampered cloudflared fixture"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := verifyCloudflaredChecksum(path); err == nil {
			t.Fatal("tampered cloudflared fixture was accepted")
		}
		return
	}

	previous, existed := cloudflaredChecksums[key]
	cloudflaredChecksums[key] = digest
	t.Cleanup(func() {
		if existed {
			cloudflaredChecksums[key] = previous
		} else {
			delete(cloudflaredChecksums, key)
		}
	})
	if err := os.WriteFile(path, data, 0700); err != nil {
		t.Fatal(err)
	}
	if err := verifyCloudflaredChecksum(path); err != nil {
		t.Fatalf("verified fixture rejected: %v", err)
	}
	if err := os.WriteFile(path, []byte("tampered cloudflared fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := verifyCloudflaredChecksum(path); err == nil {
		t.Fatal("tampered cloudflared fixture was accepted")
	}
}
