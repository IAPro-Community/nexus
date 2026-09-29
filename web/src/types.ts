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
  continuity?: string;
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
  registration_state?: string;
  binary_path?: string;
  control_level: string;
  capabilities: EffectiveCapabilities;
}

export interface NexusSystemInfo {
  apiVersion: string;
  serverVersion: string;
  build: Record<string, string>;
  providers: string[];
  capabilities: Record<string, Record<string, unknown>>;
}

export interface ProfileInfo {
  name: string;
  provider: string;
  account_email?: string;
  plan?: string;
  authenticated: boolean;
  is_default: boolean;
}

export interface AccountScope {
  provider_id: string;
  profile_id: string;
  account_id: string;
  identity_version: string;
  credential_scope?: string;
}

export interface RegisteredProfile {
  provider: string;
  name: string;
  created_at?: string;
  account_scope?: AccountScope;
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
  network_error?: boolean;
  health: string;
  last_checked: string;
  cooldown_until?: string;
  account_scope?: {
    provider_id?: string;
    account_id?: string;
    identity_version?: string;
  };
  avail_reasons?: {
    exhausted_windows?: string[];
    rate_limited?: boolean;
    unknown_quota?: boolean;
    auth_required?: boolean;
    all_ok?: boolean;
  };
  eligibility?: {
    eligible: boolean;
    reason: string;
    confidence: string;
    message?: string;
  };
  quota_view?: {
    status?: string;
    source?: string;
    plan?: string;
    account?: string;
    fetched_at?: string;
    available?: boolean;
    avail_reasons?: {
      exhausted_windows?: string[];
      rate_limited?: boolean;
      unknown_quota?: boolean;
      auth_required?: boolean;
      all_ok?: boolean;
    };
    model_groups?: Array<{
      key?: string;
      name?: string;
      windows?: Array<{
        kind?: string;
        label?: string;
        remaining?: number;
        reset_desc?: string;
        status?: string;
        bar?: string;
      }>;
    }>;
  };
}

export interface BalanceModelGroup {
  key?: string;
  name: string;
  remaining: number;
  known: boolean;
  available: boolean;
}

export interface BalanceEntity {
  id: string;
  provider: string;
  profile: string;
  display_name: string;
  usage_status: 'LIVE' | 'CACHED' | 'ESTIMATED' | 'RATE_LIMITED' | string;
  usage_source?: string;
  quota_remaining: number;
  available: boolean;
  health: string;
  eligibility?: ProviderAccount['eligibility'];
  model_groups?: BalanceModelGroup[];
}

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
export type CapacityMonitorState =
  'UNKNOWN' | 'HEALTHY' | 'WARNING' | 'CRITICAL' | 'EXHAUSTED' | 'RECOVERED';

export interface CapacityResourceStatus {
  key: string;
  provider: string;
  profile: string;
  quota_group: string;
  window_kind: string;
  state: CapacityMonitorState | string;
  fingerprint?: string;
  reason?: string;
  transitioned_at: string;
  last_observed_at: string;
  remaining_percent?: number | null;
  reset_description?: string;
  acked: boolean;
  snoozed_until?: string | null;
}

export interface CapacityPreferences {
  enabled: boolean;
  background_enabled: boolean;
  warning_threshold: number;
  critical_threshold: number;
  exhausted_threshold: number;
  recovery_threshold: number;
  reminder_interval_seconds: number;
  quiet_hours_enabled: boolean;
  quiet_hours_start_utc: string;
  quiet_hours_end_utc: string;
  exhausted_bypass_quiet_hours: boolean;
  browser_notifications_enabled: boolean;
  native_notifications_enabled: boolean;
  lock_screen_safe_copy: boolean;
  updated_at: string;
}

export interface CapacityStatus {
  enabled: boolean;
  monitored_resources: number;
  resources: CapacityResourceStatus[];
  lease_owner?: string;
  lease_expires_at?: string;
  preferences: CapacityPreferences;
}

