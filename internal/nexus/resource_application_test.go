package nexus

import (
	"context"
	"errors"
	"testing"
)

type resourceReaderStub struct {
	accounts []ProviderAccount
	listed   bool
}

type contextualResourceReaderStub struct {
	resourceReaderStub
	ctx context.Context
}

func (s *contextualResourceReaderStub) ListResourcesContext(ctx context.Context) ([]ProviderAccount, error) {
	s.ctx = ctx
	return s.accounts, nil
}

func (s *resourceReaderStub) ListResources() ([]ProviderAccount, error) {
	s.listed = true
	return s.accounts, nil
}

func (s *resourceReaderStub) AllocateResource(context.Context, string, string, string, SchedulerPolicy) (*ResourceAllocation, error) {
	return &ResourceAllocation{Persisted: true}, nil
}

func TestResourceApplicationServiceHonorsCancelledContext(t *testing.T) {
	stub := &resourceReaderStub{}
	service := NewResourceApplicationService(stub)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := service.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("list error = %v, want context canceled", err)
	}
	if stub.listed {
		t.Fatal("canceled request reached the resource reader")
	}
}

func TestResourceApplicationServiceDelegatesRecommendation(t *testing.T) {
	service := NewResourceApplicationService(&resourceReaderStub{})
	result, err := service.Recommend(context.Background(), []ProviderAccount{{Provider: "codex", Profile: "work"}}, TaskRequirements{}, PolicyBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].Account.Provider != "codex" {
		t.Fatalf("unexpected recommendation: %#v", result)
	}
}

func TestResourceApplicationServicePropagatesContextToContextualReader(t *testing.T) {
	stub := &contextualResourceReaderStub{}
	service := NewResourceApplicationService(stub)
	ctx := context.WithValue(context.Background(), struct{ name string }{}, "resource-test")

	if _, err := service.List(ctx); err != nil {
		t.Fatal(err)
	}
	if stub.ctx != ctx {
		t.Fatal("resource context was not propagated to the contextual reader")
	}
}

func TestListResourcesContextHonorsCancellationBeforeProviderDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var service Nexus
	if _, err := service.ListResourcesContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("list resources error = %v, want context canceled", err)
	}
}
