package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/driver"
)

func TestEffectiveCapabilityMapProjectsReadOnlyReview(t *testing.T) {
	caps := driver.EffectiveCapabilities{
		ReadOnlyReview: driver.CapabilityEvidence{Status: driver.CapabilitySupported},
	}
	got := effectiveCapabilityMap(caps)
	if got["read_only_review"] != string(driver.CapabilitySupported) {
		t.Fatalf("read_only_review=%q, want %q", got["read_only_review"], driver.CapabilitySupported)
	}
}
