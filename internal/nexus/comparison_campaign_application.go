package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const comparisonCampaignFactPrefix = "nexus.comparison_campaign."

type ComparisonCampaignVariantInput struct {
	Name         string `json:"name"`
	Instructions string `json:"instructions"`
}

type CreateComparisonCampaignRequest struct {
	ProjectID    string                         `json:"project_id"`
	PlanID       string                         `json:"plan_id"`
	PlanRevision int                            `json:"plan_revision"`
	VariantA     ComparisonCampaignVariantInput `json:"variant_a"`
	VariantB     ComparisonCampaignVariantInput `json:"variant_b"`
}

type LaunchComparisonCampaignRequest struct {
	ApprovalTokenA         string `json:"approval_token_a"`
	CostConfirmationTokenA string `json:"cost_confirmation_token_a"`
	ApprovalTokenB         string `json:"approval_token_b"`
	CostConfirmationTokenB string `json:"cost_confirmation_token_b"`
}

type ComparisonCampaignView struct {
	Campaign store.ComparisonCampaign `json:"campaign"`
	PlanA    store.WorkPlan           `json:"plan_a"`
	PlanB    store.WorkPlan           `json:"plan_b"`
	PreviewA *PlanExecutionPreview    `json:"preview_a,omitempty"`
	PreviewB *PlanExecutionPreview    `json:"preview_b,omitempty"`
	RunA     *runner.MissionRun       `json:"run_a,omitempty"`
	RunB     *runner.MissionRun       `json:"run_b,omitempty"`
}

// ComparisonCampaignApplicationService prepares two frozen variants and
// dispatches them through the existing Mission admission/runner boundary.
type ComparisonCampaignApplicationService struct {
	nexus        *Nexus
	runs         *RunApplicationService
	recoveryOnce sync.Once
	recoveryErr  error
}

func NewComparisonCampaignApplicationService(n *Nexus, runs *RunApplicationService) *ComparisonCampaignApplicationService {
	return &ComparisonCampaignApplicationService{nexus: n, runs: runs}
}

