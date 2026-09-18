package nexus

import "context"

// ResourceReader is the Core contract used by application-facing resource
// operations. Keeping this boundary small lets transport callers depend on a
// capability instead of the full Nexus aggregate.
type ResourceReader interface {
	ListResources() ([]ProviderAccount, error)
	AllocateResource(context.Context, string, string, string, SchedulerPolicy) (*ResourceAllocation, error)
}

// ContextualResourceReader is an optional extension used by transport-facing
// callers. The legacy ResourceReader contract remains intact for CLI and test
// implementations that do not perform cancellable I/O.
type ContextualResourceReader interface {
	ListResourcesContext(context.Context) ([]ProviderAccount, error)
}

// ResourceApplicationService coordinates resource reads, allocation and
// recommendations for API/CLI consumers. Domain scoring remains in the
// existing recommendation function; this service owns the application flow.
type ResourceApplicationService struct {
	reader ResourceReader
}

func NewResourceApplicationService(reader ResourceReader) *ResourceApplicationService {
	return &ResourceApplicationService{reader: reader}
}

func (s *ResourceApplicationService) List(ctx context.Context) ([]ProviderAccount, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if contextual, ok := s.reader.(ContextualResourceReader); ok {
		return contextual.ListResourcesContext(ctx)
	}
	return s.reader.ListResources()
}

func (s *ResourceApplicationService) Allocate(ctx context.Context, agentID, provider, profile string, policy SchedulerPolicy) (*ResourceAllocation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.reader.AllocateResource(ctx, agentID, provider, profile, policy)
}

func (s *ResourceApplicationService) Recommend(ctx context.Context, accounts []ProviderAccount, req TaskRequirements, policy SchedulerPolicy) (RecommendationResult, error) {
	if err := ctx.Err(); err != nil {
		return RecommendationResult{}, err
	}
	return RecommendResources(accounts, req, policy), nil
}
