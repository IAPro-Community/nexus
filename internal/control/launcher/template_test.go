package launcher

import (
	"reflect"
	"testing"
)

func TestResolveCommandTemplateExpandsDockerWrapper(t *testing.T) {
	bin, args, err := ResolveCommandTemplate(`docker exec -it -w "{cwd}" vpn-dev-workspace-terminal-1 opencode {args}`, "/work/project", []string{"run", "describe the task"})
	if err != nil {
		t.Fatal(err)
	}
	if bin != "docker" {
		t.Fatalf("binary = %q, want docker", bin)
	}
	want := []string{"exec", "-it", "-w", "/work/project", "vpn-dev-workspace-terminal-1", "opencode", "run", "describe the task"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestResolveCommandTemplateRejectsShellOperators(t *testing.T) {
	if _, _, err := ResolveCommandTemplate("docker exec {args}; rm -rf /", "/work", nil); err == nil {
		t.Fatal("expected shell operator rejection")
	}
}


func TestResolveCommandTemplateContextExpandsProviderTargetMetadata(t *testing.T) {
	bin, args, err := ResolveCommandTemplateContext(
		`docker exec -it -w "{cwd}" vpn-dev-workspace-terminal-1 {provider} --profile {profile} --home "{profile_home}" {args}`,
		CommandTemplateContext{
			Cwd:         "/work/project",
			Args:        []string{"run", "describe the task"},
			Provider:    "opencode",
			Profile:     "default",
			ProfileHome: "/home/developer",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bin != "docker" {
		t.Fatalf("binary = %q, want docker", bin)
	}
	want := []string{
		"exec", "-it", "-w", "/work/project", "vpn-dev-workspace-terminal-1",
		"opencode", "--profile", "default", "--home", "/home/developer",
		"run", "describe the task",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestResolveCommandTemplateContextRequiresConfiguredProfileHome(t *testing.T) {
	_, _, err := ResolveCommandTemplateContext(
		`docker exec target {provider} --home "{profile_home}" {args}`,
		CommandTemplateContext{Provider: "opencode", Args: []string{"run"}},
	)
	if err == nil {
		t.Fatal("expected missing profile_home error")
	}
}

func TestResolveCommandTemplateContextRejectsArgsInsideToken(t *testing.T) {
	_, _, err := ResolveCommandTemplateContext(
		`docker exec target opencode --args={args}`,
		CommandTemplateContext{Args: []string{"run"}},
	)
	if err == nil {
		t.Fatal("expected embedded {args} to be rejected")
	}
}


func TestResolveCommandTemplateContextPreservesQuotedWindowsExecutablePath(t *testing.T) {
	bin, args, err := ResolveCommandTemplateContext(
		`"C:\Program Files\Nexus Tools\wrapper.exe" --workspace "{cwd}" {provider} {args}`,
		CommandTemplateContext{
			Cwd:      `C:\Users\dev\project`,
			Provider: "opencode",
			Args:     []string{"run", "inspect"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bin != `C:\Program Files\Nexus Tools\wrapper.exe` {
		t.Fatalf("windows executable path corrupted: %q", bin)
	}
	want := []string{"--workspace", `C:\Users\dev\project`, "opencode", "run", "inspect"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}


func TestResolveCommandTemplateContextSeparatesHostAndTargetWorkspace(t *testing.T) {
	bin, args, err := ResolveCommandTemplateContext(
		`docker exec -w "{target_cwd}" target {provider} {args}`,
		CommandTemplateContext{
			Cwd:       `C:\Users\dev\project`,
			TargetCwd: "/workspace",
			Provider:  "opencode",
			Args:      []string{"run", "inspect"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bin != "docker" {
		t.Fatalf("bin = %q", bin)
	}
	want := []string{"exec", "-w", "/workspace", "target", "opencode", "run", "inspect"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestResolveCommandTemplateContextTargetWorkspaceFallsBackToHostCwd(t *testing.T) {
	_, args, err := ResolveCommandTemplateContext(
		`docker exec -w "{target_cwd}" target {provider} {args}`,
		CommandTemplateContext{
			Cwd:      "/srv/project",
			Provider: "opencode",
			Args:     []string{"run"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) < 3 || args[2] != "/srv/project" {
		t.Fatalf("target_cwd fallback missing: %#v", args)
	}
}
