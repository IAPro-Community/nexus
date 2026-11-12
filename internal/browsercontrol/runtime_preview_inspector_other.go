//go:build !linux

package browsercontrol

// NewSystemRuntimePreviewInspector is intentionally unavailable outside Linux
// until process ancestry and listening socket ownership can be verified there.
func NewSystemRuntimePreviewInspector() RuntimePreviewInspector { return nil }
