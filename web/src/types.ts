export interface Workspace {
  id?: string;
  name: string;
  path: string;
  provider?: string;
  profile?: string;
  is_active?: boolean;
  created_at?: string;
  last_used_at?: string;
}

export interface RuntimeSession {
  runtime_id: string;
  agent_id?: string;
  title?: string;
  provider_id?: string;
  profile_id?: string;
  provider?: string;
  profile?: string;
  provider_session_id?: string;
  workspace: string;
  pid: number;
  host_pid: number;
  state:
    | 'STARTING'
    | 'RUNNING'
    | 'HANDOFF'
    | 'WAITING'
    | 'APPROVAL'
    | 'DETACHED'
    | 'STOPPING'
    | 'STOPPED'
    | 'FAILED'
    | 'STALE';
  control_level: string;
  control_endpoint: string;
  parent_runtime_id?: string;
  handoff_type?: string;
  lineage_id?: string;
  started_at: string;
  attention_reason?: 'QUESTION' | 'APPROVAL' | 'TASK_COMPLETED' | 'WORKING' | 'IDLE' | 'ERROR';
  attention_context?: string;
  prompt_kind?: 'yn' | 'choice' | 'free_text' | 'none' | '';
  attention_kind?: 'needs_user' | 'working' | 'completed' | 'error' | 'idle' | '';
  attention_fingerprint?: string;
  project_id?: string;
  project_name?: string;
  last_task_summary?: string;
  dynamic_title?: string;
}

export interface CapabilityEvidence {
  status: 'SUPPORTED' | 'PARTIAL' | 'UNSUPPORTED' | 'UNKNOWN';
  provider_version?: string;
  mechanism?: string;
  reason?: string;
  tested: boolean;
}

export interface EffectiveCapabilities {
  process: CapabilityEvidence;
  terminal: CapabilityEvidence;
  attach: CapabilityEvidence;
  structured_events: CapabilityEvidence;
  sessions: CapabilityEvidence;
  resume: CapabilityEvidence;
  fork: CapabilityEvidence;
  submit_prompt: CapabilityEvidence;
  cancel_turn: CapabilityEvidence;
  approvals: CapabilityEvidence;
  native_ui_attach: CapabilityEvidence;
  headless: CapabilityEvidence;
  slash_control: CapabilityEvidence;
  control_level: string;
}

export interface ProviderInfo {
  id: string;
  installed: boolean;
  version: string;
  control_level: string;
  capabilities: EffectiveCapabilities;
}

export interface ProfileInfo {
  name: string;
  provider: string;
  account_email?: string;
  plan?: string;
  authenticated: boolean;
  is_default: boolean;
}

export interface ProviderAccount {
  id: string;
  provider: string;
  profile: string;
  display_name: string;
  authenticated: boolean;
  is_default: boolean;
  available: boolean;
  capabilities?: Record<string, string>;
  quota_remaining: number;
  quota_total: number;
  rate_limited: boolean;
  health: string;
  last_checked: string;
  avail_reasons?: {
    exhausted_windows?: string[];
    rate_limited?: boolean;
    unknown_quota?: boolean;
    auth_required?: boolean;
    all_ok?: boolean;
  };
  quota_view?: {
    status?: string;
    plan?: string;
    account?: string;
    model_groups?: Array<{
      key?: string;
      name?: string;
      windows?: Array<{
        kind?: string;
        label?: string;
        remaining?: number;
        reset_desc?: string;
        status?: string;
      }>;
    }>;
  };
}

export interface EventRecord {
  id: string;
  runtime_id: string;
  provider_id?: string;
  profile_id?: string;
  provider?: string;
  profile?: string;
  type: string;
  summary: string;
  data: Record<string, any>;
  timestamp: string;
}

/* ---- IAPro Nexus domain types ---- */

export interface Project {
  id: string;
  name: string;
  slug: string;
  canonical_path: string;
  repo_remote: string;
  repo_url: string;
  default_branch: string;
  maestro_mode: 'OFF' | 'ASSIST' | 'ORCHESTRATE';
  resource_policy: string;
  default_isolation: string;
  settings: string;
  created_at: string;
  updated_at: string;
  last_opened_at?: string;
}

export interface Agent {
  id: string;
  project_id: string;
  name: string;
  role: string;
  status: string;
  current_revision_id?: string;
  continuity_status: string;
  created_at: string;
  updated_at: string;
  last_started_at?: string;
}

export interface RuntimeGeneration {
  id: string;
  agent_id: string;
  revision_id: string;
  runtime_id: string;
  provider: string;
  profile: string;
  provider_session: string;
  continuity: string;
  started_at: string;
  stopped_at?: string;
  state: string;
}

export interface LineageEntry {
  id: string;
  agent_id: string;
  relation: string;
  source_runtime: string;
  source_session: string;
  target_runtime: string;
  target_session: string;
  checkpoint_id: string;
  created_at: string;
}

export interface AgentRevision {
  id: string;
  agent_id: string;
  revision: number;
  config: string;
  created_at: string;
}

export interface AgentDetail {
  agent: Agent;
  generations: RuntimeGeneration[];
  lineage: LineageEntry[];
  revisions: AgentRevision[];
}

/* ---- Gate 3: Agent Configuration ---- */

export interface AgentConfig {
  provider: string;
  profile: string;
  model?: string;
  options?: Record<string, any>;
  workspace?: string;
  isolation?: string;
  maestro_mode?: string;
  continuity_policy?: string;
  environment?: Record<string, string>;
  allocation?: {
    prefer_provider?: string;
    max_concurrent?: number;
    quota_preserve?: boolean;
    cooldown_seconds?: number;
  };
}

