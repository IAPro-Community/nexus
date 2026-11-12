package nexus

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Release Desk (capability release-desk).
//
// The desk is a read-only projection that links the evidence of one release:
// commit range, pull requests, issues, and the gates that must be verifiable
// before integration. Nexus itself never performs merges or publishes —
// agents and the maestro do — so the enforcement lives here as an explicit,
// testable decision: a merge/publish may only proceed with a recorded human
// approval bound to the exact head SHA and with every gate verified.
// Anything else is blocked with concrete reasons, never silently allowed.

// Canonical gate statuses.
const (
	ReleaseGateVerified   = "verified"
	ReleaseGateUnverified = "unverified"
)

// ReleaseCommitSummary is one commit in the release range.
type ReleaseCommitSummary struct {
	SHA     string
	Subject string
	// References are the #N tokens found in the subject.
	References []string
}

// ReleaseGate is one verifiable check of the release.
type ReleaseGate struct {
	Name          string
	VerifyCommand string
	Status        string
	// Evidence points at the artifact that proves the verified status.
	Evidence string
}

// ReleaseApproval records the explicit human authorization for one exact SHA.
type ReleaseApproval struct {
	Approver   string
	ApprovalID string
	SHA        string
	ApprovedAt time.Time
}

// ReleaseDeskReport is the full read-only projection for one release range.
type ReleaseDeskReport struct {
	Branch  string
	BaseSHA string
	HeadSHA string
	Commits []ReleaseCommitSummary
	// PullRequests come from "(#N)" trailer patterns; Issues from bare #N refs.
	PullRequests []string
	Issues       []string
	Gates        []ReleaseGate
	Approval     *ReleaseApproval
	// MergePublishAllowed is false unless approval + verified gates + SHA
	// binding all hold. Reasons enumerate every blocker.
	MergePublishAllowed bool
	BlockReasons        []string
}

var refTokenPattern = regexp.MustCompile(`#(\d+)`)
var pullRequestPattern = regexp.MustCompile(`\(#(\d+)\)\s*$`)

