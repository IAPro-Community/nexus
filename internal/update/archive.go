package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
)

var (
	ErrArchiveTraversal    = errors.New("archive contains path traversal attempt")
	ErrArchiveTooLarge     = errors.New("archive extraction exceeds maximum allowed size")
	ErrArchiveNoExecutable = errors.New("archive contains no executable binary")
	ErrArchiveInvalid      = errors.New("archive format is invalid or corrupted")
)

const (
	// MaxExtractSize is the maximum allowed extraction size (256 MB).
	MaxExtractSize = 256 * 1024 * 1024
)

// ExtractedBinary represents a binary extracted from an archive.
type ExtractedBinary struct {
	Name string
	Size int64
	Data []byte
}

// ExtractBinary safely extracts the main binary from an archive.
// It validates paths against traversal attacks and enforces size limits.
func ExtractBinary(data []byte, target ArtifactTarget) (*ExtractedBinary, error) {
	switch target {
	case TargetTarGz:
		return extractFromTarGz(data)
	case TargetZip:
		return extractFromZip(data)
	case TargetBinary:
		return &ExtractedBinary{
			Name: "nexus",
			Size: int64(len(data)),
			Data: data,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported archive target: %s", target)
	}
}

func extractFromTarGz(data []byte) (*ExtractedBinary, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrArchiveInvalid, err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var candidates []ExtractedBinary
	var totalSize int64

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrArchiveInvalid, err)
		}

		if err := validatePath(hdr.Name); err != nil {
			return nil, err
		}

		// hdr.Size is attacker-controlled: reject negatives and guard the
		// running total against int64 overflow, mirroring the zip path.
		if hdr.Size < 0 || hdr.Size > MaxExtractSize ||
			uint64(totalSize) > uint64(MaxExtractSize)-uint64(hdr.Size) {
			return nil, ErrArchiveTooLarge
		}
		totalSize += hdr.Size

		if hdr.FileInfo().Mode().IsRegular() && isBinaryName(hdr.Name) {
			buf := make([]byte, hdr.Size)
			if _, err := io.ReadFull(tr, buf); err != nil {
				return nil, fmt.Errorf("%w: failed to read %s: %v", ErrArchiveInvalid, hdr.Name, err)
			}
			candidates = append(candidates, ExtractedBinary{
				Name: path.Base(hdr.Name),
				Size: hdr.Size,
				Data: buf,
			})
		}
	}

	if len(candidates) == 0 {
		return nil, ErrArchiveNoExecutable
	}

	// Prefer "nexus" or "nexus.exe" binary
	for _, c := range candidates {
		base := strings.ToLower(c.Name)
		if base == "nexus" || base == "nexus.exe" {
			return &c, nil
		}
	}

	// Fallback to first binary found
	return &candidates[0], nil
}

func extractFromZip(data []byte) (*ExtractedBinary, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrArchiveInvalid, err)
	}

	var candidates []ExtractedBinary
	var totalSize int64

	for _, f := range r.File {
		if err := validatePath(f.Name); err != nil {
			return nil, err
		}

		// ZIP reports sizes as uint64 while the extraction limit and the
		// extracted binary contract use int64. Check before converting so a
		// malicious archive cannot overflow the running total.
		if f.UncompressedSize64 > uint64(MaxExtractSize) ||
			uint64(totalSize) > uint64(MaxExtractSize)-f.UncompressedSize64 {
			return nil, ErrArchiveTooLarge
		}
		totalSize += int64(f.UncompressedSize64)

		if !f.FileInfo().Mode().IsRegular() || !isBinaryName(f.Name) {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%w: failed to open %s: %v", ErrArchiveInvalid, f.Name, err)
		}

		buf := make([]byte, f.UncompressedSize64)
		if _, err := io.ReadFull(rc, buf); err != nil {
			rc.Close()
			return nil, fmt.Errorf("%w: failed to read %s: %v", ErrArchiveInvalid, f.Name, err)
		}
		rc.Close()

		candidates = append(candidates, ExtractedBinary{
			Name: path.Base(f.Name),
			Size: int64(f.UncompressedSize64),
			Data: buf,
		})
	}

	if len(candidates) == 0 {
		return nil, ErrArchiveNoExecutable
	}

	// Prefer "nexus" or "nexus.exe" binary
	for _, c := range candidates {
		base := strings.ToLower(c.Name)
		if base == "nexus" || base == "nexus.exe" {
			return &c, nil
		}
	}

	// Fallback to first binary found
	return &candidates[0], nil
}

// validatePath checks for path traversal attacks.
func validatePath(name string) error {
	// Normalize the path
	cleaned := filepath.ToSlash(name)

	// Reject absolute paths
	if strings.HasPrefix(cleaned, "/") {
		return fmt.Errorf("%w: absolute path %s", ErrArchiveTraversal, name)
	}

	// Reject path traversal attempts
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("%w: relative path %s", ErrArchiveTraversal, name)
	}

	// Reject backslash traversal (Windows)
	if strings.Contains(name, "\\") && strings.Contains(name, "..") {
		return fmt.Errorf("%w: backslash traversal %s", ErrArchiveTraversal, name)
	}

	return nil
}

// isBinaryName checks if the filename looks like a nexus binary.
func isBinaryName(name string) bool {
	base := strings.ToLower(path.Base(name))
	return base == "nexus" || base == "nexus.exe"
}
