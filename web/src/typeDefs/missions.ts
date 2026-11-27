import type { ConfigImpact, MissionAssignment } from './agents';
import type { CostAnomalyAssessment, CostBudgetAssessment, ProviderAccount } from './usage';
import type { RoutingDecisionReport } from './composer';
import type {
  ExecutionTaskRequirements,
  PreviewAgentDecision,
  PreviewModelDecision,
  PreviewResourceDecision,
} from './maestro';

export interface Mission {
  id: string;
  project_id: string;
  name: string;
  description: string;
  status: string;
  goal: string;
  scope: string;
  risk_level: string;
  config: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface MissionTask {
  id: string;
  mission_id: string;
  name: string;
  description: string;
  status: string;
  kind: string;
  priority: number;
  dependencies: string;
  config: string;
  result: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface MissionTaskInput {
  name: string;
  description?: string;
  kind?: string;
  priority?: number;
}

export interface MissionStats {
  total: number;
  pending: number;
  active: number;
  completed: number;
  failed: number;
}

export interface MissionDetail {
  mission: Mission;
  tasks: MissionTask[];
  assignments: MissionAssignment[];
  stats: MissionStats;
}

export interface MissionCreateInput {
  name: string;
  description?: string;
  goal?: string;
  scope?: string;
  risk_level?: string;
}

export interface MissionPatch {
  name?: string;
  description?: string;
  status?: string;
  goal?: string;
  scope?: string;
  risk_level?: string;
}

export interface MissionRun {
  id: string;
  plan_id: string;
  plan_revision: number;
  execution_snapshot_id?: string;
  execution_owner?: 'NEXUS' | 'MAESTRO';
  project_id: string;
  workspace: string;
  integrated_workspace?: string;
  delivery_integration?: DeliveryIntegrationEvidence;
  state: string;
  contract: AutonomyContract;
  current_pkg_index: number;
  current_phase_id?: string;
  total_iterations: number;
  package_runs: PackageRun[];
  phase_runs?: PhaseRun[];
  paused_reason?: string;
  started_at: string;
  updated_at: string;
  completed_at?: string;
  global_verifications?: VerificationResult[];
  needs_human?: HumanIntervention;
  resume_request?: MissionResumeRequest;
  coordination_requests?: CoordinationRequest[];
}

export interface MissionResumeRequest {
  id: string;
  idempotency_key: string;
  status: 'PENDING' | 'STARTED' | 'COMPLETED' | string;
  requested_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface MissionSchedule {
  id: string;
  plan_id: string;
  project_id: string;
  mode: 'AT' | 'AFTER_RUN' | 'WHEN_RESOURCES';
  scheduled_for?: string;
  after_run_id?: string;
  status: 'PENDING' | 'RUNNING' | 'COMPLETED' | 'CANCELED' | 'FAILED';
  run_id?: string;
  created_at: string;
  updated_at: string;
}

export interface MissionTeamSelection {
  version: number;
  strategy: string;
  preset?: string;
  direct: boolean;
  reason: string;
  signals?: string[];
  cost_class: string;
  review_required: boolean;
  created_at: string;
}

export interface DelegationDecision {
  version: number;
  mode: 'AUTO' | 'ASK' | 'OFF';
  delegate: boolean;
  pending_approval?: boolean;
  reason: string;
  expected_benefit: string;
  parallelizable: boolean;
  coordination_cost: string;
  context_transfer_cost: string;
  write_collision_risk: string;
  shared_contract_risk: string;
  workstreams?: Array<{
    id: string;
    goal: string;
    preferred_roles?: string[];
    acceptable_roles?: string[];
    required_capabilities?: string[];
    preferred_capabilities?: string[];
    domains?: string[];
    dependencies?: string[];
    owned_paths?: string[];
    shared_contracts?: string[];
    parallel_group?: string;
  }>;
  verification_strategy: string;
  lead_agent_id?: string;
  max_fan_out: number;
  created_at: string;
}

export interface PackageRun {
  id: string;
  package_id: string;
  phase_id?: string;
  phase_type?: string;
  title: string;
  goal?: string;
  state: string;
  attempt: number;
  assigned_agent: string;
  provider?: string;
  profile?: string;
  desired_provider?: string;
  desired_profile?: string;
  routing_decision?: string;
  review_required?: boolean;
  assigned_runtime?: string;
  workspace?: string;
  prompt_version_id?: string;
  dependencies?: string[];
  skill_ids?: string[];
  maestro_skills?: string[];
  parallel_group?: string;
  verifications?: VerificationResult[];
  verdicts?: ReviewVerdict[];
  review_findings?: ReviewFinding[];
  error_message?: string;
  remediation_context?: string;
  context_capsule?: ContextCapsule;
  work_receipt?: WorkReceipt;
  dynamic_work?: boolean;
  parent_package_id?: string;
  coordination_request_id?: string;
  started_at: string;
  finished_at?: string;
}

export interface ReviewFinding {
  id: string;
  severity?: string;
  category?: string;
  file?: string;
  location?: string;
  description: string;
  rationale?: string;
  required_action?: string;
  status: string;
}

export interface PhaseRun {
  id: string;
  type?: string;
  state: string;
  attempt: number;
  package_ids?: string[];
  agent_ids?: string[];
  routing_reasons?: string[];
  updated_at: string;
}

export type CoordinationOperation =
  'NEED_INFO' | 'DELEGATE' | 'ARTIFACT_REQUEST' | 'REVIEW_REQUEST';

export type CoordinationStatus = 'PENDING' | 'RESOLVED' | 'REJECTED' | 'FAILED' | 'CANCELED';

export interface CoordinationRequest {
  id: string;
  run_id: string;
  source_package_id: string;
  source_agent_id?: string;
  operation: CoordinationOperation;
  subject: string;
  objective?: string;
  preferred_role?: string;
  preferred_provider?: string;
  required_capabilities?: string[];
  artifact_refs?: string[];
  status: CoordinationStatus;
  child_package_id?: string;
  materialized_at?: string;
  created_at: string;
  resolved_at?: string;
  result_receipt_id?: string;
  resolution_summary?: string;
}

export interface RunCoordinationResult {
  request: CoordinationRequest;
  child?: PackageRun;
}

export interface WorkPackage {
  id: string;
  title: string;
  goal: string;
  priority: 'CRITICAL' | 'HIGH' | 'NORMAL' | 'LOW';
  status:
    | 'PENDING'
    | 'READY'
    | 'ALLOCATING'
    | 'COMPILING'
    | 'EXECUTING'
    | 'TESTING'
    | 'REVIEWING'
    | 'VERIFIED'
    | 'FAILED'
    | 'BLOCKED';
  dependencies: string[];
  parallel_group?: string;
  role: string;
  phase_type?: string;
  review_required?: boolean;
  task_requirements?: string;
  agent_allocation?: string;
  assignment_strategy?: 'EXISTING' | 'CREATE' | 'AUTO';
  resource_policy?: 'BALANCED' | 'PRESERVE_QUOTA' | 'PREFER_PROVIDER' | 'MANUAL' | string;
  provider?: string;
  profile?: string;
  skill_ids?: string[];
  maestro_gates?: string[];
  maestro_skills?: string[];
  relevant_paths?: string[];
  acceptance_criteria: string[];
  verification_requirements?: string[];
  shared_artifacts?: string[];
  entry_criteria?: string[];
  exit_criteria?: string[];
  required_evidence?: string[];
  handoff?: string;
  compiled_prompt?: string;
}

export interface PlanPhase {
  id: string;
  title: string;
  description?: string;
  order: number;
  type?: string;
  required_role?: string;
  failure_policy?: string;
  packages: WorkPackage[];
}

export interface WorkPlan {
  id: string;
  project_id: string;
  mission_id?: string;
  title: string;
  description: string;
  status: 'DRAFT' | 'READY' | 'EXECUTING' | 'COMPLETED' | 'BLOCKED';
  current_revision: number;
  phases: PlanPhase[];
  structured_facts?: Record<string, string>;
  created_at: string;
  updated_at: string;
}

export interface FlowLeaderPolicy {
  role: string;
  preferred_agent_id?: string;
  strategy: 'EXISTING' | 'AUTO' | 'NONE';
  skills?: string[];
  why?: string;
}

export interface HumanIntervention {
  id: string;
  reason_code: string;
  summary: string;
  question: string;
  context: string;
  recommended_actions: string[];
  impact: string;
  mission_id: string;
  task_id: string;
  source: string;
  timestamp: string;
  version: number;
  scope: string;
  options: InterventionOption[];
  resolved: boolean;
  resolved_at?: string;
  resolution_decision?: string;
  resolution_chosen?: string;
  resolution?: InterventionResolution;
}

export interface InterventionOption {
  id: string;
  operation: 'RETRY_SAFE_PACKAGE' | 'REPLAN_PACKAGE' | 'CONFIRM_EXTERNAL_COMPLETION' | string;
  label: string;
  package_id?: string;
}

export interface InterventionResolution {
  intervention_id: string;
  version: number;
  option_id: string;
  idempotency_key: string;
  resolved_at: string;
  resolved_by: string;
}

export type AttentionLevel = 'IGNORE' | 'IN_APP' | 'NOTIFY' | 'REQUIRE_USER';

export interface AttentionItem {
  mission_id: string;
  project_id: string;
  state: string;
  level: AttentionLevel;
  reason_code: string;
  summary: string;
  question: string;
  impact: string;
  recommended_actions: string[];
  age_seconds: number;
  intervention?: HumanIntervention;
}

export interface AttentionGroup {
  needs_you: AttentionItem[];
  completed: AttentionItem[];
  failed: AttentionItem[];
  all_items: AttentionItem[];
  total_needs: number;
}

export interface AutonomyContract {
  max_retries: number;
  max_total_iterations: number;
  max_no_progress: number;
  package_timeout_seconds: number;
  auto_remediate: boolean;
  require_verification: boolean;
  disallow_destructive_git: boolean;
  allowed_file_patterns?: string[];
  verification_commands?: string[];
  global_verification_commands?: string[];
  escalate_on_failure: boolean;
  allow_tool_auto_approval: boolean;
  allow_git_push: boolean;
  allow_deploy: boolean;
  allow_external_network: boolean;
  allow_secret_access: boolean;
  allow_paid_services: boolean;
}

export interface VerificationResult {
  command: string;
  passed: boolean;
  exit_code: number;
  output_snippet: string;
  duration_ms: number;
  verified_at: string;
}

export interface ReviewVerdict {
  approved: boolean;
  reviewer_agent_id: string;
  findings?: string[];
  structured_findings?: ReviewFinding[];
  remediation_tips?: string[];
  reviewed_at: string;
}

export interface WorkReceipt {
  id: string;
  run_id: string;
  step_id: string;
  status: 'VERIFIED' | 'FAILED' | string;
  summary: string;
  changed_files: string[];
  commands: string[];
  tests: VerificationResult[];
  decisions: string[];
  artifacts: string[];
  remaining_issues: string[];
  handoff?: string;
  verification: VerificationResult[];
  agent_id: string;
  base_revision: string;
  result_revision: string;
  started_at: string;
  completed_at: string;
}

export interface ContextCapsule {
  id: string;
  run_id: string;
  project_id: string;
  flow_id: string;
  flow_revision: number;
  branch: string;
  head: string;
  dirty_fingerprint: string;
  step: {
    id: string;
    title: string;
    goal: string;
    role: string;
    dependencies: string[];
    assignment_strategy: string;
    verification_requirements: string[];
  };
  relevant_paths: string[];
  durable_context_refs: string[];
  dependency_receipts: WorkReceipt[];
  skill_ids: string[];
  maestro_skills: string[];
  acceptance_criteria: string[];
  constraints: string[];
  created_at: string;
}

export interface FlowRunEvidence {
  run_id: string;
  capsules: ContextCapsule[];
  receipts: WorkReceipt[];
}

export interface DeliveryIntegrationEvidence {
  workspace?: string;
  base_revision?: string;
  package_ids?: string[];
  changed_files?: string[];
  integrated_at: string;
}

export interface MissionEvidenceBundle {
  version: string;
  run_id: string;
  plan_id?: string;
  plan_revision?: number;
  state: string;
  generated_at: string;
  completed_verified: boolean;
  evidence_complete: boolean;
  missing_evidence?: string[];
  integrated_workspace?: string;
  delivery_integration?: DeliveryIntegrationEvidence;
  capsules: ContextCapsule[];
  receipts: WorkReceipt[];
  routing: RoutingDecisionReport;
  validation: ValidationEvidenceReport;
  global_verifications?: VerificationResult[];
}

export interface ValidationEvidenceStream {
  id: string;
  project_id?: string;
  name: string;
  created_at: string;
  last_sequence: number;
  last_hash?: string;
}

export interface ValidationEvidenceEntry {
  id: string;
  stream_id: string;
  sequence: number;
  git_sha?: string;
  identity_digest?: string;
  repository_state?: string;
  environment_json?: string;
  provider?: string;
  profile?: string;
  model?: string;
  scenario: string;
  command_display?: string;
  outcome: string;
  confidence: string;
  exit_code?: number;
  duration_ms?: number;
  evidence_json?: string;
  previous_hash?: string;
  entry_hash: string;
  created_at: string;
}

export interface ValidationEvidenceReport {
  run_id: string;
  plan_id?: string;
  plan_revision?: number;
  stream?: ValidationEvidenceStream;
  entries: ValidationEvidenceEntry[];
  chain_verified: boolean;
}

export interface RuntimeControlResponse {
  ok: boolean;
  code: string;
  runtime_id: string;
  action: 'stop' | 'detach' | 'handoff' | 'continue' | string;
  state?: string;
  message?: string;
  correlation_id?: string;
  status?: string;
}

export interface NexusSystemInfo {
  apiVersion: string;
  serverVersion: string;
  build: Record<string, string>;
  providers: string[];
  capabilities: Record<string, Record<string, unknown>>;
}

export interface EventRecord {
  id: string;
  correlation_id?: string;
  runtime_id: string;
  provider_id?: string;
  profile_id?: string;
  provider?: string;
  profile?: string;
  type: string;
  summary: string;
  data: Record<string, unknown>;
  timestamp: string;
}

/* ---- IAPro Nexus domain types ---- */

export interface SchedulerRejection {
  account: ProviderAccount;
  reason: string;
}

export interface SchedulerDecision {
  selected: ProviderAccount;
  policy: string;
  reason: string;
  score: number;
  rejected?: SchedulerRejection[];
  explain_path: string[];
}

export interface ResourceAllocation {
  decision: SchedulerDecision;
  impact?: ConfigImpact;
  persisted: boolean;
}

// Capacity Monitor (revival T6): read-model projection of the SQLite monitor.
// Unknown stays unknown: an empty resources list means "no data", never healthy.

export interface ResourceCandidate {
  account: ProviderAccount;
  rank: number;
  total_score: number;
  confidence: 'LIVE' | 'CACHED' | 'ESTIMATED' | 'UNKNOWN' | 'UNSUPPORTED' | 'RATE_LIMITED';
  score_breakdown: Record<string, number>;
  pros: string[];
  cons: string[];
  eligible: boolean;
  rejection_reason?: string;
  provider_priority?: number;
}

export interface RecommendationResult {
  requirements: {
    task_kind: string;
    role: string;
    current_provider?: string;
    prefer_provider?: string;
  };
  policy: string;
  recommended?: ResourceCandidate;
  candidates: ResourceCandidate[];
  explanation: string;
}

export interface StepExecutionPreview {
  package_id: string;
  phase_id?: string;
  title: string;
  role?: string;
  assignment_strategy?: string;
  resource_policy?: string;
  task_requirements: ExecutionTaskRequirements;
  agent: PreviewAgentDecision;
  resource: PreviewResourceDecision;
  model: PreviewModelDecision;
  skills?: string[];
  acceptance_criteria?: string[];
  verification_requirements?: string[];
  required_evidence?: string[];
  ready: boolean;
  blockers?: string[];
}

export interface PlanExecutionPreview {
  plan_id: string;
  project_id: string;
  revision: number;
  ready: boolean;
  steps: StepExecutionPreview[];
  blockers?: string[];
  warnings?: string[];
  cost_budget: CostBudgetAssessment;
  cost_anomaly: CostAnomalyAssessment;
  cost_confirmation_required: boolean;
  cost_confirmation_token?: string;
  approval_token: string;
  generated_at: string;
}
