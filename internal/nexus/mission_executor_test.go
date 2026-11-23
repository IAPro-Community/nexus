package nexus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func TestBuildReviewPromptUsesValidJSONExample(t *testing.T) {
	prompt := buildReviewPrompt(&runner.PackageRun{
		Title:              "Validation task",
		Goal:               "Check the implementation",
		AcceptanceCriteria: []string{"the check passes"},
	})
	if strings.Contains(prompt, "true|false") || strings.Contains(prompt, "HIGH|MEDIUM|LOW") {
		t.Fatalf("review prompt contains non-JSON union syntax: %q", prompt)
	}
	if !strings.Contains(prompt, `{"approved":true`) {
		t.Fatalf("review prompt is missing the valid JSON example")
	}
}

func TestParseReviewVerdictAcceptsValidJSONExample(t *testing.T) {
	got, err := parseReviewVerdict(`{"approved":true,"findings":[],"structured_findings":[],"remediation_tips":[]}`)
	if err != nil {
		t.Fatalf("parseReviewVerdict returned error: %v", err)
	}
	if !got.Approved {
		t.Fatalf("review verdict was not approved")
	}
}

func TestParseReviewVerdictUsesLastValidObjectFromCodexTranscript(t *testing.T) {
	output := "codex transcript\n{" + `"approved":true,"findings":[],"structured_findings":[],"remediation_tips":[]` + "}\ntokens used\n{" + `"approved":true,"findings":[],"structured_findings":[],"remediation_tips":[]` + "}"
	got, err := parseReviewVerdict(output)
	if err != nil {
		t.Fatalf("parseReviewVerdict returned error: %v", err)
	}
	if !got.Approved {
		t.Fatalf("review verdict was not approved")
	}
}

func TestParseReviewVerdictFailsClosedWithoutValidObject(t *testing.T) {
	if _, err := parseReviewVerdict(`{"approved":true|false,"findings":[]}`); err == nil {
		t.Fatal("parseReviewVerdict accepted invalid reviewer JSON")
	}
}

func TestNexusPackageExecutorRejectsMaestroOwnedRun(t *testing.T) {
	executor := &nexusPackageExecutor{n: &Nexus{}}
	run := &runner.MissionRun{ID: "run-maestro", ExecutionOwner: runner.ExecutionOwnerMaestro}
	pkg := &runner.PackageRun{}

	tests := []struct {
		name string
		call func() error
	}{
		{name: "allocate", call: func() error {
			_, err := executor.Allocate(context.Background(), run, pkg)
			return err
		}},
		{name: "compile", call: func() error {
			_, err := executor.Compile(context.Background(), run, pkg)
			return err
		}},
		{name: "execute", call: func() error {
			_, err := executor.Execute(context.Background(), run, pkg, "prompt")
			return err
		}},
		{name: "review", call: func() error {
			_, err := executor.Review(context.Background(), run, pkg)
			return err
		}},
		{name: "integrate delivery", call: func() error {
			_, err := executor.IntegrateDelivery(context.Background(), run)
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); !errors.Is(err, runner.ErrExecutionOwnerNotLocal) {
				t.Fatalf("expected local owner guard, got %v", err)
			}
		})
	}
}

func TestNexusPackageExecutorRejectsUnknownExecutionOwner(t *testing.T) {
	executor := &nexusPackageExecutor{n: &Nexus{}}
	run := &runner.MissionRun{ID: "run-unknown", ExecutionOwner: "UNKNOWN"}

	_, err := executor.Allocate(context.Background(), run, &runner.PackageRun{})
	if !errors.Is(err, runner.ErrInvalidExecutionOwner) {
		t.Fatalf("expected invalid owner guard, got %v", err)
	}
}
