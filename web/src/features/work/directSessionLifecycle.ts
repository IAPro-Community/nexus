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
): Promise<Agent> {
  const created = await api.createAgent(input.projectId, input.name, 'developer');
  let attemptedStart = false;
  try {
    await api.selectResource(created.id, input.provider, input.profile, 'MANUAL');
    await api.applyAgentConfig(created.id, {
      provider: input.provider,
      profile: input.profile,
      ...(input.model ? { model: input.model } : {}),
      workspace: input.workspace,
      isolation: 'project',
    });
    attemptedStart = true;
    await api.startAgent(created.id);
    return created;
  } catch (reason) {
    const message = reason instanceof Error ? reason.message : String(reason);
    if (attemptedStart) {
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
    throw new DirectSessionSetupError(message, created.id, 'deleted');
  }
}
