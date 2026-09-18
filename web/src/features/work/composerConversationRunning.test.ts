import { describe, expect, it } from 'vitest';

/**
 * Chat-shaped Composer ops may flip assistant-ui `isRunning`. Mutations that
 * only update the brief (resolve/attach/skill) must not — ExternalStore +
 * Viewport MutationObserver auto-scroll remounts the thread and freezes Chrome.
 */
export function marksConversationRunning(
  operation: 'create' | 'send' | 'resolve' | 'attach' | 'skill' | 'finalize' | 'refine',
): boolean {
  return operation === 'create' || operation === 'send';
}

describe('composer conversation running gate', () => {
  it('only marks create/send as conversation runs for assistant-ui isRunning', () => {
    expect(marksConversationRunning('create')).toBe(true);
    expect(marksConversationRunning('send')).toBe(true);
    expect(marksConversationRunning('resolve')).toBe(false);
    expect(marksConversationRunning('attach')).toBe(false);
    expect(marksConversationRunning('skill')).toBe(false);
    expect(marksConversationRunning('finalize')).toBe(false);
    expect(marksConversationRunning('refine')).toBe(false);
  });
});
