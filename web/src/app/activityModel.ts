import type { EventRecord } from '../types';

export type DurableActivityEvent = {
  id: string;
  correlation_id?: string;
  agent_id?: string;
  project_id?: string;
  source?: string;
  runtime_id?: string;
  runtime_generation_id?: string;
  kind: string;
  /** Durable projection field name. */
  ts: string;
  /** Legacy alias accepted by older payloads. */
  timestamp?: string;
  summary?: string;
};

export function durableActivityTimestamp(event: DurableActivityEvent): string {
  return event.ts || event.timestamp || '';
}

export function mapDurableActivity(event: DurableActivityEvent): EventRecord {
  return {
    id: event.id,
    correlation_id: event.correlation_id,
    runtime_id: event.runtime_id || event.agent_id || 'system',
    provider_id: 'system',
    provider: 'system',
    profile: 'default',
    type: event.kind,
    summary: event.summary ?? '',
    timestamp: durableActivityTimestamp(event),
    data: {
      source: event.source ?? '',
      runtime_generation_id: event.runtime_generation_id ?? '',
      agent_id: event.agent_id ?? '',
      project_id: event.project_id ?? '',
    },
  };
}

/** Timeline phases covered by the Agent timeline contract. */
export type TimelinePhase =
  'launch' | 'attach' | 'quota' | 'handoff' | 'recovery' | 'error' | 'approval';

export interface AgentTimelineEntry {
  id: string;
  agentId: string;
  phase: TimelinePhase;
  kind: string;
  source: string;
  runtimeGenerationId: string;
  correlationId: string;
  timestamp: string;
  summary: string;
}

const timelinePhases: Readonly<Record<string, TimelinePhase>> = {
  PROCESS_STARTED: 'launch',
  RUNTIME_STARTED: 'launch',
  SESSION_STARTED: 'launch',
  TERMINAL_LEASE_GRANTED: 'attach',
  TERMINAL_LEASE_REVOKED: 'attach',
  TERMINAL_LEASE_RELEASED: 'attach',
  RATE_LIMITED: 'quota',
  QUOTA_LOW: 'quota',
  QUOTA_EXHAUSTED: 'quota',
  QUOTA_MONITOR_DEGRADED: 'quota',
  QUOTA_MONITOR_RECOVERED: 'quota',
  HANDOFF_COMPLETED: 'handoff',
  RUNTIME_REPLACED: 'recovery',
  QUOTA_FAILOVER_REQUESTED: 'recovery',
  QUOTA_FAILOVER_COMPLETED: 'recovery',
  QUOTA_FAILOVER_FAILED: 'recovery',
  MISSION_RESUME_REQUESTED: 'recovery',
  MISSION_RESUMED: 'recovery',
  SESSION_RESUMED: 'recovery',
  ERROR: 'error',
  RUNTIME_FAILED: 'error',
  PROCESS_EXITED: 'error',
  APPROVAL_REQUIRED: 'approval',
  APPROVED: 'approval',
  REJECTED: 'approval',
};

export function timelinePhase(kind: string): TimelinePhase | null {
  return timelinePhases[kind] ?? null;
}

/**
 * Builds the verifiable per-Agent timeline. Events without an Agent identity
 * are not attributed to a timeline lane; no synthetic Agent is created.
 */
export function buildAgentTimeline(events: DurableActivityEvent[]): AgentTimelineEntry[] {
  const entries: AgentTimelineEntry[] = [];
  for (const event of events) {
    const phase = timelinePhase(event.kind);
    if (!phase || !event.agent_id) continue;
    entries.push({
      id: event.id,
      agentId: event.agent_id,
      phase,
      kind: event.kind,
      source: event.source ?? '',
      runtimeGenerationId: event.runtime_generation_id ?? '',
      correlationId: event.correlation_id ?? '',
      timestamp: durableActivityTimestamp(event),
      summary: event.summary ?? '',
    });
  }
  return entries.sort((left, right) => left.timestamp.localeCompare(right.timestamp));
}
