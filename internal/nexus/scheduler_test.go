package nexus

import (
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/quota"
)

func schedulerTestAccount(id, provider, profile string) ProviderAccount {
	return ProviderAccount{
		ID: id, Provider: provider, Profile: profile,
		Authenticated: true, Available: true, Health: "healthy",
		QuotaView: &quota.QuotaView{Status: "LIVE"},
	}
}

func TestSchedulerBalancedSelectsHealthy(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "a1", Provider: "claude", Profile: "default", Authenticated: true, Available: true, Health: "healthy", QuotaRemaining: 80, QuotaTotal: 100, QuotaView: &quota.QuotaView{Status: "LIVE"}},
		{ID: "a2", Provider: "openai", Profile: "default", Authenticated: true, Available: true, Health: "degraded", QuotaRemaining: 90, QuotaTotal: 100, QuotaView: &quota.QuotaView{Status: "LIVE"}},
	}
	s := NewResourceScheduler(accounts, PolicyBalanced)
	decision := s.Select("", "", "", nil)
	if decision.Selected.ID != "a1" {
		t.Errorf("expected healthy claude, got %s (score=%.2f)", decision.Selected.ID, decision.Score)
	}
	if decision.Score <= 0 {
		t.Error("score should be positive")
	}
}

func TestSchedulerFiltersUnauthenticated(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "a1", Provider: "claude", Profile: "default", Authenticated: false, Health: "healthy"},
		{ID: "a2", Provider: "openai", Profile: "default", Authenticated: true, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}},
	}
	s := NewResourceScheduler(accounts, PolicyBalanced)
	decision := s.Select("", "", "", nil)
	if decision.Selected.ID != "a2" {
		t.Errorf("expected authenticated openai, got %s", decision.Selected.ID)
	}
}

func TestSchedulerFiltersRateLimited(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "a1", Provider: "claude", Profile: "default", Authenticated: true, Available: true, RateLimited: true, Health: "healthy"},
		{ID: "a2", Provider: "openai", Profile: "default", Authenticated: true, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}},
	}
	s := NewResourceScheduler(accounts, PolicyBalanced)
	decision := s.Select("", "", "", nil)
	if decision.Selected.ID != "a2" {
		t.Errorf("expected non-rate-limited openai, got %s", decision.Selected.ID)
	}
}

func TestSchedulerFiltersCooldown(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	accounts := []ProviderAccount{
		{ID: "a1", Provider: "claude", Profile: "default", Authenticated: true, Available: true, CooldownUntil: &future, Health: "healthy"},
		{ID: "a2", Provider: "openai", Profile: "default", Authenticated: true, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}},
	}
	s := NewResourceScheduler(accounts, PolicyBalanced)
	decision := s.Select("", "", "", nil)
	if decision.Selected.ID != "a2" {
		t.Errorf("expected non-cooldown openai, got %s", decision.Selected.ID)
	}
}

func TestSchedulerFiltersUnhealthy(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "a1", Provider: "claude", Profile: "default", Authenticated: true, Available: true, Health: "unhealthy"},
		{ID: "a2", Provider: "openai", Profile: "default", Authenticated: true, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}},
	}
	s := NewResourceScheduler(accounts, PolicyBalanced)
	decision := s.Select("", "", "", nil)
	if decision.Selected.ID != "a2" {
		t.Errorf("expected healthy openai, got %s", decision.Selected.ID)
	}
}

func TestSchedulerNoEligible(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "a1", Provider: "claude", Authenticated: false},
		{ID: "a2", Provider: "openai", Authenticated: false},
	}
	s := NewResourceScheduler(accounts, PolicyBalanced)
	decision := s.Select("", "", "", nil)
	if decision.Selected.ID != "" {
		t.Errorf("expected no selection, got %s", decision.Selected.ID)
	}
	if decision.Reason == "" {
		t.Error("expected reason for no selection")
	}
}

