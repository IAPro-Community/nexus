import { describe, expect, it } from 'vitest';
import type { ProjectFact } from '../../types';
import {
  formatProjectIdentity,
  presentProjectIntelligenceFacts,
} from './projectIntelligencePresentation';

const fact = (key: string, category = 'architecture'): ProjectFact => ({
  category,
  key,
  value_type: 'BOOL',
  value: true,
  basis: 'OBSERVED',
  confidence: 'MEDIUM',
  observed_at: '',
  provenance: [],
});

describe('project intelligence presentation', () => {
  it('shortens identity to branch and sha', () =>
    expect(
      formatProjectIdentity({
        repository_kind: 'git',
        state: 'CLEAN',
        identity_digest: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        complete: true,
        branch: 'main',
        head_sha: '0123456789abcdef',
      }),
    ).toBe('main · 0123456789ab'));

  it('truncates digest when git refs are missing', () =>
    expect(
      formatProjectIdentity({
        repository_kind: 'git',
        state: 'NON_GIT',
        identity_digest: '0123456789abcdef0123456789abcdef',
        complete: false,
      }),
    ).toBe('0123456789abcdef…'));

  it('collapses fixture-like module facts into one row', () => {
    const presented = presentProjectIntelligenceFacts([
      fact('tests.present', 'tests'),
      fact('module.src'),
      fact('module.benchmark-harness/fixtures/bug-context-noise-001/src'),
      fact('module.benchmark-harness/fixtures/bug-fix-auth/src'),
    ]);
    expect(presented.rows.some((row) => row.id === 'architecture:modules')).toBe(true);
    expect(presented.rows.some((row) => (row.label || '').includes('fixtures'))).toBe(false);
  });

  it('groups path-like convention facts instead of listing each file', () => {
    const presented = presentProjectIntelligenceFacts([
      fact('tests.present', 'tests'),
      fact('.github/workflows/ci.yml', 'ci'),
      fact('.github/workflows/release.yml', 'ci'),
      fact('DEV/AI_CONTROL_DEFERRED.md', 'conventions'),
      fact('DEV/AI_CONTROL_HARDENING_REPORT.md', 'conventions'),
      fact('AGENTS.md', 'conventions'),
    ]);
    expect(
      presented.rows.some(
        (row) => row.labelKind === 'categoryGroup' && row.category === 'conventions',
      ),
    ).toBe(true);
    expect(
      presented.rows.some((row) => row.labelKind === 'raw' && row.label === 'tests.present'),
    ).toBe(true);
    expect(presented.rows.some((row) => row.label.includes('DEV/'))).toBe(false);
    expect(presented.hiddenCount).toBe(0);
  });
});
