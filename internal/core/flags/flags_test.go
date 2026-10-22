package flags

import "testing"

func TestKillSwitchDefaultsAndOverrides(t *testing.T) {
	if !Enabled(Tunnel) || !Enabled(AutonomousMission) || !Enabled(RemoteAccess) {
		t.Fatal("risk surfaces must default on for beta")
	}
	if Enabled(ExperimentalIntelligence) {
		t.Fatal("experimental intelligence must default off")
	}
	if !Enabled(HUD) {
		t.Fatal("read-only HUD projection must be available by default for native users")
	}
	for _, flag := range []Name{
		ExecutionWorkspacesV2,
		ReviewCenter,
		BrowserDesignMode,
		CandidateRuns,
		RemoteExecution,
		NexusNodes,
		ComputerUse,
		MobileCompanion,
	} {
		if Enabled(flag) {
			t.Fatalf("Nexus 1.1 feature %q must default off", flag)
		}
	}
	t.Setenv("NEXUS_DISABLE_TUNNEL", "1")
	if Enabled(Tunnel) {
		t.Fatal("disable override must win")
	}
	t.Setenv("NEXUS_ENABLE_EXPERIMENTAL_INTELLIGENCE", "yes")
	if !Enabled(ExperimentalIntelligence) {
		t.Fatal("enable override must win")
	}
	t.Setenv("NEXUS_DISABLE_HUD", "1")
	if Enabled(HUD) {
		t.Fatal("HUD disable kill switch must win")
	}
	t.Setenv("NEXUS_DISABLE_HUD", "")
	t.Setenv("NEXUS_ENABLE_HUD", "1")
	if !Enabled(HUD) {
		t.Fatal("HUD enable override must win")
	}
	if Enabled("unknown_surface") {
		t.Fatal("unknown flags must default off")
	}
	t.Setenv("NEXUS_ENABLE_COMPUTER_USE", "true")
	if !Enabled(ComputerUse) {
		t.Fatal("computer-use enable override must win")
	}
	if len(All()) != 13 {
		t.Fatalf("unexpected flag surface, got %d", len(All()))
	}
}
