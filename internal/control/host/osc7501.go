package host

import (
	"encoding/base64"
	"strings"
	"unicode"
)

// oscStatus models the state signal, not a remote approval API.
type oscStatus struct{ State, Kind, Message, App string }

// extractOSC7501 scans a bounded PTY tail for complete program-status
// sequences. It ignores capability probes, unknown states and malformed data.
// The caller retains a short tail across ProcessChunk boundaries.
func extractOSC7501(raw string) (oscStatus, bool) {
	var last oscStatus
	found := false
	const prefix = "\x1b]7501;"
	for offset := 0; offset < len(raw); {
		at := strings.Index(raw[offset:], prefix)
		if at < 0 {
			break
		}
		at += offset + len(prefix)
		end := strings.IndexByte(raw[at:], '\a')
		size := 1
		if end < 0 {
			end = strings.Index(raw[at:], "\x1b\\")
			size = 2
		}
		if end < 0 {
			break
		}
		fields := strings.Split(raw[at:at+end], ":")
		s := oscStatus{}
		for _, field := range fields {
			key, value, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			switch key {
			case "state":
				s.State = value
			case "kind":
				s.Kind = value
			case "app":
				s.App = value
			case "msg":
				if len(value) > 2048 {
					continue
				}
				if decoded, err := base64.StdEncoding.DecodeString(value); err == nil && len(decoded) <= 1024 {
					s.Message = strings.Map(func(r rune) rune {
						if unicode.IsControl(r) {
							return -1
						}
						return r
					}, string(decoded))
				}
			}
		}
		switch s.State {
		case "working", "idle", "done", "error":
			found = true
			last = s
		case "blocked":
			if s.Kind == "question" || s.Kind == "permission" || s.Kind == "auth" {
				found = true
				last = s
			}
		}
		offset = at + end + size
	}
	return last, found
}
