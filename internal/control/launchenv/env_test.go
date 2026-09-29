package launchenv

import (
	"os"
	"strings"
	"testing"
)

func TestMergeReplacesDuplicateOverride(t *testing.T) {
	got := Merge([]string{"A=old", "PATH=/driver/bin"}, map[string]string{"A": "new"}, nil)
	joined := strings.Join(got, "\n")
	if strings.Count(joined, "A=") != 1 || !strings.Contains(joined, "A=new") {
		t.Fatalf("override was not canonicalized: %v", got)
	}
}

func TestMergePrependsGuardPathAndPreservesDriverPATH(t *testing.T) {
	sep := string(os.PathListSeparator)
	got := Merge([]string{"PATH=/driver/bin"}, nil, []string{"/guard"})
	var path string
	for _, entry := range got {
		if strings.HasPrefix(entry, "PATH=") {
			path = strings.TrimPrefix(entry, "PATH=")
		}
	}
	if path != "/guard"+sep+"/driver/bin" {
		t.Fatalf("unexpected PATH %q", path)
	}
}

func TestMergeProtectsIsolationHomes(t *testing.T) {
	got := Merge(
		[]string{"HOME=/profile/home", "CODEX_HOME=/profile/home", "PATH=/bin"},
		map[string]string{"HOME": "/host/home", "CODEX_HOME": "/host/.codex", "FOO": "bar"},
		nil,
	)
	joined := strings.Join(got, "\n")
	if strings.Contains(joined, "HOME=/host/home") || strings.Contains(joined, "CODEX_HOME=/host/.codex") {
		t.Fatalf("isolation homes were overridden: %v", got)
	}
	if !strings.Contains(joined, "HOME=/profile/home") || !strings.Contains(joined, "CODEX_HOME=/profile/home") {
		t.Fatalf("driver isolation homes missing: %v", got)
	}
	if !strings.Contains(joined, "FOO=bar") {
		t.Fatalf("non-isolation override missing: %v", got)
	}
}

func TestEnvKeyForGOOSWindowsCaseInsensitive(t *testing.T) {
	if envKeyForGOOS("Path", "windows") != "PATH" {
		t.Fatal("Windows environment key was not canonicalized")
	}
	if envKeyForGOOS("Path", "linux") != "Path" {
		t.Fatal("Unix environment key must remain case-sensitive")
	}
}
