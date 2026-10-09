package release

import (
	"encoding/hex"
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

	scripts := []string{
		"install.sh",
		"uninstall.sh",
		"scripts/package-linux-beta.sh",
		"scripts/certify-live-mission.sh",
		"scripts/certify-live-soak.sh",
		"scripts/certify-release-lifecycle.sh",
		"scripts/smoke-providers.sh",
	}
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

func TestInstallerTrustRootMatchesCanonicalAndCI(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := os.ReadFile(filepath.Join(root, "UPDATE_PUBLIC_KEY"))
	if err != nil {
		t.Fatal(err)
	}
	canonical := strings.TrimSpace(string(keyBytes))
	keyIDBytes, err := os.ReadFile(filepath.Join(root, "UPDATE_KEY_ID"))
	if err != nil {
		t.Fatal(err)
	}
	canonicalKeyID := strings.TrimSpace(string(keyIDBytes))
	if canonicalKeyID != "nexus-signing-key-2026-v1" {
		t.Fatalf("UPDATE_KEY_ID=%q does not match updater ProductionTrustRootKeyID", canonicalKeyID)
	}
	decoded, err := hex.DecodeString(canonical)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("UPDATE_PUBLIC_KEY must contain one 32-byte Ed25519 public key in hex")
	}

	shBytes, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	psBytes, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(shBytes), `NEXUS_PUBKEY="`+canonical+`"`) {
		t.Fatal("install.sh trust root does not match UPDATE_PUBLIC_KEY")
	}
	if !strings.Contains(string(psBytes), `$NexusPubKey = "`+canonical+`"`) {
		t.Fatal("install.ps1 trust root does not match UPDATE_PUBLIC_KEY")
	}
	if configured := strings.TrimSpace(os.Getenv("NEXUS_UPDATE_PUBLIC_KEY")); configured != "" && configured != canonical {
		t.Fatalf("CI/release NEXUS_UPDATE_PUBLIC_KEY=%s does not match checked-in UPDATE_PUBLIC_KEY=%s", configured, canonical)
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

	if !strings.Contains(installSh, `ARCHIVE_NAME="nexus_${OS_NAME}_${ARCH_NAME}.tar.gz"`) {
		t.Error("install.sh must construct the Nexus archive name")
	}

	for _, required := range []string{
		"- linux",
		"- amd64",
		"formats: [tar.gz]",
		"nfpms:",
	} {
		if !strings.Contains(goreleaserContent, required) {
			t.Errorf(".goreleaser.yaml missing Nexus v1 Linux release contract %q", required)
		}
	}
	for _, deferred := range []string{
		"- darwin",
		"- windows",
		"- arm64",
		"goos: windows",
		"formats: [zip]",
	} {
		if strings.Contains(goreleaserContent, deferred) {
			t.Errorf(".goreleaser.yaml must not publish deferred Nexus v1 target %q", deferred)
		}
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
	for _, required := range []string{"-Version", "-BuildFromSource", "update-manifest.json", "update-manifest.sig", "Get-FileHash", "Resolve-LatestVersion", "NEXUS_RELEASE_REPO", "Build-NexusFromSource", "bun run --cwd web build", "go build", "Ensure-Bun", "NexusPubKey", "refusing unsigned"} {
		if !strings.Contains(psText, required) {
			t.Errorf("install.ps1 missing pinned/digest guard %q", required)
		}
	}
	for _, forbidden := range []string{"falling back to checksums.txt", "skipping signature verification", "skipping hash verification"} {
		if strings.Contains(psText, forbidden) {
			t.Errorf("install.ps1 must not weaken trust with %q", forbidden)
		}
	}
	for _, forbidden := range []string{"make is required for -BuildFromSource", "Building from source via make build"} {
		if strings.Contains(psText, forbidden) {
			t.Errorf("install.ps1 Windows source build must not require GNU Make: found %q", forbidden)
		}
	}
	for _, forbidden := range []string{"git clone --depth 1 --branch $SourceRef", "git clone --depth 1 --branch"} {
		if strings.Contains(psText, forbidden) {
			t.Errorf("install.ps1 -SourceRef must support commit SHAs, not only branch/tag clone syntax: found %q", forbidden)
		}
	}
	for _, required := range []string{"git -C $CloneDir fetch --depth 1 origin $SourceRef", "git -C $CloneDir checkout --detach FETCH_HEAD"} {
		if !strings.Contains(psText, required) {
			t.Errorf("install.ps1 -SourceRef missing exact-ref fetch contract %q", required)
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
		"signed update-manifest",
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
		"signed update manifest",
		"MaestroExit = 2",
		"nexus doctor",
		"Ensure-WebView2",
		"Ensure-Bun",
		"Build-NexusFromSource",
		"NEXUS_RELEASE_REPO",
	} {
		if !strings.Contains(psText, required) {
			t.Errorf("install.ps1 missing contract element %q", required)
		}
	}
}

func TestInstallersSupportExplicitTargetsAndMacOSDesktopZip(t *testing.T) {
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

	for _, required := range []string{
		"--install-dir=",
		"NEXUS_INSTALL_DIR",
		"NEXUS_RELEASE_BASE_URL",
		"binary version mismatch",
		`*.zip)`,
		`ditto -x -k`,
		`-name '*.app'`,
		`applications_dir="${HOME}/Applications"`,
	} {
		if !strings.Contains(sh, required) {
			t.Errorf("install.sh missing explicit-target/macOS desktop contract %q", required)
		}
	}
	for _, required := range []string{
		`[string]$InstallDir = $env:NEXUS_INSTALL_DIR`,
		`[System.IO.Path]::GetFullPath($InstallDir)`,
		`NEXUS_RELEASE_BASE_URL`,
		`binary version mismatch`,
	} {
		if !strings.Contains(ps, required) {
			t.Errorf("install.ps1 missing explicit install target contract %q", required)
		}
	}
}

func TestInstallerDesktopArchitectureMatrixDoesNotBreakCLI(t *testing.T) {
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

	for _, required := range []string{
		"DESKTOP_SUPPORTED=false",
		"Linux_x86_64|Darwin_arm64",
		"Native Desktop is not published/certified",
		"[ \"$DESKTOP_SUPPORTED\" = true ]",
	} {
		if !strings.Contains(sh, required) {
			t.Errorf("install.sh missing truthful Desktop architecture guard %q", required)
		}
	}
	for _, required := range []string{
		"$Arch -ne \"x86_64\"",
		"$Arch -eq \"x86_64\"",
		"Native Desktop is not published/certified",
	} {
		if !strings.Contains(ps, required) {
			t.Errorf("install.ps1 missing truthful Desktop architecture guard %q", required)
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
	// The Linux beta desktop job must refuse an empty trust root.
	if strings.Count(ci, `test -n "${NEXUS_UPDATE_PUBLIC_KEY}"`) < 1 {
		t.Fatal("Linux desktop CI must fail-closed when NEXUS_UPDATE_PUBLIC_KEY is empty")
	}
	if !strings.Contains(ci, "NEXUS_UPDATE_PUBLIC_KEY is required for desktop CI") {
		t.Fatal("desktop CI must document fail-closed updater key requirement")
	}
	if !strings.Contains(ci, "desktop-linux:") {
		t.Fatal("expected desktop-linux job in ci.yml")
	}
	linuxIdx := strings.Index(ci, "desktop-linux:")
	snapshotIdx := strings.Index(ci, "\n  snapshot:")
	if linuxIdx < 0 || snapshotIdx < 0 || snapshotIdx <= linuxIdx {
		t.Fatal("could not locate desktop-linux job block")
	}
	linuxBlock := ci[linuxIdx:snapshotIdx]
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
	makefileText := string(makefile)
	for _, target := range []string{"build-desktop-wails: web", "build-hud-wails: web"} {
		start := strings.Index(makefileText, target)
		if start < 0 {
			t.Errorf("Makefile missing %s target", strings.TrimSuffix(target, ": web"))
			continue
		}
		section := makefileText[start:]
		if end := strings.Index(section, "\n\n"); end >= 0 {
			section = section[:end]
		}
		if !strings.Contains(section, "go build -trimpath") {
			t.Errorf("%s must invoke go build with -trimpath", strings.TrimSuffix(target, ": web"))
		}
		if !strings.Contains(section, `-ldflags="$$LDFLAGS"`) {
			t.Errorf("%s must preserve shell LDFLAGS expansion with Make escaping", strings.TrimSuffix(target, ": web"))
		}
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
	// The Linux V1 desktop job delegates to the Make target that carries
	// -trimpath into the Wails v3 direct Go build.
	if !strings.Contains(string(ci), "make build-desktop-wails") {
		t.Error("ci.yml Linux Wails build must use the trimpath-enabled Make target")
	}
}

func TestNativeReleaseChainIsFailClosed(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	read := func(path string) string {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return string(data)
	}
	ci := read(".github/workflows/ci.yml")
	release := read(".github/workflows/release.yml")
	nsis := read("packaging/windows/nexus.nsi")
	macUninstall := read("packaging/macos/uninstall.sh")

	for _, required := range []string{
		"environment: native-signing",
		"WINDOWS_SIGNING_PFX_BASE64",
		"signtool.exe",
		"makensis.exe",
		"Authenticode-sign",
		"APPLE_APP_CERT_P12_BASE64",
		"APPLE_INSTALLER_CERT_P12_BASE64",
		"xcrun notarytool submit",
		"xcrun stapler staple",
		"spctl --assess",
		"pkgbuild --root",
		"Smoke signed NSIS install and uninstall",
		"Smoke notarized PKG install and uninstall",
	} {
		if !strings.Contains(ci, required) {
			t.Errorf("native certification missing fail-closed contract %q", required)
		}
	}

	for _, required := range []string{
		"native-ci-run-id",
		"Windows native certification",
		"macOS arm64 native certification",
		"Seal native certification evidence",
		"nexus_Windows_x86_64.zip",
		"nexus-desktop_Windows_x86_64.zip",
		"nexus-hud_Windows_x86_64.zip",
		"nexus-setup-Windows_x86_64.exe",
		"nexus_Darwin_arm64.tar.gz",
		"nexus-desktop_Darwin_arm64.zip",
		"nexus-hud_Darwin_arm64.zip",
		"nexus-installer_Darwin_arm64.pkg",
		"'windows_amd64'",
		"'desktop_windows_amd64'",
		"'hud_windows_amd64'",
		"'windows_amd64_nsis'",
		"'darwin_arm64'",
		"'desktop_darwin_arm64'",
		"'hud_darwin_arm64'",
		"'darwin_arm64_pkg'",
	} {
		if !strings.Contains(release, required) {
			t.Errorf("release workflow missing native exact-SHA contract %q", required)
		}
	}

	for _, required := range []string{
		"Function CaptureOriginalUserPath",
		"Call CaptureOriginalUserPath",
		"Function un.RemoveFromUserPath",
		"Call un.RemoveFromUserPath",
		"UserPathOriginalExists",
		"UserPathOriginalType",
		"UserPathMetadataVersion",
		"RegQueryValueExW",
		"WriteRegStr HKCU \"Environment\" \"Path\" \"$0\"",
		"WriteRegExpandStr HKCU \"Environment\" \"Path\" \"$0\"",
		"DeleteRegValue HKCU \"Environment\" \"Path\"",
		"QuietUninstallString",
		"File /nonfatal \"payload\\nexus-hud.exe\"",
		"Delete \"$INSTDIR\\nexus-hud.exe\"",
		"DeleteRegKey HKCU \"Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\IAPro Nexus\"",
	} {
		if !strings.Contains(nsis, required) {
			t.Errorf("NSIS lifecycle missing %q", required)
		}
	}
	for _, required := range []string{
		"rm -f /usr/local/bin/nexus",
		"rm -rf \"/Applications/IAPro Nexus.app\"",
		"rm -rf \"/Applications/IAPro Nexus HUD.app\"",
		"pkgutil --forget com.iapro.nexus",
		"User profiles and project data were preserved",
	} {
		if !strings.Contains(macUninstall, required) {
			t.Errorf("macOS uninstaller missing %q", required)
		}
	}
}

func TestStagedNativeReleasePolicy(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	ciBytes, err := os.ReadFile(filepath.Join(root, ".github/workflows/ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	releaseBytes, err := os.ReadFile(filepath.Join(root, ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	ci, release := string(ciBytes), string(releaseBytes)

	for _, required := range []string{
		"native_platform:",
		"inputs.native_platform == 'windows'",
		"inputs.native_platform == 'macos'",
		"inputs.native_platform == 'all'",
	} {
		if !strings.Contains(ci, required) {
			t.Errorf("ci.yml missing staged native certification contract %q", required)
		}
	}

	for _, required := range []string{
		"v1.0.0-beta.3|v1.0.0-beta.*|v1.0.0-rc.*|v1.0.0)",
		"native_policy=all",
		"NATIVE_POLICY:",
		"'hud_linux_amd64'",
		"'hud_windows_amd64'",
		"'hud_darwin_arm64'",
		`if [ "$NATIVE_POLICY" = all ]; then`,
		`--source-sha "$GITHUB_SHA"`,
		"manifest.get('source_sha')",
	} {
		if !strings.Contains(release, required) {
			t.Errorf("release.yml missing staged platform policy %q", required)
		}
	}
}

func TestWindowsInstallerAssetNameMatchesNSIS(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	ps, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	nsi, err := os.ReadFile(filepath.Join(root, "packaging/windows/nexus.nsi"))
	if err != nil {
		t.Fatal(err)
	}
	const asset = "nexus-setup-Windows_x86_64.exe"
	if !strings.Contains(string(ps), asset) {
		t.Fatalf("install.ps1 must use canonical NSIS asset %s", asset)
	}
	if !strings.Contains(string(nsi), "OutFile \""+asset+"\"") {
		t.Fatalf("NSIS OutFile must match installer asset %s", asset)
	}
}
