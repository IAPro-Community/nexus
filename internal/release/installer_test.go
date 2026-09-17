package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallerScriptSyntax(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Bash script syntax check requires unix shell")
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to determine repository root: %v", err)
	}

	scripts := []string{"install.sh", "uninstall.sh", "scripts/package-linux-beta.sh"}
	for _, s := range scripts {
		scriptPath := filepath.Join(root, s)
		if _, err := os.Stat(scriptPath); err != nil {
			t.Errorf("script %s not found: %v", s, err)
			continue
		}
		cmd := exec.Command("bash", "-n", scriptPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("bash syntax error in %s: %v\nOutput:\n%s", s, err, string(output))
		}
	}
}

func TestInstallerArchiveNamingMatchesGoReleaser(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to determine repository root: %v", err)
	}

	goreleaserBytes, err := os.ReadFile(filepath.Join(root, ".goreleaser.yaml"))
	if err != nil {
		t.Fatalf("failed to read .goreleaser.yaml: %v", err)
	}
	goreleaserContent := string(goreleaserBytes)

	installShBytes, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatalf("failed to read install.sh: %v", err)
	}
	installSh := string(installShBytes)

	installPs1Bytes, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatalf("failed to read install.ps1: %v", err)
	}
	installPs1 := string(installPs1Bytes)

	// Verify archive template and formats
	expectedPatterns := []struct {
		platform string
		archive  string
	}{
		{"Linux x86_64", "nexus_Linux_x86_64.tar.gz"},
		{"Linux arm64", "nexus_Linux_arm64.tar.gz"},
		{"Darwin x86_64", "nexus_Darwin_x86_64.tar.gz"},
		{"Darwin arm64", "nexus_Darwin_arm64.tar.gz"},
		{"Windows x86_64", "nexus_Windows_x86_64.zip"},
		{"Windows arm64", "nexus_Windows_arm64.zip"},
	}

	for _, p := range expectedPatterns {
		// Verify install.sh constructs nexus_${OS_NAME}_${ARCH_NAME}.tar.gz
		if !strings.Contains(installSh, `ARCHIVE_NAME="nexus_${OS_NAME}_${ARCH_NAME}.tar.gz"`) {
			t.Errorf("install.sh missing archive format pattern for %s", p.platform)
		}
		// Verify install.ps1 constructs nexus_${OsName}_${Arch}.${ArchiveExt}
		if !strings.Contains(installPs1, `nexus_${OsName}_${Arch}.${ArchiveExt}`) {
			t.Errorf("install.ps1 missing archive format pattern for %s", p.platform)
		}
	}

	// Verify nfpms package section exists in goreleaser
	if !strings.Contains(goreleaserContent, "nfpms:") {
		t.Errorf(".goreleaser.yaml should define nfpms package section for Linux packages")
	}
}

func TestInstallerDoesNotSilentlyInstallMaestro(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to determine repository root: %v", err)
	}

	installShBytes, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatalf("failed to read install.sh: %v", err)
	}
	installSh := string(installShBytes)

	installPs1Bytes, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatalf("failed to read install.ps1: %v", err)
	}
	installPs1 := string(installPs1Bytes)

	if !strings.Contains(installSh, "--with-maestro") {
		t.Error("install.sh must guard Maestro installation behind --with-maestro")
	}
	if !strings.Contains(installSh, `[ "$WITH_MAESTRO" = true ]`) {
		t.Error("install.sh must only execute Maestro installation when WITH_MAESTRO is true")
	}

	if !strings.Contains(installPs1, "WithMaestro") {
		t.Error("install.ps1 must declare WithMaestro parameter")
	}
	if !strings.Contains(installPs1, "if ($WithMaestro)") {
		t.Error("install.ps1 must only execute Maestro installation when $WithMaestro is true")
	}
}

