package nexus

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const (
	// composerContextFileExcerptBudget bounds a single referenced file excerpt.
	composerContextFileExcerptBudget = 4 << 10 // 4KB
	// composerContextTotalBudget bounds the sum of all referenced excerpts.
	composerContextTotalBudget = 24 << 10 // 24KB
)

// AttachComposerContextFile resolves relPath under the project's canonical path,
// reads and redacts a bounded excerpt, hashes the content, and persists the
// reference on the living brief. Paths escaping the project root are rejected.
func (n *Nexus) AttachComposerContextFile(ctx context.Context, sessionID, relPath string, expectedRevision int) (*ComposerSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	if session.State == store.ComposerFinalized {
		return nil, fmt.Errorf("composer session is finalized; create a new session to keep refining the prompt")
	}
	project, err := st.GetProject(session.ProjectID)
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(project.CanonicalPath)
	if root == "" {
		return nil, fmt.Errorf("project has no canonical path; cannot attach context files")
	}
	cleanRel, absPath, err := resolveComposerContextPath(root, relPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(absPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read referenced file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("referenced path is a directory, not a file: %s", cleanRel)
	}
	raw, err := readComposerContextBytes(absPath)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])[:16]
	excerpt := security.Redact(truncateContextExcerpt(string(raw), composerContextFileExcerptBudget))

	brief := decodeComposerBrief(session.BriefJSON)
	ref := ContextFileRef{Path: cleanRel, Excerpt: excerpt, Hash: hash}
	next := make([]ContextFileRef, 0, len(brief.Context.ReferencedFiles)+1)
	for _, existing := range brief.Context.ReferencedFiles {
		if existing.Path == cleanRel {
			continue
		}
		next = append(next, existing)
	}
	next = append(next, ref)
	if total := totalContextExcerptBytes(next); total > composerContextTotalBudget {
		return nil, fmt.Errorf("referenced files exceed the %d byte context budget", composerContextTotalBudget)
	}
	brief.Context.ReferencedFiles = next
	refreshComposerBrief(&brief)
	return n.persistComposerSessionExpected(st, session, brief, expectedRevision)
}

// DetachComposerContextFile removes a previously attached file reference.
func (n *Nexus) DetachComposerContextFile(ctx context.Context, sessionID, relPath string, expectedRevision int) (*ComposerSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	if session.State == store.ComposerFinalized {
		return nil, fmt.Errorf("composer session is finalized; create a new session to keep refining the prompt")
	}
	project, err := st.GetProject(session.ProjectID)
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(project.CanonicalPath)
	target := strings.TrimSpace(relPath)
	if root != "" {
		if cleanRel, _, resolveErr := resolveComposerContextPath(root, relPath); resolveErr == nil {
			target = cleanRel
		}
	}
	brief := decodeComposerBrief(session.BriefJSON)
	next := make([]ContextFileRef, 0, len(brief.Context.ReferencedFiles))
	removed := false
	for _, existing := range brief.Context.ReferencedFiles {
		if existing.Path == target {
			removed = true
			continue
		}
		next = append(next, existing)
	}
	if !removed {
		return nil, fmt.Errorf("referenced file %q not attached to session %q", target, sessionID)
	}
	brief.Context.ReferencedFiles = next
	refreshComposerBrief(&brief)
	return n.persistComposerSessionExpected(st, session, brief, expectedRevision)
}

// resolveComposerContextPath validates relPath against root, rejecting absolute
// escapes and ".." traversal. It returns the project-relative slash path plus
// the absolute filesystem path.
func resolveComposerContextPath(root, relPath string) (string, string, error) {
	rel := strings.TrimSpace(relPath)
	if rel == "" {
		return "", "", fmt.Errorf("referenced file path is required")
	}
	rel = filepath.FromSlash(rel)
	if filepath.IsAbs(rel) {
		// Allow passing an absolute path that already lives inside the project.
		absCandidate := filepath.Clean(rel)
		relToRoot, err := filepath.Rel(root, absCandidate)
		if err != nil {
			return "", "", fmt.Errorf("invalid referenced file path")
		}
		rel = relToRoot
	}
	cleaned := filepath.Clean(rel)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || cleaned == "." {
		return "", "", fmt.Errorf("referenced file path escapes the project root")
	}
	absPath := filepath.Join(root, cleaned)
	rootWithSep := strings.TrimSuffix(filepath.Clean(root), string(filepath.Separator)) + string(filepath.Separator)
	if !strings.HasPrefix(absPath, rootWithSep) {
		return "", "", fmt.Errorf("referenced file path escapes the project root")
	}
	resolvedRoot := filepath.Clean(root)
	if realRoot, err := filepath.EvalSymlinks(root); err == nil {
		resolvedRoot = realRoot
	}
	resolvedPath := absPath
	if realPath, err := filepath.EvalSymlinks(absPath); err == nil {
		resolvedPath = realPath
	}
	relResolved, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || relResolved == ".." || strings.HasPrefix(relResolved, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("referenced file path escapes the project root")
	}
	return filepath.ToSlash(cleaned), resolvedPath, nil
}

func readComposerContextBytes(absPath string) ([]byte, error) {
	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read referenced file: %w", err)
	}
	defer file.Close()
	readCap := int64(composerContextFileExcerptBudget*2 + 1)
	raw, err := io.ReadAll(io.LimitReader(file, readCap))
	if err != nil {
		return nil, fmt.Errorf("cannot read referenced file: %w", err)
	}
	if bytes.IndexByte(raw, 0) >= 0 || !utf8.Valid(raw) {
		return nil, fmt.Errorf("referenced file is not valid UTF-8 text")
	}
	return raw, nil
}

func truncateContextExcerpt(content string, budget int) string {
	if len(content) <= budget {
		return content
	}
	// Back off to a valid UTF-8 boundary so the excerpt never ends mid-rune.
	cut := budget
	for cut > 0 && !utf8.RuneStart(content[cut]) {
		cut--
	}
	return content[:cut] + "\n…[truncated]"
}

func totalContextExcerptBytes(files []ContextFileRef) int {
	total := 0
	for _, file := range files {
		total += len(file.Excerpt)
	}
	return total
}
