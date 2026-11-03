import { describe, expect, it } from 'vitest';
import { resolveTerminalInfo } from './terminalResolution';
import type { Agent, RuntimeSession } from '../../types';

describe('resolveTerminalInfo', () => {
  it('returns Manual when no ownerRuntimeId is provided', () => {
    const res = resolveTerminalInfo();
    expect(res.isAgent).toBe(false);
    expect(res.terminalLabel).toBe('Manual');
    expect(res.provider).toBe('');
    expect(res.profile).toBe('');
  });

  it('resolves agent name when runtime references an existing agent', () => {
    const runtimes = [
      {
        runtime_id: 'rt-12345678',
        agent_id: 'ag-1',
        provider_id: 'agy',
        workspace: '/ws',
        pid: 100,
        host_pid: 100,
        state: 'RUNNING' as const,
        control_level: 'full',
        control_endpoint: '',
        started_at: '2026-10-29',
      },
    ] as RuntimeSession[];

    const agents = [
      {
        id: 'ag-1',
        project_id: 'p-1',
        name: 'AGY (kiveromegasistemas)',
        role: 'engineer',
        status: 'active',
        continuity_status: 'ready',
        created_at: '',
        updated_at: '',
      },
    ] as Agent[];

    const res = resolveTerminalInfo('rt-12345678', runtimes, agents);
    expect(res.isAgent).toBe(true);
    expect(res.terminalLabel).toBe('AGY (kiveromegasistemas)');
    expect(res.provider).toBe('AGY');
    expect(res.agentName).toBe('AGY (kiveromegasistemas)');
    expect(res.tooltip).toContain('AGY (kiveromegasistemas)');
  });

  it('resolves provider and profile when no agent is registered', () => {
    const runtimes = [
      {
        runtime_id: 'rt-88889999',
        provider_id: 'codex',
        profile_id: 'default',
        workspace: '/ws',
        pid: 101,
        host_pid: 101,
        state: 'RUNNING' as const,
        control_level: 'full',
        control_endpoint: '',
        started_at: '2026-10-29',
      },
    ] as RuntimeSession[];

    const res = resolveTerminalInfo('rt-88889999', runtimes, []);
    expect(res.isAgent).toBe(true);
    expect(res.terminalLabel).toBe('CODEX');
    expect(res.provider).toBe('CODEX');
  });

  it('formats provider with custom profile', () => {
    const runtimes = [
      {
        runtime_id: 'rt-claude',
        provider_id: 'claude',
        profile_id: 'pro-team',
        workspace: '/ws',
        pid: 102,
        host_pid: 102,
        state: 'RUNNING' as const,
        control_level: 'full',
        control_endpoint: '',
        started_at: '2026-10-29',
      },
    ] as RuntimeSession[];

    const res = resolveTerminalInfo('rt-claude', runtimes, []);
    expect(res.isAgent).toBe(true);
    expect(res.terminalLabel).toBe('CLAUDE (pro-team)');
  });

  it('falls back to runtime title if no provider or agent is found', () => {
    const runtimes = [
      {
        runtime_id: 'rt-shell',
        title: 'Bash Build Shell',
        workspace: '/ws',
        pid: 103,
        host_pid: 103,
        state: 'RUNNING' as const,
        control_level: 'full',
        control_endpoint: '',
        started_at: '2026-10-29',
      },
    ] as RuntimeSession[];

    const res = resolveTerminalInfo('rt-shell', runtimes, []);
    expect(res.isAgent).toBe(true);
    expect(res.terminalLabel).toBe('Bash Build Shell');
  });

  it('falls back to truncated ID if runtime is not found in cache yet', () => {
    const res = resolveTerminalInfo('rt-unknown-abc12345', [], []);
    expect(res.isAgent).toBe(true);
    expect(res.terminalLabel).toBe('Terminal · rt-unk');
  });
});