export interface CapacityHistoryEvent {
  id: string;
  fingerprint: string;
  event_type: string;
  provider: string;
  profile: string;
  quota_group: string;
  window_kind: string;
  severity: string;
  payload_json: string;
  emitted_at: string;
  acked: boolean;
  snoozed_until?: string | null;
}

export interface CapacityAlertState {
  fingerprint: string;
  acked: boolean;
  snoozed_until?: string | null;
  updated_at: string;
}

export type CostEvidenceState = 'EMPTY' | 'ATTRIBUTION_ONLY' | 'PARTIAL' | 'MEASURED';

export interface CostAggregate {
  observation_count: number;
  known_cost_observations: number;
  unknown_cost_observations: number;
  known_token_observations: number;
  cost_microusd: number;
  input_tokens: number;
  cached_input_tokens: number;
  output_tokens: number;
  reasoning_tokens: number;
  duration_ms: number;
}

export interface ProviderCostAggregate extends CostAggregate {
  provider: string;
  profile: string;
}

export interface CostPreferences {
  enabled: boolean;
  window_days: number;
  soft_limit_microusd: number;
  hard_limit_microusd: number;
  emergency_limit_microusd: number;
  anomaly_multiplier: number;
  anomaly_minimum_samples: number;
  anomaly_min_cost_microusd: number;
  updated_at: string;
}

export type CostBudgetLevel = 'DISABLED' | 'UNKNOWN' | 'OK' | 'SOFT' | 'HARD' | 'EMERGENCY';

export interface CostBudgetAssessment {
  level: CostBudgetLevel;
  known_cost_microusd: number;
  evidence: CostEvidenceState;
  active_limit_microusd?: number;
  requires_confirmation: boolean;
  message: string;
}

export type CostAnomalyStatus = 'UNKNOWN' | 'INSUFFICIENT' | 'NORMAL' | 'ANOMALY';

export interface CostAnomalyAssessment {
  status: CostAnomalyStatus;
  triggered: boolean;
  provider?: string;
  profile?: string;
  cost_kind?: CostObservation['cost_kind'];
  latest_cost_microusd?: number;
  baseline_median_microusd?: number;
  threshold_microusd?: number;
  historical_samples: number;
  observed_at?: string;
  message: string;
}

export interface CostSummary {
  since: string;
  generated_at: string;
  project_id?: string;
  aggregate: CostAggregate;
  by_provider: ProviderCostAggregate[];
  cost_evidence: CostEvidenceState;
  token_evidence: CostEvidenceState;
  preferences: CostPreferences;
  budget: CostBudgetAssessment;
  anomaly: CostAnomalyAssessment;
}

