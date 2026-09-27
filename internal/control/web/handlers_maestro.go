package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/buildinfo"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/update"
	"github.com/IAPro-Community/nexus/internal/executil"
)

// handleMaestroStatus GET /api/v1/maestro — returns Maestro integration status.
func (h *NexusHandler) handleMaestroStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	client := nexus.NewMaestroClient()
	status := client.Status()
	writeJSON(w, http.StatusOK, status)
}

func (h *NexusHandler) handleMaestroCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	client := nexus.NewMaestroClient()
	writeJSON(w, http.StatusOK, client.Catalog())
}

func (h *NexusHandler) handleMaestroSyncPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	preview, err := nexus.NewMaestroClient().SyncPreview()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (h *NexusHandler) handleMaestroSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var preview nexus.SkillSyncPreview
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&preview); err != nil {
		writeError(w, http.StatusBadRequest, "invalid sync preview")
		return
	}
	result, err := nexus.NewMaestroClient().ApplySyncPreview(r.Context(), preview)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleMaestroAdvice POST /api/v1/maestro/advice — request Maestro recommendations.
func (h *NexusHandler) handleMaestroAdvice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		ProjectID string `json:"project_id"`
		AgentID   string `json:"agent_id"`
		Intent    string `json:"intent"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	client := nexus.NewMaestroClient()
	ctx := nexus.AdviceContext{
		ProjectID: body.ProjectID,
		AgentID:   body.AgentID,
	}
	resp, err := client.GetAdvice(ctx, body.Intent)
	if err != nil {
		// Return degraded response, not 500.
		writeJSON(w, http.StatusOK, map[string]any{
			"mode":     "OFF",
			"error":    err.Error(),
			"degraded": true,
		})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleSystemUpdates GET /api/v1/system/updates — returns signed Nexus Update
// Service status plus explicit Maestro maintenance status. The POST endpoint
// below remains Maestro-only and never replaces the Nexus binary.
func (h *NexusHandler) handleSystemUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	client := nexus.NewMaestroClient()
	mStatus := client.Status()
	maestroVer := "unknown"
	if mStatus.Capabilities != nil {
		maestroVer = mStatus.Capabilities.Version
	}

	// Check npm registry for latest maestro version if available
	latestMaestroVer := maestroVer
	updateAvailable := false
	if npmPath, err := exec.LookPath("npm"); err == nil {
		cmd := exec.Command(npmPath, "view", "@iapro/orquestrador-maestro-cli", "version")
		executil.HideConsoleWindow(cmd)
		if out, err := cmd.Output(); err == nil {
			latestMaestroVer = strings.TrimSpace(string(out))
			if latestMaestroVer != "" && maestroVer != "unknown" && latestMaestroVer != maestroVer {
				updateAvailable = true
			}
		}
	}

	nexusUpdateCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	nexusCheck, nexusErr := checkNexusUpdate(nexusUpdateCtx)
	nexusLatestVersion := buildinfo.Version
	nexusUpdateAvailable := false
	nexusInstallationMethod := update.MethodUnknown
	nexusAllowsSelfUpdate := false
	nexusChannel := "stable"
	var nexusUpdateInstruction string
	var nexusUpdateError string
	if nexusCheck != nil {
		nexusLatestVersion = nexusCheck.LatestVersion
		nexusUpdateAvailable = nexusCheck.UpdateAvailable
		nexusInstallationMethod = nexusCheck.InstallationMethod
		nexusAllowsSelfUpdate = nexusCheck.AllowsSelfUpdate
		nexusUpdateInstruction = nexusCheck.Instruction
		if nexusCheck.Channel != "" {
			nexusChannel = nexusCheck.Channel
		}
	}
	if nexusErr != nil {
		nexusUpdateError = nexusErr.Error()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"nexus_version":             buildinfo.Version,
		"nexus_commit":              buildinfo.Commit,
		"nexus_build_date":          buildinfo.BuildDate,
		"nexus_latest_version":      nexusLatestVersion,
		"nexus_update_available":    nexusUpdateAvailable,
		"nexus_update_error":        nexusUpdateError,
		"nexus_update_instruction":  nexusUpdateInstruction,
		"channel":                   nexusChannel,
		"installation_method":       nexusInstallationMethod,
		"allows_self_update":        nexusAllowsSelfUpdate,
		"nexus_installation_method": nexusInstallationMethod,
		"nexus_allows_self_update":  nexusAllowsSelfUpdate,
		"maestro_version":           maestroVer,
		"maestro_latest_version":    latestMaestroVer,
		"maestro_available":         mStatus.Available,
		"update_available":          updateAvailable,
	})
}

// performSystemUpdate is the explicit Maestro library updater. Tests stub this
// to avoid npm.
var performSystemUpdate = nexus.PerformSystemUpdate

// handleMaestroUpdate POST /api/v1/maestro/update — updates Maestro only after
// an explicit product, target and confirmation contract.
func (h *NexusHandler) handleMaestroUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request struct {
		Product       string `json:"product"`
		TargetVersion string `json:"target_version"`
		Confirmed     bool   `json:"confirmed"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&request); err != nil ||
		request.Product != "maestro" || request.TargetVersion != "latest" || !request.Confirmed {
		writeError(w, http.StatusBadRequest, "explicit Maestro update requires product=maestro, target_version=latest, and confirmed=true")
		return
	}
	writeJSON(w, http.StatusOK, performSystemUpdate())
}