func TestInstallersRequirePinnedArtifactsOrExplicitSourceBuild(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to determine repository root: %v", err)
	}

	sh, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	ps, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	shText, psText := string(sh), string(ps)

	for _, forbidden := range []string{"releases/latest/download", "@latest", "git clone --depth 1 \"${GITHUB_URL}.git\""} {
		if strings.Contains(shText, forbidden) {
			t.Errorf("install.sh must not use mutable source/artifact reference %q", forbidden)
		}
	}
	for _, forbidden := range []string{"releases/latest/download", "@latest", "git clone --depth 1 \"$GithubUrl.git\""} {
		if strings.Contains(psText, forbidden) {
			t.Errorf("install.ps1 must not use mutable source/artifact reference %q", forbidden)
		}
	}

	for _, required := range []string{"--version=", "--build-from-source", "update-manifest.json", "update-manifest.sig", "sha256sum", "shasum -a 256", "resolve_latest_version", "--no-path", "NEXUS_RELEASE_REPO", "make build", "NEXUS_PUBKEY", "refusing unsigned install"} {
		if !strings.Contains(shText, required) {
			t.Errorf("install.sh missing pinned/digest guard %q", required)
		}
	}
	for _, forbidden := range []string{"Falling back to unsigned checksums.txt", "skipping signature verification", "skipping hash verification"} {
		if strings.Contains(shText, forbidden) {
			t.Errorf("install.sh must not weaken trust with %q", forbidden)
		}
	}
	for _, required := range []string{"-Version", "-BuildFromSource", "update-manifest.json", "update-manifest.sig", "Get-FileHash", "Resolve-LatestVersion", "NEXUS_RELEASE_REPO", "make build", "Ensure-Bun", "NexusPubKey", "refusing unsigned"} {
		if !strings.Contains(psText, required) {
			t.Errorf("install.ps1 missing pinned/digest guard %q", required)
		}
	}
	for _, forbidden := range []string{"falling back to checksums.txt", "skipping signature verification", "skipping hash verification"} {
		if strings.Contains(psText, forbidden) {
			t.Errorf("install.ps1 must not weaken trust with %q", forbidden)
		}
	}
	for _, required := range []string{"Ensure-GoCompiler", "winget", "GoLang.Go", "go.dev/dl"} {
		if !strings.Contains(psText, required) {
			t.Errorf("install.ps1 must install or explain the Go source-build dependency: missing %q", required)
		}
	}
}

func TestInstallerSupportsLatestIntegrityAndPathControls(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	sh, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	ps, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	shText, psText := string(sh), string(ps)

	for _, required := range []string{
		`"$VERSION" = "latest"`,
		"api.github.com/repos",
		"checksum integrity",
		"MAESTRO_EXIT=2",
		"nexus doctor",
		"append_path_rc",
		"--yes",
		"ensure_bun",
		"WebKitGTK",
		"--no-path",
		"NEXUS_RELEASE_REPO",
	} {
		if !strings.Contains(shText, required) {
			t.Errorf("install.sh missing contract element %q", required)
		}
	}
	for _, required := range []string{
		`Version -eq "latest"`,
		"api.github.com/repos",
		"checksum integrity",
		"MaestroExit = 2",
		"nexus doctor",
		"Ensure-WebView2",
		"Ensure-Bun",
		"make build",
		"NEXUS_RELEASE_REPO",
	} {
		if !strings.Contains(psText, required) {
			t.Errorf("install.ps1 missing contract element %q", required)
		}
	}
}

func TestInstallersRequireSignedManifestForDesktop(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	shBytes, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	psBytes, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	sh, ps := string(shBytes), string(psBytes)

	for _, forbidden := range []string{
		"DESKTOP_CHECKSUMS_URL",
		"DesktopChecksumsUrl",
		"desktop-checksums.txt\"",
		"/desktop-checksums.txt",
	} {
		if strings.Contains(sh, forbidden) {
			t.Errorf("install.sh must not trust unsigned desktop checksum sidecars: found %q", forbidden)
		}
		if strings.Contains(ps, forbidden) {
			t.Errorf("install.ps1 must not trust unsigned desktop checksum sidecars: found %q", forbidden)
		}
	}
	for _, required := range []string{
		`download_and_verify_archive "$DESKTOP_ARCHIVE_NAME"`,
		"refusing unsigned desktop-checksums.txt fallback",
		"missing from signed update-manifest",
	} {
		if !strings.Contains(sh+ps, required) {
			t.Errorf("installers missing signed-desktop guard %q", required)
		}
	}
}

