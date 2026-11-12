package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/driver"
)

func TestIntelligenceCLICapableRequiresTestedSupport(t *testing.T) {
	supported := driver.CapabilityEvidence{Status: driver.CapabilitySupported, Tested: true}
	cases := []struct {
		name string
		caps driver.EffectiveCapabilities
		want bool
	}{
		{
			name: "both capabilities supported and tested",
			caps: driver.EffectiveCapabilities{Headless: supported, SubmitPrompt: supported},
			want: true,
		},
		{
			name: "headless status is not tested",
			caps: driver.EffectiveCapabilities{
				Headless:     driver.CapabilityEvidence{Status: driver.CapabilitySupported},
				SubmitPrompt: supported,
			},
		},
		{
			name: "prompt submission status is not tested",
			caps: driver.EffectiveCapabilities{
				Headless:     supported,
				SubmitPrompt: driver.CapabilityEvidence{Status: driver.CapabilitySupported},
			},
		},
		{
			name: "partial support is not enough",
			caps: driver.EffectiveCapabilities{
				Headless:     supported,
				SubmitPrompt: driver.CapabilityEvidence{Status: driver.CapabilityPartial, Tested: true},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := intelligenceCLICapable(tc.caps); got != tc.want {
				t.Fatalf("intelligenceCLICapable()=%t, want %t", got, tc.want)
			}
		})
	}
}
