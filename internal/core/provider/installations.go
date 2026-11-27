package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sdk "github.com/IAPro-Community/IAPro-SDK/providers"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

type RegistrationState = sdk.RegistrationState
type InstallationRecord = sdk.InstallationRecord

const (
	InstalledUnregistered   = sdk.InstalledUnregistered
	RegistrationUnknown     = sdk.RegistrationUnknown
	PendingAuth             = sdk.PendingAuth
	RegisteredAuthenticated = sdk.RegisteredAuthenticated
	RegisteredAuthExpired   = sdk.RegisteredAuthExpired
	UnmanagedEphemeral      = sdk.UnmanagedEphemeral
)

// The Nexus wrapper supplies only storage and legacy API compatibility.
type InstallationRegistry struct {
	shared  *sdk.InstallationRegistry
	loadErr error
}

func NewInstallationRegistry() *InstallationRegistry {
	shared, err := sdk.NewInstallationRegistry(installationFile{}, time.Now)
	return &InstallationRegistry{shared: shared, loadErr: err}
}

func (r *InstallationRegistry) Refresh(reg *Registry, ctx context.Context) ([]InstallationRecord, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	if reg == nil {
		return nil, fmt.Errorf("provider registry is required")
	}
	detections := make(map[string]sdk.Detection)
	for id, d := range reg.DetectAll(ctx) {
		detections[id] = sdk.Detection{Installed: d.Installed, Version: d.Version, BinaryPath: d.BinaryPath, Error: d.Error}
	}
	return r.shared.Apply(detections)
}

func (r *InstallationRegistry) List() []InstallationRecord {
	if r.shared == nil {
		return nil
	}
	return r.shared.List()
}

func (r *InstallationRegistry) Observe(providerID string, d model.DetectionResult) error {
	if r.loadErr != nil {
		return r.loadErr
	}
	if !d.Installed {
		return nil
	}
	_, err := r.shared.Apply(map[string]sdk.Detection{providerID: {Installed: d.Installed, BinaryPath: d.BinaryPath, Version: d.Version, Error: d.Error}})
	return err
}

type installationFile struct{}

func (installationFile) Load() ([]InstallationRecord, error) {
	dir, err := config.StateDir()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "installations.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Only legacy records with an absent Installed field require migration.
	var records []struct {
		InstallationRecord
		Installed *bool `json:"installed"`
	}
	if err := json.Unmarshal(raw, &records); err != nil {
		return nil, err
	}
	out := make([]InstallationRecord, 0, len(records))
	for _, item := range records {
		rec := item.InstallationRecord
		if item.Installed != nil {
			rec.Installed = *item.Installed
		} else {
			rec.Installed = rec.Path != "" || rec.Version != "" || rec.Binary != ""
		}
		out = append(out, rec)
	}
	return out, nil
}

func (installationFile) Save(records []InstallationRecord) error {
	dir, err := config.StateDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".installations-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(append(raw, '\n')); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), filepath.Join(dir, "installations.json"))
}
