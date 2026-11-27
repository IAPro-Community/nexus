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

/* ---- Maestro skills, catalog and submission contracts ---- */

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

export interface MaestroComposedTask {
  id: string;
  title: string;
  objective: string;
  type?: string;
  dependsOn: string[];
  acceptanceCriteria: string[];
  requiredCapabilities?: string[];
}

export interface MaestroComposePreviewResult {
  schemaVersion: 1;
  authority: 'MAESTRO';
  source: 'maestro-semantic-planner';
  providerId: 'codex';
  reviewOnly: true;
  executable: false;
  requiresApproval: true;
  artifact: { id: string; version: number; sha256: string };
  context: { contentDigest: string };
  validation: {
    valid: boolean;
    blockers?: Array<{ code?: string; message?: string }>;
    warnings?: Array<{ code?: string; message?: string }>;
  };
  tasks: MaestroComposedTask[] | null;
}

export type MaestroRoutingProvider = 'agy' | 'opencode' | 'codex' | 'claude';

export interface MaestroWorkAssignmentInput {
  taskId: string;
  providerId: MaestroRoutingProvider;
  model?: string;
  sandbox: 'none' | 'enabled' | 'read-only' | 'workspace-write';
  worktreePath: string;
}

export type MaestroSubmissionStatus =
  | 'PENDING_WORKER'
  | 'CLAIMED'
  | 'DISPATCHED'
  | 'CANCEL_REQUESTED'
  | 'RECONCILIATION_REQUIRED'
  | 'COMPLETED'
  | 'FAILED'
  | 'BLOCKED'
  | 'TIMED_OUT'
  | 'CANCELLED';

// Durable admission receipt. A queued submission is not an execution promise:
// dispatch stays Maestro-owned and observable via the submissions snapshot.

export interface MaestroRunAck {
  schemaVersion: 1;
  submissionId: string;
  requestDigest: string;
  manifestId: string;
  manifestDigest: string;
  approvalDigest: string;
  status: MaestroSubmissionStatus;
  replayed: boolean;
  workspaceFingerprint: string;
  branch: string;
  headCommit: string;
}

// Optional measured-usage evidence block attached by the Maestro run store.
// Null means "not measured" and is never coerced to zero by the Nexus.

export interface MaestroSubmissionUsage {
  schemaVersion: 1;
  evidence: 'RUN_STORE_OBSERVED';
  tokensInput: number | null;
  tokensOutput: number | null;
  tokensCached: number | null;
  tokensReasoning: number | null;
  durationMs: number | null;
}

export interface MaestroSubmissionStatusRow {
  schemaVersion: 1;
  authority: 'MAESTRO';
  owner: 'MAESTRO';
  submissionId: string;
  status: MaestroSubmissionStatus;
  runId: string;
  manifestId: string;
  taskCount: number;
  providers: string[];
  createdAt: string;
  updatedAt: string;
  resultEvidence: 'RUN_STORE_OBSERVED';
  usage?: MaestroSubmissionUsage;
}

export interface MaestroSubmissionSnapshot {
  schemaVersion: 1;
  authority: 'MAESTRO';
  owner: 'MAESTRO';
  delivery: 'SNAPSHOT_POLL';
  replayable: false;
  projectId: string;
  cursor: string;
  changed: boolean;
  total: number;
  offset: number;
  limit: number;
  nextOffset: number | null;
  submissions: MaestroSubmissionStatusRow[];
}

export interface MaestroRoutePreviewResult {
  schemaVersion: 1;
  authority: 'MAESTRO';
  previewOnly: true;
  executable: false;
  approvalRequired: true;
  ready: boolean;
  plan: {
    authority: 'MAESTRO';
    validation?: { valid: boolean; blockers?: Array<{ code?: string; message?: string }> };
  };
  assignments: Array<{
    taskId: string;
    providerId: MaestroRoutingProvider;
    model: string;
    workspaceId?: string;
    status: 'READY_FOR_REVIEW' | 'BLOCKED';
    code?: string;
  }>;
  blockers: Array<{
    code: string;
    taskId?: string;
    providerId?: string;
    message?: string;
  }>;
}
