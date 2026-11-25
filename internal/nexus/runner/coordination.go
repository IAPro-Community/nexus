package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sdkcoordination "github.com/IAPro-Community/IAPro-SDK/coordination"
	"github.com/IAPro-Community/nexus/internal/control/ids"
)

// CoordinationOperation is a closed set of requests an executing Agent may
// submit to the Mission. Agents never spawn providers or mutate the Flow
// directly; MissionRunner remains the execution authority.
type CoordinationOperation = sdkcoordination.CoordinationOperation

const (
	CoordinationNeedInfo        = sdkcoordination.CoordinationNeedInfo
	CoordinationDelegate        = sdkcoordination.CoordinationDelegate
	CoordinationArtifactRequest = sdkcoordination.CoordinationArtifactRequest
	CoordinationReviewRequest   = sdkcoordination.CoordinationReviewRequest
)

// CoordinationStatus is the durable lifecycle of a coordination request.
type CoordinationStatus = sdkcoordination.CoordinationStatus

const (
	CoordinationPending  = sdkcoordination.CoordinationPending
	CoordinationResolved = sdkcoordination.CoordinationResolved
	CoordinationRejected = sdkcoordination.CoordinationRejected
	CoordinationFailed   = sdkcoordination.CoordinationFailed
	CoordinationCanceled = sdkcoordination.CoordinationCanceled

	// These are safety ceilings, not a competing delegation policy. The
	// canonical planner still decides whether delegation is useful; the runner
	// merely prevents provider output from exploding into unbounded child work.
	maxCoordinationRequestsPerTurn    = sdkcoordination.MaxRequestsPerTurn
	maxCoordinationRequestsPerMission = sdkcoordination.MaxRequestsPerRun
	maxCoordinationSubjectBytes       = 1024
	maxCoordinationListItems          = 16
)

// CoordinationRequest is a Mission-owned request raised by an executing Agent.
// It is intentionally provider-neutral: source/target are Agent/task concepts,
// while provider/profile/model selection stays with ResourceScheduler.
// CoordinationRequest is the shared envelope. Nexus retains durable storage,
// authorization, lineage and MissionRunner materialization.
type CoordinationRequest = sdkcoordination.CoordinationRequest

func isCoordinationTemplatePlaceholder(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "concise question or exact memory key", "what is needed", "your question", "placeholder", "todo", "tbd":
		return true
	}
	return (strings.HasPrefix(normalized, "<") && strings.HasSuffix(normalized, ">")) ||
		(strings.HasPrefix(normalized, "{{") && strings.HasSuffix(normalized, "}}"))
}

// ValidateCoordinationRequest enforces the architectural boundary before a
// request can become durable Mission state.
func ValidateCoordinationRequest(req CoordinationRequest) error {
	if err := sdkcoordination.ValidateCoordinationRequest(req); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"subject":            req.Subject,
		"objective":          req.Objective,
		"preferred_role":     req.PreferredRole,
		"preferred_provider": req.PreferredProvider,
	} {
		if isCoordinationTemplatePlaceholder(value) {
			return fmt.Errorf("coordination %s must contain a concrete value, not a template placeholder", name)
		}
	}
	for name, values := range map[string][]string{
		"required_capabilities": req.RequiredCapabilities,
		"artifact_refs":         req.ArtifactRefs,
	} {
		for _, value := range values {
			if isCoordinationTemplatePlaceholder(value) {
				return fmt.Errorf("coordination %s must contain concrete values, not template placeholders", name)
			}
		}
	}
	return nil
}

func coordinationSemanticallyEqual(left, right CoordinationRequest) bool {
	return sdkcoordination.SemanticallyEqual(left, right)
}

func cloneCoordinationRequest(in CoordinationRequest) CoordinationRequest {
	return sdkcoordination.CloneRequest(in)
}

// SubmitCoordinationRequest durably appends a validated request to the owning
// MissionRun. Replaying the same request ID is idempotent only when the
// semantic request is unchanged.
func coordinationRequestID(req CoordinationRequest) string {
	return sdkcoordination.RequestID(req)
}

