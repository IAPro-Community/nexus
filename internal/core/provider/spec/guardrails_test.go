package spec

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var (
	cliLookPathRE   = regexp.MustCompile(`(?:^|[^A-Za-z0-9_])(?:LookPath|exec\.Command)\(\s*"(codex|agy|claude|gemini|opencode|cursor-agent|agent)"`)
	quotaContainsRE = regexp.MustCompile(`(?i)strings\.Contains\([^)]*"(quota|429|rate[ _-]?limit(ed)?)`)
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

func TestGuardrailNoCLILookPathOutsideSpec(t *testing.T) {
	root := repoRoot(t)
	internal := filepath.Join(root, "internal")
	err := filepath.WalkDir(internal, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(relFrom(root, path))
		if strings.HasPrefix(rel, "internal/core/provider/spec/") {
			return nil
		}
		if strings.HasPrefix(rel, "internal/runtime/") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if loc := cliLookPathRE.FindIndex(data); loc != nil {
			t.Errorf("%s: LookPath of a provider CLI must go through spec.ResolveBinary: %s", rel, snippet(data, loc[0]))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGuardrailNoQuotaSubstringOutsideClassifier(t *testing.T) {
	root := repoRoot(t)
	internal := filepath.Join(root, "internal")
	err := filepath.WalkDir(internal, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(relFrom(root, path))
		if strings.HasPrefix(rel, "internal/core/classifier/") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if !quotaContainsRE.Match(data) {
			return nil
		}
		for i, line := range strings.Split(string(data), "\n") {
			if !quotaContainsRE.MatchString(line) {
				continue
			}
			if strings.Contains(line, `"rate_limits"`) || (strings.Contains(line, `"quota"`) && strings.Contains(line, "json")) {
				continue
			}
			t.Errorf("%s:%d: quota/rate-limit detection must use classifier.Classify: %s", rel, i+1, strings.TrimSpace(line))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func relFrom(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func snippet(data []byte, idx int) string {
	start := idx
	if start > 40 {
		start -= 40
	}
	end := idx + 80
	if end > len(data) {
		end = len(data)
	}
	return strings.TrimSpace(string(data[start:end]))
}
