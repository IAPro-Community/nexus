package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// DelegationMode is the user-facing policy boundary. It decides whether Nexus
// may create a delegated topology; it never selects an Agent or a resource.
type DelegationMode string

const (
	DelegationAuto DelegationMode = "AUTO"
	DelegationAsk  DelegationMode = "ASK"
	DelegationOff  DelegationMode = "OFF"
)

const (
	DelegationDecisionFactKey = "nexus.delegation_decision"
	DelegationModeFactKey     = "nexus.delegation_mode"
	LeadAgentFactKey          = "nexus.lead_agent_id"
)

// DelegatedWorkstream contains requirements, not an Agent identity. Concrete
// persistent Agents are selected later by MatchAgents during allocation.
type DelegatedWorkstream struct {
	ID                    string   `json:"id"`
	Goal                  string   `json:"goal"`
	PreferredRoles        []string `json:"preferred_roles,omitempty"`
	AcceptableRoles       []string `json:"acceptable_roles,omitempty"`
	RequiredCapabilities  []string `json:"required_capabilities,omitempty"`
	PreferredCapabilities []string `json:"preferred_capabilities,omitempty"`
	Domains               []string `json:"domains,omitempty"`
	Dependencies          []string `json:"dependencies,omitempty"`
	OwnedPaths            []string `json:"owned_paths,omitempty"`
	SharedContracts       []string `json:"shared_contracts,omitempty"`
	ParallelGroup         string   `json:"parallel_group,omitempty"`
}

// DelegationDecision is deterministic policy evidence persisted with the
// existing WorkPlan. Routing/provider/model decisions remain downstream.
type DelegationDecision struct {
	Version              int                   `json:"version"`
	Mode                 DelegationMode        `json:"mode"`
	Delegate             bool                  `json:"delegate"`
	PendingApproval      bool                  `json:"pending_approval,omitempty"`
	Reason               string                `json:"reason"`
	ExpectedBenefit      string                `json:"expected_benefit"`
	Parallelizable       bool                  `json:"parallelizable"`
	CoordinationCost     string                `json:"coordination_cost"`
	ContextTransferCost  string                `json:"context_transfer_cost"`
	WriteCollisionRisk   string                `json:"write_collision_risk"`
	SharedContractRisk   string                `json:"shared_contract_risk"`
	Workstreams          []DelegatedWorkstream `json:"workstreams,omitempty"`
	VerificationStrategy string                `json:"verification_strategy"`
	LeadAgentID          string                `json:"lead_agent_id,omitempty"`
	MaxFanOut            int                   `json:"max_fan_out"`
	CreatedAt            time.Time             `json:"created_at"`
}

// NormalizeDelegationMode is the canonical delegation policy boundary.
// Delegation is opt-in: empty or unknown values fail closed to OFF. AUTO and
// ASK must be explicit so an ordinary provider prompt cannot become a Mission
// merely because a caller omitted the mode.
func NormalizeDelegationMode(mode DelegationMode) DelegationMode {
	switch DelegationMode(strings.ToUpper(strings.TrimSpace(string(mode)))) {
	case DelegationAuto:
		return DelegationAuto
	case DelegationAsk:
		return DelegationAsk
	case DelegationOff:
		return DelegationOff
	default:
		return DelegationOff
	}
}

// NormalizeInteractiveDelegationMode is the terminal-facing policy boundary.
// Interactive provider sessions fail closed to OFF when a legacy/malformed
// runtime has no valid delegation label. AUTO must be an explicit value here;
// this intentionally differs from NormalizeDelegationMode's orchestration
// default.
func NormalizeInteractiveDelegationMode(mode DelegationMode) DelegationMode {
	switch DelegationMode(strings.ToUpper(strings.TrimSpace(string(mode)))) {
	case DelegationAuto:
		return DelegationAuto
	case DelegationAsk:
		return DelegationAsk
	case DelegationOff:
		return DelegationOff
	default:
		return DelegationOff
	}
}

