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

/* ---- Filesystem, git, release desk, context and Agent projection ---- */

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

export interface GitWorktree {
  path: string;
  head: string;
  branch: string;
  is_primary: boolean;
  detached: boolean;
  locked: boolean;
  prunable: boolean;
}

export interface ChangeCollisionWorktree {
  path: string;
  branch?: string;
  head?: string;
  changed_files: string[];
}

export interface ChangeCollision {
  file: string;
  worktrees: string[];
  confidence: 'hunk-overlap' | 'same-file';
  uncertain: boolean;
  reason: string;
}

export interface ChangeCollisionScanError {
  path: string;
  reason: string;
}

export interface ChangeCollisionsReport {
  project_id: string;
  scanned_at: string;
  worktrees: ChangeCollisionWorktree[];
  collisions: ChangeCollision[];
  uncertain_count: number;
  scan_errors?: ChangeCollisionScanError[];
}

export interface ReleaseCommitSummary {
  sha: string;
  subject: string;
  references: string[];
}

export interface ReleaseGate {
  name: string;
  verify_command: string;
  status: 'verified' | 'unverified';
  evidence?: string;
}

export interface ReleaseApproval {
  approver: string;
  approval_id: string;
  sha: string;
  approved_at: string;
}

/** One viewer share grant (workspace-sharing). Mirrors nexus.ObservationShareGrant. */

export interface ObservationShareGrant {
  id: string;
  project_id: string;
  agent_id?: string;
  role: string;
  created_at: string;
  expires_at?: string;
  revoked_at?: string;
}

export interface ReleaseDeskReport {
  branch: string;
  base_sha: string;
  head_sha: string;
  commits: ReleaseCommitSummary[];
  pull_requests: string[];
  issues: string[];
  gates: ReleaseGate[];
  approval?: ReleaseApproval;
  merge_publish_allowed: boolean;
  block_reasons: string[];
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
  worktrees?: GitWorktree[];
  worktrees_available?: boolean;
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

export interface ProjectAgentSummary {
  project_id: string;
  agent_count: number;
  working_count: number;
}

export interface TunnelStatus {
  active: boolean;
  starting?: boolean;
  url?: string;
  bootstrap_url?: string;
  cloudflared_installed: boolean;
  cloudflared_version?: string;
  /** True when the public origin survives a restart (Named Tunnel). */
  stable_url?: boolean;
  /** Configured stable hostname, when the Named Tunnel transport is active. */
  named_hostname?: string;
  /** Published private origin (tailnet or private DNS), when configured. */
  remote_origin?: string;
  error?: string;
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
