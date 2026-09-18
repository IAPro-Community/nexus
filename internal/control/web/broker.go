package web

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/gorilla/websocket"
)

// AgentTerminalBroker manages per-agent WebSocket connections and handles
// runtime generation switches without remounting the browser xterm (Gate 4).
//
// Browser identity is AgentID. The broker observes the current
// RuntimeGeneration and reconnects the IPC transport underneath when the
// generation changes, emitting runtime_changed / agent_state /
// continuity_state frames to all observers.
type AgentTerminalBroker struct {
	mu     sync.RWMutex
	agents map[string]*agentTerminalState // agentID → state
}

type agentTerminalState struct {
	agentID        string
	runtimeID      string // current IPC target
	connections    map[*websocket.Conn]TerminalObserver
	writerConn     *websocket.Conn // single CONTROL writer
	lease          string          // "CONTROL" | "VIEW_ONLY"
	runtimeChanged chan struct{}   // closed + recreated on generation switch
}

// TerminalObserver represents one browser tab connected to an agent's terminal.
type TerminalObserver struct {
	conn    *websocket.Conn
	role    string // "CONTROL" | "VIEW_ONLY"
	created time.Time
	writeMu *sync.Mutex
}

func writeTerminal(conn *websocket.Conn, mu *sync.Mutex, payload []byte) {
	if conn == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	_ = conn.WriteMessage(websocket.TextMessage, payload)
}

var defaultBroker = &AgentTerminalBroker{
	agents: make(map[string]*agentTerminalState),
}

// DefaultBroker returns the global terminal broker singleton.
func DefaultBroker() *AgentTerminalBroker {
	return defaultBroker
}

// Attach registers a new WebSocket connection for the given agent. If the
// agent already has a runtime, it receives bounded history replay. Returns
// the assigned lease role. writeMu must be the sole writer mutex for conn.
func (b *AgentTerminalBroker) Attach(agentID string, conn *websocket.Conn, hasRuntime bool, writeMu *sync.Mutex) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	state, ok := b.agents[agentID]
	if !ok {
		state = &agentTerminalState{
			agentID:        agentID,
			connections:    make(map[*websocket.Conn]TerminalObserver),
			runtimeChanged: make(chan struct{}),
		}
		b.agents[agentID] = state
	}
	if state.runtimeChanged == nil {
		state.runtimeChanged = make(chan struct{})
	}

	role := "VIEW_ONLY"
	if state.writerConn == nil && hasRuntime {
		role = "CONTROL"
		state.writerConn = conn
	}

	state.connections[conn] = TerminalObserver{
		conn:    conn,
		role:    role,
		created: time.Now(),
		writeMu: writeMu,
	}

	return role
}

// Detach removes a WebSocket connection and transfers writer lease if needed.
func (b *AgentTerminalBroker) Detach(agentID string, conn *websocket.Conn) {
	b.mu.Lock()

	state, ok := b.agents[agentID]
	if !ok {
		b.mu.Unlock()
		return
	}

	delete(state.connections, conn)

	var transfer TerminalObserver
	if state.writerConn == conn {
		state.writerConn = nil
		for _, obs := range state.connections {
			state.writerConn = obs.conn
			state.lease = "CONTROL"
			transfer = obs
			break
		}
	}

	if len(state.connections) == 0 {
		delete(b.agents, agentID)
	}

	b.mu.Unlock()

	if transfer.conn != nil {
		leaseMsg, _ := json.Marshal(map[string]string{
			"type": "lease",
			"role": "CONTROL",
		})
		writeTerminal(transfer.conn, transfer.writeMu, leaseMsg)
	}
}

// TakeControl steals the writer lease from the current writer.
func (b *AgentTerminalBroker) TakeControl(agentID string, conn *websocket.Conn) bool {
	b.mu.Lock()

	state, ok := b.agents[agentID]
	if !ok {
		b.mu.Unlock()
		return false
	}

	if state.writerConn == conn {
		b.mu.Unlock()
		return true // already the writer
	}

	var revoke TerminalObserver
	if state.writerConn != nil {
		if obs, ok := state.connections[state.writerConn]; ok {
			revoke = obs
		}
	}

	state.writerConn = conn
	state.lease = "CONTROL"
	b.mu.Unlock()

	if revoke.conn != nil {
		revokeMsg, _ := json.Marshal(map[string]string{
			"type": "lease",
			"role": "VIEW_ONLY",
		})
		writeTerminal(revoke.conn, revoke.writeMu, revokeMsg)
	}
	return true
}

// ReleaseControl voluntarily releases the writer lease.
func (b *AgentTerminalBroker) ReleaseControl(agentID string, conn *websocket.Conn) {
	b.mu.Lock()

	state, ok := b.agents[agentID]
	if !ok || state.writerConn != conn {
		b.mu.Unlock()
		return
	}

	state.writerConn = nil
	var transfer TerminalObserver
	for c, obs := range state.connections {
		if c == conn {
			continue
		}
		state.writerConn = obs.conn
		transfer = obs
		break
	}
	b.mu.Unlock()

	if transfer.conn != nil {
		leaseMsg, _ := json.Marshal(map[string]string{
			"type": "lease",
			"role": "CONTROL",
		})
		writeTerminal(transfer.conn, transfer.writeMu, leaseMsg)
	}
}