// ParseCommitReferences extracts the #N tokens of a commit subject, unique
// and in order of first appearance.
func ParseCommitReferences(subject string) []string {
	matches := refTokenPattern.FindAllStringSubmatch(subject, -1)
	refs := make([]string, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, match := range matches {
		ref := "#" + match[1]
		if seen[ref] {
			continue
		}
		seen[ref] = true
		refs = append(refs, ref)
	}
	return refs
}

// BuildReleaseDeskReport links the release range: commits, PRs, issues and
// gates. PRs are detected from the conventional `(#N)` subject suffix; the
// remaining refs count as issues. Output is deterministic (sorted).
func BuildReleaseDeskReport(
	branch, baseSHA, headSHA string,
	commits []ReleaseCommitSummary,
	gates []ReleaseGate,
	approval *ReleaseApproval,
) *ReleaseDeskReport {
	report := &ReleaseDeskReport{
		Branch:   branch,
		BaseSHA:  baseSHA,
		HeadSHA:  headSHA,
		Commits:  make([]ReleaseCommitSummary, 0, len(commits)),
		Gates:    make([]ReleaseGate, 0, len(gates)),
		Approval: approval,
	}
	prSeen := make(map[string]bool)
	issueSeen := make(map[string]bool)
	for _, commit := range commits {
		entry := ReleaseCommitSummary{
			SHA:        commit.SHA,
			Subject:    commit.Subject,
			References: ParseCommitReferences(commit.Subject),
		}
		if pr := pullRequestPattern.FindStringSubmatch(commit.Subject); pr != nil {
			ref := "#" + pr[1]
			if !prSeen[ref] {
				prSeen[ref] = true
				report.PullRequests = append(report.PullRequests, ref)
			}
		}
		for _, ref := range entry.References {
			if !prSeen[ref] && !issueSeen[ref] {
				issueSeen[ref] = true
				report.Issues = append(report.Issues, ref)
			}
		}
		report.Commits = append(report.Commits, entry)
	}
	sort.Strings(report.PullRequests)
	sort.Strings(report.Issues)
	report.Gates = append(report.Gates, gates...)
	report.MergePublishAllowed, report.BlockReasons = evaluateMergePublish(report)
	return report
}

// evaluateMergePublish applies the hard rule: no integration without a
// recorded approval bound to the exact head SHA and every gate verified.
func evaluateMergePublish(report *ReleaseDeskReport) (bool, []string) {
	reasons := make([]string, 0)
	if report.BaseSHA == "" || report.HeadSHA == "" {
		reasons = append(reasons, "release range is not resolved (missing base or head SHA)")
	}
	if report.BaseSHA != "" && report.HeadSHA != "" && report.BaseSHA == report.HeadSHA {
		reasons = append(reasons, "release range is empty (base equals head); nothing to integrate")
	}
	if report.Approval == nil {
		reasons = append(reasons, "no recorded approval for this release")
	} else {
		if strings.TrimSpace(report.Approval.Approver) == "" || strings.TrimSpace(report.Approval.ApprovalID) == "" {
			reasons = append(reasons, "approval record is incomplete (approver and id required)")
		}
		if report.HeadSHA != "" && report.Approval.SHA != report.HeadSHA {
			reasons = append(reasons, fmt.Sprintf(
				"approval is bound to %s but release head is %s; re-approval required",
				shortSHA(report.Approval.SHA), shortSHA(report.HeadSHA),
			))
		}
	}
	unverified := make([]string, 0)
	for _, gate := range report.Gates {
		if gate.Status != ReleaseGateVerified {
			unverified = append(unverified, gate.Name)
		}
	}
	if len(unverified) > 0 {
		reasons = append(reasons, "unverified gates: "+strings.Join(unverified, ", "))
	}
	return len(reasons) == 0, reasons
}

// ApproveRelease binds a human approval to one exact SHA. Use the returned
// record in the report; nothing is persisted here — persistence belongs to
// the signed evidence path.
func ApproveRelease(approver, approvalID, sha string, at time.Time) *ReleaseApproval {
	return &ReleaseApproval{
		Approver:   strings.TrimSpace(approver),
		ApprovalID: strings.TrimSpace(approvalID),
		SHA:        strings.ToLower(strings.TrimSpace(sha)),
		ApprovedAt: at,
	}
}

// VerifyReleaseGate returns a verified copy of the gate with evidence bound.
// It refuses to fabricate verification: callers pass the evidence reference
// only when it really exists.
func VerifyReleaseGate(gate ReleaseGate, evidence string) ReleaseGate {
	verified := gate
	verified.Status = ReleaseGateVerified
	verified.Evidence = strings.TrimSpace(evidence)
	if verified.Evidence == "" {
		// Without evidence a gate stays unverified — no silent upgrades.
		verified.Status = ReleaseGateUnverified
	}
	return verified
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	if sha == "" {
		return "(none)"
	}
	return sha
}

// CanonicalReleaseGates seeds the report with the release gates Nexus owns.
// They start unverified on purpose: the desk never inherits confidence.
func CanonicalReleaseGates() []ReleaseGate {
	return []ReleaseGate{
		{
			Name:          "agent-workspace-contract",
			VerifyCommand: "node scripts/verify-agent-workspace-v1.mjs --release",
			Status:        ReleaseGateUnverified,
		},
		{
			Name:          "evidence-map",
			VerifyCommand: "node scripts/verify-agent-workspace-evidence-map.mjs",
			Status:        ReleaseGateUnverified,
		},
		{
			Name:          "signed-offline-evidence",
			VerifyCommand: "node scripts/verify-agent-workspace-v1.mjs --release --offline",
			Status:        ReleaseGateUnverified,
		},
		{
			Name:          "cross-platform-exact-sha",
			VerifyCommand: "physical Windows/macOS/Linux runs at the exact SHA (see RELEASE_EVIDENCE_RUNBOOK.md)",
			Status:        ReleaseGateUnverified,
		},
		{
			Name:          "nex-release-readiness",
			VerifyCommand: "node scripts/verify-nex-release-readiness.mjs",
			Status:        ReleaseGateUnverified,
		},
	}
}

// ParseReleaseCommitLine parses one `%H%x00%s` log record. It returns
// zero-value and false on malformed records so callers can skip them.
func ParseReleaseCommitLine(record string) (ReleaseCommitSummary, bool) {
	parts := strings.SplitN(record, "\x00", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return ReleaseCommitSummary{}, false
	}
	sha := strings.ToLower(strings.TrimSpace(parts[0]))
	if !isHex(sha) {
		return ReleaseCommitSummary{}, false
	}
	return ReleaseCommitSummary{SHA: sha, Subject: strings.TrimSpace(parts[1])}, true
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	// A full or abbreviated SHA; accept 7..64 hex chars.
	n := len(s)
	return n >= 7 && n <= 64 && strconv.Itoa(n) != ""
}
