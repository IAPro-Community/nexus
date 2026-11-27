import type { EffectiveCapabilities } from './usage';

/* ---- Runtime, project and agent identities shared by every domain ---- */

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
  provider_generation?: number;
  host_pid: number;
  host_generation?: number;
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
  /** Last registry update; approximates when the current episode was raised. */
  updated_at?: string;
  attention_reason?:
    | 'QUESTION'
    | 'APPROVAL'
    | 'AUTH'
    | 'TASK_COMPLETED'
    | 'CANCELLED'
    | 'WORKING'
    | 'IDLE'
    | 'ERROR';
  attention_context?: string;
  prompt_kind?: 'yn' | 'choice' | 'free_text' | 'native_status' | 'none' | '';
  attention_kind?: 'needs_user' | 'working' | 'completed' | 'error' | 'idle' | '';
  attention_fingerprint?: string;
  attention_revision?: number;
  attention_generation?: string;
  attention_submitted_revision?: number;
  attention_source?: 'INFERRED' | 'SIGNALLED' | 'AUTHORITATIVE';
  attention_interaction_id?: string;
  attention_options?: Array<{ option_id: string; name: string }>;
  project_id?: string;
  project_name?: string;
  last_task_summary?: string;
  dynamic_title?: string;
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

export interface ProviderInfo {
  id: string;
  display_name?: string;
  short_badge?: string;
  install_guide_url?: string;
  auth_kind?: string;
  installed: boolean;
  version: string;
  registration_state?: string;
  binary_path?: string;
  /** Core opt-in entrypoints, not evidence of authenticated E2E support. */
  structured_modes?: string[];
  control_level: string;
  capabilities: EffectiveCapabilities;
  host_session?: {
    provider?: string;
    installed?: boolean;
    available: boolean;
    email?: string;
    plan?: string;
    auth_type?: string;
    account_id?: string;
  };
}

export interface ProfileInfo {
  name: string;
  provider: string;
  account_email?: string;
  plan?: string;
  authenticated: boolean;
  is_default: boolean;
}

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

export interface AgentGroup {
  id: string;
  project_id: string;
  name: string;
  task: string;
  worktree: string;
  feature: string;
  position: number;
  agent_ids: string[];
  created_at: string;
  updated_at: string;
}

export type DirectorActionKind =
  'DIRECTIVE' | 'STEER' | 'QUEUE_NEXT' | 'WAIT' | 'CANCEL' | 'INSPECT';

export type DirectorAvailability =
  'SUPPORTED' | 'PREPARE_ONLY' | 'OBSERVE_ONLY' | 'TERMINAL_ONLY' | 'UNSUPPORTED';

export interface DirectorAgentProjection {
  schema_version: number;
  execution_owner: 'NEXUS';
  agent_id: string;
  project_id: string;
  agent_status: string;
  process_state: 'LIVE' | 'NOT_LIVE' | 'UNKNOWN';
  process_evidence: string;
  observed_state?: 'unknown' | 'working' | 'blocked' | 'idle' | 'done' | 'error';
  observed_source?: string;
  observed_reason?: string;
  runtime_id?: string;
  generation_id?: string;
  provider?: string;
  profile?: string;
  workspace?: string;
  worktree?: string;
  quota?: {
    status: string;
    source: string;
    remaining_percent?: number;
    window?: string;
  };
  lineage?: Array<{
    id: string;
    relation: string;
    source_runtime: string;
    target_runtime: string;
    checkpoint_id: string;
    created_at: string;
  }>;
  actions: Array<{
    kind: DirectorActionKind;
    availability: DirectorAvailability;
    reason: string;
  }>;
  recent_events?: Array<{
    id: string;
    kind: string;
    correlation_id?: string;
    source?: string;
    runtime_generation_id?: string;
    ts: string;
  }>;
}

export interface DirectorFleetProjection {
  schema_version: number;
  execution_owner: 'NEXUS';
  project_id: string;
  total: number;
  truncated: boolean;
  agents: DirectorAgentProjection[];
}
