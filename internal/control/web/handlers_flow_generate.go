package web

import (
 "encoding/json"
 "errors"
 "io"
 "net/http"
 "strings"

 "github.com/IAPro-Community/nexus/internal/nexus"
)

// handleFlowGenerate creates a reviewable WorkPlan from a complete prompt.
// This endpoint NEVER starts a Mission, selects a runtime or performs work.
// It is distinct from the legacy auto_plan and does not require a Composer.
func (h *NexusHandler) handleFlowGenerate(w http.ResponseWriter, r *http.Request) {
 if r.Method != http.MethodPost {
  writeError(w, http.StatusMethodNotAllowed, "POST required")
  return
 }
 var input struct {
  ProjectID string `json:"project_id"`
  Prompt string `json:"prompt"`
 }
 dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
 if err := dec.Decode(&input); err != nil {
  writeError(w, http.StatusBadRequest, "invalid Flow generation request")
  return
 }
 input.ProjectID = strings.TrimSpace(input.ProjectID)
 input.Prompt = strings.TrimSpace(input.Prompt)
 if input.ProjectID == "" || input.Prompt == "" {
  writeError(w, http.StatusBadRequest, "project_id and prompt are required")
  return
 }
 if len(input.Prompt) > 32*1024 {
  writeError(w, http.StatusRequestEntityTooLarge, "prompt exceeds 32 KiB")
  return
 }
 plan, err := h.nexus.GeneratePlanFromIntent(r.Context(), input.ProjectID, input.Prompt)
 if err != nil {
  var clarification *nexus.ClarificationRequiredError
  if errors.As(err, &clarification) {
   writeJSON(w, http.StatusConflict, map[string]any{
    "error": "clarification_required",
    "message": "Blocking decisions require clarification before Flow planning",
    "clarification": clarification.Checkpoint,
   })
   return
  }
  writeError(w, http.StatusUnprocessableEntity, err.Error())
  return
 }
 writeJSON(w, http.StatusCreated, plan)
}
