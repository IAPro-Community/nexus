import type { EventRecord } from '../types';

export type DurableActivityEvent = {
  id: string;
  correlation_id?: string;
  agent_id?: string;
  project_id?: string;
  kind: string;
  timestamp: string;
  summary: string;
};

export function mapDurableActivity(event: DurableActivityEvent): EventRecord {
  return {
    id: event.id,
    correlation_id: event.correlation_id,
    runtime_id: event.agent_id || 'system',
    provider_id: 'system',
    provider: 'system',
    profile: 'default',
    type: event.kind,
    summary: event.summary,
    timestamp: event.timestamp,
    data: {},
  };
}
