package web

import (
	"github.com/IAPro-Community/nexus/internal/buildinfo"
	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

// APIError is the stable v1 error payload. The legacy message remains in the
// error field while code gives clients a machine-readable branch point.
type APIError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// SystemInfoResponse is the read-only contract used by clients to negotiate
// the server API and provider capabilities.
type SystemInfoResponse struct {
	APIVersion    string                                `json:"apiVersion"`
	ServerVersion string                                `json:"serverVersion"`
	Build         map[string]string                     `json:"build"`
	Capabilities  map[string]driver.ControlCapabilities `json:"capabilities"`
	Providers     []string                              `json:"providers"`
}

// RuntimeDetailResponse keeps runtime identity and capability metadata
// together without exposing registry secrets or process arguments.
type RuntimeDetailResponse struct {
	Session      registry.RuntimeSession       `json:"session"`
	Capabilities *driver.EffectiveCapabilities `json:"capabilities"`
}

func newSystemInfoResponse(capabilities map[string]driver.ControlCapabilities, providers []string) SystemInfoResponse {
	return SystemInfoResponse{
		APIVersion:    "v1",
		ServerVersion: buildinfo.Version,
		Build:         buildinfo.JSON(),
		Capabilities:  capabilities,
		Providers:     providers,
	}
}
