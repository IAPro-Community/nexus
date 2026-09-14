package contextsnapshot

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/security"
)

const (
	MaxExcerptBytes = 6 * 1024
	MaxTotalBytes   = 24 * 1024
	MaxExcerpts     = 8
)

type Metadata struct {
	ProjectID        string `json:"project_id"`
	Branch           string `json:"branch"`
	Head             string `json:"head"`
	DirtyFingerprint string `json:"dirty_fingerprint"`
	MaestroVersion   string `json:"maestro_version"`
}

type Excerpt struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
}

type Envelope struct {
	Metadata
	Excerpts  []Excerpt `json:"excerpts"`
	Truncated bool      `json:"truncated"`
	Bytes     int       `json:"bytes"`
}

var durableCandidates = []string{
	"AGENTS.md",
	filepath.Join("DEV", "INDEX.md"),
	filepath.Join("DEV", "CONTEXT.md"),
	filepath.Join("DEV", "ARCHITECTURE.md"),
	filepath.Join("DEV", "DECISIONS.md"),
	filepath.Join("DEV", "TESTING.md"),
	filepath.Join("DEV", "WORKLOG.md"),
	"README.md",
}

func readBoundedExcerpt(path string, remaining int) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", false, err
	}
	// Read beyond the output cap so redaction happens before the final slice and
	// cannot leak a secret merely because a token crosses the output boundary.
	readCap := int64(MaxExcerptBytes*2 + 1)
	raw, err := io.ReadAll(io.LimitReader(file, readCap))
	if err != nil {
		return "", false, err
	}
	redacted := security.Redact(string(raw))
	limit := MaxExcerptBytes
	if remaining < limit {
		limit = remaining
	}
	if limit <= 0 {
		return "", info.Size() > 0, nil
	}
	truncated := info.Size() > int64(len(raw)) || len(redacted) > limit
	if len(redacted) > limit {
		redacted = redacted[:limit]
	}
	return redacted, truncated, nil
}

// Build reads only a fixed allowlist of durable project-context documents.
// It never recursively walks the repository and never includes binary, secret,
// environment or arbitrary source files. Content is redacted and size bounded.
func Build(root string, metadata Metadata) (Envelope, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." {
		return Envelope{}, fmt.Errorf("project root is required")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return Envelope{}, fmt.Errorf("project root unavailable")
	}

	envelope := Envelope{Metadata: metadata, Excerpts: []Excerpt{}}
	for _, relative := range durableCandidates {
		if len(envelope.Excerpts) >= MaxExcerpts || envelope.Bytes >= MaxTotalBytes {
			envelope.Truncated = true
			break
		}
		path := filepath.Join(root, relative)
		content, truncated, readErr := readBoundedExcerpt(path, MaxTotalBytes-envelope.Bytes)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			return Envelope{}, fmt.Errorf("read context %s: %w", relative, readErr)
		}
		if content == "" && truncated {
			envelope.Truncated = true
			break
		}
		envelope.Excerpts = append(envelope.Excerpts, Excerpt{Path: filepath.ToSlash(relative), Content: content, Truncated: truncated})
		envelope.Bytes += len(content)
		if truncated {
			envelope.Truncated = true
		}
	}
	return envelope, nil
}
