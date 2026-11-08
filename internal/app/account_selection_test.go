package app

import "testing"

func TestResolveProfileSelection(t *testing.T) {
	tests := []struct {
		input       string
		profileName string
		balanced    bool
	}{
		{input: "work", profileName: "work"},
		{input: " auto ", balanced: true},
		{input: "balance", balanced: true},
		{input: "BALANCED", balanced: true},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			profileName, balanced := resolveProfileSelection(test.input)
			if profileName != test.profileName || balanced != test.balanced {
				t.Fatalf("resolveProfileSelection(%q) = (%q, %t), want (%q, %t)", test.input, profileName, balanced, test.profileName, test.balanced)
			}
		})
	}
}
