import type {
  DelegationDecision,
  MissionRun,
  MissionTeamSelection,
  PlanExecutionPreview,
  WorkPlan,
} from './missions';
import type { AccountScope } from './usage';

/* ---- Composer sessions, intelligence, routing and Flow planning ---- */

export type IntelligenceMode = 'OFF' | 'CLI' | 'OPENAI_COMPATIBLE';

export interface IntelligenceStatus {
  mode: IntelligenceMode;
  provider?: string;
  profile?: string;
  base_url?: string;
  model?: string;
  api_key_env?: string;
  api_key_file?: string;
  available: boolean;
  error?: string;
}

export type ComposerState = 'EXPLORING' | 'READY_WITH_GAPS' | 'READY' | 'FINALIZED';

export interface ComposerSession {
  id: string;
  project_id: string;
  title: string;
  state: ComposerState;
  context_fingerprint: string;
  brief_json: string;
  revision: number;
  created_at: string;
  updated_at: string;
}

export interface ComposerTurn {
  id: string;
  session_id: string;
  sequence: number;
  role: 'USER' | 'ASSISTANT';
  content: string;
  created_at: string;
  kind?: string;
  payload_json?: string;
  payload?: Record<string, unknown>;
}

export interface ComposerSkillProposal {
  session_id: string;
  skill_id: string;
  state: 'SUGGESTED' | 'ACCEPTED' | 'APPLIED' | 'REJECTED' | 'UNAVAILABLE';
  reason: string;
  applicability: string;
  risk: string;
  source?: string;
  version?: string;
  available?: boolean;
  recommendation_score?: number;
  updated_at: string;
}

export interface ComposerSkillCatalogItem {
  id: string;
  name: string;
  description?: string;
  source: string;
  version?: string;
  risk?: string;
  availability: string;
  capabilities?: string[];
  triggers?: string[];
}

export type ComposerContextDecision = 'KEEP' | 'REFERENCE_ONLY' | 'DROP';

export interface ComposerContextItem {
  id: string;
  kind: string;
  source: string;
  locator?: string;
  summary: string;
  confidence?: string;
  decision: ComposerContextDecision;
  reason: string;
  estimated_tokens: number;
}

export interface ComposerContextManifest {
  version: string;
  project_id: string;
  fingerprint: string;
  generated_at: string;
  budget: {
    max_estimated_tokens: number;
    used_estimated_tokens: number;
    kept_items: number;
    referenced_items: number;
    dropped_items: number;
  };
  items: ComposerContextItem[];
  warnings?: string[];
}

export interface PromptArtifact {
  id: string;
  session_id: string;
  version: number;
  content: string;
  hash: string;
  context_json: string;
  skill_ids_json: string;
  created_at: string;
}

export interface FlowSuitabilityAssessment {
  result: 'DIRECT_FIT' | 'FLOW_BENEFICIAL' | 'FLOW_STRONGLY_RECOMMENDED' | string;
  reasons: string[];
  signals: string[];
}

export interface ComposerPromptReview {
  missing?: string[];
  contradictions?: string[];
  diff?: string[];
}

export interface PromptReadinessCheck {
  key: string;
  label: string;
  status: string;
  summary: string;
}

export interface ContextFileRef {
  path: string;
  excerpt?: string;
  hash?: string;
}

