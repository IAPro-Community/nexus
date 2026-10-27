import { describe, expect, it } from 'vitest';
import {
  ATTENTION_ATTACH_WS_ROLE,
  ATTENTION_PUSH_AUTHORITY,
  isPtyAttentionFocused,
} from './attentionDelivery';

describe('attention delivery contract', () => {
  it('documents a single browser push authority and display-only attach WS', () => {
    expect(ATTENTION_PUSH_AUTHORITY).toBe('universal-registry-poll');
    expect(ATTENTION_ATTACH_WS_ROLE).toBe('display-only');
  });

  it('suppresses push only when Terminais tab and matching PTY are focused', () => {
    const stackActiveIds = new Set(['project:p1:terminals']);
    expect(
      isPtyAttentionFocused({
        terminalsProductSurfaceId: 'project:p1:terminals',
        agentViewId: 'view:agent:a1:terminal',
        stackActiveIds,
        activePtyViewId: 'view:agent:a1:terminal',
      }),
    ).toBe(true);
    expect(
      isPtyAttentionFocused({
        terminalsProductSurfaceId: 'project:p1:terminals',
        agentViewId: 'view:agent:a1:terminal',
        stackActiveIds: new Set(['project:p1:overview']),
        activePtyViewId: 'view:agent:a1:terminal',
      }),
    ).toBe(false);
    expect(
      isPtyAttentionFocused({
        terminalsProductSurfaceId: 'project:p1:terminals',
        agentViewId: 'view:agent:a1:terminal',
        stackActiveIds,
        activePtyViewId: 'view:agent:a2:terminal',
      }),
    ).toBe(false);
  });
});