func appendCoordinationRequestInRun(run *MissionRun, req CoordinationRequest) (*CoordinationRequest, error) {
	if run == nil {
		return nil, fmt.Errorf("mission run is required")
	}
	if req.RunID == "" {
		req.RunID = run.ID
	}
	if req.RunID != run.ID {
		return nil, fmt.Errorf("coordination request run_id %q does not match owning run %q", req.RunID, run.ID)
	}
	if req.SourceAgentID == "" {
		if source := packageByID(run, req.SourcePackageID); source != nil {
			req.SourceAgentID = source.AssignedAgent
		}
	}
	if req.Status == "" {
		req.Status = CoordinationPending
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now().UTC()
	}
	if req.ID == "" {
		req.ID = coordinationRequestID(req)
	}
	if err := ValidateCoordinationRequest(req); err != nil {
		return nil, err
	}
	if isTerminalRunState(run.State) {
		return nil, fmt.Errorf("mission run %s is terminal and cannot accept coordination", run.ID)
	}
	if packageIndex(run, req.SourcePackageID) < 0 {
		return nil, fmt.Errorf("coordination source package %q does not belong to mission run %s", req.SourcePackageID, run.ID)
	}
	for i := range run.CoordinationRequests {
		existing := &run.CoordinationRequests[i]
		if existing.ID != req.ID {
			continue
		}
		if coordinationSemanticallyEqual(*existing, req) {
			copy := cloneCoordinationRequest(*existing)
			return &copy, nil
		}
		return nil, fmt.Errorf("coordination request id %q already exists with different semantics", req.ID)
	}
	if len(run.CoordinationRequests) >= maxCoordinationRequestsPerMission {
		return nil, fmt.Errorf("mission coordination limit reached: %d requests", maxCoordinationRequestsPerMission)
	}
	run.CoordinationRequests = append(run.CoordinationRequests, cloneCoordinationRequest(req))
	run.UpdatedAt = time.Now().UTC()
	saved := cloneCoordinationRequest(req)
	return &saved, nil
}

// SubmitCoordinationRequest durably appends a validated request to the owning
// MissionRun. Replaying the same request ID is idempotent only when the
// semantic request is unchanged.
func (r *MissionRunner) SubmitCoordinationRequest(ctx context.Context, runID string, req CoordinationRequest) (*CoordinationRequest, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("mission runner repository is unavailable")
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("coordination run id is required")
	}
	current, err := r.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if err := requireLocalExecutionOwner(current); err != nil {
		return nil, err
	}
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, err
	}
	leaseToken := run.LeaseToken
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, leaseToken) }()

	saved, err := appendCoordinationRequestInRun(run, req)
	if err != nil {
		return nil, err
	}
	if err := r.saveRun(ctx, run); err != nil {
		return nil, err
	}
	return saved, nil
}

// ResolveCoordinationRequest closes a durable coordination request without
// mutating the sealed mission objective. Child-work materialization remains a
// separate MissionRunner concern.
func (r *MissionRunner) ResolveCoordinationRequest(ctx context.Context, runID, requestID string, status CoordinationStatus, summary, receiptID string) (*CoordinationRequest, error) {
	summary = strings.TrimSpace(summary)
	receiptID = strings.TrimSpace(receiptID)
	if status != CoordinationResolved && status != CoordinationRejected && status != CoordinationFailed && status != CoordinationCanceled {
		return nil, fmt.Errorf("coordination resolution status %q is not terminal", status)
	}
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("mission runner repository is unavailable")
	}
	runID = strings.TrimSpace(runID)
	current, err := r.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	// Owner is checked before the lease so a MAESTRO-owned run is never even
	// transiently supervised by the local runner.
	if err := requireLocalExecutionOwner(current); err != nil {
		return nil, err
	}
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, err
	}
	leaseToken := run.LeaseToken
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, leaseToken) }()

	for i := range run.CoordinationRequests {
		req := &run.CoordinationRequests[i]
		if req.ID != requestID {
			continue
		}
		if req.Status != CoordinationPending {
			if req.Status == status && req.ResolutionSummary == summary && req.ResultReceiptID == receiptID {
				copy := cloneCoordinationRequest(*req)
				return &copy, nil
			}
			return nil, fmt.Errorf("coordination request %q is already %s", requestID, req.Status)
		}
		now := time.Now().UTC()
		req.Status = status
		req.ResolvedAt = &now
		req.ResolutionSummary = summary
		req.ResultReceiptID = receiptID
		run.UpdatedAt = now
		if err := r.saveRun(ctx, run); err != nil {
			return nil, err
		}
		copy := cloneCoordinationRequest(*req)
		return &copy, nil
	}
	return nil, fmt.Errorf("coordination request %q not found in mission run %s", requestID, runID)
}

func coordinationPackageID(requestID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(requestID)))
	return "coord_" + hex.EncodeToString(sum[:8])
}

func coordinationRequestByID(run *MissionRun, requestID string) *CoordinationRequest {
	for i := range run.CoordinationRequests {
		if run.CoordinationRequests[i].ID == requestID {
			return &run.CoordinationRequests[i]
		}
	}
	return nil
}

func dependenciesVerified(run *MissionRun, dependencies []string) bool {
	for _, dependencyID := range dependencies {
		dependency := packageByID(run, dependencyID)
		if dependency == nil || dependency.State != StateVerified {
			return false
		}
	}
	return true
}