export interface ComposerSessionView {
  session: ComposerSession;
  brief: {
    goal: string;
    context?:
      | string[]
      | {
          existing_state?: string[];
          evidence?: string[];
          referenced_files?: ContextFileRef[];
          [key: string]: unknown;
        };
    constraints?: string[] | Record<string, unknown>;
    decisions?: string[];
    assumptions?: Array<string | { value?: string; status?: string; confidence?: string }>;
    alternatives?: string[];
    recommendations?: Array<{
      id: string;
      value: string;
      source_type?: string;
      confidence?: string;
      status: string;
    }>;
    risks?: string[];
    quality?: {
      acceptance_criteria?: string[];
      testing?: string[];
      verification?: string[];
      review?: string[];
      definition_of_done?: string[];
    };
    /** @deprecated use quality.acceptance_criteria */
    success_criteria?: string[];
    open_questions?: string[];
    intent?: { archetype?: string };
    preview?: string;
    readiness?: { score: number; state: string; summary: string; checks?: PromptReadinessCheck[] };
    unknowns?: Array<{
      id: string;
      field?: string;
      question: string;
      rationale?: string;
      severity: string;
      status: string;
      answer?: string;
      inferred_value?: string;
      confidence?: string;
      source?: string;
      suggested_options?: string[];
      default_choice?: string;
      dimension?: string;
      priority_score?: number;
    }>;
    interview?: {
      mode?: 'QUICK' | 'STANDARD' | 'DEEP' | string;
      clarity?: {
        intent?: number;
        scope?: number;
        context?: number;
        constraints?: number;
        acceptance?: number;
        risk?: number;
        decisions?: number;
      };
      ambiguity?: number;
      target_ambiguity?: number;
      weakest_dimension?: string;
      questions_answered?: number;
      max_questions?: number;
      challenge_pending?: boolean;
    };
  };
  turns: ComposerTurn[];
  skills: ComposerSkillProposal[];
  artifacts?: PromptArtifact[];
  flow_suitability?: FlowSuitabilityAssessment;
  prompt_review?: ComposerPromptReview;
  context_manifest?: ComposerContextManifest;
}

export interface IntelligenceMemory {
  id: string;
  scope: 'INSTALL' | 'PROJECT' | string;
  project_id?: string;
  key: string;
  value: string;
  source?: string;
  updated_at?: string;
}

export interface ClarificationUnknown {
  key: string;
  level: 'BLOCKING' | 'IMPORTANT' | 'LOW_IMPACT';
  question: string;
  rationale: string;
  suggested_options?: string[];
  default_choice?: string;
  answer?: string;
  is_resolved: boolean;
}

export interface ClarificationCheckpoint {
  id: string;
  project_id: string;
  goal: string;
  status: 'PENDING' | 'RESOLVED' | 'CANCELLED';
  intent: {
    intent: string;
    scope: string;
    risk_level: string;
    identified_goals: string[];
    constraints: string[];
    assumptions: string[];
    suggested_stack?: string[];
    created_at: string;
  };
  unknowns: ClarificationUnknown[];
  facts: Record<string, string>;
}

export interface PlanRevision {
  id: string;
  plan_id: string;
  revision: number;
  snapshot_json: string;
  change_summary: string;
  created_at: string;
}

export interface PlanRevisionDiff {
  from_revision: number;
  to_revision: number;
  title_changed: boolean;
  description_changed: boolean;
  added_packages: string[];
  removed_packages: string[];
  changed_packages: string[];
}

export interface RoutingAffinityPreference {
  mode: string;
  value?: string;
}

export interface RoutingAffinityPolicy {
  engine: RoutingAffinityPreference;
  provider: RoutingAffinityPreference;
  profile: RoutingAffinityPreference;
  model: RoutingAffinityPreference;
}

export interface RoutingModelCandidate {
  provider: string;
  profile: string;
  model: string;
  capabilities?: string[];
  healthy: boolean;
  authenticated: boolean;
  quota_available: boolean;
  cost_rank: number;
  reasoning_rank: number;
}

export interface RoutingSkillResolution {
  requested_capability: string;
  candidate_skills?: string[];
  selected_skill?: string;
  source?: string;
  version?: string;
  hash?: string;
  reason?: string;
}

export interface RoutingDecision {
  task_id?: string;
  work_plan_revision?: number;
  agent_id?: string;
  agent_score?: number;
  agent_confidence?: string;
  agent_reason?: string;
  task_requirements: Record<string, unknown>;
  desired: RoutingAffinityPolicy;
  affinity_policy: RoutingAffinityPolicy;
  actual: RoutingModelCandidate;
  selected_engine?: string;
  selected_provider?: string;
  selected_profile?: string;
  selected_account_scope?: AccountScope;
  selected_model?: string;
  selected_reasoning?: string;
  skill_refs?: string[];
  skill_resolutions?: RoutingSkillResolution[];
  maestro_guidance_ref?: string;
  alternatives?: RoutingModelCandidate[];
  fallback: boolean;
  reason: string;
  rejected?: string[];
  rejected_candidates?: string[];
  task_class?: string;
  created_at: string;
}

