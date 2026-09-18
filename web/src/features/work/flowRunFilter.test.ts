import { describe, expect, it } from 'vitest';
import { flowRunStateFromMission } from './flowRunModel';

describe('flow run filters', () => {
  it('maps executing missions to RUNNING for active filter', () => {
    expect(flowRunStateFromMission('EXECUTING')).toBe('RUNNING');
    expect(flowRunStateFromMission('ALLOCATING')).toBe('RUNNING');
    expect(flowRunStateFromMission('PAUSED')).toBe('BLOCKED');
  });

  it('includes RUNNING and BLOCKED in the active set used by history UI', () => {
    const active = new Set(['QUEUED', 'READY', 'RUNNING', 'VERIFYING', 'BLOCKED']);
    expect(active.has(flowRunStateFromMission('EXECUTING'))).toBe(true);
    expect(active.has(flowRunStateFromMission('PAUSED'))).toBe(true);
    expect(active.has(flowRunStateFromMission('COMPLETED_VERIFIED'))).toBe(false);
  });
});
