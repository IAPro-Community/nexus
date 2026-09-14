package nexus

import (
	"context"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type AgentPatch struct {
	Name *string
	Role *string
}

type AgentDetailView struct {
	Agent          store.Agent
	Generations    []store.RuntimeGeneration
	Lineage        []store.LineageEntry
	Revisions      []store.AgentRevision
	EffectiveState string
}

// AgentApplicationService owns durable Agent CRUD and state projection for
// transport adapters. Runtime start/stop remains on Nexus because it spans
// provider drivers and process lifecycle.
type AgentApplicationService struct {
	nexus *Nexus
}

func NewAgentApplicationService(n *Nexus) *AgentApplicationService {
	return &AgentApplicationService{nexus: n}
}

func (s *AgentApplicationService) store(ctx context.Context) (*store.Store, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.nexus.OpenProject()
}

func (s *AgentApplicationService) List(ctx context.Context, projectID string) ([]store.Agent, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	agents, err := st.ListAgents(projectID)
	if err != nil {
		return nil, err
	}
	for i := range agents {
		if state, stateErr := s.nexus.EffectiveAgentState(agents[i].ID); stateErr == nil && state != "" {
			agents[i].Status = state
		}
	}
	return agents, nil
}

func (s *AgentApplicationService) ListSummaries(ctx context.Context) ([]store.ProjectAgentSummary, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	return st.ListAgentSummaries()
}

func (s *AgentApplicationService) Create(ctx context.Context, projectID, name, role string) (store.Agent, error) {
	st, err := s.store(ctx)
	if err != nil {
		return store.Agent{}, err
	}
	return st.CreateAgent(store.Agent{ProjectID: projectID, Name: name, Role: role})
}

func (s *AgentApplicationService) Detail(ctx context.Context, id string) (AgentDetailView, error) {
	st, err := s.store(ctx)
	if err != nil {
		return AgentDetailView{}, err
	}
	agent, err := st.GetAgent(id, "")
	if err != nil {
		return AgentDetailView{}, err
	}
	state, _ := s.nexus.EffectiveAgentState(id)
	if state != "" {
		agent.Status = state
	}
	generations, _ := st.ListGenerations(id)
	lineage, _ := st.ListLineage(id)
	revisions, _ := st.ListRevisions(id)
	return AgentDetailView{Agent: agent, Generations: generations, Lineage: lineage, Revisions: revisions, EffectiveState: state}, nil
}

func (s *AgentApplicationService) Update(ctx context.Context, id string, patch AgentPatch) (store.Agent, error) {
	st, err := s.store(ctx)
	if err != nil {
		return store.Agent{}, err
	}
	agent, err := st.GetAgent(id, "")
	if err != nil {
		return store.Agent{}, err
	}
	if patch.Name != nil {
		agent.Name = *patch.Name
	}
	if patch.Role != nil {
		agent.Role = *patch.Role
	}
	if err := st.UpdateAgent(agent); err != nil {
		return store.Agent{}, err
	}
	return agent, nil
}

func (s *AgentApplicationService) Delete(ctx context.Context, id, projectID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.nexus.DeleteAgent(id, projectID)
}
