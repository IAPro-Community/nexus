package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// handleResourcesList GET /api/v1/resources — returns available provider accounts
// and the scheduler recommendation for the current context.
func (h *NexusHandler) handleResourcesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	accounts, err := h.resources.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if accounts == nil {
		accounts = []nexus.ProviderAccount{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"accounts":         accounts,
		"balance_entities": nexus.BuildBalanceInventory(accounts),
		"policy":           "BALANCED",
	})
}

// handleResourcesRefresh POST /api/v1/resources/refresh — force live quota probes then list.
func (h *NexusHandler) handleResourcesRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Provider string `json:"provider"`
		Profile  string `json:"profile"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req)
	}
	targetProv := strings.TrimSpace(req.Provider)
	targetName := strings.TrimSpace(req.Profile)
	if targetProv == "" {
		targetProv = strings.TrimSpace(r.URL.Query().Get("provider"))
	}
	if targetName == "" {
		targetName = strings.TrimSpace(r.URL.Query().Get("profile"))
	}

	if targetProv != "" && targetName != "" {
		_ = profile.RefreshUsageSnapshot(targetProv, targetName)
	} else {
		profiles, err := profile.List()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, p := range profiles {
			_ = profile.RefreshUsageSnapshot(p.Provider, p.Name)
		}
	}
	accounts, err := h.resources.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if accounts == nil {
		accounts = []nexus.ProviderAccount{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"accounts":         accounts,
		"balance_entities": nexus.BuildBalanceInventory(accounts),
		"policy":           "BALANCED",
		"refreshed":        true,
	})
}

// handleResourceSelect POST /api/v1/resources/select — persist a manually
// selected, eligible resource for an Agent.
func (h *NexusHandler) handleResourceSelect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Provider string `json:"provider"`
		Profile  string `json:"profile"`
		Policy   string `json:"policy"`
		AgentID  string `json:"agent_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if strings.TrimSpace(body.AgentID) == "" || strings.TrimSpace(body.Provider) == "" || strings.TrimSpace(body.Profile) == "" {
		writeError(w, http.StatusBadRequest, "agent_id, provider and profile are required")
		return
	}
	if body.Policy != "" && nexus.SchedulerPolicy(body.Policy) != nexus.PolicyManual {
		writeError(w, http.StatusBadRequest, "resource allocation requires MANUAL policy; use a recommendation endpoint for automatic selection")
		return
	}
	allocation, err := h.resources.Allocate(r.Context(), body.AgentID, body.Provider, body.Profile, nexus.PolicyManual)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, allocation)
}

// handleResourceRecommend POST /api/v1/resources/recommend — evaluate accounts against TaskRequirements
func (h *NexusHandler) handleResourceRecommend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Requirements nexus.TaskRequirements `json:"requirements"`
		Policy       string                 `json:"policy"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	accounts, err := h.resources.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	policy := nexus.SchedulerPolicy(strings.ToUpper(req.Policy))
	if policy == "" {
		policy = nexus.PolicyBalanced
	}
	result, err := h.resources.Recommend(r.Context(), accounts, req.Requirements, policy)
	if err != nil {
		writeError(w, http.StatusRequestTimeout, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
