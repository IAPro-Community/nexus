package contextsnapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// CodeIdentityState describes how confidently a repository can be bound to a
// validation or project-intelligence observation.
type CodeIdentityState string

const (
	CodeIdentityClean      CodeIdentityState = "CLEAN"
	CodeIdentityDirty      CodeIdentityState = "DIRTY"
	CodeIdentityUnborn     CodeIdentityState = "UNBORN"
	CodeIdentityNonGit     CodeIdentityState = "NON_GIT"
	CodeIdentityIncomplete CodeIdentityState = "INCOMPLETE"
)

// CodeIdentity is a stable, content-sensitive description of the source tree
// observed by a scan. Digests are deliberately kept separate so callers can
// explain whether a change was in the index, worktree, or untracked files.
type CodeIdentity struct {
	RepositoryKind   string            `json:"repository_kind"`
	HeadSHA          string            `json:"head_sha,omitempty"`
	Branch           string            `json:"branch,omitempty"`
	State            CodeIdentityState `json:"state"`
	IndexDigest      string            `json:"index_digest,omitempty"`
	WorktreeDigest   string            `json:"worktree_digest,omitempty"`
	UntrackedDigest  string            `json:"untracked_digest,omitempty"`
	SubmodulesDigest string            `json:"submodules_digest,omitempty"`
	IdentityDigest   string            `json:"identity_digest"`
	Complete         bool              `json:"complete"`
	Warnings         []string          `json:"warnings,omitempty"`
}

func gitOutput(root string, args ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	return command.Output()
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

type statusEntry struct {
	XY   string
	Path string
}

// parsePorcelainStatus parses the NUL-delimited v1 format. A rename has two
// paths; keeping the destination and source makes a rename observable even
// when the content is unchanged.
func parsePorcelainStatus(data []byte) []statusEntry {
	parts := strings.Split(string(data), "\x00")
	entries := make([]statusEntry, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if len(part) < 4 || part[2] != ' ' {
			continue
		}
		entry := statusEntry{XY: part[:2], Path: filepath.ToSlash(part[3:])}
		entries = append(entries, entry)
		if strings.Contains(entry.XY, "R") || strings.Contains(entry.XY, "C") {
			if i+1 < len(parts) && parts[i+1] != "" {
				entries = append(entries, statusEntry{XY: entry.XY, Path: filepath.ToSlash(parts[i+1])})
				i++
			}
		}
	}
	return entries
}

func hashStatusEntries(root string, entries []statusEntry, selector func(string) bool) (string, []string, error) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	h := sha256.New()
	warnings := []string{}
	for _, entry := range entries {
		if !selector(entry.XY) {
			continue
		}
		_, _ = io.WriteString(h, entry.XY+"\x00"+entry.Path+"\x00")
		path := filepath.Join(root, filepath.FromSlash(entry.Path))
		info, err := os.Lstat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				_, _ = io.WriteString(h, "missing\x00")
				continue
			}
			return "", nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, readErr := os.Readlink(path)
			if readErr != nil {
				return "", nil, readErr
			}
			_, _ = io.WriteString(h, "symlink\x00"+link+"\x00")
			continue
		}
		if info.IsDir() {
			_, _ = io.WriteString(h, "directory\x00")
			continue
		}
		fileDigest, readErr := digestFile(path)
		if readErr != nil {
			warnings = append(warnings, fmt.Sprintf("read %s: %v", entry.Path, readErr))
			_, _ = io.WriteString(h, "unreadable\x00")
			continue
		}
		_, _ = io.WriteString(h, fileDigest+"\x00")
	}
	return hex.EncodeToString(h.Sum(nil)), warnings, nil
}

// InspectCodeIdentity captures repository state without invoking shell
// interpolation. A changing/unreadable working tree is marked incomplete so
// consumers cannot promote an observation to a verified claim.
func InspectCodeIdentity(root string) (CodeIdentity, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." {
		return CodeIdentity{}, fmt.Errorf("project root is required")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return CodeIdentity{}, fmt.Errorf("project root unavailable")
	}
	inside, err := gitOutput(root, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(string(inside)) != "true" {
		digest, digestErr := treeDigest(root)
		if digestErr != nil {
			return CodeIdentity{}, digestErr
		}
		identity := CodeIdentity{RepositoryKind: "filesystem", State: CodeIdentityNonGit, Complete: true}
		identity.IdentityDigest = digestBytes([]byte(identity.RepositoryKind + "\x00" + digest))
		return identity, nil
	}
	identity := CodeIdentity{RepositoryKind: "git", State: CodeIdentityClean, Complete: true}
	if head, headErr := gitOutput(root, "rev-parse", "HEAD"); headErr == nil {
		identity.HeadSHA = strings.TrimSpace(string(head))
	} else {
		identity.State = CodeIdentityUnborn
	}
	if branch, branchErr := gitOutput(root, "branch", "--show-current"); branchErr == nil {
		identity.Branch = strings.TrimSpace(string(branch))
	}
	status, statusErr := gitOutput(root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if statusErr != nil {
		identity.State = CodeIdentityIncomplete
		identity.Complete = false
		identity.Warnings = append(identity.Warnings, "git status could not be read")
	} else {
		entries := parsePorcelainStatus(status)
		var warnings []string
		if identity.State != CodeIdentityUnborn {
			identity.State = CodeIdentityClean
		}
		if len(entries) > 0 {
			identity.State = CodeIdentityDirty
		}
		identity.IndexDigest, warnings, err = hashStatusEntries(root, entries, func(xy string) bool { return xy != "??" && len(xy) > 0 && xy[0] != ' ' })
		if err != nil {
			return CodeIdentity{}, err
		}
		identity.Warnings = append(identity.Warnings, warnings...)
		identity.WorktreeDigest, warnings, err = hashStatusEntries(root, entries, func(xy string) bool { return xy != "??" && len(xy) > 1 && xy[1] != ' ' })
		if err != nil {
			return CodeIdentity{}, err
		}
		identity.Warnings = append(identity.Warnings, warnings...)
		identity.UntrackedDigest, warnings, err = hashStatusEntries(root, entries, func(xy string) bool { return xy == "??" })
		if err != nil {
			return CodeIdentity{}, err
		}
		identity.Warnings = append(identity.Warnings, warnings...)
	}
	if submodules, subErr := gitOutput(root, "submodule", "status", "--recursive"); subErr == nil {
		identity.SubmodulesDigest = digestBytes(submodules)
	} else {
		identity.Warnings = append(identity.Warnings, "git submodule status could not be read")
	}
	identity.IdentityDigest = digestBytes([]byte(strings.Join([]string{
		identity.RepositoryKind, identity.HeadSHA, identity.Branch, string(identity.State),
		identity.IndexDigest, identity.WorktreeDigest, identity.UntrackedDigest, identity.SubmodulesDigest,
	}, "\x00")))
	return identity, nil
}

func treeDigest(root string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		_, _ = io.WriteString(h, filepath.ToSlash(rel)+"\x00")
		if !entry.IsDir() {
			if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
				link, err := os.Readlink(path)
				if err != nil {
					return err
				}
				_, _ = io.WriteString(h, "symlink\x00"+link+"\x00")
			} else {
				digest, err := digestFile(path)
				if err != nil {
					return err
				}
				_, _ = io.WriteString(h, digest+"\x00")
			}
		}
		return nil
	})
	return hex.EncodeToString(h.Sum(nil)), err
}
