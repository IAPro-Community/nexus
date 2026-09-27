package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func verificationSuccessCommand() string { return "exit 0" }

func verificationFailureCommand() string { return "exit 1" }

func verificationFileExistsCommand(name string) string {
	name = filepath.Clean(name)
	if runtime.GOOS == "windows" {
		return `if exist ` + name + ` (exit /b 0) else (exit /b 1)`
	}
	return `test -f "` + name + `"`
}

func TestVerificationFileExistsCommand(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "present.marker"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	engine := NewVerificationEngine()
	for _, tc := range []struct {
		name string
		want bool
	}{
		{name: "present.marker", want: true},
		{name: "missing.marker", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			results := engine.RunVerification(context.Background(), workspace, []string{verificationFileExistsCommand(tc.name)})
			if len(results) != 1 || results[0].Passed != tc.want {
				t.Fatalf("verification results = %+v, want passed=%t", results, tc.want)
			}
		})
	}
}
