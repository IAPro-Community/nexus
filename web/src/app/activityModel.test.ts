import { describe, expect, it } from 'vitest';
import { buildAgentTimeline, mapDurableActivity, timelinePhase } from './activityModel';

describe('mapDurableActivity', () => {
  it('maps durable project events to the activity view contract', () => {
    expect(
      mapDurableActivity({
        id: 'evt-1',
        correlation_id: 'run-1',
        agent_id: 'agent-1',
        project_id: 'project-1',
        kind: 'FLOW_BLOCKED',
        ts: '2026-09-05T12:00:00Z',
        summary: 'Flow blocked on verification',
      }),
    ).toMatchObject({
      id: 'evt-1',
      correlation_id: 'run-1',
      type: 'FLOW_BLOCKED',
      runtime_id: 'agent-1',
      summary: 'Flow blocked on verification',
      timestamp: '2026-09-05T12:00:00Z',
    });
  });

  it('exposes source and runtime generation in the event payload', () => {
    const mapped = mapDurableActivity({
      id: 'evt-2',
      agent_id: 'agent-2',
      project_id: 'project-2',
      source: 'runtime',
      runtime_id: 'runtime-2',
      runtime_generation_id: 'gen-2',
      kind: 'RUNTIME_STARTED',
      ts: '2026-09-05T12:01:00Z',
      summary: 'Runtime started',
    });

    expect(mapped.runtime_id).toBe('runtime-2');
    expect(mapped.data).toMatchObject({
      source: 'runtime',
      runtime_generation_id: 'gen-2',
      agent_id: 'agent-2',
      project_id: 'project-2',
    });
  });
});

describe('timelinePhase', () => {
  it('classifies the launch, attach, quota, handoff and recovery phases', () => {
    expect(timelinePhase('PROCESS_STARTED')).toBe('launch');
    expect(timelinePhase('TERMINAL_LEASE_GRANTED')).toBe('attach');
    expect(timelinePhase('QUOTA_LOW')).toBe('quota');
    expect(timelinePhase('HANDOFF_COMPLETED')).toBe('handoff');
    expect(timelinePhase('RUNTIME_REPLACED')).toBe('recovery');
    expect(timelinePhase('QUOTA_FAILOVER_COMPLETED')).toBe('recovery');
    expect(timelinePhase('APPROVAL_REQUIRED')).toBe('approval');
    expect(timelinePhase('RUNTIME_FAILED')).toBe('error');
    expect(timelinePhase('FLOW_BLOCKED')).toBeNull();
  });
});

describe('buildAgentTimeline', () => {
  it('builds a per-Agent timeline with timestamp, source, generation and correlation', () => {
    const timeline = buildAgentTimeline([
      {
        id: 'evt-launch',
        agent_id: 'agent-1',
        source: 'runtime',
        runtime_id: 'runtime-1',
        runtime_generation_id: 'gen-1',
        correlation_id: 'lineage-1',
        kind: 'RUNTIME_STARTED',
        ts: '2026-09-05T12:00:00Z',
        summary: 'Runtime started',
      },
      {
        id: 'evt-attach',
        agent_id: 'agent-1',
        source: 'terminal',
        runtime_id: 'runtime-1',
        runtime_generation_id: 'gen-1',
        correlation_id: 'lineage-1',
        kind: 'TERMINAL_LEASE_GRANTED',
        ts: '2026-09-05T12:00:05Z',
        summary: 'Lease granted',
      },
      {
        id: 'evt-quota',
        agent_id: 'agent-1',
        source: 'quota',
        runtime_id: 'runtime-1',
        runtime_generation_id: 'gen-1',
        kind: 'QUOTA_LOW',
        ts: '2026-09-05T12:00:10Z',
        summary: 'Quota low',
      },
      {
        id: 'evt-handoff',
        agent_id: 'agent-1',
        source: 'handoff',
        runtime_id: 'runtime-2',
        runtime_generation_id: 'gen-2',
        correlation_id: 'lineage-1',
        kind: 'HANDOFF_COMPLETED',
        ts: '2026-09-05T12:00:20Z',
        summary: 'Handoff completed',
      },
      {
        id: 'evt-recovery',
        agent_id: 'agent-1',
        source: 'runtime',
        runtime_id: 'runtime-2',
        runtime_generation_id: 'gen-2',
        correlation_id: 'lineage-1',
        kind: 'RUNTIME_REPLACED',
        ts: '2026-09-05T12:00:25Z',
        summary: 'Runtime replaced',
      },
    ]);

    expect(timeline.map((entry) => entry.phase)).toEqual([
      'launch',
      'attach',
      'quota',
      'handoff',
      'recovery',
    ]);
    expect(timeline.every((entry) => entry.agentId === 'agent-1')).toBe(true);
    expect(timeline.every((entry) => entry.timestamp.length > 0)).toBe(true);
    expect(timeline.every((entry) => entry.source.length > 0)).toBe(true);
    expect(timeline.every((entry) => entry.runtimeGenerationId.length > 0)).toBe(true);
    expect(timeline[3].correlationId).toBe('lineage-1');
  });

  it('does not attribute events without Agent identity', () => {
    expect(
      buildAgentTimeline([
        {
          id: 'evt-account',
          source: 'quota',
          kind: 'QUOTA_LOW',
          ts: '2026-09-05T12:00:10Z',
          summary: 'Account quota low',
        },
      ]),
    ).toEqual([]);
  });
});
