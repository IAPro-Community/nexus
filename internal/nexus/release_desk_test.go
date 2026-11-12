package nexus

import (
	"strings"
	"testing"
	"time"
)

func TestParseCommitReferencesUniqueInOrder(t *testing.T) {
	refs := ParseCommitReferences("feat(web): add panel (#55) and fix #12, refs #12")
	if len(refs) != 2 || refs[0] != "#55" || refs[1] != "#12" {
		t.Fatalf("refs = %v, want [#55 #12]", refs)
	}
	if refs := ParseCommitReferences("no refs here"); len(refs) != 0 {
		t.Fatalf("unexpected refs %v", refs)
	}
}

func TestBuildReleaseDeskReportSeparatesPullRequestsAndIssues(t *testing.T) {
	commits := []ReleaseCommitSummary{
		{SHA: "aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111", Subject: "feat: one (#55)"},
		{SHA: "bbbb2222bbbb2222bbbb2222bbbb2222bbbb2222", Subject: "fix: two (#55)"},
		{SHA: "cccc3333cccc3333cccc3333cccc3333cccc3333", Subject: "docs: three, issue #114"},
	}
	report := BuildReleaseDeskReport("release/v1.0.0", "base0000", "cccc3333", commits, CanonicalReleaseGates(), nil)
	if len(report.PullRequests) != 1 || report.PullRequests[0] != "#55" {
		t.Fatalf("pull requests = %v, want [#55]", report.PullRequests)
	}
	// #55 appears only as a PR; #114 is a plain issue.
	if len(report.Issues) != 1 || report.Issues[0] != "#114" {
		t.Fatalf("issues = %v, want [#114]", report.Issues)
	}
	if len(report.Commits) != 3 {
		t.Fatalf("commits = %d, want 3", len(report.Commits))
	}
	if report.MergePublishAllowed || len(report.BlockReasons) == 0 {
		t.Fatalf("unapproved release must be blocked: %+v", report)
	}
}

func TestMergePublishBlockedWithoutApprovalAndGates(t *testing.T) {
	report := BuildReleaseDeskReport("b", "base0000", "head1111", nil, CanonicalReleaseGates(), nil)
	found := make(map[string]bool)
	for _, reason := range report.BlockReasons {
		found[reason] = true
	}
	if !found["no recorded approval for this release"] {
		t.Fatalf("missing approval reason: %v", report.BlockReasons)
	}
	unverified := false
	for _, reason := range report.BlockReasons {
		if len(reason) > 16 && reason[:16] == "unverified gates" {
			unverified = true
		}
	}
	if !unverified {
		t.Fatalf("unverified gates must be a reason: %v", report.BlockReasons)
	}
}

func TestMergePublishBlockedWhenApprovalShaMismatch(t *testing.T) {
	approval := ApproveRelease("maestro", "GO-1", "dead00000000", time.Now())
	report := BuildReleaseDeskReport("b", "base0000", "head1111", nil, CanonicalReleaseGates(), approval)
	if report.MergePublishAllowed {
		t.Fatal("approval bound to another SHA must not authorize this release")
	}
	mismatch := false
	for _, reason := range report.BlockReasons {
		if strings.Contains(reason, "approval is bound to") && strings.Contains(reason, "re-approval required") {
			mismatch = true
		}
	}
	if !mismatch {
		t.Fatalf("SHA mismatch must be explicit: %v", report.BlockReasons)
	}
}

func TestMergePublishAllowedOnlyWithApprovalAndVerifiedGates(t *testing.T) {
	gates := CanonicalReleaseGates()
	for i := range gates {
		gates[i] = VerifyReleaseGate(gates[i], "evidence/"+gates[i].Name+".json")
	}
	approval := ApproveRelease("maestro", "GO-1", "head1111", time.Now())
	report := BuildReleaseDeskReport("b", "base0000", "head1111", nil, gates, approval)
	if !report.MergePublishAllowed || len(report.BlockReasons) != 0 {
		t.Fatalf("verified gates + matching approval must allow: %+v", report)
	}
	if report.Approval.SHA != "head1111" || report.Approval.Approver != "maestro" {
		t.Fatalf("approval binding lost: %+v", report.Approval)
	}
}

func TestVerifyReleaseGateNeverSilentlyUpgrades(t *testing.T) {
	gate := CanonicalReleaseGates()[0]
	verified := VerifyReleaseGate(gate, "evidence/contract.json")
	if verified.Status != ReleaseGateVerified || verified.Evidence != "evidence/contract.json" {
		t.Fatalf("gate with evidence must verify: %+v", verified)
	}
	still := VerifyReleaseGate(gate, "   ")
	if still.Status != ReleaseGateUnverified || still.Evidence != "" {
		t.Fatalf("empty evidence must keep gate unverified: %+v", still)
	}
}

func TestMergePublishBlockedOnEmptyRangeAndIncompleteApproval(t *testing.T) {
	gates := CanonicalReleaseGates()
	for i := range gates {
		gates[i] = VerifyReleaseGate(gates[i], "ev")
	}
	approval := ApproveRelease("maestro", "", "head1111", time.Now())
	report := BuildReleaseDeskReport("b", "head1111", "head1111", nil, gates, approval)
	if report.MergePublishAllowed {
		t.Fatal("empty range or incomplete approval must block")
	}
	if len(report.BlockReasons) < 2 {
		t.Fatalf("both blockers must be listed: %v", report.BlockReasons)
	}
}

func TestParseReleaseCommitLine(t *testing.T) {
	commit, ok := ParseReleaseCommitLine("aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111\x00feat: x (#55)")
	if !ok || commit.SHA != "aaaa1111aaaa1111aaaa1111aaaa1111aaaa1111" || commit.Subject != "feat: x (#55)" {
		t.Fatalf("valid record rejected: %+v ok=%v", commit, ok)
	}
	if _, ok := ParseReleaseCommitLine("no-shape"); ok {
		t.Fatal("malformed record must be skipped")
	}
	if _, ok := ParseReleaseCommitLine("zzzz1111zzzz\x00subject"); ok {
		t.Fatal("non-hex sha must be skipped")
	}
}