export interface ConfigImpact {
  mode: 'LIVE_SAME_RUNTIME' | 'RESTART_RUNTIME' | 'NEW_SESSION';
  changed_fields: string[];
  requires_restart: boolean;
  requires_new_session: boolean;
  current_config?: AgentConfig;
  proposed_config?: AgentConfig;
  warnings?: string[];
}

/* ---- OS Filesystem & Deep Integration ---- */

export interface FSEntry {
  name: string;
  path: string;
  is_dir: boolean;
  is_git: boolean;
  tech: string[];
  mod_time: string;
  size_bytes: number;
  child_count?: number;
  permissions: string;
}

export interface FSBookmark {
  label: string;
  path: string;
  icon: string;
}

export interface FSBrowseResult {
  current_path: string;
  parent_path: string;
  breadcrumbs: string[];
  entries: FSEntry[];
  bookmarks: FSBookmark[];
  is_git: boolean;
  git_branch?: string;
  tech?: string[];
}

export interface FSScanResult {
  name: string;
  path: string;
  branch: string;
  tech: string[];
  mod_time: string;
  is_imported: boolean;
}

export interface FSInspectResult {
  path: string;
  exists: boolean;
  is_dir: boolean;
  is_git: boolean;
  git_branch?: string;
  git_remote?: string;
  suggested_name: string;
  tech: string[];
}

export interface GitBranchesResult {
  project_id: string;
  canonical_path: string;
  current_branch: string;
  default_branch: string;
  branches: string[];
  remote_branches: string[];
  is_clean: boolean;
  modified_count: number;
}

export interface GitCheckoutResult {
  success: boolean;
  current_branch: string;
  output?: string;
  error?: string;
}

export type ContextReadinessState = 'MISSING' | 'HYDRATING' | 'READY' | 'STALE' | 'FAILED';

export interface ContextFingerprint {
  project_id: string;
  canonical_path: string;
  branch: string;
  head: string;
  dirty_fingerprint: string;
  maestro_version: string;
}

export interface ContextReadiness {
  project_id: string;
  state: ContextReadinessState;
  current_fingerprint: ContextFingerprint;
  current_fingerprint_id: string;
  hydrated_fingerprint_id?: string;
  maestro_available: boolean;
  maestro_version: string;
  error?: string;
  hydrated_at?: string;
  updated_at?: string;
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
  task_requirements?: string;
  agent_allocation?: string;
  assignment_strategy?: 'EXISTING' | 'CREATE' | 'AUTO';
  resource_policy?: 'BALANCED' | 'PRESERVE_QUOTA' | 'PREFER_PROVIDER' | 'MANUAL' | string;
  provider?: string;
  profile?: string;
  maestro_gates?: string[];
  maestro_skills?: string[];
  relevant_paths?: string[];
  acceptance_criteria: string[];
  verification_requirements?: string[];
  shared_artifacts?: string[];
  compiled_prompt?: string;
}

export interface PlanPhase {
  id: string;
  title: string;
  description?: string;
  order: number;
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
}
export interface ComposerSkillProposal {
  session_id: string;
  skill_id: string;
  state: 'SUGGESTED' | 'ACCEPTED' | 'APPLIED' | 'REJECTED' | 'UNAVAILABLE';
  reason: string;
  applicability: string;
  risk: string;
  updated_at: string;
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
export interface PromptReadinessCheck {
  key: string;
  label: string;
  score: number;
  summary: string;
}
export interface ComposerSessionView {
  session: ComposerSession;
  brief: {
    goal: string;
    context?: string[] | Record<string, unknown>;
    constraints?: string[] | Record<string, unknown>;
    decisions?: string[];
    assumptions?: Array<string | { value?: string; status?: string; confidence?: string }>;
    alternatives?: string[];
    risks?: string[];
    success_criteria?: string[];
    open_questions?: string[];
    intent?: { archetype?: string };
    readiness?: { score: number; state: string; summary: string; checks?: PromptReadinessCheck[] };
    unknowns?: Array<{
      id: string;
      question: string;
      rationale?: string;
      severity: string;
      status: string;
      answer?: string;
      inferred_value?: string;
      confidence?: string;
    }>;
  };
  turns: ComposerTurn[];
  skills: ComposerSkillProposal[];
  artifacts?: PromptArtifact[];
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

export interface PackageRun {
  id: string;
  package_id: string;
  phase_id?: string;
  title: string;
  goal?: string;
  state: string;
  attempt: number;
  assigned_agent: string;
  assigned_runtime?: string;
  workspace?: string;
  prompt_version_id?: string;
  dependencies?: string[];
  parallel_group?: string;
  verifications?: VerificationResult[];
  verdicts?: ReviewVerdict[];
  error_message?: string;
  remediation_context?: string;
  context_capsule?: ContextCapsule;
  work_receipt?: WorkReceipt;
  started_at: string;
  finished_at?: string;
}

export interface MissionRun {
  id: string;
  plan_id: string;
  plan_revision: number;
  execution_snapshot_id?: string;
  project_id: string;
  workspace: string;
  state: string;
  contract: AutonomyContract;
  current_pkg_index: number;
  total_iterations: number;
  package_runs: PackageRun[];
  paused_reason?: string;
  started_at: string;
  updated_at: string;
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

export interface ResourceCandidate {
  account: ProviderAccount;
  rank: number;
  total_score: number;
  confidence: 'LIVE' | 'CACHED' | 'ESTIMATED' | 'UNKNOWN';
  score_breakdown: Record<string, number>;
  pros: string[];
  cons: string[];
  eligible: boolean;
  rejection_reason?: string;
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
  maestro_skills?: string[];
  simple?: boolean;
}

export interface FlowDecompositionProposal {
  title: string;
  description: string;
  archetype: string;
  flow: import('./features/work/flowModel').FlowDraftModel;
  reasoning: string;
  maestro_advice?: string;
}
