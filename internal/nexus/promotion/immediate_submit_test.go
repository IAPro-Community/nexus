package promotion

import "testing"

func TestIsImmediateSubmitLineProviderSlash(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"/status", true},
		{"  /model gpt  ", true},
		{":help", true},
		{"/nexus status", true},
		{":nexus help", true},
		{"implemente o backend", false},
		{"", true},
		{"   ", true},
	}
	for _, tc := range cases {
		if got := IsImmediateSubmitLine(tc.line); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.line, got, tc.want)
		}
	}
}
