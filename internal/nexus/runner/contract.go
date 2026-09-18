package runner

// AutonomyContract defines operational boundaries, verification requirements,
// and bounded remediation for autonomous local engineering work.
type AutonomyContract struct {
	MaxRetries                 int                     `json:"max_retries"`
	MaxTotalIterations         int                     `json:"max_total_iterations"`
	MaxNoProgress              int                     `json:"max_no_progress"`
	StallTimeoutSeconds        int                     `json:"stall_timeout_seconds,omitempty"`
	PackageTimeoutSeconds      int                     `json:"package_timeout_seconds"`
	AutoRemediate              bool                    `json:"auto_remediate"`
	RequireVerification        bool                    `json:"require_verification"`
	DisallowDestructiveGit     bool                    `json:"disallow_destructive_git"`
	AllowedFilePatterns        []string                `json:"allowed_file_patterns,omitempty"`
	VerificationCommands       []string                `json:"verification_commands,omitempty"`
	GlobalVerificationCommands []string                `json:"global_verification_commands,omitempty"`
	EscalateOnFailure          bool                    `json:"escalate_on_failure"`
	AllowToolAutoApproval      bool                    `json:"allow_tool_auto_approval"`
	AllowGitPush               bool                    `json:"allow_git_push"`
	AllowDeploy                bool                    `json:"allow_deploy"`
	AllowExternalNetwork       bool                    `json:"allow_external_network"`
	AllowSecretAccess          bool                    `json:"allow_secret_access"`
	AllowPaidServices          bool                    `json:"allow_paid_services"`
	AllowedHumanOperations     []InterventionOperation `json:"allowed_human_operations,omitempty"`
}

type AutonomyContractPatch struct {
	MaxRetries             *int  `json:"max_retries,omitempty"`
	MaxTotalIterations     *int  `json:"max_total_iterations,omitempty"`
	MaxNoProgress          *int  `json:"max_no_progress,omitempty"`
	PackageTimeoutSeconds  *int  `json:"package_timeout_seconds,omitempty"`
	AutoRemediate          *bool `json:"auto_remediate,omitempty"`
	RequireVerification    *bool `json:"require_verification,omitempty"`
	DisallowDestructiveGit *bool `json:"disallow_destructive_git,omitempty"`
	EscalateOnFailure      *bool `json:"escalate_on_failure,omitempty"`
	AllowToolAutoApproval  *bool `json:"allow_tool_auto_approval,omitempty"`
	AllowGitPush           *bool `json:"allow_git_push,omitempty"`
	AllowDeploy            *bool `json:"allow_deploy,omitempty"`
	AllowExternalNetwork   *bool `json:"allow_external_network,omitempty"`
	AllowSecretAccess      *bool `json:"allow_secret_access,omitempty"`
	AllowPaidServices      *bool `json:"allow_paid_services,omitempty"`
}

func ApplyAutonomyContractPatch(p *AutonomyContractPatch) AutonomyContract {
	c := DefaultAutonomyContract()
	if p == nil {
		return c
	}
	if p.MaxRetries != nil {
		c.MaxRetries = *p.MaxRetries
	}
	if p.MaxTotalIterations != nil {
		c.MaxTotalIterations = *p.MaxTotalIterations
	}
	if p.MaxNoProgress != nil {
		c.MaxNoProgress = *p.MaxNoProgress
	}
	if p.PackageTimeoutSeconds != nil {
		c.PackageTimeoutSeconds = *p.PackageTimeoutSeconds
	}
	if p.AutoRemediate != nil {
		c.AutoRemediate = *p.AutoRemediate
	}
	if p.RequireVerification != nil {
		c.RequireVerification = *p.RequireVerification
	}
	if p.DisallowDestructiveGit != nil {
		c.DisallowDestructiveGit = *p.DisallowDestructiveGit
	}
	if p.EscalateOnFailure != nil {
		c.EscalateOnFailure = *p.EscalateOnFailure
	}
	if p.AllowToolAutoApproval != nil {
		c.AllowToolAutoApproval = *p.AllowToolAutoApproval
	}
	if p.AllowGitPush != nil {
		c.AllowGitPush = *p.AllowGitPush
	}
	if p.AllowDeploy != nil {
		c.AllowDeploy = *p.AllowDeploy
	}
	if p.AllowExternalNetwork != nil {
		c.AllowExternalNetwork = *p.AllowExternalNetwork
	}
	if p.AllowSecretAccess != nil {
		c.AllowSecretAccess = *p.AllowSecretAccess
	}
	if p.AllowPaidServices != nil {
		c.AllowPaidServices = *p.AllowPaidServices
	}
	return c
}

// DefaultAutonomyContract is intentionally local-first: it allows coding tools
// inside isolated workspaces but never authorizes push/deploy. Verification
// commands are detected from the target repository at Mission creation.
func DefaultAutonomyContract() AutonomyContract {
	return AutonomyContract{
		MaxRetries:                 3,
		MaxTotalIterations:         120,
		MaxNoProgress:              2,
		StallTimeoutSeconds:        900,
		PackageTimeoutSeconds:      3600,
		AutoRemediate:              true,
		RequireVerification:        true,
		DisallowDestructiveGit:     true,
		VerificationCommands:       nil,
		GlobalVerificationCommands: nil,
		EscalateOnFailure:          true,
		AllowToolAutoApproval:      true,
		AllowGitPush:               false,
		AllowDeploy:                false,
		AllowedHumanOperations: []InterventionOperation{
			InterventionRetrySafePackage,
			InterventionReplanPackage,
			InterventionConfirmExternalOutcome,
		},
	}
}

func (c AutonomyContract) AllowsHumanOperation(operation InterventionOperation) bool {
	allowed := c.AllowedHumanOperations
	if len(allowed) == 0 {
		allowed = DefaultAutonomyContract().AllowedHumanOperations
	}
	for _, candidate := range allowed {
		if candidate == operation {
			return true
		}
	}
	return false
}
