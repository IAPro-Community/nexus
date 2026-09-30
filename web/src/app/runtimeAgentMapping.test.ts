import { describe, expect, it } from 'vitest';
import {
  agentForRuntime,
  currentRuntimeForAgent,
  terminalSurfaceIDForRuntime,
} from './runtimeAgentMapping';

const agents = [
  { id: 'agt-a', project_id: 'p1', name: 'A' },
  { id: 'agt-b', project_id: 'p1', name: 'B' },
] as any[];

describe('runtime to Agent mapping', () => {
  it('maps only by explicit agent_id and never by project fallback', () => {
    expect(agentForRuntime({ runtime_id: 'rt-x', agent_id: 'agt-b' } as any, agents)?.id).toBe(
      'agt-b',
    );
    expect(agentForRuntime({ runtime_id: 'agt-a' } as any, agents)).toBeUndefined();
  });

  it('uses the canonical Agent terminal surface id', () => {
    expect(terminalSurfaceIDForRuntime({ runtime_id: 'rt-x', agent_id: 'agt-a' } as any)).toBe(
      'agent:agt-a:terminal',
    );
    expect(terminalSurfaceIDForRuntime({ runtime_id: 'rt-x' } as any)).toBeUndefined();
  });

  it('selects the newest live runtime generation regardless of registry order', () => {
    const runtimes = [
      {
        runtime_id: 'rt-old-active',
        agent_id: 'agt-a',
        state: 'RUNNING',
        started_at: '2026-09-29T10:00:00Z',
        provider_generation: 10,
      },
      {
        runtime_id: 'rt-new-active',
        agent_id: 'agt-a',
        state: 'RUNNING',
        started_at: '2026-09-29T11:00:00Z',
        provider_generation: 20,
      },
      {
        runtime_id: 'rt-newer-stale',
        agent_id: 'agt-a',
        state: 'STALE',
        started_at: '2026-09-29T12:00:00Z',
        provider_generation: 30,
      },
    ] as any[];

    expect(currentRuntimeForAgent(runtimes, 'agt-a')?.runtime_id).toBe('rt-new-active');
  });

  it('uses process generation as a deterministic tie-breaker', () => {
    const runtimes = [
      {
        runtime_id: 'rt-generation-1',
        agent_id: 'agt-a',
        state: 'RUNNING',
        started_at: '2026-09-29T11:00:00Z',
        provider_generation: 100,
      },
      {
        runtime_id: 'rt-generation-2',
        agent_id: 'agt-a',
        state: 'RUNNING',
        started_at: '2026-09-29T11:00:00Z',
        provider_generation: 200,
      },
    ] as any[];

    expect(currentRuntimeForAgent(runtimes, 'agt-a')?.runtime_id).toBe('rt-generation-2');
  });
});
