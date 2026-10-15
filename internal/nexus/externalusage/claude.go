// Package externalusage provides manually consented, read-only scans of local
// AI tool metadata. No prompts, message text, credentials or paths are stored.
package externalusage

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const (
	SourceScope       = "LOCAL_DEFAULT_UNVERIFIED"
	maxFiles          = 800
	maxBytesPerFile   = 8 << 20
	maxBytesPerScan   = 48 << 20
	maxLineBytes      = 2 << 20
)

type ScanReport struct {
	Files        int   `json:"files"`
	Messages     int   `json:"messages"`
	Bytes        int64 `json:"bytes"`
	HasMore      bool  `json:"has_more"`
	Unattributed bool  `json:"unattributed"`
}

type transcriptLine struct {
	Type      string `json:"type"`
	UUID      string `json:"uuid"`
	Timestamp string `json:"timestamp"`
	Message   struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Usage *struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

func digest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func parseLine(raw []byte) (store.ExternalClaudeMessage, bool) {
	if !strings.Contains(string(raw), "\"usage\"") {
		return store.ExternalClaudeMessage{}, false
	}
	var item transcriptLine
	if json.Unmarshal(raw, &item) != nil {
		return store.ExternalClaudeMessage{}, false
	}
	if item.Type != "assistant" || item.Message.Usage == nil ||
		item.Message.Model == "" || item.Message.Model == "<synthetic>" {
		return store.ExternalClaudeMessage{}, false
	}
	id := strings.TrimSpace(item.Message.ID)
	if id == "" { id = strings.TrimSpace(item.UUID) }
	when, err := time.Parse(time.RFC3339Nano, item.Timestamp)
	u := item.Message.Usage
	if id == "" || err != nil || len(item.Message.Model) > 128 ||
		when.After(time.Now().Add(time.Hour)) || when.Before(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) ||
		u.InputTokens < 0 || u.OutputTokens < 0 ||
		u.CacheReadInputTokens < 0 || u.CacheCreationInputTokens < 0 {
		return store.ExternalClaudeMessage{}, false
	}
	return store.ExternalClaudeMessage{
		MessageHash: digest(id), Model: item.Message.Model,
		InputTokens: u.InputTokens, OutputTokens: u.OutputTokens,
		CacheReadTokens: u.CacheReadInputTokens,
		CacheWriteTokens: u.CacheCreationInputTokens, ObservedAt: when.UTC(),
	}, true
}

// ScanClaudeProjects consumes only newline-terminated JSONL from a fixed,
// caller-controlled directory. Files/bytes/line and cancellation are bounded.
// Symlinks never traverse outside that directory. Caller MUST restrict root
// to the local user's default Claude Code projects, never a request path.
func ScanClaudeProjects(ctx context.Context, db *store.Store, root string) (ScanReport, error) {
	report := ScanReport{Unattributed: true}
	if db == nil { return report, errors.New("external usage store unavailable") }
	info, err := os.Lstat(root)
	if err != nil { return report, err }
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return report, errors.New("Claude projects source is not a directory")
	}
	files := make([]string, 0, 32)
	err = filepath.WalkDir(root, func(path string, item fs.DirEntry, err error) error {
		if err != nil { return err }
		if ctx.Err() != nil { return ctx.Err() }
		if item.Type()&os.ModeSymlink != 0 { return nil }
		if item.IsDir() { return nil }
		if item.Type().IsRegular() && strings.HasSuffix(item.Name(), ".jsonl") {
			files = append(files, path)
			if len(files) > maxFiles { return errors.New("too many local transcript files") }
		}
		return nil
	})
	if err != nil { return report, err }
	sort.Strings(files)
	var scanned int64
	for _, name := range files {
		if ctx.Err() != nil { return report, ctx.Err() }
		info, err := os.Lstat(name)
		if err != nil { return report, err }
		if !info.Mode().IsRegular() { continue }
		hash := digest(name)
		offset, err := db.ExternalClaudeOffset(ctx, SourceScope, hash)
		if err != nil { return report, err }
		if offset > info.Size() { offset = 0 }
		if offset == info.Size() { continue }
		allow := info.Size() - offset
		if allow > maxBytesPerFile { allow = maxBytesPerFile; report.HasMore = true }
		if allow > maxBytesPerScan-scanned { allow = maxBytesPerScan-scanned; report.HasMore = true }
		if allow <= 0 { report.HasMore = true; break }
		file, err := os.Open(name)
		if err != nil { return report, err }
		// Refuse a path swapped for a symlink or another inode after WalkDir.
		// Claude may append while we read; appends are handled by newline offsets.
		openedInfo, statErr := file.Stat()
		if statErr != nil || !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
			_ = file.Close()
			return report, errors.New("Claude transcript changed while opening")
		}
		if _, err = file.Seek(offset, io.SeekStart); err != nil {
			_ = file.Close()
			return report, err
		}
		reader := bufio.NewReader(io.LimitReader(file, allow))
		cursor := offset
		records := make([]store.ExternalClaudeMessage, 0, 64)
		for {
			if ctx.Err() != nil { _ = file.Close(); return report, ctx.Err() }
			line, err := reader.ReadBytes('\n')
			if errors.Is(err, io.EOF) {
				break // incomplete newline: re-read on next explicit scan
			}
			if err != nil { _ = file.Close(); return report, err }
			cursor += int64(len(line))
			if len(line) > maxLineBytes { continue }
			if message, ok := parseLine(line); ok {
				records = append(records, message)
			}
		}
		_ = file.Close()
		if err := db.SaveExternalClaudeBatch(ctx, SourceScope, hash, cursor, records); err != nil {
			return report, err
		}
		report.Files++
		report.Messages += len(records)
		report.Bytes += cursor - offset
		scanned += allow
		if cursor < info.Size() { report.HasMore = true }
	}
	// Retain no more than 90 days of external-source metadata. This is a
	// manual-scan cleanup, never a daemon or background filesystem read.
	if err := db.PruneExternalClaude(ctx, time.Now().UTC().Add(-90*24*time.Hour)); err != nil {
		return report, err
	}
	return report, nil
}
