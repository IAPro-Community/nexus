import { describe, expect, it } from 'vitest';
import { flowRunStateFromMission, packageRunState } from './flowRunModel';

describe('Flow Run state mapping', () => {
  it('maps durable Mission states to the canonical user states', () => {
    expect(flowRunStateFromMission('PENDING')).toBe('QUEUED');
    expect(flowRunStateFromMission('READY')).toBe('READY');
    expect(flowRunStateFromMission('ALLOCATING')).toBe('RUNNING');
    expect(flowRunStateFromMission('COMPILING')).toBe('RUNNING');
    expect(flowRunStateFromMission('EXECUTING')).toBe('RUNNING');
    expect(flowRunStateFromMission('TESTING')).toBe('VERIFYING');
    expect(flowRunStateFromMission('REVIEWING')).toBe('VERIFYING');
    expect(flowRunStateFromMission('REMEDIATING')).toBe('VERIFYING');
    expect(flowRunStateFromMission('COMPLETED_VERIFIED')).toBe('COMPLETED');
    expect(flowRunStateFromMission('FAILED_VERIFICATION')).toBe('FAILED');
    expect(flowRunStateFromMission('FAILED_NO_PROGRESS')).toBe('FAILED');
    expect(flowRunStateFromMission('FAILED_BUDGET_EXCEEDED')).toBe('FAILED');
    expect(flowRunStateFromMission('FAILED')).toBe('FAILED');
    expect(flowRunStateFromMission('BLOCKED_NEEDS_USER')).toBe('BLOCKED');
    expect(flowRunStateFromMission('CANCELED_BY_USER')).toBe('CANCELED');
  });

  it('fails closed when a future/unknown internal state appears', () => {
    expect(flowRunStateFromMission('NEW_INTERNAL_STATE')).toBe('BLOCKED');
    expect(flowRunStateFromMission('NEW_INTERNAL_STATE')).not.toBe('COMPLETED');
  });

  it('maps verified package state to completed without promoting unknown package states', () => {
    expect(packageRunState('VERIFIED')).toBe('COMPLETED');
    expect(packageRunState('EXECUTING')).toBe('RUNNING');
    expect(packageRunState('SOMETHING_NEW')).toBe('BLOCKED');
  });
});
