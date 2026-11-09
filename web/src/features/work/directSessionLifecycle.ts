import type { Agent, AgentConfig } from '../../types';

export interface DirectSessionOperations {
  createAgent: (projectId: string, name: string, role?: string) => Promise<Agent>;
  selectResource: (
    agentId: string,
    provider: string,
    profile: string,
    policy: string,
  ) => Promise<unknown>;
  applyAgentConfig: (agentId: string, config: AgentConfig) => Promise<unknown>;
  startAgent: (agentId: string) => Promise<unknown>;
  stopAgent: (agentId: string) => Promise<unknown>;
  deleteAgent: (agentId: string) => Promise<unknown>;
}

export interface DirectSessionInput {
  projectId: string;
  name: string;
  provider: string;
  profile: string;
  model?: string;
  workspace?: string;
}

export class DirectSessionSetupError extends Error {
  constructor(
    message: string,
    readonly agentId: string,
    readonly recovery: 'deleted' | 'cleanup-failed' | 'start-uncertain',
  ) {
    super(message);
    this.name = 'DirectSessionSetupError';
  }
}

/**
 * A failed SELECT or CONFIG step cannot leave a newly created Agent orphaned.
 * Once start is attempted, runtime creation may succeed despite network errors:
 * NEVER delete that Agent without knowing its actual runtime state.
 */
export async function startDirectSession(
  api: DirectSessionOperations,
  input: DirectSessionInput,
  isCancellationRequested: () => boolean = () => false,
): Promise<Agent | null> {
  const name = input.name.trim();
  if (!name) {
    throw new Error('Direct session agent name is required');
  }
  const created = await api.createAgent(input.projectId, name, 'developer');
  let attemptedStart = false;
  try {
    if (isCancellationRequested()) {
      await api.deleteAgent(created.id);
      return null;
    }
    await api.selectResource(created.id, input.provider, input.profile, 'MANUAL');
    if (isCancellationRequested()) {
      await api.deleteAgent(created.id);
      return null;
    }
    await api.applyAgentConfig(created.id, {
      provider: input.provider,
      profile: input.profile,
      ...(input.model ? { model: input.model } : {}),
      workspace: input.workspace,
      isolation: 'project',
    });
    if (isCancellationRequested()) {
      await api.deleteAgent(created.id);
      return null;
    }
    attemptedStart = true;
    await api.startAgent(created.id);
    if (isCancellationRequested()) {
      await api.stopAgent(created.id);
      await api.deleteAgent(created.id);
      return null;
    }
    return created;
  } catch (reason) {
    const message = reason instanceof Error ? reason.message : String(reason);
    if (attemptedStart) {
      if (isCancellationRequested()) {
        try {
          await api.stopAgent(created.id);
          await api.deleteAgent(created.id);
          return null;
        } catch (cleanupReason) {
          const cleanup =
            cleanupReason instanceof Error ? cleanupReason.message : String(cleanupReason);
          throw new DirectSessionSetupError(
            `${message} (cancel cleanup: ${cleanup})`,
            created.id,
            'cleanup-failed',
          );
        }
      }
      throw new DirectSessionSetupError(message, created.id, 'start-uncertain');
    }
    try {
      await api.deleteAgent(created.id);
    } catch (cleanupReason) {
      const cleanup =
        cleanupReason instanceof Error ? cleanupReason.message : String(cleanupReason);
      throw new DirectSessionSetupError(
        `${message} (cleanup: ${cleanup})`,
        created.id,
        'cleanup-failed',
      );
    }
    if (isCancellationRequested()) return null;
    throw new DirectSessionSetupError(message, created.id, 'deleted');
  }
}
