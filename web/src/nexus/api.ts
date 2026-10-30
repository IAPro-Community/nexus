/* Nexus product API client (projects, agents, layouts, config). */

import {
  Project,
  Agent,
  AgentMode,
  ExecutionRole,
  RuntimeExecutionRoleView,
  AgentProjectionPlanView,
  AgentProjectionApplyView,
  AgentProjectionDiscovery,
  AgentProjectionActivation,
  AgentProjectionEnforcement,
  AgentProjectionVerification,
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
  HostSessionImportResult,
} from '../types';
import { request, setCSRFToken } from '../api';
import { normalizeWorkPlan } from './workPlan';
import { missionAttentionQueryPath, type MissionAttentionFilter } from './missionAttentionModel';

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
    request<{
      profile: RegisteredProfile;
      state: string;
      auth_source?: {
        origin?: string;
        linked_at?: string;
        account_id_hash?: string;
        plan?: string;
      } | null;
    }>('/api/v1/providers', {
      method: 'POST',
      body: JSON.stringify({ provider, profile }),
    }),
  importHostSessions: () =>
    request<{ results: HostSessionImportResult[] }>('/api/v1/profiles/import-host-sessions', {
      method: 'POST',
      body: '{}',
    }),
  requestDesktopCommand: (action: string) =>
    request<{ status: string; action: string }>('/api/v1/desktop/commands', {
      method: 'POST',
      body: JSON.stringify({ action }),
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
  listComposerSkillCatalog: (projectId: string, query = '') => {
    const suffix = query.trim() ? `?q=${encodeURIComponent(query.trim())}` : '';
    return request<import('../types').ComposerSkillCatalogItem[]>(
      `/api/v1/projects/${projectId}/skills${suffix}`,
    );
  },
  createComposerSessionWithMode: (
    projectId: string,
    goal: string,
    inputMode: 'IDEA' | 'EXISTING_PROMPT',
    sourcePrompt?: string,
    interviewMode: 'QUICK' | 'STANDARD' | 'DEEP' = 'STANDARD',
    signal?: AbortSignal,
  ) =>
    request<import('../types').ComposerSessionView>(
      `/api/v1/projects/${projectId}/composer-sessions`,
      {
        method: 'POST',
        body: JSON.stringify({
          goal,
          input_mode: inputMode,
          interview_mode: interviewMode,
          source_prompt: sourcePrompt || '',
        }),
        signal,
      },
    ),
  getComposerSession: (id: string) =>
    request<import('../types').ComposerSessionView>(`/api/v1/composer-sessions/${id}`),
  deleteComposerSession: (id: string, expectedRevision: number) =>
    request<{ deleted: boolean }>(
      `/api/v1/composer-sessions/${encodeURIComponent(id)}?expected_revision=${expectedRevision}`,
      { method: 'DELETE' },
    ),
  addComposerTurn: (
    id: string,
    content: string,
    expectedRevision?: number,
    attachments?: Array<{
      id: string;
      name?: string;
      mimeType?: string;
      url?: string;
      size?: number;
    }>,
    signal?: AbortSignal,
  ) =>
    request<import('../types').ComposerSessionView>(`/api/v1/composer-sessions/${id}/turns`, {
      method: 'POST',
      body: JSON.stringify({
        content,
        expected_revision: expectedRevision,
        attachments: attachments || undefined,
      }),
      signal,
    }),
  uploadComposerAttachment: async (sessionId: string, file: File, signal?: AbortSignal) => {
    const { getCSRFToken, getDesktopAuthToken, getDesktopBaseUrl, NexusRequestError } =
      await import('../api');
    const headers: Record<string, string> = { Accept: 'application/json' };
    const csrf = getCSRFToken();
    if (csrf) headers['X-CSRF-Token'] = csrf;
    const auth = getDesktopAuthToken();
    if (auth) headers['Authorization'] = `Bearer ${auth}`;
    const form = new FormData();
    form.append('file', file);
    const base = getDesktopBaseUrl();
    const path = `/api/v1/composer-sessions/${sessionId}/attachments`;
    const res = await fetch(base ? `${base}${path}` : path, {
      method: 'POST',
      headers,
      body: form,
      signal,
      credentials: 'include',
    });
    if (!res.ok) {
      const errBody = (await res.json().catch(() => ({ error: res.statusText }))) as {
        error?: string;
      };
      throw new NexusRequestError(res.status, errBody, errBody.error || `HTTP ${res.status}`);
    }
    return res.json() as Promise<{
      id: string;
      name: string;
      mime_type: string;
      size: number;
    }>;
  },
  updateComposerSkillState: (id: string, skillId: string, state: string) =>
    request<import('../types').ComposerSessionView>(
      `/api/v1/composer-sessions/${id}/skills/${encodeURIComponent(skillId)}`,
      { method: 'PATCH', body: JSON.stringify({ state }) },
    ),
  selectComposerSkill: (id: string, skillId: string) =>
    request<import('../types').ComposerSessionView>(`/api/v1/composer-sessions/${id}/skills`, {
      method: 'POST',
      body: JSON.stringify({ skill_id: skillId }),
    }),
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

  listAgentModes: () => request<AgentMode[]>('/api/v1/agent-modes'),
  listExecutionRoles: () => request<ExecutionRole[]>('/api/v1/execution-roles'),
  getRuntimeExecutionRole: (runtimeId: string) =>
    request<RuntimeExecutionRoleView>(
      `/api/v1/runtimes/${encodeURIComponent(runtimeId)}/execution-role`,
    ),
  setRuntimeExecutionRole: (runtimeId: string, roleId: string) =>
    request<RuntimeExecutionRoleView>(
      `/api/v1/runtimes/${encodeURIComponent(runtimeId)}/execution-role`,
      {
        method: 'POST',
        body: JSON.stringify({ role_id: roleId }),
      },
    ),
  planAgentProjection: (runtimeId: string) =>
    request<AgentProjectionPlanView>('/api/v1/agent-projections/plan', {
      method: 'POST',
      body: JSON.stringify({ runtime_id: runtimeId }),
    }),
  applyAgentProjection: (runtimeId: string) =>
    request<AgentProjectionApplyView>('/api/v1/agent-projections/apply', {
      method: 'POST',
      body: JSON.stringify({ runtime_id: runtimeId }),
    }),
  probeAgentProjectionDiscovery: (runtimeId: string) =>
    request<AgentProjectionDiscovery>('/api/v1/agent-projections/discovery', {
      method: 'POST',
      body: JSON.stringify({ runtime_id: runtimeId }),
    }),
  probeAgentProjectionActivation: (runtimeId: string, confirmExternalExecution = false) =>
    request<AgentProjectionActivation>('/api/v1/agent-projections/activation', {
      method: 'POST',
      body: JSON.stringify({
        runtime_id: runtimeId,
        confirm_external_execution: confirmExternalExecution,
      }),
    }),
  probeAgentProjectionEnforcement: (runtimeId: string, confirmExternalExecution = false) =>
    request<AgentProjectionEnforcement>('/api/v1/agent-projections/enforcement', {
      method: 'POST',
      body: JSON.stringify({
        runtime_id: runtimeId,
        confirm_external_execution: confirmExternalExecution,
      }),
    }),
  verifyAgentProjection: (receiptId: string) =>
    request<AgentProjectionVerification>(
      `/api/v1/agent-projections/${encodeURIComponent(receiptId)}/verify`,
      { method: 'POST', body: '{}' },
    ),
  removeAgentProjection: (receiptId: string) =>
    request<{ status: string }>(`/api/v1/agent-projections/${encodeURIComponent(receiptId)}`, {
      method: 'DELETE',
    }),
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
      nexus_latest_sha?: string;
      nexus_update_available?: boolean;
      nexus_update_error?: string;
      nexus_update_instruction?: string;
      nexus_installer_url?: string;
      nexus_release_url?: string;
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
  deletePlan: (planId: string, expectedRevision: number) =>
    request<{ deleted: boolean }>(
      `/api/v1/plans/${encodeURIComponent(planId)}?expected_revision=${expectedRevision}`,
      { method: 'DELETE' },
    ),
  // Direct-to-Flow semantic planning. No Agent or Mission is launched.
  generateFlowPlan: async (projectId: string, prompt: string, init?: RequestInit) =>
    normalizeWorkPlan(
      await request<import('../types').WorkPlan>('/api/v1/flow/generate', {
        method: 'POST',
        body: JSON.stringify({ project_id: projectId, prompt }),
        ...init,
      }),
    ),
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
  previewPlanExecution: (planId: string) =>
    request<import('../types').PlanExecutionPreview>(`/api/v1/plans/${planId}/execution-preview`, {
      method: 'POST',
    }),
  preflightFlow: (planId: string) =>
    request<import('../types').FlowPreflightReport>(`/api/v1/plans/${planId}/preflight`, {
      method: 'POST',
    }),
  compilePackagePrompt: (planId: string, packageId: string, phaseId?: string) =>
    request<{ compiled_prompt: string; package_id: string }>(`/api/v1/plans/${planId}/compile`, {
      method: 'POST',
      body: JSON.stringify({ package_id: packageId, phase_id: phaseId }),
    }),
  runPlan: (
    planId: string,
    planRevision: number,
    agentId?: string,
    maxRetries?: number,
    costConfirmationToken?: string,
    executionPreviewToken?: string,
  ) =>
    request<import('../types').MissionRun>(`/api/v1/plans/${planId}/run`, {
      method: 'POST',
      body: JSON.stringify({
        agent_id: agentId,
        plan_revision: planRevision,
        max_retries: maxRetries,
        autonomous: true,
        cost_confirmation_token: costConfirmationToken,
        execution_preview_token: executionPreviewToken,
      }),
    }),
  getRuns: () => request<import('../types').MissionRun[]>('/api/v1/runs'),
  getRun: (runId: string) => request<import('../types').MissionRun>(`/api/v1/runs/${runId}`),
  requestMissionPlanningReview: (runId: string, reason: string) =>
    request<import('../types').ComposerSessionView>(`/api/v1/runs/${runId}/planning-review`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),
  getRunCoordination: (runId: string) =>
    request<import('../types').CoordinationRequest[]>(`/api/v1/runs/${runId}/coordination`),
  submitRunCoordination: (
    runId: string,
    requestBody: {
      request_id: string;
      source_package_id: string;
      source_agent_id?: string;
      operation: import('../types').CoordinationOperation;
      subject: string;
      objective?: string;
      preferred_role?: string;
      preferred_provider?: string;
      required_capabilities?: string[];
      artifact_refs?: string[];
      materialize?: boolean;
    },
  ) =>
    request<import('../types').RunCoordinationResult>(`/api/v1/runs/${runId}/coordination`, {
      method: 'POST',
      body: JSON.stringify(requestBody),
    }),
  materializeRunCoordination: (runId: string, requestId: string) =>
    request<import('../types').RunCoordinationResult>(
      `/api/v1/runs/${runId}/coordination-materialize`,
      {
        method: 'POST',
        body: JSON.stringify({ request_id: requestId }),
      },
    ),
  resolveRunCoordination: (
    runId: string,
    requestId: string,
    status: import('../types').CoordinationStatus,
    options?: { resolutionSummary?: string; resultReceiptId?: string },
  ) =>
    request<import('../types').CoordinationRequest>(`/api/v1/runs/${runId}/coordination-resolve`, {
      method: 'POST',
      body: JSON.stringify({
        request_id: requestId,
        status,
        resolution_summary: options?.resolutionSummary,
        result_receipt_id: options?.resultReceiptId,
      }),
    }),
  getRunEvidence: (runId: string) =>
    request<import('../types').FlowRunEvidence>(`/api/v1/runs/${runId}/evidence`),
  getRunEvidenceBundle: (runId: string) =>
    request<import('../types').MissionEvidenceBundle>(`/api/v1/runs/${runId}/evidence-bundle`),
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
      domains?: string[];
      estimated_complexity?: string;
      source?: string;
    },
    policy?: string,
  ) =>
    request<import('../types').RecommendationResult>('/api/v1/resources/recommend', {
      method: 'POST',
      body: JSON.stringify({ requirements, policy }),
    }),

  // Voice accessibility: provider settings remain server-side and multipart audio
  // never exposes provider credentials to the browser.
  getVoiceSettings: () =>
    request<import('../features/voice/voiceModel').VoiceSettingsView>('/api/v1/voice/settings'),
  putVoiceSettings: (settings: import('../features/voice/voiceModel').VoiceSettingsUpdate) =>
    request<import('../features/voice/voiceModel').VoiceSettingsView>('/api/v1/voice/settings', {
      method: 'PUT',
      body: JSON.stringify(settings),
    }),
  transcribeVoice: (form: FormData) =>
    request<import('../features/voice/voiceModel').VoiceTranscribeResult>(
      '/api/v1/voice/transcribe',
      { method: 'POST', body: form },
    ),

  // Nexus HUD: read-only projection over canonical Capacity/Resource/Agent/Mission/Cost state.
  getHUDState: (projectId?: string) => {
    const params = new URLSearchParams();
    if (projectId) params.set('project_id', projectId);
    const qs = params.toString();
    return request<import('../features/hud/hudModel').HudResponse>(
      `/api/v1/hud${qs ? `?${qs}` : ''}`,
    );
  },

  // Capacity Monitor (revival T6)
  getCapacityStatus: () => request<import('../types').CapacityStatus>('/api/v1/capacity/status'),
  getPublicServiceStatus: () =>
    request<import('../types').PublicServiceStatus[]>('/api/v1/providers/public-status'),
  getCapacityHistory: (limit?: number) =>
    request<import('../types').CapacityHistoryEvent[]>(
      `/api/v1/capacity/history?limit=${encodeURIComponent(String(limit ?? 50))}`,
    ),
  getCapacityPreferences: () =>
    request<import('../types').CapacityPreferences>('/api/v1/capacity/preferences'),
  putCapacityPreferences: (prefs: import('../types').CapacityPreferences) =>
    request<import('../types').CapacityPreferences>('/api/v1/capacity/preferences', {
      method: 'PUT',
      body: JSON.stringify(prefs),
    }),
  ackCapacityAlert: (fingerprint: string) =>
    request<import('../types').CapacityAlertState>('/api/v1/capacity/ack', {
      method: 'POST',
      body: JSON.stringify({ fingerprint }),
    }),
  snoozeCapacityAlert: (fingerprint: string, until?: string) =>
    request<import('../types').CapacityAlertState>('/api/v1/capacity/snooze', {
      method: 'POST',
      body: JSON.stringify({ fingerprint, until: until ?? '' }),
    }),

  // Local Cost Intelligence. Unknown token/cost evidence stays absent rather
  // than being coerced to zero.
  getCostSummary: (days = 7, projectId?: string) => {
    const params = new URLSearchParams({ days: String(days) });
    if (projectId) params.set('project_id', projectId);
    return request<import('../types').CostSummary>(`/api/v1/cost/summary?${params.toString()}`);
  },
  getExternalClaudeUsage: (days = 7) =>
    request<import('../types').ExternalClaudeSummary>(
      `/api/v1/cost/external-claude?days=${days === 30 ? 30 : 7}`,
    ),
  scanExternalClaudeUsage: () =>
    request<import('../types').ExternalClaudeScanResult>('/api/v1/cost/external-claude', {
      method: 'POST',
      body: JSON.stringify({ consent: true }),
    }),
  clearExternalClaudeUsage: () =>
    request<{ cleared: boolean }>('/api/v1/cost/external-claude', { method: 'DELETE' }),
  getCostPreferences: () => request<import('../types').CostPreferences>('/api/v1/cost/preferences'),
  putCostPreferences: (prefs: import('../types').CostPreferences) =>
    request<import('../types').CostPreferences>('/api/v1/cost/preferences', {
      method: 'PUT',
      body: JSON.stringify(prefs),
    }),
  getCostObservations: (options?: {
    limit?: number;
    days?: number;
    projectId?: string;
    missionRunId?: string;
    agentId?: string;
    provider?: string;
    profile?: string;
  }) => {
    const params = new URLSearchParams();
    if (options?.limit) params.set('limit', String(options.limit));
    if (options?.days) params.set('days', String(options.days));
    if (options?.projectId) params.set('project_id', options.projectId);
    if (options?.missionRunId) params.set('mission_run_id', options.missionRunId);
    if (options?.agentId) params.set('agent_id', options.agentId);
    if (options?.provider) params.set('provider', options.provider);
    if (options?.profile) params.set('profile', options.profile);
    const qs = params.toString();
    return request<import('../types').CostObservation[]>(
      `/api/v1/cost/observations${qs ? `?${qs}` : ''}`,
    );
  },
  // Tunnel / Remote Access
  getTunnelStatus: () => request<import('../types').TunnelStatus>('/api/v1/tunnel/status'),
  startTunnel: () =>
    request<import('../types').TunnelStatus>('/api/v1/tunnel/start', { method: 'POST' }),
  stopTunnel: () =>
    request<import('../types').TunnelStatus>('/api/v1/tunnel/stop', { method: 'POST' }),
  createTunnelInvite: () =>
    request<{ url: string; bootstrap_url: string; expires_in_seconds: number }>(
      '/api/v1/tunnel/invite',
      { method: 'POST', body: '{}' },
    ),

  // Attention Center
  getAttention: (filter?: MissionAttentionFilter) =>
    request<import('../types').AttentionGroup>(missionAttentionQueryPath(filter)),
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
