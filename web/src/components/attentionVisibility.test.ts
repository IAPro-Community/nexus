import { describe, expect, it } from 'vitest';
import { isVisibleRuntimeAttention } from './attentionVisibility';

describe('isVisibleRuntimeAttention', () => {
  it('does not resurrect attention from a stopped or stale runtime', () => {
    expect(isVisibleRuntimeAttention({ state: 'STOPPED', attention_reason: 'QUESTION' })).toBe(false);
    expect(isVisibleRuntimeAttention({ state: 'STALE', attention_reason: 'APPROVAL' })).toBe(false);
  });

  it('shows active questions and approvals', () => {
    expect(isVisibleRuntimeAttention({ state: 'WAITING', attention_reason: 'QUESTION' })).toBe(true);
    expect(isVisibleRuntimeAttention({ state: 'APPROVAL', attention_reason: 'APPROVAL' })).toBe(true);
  });
});
