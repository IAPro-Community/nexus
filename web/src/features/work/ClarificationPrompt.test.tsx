import { describe, expect, it } from 'vitest';
import { nextOptionFocusIndex, resolveClarificationAnswer } from './ClarificationPrompt';

describe('ClarificationPrompt helpers', () => {
  it('wraps keyboard focus for option listbox', () => {
    expect(nextOptionFocusIndex(0, 3, 1)).toBe(1);
    expect(nextOptionFocusIndex(2, 3, 1)).toBe(0);
    expect(nextOptionFocusIndex(0, 3, -1)).toBe(2);
  });

  it('resolves custom vs selected answer', () => {
    expect(resolveClarificationAnswer(false, 'ignored', 'make quality')).toBe('make quality');
    expect(resolveClarificationAnswer(true, '  custom gate  ', 'make quality')).toBe('custom gate');
  });
});
