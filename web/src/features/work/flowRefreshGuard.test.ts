import { describe, expect, it } from 'vitest';
import { createRefreshGuard } from './flowRefreshGuard';

describe('createRefreshGuard', () => {
  it('discards a delayed earlier refresh when a newer one already resolved', async () => {
    const guard = createRefreshGuard();
    let applied: string | null = null;

    // Request A starts first but resolves late; request B starts after.
    const seqA = guard.next();
    const seqB = guard.next();

    const resolveA = Promise.resolve('stale-A').then((value) => {
      if (guard.isLatest(seqA)) applied = value;
    });
    const resolveB = Promise.resolve('fresh-B').then((value) => {
      if (guard.isLatest(seqB)) applied = value;
    });

    // B resolves first, A resolves after: final state must be B.
    await resolveB;
    await resolveA;
    expect(applied).toBe('fresh-B');
  });

  it('applies the only in-flight refresh', () => {
    const guard = createRefreshGuard();
    const seq = guard.next();
    expect(guard.isLatest(seq)).toBe(true);
  });

  it('invalidates every older sequence when a new request starts', () => {
    const guard = createRefreshGuard();
    const first = guard.next();
    guard.next();
    guard.next();
    expect(guard.isLatest(first)).toBe(false);
  });
});