// MaterializeCoordinationWork turns one durable coordination request into a
// Mission-owned child PackageRun. The source package is never rewound or
// re-dispatched; dynamic work is additive and therefore cannot duplicate an
// already-crossed provider side effect.
func defaultCoordinationRole(operation CoordinationOperation) string {
	switch operation {
	case CoordinationNeedInfo, CoordinationArtifactRequest:
		return "investigator"
	case CoordinationReviewRequest:
		return "reviewer"
	case CoordinationDelegate:
		return "implementer"
	default:
		return ""
	}
}

func materializeCoordinationWorkInRun(run *MissionRun, requestID string) (*PackageRun, error) {
	if run == nil {
		return nil, fmt.Errorf("mission run is required")
	}
	if isTerminalRunState(run.State) {
		return nil, fmt.Errorf("mission run %s is terminal and cannot materialize coordination work", run.ID)
	}
	req := coordinationRequestByID(run, strings.TrimSpace(requestID))
	if req == nil {
		return nil, fmt.Errorf("coordination request %q not found in mission run %s", requestID, run.ID)
	}
	if req.Status != CoordinationPending {
		return nil, fmt.Errorf("coordination request %q is %s and cannot materialize work", requestID, req.Status)
	}
	if req.ChildPackageID != "" {
		if existing := packageByID(run, req.ChildPackageID); existing != nil {
			copy := *existing
			return &copy, nil
		}
		return nil, fmt.Errorf("coordination request %q references missing child package %q", requestID, req.ChildPackageID)
	}

	source := packageByID(run, req.SourcePackageID)
	if source == nil {
		return nil, fmt.Errorf("coordination source package %q does not belong to mission run %s", req.SourcePackageID, run.ID)
	}

	packageID := coordinationPackageID(req.ID)
	if existing := packageByID(run, packageID); existing != nil {
		req.ChildPackageID = existing.PackageID
		now := time.Now().UTC()
		req.MaterializedAt = &now
		run.UpdatedAt = now
		copy := *existing
		return &copy, nil
	}

	goal := strings.TrimSpace(req.Objective)
	if goal == "" {
		goal = strings.TrimSpace(req.Subject)
	}
	role := strings.TrimSpace(req.PreferredRole)
	if role == "" {
		role = defaultCoordinationRole(req.Operation)
	}
	resourcePolicy := source.ResourcePolicy
	if strings.TrimSpace(req.PreferredProvider) != "" {
		// A coordination request may prefer a provider, but never upgrades that
		// preference into the source package's MANUAL/PIN authority.
		resourcePolicy = "PREFER_PROVIDER"
	}
	dependencies := append([]string(nil), source.Dependencies...)
	state := StatePending
	if dependenciesVerified(run, dependencies) {
		state = StateReady
	}
	taskRequirements, err := json.Marshal(map[string]any{
		"task_kind":             "coordination",
		"role":                  role,
		"required_capabilities": append([]string(nil), req.RequiredCapabilities...),
		"source":                "mission_coordination",
	})
	if err != nil {
		return nil, fmt.Errorf("encode coordination task requirements: %w", err)
	}
	handoff, _ := json.Marshal(map[string]any{
		"coordination_request_id": req.ID,
		"operation":               req.Operation,
		"subject":                 req.Subject,
		"objective":               req.Objective,
		"source_package_id":       req.SourcePackageID,
		"source_agent_id":         req.SourceAgentID,
		"artifact_refs":           req.ArtifactRefs,
		"required_capabilities":   req.RequiredCapabilities,
	})
	now := time.Now().UTC()
	child := PackageRun{
		ID:                    "pkgrun_" + ids.NewRuntimeID(),
		PackageID:             packageID,
		PhaseID:               packageID,
		PhaseType:             "COORDINATION",
		Title:                 fmt.Sprintf("%s · %s", req.Operation, req.Subject),
		Goal:                  goal,
		Priority:              source.Priority,
		Role:                  role,
		TaskRequirements:      string(taskRequirements),
		Dependencies:          dependencies,
		AssignmentStrategy:    "AUTO",
		ResourcePolicy:        resourcePolicy,
		DesiredProvider:       strings.TrimSpace(req.PreferredProvider),
		RelevantPaths:         append([]string(nil), source.RelevantPaths...),
		HandoffJSON:           string(handoff),
		DynamicWork:           true,
		ParentPackageID:       source.PackageID,
		CoordinationRequestID: req.ID,
		State:                 state,
		Attempt:               1,
		StartedAt:             now,
	}
	run.PackageRuns = append(run.PackageRuns, child)
	req.ChildPackageID = child.PackageID
	req.MaterializedAt = &now
	run.UpdatedAt = now
	saved := child
	return &saved, nil
}

