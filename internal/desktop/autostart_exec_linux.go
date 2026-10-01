//go:build linux

package desktop

import "strings"

// desktopExecArg quotes one argument according to the freedesktop Desktop Entry
// Exec grammar.
func desktopExecArg(value string) string {
	var b strings.Builder
	b.Grow(len(value) + 2)
	b.WriteByte('"')
	for _, r := range value {
		if r == '\\' || r == '"' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
	return b.String()
}
