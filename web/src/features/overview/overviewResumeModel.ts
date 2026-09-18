import type { Agent, RuntimeSession } from '../../types';

export type ResumeLane = 'needsYou' | 'active' | 'recoverable' | 'recent';

export interface ResumeItem {
  lane: ResumeLane;
  runtime: RuntimeSession;
  agent?: Agent;
}

const attentionStates = new Set(['APPROVAL', 'WAITING']);
const activeStates = new Set(['STARTING', 'RUNNING', 'HANDOFF']);
const recoverableStates = new Set(['FAILED', 'STALE', 'STOPPED']);

export function buildOverviewResume(
  projectId: string,
  runtimes: RuntimeSession[],
  agents: Agent[],
): ResumeItem[] {
  const agentById = new Map(
    (Array.isArray(agents) ? agents : []).map((agent) => [agent.id, agent]),
  );
  return (Array.isArray(runtimes) ? runtimes : [])
    .filter((runtime) => !runtime.project_id || runtime.project_id === projectId)
    .map((runtime) => {
      const agent = runtime.agent_id ? agentById.get(runtime.agent_id) : undefined;
      const lane: ResumeLane =
        runtime.attention_kind === 'needs_user' || attentionStates.has(runtime.state)
          ? 'needsYou'
          : activeStates.has(runtime.state)
            ? 'active'
            : recoverableStates.has(runtime.state) || agent?.status === 'RECOVERABLE'
              ? 'recoverable'
              : 'recent';
      return { lane, runtime, agent };
    })
    .sort(
      (a, b) => Date.parse(b.runtime.started_at || '') - Date.parse(a.runtime.started_at || ''),
    );
}