// IsWriter returns true if the given WebSocket connection currently holds the
// authoritative CONTROL lease for the agent (§45, A6 single writer authority).
func (b *AgentTerminalBroker) IsWriter(agentID string, conn *websocket.Conn) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	state, ok := b.agents[agentID]
	if !ok {
		return false
	}
	return state.writerConn == conn
}

// Writer returns the current authoritative writer connection for the agent.
func (b *AgentTerminalBroker) Writer(agentID string) *websocket.Conn {
	b.mu.RLock()
	defer b.mu.RUnlock()

	state, ok := b.agents[agentID]
	if !ok {
		return nil
	}
	return state.writerConn
}

// WatchRuntimeChanged returns a channel that is closed when the runtime
// generation changes for the given agent. Returns nil if agent has no state.
func (b *AgentTerminalBroker) WatchRuntimeChanged(agentID string) <-chan struct{} {
	b.mu.Lock()
	defer b.mu.Unlock()

	state, ok := b.agents[agentID]
	if !ok {
		return nil
	}
	if state.runtimeChanged == nil {
		state.runtimeChanged = make(chan struct{})
	}
	return state.runtimeChanged
}

// CurrentRuntimeID returns the current runtime ID for an agent, or empty string.
func (b *AgentTerminalBroker) CurrentRuntimeID(agentID string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	state, ok := b.agents[agentID]
	if !ok {
		return ""
	}
	return state.runtimeID
}

// NotifyRuntimeChanged broadcasts a runtime_changed frame to all observers
// of the given agent. Called by the Nexus layer when a generation switch occurs.
func (b *AgentTerminalBroker) NotifyRuntimeChanged(agentID string, oldRuntimeID, newRuntimeID, provider, profile, continuity string) {
	b.mu.Lock()

	state, ok := b.agents[agentID]
	if !ok {
		b.mu.Unlock()
		return
	}

	state.runtimeID = newRuntimeID

	if state.runtimeChanged != nil {
		close(state.runtimeChanged)
	}
	state.runtimeChanged = make(chan struct{})

	payload := protocol.RuntimeChangedPayload{
		AgentID:      agentID,
		OldRuntimeID: oldRuntimeID,
		NewRuntimeID: newRuntimeID,
		Provider:     provider,
		Profile:      profile,
		Continuity:   continuity,
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := TerminalMessage{
		Type: "runtime_changed",
		Data: string(payloadBytes),
	}
	msgBytes, _ := json.Marshal(msg)

	observers := make([]TerminalObserver, 0, len(state.connections))
	for _, obs := range state.connections {
		observers = append(observers, obs)
	}
	b.mu.Unlock()
	for _, obs := range observers {
		writeTerminal(obs.conn, obs.writeMu, msgBytes)
	}
}

// NotifyAgentState broadcasts agent state to all observers.
func (b *AgentTerminalBroker) NotifyAgentState(agentID, agentState string) {
	b.mu.RLock()

	ats, ok := b.agents[agentID]
	if !ok {
		b.mu.RUnlock()
		return
	}

	payload := protocol.AgentStatePayload{
		AgentID: agentID,
		State:   agentState,
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := TerminalMessage{
		Type: "agent_state",
		Data: string(payloadBytes),
	}
	msgBytes, _ := json.Marshal(msg)

	observers := make([]TerminalObserver, 0, len(ats.connections))
	for _, obs := range ats.connections {
		observers = append(observers, obs)
	}
	b.mu.RUnlock()
	for _, obs := range observers {
		writeTerminal(obs.conn, obs.writeMu, msgBytes)
	}
}

// NotifyContinuity broadcasts continuity status to all observers.
func (b *AgentTerminalBroker) NotifyContinuity(agentID, continuity string) {
	b.mu.RLock()

	ats, ok := b.agents[agentID]
	if !ok {
		b.mu.RUnlock()
		return
	}

	payload := protocol.ContinuityStatePayload{
		AgentID:    agentID,
		Continuity: continuity,
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := TerminalMessage{
		Type: "continuity_state",
		Data: string(payloadBytes),
	}
	msgBytes, _ := json.Marshal(msg)

	observers := make([]TerminalObserver, 0, len(ats.connections))
	for _, obs := range ats.connections {
		observers = append(observers, obs)
	}
	b.mu.RUnlock()
	for _, obs := range observers {
		writeTerminal(obs.conn, obs.writeMu, msgBytes)
	}
}

// ObserverCount returns the number of connected observers for an agent.
func (b *AgentTerminalBroker) ObserverCount(agentID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if state, ok := b.agents[agentID]; ok {
		return len(state.connections)
	}
	return 0
}
