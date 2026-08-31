import type { Agent } from '../../types';

export type AskAgentAction = 'ASK' | 'START_AND_ASK' | 'UNAVAILABLE';

export function askAgentAction(agent: Pick<Agent, 'status'>): AskAgentAction {
  if (agent.status === 'WORKING') return 'ASK';
  if (agent.status === 'STOPPED' || agent.status === 'RECOVERABLE') return 'START_AND_ASK';
  return 'UNAVAILABLE';
}