// DecideDelegation applies the dispatch brake before Flow expansion. The
// heuristics are intentionally explainable and provider-independent.
func DecideDelegation(goal string, mode DelegationMode) DelegationDecision {
	goal = strings.TrimSpace(goal)
	mode = NormalizeDelegationMode(mode)
	decision := DelegationDecision{
		Version: 1, Mode: mode, ExpectedBenefit: "LOW", CoordinationCost: "LOW",
		ContextTransferCost: "LOW", WriteCollisionRisk: "LOW", SharedContractRisk: "LOW",
		VerificationStrategy: "lead verifies the integrated Mission result", MaxFanOut: 3,
	}
	if mode == DelegationOff {
		decision.Reason, decision.VerificationStrategy = "delegation mode is OFF", "lead executes and verifies the single WorkUnit"
		return decision
	}
	workstreams := DelegationWorkstreams(goal)
	decision.Workstreams = workstreams
	if len(workstreams) < 2 {
		decision.Reason = "request is atomic or has no independent specialized workstreams"
		return decision
	}
	if hasDelegationCollision(goal) {
		decision.WriteCollisionRisk, decision.SharedContractRisk = "HIGH", "HIGH"
		decision.CoordinationCost, decision.ContextTransferCost = "HIGH", "MEDIUM"
		decision.Reason = "workstreams share an evolving artifact or contract; lead keeps execution serial"
		decision.Workstreams = nil
		return decision
	}
	decision.Delegate = true
	decision.Parallelizable = true
	decision.ExpectedBenefit, decision.CoordinationCost = "HIGH", "MEDIUM"
	decision.ContextTransferCost = "MEDIUM"
	decision.Reason = fmt.Sprintf("request contains %d bounded specialized workstreams with compatible ownership", len(workstreams))
	if mode == DelegationAsk {
		decision.PendingApproval = true
		decision.Reason += "; approval is required before dispatch"
	}
	return decision
}

// DelegationWorkstreams extracts only stable domain signals. It does not
// inspect or select concrete Agents.
func DelegationWorkstreams(goal string) []DelegatedWorkstream {
	text := strings.ToLower(strings.TrimSpace(goal))
	var out []DelegatedWorkstream
	if containsAny(text, "backend", "back-end", "api", "servidor", "server", "golang", "sql", "database", "banco", "auth", "autenticação", "autenticacao") {
		out = append(out, DelegatedWorkstream{ID: "backend", Goal: "Implementar o backend e seus contratos", PreferredRoles: []string{"backend-engineer"}, AcceptableRoles: []string{"backend-engineer", "fullstack-engineer"}, PreferredCapabilities: []string{"go", "api", "sql", "testing"}, Domains: []string{"backend", "api"}, OwnedPaths: []string{"internal/**", "api/**"}, ParallelGroup: "implementation"})
	}
	if containsAny(text, "frontend", "front-end", "react", "ui", "tela", "web", "formulário", "formulario") {
		out = append(out, DelegatedWorkstream{ID: "frontend", Goal: "Implementar a interface frontend e seus contratos", PreferredRoles: []string{"frontend-engineer"}, AcceptableRoles: []string{"frontend-engineer", "fullstack-engineer"}, PreferredCapabilities: []string{"react", "accessibility", "frontend-testing"}, Domains: []string{"frontend", "web"}, OwnedPaths: []string{"web/src/**", "frontend/**"}, ParallelGroup: "implementation"})
	}
	if containsAny(text, "e2e", "end-to-end", "teste", "testes", "qa", "quality", "qualidade", "automated test") {
		deps := []string{}
		for _, item := range out {
			if item.ID == "backend" || item.ID == "frontend" {
				deps = append(deps, item.ID)
			}
		}
		out = append(out, DelegatedWorkstream{ID: "qa", Goal: "Criar e executar testes automatizados/E2E", PreferredRoles: []string{"qa-engineer"}, AcceptableRoles: []string{"qa-engineer", "tester", "reviewer"}, RequiredCapabilities: []string{"testing"}, PreferredCapabilities: []string{"e2e", "accessibility"}, Domains: []string{"quality", "verification"}, Dependencies: deps, OwnedPaths: []string{"tests/**", "e2e/**", "web/e2e/**"}, ParallelGroup: "verification"})
	}
	if len(out) > 3 {
		return out[:3]
	}
	return out
}

