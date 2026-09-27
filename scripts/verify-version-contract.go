//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// VERSION_CONTRACT_EXIT_PASS = 0, FAIL = 1

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	errors := validateContract(root)
	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "VERSION CONTRACT FAIL (%d violations):\n", len(errors))
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "  ✗ %s\n", e)
		}
		os.Exit(1)
	}
	fmt.Println("VERSION CONTRACT PASS: all sources consistent")
}

func validateContract(root string) []string {
	var errs []string

	version := readTrimmed(filepath.Join(root, "VERSION"))
	if version == "" {
		return append(errs, "VERSION file is empty or missing")
	}
	fmt.Printf("  VERSION = %s\n", version)

	// web/package.json
	var pkgJSON struct {
		Version      string            `json:"version"`
		Dependencies map[string]string `json:"dependencies"`
	}
	if b, err := os.ReadFile(filepath.Join(root, "web", "package.json")); err == nil {
		if err := json.Unmarshal(b, &pkgJSON); err != nil {
			errs = append(errs, fmt.Sprintf("web/package.json: parse error: %v", err))
		} else if pkgJSON.Version != version {
			errs = append(errs, fmt.Sprintf("web/package.json version=%s != VERSION=%s", pkgJSON.Version, version))
		} else {
			fmt.Printf("  web/package.json.version = %s ✓\n", pkgJSON.Version)
		}
	} else {
		errs = append(errs, fmt.Sprintf("web/package.json: %v", err))
	}

	// Wails v3 no longer uses per-app wails.json metadata. VERSION remains the
	// single source of truth and is injected into CLI/Desktop/HUD binaries by
	// build ldflags. Keep the native framework/runtime contract explicit and
	// reject Wails v2 metadata residue.
	const wailsVersion = "3.0.0-beta.25"
	if got := strings.TrimSpace(pkgJSON.Dependencies["@wailsio/runtime"]); got != wailsVersion {
		errs = append(errs, fmt.Sprintf("web @wailsio/runtime=%s != %s", got, wailsVersion))
	} else {
		fmt.Printf("  @wailsio/runtime = %s ✓\n", got)
	}

	goModPath := filepath.Join(root, "go.mod")
	if b, err := os.ReadFile(goModPath); err == nil {
		text := string(b)
		if !strings.Contains(text, "github.com/wailsapp/wails/v3 v"+wailsVersion) {
			errs = append(errs, "go.mod does not pin Wails v3 "+wailsVersion)
		}
		if strings.Contains(text, "github.com/wailsapp/wails/v2") {
			errs = append(errs, "go.mod still references Wails v2")
		}
	} else {
		errs = append(errs, fmt.Sprintf("go.mod: %v", err))
	}

	for _, rel := range []string{
		filepath.Join("cmd", "nexus-desktop", "wails.json"),
		filepath.Join("cmd", "nexus-hud", "wails.json"),
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			errs = append(errs, rel+" is Wails v2 metadata and must not exist in the Wails v3 line")
		} else if !os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("%s: %v", rel, err))
		}
	}

	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		errs = append(errs, fmt.Sprintf("Makefile: %v", err))
	} else {
		text := string(makefile)
		for _, target := range []string{"build-desktop-wails:", "build-hud-wails:"} {
			if !strings.Contains(text, target) {
				errs = append(errs, "Makefile missing "+target)
			}
		}
		if strings.Count(text, "internal/buildinfo.Version=$$VERSION") < 2 {
			errs = append(errs, "Desktop/HUD build targets must inject VERSION through buildinfo ldflags")
		}
	}

	// git tag (if on a release tag)
	if tag := gitTag(); tag != "" {
		expected := "v" + version
		if tag != expected {
			errs = append(errs, fmt.Sprintf("git tag=%s != expected %s", tag, expected))
		} else {
			fmt.Printf("  git tag = %s ✓\n", tag)
		}
	}

	// goreleaser snapshot (dry check)
	if _, err := os.Stat(filepath.Join(root, ".goreleaser.yaml")); err == nil {
		fmt.Printf("  .goreleaser.yaml exists ✓\n")
	}

	return errs
}

func readTrimmed(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func gitTag() string {
	cmd := exec.Command("git", "describe", "--tags", "--exact-match", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
