package nexus

import (
	"testing"
)

func TestIsProfileQuotaExhaustedOnUnknownProfile(t *testing.T) {
	// An unknown profile has no cached snapshot and must not be reported as exhausted
	if isProfileQuotaExhausted("nonexistent_provider", "nonexistent_profile") {
		t.Fatal("expected non-existent profile not to be marked exhausted")
	}
}
