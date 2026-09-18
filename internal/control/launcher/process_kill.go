package launcher

import "os"

// KillProcess best-effort SIGKILLs a PID. Used after failed handshake and as
// Stop fallback when the control protocol does not respond.
func KillProcess(pid int) {
	if pid <= 0 {
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Kill()
}
