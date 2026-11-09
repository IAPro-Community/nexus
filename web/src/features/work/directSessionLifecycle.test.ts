import { describe, expect, it, vi } from 'vitest';
import type { Agent } from '../../types';
import {
  startDirectSession,
  DirectSessionSetupError,
  type DirectSessionOperations,
} from './directSessionLifecycle';

const agent = { id: 'agent-a', project_id: 'project-a' } as Agent;
const input = {
  projectId: 'project-a',
  name: 'Review agent',
  provider: 'codex',
  profile: 'work',
  model: 'example-model',
  workspace: '/repo',
};

function operations() {
  return {
    createAgent: vi.fn().mockResolvedValue(agent),
    selectResource: vi.fn().mockResolvedValue({}),
    applyAgentConfig: vi.fn().mockResolvedValue({}),
    startAgent: vi.fn().mockResolvedValue({}),
    stopAgent: vi.fn().mockResolvedValue({}),
    deleteAgent: vi.fn().mockResolvedValue({}),
  } satisfies DirectSessionOperations;
}

describe('Direct session transactional setup', () => {
  it('creates, allocates, configures and starts the requested resource', async () => {
    const api = operations();
    await expect(startDirectSession(api, { ...input, name: '  Review agent  ' })).resolves.toBe(
      agent,
    );
    expect(api.createAgent).toHaveBeenCalledWith('project-a', 'Review agent', 'developer');
    expect(api.selectResource).toHaveBeenCalledWith('agent-a', 'codex', 'work', 'MANUAL');
    expect(api.applyAgentConfig).toHaveBeenCalledWith(
      'agent-a',
      expect.objectContaining({
        provider: 'codex',
        profile: 'work',
        model: 'example-model',
        isolation: 'project',
      }),
    );
    expect(api.startAgent).toHaveBeenCalledWith('agent-a');
    expect(api.deleteAgent).not.toHaveBeenCalled();
  });
  it('rejects a blank name before making an API request', async () => {
    const api = operations();
    await expect(startDirectSession(api, { ...input, name: '  ' })).rejects.toThrow(
      'Direct session agent name is required',
    );
    expect(api.createAgent).not.toHaveBeenCalled();
  });
  it.each(['selectResource', 'applyAgentConfig'] as const)(
    'rolls back the unstarted Agent when %s fails',
    async (step) => {
      const api = operations();
      api[step].mockRejectedValue(new Error('config unavailable'));
      await expect(startDirectSession(api, input)).rejects.toMatchObject({
        agentId: 'agent-a',
        recovery: 'deleted',
      });
      expect(api.deleteAgent).toHaveBeenCalledWith('agent-a');
      expect(api.startAgent).not.toHaveBeenCalled();
    },
  );
  it('reports failed cleanup and preserves the Agent identity for recovery', async () => {
    const api = operations();
    api.selectResource.mockRejectedValue(new Error('selection rejected'));
    api.deleteAgent.mockRejectedValue(new Error('cannot delete'));
    await expect(startDirectSession(api, input)).rejects.toMatchObject({
      agentId: 'agent-a',
      recovery: 'cleanup-failed',
    });
  });
  it('never deletes a runtime when its start response is uncertain', async () => {
    const api = operations();
    api.startAgent.mockRejectedValue(new Error('network timeout'));
    await expect(startDirectSession(api, input)).rejects.toMatchObject({
      agentId: 'agent-a',
      recovery: 'start-uncertain',
    });
    expect(api.deleteAgent).not.toHaveBeenCalled();
  });
  it('deletes the unstarted Agent when cancellation is requested during setup', async () => {
    const api = operations();
    let cancelled = false;
    api.selectResource.mockImplementation(async () => {
      cancelled = true;
      return {};
    });

    await expect(startDirectSession(api, input, () => cancelled)).resolves.toBeNull();
    expect(api.deleteAgent).toHaveBeenCalledWith('agent-a');
    expect(api.applyAgentConfig).not.toHaveBeenCalled();
    expect(api.startAgent).not.toHaveBeenCalled();
    expect(api.stopAgent).not.toHaveBeenCalled();
  });
  it('treats a failed setup stage as a completed cancellation after rollback', async () => {
    const api = operations();
    let cancelled = false;
    api.selectResource.mockImplementation(async () => {
      cancelled = true;
      throw new Error('selection was interrupted');
    });

    await expect(startDirectSession(api, input, () => cancelled)).resolves.toBeNull();
    expect(api.deleteAgent).toHaveBeenCalledWith('agent-a');
    expect(api.startAgent).not.toHaveBeenCalled();
  });
  it('stops and deletes the Agent when cancellation arrives during start', async () => {
    const api = operations();
    let resolveStart: (() => void) | undefined;
    api.startAgent.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveStart = resolve;
        }),
    );
    let cancelled = false;

    const result = startDirectSession(api, input, () => cancelled);
    await vi.waitFor(() => expect(resolveStart).toBeDefined());
    cancelled = true;
    resolveStart?.();

    await expect(result).resolves.toBeNull();
    expect(api.stopAgent).toHaveBeenCalledWith('agent-a');
    expect(api.deleteAgent).toHaveBeenCalledWith('agent-a');
  });
  it('preserves the Agent when runtime cleanup fails after cancellation', async () => {
    const api = operations();
    api.stopAgent.mockRejectedValue(new Error('stop could not be verified'));
    let cancelled = false;
    api.startAgent.mockImplementation(async () => {
      cancelled = true;
      return {};
    });

    await expect(startDirectSession(api, input, () => cancelled)).rejects.toMatchObject({
      agentId: 'agent-a',
      recovery: 'cleanup-failed',
    });
    expect(api.stopAgent).toHaveBeenCalledWith('agent-a');
    expect(api.deleteAgent).not.toHaveBeenCalled();
  });
  it('never cleans up an Agent when creation itself fails', async () => {
    const api = operations();
    api.createAgent.mockRejectedValue(new Error('create failed'));
    await expect(startDirectSession(api, input)).rejects.toThrow('create failed');
    expect(api.deleteAgent).not.toHaveBeenCalled();
  });
  it('uses the provider default model when no model is selected', async () => {
    const api = operations();
    await startDirectSession(api, { ...input, model: undefined });
    const config = api.applyAgentConfig.mock.calls[0][1];
    expect(config).not.toHaveProperty('model');
  });
  it('labels recoverable failures with stable identities', () => {
    const error = new DirectSessionSetupError('failure', 'agent-a', 'start-uncertain');
    expect(error.agentId).toBe('agent-a');
  });
});
