package web

import "testing"

func TestSanitizeCSVCellPreventsFormulaExecution(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"profile-a", "profile-a"},
		{"=2+2", "'=2+2"},
		{"+SUM(A1)", "'+SUM(A1)"},
		{"-3+4", "'-3+4"},
		{"@SUM(A1)", "'@SUM(A1)"},
		{"  =HYPERLINK(A1)", "'  =HYPERLINK(A1)"},
		{"\t\r=2+2", "'\t\r=2+2"},
		{"\uFEFF=1+1", "'\uFEFF=1+1"},
		{"\u200B@SUM(A1)", "'\u200B@SUM(A1)"},
		{"'already-escaped", "'already-escaped"},
	}
	for _, tc := range cases {
		if got := sanitizeCSVCell(tc.raw); got != tc.want {
			t.Errorf("sanitizeCSVCell(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}
