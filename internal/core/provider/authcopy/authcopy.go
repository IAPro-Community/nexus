// Package authcopy provides a narrow, no-overwrite path for importing known
// provider credential files into isolated Nexus profile homes.
package authcopy

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

const maxCredentialFileSize = 4 << 20

var ErrDestinationExists = errors.New("profile already contains provider credentials")

// CopyValidatedFile copies one validated credential file. Source symlinks,
// non-regular files, oversized files, invalid contents, and existing targets
// are never followed or overwritten. The source file is never modified.
func CopyValidatedFile(source, destination string, validate func([]byte) bool) (bool, error) {
	if validate == nil {
		return false, errors.New("credential validator is required")
	}
	data, err := readRegularFile(source)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !validate(data) {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return false, errors.New("could not prepare the isolated profile directory")
	}
	directory, err := os.Lstat(filepath.Dir(destination))
	if err != nil || !directory.IsDir() || directory.Mode()&os.ModeSymlink != 0 {
		return false, errors.New("isolated profile directory is not a regular directory")
	}
	if _, err := os.Lstat(destination); err == nil {
		return false, ErrDestinationExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, errors.New("could not inspect the isolated profile")
	}

	temp, err := os.CreateTemp(filepath.Dir(destination), ".nexus-auth-import-*")
	if err != nil {
		return false, errors.New("could not stage host credentials")
	}
	tempName := temp.Name()
	defer func() { _ = os.Remove(tempName) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return false, errors.New("could not secure imported credentials")
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return false, errors.New("could not stage host credentials")
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return false, errors.New("could not persist imported credentials")
	}
	if err := temp.Close(); err != nil {
		return false, errors.New("could not persist imported credentials")
	}

	// A hard link makes publication atomic and fails rather than replacing a
	// credential written concurrently by a login flow.
	if err := os.Link(tempName, destination); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, ErrDestinationExists
		}
		return false, errors.New("could not publish imported credentials")
	}
	return true, nil
}

// ReadValidatedFile checks one known host credential path without returning
// its contents. It rejects symlinks, special files, and oversized files.
func ReadValidatedFile(source string, validate func([]byte) bool) (bool, error) {
	if validate == nil {
		return false, errors.New("credential validator is required")
	}
	data, err := readRegularFile(source)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return validate(data), nil
}

func readRegularFile(source string) ([]byte, error) {
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, errors.New("could not inspect host credentials")
	}
	if !sourceInfo.Mode().IsRegular() || sourceInfo.Size() <= 0 || sourceInfo.Size() > maxCredentialFileSize {
		return nil, errors.New("host credential file is not a supported regular file")
	}
	file, err := os.Open(source)
	if err != nil {
		return nil, errors.New("could not read host credentials")
	}
	openedInfo, statErr := file.Stat()
	if statErr != nil || !os.SameFile(sourceInfo, openedInfo) || !openedInfo.Mode().IsRegular() {
		_ = file.Close()
		return nil, errors.New("host credentials changed during import")
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxCredentialFileSize+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(data) == 0 || len(data) > maxCredentialFileSize {
		return nil, errors.New("could not read host credentials")
	}
	return data, nil
}
