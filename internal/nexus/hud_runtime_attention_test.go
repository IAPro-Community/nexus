package nexus

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestHUDRuntimeAttentionOnlyHonestLiveWaitsAcrossProjects(t *testing.T) {
	now := time.Now()
	wait := func(id, project, prompt string) registry.RuntimeSession {
		return registry.RuntimeSession{
			RuntimeID: id, ProjectID: project, ProviderID: "codex",
			Transport: "mock", State: registry.StateWaiting,
			AttentionKind: "needs_user", AttentionReason: "QUESTION",
			AttentionContext: "private terminal question", PromptKind: prompt,
			AttentionFingerprint: id + ":v1", UpdatedAt: now,
		}
	}
	sessions := []registry.RuntimeSession{
		wait("codex-a", "project-a", "yn"),
		wait("claude-b", "project-b", "choice"),
		{RuntimeID: "shell-idle", ProjectID: "project-b", State: registry.StateWaiting, Transport: "mock"},
		{RuntimeID: "ended", ProjectID: "project-a", State: registry.StateStopped,
			Transport: "mock", AttentionKind: "needs_user", AttentionContext: "old question", PromptKind: "yn"},
		wait("no-kind", "project-c", "none"),
	}
	all := BuildHUDRuntimeAttention(sessions, "")
	if len(all) != 2 {
		t.Fatalf("expected two live explicit waits across projects, got %+v", all)
	}
	if all[0].RuntimeID != "claude-b" && all[0].RuntimeID != "codex-a" {
		t.Fatalf("unexpected source: %+v", all)
	}
	single := BuildHUDRuntimeAttention(sessions, "project-a")
	if len(single) != 1 || single[0].RuntimeID != "codex-a" {
		t.Fatalf("project scoped attention leaked: %+v", single)
	}
	if single[0].Fingerprint != "codex-a:v1" || single[0].PromptKind != "yn" {
		t.Fatalf("canonical wait metadata changed: %+v", single[0])
	}
}

func TestHUDRuntimeAttentionNeverLeaksProviderOutput(t *testing.T) {
	now := time.Now()
	results := BuildHUDRuntimeAttention([]registry.RuntimeSession{{
		RuntimeID: "r1", Transport: "mock", State: registry.StateApproval,
		AttentionReason: "APPROVAL", AttentionContext: "SECRET_API_KEY",
		PromptKind: "yn", ProviderID: "claude", UpdatedAt: now,
	}}, "")
	if len(results) != 1 {
		t.Fatalf("structured approval should be visible: %+v", results)
	}
	item := results[0]
	if item.Fingerprint == "SECRET_API_KEY" || item.Reason == "SECRET_API_KEY" ||
		item.ProjectName == "SECRET_API_KEY" {
		t.Fatal("HUD must exclude sensitive terminal prompt text")
	}
}

func TestHUDRuntimeAttentionIgnoresDeadHost(t *testing.T) {
	got := BuildHUDRuntimeAttention([]registry.RuntimeSession{{
		RuntimeID: "dead", ProviderID: "codex", State: registry.StateWaiting,
		AttentionKind: "needs_user", AttentionContext: "Continue?", PromptKind: "yn",
		HostPID: 0, Transport: "",
	}}, "")
	if len(got) != 0 {
		t.Fatalf("dead runtime must not keep attention active: %+v", got)
	}
}