func taskRequirementsForWorkstream(workstream DelegatedWorkstream) string {
	req := ClassifyTaskRequirements(workstream.Goal)
	if len(workstream.PreferredRoles) > 0 {
		req.Role = workstream.PreferredRoles[0]
		req.PreferredRoles = append([]string(nil), workstream.PreferredRoles...)
	}
	req.AcceptableRoles = append([]string(nil), workstream.AcceptableRoles...)
	req.RequiredCapabilities = appendUniqueStrings(req.RequiredCapabilities, workstream.RequiredCapabilities...)
	req.PreferredCapabilities = appendUniqueStrings(req.PreferredCapabilities, workstream.PreferredCapabilities...)
	req.Domains = appendUniqueStrings(req.Domains, workstream.Domains...)
	req.RequiresDecomposition = false
	raw, err := json.Marshal(req)
	if err != nil {
		return ""
	}
	return string(raw)
}

func appendUniqueStrings(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	out := make([]string, 0, len(values)+len(additions))
	for _, value := range append(append([]string(nil), values...), additions...) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func hasDelegationCollision(goal string) bool {
	text := strings.ToLower(goal)
	return containsAny(text, "same file", "single file", "one file", "mesmo arquivo", "um único arquivo", "um unico arquivo", "shared contract", "contrato compartilhado", "schema and", "migration")
}

func PersistDelegationDecisionFacts(existing map[string]string, decision DelegationDecision) (map[string]string, error) {
	if decision.CreatedAt.IsZero() {
		decision.CreatedAt = time.Now().UTC()
	}
	raw, err := json.Marshal(decision)
	if err != nil {
		return nil, fmt.Errorf("encode delegation decision: %w", err)
	}
	facts := make(map[string]string, len(existing)+3)
	for key, value := range existing {
		facts[key] = value
	}
	facts[DelegationDecisionFactKey], facts[DelegationModeFactKey] = string(raw), string(decision.Mode)
	if decision.LeadAgentID != "" {
		facts[LeadAgentFactKey] = decision.LeadAgentID
	}
	return facts, nil
}

func ReadDelegationDecisionFacts(facts map[string]string) (DelegationDecision, error) {
	if strings.TrimSpace(facts[DelegationDecisionFactKey]) == "" {
		return DelegationDecision{Mode: NormalizeDelegationMode(DelegationMode(facts[DelegationModeFactKey]))}, nil
	}
	var decision DelegationDecision
	if err := json.Unmarshal([]byte(facts[DelegationDecisionFactKey]), &decision); err != nil {
		return DelegationDecision{}, fmt.Errorf("decode delegation decision: %w", err)
	}
	decision.Mode = NormalizeDelegationMode(decision.Mode)
	return decision, nil
}

func WithLeadAgent(decision DelegationDecision, leadAgentID string) DelegationDecision {
	decision.LeadAgentID = strings.TrimSpace(leadAgentID)
	return decision
}

// ApproveDelegation records the user's ASK-mode approval as a WorkPlan
// revision. Dispatch cannot proceed while PendingApproval is true.
func (n *Nexus) ApproveDelegation(ctx context.Context, planID string) (*store.WorkPlan, error) {
	plan, err := n.GetWorkPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	decision, err := ReadDelegationDecisionFacts(plan.StructuredFacts)
	if err != nil {
		return nil, err
	}
	if !decision.PendingApproval {
		return plan, nil
	}
	decision.PendingApproval = false
	facts, err := PersistDelegationDecisionFacts(plan.StructuredFacts, decision)
	if err != nil {
		return nil, err
	}
	plan.StructuredFacts = facts
	updated, _, err := n.UpdateWorkPlan(ctx, *plan, "Delegation approval recorded")
	return updated, err
}

func ensureDelegationApproved(plan *store.WorkPlan) error {
	if plan == nil {
		return nil
	}
	decision, err := ReadDelegationDecisionFacts(plan.StructuredFacts)
	if err != nil {
		return err
	}
	if decision.PendingApproval {
		return fmt.Errorf("delegation approval is pending for plan %s", plan.ID)
	}
	return nil
}
