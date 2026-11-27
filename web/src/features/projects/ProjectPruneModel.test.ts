import { describe, expect, it } from 'vitest';
import { pruneCandidates, pruneReasonLabel, type PruneCandidate } from './projectPruneModel';

const candidate = (over: Partial<PruneCandidate> = {}): PruneCandidate => ({
  id: 'prj_1',
  name: 'tmp',
  canonical_path: '/tmp/tmp.abc',
  agent_count: 0,
  layout_saved: false,
  event_count: 0,
  reason: 'never opened, no agents, no saved layout, no events',
  ...over,
});

describe('project prune model', () => {
  it('keeps only rows without any usage evidence', () => {
    const rows = [
      candidate(),
      candidate({ id: 'prj_2', agent_count: 1, reason: '' }),
      candidate({ id: 'prj_3', layout_saved: true, reason: '' }),
      candidate({ id: 'prj_4', event_count: 2, reason: '' }),
      candidate({ id: 'prj_5', last_opened_at: '2026-11-26T00:00:00Z', reason: '' }),
    ];
    expect(pruneCandidates(rows).map((row) => row.id)).toEqual(['prj_1']);
  });

  it('explains why a kept project is not removable', () => {
    expect(pruneReasonLabel(candidate())).toContain('never opened');
    expect(pruneReasonLabel(candidate({ reason: '', agent_count: 2, event_count: 5 }))).toBe(
      'kept: 2 agent(s), layout=false, 5 event(s), opened=false',
    );
  });
});
