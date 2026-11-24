package nexus

import (
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// newEventMetadataRecorder projects normalized bus events into the durable
// timeline. It records the producer family, the concrete runtime and, when the
// runtime already has a durable generation, the Agent and project that own it.
// Missing identity stays empty instead of being guessed.
func newEventMetadataRecorder(st *store.Store) func(events.Event) {
	return func(e events.Event) {
		if st == nil {
			return
		}
		projectID, _ := e.Data["project_id"].(string)
		agentID, _ := e.Data["agent_id"].(string)
		runtimeID := strings.TrimSpace(e.RuntimeID)
		if runtimeID == "" {
			runtimeID, _ = e.Data["affected_runtime_id"].(string)
		}
		generationID := ""
		if runtimeID != "" {
			if gen, err := st.GenerationByRuntimeID(runtimeID); err == nil {
				generationID = gen.ID
				if agentID == "" {
					agentID = gen.AgentID
				}
			}
		}
		if projectID == "" && agentID != "" {
			if agent, err := st.GetAgent(agentID, ""); err == nil {
				projectID = agent.ProjectID
			}
		}
		source := e.Type.Source()
		if override, ok := e.Data["source"].(string); ok && strings.TrimSpace(override) != "" {
			source = strings.TrimSpace(override)
		}
		_, _ = st.RecordEventMetadata(store.EventMetadata{
			ID:                  e.ID,
			AgentID:             agentID,
			ProjectID:           projectID,
			CorrelationID:       e.CorrelationID,
			Source:              source,
			RuntimeID:           runtimeID,
			RuntimeGenerationID: generationID,
			Kind:                string(e.Type),
			Timestamp:           e.Timestamp,
			Summary:             e.Summary,
			AccountScope:        e.AccountScope,
		})
	}
}
