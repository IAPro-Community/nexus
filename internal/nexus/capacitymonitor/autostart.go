package capacitymonitor

import (
	"context"
	"errors"
)

var (
	// ErrAutostartUnsupported indicates the current platform/OS cannot configure autostart.
	ErrAutostartUnsupported = errors.New("autostart is only supported on Linux systemd environments")
	// ErrAutostartDisabled indicates background autostart is disabled by user preferences.
	ErrAutostartDisabled = errors.New("background autostart is disabled by user preferences")
)

// AutostartStatus represents the operating system level registration of the capacity monitor service.
type AutostartStatus struct {
	Supported bool   `json:"supported"`
	Installed bool   `json:"installed"`
	Enabled   bool   `json:"enabled"`
	Active    bool   `json:"active"`
	UnitPath  string `json:"unit_path,omitempty"`
}

// AutostartManager controls installation, inspection, and removal of background autostart units.
type AutostartManager interface {
	Status(ctx context.Context) (AutostartStatus, error)
	InstallAndEnable(ctx context.Context, execPath string) error
	DisableAndRemove(ctx context.Context) error
}