func TestCIDesktopJobsFailClosedWithoutUpdatePublicKey(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	ciBytes, err := os.ReadFile(filepath.Join(root, ".github/workflows/ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	ci := string(ciBytes)
	if strings.Contains(ci, "ProductionTrustRootHex=${NEXUS_UPDATE_PUBLIC_KEY:-}") {
		t.Fatal("desktop CI must not default missing updater public key to empty")
	}
	// Linux, macOS (and Windows via PowerShell) must refuse empty trust roots.
	if strings.Count(ci, `test -n "${NEXUS_UPDATE_PUBLIC_KEY}"`) < 2 {
		t.Fatal("Linux and macOS desktop CI must each fail-closed when NEXUS_UPDATE_PUBLIC_KEY is empty")
	}
	if !strings.Contains(ci, "NEXUS_UPDATE_PUBLIC_KEY is required for desktop CI") {
		t.Fatal("desktop CI must document fail-closed updater key requirement")
	}
	if !strings.Contains(ci, "desktop-linux:") {
		t.Fatal("expected desktop-linux job in ci.yml")
	}
	linuxIdx := strings.Index(ci, "desktop-linux:")
	windowsIdx := strings.Index(ci, "desktop-windows:")
	if linuxIdx < 0 || windowsIdx < 0 || windowsIdx <= linuxIdx {
		t.Fatal("could not locate desktop-linux job block")
	}
	linuxBlock := ci[linuxIdx:windowsIdx]
	if !strings.Contains(linuxBlock, `test -n "${NEXUS_UPDATE_PUBLIC_KEY}"`) {
		t.Fatal("desktop-linux job must fail-closed when NEXUS_UPDATE_PUBLIC_KEY is empty")
	}
}

func TestInstallersFailClosedWhenSignedManifestCannotBeVerified(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	shBytes, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	psBytes, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	sh, ps := string(shBytes), string(psBytes)

	for _, forbidden := range []string{
		"skipping signature verification",
		"skipping verification",
		"no Ed25519 verification tool available, skipping",
		"falling back to unsigned checksums",
		"Warning: no checksum available, skipping hash verification",
	} {
		if strings.Contains(sh, forbidden) {
			t.Errorf("install.sh must not accept unverifiable releases: found %q", forbidden)
		}
		if strings.Contains(ps, forbidden) {
			t.Errorf("install.ps1 must not accept unverifiable releases: found %q", forbidden)
		}
	}

	for _, required := range []string{
		"return 1",
		"verify_manifest_signature",
		"update-manifest.sig",
		"throw \"Release manifest signature verification failed.\"",
	} {
		if !strings.Contains(sh+ps, required) {
			t.Errorf("installers missing fail-closed signature guard %q", required)
		}
	}
}

func TestPowerShellInstallerUsesNexusPathAndPreservesLegacyPath(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, required := range []string{
		`"Programs\IAPro Nexus"`,
		`"Programs\ai-cli"`,
		"leaving it untouched",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("install.ps1 must preserve legacy compatibility while using Nexus branding: missing %q", required)
		}
	}
}

func TestPublicLicenseMetadataMatchesLicenseFile(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	license, err := os.ReadFile(filepath.Join(root, "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(license), "# IAPro Nexus — Proprietary Beta License") {
		t.Fatalf("expected the vigente LICENSE file to be the proprietary beta license, got %q", strings.SplitN(string(license), "\n", 2)[0])
	}

	for _, name := range []string{"README.md", "README.en.md", "README.es.md"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if strings.Contains(strings.ToLower(text), "apache") {
			t.Errorf("%s still advertises Apache despite the proprietary LICENSE", name)
		}
		if strings.Contains(text, "MIT License") || strings.Contains(text, "Licencia MIT") || strings.Contains(text, "licença MIT") {
			t.Errorf("%s still advertises MIT despite the proprietary LICENSE", name)
		}
		if !strings.Contains(strings.ToLower(text), "licen") {
			t.Errorf("%s does not reference the vigente license", name)
		}
	}
}

// TestReleaseBuildsStripLocalPaths guards the closed-binary requirement:
// every Go build that can reach beta users must pass -trimpath as a BUILD
// flag (never inside -ldflags, which the linker rejects) so build machine
// paths never embed in distributed artifacts.
func TestReleaseBuildsStripLocalPaths(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	goreleaser, err := os.ReadFile(filepath.Join(root, ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goreleaser), "-trimpath") {
		t.Error(".goreleaser.yaml builds must pass -trimpath")
	}
	for _, line := range strings.Split(string(goreleaser), "\n") {
		if strings.Contains(line, "-trimpath") && strings.Contains(strings.ToLower(line), "ldflag") {
			t.Errorf(".goreleaser.yaml must not put -trimpath inside ldflags (linker rejects it): %q", strings.TrimSpace(line))
		}
	}
	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(makefile), "\n") {
		if strings.Contains(line, "LDFLAGS=") && strings.Contains(line, "-trimpath") {
			t.Errorf("Makefile must pass -trimpath as a build flag, not inside LDFLAGS: %q", strings.TrimSpace(line))
		}
	}
	if !strings.Contains(string(makefile), "go build -trimpath") {
		t.Error("Makefile go builds must pass -trimpath")
	}
	if !strings.Contains(string(makefile), "GOFLAGS=-trimpath") {
		t.Error("Makefile wails build must forward -trimpath via GOFLAGS")
	}
	ci, err := os.ReadFile(filepath.Join(root, ".github/workflows/ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{"go build -trimpath -o nexus", "go build -trimpath -v"} {
		if !strings.Contains(string(ci), cmd) {
			t.Errorf("ci.yml must build test binaries with %q", cmd)
		}
	}
	// Wails desktop builds (promoted into releases) must also strip paths.
	if !strings.Contains(string(ci), "export GOFLAGS=-trimpath") {
		t.Error("ci.yml macOS wails build must forward -trimpath via GOFLAGS")
	}
	if !strings.Contains(string(ci), "$env:GOFLAGS = '-trimpath'") {
		t.Error("ci.yml Windows wails build must forward -trimpath via GOFLAGS")
	}
}
