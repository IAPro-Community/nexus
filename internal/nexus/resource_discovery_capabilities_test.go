package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/driver"
)

func TestEffectiveCapabilityMapPreservesTestedEvidence(t *testing.T) {
	caps := driver.EffectiveCapabilities{
		Process:          driver.CapabilityEvidence{Status: driver.CapabilitySupported, Tested: true},
		ReadOnlyReview:   driver.CapabilityEvidence{Status: driver.CapabilitySupported},
		StructuredEvents: driver.CapabilityEvidence{Status: driver.CapabilityPartial, Tested: true},
	}
	got := effectiveCapabilityMap(caps)
	want := map[string]string{
		"process":           string(driver.CapabilitySupported),
		"read_only_review":  string(driver.CapabilityNotTested),
		"structured_events": string(driver.CapabilityPartial),
	}
	for key, expected := range want {
		if got[key] != expected {
			t.Errorf("%s=%q, want %q", key, got[key], expected)
		}
	}
}
