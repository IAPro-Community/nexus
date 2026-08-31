/* Nexus product API client (projects, agents, layouts, config). */

import { Project, Agent, AgentDetail, RuntimeSession, AgentConfig, ConfigImpact } from '../types';
import type { AutonomyContract } from '../types';

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
  getProject: (id: string) => request<{ project: Project; layout: string }>(`/api/v1/projects/${id}`),
  updateProject: (id: string, data: Partial<Project>) =>
    request<Project>(`/api/v1/projects/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
  deleteProject: (id: string) => request<{ status: string }>(`/api/v1/projects/${id}`, { method: 'DELETE' }),
  getLayout: (projectId: string) => request<{ layout: string }>(`/api/v1/projects/${projectId}/layout`),
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
  deleteAgent: (id: string) => request<{ status: string }>(`/api/v1/agents/${id}`, { method: 'DELETE' }),
  startAgent: (id: string) =>
    request<{ runtime: RuntimeSession }>(`/api/v1/agents/${id}/start`, {
      method: 'POST',
      body: '{}',
    }),
  stopAgent: (id: string) =>
    request<{ status: string }>(`/api/v1/agents/${id}/stop`, { method: 'POST', body: '{}' }),
  recoverAgent: (id: string) =>
    request<{ runtime: RuntimeSession }>(`/api/v1/agents/${id}/recover`, { method: 'POST', body: '{}' }),
  askAgent: (id: string, instruction: string) =>
    request<Agent>(`/api/v1/agents/${id}/ask`, { method: 'POST', body: JSON.stringify({ instruction }) }),

  // Agent config (Gate 3)
  getAgentConfig: (id: string) =>
    request<{ config: AgentConfig; revision: string; revisions: any[] }>(`/api/v1/agents/${id}/config`),
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
    request<{ accounts: any[]; policy: string }>('/api/v1/resources'),
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
  createMission: (projectId: string, data: { name: string; description?: string; goal?: string; scope?: string; risk_level?: string }) =>
    request<any>(`/api/v1/projects/${projectId}/missions`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  getMission: (missionId: string) =>
    request<any>(`/api/v1/missions/${missionId}`),
  updateMission: (missionId: string, data: any) =>
    request<any>(`/api/v1/missions/${missionId}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),
  deleteMission: (missionId: string) =>
    request<any>(`/api/v1/missions/${missionId}`, { method: 'DELETE' }),
  addMissionTask: (missionId: string, data: { name: string; description?: string; kind?: string; priority?: number }) =>
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
  getSystemUpdates: () => request<{
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
      `/api/v1/fs/browse${path ? `?path=${encodeURIComponent(path)}` : ''}`
    ),
  scanFS: (root?: string) =>
    request<import('../types').FSScanResult[]>(
      `/api/v1/fs/scan${root ? `?root=${encodeURIComponent(root)}` : ''}`
    ),
  inspectFS: (path: string) =>
    request<import('../types').FSInspectResult>(
      `/api/v1/fs/inspect?path=${encodeURIComponent(path)}`
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
      }
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
      `/api/v1/intelligence${projectId ? `?project_id=${encodeURIComponent(projectId)}` : ''}`
    ),
  updateIntelligence: (
    data: Omit<import('../types').IntelligenceStatus, 'available' | 'error'>,
    projectId?: string
  ) =>
    request<import('../types').IntelligenceStatus>(
      `/api/v1/intelligence${projectId ? `?project_id=${encodeURIComponent(projectId)}` : ''}`,
      { method: 'PUT', body: JSON.stringify(data) }
    ),
  getClarification: (id: string) =>
    request<import('../types').ClarificationCheckpoint>(`/api/v1/clarifications/${id}`),
  resolveClarification: (id: string, answers: Record<string, string>) =>
    request<{ plan: import('../types').WorkPlan; clarification: import('../types').ClarificationCheckpoint }>(
      `/api/v1/clarifications/${id}/resolve`,
      { method: 'POST', body: JSON.stringify({ answers }) }
    ),

  // WorkPlans & Autonomous Mission Runner (Phase D, E, F, H)
  getPlans: (projectId: string) =>
    request<import('../types').WorkPlan[]>(`/api/v1/projects/${projectId}/plans`),
  createPlan: (projectId: string, data: { title?: string; description?: string; goal?: string; auto_plan?: boolean; phases?: any[]; facts?: any }) =>
    request<import('../types').WorkPlan>(`/api/v1/projects/${projectId}/plans`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  getPlan: (planId: string) =>
    request<{ plan: import('../types').WorkPlan; revisions: import('../types').PlanRevision[] }>(`/api/v1/plans/${planId}`),
  updatePlan: (planId: string, plan: import('../types').WorkPlan, change_summary?: string) =>
    request<{ plan: import('../types').WorkPlan; revision: import('../types').PlanRevision }>(`/api/v1/plans/${planId}`, {
      method: 'PUT',
      body: JSON.stringify({ plan, change_summary }),
    }),
  deletePlan: (planId: string) =>
    request<{ deleted: boolean }>(`/api/v1/plans/${planId}`, { method: 'DELETE' }),
  restorePlanRevision: (planId: string, revision: number) =>
    request<{ plan: import('../types').WorkPlan; revision: import('../types').PlanRevision }>(`/api/v1/plans/${planId}/restore`, {
      method: 'POST', body: JSON.stringify({ revision }),
    }),
  diffPlanRevisions: (planId: string, from: number, to: number) =>
    request<import('../types').PlanRevisionDiff>(`/api/v1/plans/${planId}/diff?from=${from}&to=${to}`),
  compilePackagePrompt: (planId: string, packageId: string, phaseId?: string) =>
    request<any>(`/api/v1/plans/${planId}/compile`, {
      method: 'POST',
      body: JSON.stringify({ package_id: packageId, phase_id: phaseId }),
    }),
  runPlan: (planId: string, options?: { agentId?: string; contract?: AutonomyContract; autonomous?: boolean; approvedRevision?: number }) =>
    request<import('../types').MissionRun>(`/api/v1/plans/${planId}/run`, {
      method: 'POST',
      body: JSON.stringify({
        agent_id: options?.agentId,
        contract: options?.contract,
        autonomous: options?.autonomous ?? true,
        approved_revision: options?.approvedRevision,
      }),
    }),
  getRuns: () =>
    request<import('../types').MissionRun[]>('/api/v1/runs'),
  getRun: (runId: string) =>
    request<import('../types').MissionRun>(`/api/v1/runs/${runId}`),
  stepRun: (runId: string) =>
    request<{ run: import('../types').MissionRun; completed: boolean }>(`/api/v1/runs/${runId}/step`, { method: 'POST' }),
  pauseRun: (runId: string, reason?: string) =>
    request<import('../types').MissionRun>(`/api/v1/runs/${runId}/pause`, { method: 'POST', body: JSON.stringify({ reason }) }),
  takeControlRun: (runId: string, reason?: string) =>
    request<import('../types').MissionRun>(`/api/v1/runs/${runId}/take-control`, { method: 'POST', body: JSON.stringify({ reason }) }),
  resumeRun: (runId: string) =>
    request<import('../types').MissionRun>(`/api/v1/runs/${runId}/resume`, { method: 'POST' }),
  returnToMission: (runId: string) =>
    request<import('../types').MissionRun>(`/api/v1/runs/${runId}/return-to-mission`, { method: 'POST' }),
  cancelRun: (runId: string, reason?: string) =>
    request<import('../types').MissionRun>(`/api/v1/runs/${runId}/cancel`, { method: 'POST', body: JSON.stringify({ reason }) }),
  getSchedules: (projectId: string) =>
    request<import('../types').MissionSchedule[]>(`/api/v1/schedules?project_id=${encodeURIComponent(projectId)}`),
  schedulePlan: (planId: string, mode: 'AT' | 'AFTER_RUN' | 'WHEN_RESOURCES', options?: { scheduledFor?: string; afterRunId?: string; agentId?: string; contract?: AutonomyContract; approvedRevision?: number }) =>
    request<import('../types').MissionSchedule>('/api/v1/schedules', {
      method: 'POST',
      body: JSON.stringify({
        plan_id: planId,
        mode,
        scheduled_for: options?.scheduledFor,
        after_run_id: options?.afterRunId,
        agent_id: options?.agentId,
        contract: options?.contract,
        approved_revision: options?.approvedRevision,
      }),
    }),
  cancelSchedule: (scheduleId: string) =>
    request<{ canceled: boolean }>('/api/v1/schedules', { method: 'POST', body: JSON.stringify({ cancel_id: scheduleId }) }),
  recommendResources: (requirements: any, policy?: string) =>
    request<import('../types').RecommendationResult>('/api/v1/resources/recommend', {
      method: 'POST',
      body: JSON.stringify({ requirements, policy }),
    }),
};

export const nexusApi = nexus;
