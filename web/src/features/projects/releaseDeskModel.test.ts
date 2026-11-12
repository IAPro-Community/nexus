import { describe, expect, it } from 'vitest';
import type { ReleaseDeskReport } from '../../types';
import {
  deskTone,
  formatApprovalDate,
  gateTone,
  shortSha,
  summarizeRelease,
} from './releaseDeskModel';

const report = (overrides: Partial<ReleaseDeskReport> = {}): ReleaseDeskReport => ({
  branch: 'release/v1.0.0',
  base_sha: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  head_sha: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
  commits: [
    { sha: 'c1', subject: 'feat(web): card', references: ['#1'] },
    { sha: 'c2', subject: 'fix(go): desk', references: [] },
  ],
  pull_requests: ['#193'],
  issues: ['#10', '#11'],
  gates: [
    { name: 'unit', verify_command: 'make test', status: 'verified' },
    { name: 'e2e', verify_command: 'make e2e', status: 'unverified' },
    { name: 'perf', verify_command: 'make perf', status: 'unverified' },
  ],
  approval: {
    approver: 'maestro',
    approval_id: 'GO-1',
    sha: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
    approved_at: '2026-11-10T12:00:00Z',
  },
  merge_publish_allowed: false,
  block_reasons: ['gates unverified'],
  ...overrides,
});

describe('releaseDeskModel', () => {
  it('summarizes gates, refs and verdict without inventing authorization', () => {
    const summary = summarizeRelease(report());
    expect(summary.totalGates).toBe(3);
    expect(summary.verifiedGates).toBe(1);
    expect(summary.commitCount).toBe(2);
    expect(summary.prCount).toBe(1);
    expect(summary.issueCount).toBe(2);
    expect(summary.hasApproval).toBe(true);
    expect(summary.mergePublishAllowed).toBe(false);
  });

  it('normalizes null-ish collections from the API', () => {
    const summary = summarizeRelease(
      report({
        commits: null as unknown as ReleaseDeskReport['commits'],
        pull_requests: null as unknown as ReleaseDeskReport['pull_requests'],
        issues: null as unknown as ReleaseDeskReport['issues'],
        gates: null as unknown as ReleaseDeskReport['gates'],
        approval: undefined,
        merge_publish_allowed: undefined,
      }),
    );
    expect(summary.commitCount).toBe(0);
    expect(summary.prCount).toBe(0);
    expect(summary.issueCount).toBe(0);
    expect(summary.totalGates).toBe(0);
    expect(summary.hasApproval).toBe(false);
    expect(summary.mergePublishAllowed).toBe(false);
  });

  it('maps tones without leaking pass/fail guesses', () => {
    expect(gateTone('verified')).toBe('success');
    expect(gateTone('unverified')).toBe('warning');
    expect(gateTone('anything-else')).toBe('warning');
    expect(deskTone(true)).toBe('success');
    expect(deskTone(false)).toBe('danger');
  });

  it('shortens SHAs for display', () => {
    expect(shortSha('abcdefghij0123456789abcdef0123456789abcd')).toBe('abcdefg');
    expect(shortSha('')).toBe('');
  });

  it('formats approval dates with Intl and never crashes on junk', () => {
    expect(formatApprovalDate('2026-11-10T12:00:00Z', 'en-US')).toMatch(/2026/);
    expect(formatApprovalDate('not-a-date', 'en-US')).toBe('not-a-date');
  });
});
