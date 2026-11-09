package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

const (
	ComparisonCampaignPrepared  = "PREPARED"
	ComparisonCampaignLaunching = "LAUNCHING"
	ComparisonCampaignRunning   = "RUNNING"
	ComparisonCampaignPartial   = "PARTIAL"
	ComparisonCampaignCompleted = "COMPLETED"
	ComparisonCampaignFailed    = "FAILED"
)

type ComparisonCampaign struct {
	ID                   string    `json:"id"`
	ProjectID            string    `json:"project_id"`
	SourcePlanID         string    `json:"source_plan_id"`
	SourcePlanRevision   int       `json:"source_plan_revision"`
	BaseRevision         string    `json:"base_revision"`
	VariantAPlanID       string    `json:"variant_a_plan_id"`
	VariantBPlanID       string    `json:"variant_b_plan_id"`
	VariantAName         string    `json:"variant_a_name"`
	VariantAInstructions string    `json:"variant_a_instructions"`
	VariantBName         string    `json:"variant_b_name"`
	VariantBInstructions string    `json:"variant_b_instructions"`
	RunAID               string    `json:"run_a_id,omitempty"`
	RunBID               string    `json:"run_b_id,omitempty"`
	State                string    `json:"state"`
	LastError            string    `json:"last_error,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// CreateComparisonCampaign commits both variant WorkPlans and their campaign
// record in one transaction, so a failed prepare cannot leave one orphan plan.
func (s *Store) CreateComparisonCampaign(campaign ComparisonCampaign, variantA, variantB WorkPlan) (*ComparisonCampaign, error) {
	if campaign.ID == "" {
		campaign.ID = "comparison_campaign_" + ids.NewRuntimeID()
	}
	if strings.TrimSpace(campaign.ProjectID) == "" || strings.TrimSpace(campaign.SourcePlanID) == "" || campaign.SourcePlanRevision <= 0 || strings.TrimSpace(campaign.BaseRevision) == "" {
		return nil, fmt.Errorf("comparison campaign requires a source plan revision and base")
	}
	if variantA.ID == "" || variantB.ID == "" || variantA.ID == variantB.ID || variantA.ProjectID != campaign.ProjectID || variantB.ProjectID != campaign.ProjectID {
		return nil, fmt.Errorf("comparison campaign requires two distinct variant plans in the selected project")
	}
	if strings.TrimSpace(campaign.VariantAName) == "" || strings.TrimSpace(campaign.VariantBName) == "" || strings.TrimSpace(campaign.VariantAInstructions) == "" || strings.TrimSpace(campaign.VariantBInstructions) == "" {
		return nil, fmt.Errorf("comparison campaign requires names and instructions for both alternatives")
	}
	now := time.Now().UTC()
	campaign.State = ComparisonCampaignPrepared
	campaign.CreatedAt = now
	campaign.UpdatedAt = now
	variantA.CurrentRevision, variantB.CurrentRevision = 1, 1
	if variantA.Status == "" {
		variantA.Status = "READY"
	}
	if variantB.Status == "" {
		variantB.Status = "READY"
	}
	variantA.CreatedAt, variantA.UpdatedAt = now, now
	variantB.CreatedAt, variantB.UpdatedAt = now, now

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := insertComparisonVariantPlan(tx, variantA); err != nil {
		return nil, err
	}
	if err := insertComparisonVariantPlan(tx, variantB); err != nil {
		return nil, err
	}
	_, err = tx.Exec(`INSERT INTO comparison_campaigns
		(id, project_id, source_plan_id, source_plan_revision, base_revision, variant_a_plan_id, variant_b_plan_id,
		 variant_a_name, variant_a_instructions, variant_b_name, variant_b_instructions, state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		campaign.ID, campaign.ProjectID, campaign.SourcePlanID, campaign.SourcePlanRevision, campaign.BaseRevision,
		variantA.ID, variantB.ID, campaign.VariantAName, campaign.VariantAInstructions, campaign.VariantBName,
		campaign.VariantBInstructions, campaign.State, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("insert comparison campaign: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	campaign.VariantAPlanID, campaign.VariantBPlanID = variantA.ID, variantB.ID
	return &campaign, nil
}

func insertComparisonVariantPlan(tx *sql.Tx, plan WorkPlan) error {
	phasesJSON, err := json.Marshal(plan.Phases)
	if err != nil {
		return err
	}
	factsJSON, err := json.Marshal(plan.StructuredFacts)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO work_plans(id, project_id, mission_id, title, description, status, current_revision, phases_json, facts_json, created_at, updated_at)
		VALUES (?, ?, NULL, ?, ?, ?, 1, ?, ?, ?, ?)`, plan.ID, plan.ProjectID, plan.Title, plan.Description, plan.Status,
		string(phasesJSON), string(factsJSON), plan.CreatedAt.Format(time.RFC3339Nano), plan.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert comparison variant plan: %w", err)
	}
	snapshot, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO plan_revisions(id, plan_id, revision, snapshot_json, change_summary, created_at) VALUES (?, ?, 1, ?, ?, ?)`,
		"rev_"+ids.NewRuntimeID(), plan.ID, string(snapshot), "Comparison Lab variant frozen from source plan", plan.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert comparison variant revision: %w", err)
	}
	return nil
}

func (s *Store) GetComparisonCampaign(id string) (*ComparisonCampaign, error) {
	return scanComparisonCampaign(s.db.QueryRow(`SELECT id, project_id, source_plan_id, source_plan_revision, base_revision,
		variant_a_plan_id, variant_b_plan_id, variant_a_name, variant_a_instructions, variant_b_name, variant_b_instructions,
		run_a_id, run_b_id, state, last_error, created_at, updated_at FROM comparison_campaigns WHERE id=?`, id))
}

func (s *Store) ListComparisonCampaigns(projectID string) ([]ComparisonCampaign, error) {
	rows, err := s.db.Query(`SELECT id, project_id, source_plan_id, source_plan_revision, base_revision,
		variant_a_plan_id, variant_b_plan_id, variant_a_name, variant_a_instructions, variant_b_name, variant_b_instructions,
		run_a_id, run_b_id, state, last_error, created_at, updated_at FROM comparison_campaigns WHERE project_id=? ORDER BY created_at DESC LIMIT 50`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ComparisonCampaign, 0)
	for rows.Next() {
		item, scanErr := scanComparisonCampaign(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *Store) ListAllComparisonCampaigns() ([]ComparisonCampaign, error) {
	rows, err := s.db.Query(`SELECT id, project_id, source_plan_id, source_plan_revision, base_revision,
		variant_a_plan_id, variant_b_plan_id, variant_a_name, variant_a_instructions, variant_b_name, variant_b_instructions,
		run_a_id, run_b_id, state, last_error, created_at, updated_at FROM comparison_campaigns ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ComparisonCampaign, 0)
	for rows.Next() {
		item, scanErr := scanComparisonCampaign(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *Store) BeginComparisonCampaignLaunch(id string) (*ComparisonCampaign, error) {
	result, err := s.db.Exec(`UPDATE comparison_campaigns SET state=?, last_error='', updated_at=? WHERE id=? AND state=? AND run_a_id IS NULL AND run_b_id IS NULL`,
		ComparisonCampaignLaunching, time.Now().UTC().Format(time.RFC3339Nano), id, ComparisonCampaignPrepared)
	if err != nil {
		return nil, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if changed == 0 {
		return nil, fmt.Errorf("comparison campaign is not available to launch")
	}
	return s.GetComparisonCampaign(id)
}

func (s *Store) AttachComparisonCampaignRun(id string, variant string, runID string) (*ComparisonCampaign, error) {
	column := "run_a_id"
	if variant == "B" {
		column = "run_b_id"
	} else if variant != "A" {
		return nil, fmt.Errorf("comparison variant must be A or B")
	}
	result, err := s.db.Exec(`UPDATE comparison_campaigns SET `+column+`=?, updated_at=? WHERE id=? AND state IN (?, ?) AND `+column+` IS NULL`,
		runID, time.Now().UTC().Format(time.RFC3339Nano), id, ComparisonCampaignLaunching, ComparisonCampaignPartial)
	if err != nil {
		return nil, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if changed == 0 {
		return nil, fmt.Errorf("comparison campaign variant already has a run or launch is not active")
	}
	if _, err := s.db.Exec(`UPDATE comparison_campaigns SET state=?, last_error='', updated_at=? WHERE id=? AND state IN (?, ?) AND run_a_id IS NOT NULL AND run_b_id IS NOT NULL`,
		ComparisonCampaignRunning, time.Now().UTC().Format(time.RFC3339Nano), id, ComparisonCampaignLaunching, ComparisonCampaignPartial); err != nil {
		return nil, err
	}
	return s.GetComparisonCampaign(id)
}

func (s *Store) FailComparisonCampaignLaunch(id, message string) (*ComparisonCampaign, error) {
	message = strings.TrimSpace(message)
	if len(message) > 2000 {
		message = message[:2000]
	}
	state := ComparisonCampaignFailed
	var count int
	if err := s.db.QueryRow(`SELECT (run_a_id IS NOT NULL) + (run_b_id IS NOT NULL) FROM comparison_campaigns WHERE id=?`, id).Scan(&count); err != nil {
		return nil, err
	}
	if count > 0 {
		state = ComparisonCampaignPartial
	}
	if _, err := s.db.Exec(`UPDATE comparison_campaigns SET state=?, last_error=?, updated_at=? WHERE id=? AND state=?`,
		state, message, time.Now().UTC().Format(time.RFC3339Nano), id, ComparisonCampaignLaunching); err != nil {
		return nil, err
	}
	return s.GetComparisonCampaign(id)
}

// ResetComparisonCampaignLaunch is startup recovery only. It makes a campaign
// with no persisted Mission runs retryable after an interrupted process, but
// never clears ownership once any run exists.
func (s *Store) ResetComparisonCampaignLaunch(id string) error {
	result, err := s.db.Exec(`UPDATE comparison_campaigns SET state=?, last_error='', updated_at=?
		WHERE id=? AND state=? AND run_a_id IS NULL AND run_b_id IS NULL`,
		ComparisonCampaignPrepared, time.Now().UTC().Format(time.RFC3339Nano), id, ComparisonCampaignLaunching)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return fmt.Errorf("interrupted campaign has persisted run ownership")
	}
	return nil
}

func (s *Store) MarkComparisonCampaignInterrupted(id, message string) (*ComparisonCampaign, error) {
	message = strings.TrimSpace(message)
	if len(message) > 2000 {
		message = message[:2000]
	}
	_, err := s.db.Exec(`UPDATE comparison_campaigns SET state=?, last_error=?, updated_at=?
		WHERE id=? AND state=?`, ComparisonCampaignPartial, message, time.Now().UTC().Format(time.RFC3339Nano), id, ComparisonCampaignLaunching)
	if err != nil {
		return nil, err
	}
	return s.GetComparisonCampaign(id)
}

func (s *Store) FindComparisonCampaignByRuns(runAID, runBID string) (*ComparisonCampaign, error) {
	item, err := scanComparisonCampaign(s.db.QueryRow(`SELECT id, project_id, source_plan_id, source_plan_revision, base_revision,
		variant_a_plan_id, variant_b_plan_id, variant_a_name, variant_a_instructions, variant_b_name, variant_b_instructions,
		run_a_id, run_b_id, state, last_error, created_at, updated_at FROM comparison_campaigns
		WHERE (run_a_id=? AND run_b_id=?) OR (run_a_id=? AND run_b_id=?) LIMIT 1`, runAID, runBID, runBID, runAID))
	return item, err
}

type comparisonCampaignScanner interface{ Scan(...any) error }

func scanComparisonCampaign(row comparisonCampaignScanner) (*ComparisonCampaign, error) {
	var item ComparisonCampaign
	var runA, runB sql.NullString
	var created, updated string
	if err := row.Scan(&item.ID, &item.ProjectID, &item.SourcePlanID, &item.SourcePlanRevision, &item.BaseRevision,
		&item.VariantAPlanID, &item.VariantBPlanID, &item.VariantAName, &item.VariantAInstructions, &item.VariantBName,
		&item.VariantBInstructions, &runA, &runB, &item.State, &item.LastError, &created, &updated); err != nil {
		return nil, err
	}
	item.RunAID, item.RunBID = runA.String, runB.String
	item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &item, nil
}
