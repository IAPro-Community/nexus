package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// handleProjectPlans GET/POST /api/v1/projects/{projectID}/plans
func (h *NexusHandler) handleProjectPlans(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		plans, err := h.plans.List(r.Context(), projectID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if plans == nil {
			plans = []store.WorkPlan{}
		}
		writeJSON(w, http.StatusOK, plans)

	case http.MethodPost:
		var body struct {
			Title       string            `json:"title"`
			Description string            `json:"description"`
			Goal        string            `json:"goal"`
			AutoPlan    bool              `json:"auto_plan"`
			Phases      []store.PlanPhase `json:"phases"`
			Facts       map[string]string `json:"facts"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}

		if body.AutoPlan {
			writeError(w, http.StatusBadRequest, "auto_plan is retired; create an empty draft or use Composer then Turn into Flow")
			return
		}

		if strings.TrimSpace(body.Title) == "" {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}

		plan, err := h.plans.Create(r.Context(), projectID, body.Title, body.Description, body.Phases, body.Facts)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, plan)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleProjectComposerSessions creates or lists durable prompt elaborations.
func (h *NexusHandler) handleProjectComposerSessions(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFromPath(r.URL.Path)
	if projectID == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := h.composer.List(r.Context(), projectID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if items == nil {
			items = []store.ComposerSession{}
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var body struct {
			Goal         string `json:"goal"`
			InputMode    string `json:"input_mode"`
			SourcePrompt string `json:"source_prompt"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if wantsComposerProgressStream(r) {
			h.streamComposerCreate(w, r, projectID, body.Goal, body.SourcePrompt, body.InputMode)
			return
		}
		view, err := h.composer.Create(r.Context(), projectID, body.Goal, body.SourcePrompt, body.InputMode)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, view)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleComposerSession exposes one conversation, adds turns and finalizes its prompt.
func (h *NexusHandler) handleComposerSession(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/composer-sessions/"), "/")
	if id == "" {
		writeError(w, http.StatusNotFound, "missing composer session id")
		return
	}
	if strings.HasSuffix(id, "/attachments") {
		sessionID := strings.TrimSuffix(id, "/attachments")
		h.handleComposerAttachments(w, r, sessionID)
		return
	}
	if strings.HasSuffix(id, "/refine") {
		id = strings.TrimSuffix(id, "/refine")
		var body struct {
			Goal string `json:"goal"`
		}
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required for refine")
			return
		}
		_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body)
		artifact, err := h.composer.Refine(r.Context(), id, body.Goal)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, artifact)
		return
	}
	if strings.Contains(id, "/unknowns/") && strings.HasSuffix(id, "/resolve") {
		prefix := strings.TrimSuffix(id, "/resolve")
		parts := strings.SplitN(prefix, "/unknowns/", 2)
		if r.Method != http.MethodPost || len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			writeError(w, http.StatusBadRequest, "session id, unknown id and POST are required")
			return
		}
		sessionID := parts[0]
		unknownID := strings.Trim(parts[1], "/")
		var body struct {
			Answer           string `json:"answer"`
			Status           string `json:"status"`
			ExpectedRevision int    `json:"expected_revision,omitempty"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if strings.TrimSpace(body.Status) == "" {
			body.Status = "ANSWERED"
		}
		view, err := h.composer.ResolveUnknown(r.Context(), sessionID, unknownID, body.Answer, body.Status, body.ExpectedRevision)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	if strings.HasSuffix(id, "/turns") {
		id = strings.TrimSuffix(id, "/turns")
		var body struct {
			Content          string                        `json:"content"`
			ExpectedRevision int                           `json:"expected_revision,omitempty"`
			Attachments      []nexus.ComposerAttachmentRef `json:"attachments,omitempty"`
		}
		if r.Method != http.MethodPost || json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body) != nil {
			writeError(w, http.StatusBadRequest, "content is required")
			return
		}
		if strings.TrimSpace(body.Content) == "" && len(body.Attachments) == 0 {
			writeError(w, http.StatusBadRequest, "content or attachments are required")
			return
		}
		view, err := h.composer.AddTurn(r.Context(), id, body.Content, body.ExpectedRevision, body.Attachments)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, view)
		return
	}
	if strings.HasSuffix(id, "/finalize") {
		id = strings.TrimSuffix(id, "/finalize")
		var body struct {
			SkillIDs    []string `json:"skill_ids"`
			ConfirmGaps bool     `json:"confirm_gaps"`
		}
		if r.Method != http.MethodPost || json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body) != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		artifact, err := h.composer.Finalize(r.Context(), id, body.SkillIDs, body.ConfirmGaps)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, artifact)
		return
	}
	if strings.HasSuffix(id, "/context/remove") {
		id = strings.TrimSuffix(id, "/context/remove")
		var body struct {
			Path             string `json:"path"`
			ExpectedRevision int    `json:"expected_revision,omitempty"`
		}
		if r.Method != http.MethodPost || json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body) != nil || strings.TrimSpace(body.Path) == "" {
			writeError(w, http.StatusBadRequest, "path and POST are required")
			return
		}
		view, err := h.composer.DetachContextFile(r.Context(), id, body.Path, body.ExpectedRevision)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	if strings.HasSuffix(id, "/context") {
		id = strings.TrimSuffix(id, "/context")
		var body struct {
			Path             string `json:"path"`
			ExpectedRevision int    `json:"expected_revision,omitempty"`
		}
		if r.Method != http.MethodPost || json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body) != nil || strings.TrimSpace(body.Path) == "" {
			writeError(w, http.StatusBadRequest, "path and POST are required")
			return
		}
		view, err := h.composer.AttachContextFile(r.Context(), id, body.Path, body.ExpectedRevision)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, view)
		return
	}
	if strings.Contains(id, "/skills/") {
		parts := strings.SplitN(id, "/skills/", 2)
		if r.Method != http.MethodPatch || len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			writeError(w, http.StatusBadRequest, "skill id and PATCH are required")
			return
		}
		var body struct {
			State string `json:"state"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body) != nil || strings.TrimSpace(body.State) == "" {
			writeError(w, http.StatusBadRequest, "state is required")
			return
		}
		view, err := h.composer.UpdateSkill(r.Context(), parts[0], parts[1], body.State)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	view, err := h.composer.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *NexusHandler) handlePromptArtifact(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/prompt-artifacts/")
	if !strings.HasSuffix(path, "/flow") || r.Method != http.MethodPost {
		writeError(w, http.StatusNotFound, "prompt artifact flow materialization not found")
		return
	}
	id := strings.TrimSuffix(path, "/flow")
	id = strings.Trim(id, "/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing prompt artifact id")
		return
	}
	plan, err := h.nexus.MaterializePromptArtifactAsFlow(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, plan)
}

