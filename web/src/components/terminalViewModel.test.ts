import { describe, expect, it } from 'vitest';
import { normalizeOpenRuntimeIds, visibleTerminalRuntimes } from './terminalViewModel';
import type { RuntimeSession } from '../types';

const runtime = (runtime_id: string): RuntimeSession => ({ runtime_id }) as RuntimeSession;

describe('terminalViewModel', () => {
  it('drops stale and duplicate views while preserving the explicit order', () => {
    expect(normalizeOpenRuntimeIds(['a', 'a', 'gone'], [runtime('a'), runtime('b')])).toEqual([
      'a',
    ]);
  });

  it('opens the selected session without duplicating an existing view', () => {
    expect(normalizeOpenRuntimeIds(['a'], [runtime('a'), runtime('b')], 'b')).toEqual(['a', 'b']);
  });

  it('shows only explicitly opened sessions in a split', () => {
    const runtimes = ['a', 'b', 'c', 'd'].map(runtime);
    expect(
      visibleTerminalRuntimes(runtimes, ['c', 'a'], 'c', 'split-h').map((r) => r.runtime_id),
    ).toEqual(['c', 'a']);
    expect(
      visibleTerminalRuntimes(runtimes, ['d', 'b', 'a'], 'b', 'grid').map((r) => r.runtime_id),
    ).toEqual(['d', 'b', 'a']);
  });

  it('keeps the active runtime visible when it falls outside the split limit', () => {
    const runtimes = ['a', 'b', 'c', 'd', 'e'].map(runtime);
    expect(
      visibleTerminalRuntimes(runtimes, ['a', 'b', 'c'], 'c', 'split-h').map((r) => r.runtime_id),
    ).toEqual(['a', 'c']);
    expect(
      visibleTerminalRuntimes(runtimes, ['a', 'b', 'c', 'd', 'e'], 'e', 'grid').map(
        (r) => r.runtime_id,
      ),
    ).toEqual(['a', 'b', 'c', 'e']);
  });

  it('never renders every runtime as an implicit split view', () => {
    const runtimes = ['a', 'b', 'c'].map(runtime);
    expect(
      visibleTerminalRuntimes(runtimes, ['b'], 'b', 'single').map((item) => item.runtime_id),
    ).toEqual(['b']);
    expect(
      visibleTerminalRuntimes(runtimes, ['b'], 'b', 'split-h').map((item) => item.runtime_id),
    ).toEqual(['b']);
  });
});
