package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/handoff"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type maestroRunSubmitBody struct {
	ArtifactID string                          `json:"artifact_id"`
	PlanID     string                          `json:"plan_id"`
	Assignments []maestroRunAssignmentBody     `json:"assignments"`
	ApprovedBy string                          `json:"approved_by,omitempty"`
	MaxCostUSD *float64                        `json:"max_cost_usd,omitempty"`
}

type maestroRunAssignmentBody struct {
	TaskID       string `json:"task_id"`
	ProviderID   string `json:"provider_id"`
	Model        string `json:"model,omitempty"`
	Sandbox      string `json:"sandbox"`
	WorktreePath string `json:"worktree_path"`
}

// handleProjectMaestroRunSubmit is an authenticated, CSRF-protected durable
// admission into the external Maestro queue. Plan content comes exclusively
// from the persisted, project-scoped WorkPlan; artifact content and hash come
// from the finalized Composer session. The ACK is a queued submission
// receipt — never an execution promise — and dispatch remains Maestro-owned
// (`maestro runs worker`). No Nexus fallback scheduler is involved.
func (h *NexusHandler) handleProjectMaestroRunSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required for Maestro run submission")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if strings.TrimSpace(projectID) == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	var body maestroRunSubmitBody
	decoder := json.NewDecoder(io.LimitReader(r.Body, 128*1024+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || body.ArtifactID == "" || body.PlanID == "" ||
		len(body.Assignments) == 0 {
		writeError(w, http.StatusBadRequest, "artifact_id, plan_id and assignments are required")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "exactly one request object is required")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Nexus project store unavailable")
		return
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	artifact, err := st.GetPromptArtifact(body.ArtifactID)
	if err != nil {
		writeError(w, http.StatusNotFound, "PromptArtifact not found")
		return
	}
	session, err := st.GetComposerSession(artifact.SessionID)
	if err != nil || session.ProjectID != projectID || session.State != "FINALIZED" {
		writeError(w, http.StatusConflict, "PromptArtifact must belong to a finalized Composer session in this project")
		return
	}
	plan, err := st.GetWorkPlan(body.PlanID)
	if err != nil || plan.ProjectID != projectID {
		writeError(w, http.StatusNotFound, "WorkPlan not found in this project")
		return
	}
	planTasks, err := maestroPlanTasksFromWorkPlan(plan)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	assignments := make([]handoff.MaestroWorkAssignment, 0, len(body.Assignments))
	seen := make(map[string]bool, len(body.Assignments))
	for _, row := range body.Assignments {
		if row.TaskID == "" || seen[row.TaskID] {
			writeError(w, http.StatusBadRequest, "each task requires exactly one assignment")
			return
		}
		seen[row.TaskID] = true
		assignments = append(assignments, handoff.MaestroWorkAssignment{
			TaskID:       row.TaskID,
			ProviderID:   row.ProviderID,
			Model:        row.Model,
			Sandbox:      row.Sandbox,
			WorktreePath: row.WorktreePath,
		})
	}
	if len(assignments) != len(planTasks) {
		writeError(w, http.StatusConflict, "every plan task requires exactly one explicit assignment")
		return
	}
	ack, err := handoff.SubmitMaestroRun(r.Context(), project.CanonicalPath, handoff.MaestroRunSubmitRequest{
		ArtifactID:       artifact.ID,
		Version:          artifact.Version,
		Content:          artifact.Content,
		SHA256:           artifact.Hash,
		ApprovedModelUse: true,
		Plan:             planTasks,
		Assignments:      assignments,
		ApprovedBy:       body.ApprovedBy,
		MaxCostUSD:       body.MaxCostUSD,
	})
	if err != nil {
		if code := handoff.MaestroDomainCode(err); code != "" {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "Maestro rejected the run submission",
				"code":  code,
			})
			return
		}
		// Do not leak external CLI stderr, local paths or provider diagnostics.
		writeError(w, http.StatusServiceUnavailable, "Maestro run admission unavailable; verify CLI, capabilities and worker deployment")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, ack)
}

type maestroSubmissionCancelBody struct {
	SubmissionID string `json:"submission_id"`
}

// POST /api/v1/projects/{id}/maestro/submissions/cancel requests durable
// cancellation of a Maestro-owned submission. Canceling CLAIMED/DISPATCHED
// work is a request to the Maestro worker, not a fabricated confirmation.
func (h *NexusHandler) handleProjectMaestroSubmissionCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required for Maestro submission cancellation")
		return
	}
	projectID := projectIDFromPath(r.URL.Path)
	if strings.TrimSpace(projectID) == "" {
		writeError(w, http.StatusNotFound, "missing project id")
		return
	}
	var body maestroSubmissionCancelBody
	decoder := json.NewDecoder(io.LimitReader(r.Body, 4096+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || body.SubmissionID == "" {
		writeError(w, http.StatusBadRequest, "submission_id is required")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "exactly one request object is required")
		return
	}
	st, err := h.nexus.OpenProject()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Nexus project store unavailable")
		return
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	row, err := handoff.CancelMaestroSubmission(r.Context(), project.CanonicalPath, body.SubmissionID)
	if err != nil {
		if code := handoff.MaestroDomainCode(err); code != "" {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "Maestro rejected the cancellation request",
				"code":  code,
			})
			return
		}
		writeError(w, http.StatusServiceUnavailable, "Maestro submission cancellation unavailable or incompatible")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, row)
}

// maestroPlanTasksFromWorkPlan projects the persisted WorkPlan packages into
// the bounded Maestro submission plan contract. Routing stays out of the
// plan: provider/model/sandbox/worktree are explicit per-task assignments.
func maestroPlanTasksFromWorkPlan(plan *store.WorkPlan) (json.RawMessage, error) {
	type planTask struct {
		ID                 string   `json:"id"`
		Title              string   `json:"title"`
		Objective          string   `json:"objective"`
		DependsOn          []string `json:"dependsOn"`
		AcceptanceCriteria []string `json:"acceptanceCriteria"`
	}
	tasks := make([]planTask, 0, 32)
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			criteria := make([]string, 0, len(pkg.AcceptanceCriteria))
			for _, criterion := range pkg.AcceptanceCriteria {
				if strings.TrimSpace(criterion) != "" {
					criteria = append(criteria, criterion)
				}
			}
			if strings.TrimSpace(pkg.ID) == "" || strings.TrimSpace(pkg.Title) == "" ||
				strings.TrimSpace(pkg.Goal) == "" || len(criteria) == 0 {
				return nil, fmt.Errorf("plan tasks require id, title, objective and concrete acceptance criteria")
			}
			dependencies := make([]string, 0, len(pkg.Dependencies))
			dependencies = append(dependencies, pkg.Dependencies...)
			tasks = append(tasks, planTask{
				ID:                 pkg.ID,
				Title:              pkg.Title,
				Objective:          pkg.Goal,
				DependsOn:          dependencies,
				AcceptanceCriteria: criteria,
			})
		}
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("plan has no executable tasks")
	}
	payload, err := json.Marshal(map[string]any{"tasks": tasks})
	if err != nil {
		return nil, fmt.Errorf("plan projection failed")
	}
	return payload, nil
}