export interface RoutingDecisionReportEntry {
  package_id: string;
  title?: string;
  state?: string;
  decision: RoutingDecision;
}

export interface RoutingDecisionReport {
  run_id: string;
  plan_id?: string;
  work_plan_revision?: number;
  intent?: {
    strategy: string;
    confidence: string;
    known_facts?: string[];
    assumptions?: string[];
    unknowns?: string[];
    blocking_questions?: string[];
    evidence?: string[];
    recommended_next_action: string;
    snapshot_identity?: string;
    snapshot_completeness?: string;
  };
  delegation?: DelegationDecision;
  mission_team?: MissionTeamSelection;
  decisions: RoutingDecisionReportEntry[];
}

export interface FlowPreflightCheck {
  key: string;
  label: string;
  status: 'PASS' | 'WARN' | 'FAIL';
  summary: string;
}

export interface FlowPreflightReport {
  plan_id: string;
  revision: number;
  ready: boolean;
  checks: FlowPreflightCheck[];
  generated_at: string;
}

export interface FlowDecompositionRequest {
  project_id: string;
  artifact_id?: string;
  goal: string;
  source_prompt?: string;
  skill_ids?: string[];
  maestro_skills?: string[];
  simple?: boolean;
  delegation_mode?: 'AUTO' | 'ASK' | 'OFF';
  mission_team_preset?: string;
  mission_team_runtime_overrides?: Record<string, RoutingAffinityPolicy>;
}

export interface FlowDecompositionProposal {
  title: string;
  description: string;
  archetype: string;
  flow: import('../features/work/flowModel').FlowDraftModel;
  reasoning: string;
  maestro_advice?: string;
}

export interface ComparisonCheck {
  name: string;
  passed: boolean;
  exit_code: number;
  duration_ms: number;
}

export interface ComparisonCostEvidence {
  observation_count: number;
  known_cost_observations: number;
  unknown_cost_observations: number;
  provider_reported_microusd: number;
  api_equivalent_microusd: number;
  inferred_microusd: number;
  unclassified_microusd: number;
}

export interface ComparisonRunEvidence {
  run_id: string;
  variant_name?: string;
  variant_instructions?: string;
  state: string;
  base_revision: string;
  changed_files: string[];
  diff: string;
  diff_truncated: boolean;
  checks: ComparisonCheck[];
  cost: ComparisonCostEvidence;
  cost_evidence_status: 'COMPLETE' | 'PARTIAL' | 'UNKNOWN';
}

export interface ComparisonLab {
  id: string;
  project_id: string;
  plan_id: string;
  plan_revision: number;
  run_a_id: string;
  run_b_id: string;
  decision: 'PENDING' | 'SELECT_A' | 'SELECT_B' | 'REJECT_BOTH';
  decision_reason?: string;
  decided_at?: string;
  created_at: string;
  updated_at: string;
  run_a: ComparisonRunEvidence;
  run_b: ComparisonRunEvidence;
}

export interface ComparisonCampaign {
  id: string;
  project_id: string;
  source_plan_id: string;
  source_plan_revision: number;
  base_revision: string;
  variant_a_plan_id: string;
  variant_b_plan_id: string;
  variant_a_name: string;
  variant_a_instructions: string;
  variant_b_name: string;
  variant_b_instructions: string;
  run_a_id?: string;
  run_b_id?: string;
  state: 'PREPARED' | 'LAUNCHING' | 'RUNNING' | 'PARTIAL' | 'COMPLETED' | 'FAILED';
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface ComparisonCampaignView {
  campaign: ComparisonCampaign;
  plan_a: WorkPlan;
  plan_b: WorkPlan;
  preview_a?: PlanExecutionPreview;
  preview_b?: PlanExecutionPreview;
  run_a?: MissionRun;
  run_b?: MissionRun;
}

export interface CreateComparisonCampaignRequest {
  project_id: string;
  plan_id: string;
  plan_revision: number;
  variant_a: { name: string; instructions: string };
  variant_b: { name: string; instructions: string };
}

export interface LaunchComparisonCampaignRequest {
  approval_token_a?: string;
  cost_confirmation_token_a?: string;
  approval_token_b?: string;
  cost_confirmation_token_b?: string;
}
