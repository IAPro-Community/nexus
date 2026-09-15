/* Nexus product API client (projects, agents, layouts, config). */

import {
  Project,
  Agent,
  AgentDetail,
  RuntimeSession,
  AgentConfig,
  ConfigImpact,
  AgentRevision,
  MaestroCatalog,
  MaestroAdvice,
  MaestroStatus,
  SkillSyncPreview,
  NexusSystemInfo,
  ProviderAccount,
  ResourceAllocation,
  ProviderInfo,
  RegisteredProfile,
} from '../types';
import { request, setCSRFToken } from '../api';
import { normalizeWorkPlan } from './workPlan';

export type SystemDoctorCheck = {
  id: string;
  status: 'PASS' | 'WARN' | 'FAIL' | 'SKIPPED' | string;
  summary: string;
  remediation?: string;
};

export type SystemDoctorReport = {
  schema: string;
  generated_at: string;
  version: string;
  os: string;
  arch: string;
  checks: SystemDoctorCheck[];
  providers: Record<
    string,
    { installed: boolean; version?: string; binary_path?: string; error?: string }
  >;
  credentials: { status: string; mechanism: string; reason?: string };
};

export type DurableActivityEvent = import('../app/activityModel').DurableActivityEvent;

export type NexusErrorPayload = {
  error?: string;
  code?: string;
  [key: string]: unknown;
};

export function setNexusCSRF(token: string) {
  setCSRFToken(token);
}

export { NexusRequestError as NexusAPIError } from '../api';

