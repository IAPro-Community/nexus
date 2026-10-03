package web

import (
	"context"
	"net/http"
	"os"
	stdruntime "runtime"

	"github.com/IAPro-Community/nexus/internal/buildinfo"
	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/doctor"
	"github.com/IAPro-Community/nexus/internal/nexus"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
	"github.com/IAPro-Community/nexus/internal/update"
	"github.com/IAPro-Community/nexus/internal/voice"
)

// NexusHandler serves the Nexus product API (projects, agents, generations,
// lineage, layouts) on top of the shared control core.
type NexusHandler struct {
	auth                  *AuthManager
	nexus                 *nexus.Nexus
	drivers               *driver.Registry
	resources             *nexus.ResourceApplicationService
	projects              *nexus.ProjectApplicationService
	agents                *nexus.AgentApplicationService
	plans                 *nexus.PlanApplicationService
	composer              *nexus.ComposerApplicationService
	runs                  *nexus.RunApplicationService
	hud                   *nexus.HUDApplicationService
	voice                 *voice.Service
	designMode            *designModeManager
	computerScreenshots   *computerScreenshotManager
	hostFilesystemEnabled bool
}

// checkNexusUpdate is injectable so Web API tests never depend on the remote
// update registry. Production uses the same signed-manifest Update Service as
// the CLI and Desktop surfaces.
var checkNexusUpdate = func(ctx context.Context) (*update.CheckResult, error) {
	execPath, _ := os.Executable()
	service := update.NewService(update.ServiceConfig{
		CurrentVer: buildinfo.Version,
		ExecPath:   execPath,
	})
	return service.Check(ctx)
}

// handleSystemDoctor exposes the same read-only diagnostic report used by the
// CLI. The Web layer does not probe or mutate a second set of state.
func (h *NexusHandler) handleSystemDoctor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	detections := make(map[string]model.DetectionResult)
	for _, providerDriver := range h.controlDrivers().List() {
		if providerDriver.ProviderID() == "fake" {
			continue
		}
		detection, _ := providerDriver.Detect(r.Context())
		detections[providerDriver.ProviderID()] = detection
	}
	report := doctor.BuildReport(buildinfo.Version, detections, nexusruntime.DefaultCredentialIsolator().Capability())
	// Keep the response shape stable and make the runtime platform explicit for
	// clients rendering remediation guidance.
	if report.OS == "" {
		report.OS = stdruntime.GOOS
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *NexusHandler) controlDrivers() *driver.Registry {
	if h != nil && h.drivers != nil {
		return h.drivers
	}
	return driver.DefaultRegistry()
}

func NewNexusHandler(auth *AuthManager) *NexusHandler {
	n := nexus.Default()
	// Wire runtime change notifications to the terminal broker (Gate 4).
	broker := DefaultBroker()
	n.SetRuntimeObservers(
		broker.NotifyRuntimeChanged,
		broker.NotifyAgentState,
		broker.NotifyContinuity,
	)
	return &NexusHandler{
		auth:                  auth,
		nexus:                 n,
		drivers:               driver.DefaultRegistry(),
		resources:             nexus.NewResourceApplicationService(n),
		projects:              nexus.NewProjectApplicationService(n),
		agents:                nexus.NewAgentApplicationService(n),
		plans:                 nexus.NewPlanApplicationService(n),
		composer:              nexus.NewComposerApplicationService(n),
		runs:                  nexus.NewRunApplicationService(n),
		hud:                   nexus.NewHUDApplicationService(n),
		voice:                 voice.NewService(),
		designMode:            newDesignModeManager(),
		computerScreenshots:   newComputerScreenshotManager(),
		hostFilesystemEnabled: true,
	}
}

func (h *NexusHandler) setHostFilesystemEnabled(enabled bool) {
	h.hostFilesystemEnabled = enabled
}
