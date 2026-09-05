/* Nexus product API client (projects, agents, layouts, config). */

import { Project, Agent, AgentDetail, RuntimeSession, AgentConfig, ConfigImpact } from '../types';
import { normalizeWorkPlan } from './workPlan';

export class NexusAPIError<TPayload = unknown> extends Error {
  readonly status: number;
  readonly payload: TPayload;

  constructor(status: number, payload: TPayload, message: string) {
    super(message);
    this.name = 'NexusAPIError';
    this.status = status;
    this.payload = payload;
  }
}

let csrf = '';
export function setNexusCSRF(token: string) {
  csrf = token;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers || {});
  headers.set('Accept', 'application/json');
  if (options.method && options.method !== 'GET' && options.method !== 'HEAD') {
    headers.set('Content-Type', 'application/json');
    if (csrf) headers.set('X-CSRF-Token', csrf);
  }
  const res = await fetch(path, { ...options, headers });
  if (!res.ok) {
    if (res.status === 401 && typeof window !== 'undefined')
      window.dispatchEvent(new CustomEvent('nexus:session-expired'));
    const errBody = await res.json().catch(() => ({ error: res.statusText }));
    const message = typeof errBody?.error === 'string' ? errBody.error : `HTTP ${res.status}`;
    throw new NexusAPIError(res.status, errBody, message);
  }
  return res.json();
}

