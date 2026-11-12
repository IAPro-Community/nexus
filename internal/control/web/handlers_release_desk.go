package web

import (
	"context"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/executil"
	"github.com/IAPro-Community/nexus/internal/nexus"
)

// releaseRefPattern accepts branch names and SHAs only. It blocks shell
// metacharacters because the value is passed to git as an argument.
var releaseRefPattern = regexp.MustCompile(`^[A-Za-z0-9._/\-]{1,120}$`)

// maxReleaseDeskCommits caps the log projection.
const maxReleaseDeskCommits = 200

// handleProjectReleaseDesk GET /api/v1/projects/{projectID}/release-desk?base=<ref>
//
// Read-only release projection (capability release-desk): links the commit
// range between the requested base and HEAD with PRs, issues and the
// canonical release gates. The report never authorizes integration: without
// a recorded approval bound to HEAD and verified gates, merge/publish is
// blocked with explicit reasons. Nexus performs no merges itself.
func (h *NexusHandler) handleProjectReleaseDesk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	proj, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	baseRef := strings.TrimSpace(r.URL.Query().Get("base"))
	if baseRef != "" && !releaseRefPattern.MatchString(baseRef) {
		writeError(w, http.StatusBadRequest, "invalid base ref")
		return
	}

	dir := proj.CanonicalPath
	headSHA, err := gitResolveRef(r.Context(), dir, "HEAD")
	if err != nil {
		writeError(w, http.StatusConflict, "release desk requires a git repository: "+err.Error())
		return
	}
	baseSHA := headSHA
	if baseRef != "" {
		baseSHA, err = gitResolveRef(r.Context(), dir, baseRef)
		if err != nil {
			writeError(w, http.StatusConflict, "cannot resolve base ref: "+err.Error())
			return
		}
	}

	branch := getGitBranch(dir)
	commits := make([]nexus.ReleaseCommitSummary, 0, 32)
	mergeBase, err := gitMergeBase(r.Context(), dir, baseSHA, headSHA)
	if err == nil && mergeBase != headSHA {
		logRecords, logErr := gitLogRange(r.Context(), dir, mergeBase, headSHA)
		if logErr == nil {
			for _, record := range logRecords {
				if commit, ok := nexus.ParseReleaseCommitLine(record); ok {
					commits = append(commits, commit)
					if len(commits) >= maxReleaseDeskCommits {
						break
					}
				}
			}
		}
	}

	// Approval comes exclusively from the signed artifact in the repo
	// (docs/reference/release-approval.json, signed by an enrolled key from
	// the trust store). Any problem fails closed: no approval, integration
	// stays blocked with an explicit reason.
	approval, _ := nexus.LoadReleaseApproval(
		filepath.Join(dir, "docs", "reference", "release-approval.json"),
		filepath.Join(dir, "docs", "reference", "release-reviewers.json"),
		headSHA,
	)
	report := nexus.BuildReleaseDeskReport(branch, baseSHA, headSHA, commits, nexus.CanonicalReleaseGates(), approval)
	writeJSON(w, http.StatusOK, report)
}

// gitResolveRef resolves one ref to its full SHA.
func gitResolveRef(parent context.Context, dir, ref string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	output, err := executil.CommandContext(ctx, "git", "-C", dir, "rev-parse", ref).Output()
	if err != nil {
		return "", normalizeCollisionScanError(err)
	}
	sha := strings.ToLower(strings.TrimSpace(string(output)))
	if !isReleaseSHA(sha) {
		return "", collisionScanError("unexpected git output for ref")
	}
	return sha, nil
}

// gitMergeBase resolves the merge base of two SHAs.
func gitMergeBase(parent context.Context, dir, left, right string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	output, err := executil.CommandContext(ctx, "git", "-C", dir, "merge-base", left, right).Output()
	if err != nil {
		return "", normalizeCollisionScanError(err)
	}
	sha := strings.ToLower(strings.TrimSpace(string(output)))
	if !isReleaseSHA(sha) {
		return "", collisionScanError("unexpected merge-base output")
	}
	return sha, nil
}

// gitLogRange returns `%H%x00%s` records for mergeBase..head, capped.
func gitLogRange(parent context.Context, dir, from, to string) ([]string, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	output, err := executil.CommandContext(ctx, "git", "-C", dir,
		"log", from+".."+to, "--format=%H%x00%s").Output()
	if err != nil {
		return nil, normalizeCollisionScanError(err)
	}
	if len(output) > 2*1024*1024 {
		return nil, errDiffOutputTooLarge
	}
	records := strings.Split(strings.TrimRight(string(output), "\n"), "\n")
	if len(records) == 1 && records[0] == "" {
		return []string{}, nil
	}
	return records, nil
}

// isHexDigit reports whether r is an ASCII lowercase hexadecimal digit.
func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')
}

func isReleaseSHA(sha string) bool {
	if len(sha) < 7 || len(sha) > 64 {
		return false
	}
	for _, r := range sha {
		if !isHexDigit(r) {
			return false
		}
	}
	return true
}
