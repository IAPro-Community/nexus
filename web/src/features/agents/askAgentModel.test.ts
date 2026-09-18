import { describe, expect, it } from 'vitest';
import { askActionForStatus } from './askAgentModel';

describe('Ask existing Agent', () => {
  it('reuses an active Agent without asking Nexus to start another runtime', () => {
    expect(askActionForStatus('WORKING')).toEqual({ label: 'Prepare Task', startIfNeeded: false });
    expect(askActionForStatus('WAITING')).toEqual({ label: 'Prepare Task', startIfNeeded: false });
  });

  it('uses Start & Send Task for stopped or recoverable Agents without changing Agent identity', () => {
    expect(askActionForStatus('STOPPED')).toEqual({
      label: 'Start & Send Task',
      startIfNeeded: true,
    });
    expect(askActionForStatus('RECOVERABLE')).toEqual({
      label: 'Start & Send Task',
      startIfNeeded: true,
    });
  });
});
