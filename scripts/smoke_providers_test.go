package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProviderSmokeHarnessFailsClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("provider smoke harness is a Bash/Make local release gate")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is unavailable")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is unavailable")
	}

	root := t.TempDir()
	fake := filepath.Join(root, "fake-nexus")
	script := `#!/usr/bin/env bash
set -eu
if [[ "$1" == "usage" ]]; then
  cat <<'JSON'
[
  {"provider_id":"codex","status":"LIVE"},
  {"provider_id":"agy","status":"LIVE"}
]
JSON
  exit 0
fi
provider="$1"
if [[ "$provider" == "codex" ]]; then
  echo "NEXUS_PROVIDER_SMOKE_OK"
  exit 0
fi
if [[ "$provider" == "agy" && "${FAKE_AGY_OK:-0}" == "1" ]]; then
  printf '\033[32mNEXUS_PROVIDER_SMOKE_OK\033[0m\n'
  exit 0
fi
echo "provider answered, but without the required marker"
exit 0
`
	if err := os.WriteFile(fake, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	run := func(agyOK bool) ([]byte, error) {
		cmd := exec.Command(bash, "smoke-providers.sh")
		cmd.Dir = "."
		env := append(os.Environ(),
			"NEXUS_BIN="+fake,
			"PYTHON_BIN="+python,
			"SMOKE_REQUIRED_PROVIDERS=codex agy",
			"SMOKE_PROVIDER_MARKER=NEXUS_PROVIDER_SMOKE_OK",
			"NEXUS_SMOKE_TMPDIR="+root,
		)
		if agyOK {
			env = append(env, "FAKE_AGY_OK=1")
		} else {
			env = append(env, "FAKE_AGY_OK=0")
		}
		cmd.Env = env
		return cmd.CombinedOutput()
	}

	output, err := run(false)
	if err == nil {
		t.Fatalf("missing provider marker must fail closed; output=%s", output)
	}
	if !strings.Contains(string(output), "provider smoke failed for: agy") {
		t.Fatalf("failure output does not identify the provider: %s", output)
	}

	output, err = run(true)
	if err != nil {
		t.Fatalf("valid provider markers must pass: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "smoke-providers PASS: codex agy") {
		t.Fatalf("success output missing final evidence line: %s", output)
	}
}
