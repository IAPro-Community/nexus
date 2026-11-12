package web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
)

// newTerminalClientIdentity creates a connection ID and a stable pseudonym
// for the authenticated session. The bearer session ID itself is never sent
// to the SessionHost or written to the durable activity log.
func newTerminalClientIdentity(sessionID, agentID string) (protocol.TerminalClientPayload, error) {
	var connection [16]byte
	if _, err := rand.Read(connection[:]); err != nil {
		return protocol.TerminalClientPayload{}, fmt.Errorf("create terminal connection identity: %w", err)
	}
	if sessionID == "" {
		sessionID = "local"
	}
	actor := sha256.Sum256([]byte(agentID + "\x00" + sessionID))
	return protocol.TerminalClientPayload{
		ClientID: "connection-" + hex.EncodeToString(connection[:]),
		ActorID:  "session-" + hex.EncodeToString(actor[:12]),
	}, nil
}