export const nexus = {
  listProjects: () => request<Project[]>('/api/v1/projects'),
  createProject: (path: string, name?: string) =>
    request<Project>('/api/v1/projects', { method: 'POST', body: JSON.stringify({ path, name }) }),
  getProject: (id: string) =>
    request<{ project: Project; layout: string }>(`/api/v1/projects/${id}`),
  updateProject: (id: string, data: Partial<Project>) =>
    request<Project>(`/api/v1/projects/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
  deleteProject: (id: string) =>
    request<{ status: string }>(`/api/v1/projects/${id}`, { method: 'DELETE' }),
  getLayout: (projectId: string) =>
    request<{ layout: string }>(`/api/v1/projects/${projectId}/layout`),
  getContextReadiness: (projectId: string) =>
    request<import('../types').ContextReadiness>(`/api/v1/projects/${projectId}/context`),
  prepareContext: (projectId: string, createContext = false) =>
    request<import('../types').ContextReadiness>(`/api/v1/projects/${projectId}/context/prepare`, {
      method: 'POST',
      body: JSON.stringify({ create_context: createContext }),
    }),
  listComposerSessions: (projectId: string) =>
    request<import('../types').ComposerSession[]>(
      `/api/v1/projects/${projectId}/composer-sessions`,
    ),
  createComposerSession: (projectId: string, goal: string) =>
    request<import('../types').ComposerSessionView>(
      `/api/v1/projects/${projectId}/composer-sessions`,
      { method: 'POST', body: JSON.stringify({ goal }) },
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
  addComposerTurn: (id: string, content: string) =>
    request<import('../types').ComposerSessionView>(`/api/v1/composer-sessions/${id}/turns`, {
      method: 'POST',
      body: JSON.stringify({ content }),
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
  resolveComposerUnknown: (id: string, unknownId: string, answer: string, status: string) =>
    request<import('../types').ComposerSessionView>(
      `/api/v1/composer-sessions/${id}/unknowns/${encodeURIComponent(unknownId)}/resolve`,
      {
        method: 'POST',
        body: JSON.stringify({ answer, status }),
      },
    ),
  materializePromptArtifact: async (id: string) =>
    normalizeWorkPlan(
      await request<unknown>(`/api/v1/prompt-artifacts/${id}/flow`, { method: 'POST' }),
    ),
  saveLayout: (projectId: string, layout: string) =>
    request<{ status: string }>(`/api/v1/projects/${projectId}/layout`, {
      method: 'PUT',
      body: JSON.stringify({ layout }),
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
  askAgent: (id: string, prompt: string, startIfNeeded = false, skillIds?: string[]) => {
    const body: Record<string, any> = { prompt, start_if_needed: startIfNeeded };
    if (skillIds && skillIds.length > 0) {
      body.skill_ids = skillIds;
      body.scope = 'NEXT_PROMPT';
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
    request<{ config: AgentConfig; revision: string; revisions: any[] }>(
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
  listResources: () => request<{ accounts: any[]; policy: string }>('/api/v1/resources'),
  selectResource: (agentId: string, provider: string, profile: string, policy = 'MANUAL') =>
    request<{ decision: any; persisted: boolean }>('/api/v1/resources/select', {
      method: 'POST',
      body: JSON.stringify({ provider, profile, policy, agent_id: agentId }),
    }),

  // Maestro Assist (Gate 6)
  getMaestroStatus: () => request<any>('/api/v1/maestro'),
  getMaestroAdvice: (projectId: string, agentId?: string, intent?: string) =>
    request<any>('/api/v1/maestro/advice', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId, agent_id: agentId, intent }),
    }),

  // Missions (Gate 7 Beta)
  listMissions: (projectId: string) =>
    request<{ missions: any[] }>(`/api/v1/projects/${projectId}/missions`),
  createMission: (
    projectId: string,
    data: {
      name: string;
      description?: string;
      goal?: string;
      scope?: string;
      risk_level?: string;
    },
  ) =>
    request<any>(`/api/v1/projects/${projectId}/missions`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  getMission: (missionId: string) => request<any>(`/api/v1/missions/${missionId}`),
  updateMission: (missionId: string, data: any) =>
    request<any>(`/api/v1/missions/${missionId}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),
  deleteMission: (missionId: string) =>
    request<any>(`/api/v1/missions/${missionId}`, { method: 'DELETE' }),
  addMissionTask: (
    missionId: string,
    data: { name: string; description?: string; kind?: string; priority?: number },
  ) =>
    request<any>(`/api/v1/missions/${missionId}/tasks`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  assignMissionAgent: (missionId: string, data: { task_id: string; agent_id: string }) =>
    request<any>(`/api/v1/missions/${missionId}/assign`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // System Updates
  getSystemUpdates: () =>
    request<{
      nexus_version: string;
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
    }>('/api/v1/system/update', { method: 'POST', body: '{}' }),

  // OS Filesystem & Desktop Launchers
  browseFS: (path?: string) =>
    request<import('../types').FSBrowseResult>(
      `/api/v1/fs/browse${path ? `?path=${encodeURIComponent(path)}` : ''}`,
    ),
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
  getClarification: (id: string) =>
    request<import('../types').ClarificationCheckpoint>(`/api/v1/clarifications/${id}`),
  resolveClarification: async (id: string, answers: Record<string, string>, init?: RequestInit) => {
    const result = await request<{
      plan: unknown;
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
    const plans = await request<unknown>(`/api/v1/projects/${projectId}/plans`);
    return Array.isArray(plans) ? plans.map(normalizeWorkPlan) : [];
  },
  createPlan: async (
    projectId: string,
    data: {
      title?: string;
      description?: string;
      goal?: string;
      auto_plan?: boolean;
      phases?: any[];
      facts?: any;
    },
    init?: RequestInit,
  ) =>
    normalizeWorkPlan(
      await request<unknown>(`/api/v1/projects/${projectId}/plans`, {
        method: 'POST',
        body: JSON.stringify(data),
        ...init,
      }),
    ),
  getPlan: async (planId: string) => {
    const detail = await request<{ plan: unknown; revisions: import('../types').PlanRevision[] }>(
      `/api/v1/plans/${planId}`,
    );
    return { ...detail, plan: normalizeWorkPlan(detail.plan) };
  },
  updatePlan: async (
    planId: string,
    plan: import('../types').WorkPlan,
    change_summary?: string,
  ) => {
    const result = await request<{ plan: unknown; revision: import('../types').PlanRevision }>(
      `/api/v1/plans/${planId}`,
      {
        method: 'PUT',
        body: JSON.stringify({ plan, change_summary }),
      },
    );
    return { ...result, plan: normalizeWorkPlan(result.plan) };
  },
  deletePlan: (planId: string) =>
    request<{ deleted: boolean }>(`/api/v1/plans/${planId}`, { method: 'DELETE' }),
  restorePlanRevision: async (planId: string, revision: number) => {
    const result = await request<{ plan: unknown; revision: import('../types').PlanRevision }>(
      `/api/v1/plans/${planId}/restore`,
      {
        method: 'POST',
        body: JSON.stringify({ revision }),
      },
    );
    return { ...result, plan: normalizeWorkPlan(result.plan) };
  },
  diffPlanRevisions: (planId: string, from: number, to: number) =>
    request<import('../types').PlanRevisionDiff>(
      `/api/v1/plans/${planId}/diff?from=${from}&to=${to}`,
    ),
  getFlowLeader: (planId: string) =>
    request<import('../types').FlowLeaderPolicy>(`/api/v1/plans/${planId}/leader`),
  setFlowLeader: (planId: string, leader: import('../types').FlowLeaderPolicy) =>
    request<{ plan: unknown; leader: import('../types').FlowLeaderPolicy }>(
      `/api/v1/plans/${planId}/leader`,
      { method: 'PUT', body: JSON.stringify(leader) },
    ),
  cloneFlow: (planId: string, projectId: string) =>
    request<import('../types').WorkPlan>(`/api/v1/plans/${planId}/clone`, {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId }),
    }),
  preflightFlow: (planId: string) =>
    request<import('../types').FlowPreflightReport>(`/api/v1/plans/${planId}/preflight`, {
      method: 'POST',
    }),
  decomposePrompt: (data: import('../types').FlowDecompositionRequest) =>
    request<import('../types').FlowDecompositionProposal>('/api/v1/flows/decompose', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  compilePackagePrompt: (planId: string, packageId: string, phaseId?: string) =>
    request<any>(`/api/v1/plans/${planId}/compile`, {
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
  stepRun: (runId: string) =>
    request<{ run: import('../types').MissionRun; completed: boolean }>(
      `/api/v1/runs/${runId}/step`,
      { method: 'POST' },
    ),
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
  recommendResources: (requirements: any, policy?: string) =>
    request<import('../types').RecommendationResult>('/api/v1/resources/recommend', {
      method: 'POST',
      body: JSON.stringify({ requirements, policy }),
    }),

  // Capacity Monitor API (Task 7)
  getCapacityMonitorStatus: () =>
    request<import('../types').CapacityMonitorStatus>('/api/v1/capacity-monitor'),
  getCapacityMonitorResources: () =>
    request<import('../types').CapacityResourceState[]>('/api/v1/capacity-monitor/resources'),
  getCapacityMonitorHistory: (limit?: number) =>
    request<import('../types').CapacityHistoryPage>(
      `/api/v1/capacity-monitor/history${limit ? `?limit=${limit}` : ''}`,
    ),
  getCapacityMonitorPreferences: () =>
    request<import('../types').CapacityPreferences>('/api/v1/capacity-monitor/preferences'),
  updateCapacityMonitorPreferences: (prefs: import('../types').CapacityPreferences) =>
    request<import('../types').CapacityPreferences>('/api/v1/capacity-monitor/preferences', {
      method: 'PUT',
      body: JSON.stringify(prefs),
    }),
  acknowledgeCapacityAlert: (alertId: string) =>
    request<{ status: string; alert_id: string }>(
      `/api/v1/capacity-monitor/alerts/${encodeURIComponent(alertId)}/ack`,
      { method: 'POST', body: '{}' },
    ),
  snoozeCapacityAlert: (alertId: string, durationSeconds?: number) =>
    request<{ status: string; alert_id: string; duration_seconds: number }>(
      `/api/v1/capacity-monitor/alerts/${encodeURIComponent(alertId)}/snooze`,
      {
        method: 'POST',
        body: JSON.stringify({ duration_seconds: durationSeconds || 3600 }),
      },
    ),
  switchAgentCapacity: (agentId: string, req: import('../types').CapacitySwitchRequest) =>
    request<import('../types').ResourceAllocation>(`/api/v1/agents/${agentId}/capacity-switch`, {
      method: 'POST',
      body: JSON.stringify(req),
    }),
};

export const nexusApi = nexus;
