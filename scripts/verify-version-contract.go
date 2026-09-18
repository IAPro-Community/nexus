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
		Version string `json:"version"`
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

	// wails.json
	var wailsJSON struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if b, err := os.ReadFile(filepath.Join(root, "cmd", "nexus-desktop", "wails.json")); err == nil {
		if err := json.Unmarshal(b, &wailsJSON); err != nil {
			errs = append(errs, fmt.Sprintf("wails.json: parse error: %v", err))
		} else if wailsJSON.Info.ProductVersion != version {
			errs = append(errs, fmt.Sprintf("wails.json info.productVersion=%s != VERSION=%s", wailsJSON.Info.ProductVersion, version))
		} else {
			fmt.Printf("  wails.json info.productVersion = %s ✓\n", wailsJSON.Info.ProductVersion)
		}
	} else {
		errs = append(errs, fmt.Sprintf("wails.json: %v", err))
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