func TestSchedulerPreserveQuotaPrefersHigherQuota(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "a1", Provider: "claude", Profile: "default", Authenticated: true, Available: true, Health: "healthy", QuotaRemaining: 20, QuotaTotal: 100, QuotaView: &quota.QuotaView{Status: "LIVE"}},
		{ID: "a2", Provider: "openai", Profile: "default", Authenticated: true, Available: true, Health: "healthy", QuotaRemaining: 90, QuotaTotal: 100, QuotaView: &quota.QuotaView{Status: "LIVE"}},
	}
	s := NewResourceScheduler(accounts, PolicyPreserveQuota)
	decision := s.Select("", "", "", nil)
	if decision.Selected.ID != "a2" {
		t.Errorf("expected higher-quota openai, got %s (score=%.2f)", decision.Selected.ID, decision.Score)
	}
}

func TestSchedulerPreferProvider(t *testing.T) {
	accounts := []ProviderAccount{
		schedulerTestAccount("a1", "claude", "default"),
		schedulerTestAccount("a2", "openai", "default"),
	}
	s := NewResourceScheduler(accounts, PolicyBalanced)
	decision := s.Select("openai", "", "", nil)
	if decision.Selected.Provider != "openai" {
		t.Errorf("expected preferred openai, got %s", decision.Selected.Provider)
	}
}

func TestSchedulerManualOnlyMatchesExplicit(t *testing.T) {
	accounts := []ProviderAccount{
		schedulerTestAccount("a1", "claude", "default"),
		schedulerTestAccount("a2", "openai", "default"),
	}
	s := NewResourceScheduler(accounts, PolicyManual)

	// No prefer → no selection.
	decision := s.Select("", "", "", nil)
	if decision.Selected.ID != "" {
		t.Errorf("manual without prefer should select nothing, got %s", decision.Selected.ID)
	}

	// Explicit prefer → matches.
	decision = s.Select("claude", "", "", nil)
	if decision.Selected.Provider != "claude" {
		t.Errorf("manual with prefer=claude should select claude, got %s", decision.Selected.Provider)
	}
}

func TestSchedulerContinuityPrefersSameProvider(t *testing.T) {
	accounts := []ProviderAccount{
		schedulerTestAccount("a1", "claude", "default"),
		{ID: "a2", Provider: "openai", Profile: "default", Authenticated: true, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}},
	}
	s := NewResourceScheduler(accounts, PolicyBalanced)
	decision := s.Select("", "claude", "NATIVE_RESUME_UNVERIFIED", nil)
	if decision.Selected.Provider != "claude" {
		t.Errorf("continuity should prefer same provider claude, got %s", decision.Selected.Provider)
	}
}

func TestSchedulerExplainPath(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "a1", Provider: "claude", Profile: "default", Authenticated: true, Available: true, Health: "healthy"},
	}
	s := NewResourceScheduler(accounts, PolicyBalanced)
	decision := s.Select("", "", "", nil)
	if len(decision.ExplainPath) < 2 {
		t.Errorf("explain path should have at least 2 entries, got %d", len(decision.ExplainPath))
	}
}

func TestSchedulerFiltersUnavailable(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "a1", Provider: "claude", Profile: "default", Authenticated: true, Available: false, Health: "healthy"},
		{ID: "a2", Provider: "openai", Profile: "default", Authenticated: true, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}},
	}
	s := NewResourceScheduler(accounts, PolicyBalanced)
	decision := s.Select("", "", "", nil)
	if decision.Selected.ID != "a2" {
		t.Fatalf("expected available account a2, got %q", decision.Selected.ID)
	}
}

func TestSchedulerRejectsUnknownQuota(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "unknown", Provider: "codex", Profile: "default", Authenticated: true, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "UNKNOWN"}},
	}
	decision := NewResourceScheduler(accounts, PolicyBalanced).Select("", "", "", nil)
	if decision.Selected.ID != "" {
		t.Fatalf("unknown quota must not be selected automatically, got %q", decision.Selected.ID)
	}
}
