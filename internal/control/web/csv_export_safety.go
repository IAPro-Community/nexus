package web

import (
	"strings"
	"unicode"
)

// sanitizeCSVCell neutralizes spreadsheet formulas, including payloads that
// hide a formula trigger after whitespace or Unicode invisible prefixes.
// Preserve the original visible value rather than silently stripping input.
func sanitizeCSVCell(value string) string {
	trimmed := strings.TrimLeftFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || r == '\uFEFF' || r == '\u200B'
	})
	if trimmed == "" {
		return value
	}
	switch []rune(trimmed)[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}
