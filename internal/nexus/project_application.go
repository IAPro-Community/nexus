package nexus

import (
	"context"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// ProjectPatch contains the mutable project fields exposed by the application
// API. Keeping the patch in Core prevents transport packages from owning
// project mutation rules.
type ProjectPatch struct {
	Name             *string
	MaestroMode      *string
	DefaultIsolation *string
	DefaultBranch    *string
	ResourcePolicy   *string
}

// ProjectApplicationService owns project CRUD and layout/event application
// flows shared by Web, CLI, and Desktop adapters.
type ProjectApplicationService struct {
	nexus *Nexus
}

func NewProjectApplicationService(n *Nexus) *ProjectApplicationService {
	return &ProjectApplicationService{nexus: n}
}

func (s *ProjectApplicationService) store(ctx context.Context) (*store.Store, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.nexus.OpenProject()
}

func (s *ProjectApplicationService) List(ctx context.Context) ([]store.Project, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	return st.ListProjects()
}

func (s *ProjectApplicationService) Create(ctx context.Context, name, path string) (store.Project, error) {
	st, err := s.store(ctx)
	if err != nil {
		return store.Project{}, err
	}
	if strings.TrimSpace(path) == "" {
		return store.Project{}, fmt.Errorf("path is required")
	}
	canonical, err := store.CanonicalPath(path)
	if err != nil {
		return store.Project{}, err
	}
	return st.CreateProject(store.Project{Name: name, CanonicalPath: canonical})
}

func (s *ProjectApplicationService) Get(ctx context.Context, id string) (store.Project, store.ProjectLayoutRecord, error) {
	st, err := s.store(ctx)
	if err != nil {
		return store.Project{}, store.ProjectLayoutRecord{}, err
	}
	project, err := st.GetProject(id)
	if err != nil {
		return store.Project{}, store.ProjectLayoutRecord{}, err
	}
	_ = st.TouchProject(id)
	record, _ := st.GetLayoutRecord(id)
	return project, record, nil
}

func (s *ProjectApplicationService) Update(ctx context.Context, id string, patch ProjectPatch) (store.Project, error) {
	st, err := s.store(ctx)
	if err != nil {
		return store.Project{}, err
	}
	project, err := st.GetProject(id)
	if err != nil {
		return store.Project{}, err
	}
	if patch.Name != nil {
		project.Name = *patch.Name
		project.Slug = store.Slugify(*patch.Name)
	}
	if patch.MaestroMode != nil {
		project.MaestroMode = *patch.MaestroMode
	}
	if patch.DefaultIsolation != nil {
		project.DefaultIsolation = *patch.DefaultIsolation
	}
	if patch.DefaultBranch != nil {
		project.DefaultBranch = *patch.DefaultBranch
	}
	if patch.ResourcePolicy != nil {
		project.ResourcePolicy = *patch.ResourcePolicy
	}
	if err := st.UpdateProject(project); err != nil {
		return store.Project{}, err
	}
	return project, nil
}

func (s *ProjectApplicationService) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.nexus.DeleteProject(id)
}

func (s *ProjectApplicationService) Events(ctx context.Context, id, agentID string, limit int) ([]store.EventMetadata, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	return st.ListEventsMetadata(id, agentID, limit)
}

func (s *ProjectApplicationService) Layout(ctx context.Context, id string) (store.ProjectLayoutRecord, error) {
	st, err := s.store(ctx)
	if err != nil {
		return store.ProjectLayoutRecord{}, err
	}
	return st.GetLayoutRecord(id)
}

func (s *ProjectApplicationService) SaveLayout(ctx context.Context, id, layout string, revision int64) (store.ProjectLayoutRecord, error) {
	st, err := s.store(ctx)
	if err != nil {
		return store.ProjectLayoutRecord{}, err
	}
	return st.SaveLayoutWithRevision(id, layout, revision)
}
