package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/buildinfo"
	"github.com/IAPro-Community/nexus/internal/executil"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/update"
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
		npmCtx, cancelNpm := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancelNpm()
		cmd := exec.CommandContext(npmCtx, npmPath, "view", "@iapro/orquestrador-maestro-cli", "version")
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
	if nexusCheck != nil && nexusCheck.Channel == "beta" && !nexusCheck.UpdateAvailable {
		// A beta should discover the signed stable channel once v1.0.0 ships.
		// Do not replace a newer beta with an older stable release.
		stableCheck, stableErr := checkNexusStableUpdate(nexusUpdateCtx)
		if stableErr == nil && stableCheck != nil && stableCheck.UpdateAvailable {
			nexusCheck, nexusErr = stableCheck, nil
		}
	}
	nexusLatestVersion := buildinfo.Version
	nexusUpdateAvailable := false
	nexusInstallationMethod := update.MethodUnknown
	nexusAllowsSelfUpdate := false
	nexusChannel := "stable"
	var nexusUpdateInstruction string
	var nexusUpdateError string
	var nexusInstallerURL string
	var nexusReleaseURL string
	if nexusCheck != nil {
		nexusLatestVersion = nexusCheck.LatestVersion
		nexusUpdateAvailable = nexusCheck.UpdateAvailable
		nexusInstallationMethod = nexusCheck.InstallationMethod
		nexusAllowsSelfUpdate = nexusCheck.AllowsSelfUpdate
		nexusUpdateInstruction = nexusCheck.Instruction
		if nexusCheck.Channel == "stable" && strings.Contains(buildinfo.Version, "-beta.") &&
			nexusCheck.InstallationMethod == update.MethodStandalone {
			nexusUpdateInstruction = "nexus update --channel=stable"
		}
		if nexusCheck.Channel != "" {
			nexusChannel = nexusCheck.Channel
		}
		if nexusCheck.UpdateAvailable {
			// Only advertise links after the signed manifest validates. Downloads
			// outside the public canonical repository are not presented as an
			// "official installer", even when a custom registry was configured.
			nexusReleaseURL = "https://github.com/IAPro-Community/nexus/releases/tag/" +
				update.ChannelPointerTag(nexusChannel)
			if nexusCheck.InstallerArtifact != nil &&
				isOfficialNexusInstallerURL(nexusCheck.InstallerArtifact.URL) {
				nexusInstallerURL = nexusCheck.InstallerArtifact.URL
			}
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
		"nexus_installer_url":       nexusInstallerURL,
		"nexus_release_url":         nexusReleaseURL,
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

// isOfficialNexusInstallerURL prevents the native app's external-link action
// from representing a third-party URL as a verified official package.
func isOfficialNexusInstallerURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Hostname(), "github.com") ||
		u.Port() != "" || u.User != nil {
		return false
	}
	return strings.HasPrefix(u.EscapedPath(), "/IAPro-Community/nexus/releases/download/") &&
		!strings.HasSuffix(u.Path, "/")
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
