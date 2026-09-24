package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPKGInstallationIsPackageManagedAndBlocksSelfUpdate(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "nexus")
	if err := os.WriteFile(execPath, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".nexus-pkg"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	method := DetectInstallationMethod(execPath)
	if method != MethodPKG {
		t.Fatalf("DetectInstallationMethod()=%s, want %s", method, MethodPKG)
	}
	if !method.IsPackageManaged() {
		t.Fatal("PKG install must be package-managed")
	}
	if method.AllowsSelfUpdate() {
		t.Fatal("PKG install must not allow in-place self-update")
	}
	if got := method.UpgradeInstruction(); !strings.Contains(strings.ToLower(got), "pkg") {
		t.Fatalf("PKG upgrade instruction must direct to native installer, got %q", got)
	}
}