export const nexus = {
  listProviderInstallations: () => request<ProviderInfo[]>('/api/v1/providers'),
  registerProvider: (provider: string, profile?: string) =>
    request<{ profile: RegisteredProfile; state: string }>('/api/v1/providers', {
      method: 'POST',
      body: JSON.stringify({ provider, profile }),
    }),
  listProjects: () => request<Project[]>('/api/v1/projects'),
  listProjectSummaries: () =>
    request<import('../types').ProjectAgentSummary[]>('/api/v1/projects/summaries'),
  createProject: (path: string, name?: string) =>
    request<Project>('/api/v1/projects', { method: 'POST', body: JSON.stringify({ path, name }) }),
  getProject: (id: string) =>
    request<{ project: Project; layout: string; revision?: number }>(`/api/v1/projects/${id}`),
  updateProject: (id: string, data: Partial<Project>) =>
    request<Project>(`/api/v1/projects/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
  deleteProject: (id: string) =>
    request<{ status: string }>(`/api/v1/projects/${id}`, { method: 'DELETE' }),
  getLayout: (projectId: string) =>
    request<{ layout: string; revision?: number; record?: unknown }>(
      `/api/v1/projects/${projectId}/layout`,
    ),
  listProjectEvents: (projectId: string, limit = 50) =>
    request<DurableActivityEvent[]>(
      `/api/v1/projects/${encodeURIComponent(projectId)}/events?limit=${limit}`,
    ),
  getContextReadiness: (projectId: string) =>
    request<import('../types').ContextReadiness>(`/api/v1/projects/${projectId}/context`),
  prepareContext: (projectId: string, createContext = false) =>
    request<import('../types').ContextReadiness>(`/api/v1/projects/${projectId}/context/prepare`, {
      method: 'POST',
      body: JSON.stringify({ create_context: createContext }),
    }),
  getProjectIntelligence: (projectId: string) =>
    request<import('../types').ProjectIntelligenceView>(
      `/api/v1/projects/${encodeURIComponent(projectId)}/project-intelligence`,
    ),
  requestProjectIntelligenceScan: (projectId: string) =>
    request<import('../types').ProjectIntelligenceScan>(
      `/api/v1/projects/${encodeURIComponent(projectId)}/project-intelligence/scans`,
      { method: 'POST', body: '{}' },
    ),
  listComposerSessions: (projectId: string) =>
    request<import('../types').ComposerSession[]>(
      `/api/v1/projects/${projectId}/composer-sessions`,
    ),
  createComposerSessionWithMode: (
    projectId: string,
    goal: string,
    inputMode: 'IDEA' | 'EXISTING_PROMPT',
    sourcePrompt?: string,
  ) =>
    request<import('../types').ComposerSessionView>(
      `/api/v1/projects/${projectId}/composer-sessions`,
      {
        method: 'POST',
        body: JSON.stringify({ goal, input_mode: inputMode, source_prompt: sourcePrompt || '' }),
      },
    ),
  getComposerSession: (id: string) =>
    request<import('../types').ComposerSessionView>(`/api/v1/composer-sessions/${id}`),
  addComposerTurn: (id: string, content: string, expectedRevision?: number) =>
    request<import('../types').ComposerSessionView>(`/api/v1/composer-sessions/${id}/turns`, {
      method: 'POST',
      body: JSON.stringify({ content, expected_revision: expectedRevision }),
    }),
  updateComposerSkillState: (id: string, skillId: string, state: string) =>
    request<import('../types').ComposerSessionView>(
      `/api/v1/composer-sessions/${id}/skills/${encodeURIComponent(skillId)}`,
      { method: 'PATCH', body: JSON.stringify({ state }) },
    ),
  finalizeComposerSession: (id: string, skillIds: string[] = [], confirmGaps = false) =>
    request<import('../types').PromptArtifact>(`/api/v1/composer-sessions/${id}/finalize`, {
      method: 'POST',
      body: JSON.stringify({ skill_ids: skillIds, confirm_gaps: confirmGaps }),
    }),
  refineComposerArtifact: (id: string, refinementGoal?: string) =>
    request<import('../types').PromptArtifact>(`/api/v1/composer-sessions/${id}/refine`, {
      method: 'POST',
      body: JSON.stringify({ goal: refinementGoal || '' }),
    }),
  resolveComposerUnknown: (
    id: string,
    unknownId: string,
    answer: string,
    status: string,
    expectedRevision?: number,
  ) =>
    request<import('../types').ComposerSessionView>(
      `/api/v1/composer-sessions/${id}/unknowns/${encodeURIComponent(unknownId)}/resolve`,
      {
        method: 'POST',
        body: JSON.stringify({ answer, status, expected_revision: expectedRevision }),
      },
    ),
  attachComposerContext: (id: string, path: string, expectedRevision?: number) =>
    request<import('../types').ComposerSessionView>(`/api/v1/composer-sessions/${id}/context`, {
      method: 'POST',
      body: JSON.stringify({ path, expected_revision: expectedRevision }),
    }),
  detachComposerContext: (id: string, path: string, expectedRevision?: number) =>
    request<import('../types').ComposerSessionView>(
      `/api/v1/composer-sessions/${id}/context/remove`,
      {
        method: 'POST',
        body: JSON.stringify({ path, expected_revision: expectedRevision }),
      },
    ),
  materializePromptArtifact: async (id: string) =>
    normalizeWorkPlan(
      await request<import('../types').WorkPlan>(`/api/v1/prompt-artifacts/${id}/flow`, {
        method: 'POST',
      }),
    ),
  saveLayout: (projectId: string, layout: string, revision?: number) =>
    request<{ status: string; revision?: number; layout?: string }>(
      `/api/v1/projects/${projectId}/layout`,
      {
        method: 'PUT',
        body: JSON.stringify({ layout, revision }),
      },
    ),

  listAgents: (projectId: string) => request<Agent[]>(`/api/v1/projects/${projectId}/agents`),
  createAgent: (projectId: string, name: string, role?: string) =>
    request<Agent>(`/api/v1/projects/${projectId}/agents`, {
      method: 'POST',
      body: JSON.stringify({ name, role }),
    }),
  getAgent: (id: string) => request<AgentDetail>(`/api/v1/agents/${id}`),
  deleteAgent: (id: string) =>
    request<{ status: string }>(`/api/v1/agents/${id}`, { method: 'DELETE' }),
  startAgent: (id: string) =>
    request<{ runtime: RuntimeSession }>(`/api/v1/agents/${id}/start`, {
      method: 'POST',
      body: '{}',
    }),
  stopAgent: (id: string) =>
    request<{ status: string }>(`/api/v1/agents/${id}/stop`, { method: 'POST', body: '{}' }),
  recoverAgent: (id: string) =>
    request<{ runtime: RuntimeSession }>(`/api/v1/agents/${id}/recover`, {
      method: 'POST',
      body: '{}',
    }),
  askAgent: (
    id: string,
    prompt: string,
    startIfNeeded = false,
    skillIds?: string[],
    prepared?: { projectId: string; contextFingerprintId: string },
  ) => {
    const body: Record<string, unknown> = { prompt, start_if_needed: startIfNeeded };
    if (skillIds && skillIds.length > 0) {
      body.skill_ids = skillIds;
      body.scope = 'NEXT_PROMPT';
    }
    if (prepared) {
      body.project_id = prepared.projectId;
      body.context_fingerprint_id = prepared.contextFingerprintId;
    }
    return request<{ agent_id: string; runtime_id: string; started: boolean; accepted: boolean }>(
      `/api/v1/agents/${id}/ask`,
      {
        method: 'POST',
        body: JSON.stringify(body),
      },
    );
  },
  startProjectShell: (projectId: string) =>
    request<{ runtime: RuntimeSession }>(`/api/v1/projects/${projectId}/shell`, {
      method: 'POST',
      body: '{}',
    }),

  // Agent config (Gate 3)
  getAgentConfig: (id: string) =>
    request<{ config: AgentConfig; revision: string; revisions: AgentRevision[] }>(
      `/api/v1/agents/${id}/config`,
    ),
  applyAgentConfig: (id: string, config: AgentConfig) =>
    request<{ impact: ConfigImpact }>(`/api/v1/agents/${id}/config/apply`, {
      method: 'POST',
      body: JSON.stringify(config),
    }),
  previewAgentConfig: (id: string, config: AgentConfig) =>
    request<{ impact: ConfigImpact }>(`/api/v1/agents/${id}/config/impact`, {
      method: 'POST',
      body: JSON.stringify(config),
    }),

  // Resource Scheduler (Gate 5)
  listResources: () =>
    request<{ accounts: ProviderAccount[]; policy: string }>('/api/v1/resources'),
  refreshResources: (provider?: string, profile?: string) =>
    request<{ accounts: ProviderAccount[]; policy: string; refreshed?: boolean }>(
      '/api/v1/resources/refresh',
      {
        method: 'POST',
        body: JSON.stringify({ provider, profile }),
      },
    ),
  setDefaultProfile: (provider: string, profile: string) =>
    request<{ status: string; provider: string; profile: string }>('/api/v1/profiles/default', {
      method: 'POST',
      body: JSON.stringify({ provider, profile }),
    }),
  deleteProfile: (provider: string, profile: string) =>
    request<{ status: string; deleted: boolean }>(
      `/api/v1/profiles?provider=${encodeURIComponent(provider)}&profile=${encodeURIComponent(profile)}`,
      {
        method: 'DELETE',
      },
    ),
  logoutProfile: (provider: string, profile: string) =>
    request<{ status: string; logged_out: boolean }>('/api/v1/profiles/logout', {
      method: 'POST',
      body: JSON.stringify({ provider, profile }),
    }),
  selectResource: (agentId: string, provider: string, profile: string, policy = 'MANUAL') =>
    request<ResourceAllocation>('/api/v1/resources/select', {
      method: 'POST',
      body: JSON.stringify({ provider, profile, policy, agent_id: agentId }),
    }),

  // Maestro Assist (Gate 6)
  getMaestroStatus: () => request<MaestroStatus>('/api/v1/maestro'),
  getMaestroCatalog: () => request<MaestroCatalog>('/api/v1/maestro/catalog'),
  previewMaestroSync: () =>
    request<SkillSyncPreview>('/api/v1/maestro/sync/preview', {
      method: 'POST',
      body: '{}',
    }),
  applyMaestroSync: (preview: SkillSyncPreview) =>
    request<SkillSyncPreview>('/api/v1/maestro/sync', {
      method: 'POST',
      body: JSON.stringify(preview),
    }),
  getMaestroAdvice: (projectId: string, agentId?: string, intent?: string) =>
    request<MaestroAdvice>('/api/v1/maestro/advice', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId, agent_id: agentId, intent }),
    }),

  // System Updates
  getSystemInfo: () => request<NexusSystemInfo>('/api/v1/system/info'),
  getSystemDoctor: () => request<SystemDoctorReport>('/api/v1/system/doctor'),
  getSystemUpdates: () =>
    request<{
      nexus_version: string;
      nexus_commit?: string;
      nexus_build_date?: string;
      nexus_latest_version?: string;
      nexus_update_available?: boolean;
      nexus_update_error?: string;
      nexus_update_instruction?: string;
      nexus_installation_method?: string;
      nexus_allows_self_update?: boolean;
      channel?: string;
      installation_method?: string;
      allows_self_update?: boolean;
      maestro_version: string;
      maestro_latest_version?: string;
      maestro_available: boolean;
      update_available: boolean;
    }>('/api/v1/system/updates'),
  performSystemUpdate: () =>
    request<{
      nexus_updated: boolean;
      nexus_version: string;
      maestro_updated: boolean;
      maestro_version: string;
      error?: string;
    }>('/api/v1/maestro/update', {
      method: 'POST',
      body: JSON.stringify({ product: 'maestro', target_version: 'latest', confirmed: true }),
    }),

  // OS Filesystem & Desktop Launchers
  browseFS: (path?: string, includeEntryMetadata = false) => {
    const params = new URLSearchParams();
    if (path) params.set('path', path);
    if (includeEntryMetadata) params.set('details', 'full');
    const query = params.toString();
    return request<import('../types').FSBrowseResult>(
      `/api/v1/fs/browse${query ? `?${query}` : ''}`,
    );
  },
  scanFS: (root?: string) =>
    request<import('../types').FSScanResult[]>(
      `/api/v1/fs/scan${root ? `?root=${encodeURIComponent(root)}` : ''}`,
    ),
  inspectFS: (path: string) =>
    request<import('../types').FSInspectResult>(
      `/api/v1/fs/inspect?path=${encodeURIComponent(path)}`,
    ),
  mkdirFS: (path: string) =>
    request<{ path: string; status: string }>('/api/v1/fs/mkdir', {
      method: 'POST',
      body: JSON.stringify({ path }),
    }),
  openProjectInOS: (projectId: string, action: 'filemanager' | 'terminal' | 'editor') =>
    request<{ status: string; action: string; path: string }>(
      `/api/v1/projects/${projectId}/open-os`,
      {
        method: 'POST',
        body: JSON.stringify({ action }),
      },
    ),
  getProjectBranches: (projectId: string) =>
    request<import('../types').GitBranchesResult>(`/api/v1/projects/${projectId}/git/branches`),
  checkoutProjectBranch: (projectId: string, branch: string, create = false) =>
    request<import('../types').GitCheckoutResult>(`/api/v1/projects/${projectId}/git/checkout`, {
      method: 'POST',
      body: JSON.stringify({ branch, create }),
    }),

  // Nexus Intelligence (optional; Direct mode never requires it)
  getIntelligence: (projectId?: string) =>
    request<import('../types').IntelligenceStatus>(
      `/api/v1/intelligence${projectId ? `?project_id=${encodeURIComponent(projectId)}` : ''}`,
    ),
  probeIntelligence: (projectId?: string, init?: RequestInit) =>
    request<{ ok: boolean; provider?: string; error?: string; detail?: string }>(
      `/api/v1/intelligence/probe${projectId ? `?project_id=${encodeURIComponent(projectId)}` : ''}`,
      { method: 'POST', body: '{}', ...init },
    ),
  updateIntelligence: (
    data: Omit<import('../types').IntelligenceStatus, 'available' | 'error'>,
    projectId?: string,
  ) =>
    request<import('../types').IntelligenceStatus>(
      `/api/v1/intelligence${projectId ? `?project_id=${encodeURIComponent(projectId)}` : ''}`,
      { method: 'PUT', body: JSON.stringify(data) },
    ),
  listIntelligenceMemories: (projectId?: string) =>
    request<{ memories: import('../types').IntelligenceMemory[] }>(
      `/api/v1/intelligence/memories${projectId ? `?project_id=${encodeURIComponent(projectId)}` : ''}`,
    ).then((res) => res.memories || []),
  upsertIntelligenceMemory: (body: {
    scope: string;
    project_id?: string;
    key: string;
    value: string;
    source?: string;
    force?: boolean;
  }) =>
    request<import('../types').IntelligenceMemory>(`/api/v1/intelligence/memories`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  deleteIntelligenceMemory: (scope: string, id: string, projectId?: string) =>
    request<{ status: string }>(
      `/api/v1/intelligence/memories/${encodeURIComponent(scope)}/${encodeURIComponent(id)}${
        projectId ? `?project_id=${encodeURIComponent(projectId)}` : ''
      }`,
      { method: 'DELETE' },
    ),
  getClarification: (id: string) =>
    request<import('../types').ClarificationCheckpoint>(`/api/v1/clarifications/${id}`),
  resolveClarification: async (id: string, answers: Record<string, string>, init?: RequestInit) => {
    const result = await request<{
      plan: import('../types').WorkPlan;
      clarification: import('../types').ClarificationCheckpoint;
    }>(`/api/v1/clarifications/${id}/resolve`, {
      method: 'POST',
      body: JSON.stringify({ answers }),
      ...init,
    });
    return { ...result, plan: normalizeWorkPlan(result.plan) };
  },

  // WorkPlans & Autonomous Mission Runner (Phase D, E, F, H)
  getPlans: async (projectId: string) => {
    const plans = await request<import('../types').WorkPlan[] | null>(
      `/api/v1/projects/${projectId}/plans`,
    );
    return Array.isArray(plans) ? plans.map(normalizeWorkPlan) : [];
  },
  createPlan: async (
    projectId: string,
    data: {
      title?: string;
      description?: string;
      goal?: string;
      auto_plan?: boolean;
      phases?: import('../types').PlanPhase[];
      facts?: Record<string, string>;
    },
    init?: RequestInit,
  ) =>
    normalizeWorkPlan(
      await request<import('../types').WorkPlan>(`/api/v1/projects/${projectId}/plans`, {
        method: 'POST',
        body: JSON.stringify(data),
        ...init,
      }),
    ),
  getPlan: async (planId: string) => {
    const detail = await request<{
      plan: import('../types').WorkPlan;
      revisions: import('../types').PlanRevision[];
    }>(`/api/v1/plans/${planId}`);
    return { ...detail, plan: normalizeWorkPlan(detail.plan) };
  },
  updatePlan: async (
    planId: string,
    plan: import('../types').WorkPlan,
    change_summary?: string,
  ) => {
    const result = await request<{
      plan: import('../types').WorkPlan;
      revision: import('../types').PlanRevision;
    }>(`/api/v1/plans/${planId}`, {
      method: 'PUT',
      body: JSON.stringify({ plan, change_summary }),
    });
    return { ...result, plan: normalizeWorkPlan(result.plan) };
  },
  restorePlanRevision: async (planId: string, revision: number) => {
    const result = await request<{
      plan: import('../types').WorkPlan;
      revision: import('../types').PlanRevision;
    }>(`/api/v1/plans/${planId}/restore`, {
      method: 'POST',
      body: JSON.stringify({ revision }),
    });
    return { ...result, plan: normalizeWorkPlan(result.plan) };
  },
  diffPlanRevisions: (planId: string, from: number, to: number) =>
    request<import('../types').PlanRevisionDiff>(
      `/api/v1/plans/${planId}/diff?from=${from}&to=${to}`,
    ),
  getFlowLeader: (planId: string) =>
    request<import('../types').FlowLeaderPolicy>(`/api/v1/plans/${planId}/leader`),
  setFlowLeader: (planId: string, leader: import('../types').FlowLeaderPolicy) =>
    request<{
      plan: import('../types').WorkPlan;
      leader: import('../types').FlowLeaderPolicy;
    }>(`/api/v1/plans/${planId}/leader`, { method: 'PUT', body: JSON.stringify(leader) }),
  preflightFlow: (planId: string) =>
    request<import('../types').FlowPreflightReport>(`/api/v1/plans/${planId}/preflight`, {
      method: 'POST',
    }),
  compilePackagePrompt: (planId: string, packageId: string, phaseId?: string) =>
    request<{ compiled_prompt: string; package_id: string }>(`/api/v1/plans/${planId}/compile`, {
      method: 'POST',
      body: JSON.stringify({ package_id: packageId, phase_id: phaseId }),
    }),
  runPlan: (planId: string, planRevision: number, agentId?: string, maxRetries?: number) =>
    request<import('../types').MissionRun>(`/api/v1/plans/${planId}/run`, {
      method: 'POST',
      body: JSON.stringify({
        agent_id: agentId,
        plan_revision: planRevision,
        max_retries: maxRetries,
        autonomous: true,
      }),
    }),
  getRuns: () => request<import('../types').MissionRun[]>('/api/v1/runs'),
  getRun: (runId: string) => request<import('../types').MissionRun>(`/api/v1/runs/${runId}`),
  getRunEvidence: (runId: string) =>
    request<import('../types').FlowRunEvidence>(`/api/v1/runs/${runId}/evidence`),
  pauseRun: (runId: string, reason?: string) =>
    request<import('../types').MissionRun>(`/api/v1/runs/${runId}/pause`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),
  takeControlRun: (runId: string, reason?: string) =>
    request<import('../types').MissionRun>(`/api/v1/runs/${runId}/take-control`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),
  resumeRun: (runId: string) =>
    request<import('../types').MissionRun>(`/api/v1/runs/${runId}/resume`, { method: 'POST' }),
  returnToMission: (runId: string) =>
    request<import('../types').MissionRun>(`/api/v1/runs/${runId}/return-to-mission`, {
      method: 'POST',
    }),
  cancelRun: (runId: string, reason?: string) =>
    request<import('../types').MissionRun>(`/api/v1/runs/${runId}/cancel`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),
  getSchedules: (projectId: string) =>
    request<import('../types').MissionSchedule[]>(
      `/api/v1/schedules?project_id=${encodeURIComponent(projectId)}`,
    ),
  schedulePlan: (
    planId: string,
    mode: 'AT' | 'AFTER_RUN' | 'WHEN_RESOURCES',
    options?: { scheduledFor?: string; afterRunId?: string; agentId?: string },
  ) =>
    request<import('../types').MissionSchedule>('/api/v1/schedules', {
      method: 'POST',
      body: JSON.stringify({
        plan_id: planId,
        mode,
        scheduled_for: options?.scheduledFor,
        after_run_id: options?.afterRunId,
        agent_id: options?.agentId,
      }),
    }),
  cancelSchedule: (scheduleId: string) =>
    request<{ canceled: boolean }>('/api/v1/schedules', {
      method: 'POST',
      body: JSON.stringify({ cancel_id: scheduleId }),
    }),
  recommendResources: (
    requirements: {
      task_kind: string;
      role: string;
      current_provider?: string;
      prefer_provider?: string;
    },
    policy?: string,
  ) =>
    request<import('../types').RecommendationResult>('/api/v1/resources/recommend', {
      method: 'POST',
      body: JSON.stringify({ requirements, policy }),
    }),

  // Tunnel / Remote Access
  getTunnelStatus: () => request<import('../types').TunnelStatus>('/api/v1/tunnel/status'),
  startTunnel: () =>
    request<import('../types').TunnelStatus>('/api/v1/tunnel/start', { method: 'POST' }),
  stopTunnel: () =>
    request<import('../types').TunnelStatus>('/api/v1/tunnel/stop', { method: 'POST' }),
  getTunnelQR: () => request<{ url: string; bootstrap_url: string }>('/api/v1/tunnel/qr'),

  // Attention Center
  getAttention: () => request<import('../types').AttentionGroup>('/api/v1/attention'),
  resolveIntervention: (
    runId: string,
    interventionId: string,
    version: number,
    optionId: string,
    resolvedBy?: string,
  ) =>
    request<import('../types').MissionRun>(`/api/v1/runs/${runId}/resolve-intervention`, {
      method: 'POST',
      body: JSON.stringify({
        intervention_id: interventionId,
        version,
        option_id: optionId,
        resolved_by: resolvedBy,
      }),
    }),

  // Telemetry & Historical Audit
  getTelemetryStats: (days = 7) =>
    request<import('../types').TelemetryStats>(`/api/v1/telemetry/stats?days=${days}`),
  getTelemetryHistory: (limit = 50, provider?: string) => {
    const params = new URLSearchParams();
    if (limit) params.set('limit', String(limit));
    if (provider) params.set('provider', provider);
    const qs = params.toString();
    return request<import('../types').TelemetryHistoryEvent[]>(
      `/api/v1/telemetry/history${qs ? `?${qs}` : ''}`,
    );
  },
};

export const nexusApi = nexus;
