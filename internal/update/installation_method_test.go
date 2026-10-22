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

func TestDesktopAndHUDBinariesNeverSelfUpdateAsCLI(t *testing.T) {
	for _, name := range []string{"nexus-desktop", "nexus-hud", "nexus-desktop.exe", "nexus-hud.exe"} {
		t.Run(name, func(t *testing.T) {
			method := DetectInstallationMethod(filepath.Join(t.TempDir(), name))
			if method != MethodDesktop {
				t.Fatalf("DetectInstallationMethod(%q)=%s, want %s", name, method, MethodDesktop)
			}
			if method.AllowsSelfUpdate() {
				t.Fatal("GUI process must not overwrite itself with the signed CLI artifact")
			}
			if !method.IsPackageManaged() {
				t.Fatal("GUI process must use native installation flow")
			}
			if got := method.UpgradeInstruction(); !strings.Contains(got, "GitHub Releases") {
				t.Fatalf("GUI upgrade should direct to official release: %q", got)
			}
		})
	}
}

func TestDesktopSelfUpdateCannotBeForcedThroughInstallMethodOverride(t *testing.T) {
	t.Setenv("NEXUS_INSTALL_METHOD", "STANDALONE")
	for _, name := range []string{"nexus-desktop", "nexus-hud.exe"} {
		method := DetectInstallationMethod(filepath.Join(t.TempDir(), name))
		if method != MethodDesktop || method.AllowsSelfUpdate() {
			t.Fatalf("%s: override must not enable GUI binary replacement, got %s", name, method)
		}
	}
}
