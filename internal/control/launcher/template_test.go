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
