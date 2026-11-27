import type { Agent, RuntimeGeneration } from './core';

/* ---- Agent detail, configuration and execution-role contracts ---- */

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

export interface AgentRevision {
  id: string;
  agent_id: string;
  revision: number;
  config: string;
  created_at: string;
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

export interface MissionAssignment {
  id: string;
  mission_id: string;
  task_id: string;
  agent_id: string;
  status: string;
  assigned_at: string;
  completed_at?: string;
}

export interface MissionAssignmentInput {
  task_id: string;
  agent_id: string;
}
