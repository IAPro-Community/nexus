package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Status string

const (
	StatusSuccess    Status = "SUCCESS"
	StatusRolledBack Status = "ROLLED_BACK"
	StatusFailed     Status = "FAILED"
)

type Receipt struct {
	Timestamp       time.Time `json:"timestamp"`
	PreviousVersion string    `json:"previous_version"`
	AppliedVersion  string    `json:"applied_version"`
	SHA256          string    `json:"sha256"`
	Status          Status    `json:"status"`
	Message         string    `json:"message,omitempty"`
}

type Updater struct {
	BinaryPath string
	DataDir    string
}

// ApplyManifest validates the signed-manifest policy and binds the bytes being
// installed to the selected target artifact. Callers must verify the manifest
// signature with KeyRing before invoking this method.
func (u *Updater) ApplyManifest(manifest Manifest, policy ManifestPolicy, data []byte) (*Receipt, error) {
	if err := manifest.Validate(policy); err != nil {
		return nil, err
	}
	artifact := manifest.Artifacts[policy.Target]
	if artifact.Target == "" {
		lowerURL := strings.ToLower(artifact.URL)
		if strings.HasSuffix(lowerURL, ".tar.gz") || strings.HasSuffix(lowerURL, ".zip") {
			return nil, fmt.Errorf("%w: archive artifact is missing an extraction target", ErrManifestArtifact)
		}
	}
	if err := u.VerifyArtifactChecksum(data, artifact.SHA256); err != nil {
		return nil, fmt.Errorf("downloaded artifact failed manifest checksum: %w", err)
	}
	if artifact.Size != int64(len(data)) {
		return nil, fmt.Errorf("artifact size mismatch: expected %d, got %d", artifact.Size, len(data))
	}

	binaryData := data
	if artifact.Target == TargetTarGz || artifact.Target == TargetZip {
		extracted, err := ExtractBinary(data, artifact.Target)
		if err != nil {
			return nil, fmt.Errorf("archive extraction failed: %w", err)
		}
		binaryData = extracted.Data
	}
	return u.ApplyUpdate(policy.CurrentVersion, manifest.Version, binaryData, sha256Hex(binaryData))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func NewUpdater(binaryPath, dataDir string) *Updater {
	return &Updater{
		BinaryPath: binaryPath,
		DataDir:    dataDir,
	}
}

func (u *Updater) VerifyArtifactChecksum(data []byte, expectedHex string) error {
	sum := sha256.Sum256(data)
	actualHex := hex.EncodeToString(sum[:])
	if !strings.EqualFold(actualHex, expectedHex) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHex, actualHex)
	}
	return nil
}

func (u *Updater) ApplyUpdate(prevVersion, newVersion string, newBinaryData []byte, sha256Hex string) (*Receipt, error) {
	if err := u.VerifyArtifactChecksum(newBinaryData, sha256Hex); err != nil {
		receipt := &Receipt{
			Timestamp:       time.Now().UTC(),
			PreviousVersion: prevVersion,
			AppliedVersion:  newVersion,
			SHA256:          sha256Hex,
			Status:          StatusFailed,
			Message:         err.Error(),
		}
		_ = u.saveReceipt(receipt)
		return receipt, err
	}

	backupPath := u.BinaryPath + ".bak"
	if err := os.MkdirAll(filepath.Dir(u.BinaryPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to prepare binary directory: %w", err)
	}

	tempFile := u.BinaryPath + ".tmp"
	_ = os.Remove(tempFile)
	if err := os.WriteFile(tempFile, newBinaryData, 0755); err != nil {
		return nil, fmt.Errorf("failed to write new binary: %w", err)
	}

	if err := replaceExecutable(u.BinaryPath, tempFile, backupPath); err != nil {
		_ = os.Remove(tempFile)
		return nil, err
	}

	receipt := &Receipt{
		Timestamp:       time.Now().UTC(),
		PreviousVersion: prevVersion,
		AppliedVersion:  newVersion,
		SHA256:          sha256Hex,
		Status:          StatusSuccess,
		Message:         "Update applied successfully with atomic backup",
	}
	if err := u.saveReceipt(receipt); err != nil {
		return receipt, fmt.Errorf("update applied but receipt could not be persisted: %w", err)
	}
	return receipt, nil
}

func (u *Updater) Rollback() (*Receipt, error) {
	backupPath := u.BinaryPath + ".bak"
	if _, err := os.Stat(backupPath); err != nil {
		return nil, errors.New("no backup binary found for rollback")
	}

	restorePath := u.BinaryPath + ".restore"
	_ = os.Remove(restorePath)
	if err := os.Rename(backupPath, restorePath); err != nil {
		return nil, fmt.Errorf("failed to stage backup binary: %w", err)
	}
	if err := os.Rename(restorePath, u.BinaryPath); err != nil {
		_ = os.Rename(restorePath, backupPath)
		return nil, fmt.Errorf("failed to restore backup binary: %w", err)
	}

	receipt := &Receipt{
		Timestamp: time.Now().UTC(),
		Status:    StatusRolledBack,
		Message:   "Rolled back to previous binary version from backup",
	}
	if err := u.saveReceipt(receipt); err != nil {
		return receipt, fmt.Errorf("rollback completed but receipt could not be persisted: %w", err)
	}
	return receipt, nil
}

func (u *Updater) saveReceipt(receipt *Receipt) error {
	receiptDir := filepath.Join(u.DataDir, "receipts")
	if err := os.MkdirAll(receiptDir, 0755); err != nil {
		return err
	}
	receiptPath := filepath.Join(receiptDir, fmt.Sprintf("update-%d.json", time.Now().UnixNano()))
	b, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(receiptPath, b, 0644)
}
