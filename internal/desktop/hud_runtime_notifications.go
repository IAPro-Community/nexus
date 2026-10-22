package desktop

import (
	"errors"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/notify"
)

// NotifyRuntimeAttention is invoked solely by the standalone native HUD
// capsule when a NEW canonical runtime attention fingerprint appears.
// It does not read or transmit prompts, credentials or terminal output.
// The frontend deduplicates current waits; the notifier also throttles tags.
func (h *HUDController) NotifyRuntimeAttention(runtimeID, fingerprint string) error {
	runtimeID = strings.TrimSpace(runtimeID)
	fingerprint = strings.TrimSpace(fingerprint)
	if runtimeID == "" || fingerprint == "" || len(runtimeID) > 128 || len(fingerprint) > 128 {
		return errors.New("invalid runtime attention identity")
	}
	return notify.Default().Notify(notify.Payload{
		Title: "Nexus · Intervenção necessária",
		Body:  "Um terminal ou agente aguarda uma resposta. Abra o Nexus Guardian para verificar.",
		Tag:   "hud-attention:" + runtimeID + ":" + fingerprint,
	})
}
