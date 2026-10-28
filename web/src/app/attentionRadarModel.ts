import type { RuntimeSession } from '../types';
import { sanitizeAttentionText } from '../components/attentionText';
import {
  attentionFingerprintOf,
  isActiveAttentionRuntime,
  isHonestNeedsUser,
} from './documentTitle';

export type RadarBadge = 'needs_user' | 'working' | 'completed' | 'error' | 'idle';

export type RadarRuntimeItem = {
  runtimeId: string;
  agentId?: string;
  projectId: string;
  projectName: string;
  title: string;
  badge: RadarBadge;
  context?: string;
  fingerprint?: string;
  promptKind?: string;
};

export type RadarProjectGroup = {
  projectId: string;
  projectName: string;
  items: RadarRuntimeItem[];
  needsUserCount: number;
};

function badgeFor(runtime: RuntimeSession): RadarBadge {
  if (runtime.attention_kind === 'needs_user' || isHonestNeedsUser(runtime)) return 'needs_user';
  if (
    runtime.attention_kind === 'error' ||
    runtime.attention_reason === 'ERROR' ||
    runtime.state === 'FAILED'
  ) {
    return 'error';
  }
  if (runtime.attention_kind === 'completed' || runtime.attention_reason === 'TASK_COMPLETED') {
    return 'completed';
  }
  if (runtime.attention_kind === 'working' || runtime.attention_reason === 'WORKING') {
    return 'working';
  }
  return 'idle';
}

export function buildAttentionRadar(runtimes: RuntimeSession[]): RadarProjectGroup[] {
  const list = Array.isArray(runtimes) ? runtimes : [];
  const active = list.filter(isActiveAttentionRuntime);

  const groups = new Map<string, RadarProjectGroup>();
  for (const runtime of active) {
    const projectId = runtime.project_id || runtime.workspace || 'unknown';
    const projectName = runtime.project_name || projectId;
    const badge = badgeFor(runtime);
    const honest = isHonestNeedsUser(runtime);
    const sanitizedContext = honest ? sanitizeAttentionText(runtime.attention_context, '') : '';
    const messageKey = honest
      ? `${runtime.runtime_id}\0${attentionFingerprintOf(runtime)}`
      : undefined;
    const item: RadarRuntimeItem = {
      runtimeId: runtime.runtime_id,
      agentId: runtime.agent_id || undefined,
      projectId,
      projectName,
      title: runtime.dynamic_title || runtime.title || runtime.runtime_id,
      badge: honest ? 'needs_user' : badge === 'needs_user' ? 'idle' : badge,
      context: sanitizedContext || undefined,
      fingerprint: messageKey,
      promptKind: runtime.prompt_kind,
    };

    const existing = groups.get(projectId);
    if (!existing) {
      groups.set(projectId, {
        projectId,
        projectName,
        items: [item],
        needsUserCount: item.badge === 'needs_user' ? 1 : 0,
      });
      continue;
    }
    // Each independently actionable runtime gets a separate row and counter.
    if (item.badge === 'needs_user' && item.fingerprint) {
      const already = existing.items.some((other) => other.runtimeId === item.runtimeId);
      if (already) continue;
      existing.needsUserCount += 1;
    }
    existing.items.push(item);
  }

  return [...groups.values()].sort((a, b) => {
    if (b.needsUserCount !== a.needsUserCount) return b.needsUserCount - a.needsUserCount;
    return a.projectName.localeCompare(b.projectName);
  });
}

export type FocusAttentionTarget = {
  projectId?: string;
  agentId?: string;
  runtimeId: string;
};

export type FocusAttentionAction =
  | { type: 'switch-project'; projectId: string }
  | { type: 'open-agent-terminal'; agentId: string; title: string; runtimeId?: string }
  | { type: 'open-project-shell'; projectId: string; runtimeId: string; title: string }
  | { type: 'refresh-agents'; projectId: string };

export function planFocusAttention(
  target: FocusAttentionTarget,
  opts: {
    currentProjectId: string;
    runtime?: RuntimeSession;
    agentName?: string;
  },
): FocusAttentionAction[] {
  const actions: FocusAttentionAction[] = [];
  const projectId = target.projectId || opts.runtime?.project_id || '';
  if (projectId && projectId !== opts.currentProjectId) {
    actions.push({ type: 'switch-project', projectId });
  }

  const agentId = target.agentId || opts.runtime?.agent_id || '';
  const title =
    opts.agentName ||
    opts.runtime?.dynamic_title ||
    opts.runtime?.title ||
    (agentId ? `Agent ${agentId.slice(0, 8)}` : 'Project Shell');

  if (agentId) {
    actions.push({
      type: 'open-agent-terminal',
      agentId,
      title,
      runtimeId: target.runtimeId || opts.runtime?.runtime_id,
    });
  } else if (projectId) {
    actions.push({
      type: 'open-project-shell',
      projectId,
      runtimeId: target.runtimeId,
      title,
    });
  }

  if (projectId) {
    actions.push({ type: 'refresh-agents', projectId });
  }
  return actions;
}
