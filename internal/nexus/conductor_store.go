package nexus

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// RecordAgentInvocation persists an agent invocation record as event metadata.
// The record is stored under the "agent_invocation" event kind.
func RecordAgentInvocation(st *store.Store, inv AgentInvocation) error {
	data, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("encode agent invocation: %w", err)
	}
	_, err = st.RecordEventMetadata(store.EventMetadata{
		AgentID:   inv.AgentID,
		ProjectID: inv.ProjectID,
		Kind:      "agent_invocation",
		Timestamp: time.Now().UTC(),
		Summary:   string(data),
	})
	return err
}

// PurgeExpiredEvents removes activity events older than retentionDays.
func PurgeExpiredEvents(st *store.Store, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 90
	}
	return st.PurgeExpiredEventsSince(time.Now().UTC(), retentionDays)
}
