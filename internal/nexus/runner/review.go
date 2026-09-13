package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// NormalizeReviewFinding makes a finding identity independent of provider
// prose formatting. It deliberately keeps file/location/category in the key
// so two unrelated defects with the same description do not collapse.
func NormalizeReviewFinding(finding ReviewFinding) ReviewFinding {
	finding.Severity = strings.ToUpper(strings.TrimSpace(finding.Severity))
	finding.Category = strings.ToUpper(strings.TrimSpace(finding.Category))
	finding.File = strings.TrimSpace(finding.File)
	finding.Location = strings.TrimSpace(finding.Location)
	finding.Description = strings.TrimSpace(finding.Description)
	finding.Rationale = strings.TrimSpace(finding.Rationale)
	finding.RequiredAction = strings.TrimSpace(finding.RequiredAction)
	if finding.ID == "" {
		finding.ID = ReviewFindingIdentity(finding)
	}
	if finding.Status == "" {
		finding.Status = FindingOpen
	}
	return finding
}

// ReviewFindingIdentity returns a stable semantic identity for a finding.
func ReviewFindingIdentity(finding ReviewFinding) string {
	value := fmt.Sprintf("%s\x00%s\x00%s\x00%s", strings.ToUpper(strings.TrimSpace(finding.Category)), strings.TrimSpace(finding.File), strings.TrimSpace(finding.Location), strings.ToLower(strings.TrimSpace(finding.Description)))
	sum := sha256.Sum256([]byte(value))
	category := strings.ToUpper(strings.TrimSpace(finding.Category))
	if category == "" {
		category = "REVIEW"
	}
	category = strings.NewReplacer(" ", "-", "_", "-", "/", "-").Replace(category)
	return "FINDING-" + category + "-" + strings.ToUpper(hex.EncodeToString(sum[:]))[:10]
}

// NormalizeReviewVerdict preserves the legacy string transport while giving
// the runner a structured canonical form.
func NormalizeReviewVerdict(verdict ReviewVerdict) ReviewVerdict {
	if len(verdict.StructuredFindings) == 0 {
		for _, description := range verdict.Findings {
			verdict.StructuredFindings = append(verdict.StructuredFindings, NormalizeReviewFinding(ReviewFinding{Description: description}))
		}
	} else {
		for i := range verdict.StructuredFindings {
			verdict.StructuredFindings[i] = NormalizeReviewFinding(verdict.StructuredFindings[i])
		}
	}
	verdict.StructuredFindings = DeduplicateReviewFindings(verdict.StructuredFindings)
	return verdict
}

// DeduplicateReviewFindings keeps the first semantic finding and merges
// evidence/action fields from later duplicates deterministically.
func DeduplicateReviewFindings(findings []ReviewFinding) []ReviewFinding {
	byID := make(map[string]ReviewFinding, len(findings))
	for _, raw := range findings {
		finding := NormalizeReviewFinding(raw)
		previous, exists := byID[finding.ID]
		if !exists {
			byID[finding.ID] = finding
			continue
		}
		if previous.Rationale == "" {
			previous.Rationale = finding.Rationale
		}
		if previous.RequiredAction == "" {
			previous.RequiredAction = finding.RequiredAction
		}
		previous.Evidence = uniqueStrings(append(previous.Evidence, finding.Evidence...))
		byID[finding.ID] = previous
	}
	out := make([]ReviewFinding, 0, len(byID))
	for _, finding := range byID {
		out = append(out, finding)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// MergeReviewFindings updates the durable finding ledger for one package. A
// successful review resolves all previously open findings; a rejected review
// leaves absent findings open so a transient reviewer omission cannot erase a
// defect. Reappearing resolved findings are marked REGRESSED.
func MergeReviewFindings(history, current []ReviewFinding, approved bool, attempt int) []ReviewFinding {
	ledger := make(map[string]ReviewFinding, len(history)+len(current))
	for _, raw := range history {
		finding := NormalizeReviewFinding(raw)
		ledger[finding.ID] = finding
	}
	current = DeduplicateReviewFindings(current)
	seen := make(map[string]struct{}, len(current))
	for _, finding := range current {
		seen[finding.ID] = struct{}{}
		previous, exists := ledger[finding.ID]
		if !exists {
			finding.Status = FindingNew
			finding.FirstSeenAttempt = attempt
		} else {
			if previous.Status == FindingResolved {
				finding.Status = FindingRegressed
			} else {
				finding.Status = FindingOpen
			}
			finding.FirstSeenAttempt = previous.FirstSeenAttempt
		}
		finding.LastSeenAttempt = attempt
		ledger[finding.ID] = finding
	}
	if approved {
		for id, finding := range ledger {
			if _, present := seen[id]; !present || len(current) > 0 {
				finding.Status = FindingResolved
				finding.LastSeenAttempt = attempt
				ledger[id] = finding
			}
		}
	}
	out := make([]ReviewFinding, 0, len(ledger))
	for _, finding := range ledger {
		out = append(out, finding)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func OpenReviewFindings(findings []ReviewFinding) []ReviewFinding {
	out := make([]ReviewFinding, 0, len(findings))
	for _, finding := range findings {
		if finding.Status != FindingResolved {
			out = append(out, finding)
		}
	}
	return out
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
