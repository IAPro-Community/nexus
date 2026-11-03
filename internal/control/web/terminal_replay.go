package web

// sendTerminalReplay first forwards the scrubbed PTY ring history (if any),
// then marks its end. The frontend can synchronize a forced TUI redraw after
// the replay has been accepted by xterm rather than relying on a fixed timer.
// This control event is additive: older Web/Desktop clients safely ignore it.
func sendTerminalReplay(send func(any) error, history string) error {
	if cleaned := scrubProtocolFramesFromOutput(history); cleaned != "" {
		if err := send(TerminalMessage{Type: "output", Data: cleaned}); err != nil {
			return err
		}
	}
	return send(TerminalMessage{Type: "replay_complete"})
}
