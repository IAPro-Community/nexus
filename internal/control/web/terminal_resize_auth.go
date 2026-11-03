package web

// Only the active terminal writer is allowed to resize the shared PTY.
// A VIEW_ONLY observer has its own viewport, but must not change the
// dimensions reported to the interactive process owned by another client.
func validTerminalResizeRequest(isWriter bool, rows, cols int) bool {
	const maxPTYDimension = 65535 // Unix PTY winsize uses uint16.
	return isWriter && rows > 0 && cols > 0 && rows <= maxPTYDimension && cols <= maxPTYDimension
}
