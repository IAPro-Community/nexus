package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

type RegistrationState string

const (
	InstalledUnregistered   RegistrationState = "INSTALLED_UNREGISTERED"
	PendingAuth             RegistrationState = "PENDING_AUTH"
	RegisteredAuthenticated RegistrationState = "REGISTERED_AUTHENTICATED"
	RegisteredAuthExpired   RegistrationState = "REGISTERED_AUTH_EXPIRED"
	UnmanagedEphemeral      RegistrationState = "UNMANAGED_EPHEMERAL"
)

// InstallationRecord describes a discovered binary independently of profiles.
type InstallationRecord struct {
	Provider   string            `json:"provider"`
	Binary     string            `json:"binary"`
	Version    string            `json:"version,omitempty"`
	Path       string            `json:"path,omitempty"`
	Installed  bool              `json:"installed"`
	DetectedAt time.Time         `json:"detected_at"`
	State      RegistrationState `json:"state"`
}

type InstallationRegistry struct {
	mu      sync.Mutex
	records map[string]InstallationRecord
}

func NewInstallationRegistry() *InstallationRegistry {
	r := &InstallationRegistry{records: make(map[string]InstallationRecord)}
	_ = r.load()
	return r
}

func (r *InstallationRegistry) Refresh(reg *Registry, ctx context.Context) ([]InstallationRecord, error) {
	if reg == nil {
		return nil, fmt.Errorf("provider registry is required")
	}
	detections := reg.DetectAll(ctx)
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range reg.List() {
		id := string(p.ID())
		d := detections[id]
		record := r.records[id]
		record.Provider = id
		record.DetectedAt = time.Now().UTC()
		record.Installed = d.Installed
		if d.Installed {
			record.Binary = filepath.Base(d.BinaryPath)
			record.Version = d.Version
			record.Path = d.BinaryPath
			if record.State == "" {
				record.State = InstalledUnregistered
			}
		}
		r.records[id] = record
	}
	return r.listLocked(), r.saveLocked()
}

func (r *InstallationRegistry) List() []InstallationRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.listLocked()
}

// Observe records one successful installation discovery without coupling the
// discovery adapter to account/profile registration.
func (r *InstallationRegistry) Observe(providerID string, detection model.DetectionResult) error {
	if !detection.Installed {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	record := r.records[providerID]
	record.Provider = providerID
	record.Binary = filepath.Base(detection.BinaryPath)
	record.Version = detection.Version
	record.Path = detection.BinaryPath
	record.Installed = true
	record.DetectedAt = time.Now().UTC()
	if record.State == "" {
		record.State = InstalledUnregistered
	}
	r.records[providerID] = record
	return r.saveLocked()
}
func (r *InstallationRegistry) listLocked() []InstallationRecord {
	out := make([]InstallationRecord, 0, len(r.records))
	for _, v := range r.records {
		out = append(out, v)
	}
	return out
}

func (r *InstallationRegistry) load() error {
	dir, err := config.StateDir()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(dir, "installations.json"))
	if err != nil {
		return err
	}
	var records []InstallationRecord
	if err := json.Unmarshal(b, &records); err != nil {
		return err
	}
	for _, record := range records {
		// Records written before the explicit Installed field represent a
		// successful discovery unless they were empty placeholders.
		if !record.Installed && (record.Path != "" || record.Version != "" || record.Binary != "") {
			record.Installed = true
		}
		r.records[record.Provider] = record
	}
	return nil
}
func (r *InstallationRegistry) saveLocked() error {
	dir, err := config.StateDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(r.listLocked(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "installations.json"), append(b, '\n'), 0600)
}
