// Package artifacts owns project-scoped, provider-independent generated output discovery.
// All opened paths are confined by os.Root, including symlinks and concurrent changes.
package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxEntries = 3500
const maxDepth = 7
const maxPreviewBytes = int64(24 << 20)
const maxDigestBytes = int64(64 << 20)

type Item struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	MIME      string    `json:"mime"`
	Size      int64     `json:"size"`
	Modified  time.Time `json:"modified"`
	SHA256    string    `json:"sha256,omitempty"`
	Origin    string    `json:"origin"`
	AgentID   string    `json:"agent_id,omitempty"`
	RuntimeID string    `json:"runtime_id,omitempty"`
}
type Entry struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	IsDir    bool      `json:"is_dir"`
	Kind     string    `json:"kind,omitempty"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}
type List struct {
	Items     []Item `json:"items"`
	Total     int    `json:"total"`
	HasMore   bool   `json:"has_more"`
	Scanned   int    `json:"scanned"`
	Truncated bool   `json:"truncated"`
}

var ErrForbidden = errors.New("project path is not permitted")
var ErrUnsupported = errors.New("file type is not previewable or downloadable through Nexus Outputs")

// SafeRelative accepts only paths below the project root. Hidden files/dirs,
// symlinks outside the root, and credential-like filenames are never exposed.
func SafeRelative(raw string) (string, error) {
	if raw == "" || raw == "." {
		return ".", nil
	}
	if strings.ContainsRune(raw, 0) || strings.Contains(raw, "\\") || strings.HasPrefix(raw, "/") {
		return "", ErrForbidden
	}
	clean := path.Clean(raw)
	if clean == ".." || strings.HasPrefix(clean, "../") || clean != raw {
		return "", ErrForbidden
	}
	for _, part := range strings.Split(clean, "/") {
		name := strings.ToLower(part)
		if part == "" || strings.HasPrefix(part, ".") || name == "node_modules" || name == "vendor" || name == "dist" || name == "build" || name == "target" || name == "coverage" || name == "__pycache__" || name == "secrets" || name == "credentials" || name == "private" || name == "keys" || strings.HasSuffix(name, ".key") || strings.HasSuffix(name, ".pem") || strings.HasSuffix(name, ".p12") || strings.Contains(name, "secret") || strings.Contains(name, "token") || strings.Contains(name, "credential") || strings.Contains(name, "password") {
			return "", ErrForbidden
		}
	}
	return clean, nil
}
func Root(projectPath string) (*os.Root, error) {
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, ErrForbidden
	}
	return os.OpenRoot(abs)
}
func class(name string) (kind, mimeType string) {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png":
		return "image", "image/png"
	case ".jpg", ".jpeg":
		return "image", "image/jpeg"
	case ".webp":
		return "image", "image/webp"
	case ".gif":
		return "image", "image/gif"
	case ".svg":
		return "download", "image/svg+xml"
	case ".pdf":
		return "document", "application/pdf"
	case ".mp4":
		return "video", "video/mp4"
	case ".webm":
		return "video", "video/webm"
	case ".mp3":
		return "audio", "audio/mpeg"
	case ".wav":
		return "audio", "audio/wav"
	case ".ogg":
		return "audio", "audio/ogg"
	case ".txt", ".md", ".log", ".csv", ".tsv", ".go", ".ts", ".tsx", ".js", ".jsx", ".css", ".py", ".rs", ".java", ".sh", ".html", ".htm":
		return "text", "text/plain; charset=utf-8"
	case ".json":
		return "text", "application/json"
	default:
		return "", ""
	}
}
func inspected(root *os.Root, rel string) (*os.File, fs.FileInfo, string, string, error) {
	safe, err := SafeRelative(rel)
	if err != nil || safe == "." {
		return nil, nil, "", "", ErrForbidden
	}
	kind, expected := class(path.Base(safe))
	if kind == "" {
		return nil, nil, "", "", ErrUnsupported
	}
	f, err := root.Open(safe)
	if err != nil {
		return nil, nil, "", "", err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		return nil, nil, "", "", ErrForbidden
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	_, _ = f.Seek(0, io.SeekStart)
	detected := http.DetectContentType(head[:n])
	switch kind {
	case "image":
		if detected != expected && !(expected == "image/webp" && strings.HasPrefix(string(head[:n]), "RIFF") && n >= 12 && string(head[8:12]) == "WEBP") {
			f.Close()
			return nil, nil, "", "", ErrUnsupported
		}
	case "document":
		if !strings.HasPrefix(string(head[:n]), "%PDF-") {
			f.Close()
			return nil, nil, "", "", ErrUnsupported
		}
	case "download":
		// SVG must never be rendered in the Nexus origin.
		if !strings.Contains(strings.ToLower(string(head[:n])), "<svg") {
			f.Close()
			return nil, nil, "", "", ErrUnsupported
		}
	case "text":
		if strings.Contains(detected, "octet-stream") || strings.ContainsRune(string(head[:n]), 0) {
			f.Close()
			return nil, nil, "", "", ErrUnsupported
		}
	}
	return f, info, kind, expected, nil
}
func Describe(root *os.Root, rel string) (Item, error) {
	f, info, kind, mimeType, err := inspected(root, rel)
	if err != nil {
		return Item{}, err
	}
	defer f.Close()
	h := sha256.New()
	digest := ""
	if info.Size() <= maxDigestBytes {
		_, _ = io.Copy(h, f)
		digest = hex.EncodeToString(h.Sum(nil))
	}
	idSum := sha256.Sum256([]byte(rel))
	return Item{ID: hex.EncodeToString(idSum[:12]), Path: rel, Name: path.Base(rel), Kind: kind, MIME: mimeType, Size: info.Size(), Modified: info.ModTime(), SHA256: digest, Origin: "filesystem"}, nil
}
func Directory(root *os.Root, rel string) ([]Entry, error) {
	safe, err := SafeRelative(rel)
	if err != nil {
		return nil, err
	}
	dir, err := root.Open(safe)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	st, err := dir.Stat()
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, ErrForbidden
	}
	names, err := dir.ReadDir(501)
	if err != nil {
		return nil, err
	}
	if len(names) > 500 {
		return nil, fmt.Errorf("directory exceeds 500 visible-entry scan limit")
	}
	out := make([]Entry, 0, len(names))
	for _, d := range names {
		fileRel := d.Name()
		if safe != "." {
			fileRel = safe + "/" + fileRel
		}
		if _, err := SafeRelative(fileRel); err != nil {
			continue
		}
		info, err := d.Info()
		if err != nil || d.Type()&os.ModeSymlink != 0 {
			continue
		}
		if !info.IsDir() {
			if kind, _ := class(d.Name()); kind == "" {
				continue
			}
		}
		kind, _ := class(d.Name())
		out = append(out, Entry{Name: d.Name(), Path: fileRel, IsDir: info.IsDir(), Kind: kind, Size: info.Size(), Modified: info.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}
func Scan(root *os.Root, offset, limit int, kind string) (List, error) {
	if offset < 0 || offset > maxEntries || limit < 1 || limit > 100 {
		return List{}, ErrForbidden
	}
	result := List{Items: []Item{}}
	var paths []string
	var walk func(string, int) error
	walk = func(rel string, depth int) error {
		if depth > maxDepth || result.Scanned >= maxEntries {
			result.Truncated = true
			return nil
		}
		dir, err := root.Open(rel)
		if err != nil {
			return nil
		}
		defer dir.Close()
		entries, err := dir.ReadDir(600)
		if err != nil {
			return nil
		}
		if len(entries) > 500 {
			result.Truncated = true
			entries = entries[:500]
		}
		for _, d := range entries {
			if result.Scanned >= maxEntries {
				result.Truncated = true
				break
			}
			child := d.Name()
			if rel != "." {
				child = rel + "/" + child
			}
			if _, err := SafeRelative(child); err != nil {
				continue
			}
			if d.Type()&os.ModeSymlink != 0 {
				continue
			}
			result.Scanned++
			if d.IsDir() {
				if depth < maxDepth {
					_ = walk(child, depth+1)
				} else {
					result.Truncated = true
				}
				continue
			}
			k, _ := class(d.Name())
			if k != "" && (kind == "" || kind == "all" || kind == k) {
				paths = append(paths, child)
			}
		}
		return nil
	}
	if err := walk(".", 0); err != nil {
		return List{}, err
	}
	sort.Strings(paths)
	valid := make([]string, 0, len(paths))
	for _, candidate := range paths {
		f, _, _, _, err := inspected(root, candidate)
		if err != nil {
			continue
		}
		_ = f.Close()
		valid = append(valid, candidate)
	}
	result.Total = len(valid)
	if offset >= len(valid) {
		return result, nil
	}
	end := offset + limit
	if end > len(valid) {
		end = len(valid)
	}
	for _, rel := range valid[offset:end] {
		item, err := Describe(root, rel)
		if err == nil {
			result.Items = append(result.Items, item)
		}
	}
	result.HasMore = end < len(valid)
	return result, nil
}
func Serve(w http.ResponseWriter, r *http.Request, root *os.Root, rel string, inline bool) error {
	f, info, kind, mimeType, err := inspected(root, rel)
	if err != nil {
		return err
	}
	defer f.Close()
	if info.Size() > 2<<30 {
		return ErrForbidden
	}
	name := path.Base(rel)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", mimeType)
	disposition := "attachment"
	if inline && kind != "download" {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", disposition+"; filename="+strconvQuote(name))
	http.ServeContent(w, r, name, info.ModTime(), f)
	return nil
}
func strconvQuote(name string) string {
	// RFC-compatible escaped filename; avoid injecting header metacharacters.
	name = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '\\' || r == '"' || r < 32 {
			return '_'
		}
		return r
	}, name)
	return fmt.Sprintf("%q", name)
}
func NativeType(path string) string {
	_, v := class(path)
	if v == "" {
		return mime.TypeByExtension(filepath.Ext(path))
	}
	return v
}
