package nexus

import "testing"

func TestMergeSkillCatalogDeduplicatesAndKeepsOperationalCopy(t *testing.T) {
	got := MergeSkillCatalog(
		[]CatalogSkill{{MaestroSkillDesc: MaestroSkillDesc{ID: "shared"}, Availability: SkillSynchronizable, Source: SkillSourceCommunity}},
		[]CatalogSkill{{MaestroSkillDesc: MaestroSkillDesc{ID: "shared"}, Availability: SkillAvailable, Source: SkillSourceCanonical}},
		[]CatalogSkill{{MaestroSkillDesc: MaestroSkillDesc{ID: "task-only"}, Availability: SkillTaskOnly, Source: SkillSourceCodex}},
	)
	if got.Counts.Library != 2 || got.Counts.Operational != 1 || got.Counts.Copies != 3 {
		t.Fatalf("unexpected catalog counts: %+v", got.Counts)
	}
	if len(got.Operational) != 1 || got.Operational[0].ID != "shared" || got.Operational[0].Source != SkillSourceCanonical {
		t.Fatalf("expected canonical operational copy, got %+v", got.Operational)
	}
}
