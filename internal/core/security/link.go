package security

import (
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/IAPro-Community/nexus/internal/executil"
)

// SafeLinkOrCopy links or copies a source file/directory to destination across Linux, macOS, and Windows.
// On Unix, it creates a symlink.
// On Windows, if symlinking fails due to permissions (un-elevated process), it creates hardlinks for files
// and directory junctions / directory copies for directories.
func SafeLinkOrCopy(src, dst string) error {
	srcFi, err := os.Stat(src)
	if err != nil {
		return err
	}

	_ = os.MkdirAll(filepath.Dir(dst), 0700)

	// Clean up existing destination if needed
	if fi, err := os.Lstat(dst); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(dst)
			if err == nil && target == src {
				return nil
			}
			_ = os.Remove(dst)
		} else if fi.IsDir() && srcFi.IsDir() {
			// Directory already exists, will populate contents below
		} else {
			_ = os.RemoveAll(dst)
		}
	}

	// 1. Try Symlink first
	err = os.Symlink(src, dst)
	if err == nil {
		return nil
	}

	// 2. Fallback on Windows or systems without symlink privileges
	if !srcFi.IsDir() {
		// Try hardlink for files
		if linkErr := os.Link(src, dst); linkErr == nil {
			return nil
		}
		// Fallback: Copy file
		return CopyFile(src, dst)
	}

	// For directories on Windows, try directory junction (mklink /J dst src)
	if runtime.GOOS == "windows" {
		cmd := executil.Command("cmd", "/c", "mklink", "/J", dst, src)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// Fallback: Recursive copy for directories
	return CopyDir(src, dst)
}

// CopyFile copies a single file from src to dst with permissions.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// CopyDir recursively copies directory contents from src to dst.
func CopyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0700); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := CopyFile(srcPath, dstPath); err != nil {
				continue
			}
		}
	}
	return nil
}