// MaterializeCoordinationWork turns one durable coordination request into a
// Mission-owned child PackageRun. The source package is never rewound or
// re-dispatched; dynamic work is additive and therefore cannot duplicate an
// already-crossed provider side effect.
func (r *MissionRunner) MaterializeCoordinationWork(ctx context.Context, runID, requestID string) (*PackageRun, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("mission runner repository is unavailable")
	}
	runID = strings.TrimSpace(runID)
	current, err := r.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	// Owner is checked before the lease so a MAESTRO-owned run is never even
	// transiently supervised by the local runner.
	if err := requireLocalExecutionOwner(current); err != nil {
		return nil, err
	}
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, err
	}
	leaseToken := run.LeaseToken
	defer func() { _ = r.repo.ReleaseLease(context.Background(), run.ID, r.owner, leaseToken) }()

	saved, err := materializeCoordinationWorkInRun(run, requestID)
	if err != nil {
		return nil, err
	}
	if err := r.saveRun(ctx, run); err != nil {
		return nil, err
	}
	return saved, nil
}

func appendUniquePackageID(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func coordinationOperationForPackage(run *MissionRun, pkg *PackageRun) CoordinationOperation {
	if run == nil || pkg == nil || strings.TrimSpace(pkg.CoordinationRequestID) == "" {
		return ""
	}
	if req := coordinationRequestByID(run, pkg.CoordinationRequestID); req != nil {
		return req.Operation
	}
	return ""
}

func boundedCoordinationSummary(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	const maxRunes = 8192
	runes := []rune(value)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "…"
	}
	return value
}

func (r *MissionRunner) acceptExecutionCoordination(run *MissionRun, sourcePackageID string, requests []CoordinationRequest) error {
	if len(requests) == 0 {
		return nil
	}
	if len(requests) > maxCoordinationRequestsPerTurn {
		return fmt.Errorf("coordination fan-out exceeds safe turn limit: %d > %d", len(requests), maxCoordinationRequestsPerTurn)
	}
	source := packageByID(run, sourcePackageID)
	if source == nil {
		return fmt.Errorf("coordination source package %q not found", sourcePackageID)
	}
	childIDs := make([]string, 0, len(requests))
	for _, incoming := range requests {
		incoming.RunID = run.ID
		incoming.SourcePackageID = source.PackageID
		if strings.TrimSpace(incoming.SourceAgentID) == "" {
			incoming.SourceAgentID = source.AssignedAgent
		}
		incoming.Status = CoordinationPending
		saved, err := appendCoordinationRequestInRun(run, incoming)
		if err != nil {
			return err
		}
		if saved.Status != CoordinationPending {
			return fmt.Errorf("coordination request %q is already %s", saved.ID, saved.Status)
		}
		child, err := materializeCoordinationWorkInRun(run, saved.ID)
		if err != nil {
			return err
		}
		childIDs = append(childIDs, child.PackageID)
	}

	source = packageByID(run, sourcePackageID)
	if source == nil {
		return fmt.Errorf("coordination source package %q disappeared after materialization", sourcePackageID)
	}
	for _, childID := range childIDs {
		source.Dependencies = appendUniquePackageID(source.Dependencies, childID)
	}
	source.ResumeAgentID = source.AssignedAgent
	source.State = StatePending
	source.ContextCapsule = nil
	source.ContextRefreshRequired = true
	source.PromptVersionID = ""
	source.CompiledPrompt = ""
	source.ResultSummary = ""
	source.Verifications = nil
	source.FinishedAt = nil
	source.ErrorMessage = ""
	source.FailureKind = ""
	resetDispatch(source)
	run.UpdatedAt = time.Now().UTC()
	return nil
}

func reconcileCoordinationRequests(run *MissionRun) {
	if run == nil {
		return
	}
	for i := range run.CoordinationRequests {
		req := &run.CoordinationRequests[i]
		if req.Status != CoordinationPending || req.ChildPackageID == "" {
			continue
		}
		child := packageByID(run, req.ChildPackageID)
		if child == nil || child.WorkReceipt == nil {
			continue
		}
		now := time.Now().UTC()
		req.ResolvedAt = &now
		req.ResultReceiptID = child.WorkReceipt.ID
		req.ResolutionSummary = child.WorkReceipt.Summary
		switch child.WorkReceipt.Status {
		case "VERIFIED":
			req.Status = CoordinationResolved
		case "FAILED":
			req.Status = CoordinationFailed
		default:
			req.ResolvedAt = nil
			req.ResultReceiptID = ""
			req.ResolutionSummary = ""
		}
	}
}
