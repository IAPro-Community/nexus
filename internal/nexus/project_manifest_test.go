package nexus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectManifestUsesStrictYAML(t *testing.T) {
	dir := t.TempDir()
	body := `version: 1
project:
  name: nexus
workspace:
  strategy: worktree
  baseRef: HEAD
  sharedDirectories:
    - web/node_modules
  copiedFiles:
    - .env.local
setup:
  commands:
    - go mod download
validation:
  commands:
    - go test ./...
execution:
  defaultTarget: local
routing:
  strategy: quality-first
`
	if err := os.WriteFile(filepath.Join(dir, "nexus.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, path, err := LoadProjectManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "nexus.yaml" || manifest.Version != 1 {
		t.Fatalf("unexpected manifest path/version: %q %+v", path, manifest)
	}
	if len(manifest.Workspace.SharedDirectories) != 1 || manifest.Workspace.SharedDirectories[0] != "web/node_modules" {
		t.Fatalf("shared directories not decoded: %+v", manifest.Workspace.SharedDirectories)
	}
}

func TestLoadProjectManifestRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nexus.yaml"), []byte("version: 1\nunknownThing: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := LoadProjectManifest(dir)
	if err == nil || !strings.Contains(err.Error(), "field") {
		t.Fatalf("unknown manifest fields must fail closed, got %v", err)
	}
}

func TestLoadProjectManifestRejectsEscapingPaths(t *testing.T) {
	dir := t.TempDir()
	body := "version: 1\nworkspace:\n  copiedFiles:\n    - ../secret\n"
	if err := os.WriteFile(filepath.Join(dir, "nexus.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := LoadProjectManifest(dir)
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("escaping manifest path must fail closed, got %v", err)
	}
}

func TestLoadProjectManifestMissingIsNotAnErrorShape(t *testing.T) {
	_, _, err := LoadProjectManifest(t.TempDir())
	if !os.IsNotExist(err) {
		t.Fatalf("missing manifest must preserve os.ErrNotExist, got %v", err)
	}
}

func TestLoadProjectManifestRejectsGitMetadata(t *testing.T) {
	dir := t.TempDir()
	body := "version: 1\nworkspace:\n  sharedDirectories:\n    - .git/objects\n"
	if err := os.WriteFile(filepath.Join(dir, "nexus.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := LoadProjectManifest(dir)
	if err == nil || !strings.Contains(err.Error(), "Git metadata") {
		t.Fatalf("Git metadata must not be shareable through project manifest, got %v", err)
	}
}

func TestLoadProjectManifestQuickCommandsAreSingleLineAndUnique(t *testing.T) {
	dir := t.TempDir()
	body := "version: 1\ncommands:\n  - id: quality\n    label: Quality\n    command: make quality\n"
	if err := os.WriteFile(filepath.Join(dir, "nexus.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, _, err := LoadProjectManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Commands) != 1 || manifest.Commands[0].ID != "quality" {
		t.Fatalf("quick command not decoded: %+v", manifest.Commands)
	}

	bad := "version: 1\ncommands:\n  - id: hidden\n    label: Hidden\n    command: |\n      echo one\n      echo two\n"
	if err := os.WriteFile(filepath.Join(dir, "nexus.yaml"), []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err = LoadProjectManifest(dir)
	if err == nil || !strings.Contains(err.Error(), "single terminal line") {
		t.Fatalf("multi-line quick command must fail closed, got %v", err)
	}

	dup := "version: 1\ncommands:\n  - id: test\n    label: One\n    command: go test ./...\n  - id: test\n    label: Two\n    command: make test\n"
	if err := os.WriteFile(filepath.Join(dir, "nexus.yaml"), []byte(dup), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err = LoadProjectManifest(dir)
	if err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("duplicate quick command ids must fail closed, got %v", err)
	}
}


func TestLoadProjectManifestExecutionTargetsAreStrictAndNormalized(t *testing.T) {
	dir := t.TempDir()
	body := `version: 1
execution:
  defaultTarget: " devbox "
  targets:
    - id: " devbox "
      kind: " SSH "
      location: " devbox "
      machineId: " machine-devbox "
      workspace: " /srv/project "
      commandTemplate: " ssh -tt devbox {provider} {args} "
      probeTemplate: " ssh devbox {provider} {args} "
`
	if err := os.WriteFile(filepath.Join(dir, "nexus.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, _, err := LoadProjectManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Execution.DefaultTarget != "devbox" {
		t.Fatalf("default target was not normalized: %q", manifest.Execution.DefaultTarget)
	}
	target, ok := manifest.ProjectExecutionTarget("devbox")
	if !ok {
		t.Fatal("normalized SSH target not found")
	}
	if target.Kind != "ssh" || target.Location != "devbox" || target.Workspace != "/srv/project" {
		t.Fatalf("unexpected normalized target: %+v", target)
	}
}

func TestLoadProjectManifestRejectsUnknownDefaultExecutionTarget(t *testing.T) {
	dir := t.TempDir()
	body := "version: 1\nexecution:\n  defaultTarget: missing\n"
	if err := os.WriteFile(filepath.Join(dir, "nexus.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := LoadProjectManifest(dir)
	if err == nil || !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("unknown default target must fail closed, got %v", err)
	}
}

func TestLoadProjectManifestRejectsUnsafeSSHExecutionTarget(t *testing.T) {
	dir := t.TempDir()
	cases := []string{
		`version: 1
execution:
  targets:
    - id: dockerish
      kind: custom
      location: devbox
      workspace: /srv/project
      commandTemplate: ssh devbox {provider} {args}
`,
		`version: 1
execution:
  targets:
    - id: shell
      kind: ssh
      location: devbox
      workspace: /srv/project
      commandTemplate: "ssh devbox sh -lc 'cd /srv/project && {provider} {args}'"
`,
		`version: 1
execution:
  targets:
    - id: wrong-bin
      kind: ssh
      location: devbox
      workspace: /srv/project
      commandTemplate: docker exec box {provider} {args}
`,
	}
	for index, body := range cases {
		if err := os.WriteFile(filepath.Join(dir, "nexus.yaml"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := LoadProjectManifest(dir); err == nil {
			t.Fatalf("unsafe SSH target case %d was accepted", index)
		}
	}
}
