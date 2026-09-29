package provider_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
)

var (
	substringQuotaRE = regexp.MustCompile(`(?i)strings\.Contains\([^)]*"(quota|429|rate[ _-]?limit(ed)?)`)
	lookPathCLIRE    = regexp.MustCompile(`(?:^|[^A-Za-z0-9_])(?:LookPath|exec\.Command)\(\s*"(codex|agy|claude|gemini|opencode|cursor-agent|agent)"`)
)

// Packages allowed to match failure text for classification.
var substringAllowDirs = []string{
	"internal/core/classifier/",
}

// Packages allowed to resolve provider binaries by name.
var lookPathAllowDirs = []string{
	"internal/core/provider/spec/",
	"internal/runtime/",
}

func TestNoSubstringQuotaFailoverOutsideClassifier(t *testing.T) {
	root := repoRoot(t)
	err := filepath.Walk(filepath.Join(root, "internal"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if allowedPrefix(rel, substringAllowDirs) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if substringQuotaRE.Match(data) {
			// JSON field filters like `"rate_limits"` are parser contracts, not failover gates.
			lines := strings.Split(string(data), "\n")
			for i, line := range lines {
				if !substringQuotaRE.MatchString(line) {
					continue
				}
				if strings.Contains(line, `"rate_limits"`) || strings.Contains(line, `"quota"`) && strings.Contains(line, "json") {
					continue
				}
				t.Errorf("%s:%d: strings.Contains quota/429/rate-limit must live in classifier only: %s", rel, i+1, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestProviderBinariesResolvedViaSpec(t *testing.T) {
	root := repoRoot(t)
	err := filepath.Walk(filepath.Join(root, "internal"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if allowedPrefix(rel, lookPathAllowDirs) {
			return nil
		}
		// Drivers/adapters may still call ResolveBinary which uses LookPath internally —
		// flag only string-literal LookPath/Command of CLI names.
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if lookPathCLIRE.Match(data) {
			t.Errorf("%s: resolve provider binaries via spec.MustGet(...).ResolveBinary()", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func allowedPrefix(rel string, prefixes []string) bool {
	rel = filepath.ToSlash(rel)
	for _, prefix := range prefixes {
		if strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	return false
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}


func TestKnownProvidersKeepExplicitQuotaContracts(t *testing.T) {
	expected := map[string]spec.QuotaSource{
		"codex":    spec.QuotaLiveRPC,
		"agy":      spec.QuotaLiveCLI,
		"claude":   spec.QuotaLiveHTTP,
		"gemini":   spec.QuotaLiveHTTP,
		"opencode": spec.QuotaLiveHTTP,
		"cursor":   spec.QuotaUnsupported,
	}
	for providerID, want := range expected {
		providerSpec, ok := spec.Get(providerID)
		if !ok {
			t.Fatalf("known provider %s disappeared from ProviderSpec registry", providerID)
		}
		if providerSpec.Quota != want {
			t.Fatalf("%s quota source=%s want %s", providerID, providerSpec.Quota, want)
		}
	}
}
