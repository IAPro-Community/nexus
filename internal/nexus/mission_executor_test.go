package nexus

import (
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
