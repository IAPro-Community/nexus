// Package agentprojection defines provider-neutral contracts for materializing
// Nexus execution roles into host-native agent definitions. The package lives
// below internal/nexus intentionally: provider projection must not own Agent
// identity, scheduling, Mission state, or durable orchestration.
package agentprojection

import (
	"context"
	"io/fs"
	"time"
)

type Scope string

const (
	ScopeNexusProfile Scope = "NEXUS_PROFILE"
	ScopeProject      Scope = "PROJECT"
	ScopeUser         Scope = "USER"
)

type ActivationStrategy string

const (
	ActivationNativeHotSwitch ActivationStrategy = "NATIVE_HOT_SWITCH"
	ActivationNativeOnLaunch  ActivationStrategy = "NATIVE_ON_LAUNCH"
	ActivationNativeSubagent  ActivationStrategy = "NATIVE_SUBAGENT"
	ActivationNexusOverlay    ActivationStrategy = "NEXUS_OVERLAY"
	ActivationUnsupported     ActivationStrategy = "UNSUPPORTED"
)

type CapabilityStatus string

const (
	CapabilitySupported   CapabilityStatus = "SUPPORTED"
	CapabilityPartial     CapabilityStatus = "PARTIAL"
	CapabilityUnsupported CapabilityStatus = "UNSUPPORTED"
	CapabilityUnknown     CapabilityStatus = "UNKNOWN"
	CapabilityNotTested   CapabilityStatus = "NOT_TESTED"
)

type CapabilityEvidence struct {
	Status      CapabilityStatus `json:"status"`
	Tested      bool             `json:"tested"`
	Evidence    string           `json:"evidence,omitempty"`
	ProviderVer string           `json:"provider_version,omitempty"`
	VerifiedAt  *time.Time       `json:"verified_at,omitempty"`
}

type Capabilities struct {
	Definitions     CapabilityEvidence `json:"definitions"`
	PrimaryAgent    CapabilityEvidence `json:"primary_agent"`
	Subagents       CapabilityEvidence `json:"subagents"`
	InSessionSwitch CapabilityEvidence `json:"in_session_switch"`
	DynamicReload   CapabilityEvidence `json:"dynamic_reload"`
	ToolPermissions CapabilityEvidence `json:"tool_permissions"`
	ModelOverride   CapabilityEvidence `json:"model_override"`
	ProjectScope    CapabilityEvidence `json:"project_scope"`
	UserScope       CapabilityEvidence `json:"user_scope"`
	RemoteProtocol  CapabilityEvidence `json:"remote_protocol"`
}

type Request struct {
	Provider         string   `json:"provider"`
	Profile          string   `json:"profile"`
	ProviderHome     string   `json:"provider_home"`
	Workspace        string   `json:"workspace,omitempty"`
	Scope            Scope    `json:"scope"`
	RoleID           string   `json:"role_id"`
	Role             string   `json:"role"`
	AccessIntent     string   `json:"access_intent"`
	Instructions     []string `json:"instructions,omitempty"`
	Responsibilities []string `json:"responsibilities,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	Constraints      []string `json:"constraints,omitempty"`
	Model            string   `json:"model,omitempty"`
	SourceHash       string   `json:"source_hash"`
}

type GeneratedArtifact struct {
	Path        string      `json:"path"`
	Content     []byte      `json:"-"`
	Mode        fs.FileMode `json:"mode"`
	ContentHash string      `json:"content_hash"`
}

type ConflictKind string

const (
	ConflictManagedModified ConflictKind = "MANAGED_MODIFIED"
	ConflictUnmanaged       ConflictKind = "UNMANAGED"
	ConflictUnsafePath      ConflictKind = "UNSAFE_PATH"
)

type Conflict struct {
	Path   string       `json:"path"`
	Kind   ConflictKind `json:"kind"`
	Detail string       `json:"detail,omitempty"`
}

type Plan struct {
	Provider  string              `json:"provider"`
	Scope     Scope               `json:"scope"`
	Strategy  ActivationStrategy  `json:"strategy"`
	Artifacts []GeneratedArtifact `json:"artifacts,omitempty"`
	Conflicts []Conflict          `json:"conflicts,omitempty"`
	Warnings  []string            `json:"warnings,omitempty"`
}

type ArtifactReceipt struct {
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
}

type Receipt struct {
	ID              string             `json:"id"`
	Provider        string             `json:"provider"`
	Profile         string             `json:"profile"`
	RoleID          string             `json:"role_id"`
	Scope           Scope              `json:"scope"`
	Strategy        ActivationStrategy `json:"strategy"`
	RendererVersion string             `json:"renderer_version"`
	SourceHash      string             `json:"source_hash"`
	Artifacts       []ArtifactReceipt  `json:"artifacts"`
	AppliedAt       time.Time          `json:"applied_at"`
	VerifiedAt      *time.Time         `json:"verified_at,omitempty"`
}

type Verification struct {
	Verified bool     `json:"verified"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type Adapter interface {
	ProviderID() string
	Capabilities(context.Context, string) Capabilities
	Plan(context.Context, Request) (*Plan, error)
	Render(context.Context, Request) ([]GeneratedArtifact, error)
	Apply(context.Context, *Plan) (*Receipt, error)
	Verify(context.Context, *Receipt) (*Verification, error)
	Remove(context.Context, *Receipt) error
}
