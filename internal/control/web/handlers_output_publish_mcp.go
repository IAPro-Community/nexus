package web

import (
	"context"
	"encoding/json"

	"github.com/IAPro-Community/nexus/internal/browsercontrol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/artifacts"
)

// Nexus Outputs publication uses the same short-lived runtime capability as
// the Browser MCP. It is scoped to a live supervised runtime and an existing
// Nexus Project: no new identity system and no global filesystem privileges.
func publishRuntimeOutput(ctx context.Context, cap browsercontrol.RuntimeCapability, raw json.RawMessage) any {
	var input struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(raw, &input); err != nil || input.Path == "" {
		return browserMCPError("path is required (relative to the Nexus Project root)")
	}
	if _, err := artifacts.SafeRelative(input.Path); err != nil {
		return browserMCPError("path outside authorized project")
	}
	runtimeSession, ok := registry.DefaultRegistry().Get(cap.RuntimeID)
	if !ok || !runtimeSession.HostLive() || (runtimeSession.ProjectID != "" && runtimeSession.ProjectID != cap.ProjectID) {
		return browserMCPError("supervised runtime unavailable or no longer authorized for this project")
	}
	if err := ctx.Err(); err != nil {
		return browserMCPError("request canceled")
	}
	st, err := nexus.Default().OpenProject()
	if err != nil {
		return browserMCPError("project storage unavailable")
	}
	project, err := st.GetProject(cap.ProjectID)
	if err != nil {
		return browserMCPError("project unavailable")
	}
	root, err := artifacts.Root(project.CanonicalPath)
	if err != nil {
		return browserMCPError("project root unavailable")
	}
	defer root.Close()
	item, err := artifacts.Describe(root, input.Path)
	if err != nil {
		return browserMCPError("file unavailable or unsupported")
	}
	dataDir, err := config.DataDir()
	if err != nil {
		return browserMCPError("artifact storage unavailable")
	}
	agentID := runtimeSession.AgentID
	published, err := artifacts.Observe(root, dataDir, cap.ProjectID, item, "agent", agentID, cap.RuntimeID)
	if err != nil {
		return browserMCPError("could not register artifact")
	}
	return browserMCPText(map[string]any{"published": true, "artifact": published,
		"message": "The file is now discoverable through Outputs on desktop and mobile. Original bytes remain in the project.",
	})
}