// RecoverInterrupted reconciles durable campaign state with Mission runs after
// a process restart. It never starts work: missing run records restore PREPARED;
// one-sided ownership becomes PARTIAL and requires human review.
func (s *ComparisonCampaignApplicationService) RecoverInterrupted(ctx context.Context) error {
	st, err := s.baseReady(ctx)
	if err != nil {
		return err
	}
	campaigns, err := st.ListAllComparisonCampaigns()
	if err != nil {
		return err
	}
	runs, err := s.nexus.Runner().ListRuns(ctx)
	if err != nil {
		return err
	}
	byPlan := make(map[string]runner.MissionRun, len(runs))
	for _, run := range runs {
		if run != nil {
			byPlan[run.PlanID] = *run
		}
	}
	for _, campaign := range campaigns {
		if campaign.State != store.ComparisonCampaignLaunching && campaign.State != store.ComparisonCampaignPartial {
			continue
		}
		if campaign.RunAID == "" {
			if run, ok := byPlan[campaign.VariantAPlanID]; ok {
				if _, attachErr := st.AttachComparisonCampaignRun(campaign.ID, "A", run.ID); attachErr != nil {
					return fmt.Errorf("recover comparison campaign %s alternative A: %w", campaign.ID, attachErr)
				}
			}
		}
		if campaign.RunBID == "" {
			if run, ok := byPlan[campaign.VariantBPlanID]; ok {
				if _, attachErr := st.AttachComparisonCampaignRun(campaign.ID, "B", run.ID); attachErr != nil {
					return fmt.Errorf("recover comparison campaign %s alternative B: %w", campaign.ID, attachErr)
				}
			}
		}
		current, getErr := st.GetComparisonCampaign(campaign.ID)
		if getErr != nil {
			return getErr
		}
		hasA, hasB := current.RunAID != "", current.RunBID != ""
		switch {
		case !hasA && !hasB && campaign.State == store.ComparisonCampaignLaunching:
			if err := st.ResetComparisonCampaignLaunch(campaign.ID); err != nil {
				return err
			}
		case hasA != hasB && campaign.State == store.ComparisonCampaignLaunching:
			if _, err := st.MarkComparisonCampaignInterrupted(campaign.ID, "process stopped after only one alternative acquired a Mission run; inspect the existing run before continuing"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ComparisonCampaignApplicationService) ready(ctx context.Context) (*store.Store, error) {
	st, err := s.baseReady(ctx)
	if err != nil {
		return nil, err
	}
	s.recoveryOnce.Do(func() {
		recoveryCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.recoveryErr = s.RecoverInterrupted(recoveryCtx)
	})
	if s.recoveryErr != nil {
		return nil, fmt.Errorf("comparison campaign recovery failed: %w", s.recoveryErr)
	}
	return st, nil
}

func (s *ComparisonCampaignApplicationService) baseReady(ctx context.Context) (*store.Store, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.nexus == nil || s.runs == nil || s.nexus.Store() == nil {
		return nil, fmt.Errorf("comparison campaign service unavailable")
	}
	return s.nexus.Store(), nil
}

func (s *ComparisonCampaignApplicationService) Create(ctx context.Context, req CreateComparisonCampaignRequest) (*ComparisonCampaignView, error) {
	st, err := s.ready(ctx)
	if err != nil {
		return nil, err
	}
	req.ProjectID, req.PlanID = strings.TrimSpace(req.ProjectID), strings.TrimSpace(req.PlanID)
	if req.ProjectID == "" || req.PlanID == "" || req.PlanRevision <= 0 {
		return nil, fmt.Errorf("project, plan, and frozen revision are required")
	}
	if err := validateComparisonVariantInput(req.VariantA); err != nil {
		return nil, fmt.Errorf("alternative A: %w", err)
	}
	if err := validateComparisonVariantInput(req.VariantB); err != nil {
		return nil, fmt.Errorf("alternative B: %w", err)
	}
	if strings.EqualFold(strings.TrimSpace(req.VariantA.Name), strings.TrimSpace(req.VariantB.Name)) {
		return nil, fmt.Errorf("alternative names must be distinct")
	}
	source, err := st.GetWorkPlan(req.PlanID)
	if err != nil {
		return nil, err
	}
	if source.ProjectID != req.ProjectID {
		return nil, fmt.Errorf("source plan belongs to a different project")
	}
	if source.CurrentRevision != req.PlanRevision {
		return nil, fmt.Errorf("source plan revision changed; refresh it before creating alternatives")
	}
	frozen, _, err := loadWorkPlanRevision(st, req.PlanID, req.PlanRevision)
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(req.ProjectID)
	if err != nil {
		return nil, err
	}
	base, err := comparisonGitOutput(ctx, project.CanonicalPath, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("resolve comparison base: %w", err)
	}
	status, err := comparisonGitOutput(ctx, project.CanonicalPath, "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return nil, fmt.Errorf("inspect comparison base: %w", err)
	}
	if strings.TrimSpace(status) != "" {
		return nil, fmt.Errorf("comparison campaign requires a clean project checkout")
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return nil, fmt.Errorf("comparison base revision is empty")
	}
	campaignID := "comparison_campaign_" + ids.NewRuntimeID()
	planA, err := comparisonVariantPlan(*frozen, campaignID, "A", req.VariantA)
	if err != nil {
		return nil, err
	}
	planB, err := comparisonVariantPlan(*frozen, campaignID, "B", req.VariantB)
	if err != nil {
		return nil, err
	}
	campaign, err := st.CreateComparisonCampaign(store.ComparisonCampaign{
		ID: campaignID, ProjectID: req.ProjectID, SourcePlanID: source.ID, SourcePlanRevision: req.PlanRevision,
		BaseRevision: base, VariantAName: strings.TrimSpace(req.VariantA.Name), VariantAInstructions: strings.TrimSpace(req.VariantA.Instructions),
		VariantBName: strings.TrimSpace(req.VariantB.Name), VariantBInstructions: strings.TrimSpace(req.VariantB.Instructions),
	}, planA, planB)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, *campaign)
}

func validateComparisonVariantInput(input ComparisonCampaignVariantInput) error {
	name, instructions := strings.TrimSpace(input.Name), strings.TrimSpace(input.Instructions)
	if name == "" || len([]rune(name)) > 80 {
		return fmt.Errorf("name is required and must be at most 80 characters")
	}
	if instructions == "" || len([]rune(instructions)) > 4000 {
		return fmt.Errorf("instructions are required and must be at most 4000 characters")
	}
	return nil
}

func comparisonVariantPlan(source store.WorkPlan, campaignID, variant string, input ComparisonCampaignVariantInput) (store.WorkPlan, error) {
	if len(source.Phases) == 0 {
		return store.WorkPlan{}, fmt.Errorf("source plan has no executable phases")
	}
	snapshot, err := json.Marshal(source)
	if err != nil {
		return store.WorkPlan{}, err
	}
	var plan store.WorkPlan
	if err := json.Unmarshal(snapshot, &plan); err != nil {
		return store.WorkPlan{}, err
	}
	plan.ID = "plan_" + ids.NewRuntimeID()
	plan.MissionID = ""
	plan.Title = strings.TrimSpace(source.Title) + " · " + strings.TrimSpace(input.Name)
	plan.Status = "READY"
	plan.CurrentRevision = 1
	plan.CreatedAt, plan.UpdatedAt = plan.CreatedAt.UTC(), plan.UpdatedAt.UTC()
	plan.StructuredFacts = cloneComparisonFacts(source.StructuredFacts)
	plan.StructuredFacts[comparisonCampaignFactPrefix+"id"] = campaignID
	plan.StructuredFacts[comparisonCampaignFactPrefix+"source_plan_id"] = source.ID
	plan.StructuredFacts[comparisonCampaignFactPrefix+"source_revision"] = fmt.Sprint(source.CurrentRevision)
	plan.StructuredFacts[comparisonCampaignFactPrefix+"variant"] = variant
	plan.StructuredFacts[comparisonCampaignFactPrefix+"name"] = strings.TrimSpace(input.Name)
	plan.StructuredFacts[comparisonCampaignFactPrefix+"instructions"] = strings.TrimSpace(input.Instructions)
	packageIDs := make(map[string]string)
	for phaseIndex := range plan.Phases {
		plan.Phases[phaseIndex].ID = "phase_" + ids.NewRuntimeID()
		for packageIndex := range plan.Phases[phaseIndex].Packages {
			pkg := &plan.Phases[phaseIndex].Packages[packageIndex]
			oldID := pkg.ID
			pkg.ID = "pkg_" + ids.NewRuntimeID()
			packageIDs[oldID] = pkg.ID
			pkg.Goal = strings.TrimSpace(pkg.Goal) + "\n\nComparison alternative " + variant + " (" + strings.TrimSpace(input.Name) + "): " + strings.TrimSpace(input.Instructions)
		}
	}
	for phaseIndex := range plan.Phases {
		for packageIndex := range plan.Phases[phaseIndex].Packages {
			pkg := &plan.Phases[phaseIndex].Packages[packageIndex]
			for dependencyIndex, dependency := range pkg.Dependencies {
				mapped, ok := packageIDs[dependency]
				if !ok {
					return store.WorkPlan{}, fmt.Errorf("source package %s has an unknown dependency %s", pkg.ID, dependency)
				}
				pkg.Dependencies[dependencyIndex] = mapped
			}
		}
	}
	return plan, nil
}

func cloneComparisonFacts(facts map[string]string) map[string]string {
	copy := make(map[string]string, len(facts)+6)
	for key, value := range facts {
		copy[key] = value
	}
	return copy
}

func comparisonGitOutput(ctx context.Context, workspace string, args ...string) (string, error) {
	cmdArgs := append([]string{"-C", workspace}, args...)
	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *ComparisonCampaignApplicationService) Get(ctx context.Context, id string) (*ComparisonCampaignView, error) {
	_, err := s.ready(ctx)
	if err != nil {
		return nil, err
	}
	campaign, err := s.nexus.Store().GetComparisonCampaign(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	return s.view(ctx, *campaign)
}

func (s *ComparisonCampaignApplicationService) List(ctx context.Context, projectID string) ([]ComparisonCampaignView, error) {
	st, err := s.ready(ctx)
	if err != nil {
		return nil, err
	}
	campaigns, err := st.ListComparisonCampaigns(strings.TrimSpace(projectID))
	if err != nil {
		return nil, err
	}
	items := make([]ComparisonCampaignView, 0, len(campaigns))
	for _, campaign := range campaigns {
		view, viewErr := s.view(ctx, campaign)
		if viewErr != nil {
			return nil, viewErr
		}
		items = append(items, *view)
	}
	return items, nil
}

func (s *ComparisonCampaignApplicationService) view(ctx context.Context, campaign store.ComparisonCampaign) (*ComparisonCampaignView, error) {
	st, err := s.ready(ctx)
	if err != nil {
		return nil, err
	}
	planA, err := st.GetWorkPlan(campaign.VariantAPlanID)
	if err != nil {
		return nil, err
	}
	planB, err := st.GetWorkPlan(campaign.VariantBPlanID)
	if err != nil {
		return nil, err
	}
	view := &ComparisonCampaignView{Campaign: campaign, PlanA: *planA, PlanB: *planB}
	if campaign.State == store.ComparisonCampaignPrepared {
		view.PreviewA, err = s.nexus.PreviewPlanExecution(ctx, planA.ID)
		if err != nil {
			return nil, err
		}
		view.PreviewB, err = s.nexus.PreviewPlanExecution(ctx, planB.ID)
		if err != nil {
			return nil, err
		}
	}
	if campaign.RunAID != "" {
		view.RunA, _ = s.nexus.Runner().GetRun(ctx, campaign.RunAID)
	}
	if campaign.RunBID != "" {
		view.RunB, _ = s.nexus.Runner().GetRun(ctx, campaign.RunBID)
	}
	if view.RunA != nil && view.RunB != nil && isTerminalComparisonRun(view.RunA.State) && isTerminalComparisonRun(view.RunB.State) && campaign.State == store.ComparisonCampaignRunning {
		// State is derived only from persisted terminal run facts; a read never
		// mutates campaign state or initiates work.
		view.Campaign.State = store.ComparisonCampaignCompleted
	}
	return view, nil
}

func (s *ComparisonCampaignApplicationService) Launch(ctx context.Context, id string, approval LaunchComparisonCampaignRequest) (*ComparisonCampaignView, error) {
	st, err := s.ready(ctx)
	if err != nil {
		return nil, err
	}
	campaign, err := st.GetComparisonCampaign(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if campaign.State != store.ComparisonCampaignPrepared {
		return nil, fmt.Errorf("comparison campaign is not prepared for launch")
	}
	project, err := st.GetProject(campaign.ProjectID)
	if err != nil {
		return nil, err
	}
	base, err := comparisonGitOutput(ctx, project.CanonicalPath, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(base) != campaign.BaseRevision {
		message := "project base changed since alternatives were prepared"
		if err != nil {
			message = "resolve project base before launch: " + err.Error()
		}
		return nil, fmt.Errorf("%s", message)
	}
	status, err := comparisonGitOutput(ctx, project.CanonicalPath, "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return nil, fmt.Errorf("inspect comparison base before launch: %w", err)
	}
	if strings.TrimSpace(status) != "" {
		return nil, fmt.Errorf("project checkout is dirty; reconcile it before launching the prepared alternatives")
	}
	planA, err := st.GetWorkPlan(campaign.VariantAPlanID)
	if err != nil {
		return nil, err
	}
	planB, err := st.GetWorkPlan(campaign.VariantBPlanID)
	if err != nil {
		return nil, err
	}
	previewA, err := s.nexus.PreviewPlanExecution(ctx, planA.ID)
	if err != nil {
		return nil, err
	}
	previewB, err := s.nexus.PreviewPlanExecution(ctx, planB.ID)
	if err != nil {
		return nil, err
	}
	if !previewA.Ready || !previewB.Ready {
		return nil, fmt.Errorf("both alternative execution previews must be ready")
	}
	if strings.TrimSpace(approval.ApprovalTokenA) == "" || strings.TrimSpace(approval.ApprovalTokenA) != previewA.ApprovalToken || strings.TrimSpace(approval.ApprovalTokenB) == "" || strings.TrimSpace(approval.ApprovalTokenB) != previewB.ApprovalToken {
		return nil, fmt.Errorf("both alternatives require explicit approval of their current execution previews")
	}
	if previewA.CostConfirmationRequired && strings.TrimSpace(approval.CostConfirmationTokenA) != previewA.CostConfirmationToken {
		return nil, fmt.Errorf("alternative A requires explicit cost confirmation")
	}
	if previewB.CostConfirmationRequired && strings.TrimSpace(approval.CostConfirmationTokenB) != previewB.CostConfirmationToken {
		return nil, fmt.Errorf("alternative B requires explicit cost confirmation")
	}
	campaign, err = st.BeginComparisonCampaignLaunch(campaign.ID)
	if err != nil {
		return nil, err
	}
	type launchResult struct {
		variant string
		run     *runner.MissionRun
		err     error
	}
	results := make(chan launchResult, 2)
	start := func(variant string, planID, approvalToken, costToken string) {
		run, startErr := s.runs.Start(ctx, RunStartRequest{
			PlanID: planID, Revision: 1, Contract: runner.DefaultAutonomyContract(), Autonomous: true,
			CostConfirmationToken: costToken, ExecutionPreviewToken: approvalToken,
		})
		results <- launchResult{variant: variant, run: run, err: startErr}
	}
	go start("A", planA.ID, approval.ApprovalTokenA, approval.CostConfirmationTokenA)
	go start("B", planB.ID, approval.ApprovalTokenB, approval.CostConfirmationTokenB)
	var launchErrors []string
	for i := 0; i < 2; i++ {
		result := <-results
		if result.err != nil {
			launchErrors = append(launchErrors, result.variant+": "+result.err.Error())
			continue
		}
		if _, attachErr := st.AttachComparisonCampaignRun(campaign.ID, result.variant, result.run.ID); attachErr != nil {
			launchErrors = append(launchErrors, result.variant+": run started but campaign link failed: "+attachErr.Error())
		}
	}
	if len(launchErrors) > 0 {
		partial, _ := st.FailComparisonCampaignLaunch(campaign.ID, strings.Join(launchErrors, " | "))
		if partial != nil {
			return s.view(ctx, *partial)
		}
		return nil, fmt.Errorf("comparison campaign launch partially failed: %s", strings.Join(launchErrors, " | "))
	}
	updated, err := st.GetComparisonCampaign(campaign.ID)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, *updated)
}
