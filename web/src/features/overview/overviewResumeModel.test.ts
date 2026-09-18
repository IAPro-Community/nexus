import { describe, expect, it } from 'vitest';
import { buildOverviewResume } from './overviewResumeModel';
import type { Agent, RuntimeSession } from '../../types';

const runtime = (overrides: Partial<RuntimeSession>): RuntimeSession => ({
  runtime_id: 'runtime',
  workspace: '/workspace',
  pid: 1,
  host_pid: 1,
  state: 'RUNNING',
  control_level: 'structured',
  control_endpoint: 'local',
  started_at: '2026-09-07T10:00:00Z',
  ...overrides,
});

const agent = (overrides: Partial<Agent>): Agent => ({
  id: 'agent',
  project_id: 'project',
  name: 'Agent',
  role: 'Developer',
  status: 'WORKING',
  continuity_status: 'LIVE_SAME_RUNTIME',
  created_at: '',
  updated_at: '',
  ...overrides,
});

describe('buildOverviewResume', () => {
  it('prioritizes user attention, then active, recoverable and recent work', () => {
    const result = buildOverviewResume(
      'project',
      [
        runtime({ runtime_id: 'recent', state: 'DETACHED', started_at: '2026-09-07T14:00:00Z' }),
        runtime({ runtime_id: 'active', state: 'RUNNING', started_at: '2026-09-07T13:00:00Z' }),
        runtime({ runtime_id: 'needs', state: 'WAITING', started_at: '2026-09-07T12:00:00Z' }),
        runtime({ runtime_id: 'failed', state: 'FAILED', started_at: '2026-09-07T11:00:00Z' }),
      ],
      [],
    );
    expect(result.map((item) => [item.runtime.runtime_id, item.lane])).toEqual([
      ['recent', 'recent'],
      ['active', 'active'],
      ['needs', 'needsYou'],
      ['failed', 'recoverable'],
    ]);
  });

  it('filters runtimes from another project and joins the owning agent', () => {
    const result = buildOverviewResume(
      'project',
      [
        runtime({ runtime_id: 'owned', agent_id: 'agent', project_id: 'project' }),
        runtime({ runtime_id: 'other', project_id: 'other' }),
      ],
      [agent({ id: 'agent' })],
    );
    expect(result).toHaveLength(1);
    expect(result[0].agent?.name).toBe('Agent');
  });
});
