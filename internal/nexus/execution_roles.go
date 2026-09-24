package nexus

import (
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
)

type ExecutionRoleScope string

const (
	ExecutionRoleTurn    ExecutionRoleScope = "TURN"
	ExecutionRoleRuntime ExecutionRoleScope = "RUNTIME"
)

type AccessIntent string

const (
	AccessWrite      AccessIntent = "WRITE"
	AccessReadOnly   AccessIntent = "READ_ONLY"
	AccessVerifyOnly AccessIntent = "VERIFY_ONLY"
)

// ExecutionRole is a transient operational role layered over a persistent AgentSpec.
// It never changes Agent identity or AgentRevision by itself.
type ExecutionRole struct {
	ID                   string                          `json:"id"`
	Label                string                          `json:"label"`
	BaseAgentModeID      string                          `json:"base_agent_mode_id"`
	AccessIntent         AccessIntent                    `json:"access_intent"`
	RequiredCapabilities []string                        `json:"required_capabilities,omitempty"`
	RecommendedSkillIDs  []string                        `json:"recommended_skill_ids,omitempty"`
	DefaultScope         ExecutionRoleScope              `json:"default_scope"`
	VerificationPolicy   intelligence.VerificationPolicy `json:"verification_policy,omitempty"`
}

var canonicalExecutionRoles = []ExecutionRole{
	{
		ID: "build", Label: "Build", BaseAgentModeID: "developer", AccessIntent: AccessWrite,
		RequiredCapabilities: []string{"implementation"},
		RecommendedSkillIDs:  []string{"coding", "testing", "verification"},
		DefaultScope:         ExecutionRoleRuntime,
		VerificationPolicy:   intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
	},
	{
		ID: "investigate", Label: "Investigate", BaseAgentModeID: "investigator", AccessIntent: AccessReadOnly,
		RequiredCapabilities: []string{"debugging", "root_cause_analysis"},
		RecommendedSkillIDs:  []string{"debugging", "verification"},
		DefaultScope:         ExecutionRoleRuntime,
		VerificationPolicy:   intelligence.VerificationPolicy{RequireEvidence: true},
	},
	{
		ID: "review", Label: "Review", BaseAgentModeID: "reviewer", AccessIntent: AccessReadOnly,
		RequiredCapabilities: []string{"code_review", "read_only_review"},
		RecommendedSkillIDs:  []string{"review", "verification"},
		DefaultScope:         ExecutionRoleRuntime,
		VerificationPolicy:   intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
	},
	{
		ID: "verify", Label: "Verify", BaseAgentModeID: "qa-engineer", AccessIntent: AccessVerifyOnly,
		RequiredCapabilities: []string{"testing", "verification"},
		RecommendedSkillIDs:  []string{"testing", "verification"},
		DefaultScope:         ExecutionRoleRuntime,
		VerificationPolicy:   intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
	},
	{
		ID: "architect", Label: "Architect", BaseAgentModeID: "architect", AccessIntent: AccessReadOnly,
		RequiredCapabilities: []string{"architecture", "system_design"},
		RecommendedSkillIDs:  []string{"architecture", "review"},
		DefaultScope:         ExecutionRoleRuntime,
		VerificationPolicy:   intelligence.VerificationPolicy{RequireEvidence: true},
	},
	{
		ID: "security", Label: "Security", BaseAgentModeID: "security-reviewer", AccessIntent: AccessReadOnly,
		RequiredCapabilities: []string{"security", "threat_modeling"},
		RecommendedSkillIDs:  []string{"security", "review", "verification"},
		DefaultScope:         ExecutionRoleRuntime,
		VerificationPolicy:   intelligence.VerificationPolicy{RequireEvidence: true},
	},
}

func CanonicalExecutionRoles() []ExecutionRole {
	out := make([]ExecutionRole, len(canonicalExecutionRoles))
	for i := range canonicalExecutionRoles {
		out[i] = cloneExecutionRole(canonicalExecutionRoles[i])
	}
	return out
}

func ExecutionRoleForID(id string) (ExecutionRole, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, role := range canonicalExecutionRoles {
		if role.ID == id {
			return cloneExecutionRole(role), true
		}
	}
	return ExecutionRole{}, false
}

func cloneExecutionRole(role ExecutionRole) ExecutionRole {
	role.RequiredCapabilities = append([]string(nil), role.RequiredCapabilities...)
	role.RecommendedSkillIDs = append([]string(nil), role.RecommendedSkillIDs...)
	return role
}

func (r ExecutionRole) IntelligenceContext() *intelligence.ExecutionRoleContext {
	if strings.TrimSpace(r.ID) == "" {
		return nil
	}
	mode, _ := AgentModeForID(r.BaseAgentModeID)
	return &intelligence.ExecutionRoleContext{
		ID:                   r.ID,
		Role:                 mode.Role,
		AccessIntent:         string(r.AccessIntent),
		RequiredCapabilities: append([]string(nil), r.RequiredCapabilities...),
		Scope:                string(r.DefaultScope),
	}
}
