package store

import (
	"path/filepath"
	"testing"
)

// Regression: project_intelligence_scans.lease_owner is nullable and INSERT
// leaves it NULL. Scanning it into a plain string fails with:
// sql: Scan error on column index 8, name "lease_owner":
// converting NULL to string is unsupported.
func TestProjectIntelligenceScanNullLeaseOwner(t *testing.T) {
	tmpDir := t.TempDir()
	st, err := Open(filepath.Join(tmpDir, "test_nexus.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	proj, err := st.CreateProject(Project{
		CanonicalPath: tmpDir,
		Name:          "Intel NULL lease test",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	created, err := st.CreateProjectIntelligenceScan(ProjectIntelligenceScan{
		ProjectID:      proj.ID,
		IdentityDigest: "0d5c551323af9fe1009f6ac69054a9af7fb821f403459bb6411a0fadaa463521",
	})
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}

	got, err := st.GetProjectIntelligenceScan(created.ID)
	if err != nil {
		t.Fatalf("get scan with NULL lease_owner: %v", err)
	}
	if got.LeaseOwner != "" {
		t.Fatalf("expected empty lease owner, got %q", got.LeaseOwner)
	}

	list, err := st.ListProjectIntelligenceScans(proj.ID)
	if err != nil {
		t.Fatalf("list scans with NULL lease_owner: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 scan, got %d", len(list))
	}
}
