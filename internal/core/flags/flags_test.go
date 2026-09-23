package flags

import "testing"

func TestKillSwitchDefaultsAndOverrides(t *testing.T) {
	if !Enabled(Tunnel) || !Enabled(AutonomousMission) || !Enabled(RemoteAccess) {
		t.Fatal("risk surfaces must default on for beta")
	}
	if Enabled(ExperimentalIntelligence) {
		t.Fatal("experimental intelligence must default off")
	}
	if Enabled(HUD) {
		t.Fatal("experimental HUD must default off")
	}
	t.Setenv("NEXUS_DISABLE_TUNNEL", "1")
	if Enabled(Tunnel) {
		t.Fatal("disable override must win")
	}
	t.Setenv("NEXUS_ENABLE_EXPERIMENTAL_INTELLIGENCE", "yes")
	if !Enabled(ExperimentalIntelligence) {
		t.Fatal("enable override must win")
	}
	t.Setenv("NEXUS_ENABLE_HUD", "1")
	if !Enabled(HUD) {
		t.Fatal("HUD enable override must win")
	}
	if Enabled("unknown_surface") {
		t.Fatal("unknown flags must default off")
	}
	if len(All()) != 5 {
		t.Fatalf("flag surface must stay small, got %d", len(All()))
	}
}