export interface CostObservation {
  id: string;
  project_id?: string;
  mission_run_id?: string;
  package_id?: string;
  invocation_id?: string;
  agent_id?: string;
  runtime_generation_id?: string;
  runtime_id?: string;
  provider?: string;
  profile?: string;
  model?: string;
  branch?: string;
  workspace_kind?: string;
  input_tokens?: number;
  cached_input_tokens?: number;
  output_tokens?: number;
  reasoning_tokens?: number;
  duration_ms: number;
  cost_microusd?: number;
  cost_kind: 'PROVIDER_REPORTED' | 'API_EQUIVALENT_ESTIMATE' | 'INFERRED' | 'UNKNOWN';
  source: string;
  status?: string;
  account_scope?: string;
  observed_at: string;
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

/** Lightweight per-project agent counts for the project rail outline. */
export interface ProjectAgentSummary {
  project_id: string;
  agent_count: number;
  working_count: number;
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

export interface MissionAssignment {
  id: string;
  mission_id: string;
  task_id: string;
  agent_id: string;
  status: string;
  assigned_at: string;
  completed_at?: string;
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

export interface MissionTaskInput {
  name: string;
  description?: string;
  kind?: string;
  priority?: number;
}

export interface MissionAssignmentInput {
  task_id: string;
  agent_id: string;
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
  options?: Record<string, unknown>;
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
  agent_spec?: AgentSpec;
}

export interface AgentMode {
  id: string;
  role: string;
  default_provider?: string;
  default_execution_mode: 'Safe' | 'YOLO' | string;
  recommended_skill_ids?: string[];
  read_only_by_default?: boolean;
  agent_spec: AgentSpec;
}

export type ExecutionRoleScope = 'TURN' | 'RUNTIME';
export type AccessIntent = 'WRITE' | 'READ_ONLY' | 'VERIFY_ONLY';
export type ExecutionRoleEnforcement = 'ENFORCED' | 'BEST_EFFORT' | 'UNVERIFIED' | 'UNSUPPORTED';

export interface ExecutionRole {
  id: string;
  label: string;
  base_agent_mode_id: string;
  access_intent: AccessIntent;
  required_capabilities?: string[];
  recommended_skill_ids?: string[];
  default_scope: ExecutionRoleScope;
  verification_policy?: {
    require_evidence?: boolean;
    require_tests?: boolean;
  };
}

export interface RuntimeExecutionRoleBinding {
  id?: string;
  runtime_generation_id: string;
  agent_id: string;
  role_id: string;
  scope: ExecutionRoleScope | string;
  activation_strategy: string;
  enforcement: ExecutionRoleEnforcement | string;
  enforcement_evidence?: string;
  enforcement_verified_at?: string;
  projection_receipt_id?: string;
  activated_by?: string;
  activated_at?: string;
}

export interface RuntimeExecutionRoleView {
  binding: RuntimeExecutionRoleBinding;
  role: ExecutionRole;
  persisted: boolean;
}

export interface AgentSpec {
  role?: string;
  instructions?: string[];
  responsibilities?: string[];
  capabilities?: string[];
  constraints?: string[];
  domains?: string[];
  strengths?: string[];
  tags?: string[];
  verification_policy?: {
    require_evidence?: boolean;
    require_tests?: boolean;
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

/**
 * Static project-intelligence contracts. These types intentionally stay
 * transport-agnostic until the Core scan/attempt service exposes a persisted
 * API. They mirror the immutable snapshot produced by contextsnapshot and
 * keep provenance visible to the UI rather than reducing facts to strings.
 */
export type ProjectContextSnapshotCompleteness = 'COMPLETE' | 'PARTIAL';

export type ProjectFactValueType = 'STRING' | 'LIST' | 'MAP' | 'BOOL';
export type ProjectFactBasis = 'OBSERVED' | 'DERIVED';
export type ProjectFactConfidence = 'HIGH' | 'MEDIUM' | 'LOW';

export type ProjectCodeIdentityState = 'CLEAN' | 'DIRTY' | 'UNBORN' | 'NON_GIT' | 'INCOMPLETE';

export interface ProjectCodeIdentity {
  repository_kind: string;
  head_sha?: string;
  branch?: string;
  state: ProjectCodeIdentityState;
  index_digest?: string;
  worktree_digest?: string;
  untracked_digest?: string;
  submodules_digest?: string;
  identity_digest: string;
  complete: boolean;
  warnings?: string[];
}

export interface ProjectFactProvenance {
  source: string;
  locator?: string;
  extractor: string;
  digest?: string;
  observed_at: string;
}

export interface ProjectFact {
  category: string;
  key: string;
  value_type: ProjectFactValueType;
  value: unknown;
  basis: ProjectFactBasis;
  confidence: ProjectFactConfidence;
  observed_at: string;
  provenance: ProjectFactProvenance[];
}

export interface ProjectContextSnapshot {
  id?: string;
  project_id?: string;
  canonical_path?: string;
  scanner_version: string;
  identity: ProjectCodeIdentity;
  completeness: ProjectContextSnapshotCompleteness;
  facts: ProjectFact[];
  warnings?: string[];
  observed_at: string;
}

export type ProjectIntelligenceScanState =
  'QUEUED' | 'RUNNING' | 'SUCCEEDED' | 'FAILED' | 'CANCELED';

export interface ProjectIntelligenceScan {
  id: string;
  project_id: string;
  identity_digest: string;
  state: ProjectIntelligenceScanState;
  scanner_version: string;
  requested_at: string;
  started_at?: string;
  finished_at?: string;
  error?: string;
}

export interface ProjectIntelligenceView {
  project_id: string;
  identity: ProjectCodeIdentity;
  current_snapshot?: ProjectContextSnapshot;
  current_scan?: ProjectIntelligenceScan;
}

export interface MaestroSkill {
  id: string;
  name?: string;
  description?: string;
  category?: string;
  risk?: string;
  triggers?: string[];
  aliases?: string[];
  prompt?: string;
}

export interface CatalogSkill extends MaestroSkill {
  source: string;
  availability: string;
  activation_mode?: string;
  copies: number;
  contract?: string;
}

export interface MaestroCatalog {
  operational: CatalogSkill[];
  library: CatalogSkill[];
  counts: { operational: number; library: number; copies: number };
}

export interface MaestroStatus {
  available: boolean;
  mode: string;
  capabilities?: {
    version: string;
    modes: string[];
    skills: MaestroSkill[];
    gates?: string[];
    processes?: string[];
  };
  last_check?: string;
  error?: string;
}

export interface MaestroRecommendation {
  id: string;
  type: string;
  title: string;
  description: string;
  apply: string;
  why: string;
  risk: string;
  gates?: string[];
  skills?: string[];
  verify?: string;
  metadata?: Record<string, unknown>;
}

export interface MaestroAdvice {
  version?: string;
  mode?: string;
  required?: MaestroRecommendation[];
  recommended?: MaestroRecommendation[];
  optional?: MaestroRecommendation[];
  explanation?: string;
  degraded?: boolean;
}

export interface SkillSyncPreview {
  id: string;
  dry_run: boolean;
  tool: string;
  roots: string[];
  skills: string[];
  command: string;
  output?: string;
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
    }>;
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

export interface MissionRun {
  id: string;
  plan_id: string;
  plan_revision: number;
  execution_snapshot_id?: string;
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

export interface ExecutionTaskRequirements {
  task_kind: string;
  role: string;
  preferred_roles?: string[];
  acceptable_roles?: string[];
  domains?: string[];
  required_capabilities?: string[];
  preferred_capabilities?: string[];
  desired_strengths?: string[];
  constraints?: string[];
  estimated_complexity?: string;
  estimated_tokens?: number;
  requires_decomposition?: boolean;
  confidence?: string;
  source?: string;
  prefer_provider?: string;
  project_policy?: string;
}

export interface PreviewAgentDecision {
  agent_id?: string;
  name?: string;
  role?: string;
  score?: number;
  confidence?: string;
  reason: string;
  will_create?: boolean;
  candidate_count?: number;
}

export interface PreviewResourceDecision {
  account_id?: string;
  provider?: string;
  profile?: string;
  display_name?: string;
  policy?: string;
  score?: number;
  confidence?: string;
  reason: string;
  health?: string;
  freshness?: string;
  quota_remaining?: number;
  candidate_count?: number;
}

export interface PreviewModelDecision {
  model?: string;
  fallback?: boolean;
  reason: string;
  evidence: string;
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
  flow: import('./features/work/flowModel').FlowDraftModel;
  reasoning: string;
  maestro_advice?: string;
}

export interface TunnelStatus {
  active: boolean;
  url?: string;
  bootstrap_url?: string;
  cloudflared_installed: boolean;
  cloudflared_version?: string;
  error?: string;
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

export interface TelemetryStats {
  period_days: number;
  total_sessions: number;
  total_fallbacks: number;
  total_rate_limits: number;
  by_provider: Record<string, number>;
  by_profile: Record<string, number>;
  rate_limits_by_profile: Record<string, number>;
}

export interface TelemetryHistoryEvent {
  timestamp: string;
  type: string;
  provider_id: string;
  profile_id: string;
  workspace?: string;
  duration_ms?: number;
  reason?: string;
  failure_kind?: string;
  fallback_profile?: string;
  account_scope?: {
    provider_id: string;
    profile_id: string;
    account_id: string;
    identity_version: string;
  };
}

export type AgentProjectionCapabilityStatus =
  'SUPPORTED' | 'PARTIAL' | 'UNSUPPORTED' | 'UNKNOWN' | 'NOT_TESTED';

export interface AgentProjectionCapabilityEvidence {
  status: AgentProjectionCapabilityStatus | string;
  tested: boolean;
  evidence?: string;
  provider_version?: string;
  verified_at?: string;
}

export interface AgentProjectionCapabilities {
  definitions: AgentProjectionCapabilityEvidence;
  primary_agent: AgentProjectionCapabilityEvidence;
  subagents: AgentProjectionCapabilityEvidence;
  in_session_switch: AgentProjectionCapabilityEvidence;
  dynamic_reload: AgentProjectionCapabilityEvidence;
  tool_permissions: AgentProjectionCapabilityEvidence;
  model_override: AgentProjectionCapabilityEvidence;
  project_scope: AgentProjectionCapabilityEvidence;
  user_scope: AgentProjectionCapabilityEvidence;
  remote_protocol: AgentProjectionCapabilityEvidence;
}

export interface AgentProjectionConflict {
  path: string;
  kind: 'MANAGED_MODIFIED' | 'UNMANAGED' | 'UNSAFE_PATH' | string;
  detail?: string;
}

export interface AgentProjectionArtifact {
  path: string;
  mode: number;
  content_hash: string;
}

export interface AgentProjectionPlan {
  provider: string;
  profile: string;
  role_id: string;
  scope: string;
  strategy: string;
  managed_root: string;
  renderer_version: string;
  source_hash: string;
  artifacts?: AgentProjectionArtifact[];
  conflicts?: AgentProjectionConflict[];
  warnings?: string[];
}

export interface AgentProjectionPlanView {
  runtime_id: string;
  runtime_generation_id: string;
  agent_id: string;
  role_id: string;
  provider: string;
  profile: string;
  provider_version: string;
  version_source: string;
  target_kind: string;
  target_workspace: string;
  custom_command: boolean;
  capabilities: AgentProjectionCapabilities;
  plan: AgentProjectionPlan;
}

export interface AgentProjectionReceipt {
  id: string;
  provider: string;
  profile: string;
  role_id: string;
  scope: string;
  strategy: string;
  managed_root: string;
  renderer_version: string;
  source_hash: string;
  artifacts: Array<{ path: string; content_hash: string }>;
  applied_at: string;
  verified_at?: string;
}

export interface AgentProjectionDiscovery {
  provider: string;
  provider_version: string;
  target_kind: string;
  agent_name: string;
  discovered: boolean;
  status: string;
  evidence?: string;
  error?: string;
}

export interface AgentProjectionActivation {
  provider: string;
  provider_version: string;
  target_kind: string;
  agent_name: string;
  activated: boolean;
  status: string;
  evidence?: string;
  error?: string;
  external_execution: boolean;
}

export interface AgentProjectionEnforcement {
  provider: string;
  provider_version: string;
  target_kind: string;
  agent_name: string;
  enforced: boolean;
  status: string;
  tool?: string;
  canary_absent: boolean;
  evidence?: string;
  error?: string;
  external_execution: boolean;
}

export interface AgentProjectionApplyView {
  plan: AgentProjectionPlanView;
  receipt: AgentProjectionReceipt;
  discovery?: AgentProjectionDiscovery;
}

export interface AgentProjectionVerification {
  verified: boolean;
  errors?: string[];
  warnings?: string[];
}
