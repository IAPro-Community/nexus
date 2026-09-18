// Package device owns the privacy-respecting installation identity.
//
// installation_id is a random value minted on first run and persisted with
// restrictive permissions. It is NOT a hardware fingerprint: no MAC, serial,
// machine-id, or hostname is ever read. Users can reset it (logout/reset),
// which mints a fresh unrelated identity.
package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

// FileName is the installation identity file inside the data directory.
const FileName = "installation_id"

// LoadOrCreate returns the persisted installation ID, minting and storing a
// fresh one on first run.
func LoadOrCreate(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", fmt.Errorf("device: data directory is required")
	}
	path := filepath.Join(dataDir, FileName)
	raw, err := os.ReadFile(path)
	if err == nil {
		if id := strings.TrimSpace(string(raw)); validInstallationID(id) {
			return id, nil
		}
	}
	id := ids.NewULID()
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(id+"\n"), 0o600); err != nil {
		return "", err
	}
	return id, nil
}

// Reset discards the current identity; the next LoadOrCreate mints a fresh
// unrelated one.
func Reset(dataDir string) error {
	path := filepath.Join(dataDir, FileName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func validInstallationID(id string) bool {
	if len(id) != 26 {
		return false
	}
	for _, r := range id {
		if r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' {
			continue
		}
		return false
	}
	return true
}
