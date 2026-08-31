import { describe, expect, it } from 'vitest';
import { askAgentAction } from './askAgentModel';

describe('askAgentAction', () => {
  it('asks the existing working agent without creating one', () => {
    expect(askAgentAction({ status: 'WORKING' })).toBe('ASK');
  });

  it('starts a stopped existing agent before asking', () => {
    expect(askAgentAction({ status: 'STOPPED' })).toBe('START_AND_ASK');
    expect(askAgentAction({ status: 'RECOVERABLE' })).toBe('START_AND_ASK');
  });

  it('does not offer ask while the agent is in an incompatible state', () => {
    expect(askAgentAction({ status: 'FAILED' })).toBe('UNAVAILABLE');
  });
});
