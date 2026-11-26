package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

func (h *NexusHandler) handleNexActions(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.nexActions == nil {
		writeError(w, http.StatusServiceUnavailable, "Nex actions unavailable")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"catalog":  h.nexActions.Catalog(),
		"receipts": h.nexActions.Receipts(),
	})
}

func (h *NexusHandler) handleNexActionPropose(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.nexActions == nil {
		writeError(w, http.StatusServiceUnavailable, "Nex actions unavailable")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		ActionID string            `json:"action_id"`
		Params   map[string]string `json:"params"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid action payload")
		return
	}
	receipt, err := h.nexActions.Propose(body.ActionID, body.Params, time.Now().UTC())
	switch {
	case errors.Is(err, nexus.ErrNexActionForbidden):
		writeError(w, http.StatusForbidden, "action forbidden")
		return
	case errors.Is(err, nexus.ErrNexActionUnknown):
		writeError(w, http.StatusBadRequest, "unknown action")
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "could not record action")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"receipt": receipt})
}

func (h *NexusHandler) handleNexActionConfirm(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.nexActions == nil {
		writeError(w, http.StatusServiceUnavailable, "Nex actions unavailable")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		ReceiptID string `json:"receipt_id"`
		Approved  bool   `json:"approved"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid confirmation payload")
		return
	}
	ctxReq := r
	receipt, err := h.nexActions.Confirm(body.ReceiptID, body.Approved, h.nexActionExecutor(ctxReq), time.Now().UTC())
	if errors.Is(err, nexus.ErrNexReceiptUnknown) {
		writeError(w, http.StatusNotFound, "receipt not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not complete action")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"receipt": receipt})
}

// nexActionExecutor routes every allowlisted action to its canonical owner.
// It never shells out, starts runners, or falls back to a paid provider.
func (h *NexusHandler) nexActionExecutor(r *http.Request) nexus.NexActionExecutor {
	return func(spec nexus.NexActionSpec, params map[string]string) (string, string) {
		ctx := r.Context()
		switch spec.ID {
		case "read_hud_summary":
			if h.hud == nil {
				return nexus.NexActionStatusUnavailable, "HUD projection unavailable"
			}
			state, err := h.hud.State(ctx, strings.TrimSpace(params["project_id"]))
			if err != nil {
				return nexus.NexActionStatusFailed, "HUD projection failed"
			}
			return nexus.NexActionStatusExecuted, nexus.BuildNexContextSummary(state, "", 400)
		case "read_model_catalog":
			if h.modelWatch == nil {
				return nexus.NexActionStatusUnavailable, "model watch unavailable"
			}
			snap := h.modelWatch.Snapshot(time.Now().UTC())
			return nexus.NexActionStatusExecuted, "freshness=" + snap.Freshness
		case "ack_capacity_alert", "snooze_capacity_alert":
			svc, ok := h.capacityService()
			if !ok {
				return nexus.NexActionStatusUnavailable, "capacity store unavailable"
			}
			fp := strings.TrimSpace(params["fingerprint"])
			if fp == "" {
				return nexus.NexActionStatusFailed, "fingerprint required"
			}
			if spec.ID == "ack_capacity_alert" {
				if _, err := svc.Ack(ctx, fp); err != nil {
					return nexus.NexActionStatusFailed, "ack failed"
				}
				return nexus.NexActionStatusExecuted, "alert acknowledged"
			}
			if _, err := svc.Snooze(ctx, fp, time.Now().UTC().Add(30*time.Minute)); err != nil {
				return nexus.NexActionStatusFailed, "snooze failed"
			}
			return nexus.NexActionStatusExecuted, "alert snoozed 30m"
		case "open_nexus_surface":
			return nexus.NexActionStatusExecuted, "navigation approved: " + strings.TrimSpace(params["target"])
		case "delegate_agent_task":
			return nexus.NexActionStatusUnavailable,
				"delegation must be dispatched from the Agents surface; Nex does not start runners"
		}
		return nexus.NexActionStatusFailed, "unsupported action"
	}
}