// handlePlanDetail GET/PUT/DELETE /api/v1/plans/{id}
func (h *NexusHandler) handlePlanDetail(w http.ResponseWriter, r *http.Request) {
	planID := strings.TrimPrefix(r.URL.Path, "/api/v1/plans/")
	slashIdx := strings.Index(planID, "/")
	if slashIdx != -1 {
		planID = planID[:slashIdx]
	}
	if planID == "" {
		writeError(w, http.StatusNotFound, "missing plan id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		plan, err := h.plans.Get(r.Context(), planID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		revisions, _ := h.plans.Revisions(r.Context(), planID)
		writeJSON(w, http.StatusOK, map[string]any{
			"plan":      plan,
			"revisions": revisions,
		})

	case http.MethodPut:
		var body struct {
			Plan          store.WorkPlan `json:"plan"`
			ChangeSummary string         `json:"change_summary"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		body.Plan.ID = planID
		updated, rev, err := h.plans.Update(r.Context(), body.Plan, body.ChangeSummary)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"plan":     updated,
			"revision": rev,
		})

	case http.MethodDelete:
		if err := h.plans.Delete(r.Context(), planID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handlePlanCompile POST /api/v1/plans/{id}/compile
func (h *NexusHandler) handlePlanCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimPrefix(r.URL.Path, "/api/v1/plans/")
	planID = strings.TrimSuffix(planID, "/compile")
	if planID == "" {
		writeError(w, http.StatusNotFound, "missing plan id")
		return
	}

	var body struct {
		PhaseID   string `json:"phase_id"`
		PackageID string `json:"package_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || body.PackageID == "" {
		writeError(w, http.StatusBadRequest, "package_id is required")
		return
	}

	compiled, err := h.nexus.CompilePackagePrompt(r.Context(), planID, body.PhaseID, body.PackageID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, compiled)
}

func (h *NexusHandler) handleFlowLeader(w http.ResponseWriter, r *http.Request) {
	planID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/leader")
	if planID == "" {
		writeError(w, http.StatusBadRequest, "plan id is required")
		return
	}
	if r.Method == http.MethodGet {
		policy, err := h.nexus.GetFlowLeaderPolicy(r.Context(), planID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, policy)
		return
	}
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var policy nexus.FlowLeaderPolicy
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&policy); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	plan, err := h.nexus.SetFlowLeaderPolicy(r.Context(), planID, policy)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan": plan, "leader": policy})
}

func (h *NexusHandler) handleFlowClone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/clone")
	var body struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || strings.TrimSpace(body.ProjectID) == "" {
		writeError(w, http.StatusBadRequest, "destination project_id is required")
		return
	}
	clone, err := h.nexus.CloneFlowToProject(r.Context(), planID, body.ProjectID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, clone)
}

