package nexus

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// ProjectIntelligenceView is the transport-neutral read model used by REST
// and Web. Readiness remains a separate legacy contract.
type ProjectIntelligenceView struct {
	ProjectID       string                                  `json:"project_id"`
	Identity        contextsnapshot.CodeIdentity            `json:"identity"`
	CurrentSnapshot *contextsnapshot.ProjectContextSnapshot `json:"current_snapshot,omitempty"`
	CurrentScan     *store.ProjectIntelligenceScan          `json:"current_scan,omitempty"`
}

var ErrProjectIntelligencePending = errors.New("project intelligence scan pending")

// RequestProjectIntelligenceScan queues one bounded scan for the current
// identity. Concurrent requests for the same identity reuse the existing
// attempt and do not create duplicate work.
func (n *Nexus) RequestProjectIntelligenceScan(ctx context.Context, projectID string) (*store.ProjectIntelligenceScan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	identity, err := contextsnapshot.InspectCodeIdentity(project.CanonicalPath)
	if err != nil {
		return nil, err
	}
	scans, err := st.ListProjectIntelligenceScans(projectID)
	if err != nil {
		return nil, err
	}
	for _, candidate := range scans {
		if candidate.IdentityDigest != identity.IdentityDigest {
			continue
		}
		switch candidate.State {
		case store.ScanQueued:
			// Re-dispatch stuck QUEUED rows (e.g. OpenProject failed in the worker).
			go n.executeProjectIntelligenceScan(projectID, candidate.ID)
			return &candidate, nil
		case store.ScanRunning, store.ScanSucceeded:
			return &candidate, nil
		case store.ScanFailed, store.ScanCanceled:
			candidate.State = store.ScanQueued
			candidate.Error = ""
			candidate.StartedAt = nil
			candidate.FinishedAt = nil
			candidate.LeaseOwner = ""
			if err := st.UpdateProjectIntelligenceScan(candidate); err != nil {
				return nil, err
			}
			go n.executeProjectIntelligenceScan(projectID, candidate.ID)
			return &candidate, nil
		}
	}
	scan, err := st.CreateProjectIntelligenceScan(store.ProjectIntelligenceScan{
		ID: "pis_" + ids.NewRuntimeID(), ProjectID: projectID,
		IdentityDigest: identity.IdentityDigest, State: store.ScanQueued,
		ScannerVersion: contextsnapshot.ScannerVersion,
	})
	if err != nil {
		// A concurrent creator won the unique identity key. Re-read and reuse
		// (including FAILED → requeue) instead of surfacing a spurious conflict.
		latest, listErr := st.ListProjectIntelligenceScans(projectID)
		if listErr == nil {
			for _, candidate := range latest {
				if candidate.IdentityDigest != identity.IdentityDigest {
					continue
				}
				switch candidate.State {
				case store.ScanFailed, store.ScanCanceled:
					candidate.State = store.ScanQueued
					candidate.Error = ""
					candidate.StartedAt = nil
					candidate.FinishedAt = nil
					candidate.LeaseOwner = ""
					if updateErr := st.UpdateProjectIntelligenceScan(candidate); updateErr == nil {
						go n.executeProjectIntelligenceScan(projectID, candidate.ID)
					}
				case store.ScanQueued:
					go n.executeProjectIntelligenceScan(projectID, candidate.ID)
				}
				return &candidate, nil
			}
		}
		return nil, err
	}
	go n.executeProjectIntelligenceScan(projectID, scan.ID)
	return &scan, nil
}

// executeProjectIntelligenceScan performs static discovery outside the HTTP
// request. It never executes project commands and stores only bounded facts.
func (n *Nexus) executeProjectIntelligenceScan(projectID, scanID string) {
	st, err := n.OpenProject()
	if err != nil {
		return
	}
	scan, err := st.GetProjectIntelligenceScan(scanID)
	if err != nil {
		return
	}
	n.projectIntelligenceMu.Lock()
	if n.projectIntelligenceRunning == nil {
		n.projectIntelligenceRunning = map[string]bool{}
	}
	if n.projectIntelligenceRunning[scanID] {
		n.projectIntelligenceMu.Unlock()
		return
	}
	n.projectIntelligenceRunning[scanID] = true
	n.projectIntelligenceMu.Unlock()
	defer func() {
		n.projectIntelligenceMu.Lock()
		delete(n.projectIntelligenceRunning, scanID)
		n.projectIntelligenceMu.Unlock()
	}()

	now := time.Now().UTC()
	scan.State, scan.StartedAt, scan.LeaseOwner = store.ScanRunning, &now, "nexus"
	_ = st.UpdateProjectIntelligenceScan(scan)
	project, err := st.GetProject(projectID)
	if err != nil {
		scan.State, scan.Error = store.ScanFailed, err.Error()
		_ = st.UpdateProjectIntelligenceScan(scan)
		return
	}
	snapshot, err := contextsnapshot.Discover(project.CanonicalPath, contextsnapshot.Metadata{ProjectID: projectID})
	if err != nil {
		scan.State, scan.Error = store.ScanFailed, err.Error()
		_ = st.UpdateProjectIntelligenceScan(scan)
		return
	}
	if snapshot.Identity.IdentityDigest != scan.IdentityDigest {
		scan.State, scan.Error = store.ScanFailed, "project changed while intelligence scan was running"
		_ = st.UpdateProjectIntelligenceScan(scan)
		return
	}
	if _, err := st.SaveProjectContextSnapshot(snapshot, scanID); err != nil {
		scan.State, scan.Error = store.ScanFailed, fmt.Sprintf("save snapshot: %v", err)
		_ = st.UpdateProjectIntelligenceScan(scan)
		return
	}
	finished := time.Now().UTC()
	scan.State, scan.FinishedAt, scan.LeaseOwner = store.ScanSucceeded, &finished, ""
	_ = st.UpdateProjectIntelligenceScan(scan)
}

// GetProjectIntelligence returns the snapshot matching the current identity,
// plus the latest scan attempt for explainability.
func (n *Nexus) GetProjectIntelligence(ctx context.Context, projectID string) (*ProjectIntelligenceView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	identity, err := contextsnapshot.InspectCodeIdentity(project.CanonicalPath)
	if err != nil {
		return nil, err
	}
	view := &ProjectIntelligenceView{ProjectID: projectID, Identity: identity}
	if snapshot, snapshotErr := st.ProjectContextSnapshotByIdentity(projectID, identity.IdentityDigest); snapshotErr == nil {
		view.CurrentSnapshot = snapshot
	}
	if scans, scansErr := st.ListProjectIntelligenceScans(projectID); scansErr == nil {
		for i := range scans {
			if scans[i].IdentityDigest == identity.IdentityDigest {
				view.CurrentScan = &scans[i]
				break
			}
		}
	}
	return view, nil
}
