import { describe, expect, it, vi } from 'vitest';
import type { Agent, RuntimeSession } from '../../types';
import { attachOverviewRuntime } from './overviewRuntimeActions';

const runtime: RuntimeSession = {
  runtime_id: 'rt_existing_shell',
  workspace: '/workspace/nexus',
  pid: 42,
  host_pid: 42,
  state: 'RUNNING',
  control_level: 'FULL',
  control_endpoint: '',
  started_at: '2026-10-25T12:00:00Z',
  project_id: 'project_nexus',
  title: 'Existing shell',
};

const agent: Agent = {
  id: 'agent_nexus',
  project_id: 'project_nexus',
  name: 'NEX',
  role: 'assistant',
  status: 'WORKING',
  continuity_status: 'ACTIVE',
  created_at: '2026-10-25T12:00:00Z',
  updated_at: '2026-10-25T12:00:00Z',
};

describe('overview runtime attachment', () => {
  it('attaches an existing shell runtime without starting another shell', () => {
    const onAttachRuntime = vi.fn();

    attachOverviewRuntime(runtime, undefined, {
      onOpenAgent: vi.fn(),
      onAttachRuntime,
    });

    expect(onAttachRuntime).toHaveBeenCalledWith(runtime);
  });

  it('opens an agent runtime through its existing agent terminal route', () => {
    const onOpenAgent = vi.fn();

    attachOverviewRuntime(runtime, agent, {
      onOpenAgent,
      onAttachRuntime: vi.fn(),
    });

    expect(onOpenAgent).toHaveBeenCalledWith(agent, runtime.runtime_id);
  });
});