func (h *NexusHandler) handlePlanExecutionPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/execution-preview")
	if planID == "" {
		writeError(w, http.StatusBadRequest, "plan id is required")
		return
	}
	preview, err := h.nexus.PreviewPlanExecution(r.Context(), planID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (h *NexusHandler) handleFlowPreflight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/preflight")
	if planID == "" {
		writeError(w, http.StatusBadRequest, "plan id is required")
		return
	}
	report, err := h.nexus.PreflightFlow(r.Context(), planID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *NexusHandler) handlePlanDelegationApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/delegation/approve")
	if planID == "" {
		writeError(w, http.StatusBadRequest, "plan id is required")
		return
	}
	plan, err := h.nexus.ApproveDelegation(r.Context(), planID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

// handlePlanRun POST /api/v1/plans/{id}/run
func (h *NexusHandler) handlePlanRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	planID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/run")
	if planID == "" {
		writeError(w, http.StatusBadRequest, "plan id is required")
		return
	}
	var body struct {
		AgentID               string   `json:"agent_id"`
		PlanRevision          int      `json:"plan_revision"`
		MaxRetry              int      `json:"max_retries"`
		MaxTotalIterations    int      `json:"max_total_iterations"`
		PackageTimeoutSeconds int      `json:"package_timeout_seconds"`
		VerificationCommands  []string `json:"verification_commands"`
		Autonomous            *bool    `json:"autonomous"`
		CostConfirmationToken string   `json:"cost_confirmation_token"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.PlanRevision <= 0 {
		writeError(w, http.StatusBadRequest, "plan_revision is required")
		return
	}
	contract := runner.DefaultAutonomyContract()
	if body.MaxRetry > 0 {
		contract.MaxRetries = body.MaxRetry
	}
	if body.MaxTotalIterations > 0 {
		contract.MaxTotalIterations = body.MaxTotalIterations
	}
	if body.PackageTimeoutSeconds > 0 {
		contract.PackageTimeoutSeconds = body.PackageTimeoutSeconds
	}
	if len(body.VerificationCommands) > 0 {
		contract.VerificationCommands = body.VerificationCommands
	}
	autonomous := true
	if body.Autonomous != nil {
		autonomous = *body.Autonomous
	}
	run, err := h.runs.Start(r.Context(), nexus.RunStartRequest{
		PlanID:                planID,
		Revision:              body.PlanRevision,
		AgentID:               body.AgentID,
		Contract:              contract,
		Autonomous:            autonomous,
		CostConfirmationToken: body.CostConfirmationToken,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

// handleRunsList GET /api/v1/runs
func (h *NexusHandler) handleRunsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	runs, err := h.runs.List(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

// handleRunDetail exposes durable MissionRun state and explicit control actions.
// POST /step exists for diagnostics; normal product execution uses the background worker.
func (h *NexusHandler) handleRunDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/runs/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	runID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = strings.ToLower(parts[1])
	}

	if r.Method == http.MethodGet && action == "" {
		run, err := h.runs.Get(r.Context(), runID)
		if err != nil {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeJSON(w, http.StatusOK, run)
		return
	}
	if r.Method == http.MethodGet && action == "evidence" {
		h.handleRunEvidence(w, r, runID)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Reason         string `json:"reason"`
		InterventionID string `json:"intervention_id"`
		Version        int    `json:"version"`
		OptionID       string `json:"option_id"`
		ResolvedBy     string `json:"resolved_by"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body)
	var (
		run *runner.MissionRun
		err error
	)
	switch action {
	case "pause":
		run, err = h.runs.Pause(r.Context(), runID, firstNonEmpty(body.Reason, "paused by user"))
	case "take-control":
		run, err = h.runs.TakeControl(r.Context(), runID, firstNonEmpty(body.Reason, "manual takeover"))
	case "resume":
		run, err = h.runs.Resume(r.Context(), runID)
	case "return-to-mission":
		run, err = h.runs.ReturnToMission(r.Context(), runID)
	case "cancel":
		run, err = h.runs.Cancel(r.Context(), runID, firstNonEmpty(body.Reason, "canceled by user"))
	case "resolve-intervention":
		if body.InterventionID == "" || body.OptionID == "" {
			writeError(w, http.StatusBadRequest, "intervention_id, version and option_id are required")
			return
		}
		run, err = h.runs.ResolveIntervention(r.Context(), runID, body.InterventionID, body.Version, body.OptionID, body.ResolvedBy)
	case "step", "":
		result, stepErr := h.runs.Step(r.Context(), runID)
		err = stepErr
		if err == nil {
			writeJSON(w, http.StatusOK, map[string]any{"run": result.Run, "completed": result.Completed})
			return
		}
	default:
		writeError(w, http.StatusNotFound, "unknown run action")
		return
	}
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// handleRunEvidence returns only typed, bounded Flow evidence. Raw Agent or
// provider transcripts are intentionally absent from this API because they are
// not part of ContextCapsule/WorkReceipt persistence.
func (h *NexusHandler) handleRunEvidence(w http.ResponseWriter, r *http.Request, runID string) {
	if _, err := h.runs.Get(r.Context(), runID); err != nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	capsuleRecords, err := st.ListFlowContextCapsules(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	receiptRecords, err := st.ListFlowWorkReceipts(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	capsules := make([]runner.ContextCapsule, 0, len(capsuleRecords))
	for _, record := range capsuleRecords {
		var capsule runner.ContextCapsule
		if err := json.Unmarshal([]byte(record.ContentJSON), &capsule); err != nil {
			writeError(w, http.StatusInternalServerError, "decode persisted context capsule")
			return
		}
		capsules = append(capsules, capsule)
	}
	receipts := make([]runner.WorkReceipt, 0, len(receiptRecords))
	for _, record := range receiptRecords {
		var receipt runner.WorkReceipt
		if err := json.Unmarshal([]byte(record.ContentJSON), &receipt); err != nil {
			writeError(w, http.StatusInternalServerError, "decode persisted work receipt")
			return
		}
		receipts = append(receipts, receipt)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"run_id":   runID,
		"capsules": capsules,
		"receipts": receipts,
	})
}

// handleRunRouting exposes the persisted allocation explanation for one run.
// The response is a read model of the decision captured before provider
// execution; it never re-runs routing against current health or quota.
func (h *NexusHandler) handleRunRouting(w http.ResponseWriter, r *http.Request, runID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, err := h.runs.Get(r.Context(), runID); err != nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	report, err := h.runs.Routing(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// handleRunValidationEvidence exposes the canonical append-only validation
// evidence projection for one run. A missing stream is returned as an empty
// report with chain_verified=false; the API never manufactures a PASS claim.
func (h *NexusHandler) handleRunValidationEvidence(w http.ResponseWriter, r *http.Request, runID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, err := h.runs.Get(r.Context(), runID); err != nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	report, err := h.runs.ValidationEvidence(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *NexusHandler) handleRunEvidenceBundle(w http.ResponseWriter, r *http.Request, runID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, err := h.runs.Get(r.Context(), runID); err != nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	bundle, err := h.runs.EvidenceBundle(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, bundle)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// handleAttentionCenter returns the aggregated attention items across all
// mission runs, grouped by category (NeedsYou, Completed, Failed).
func (h *NexusHandler) handleAttentionCenter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	group, err := h.runs.Attention(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, group)
}
