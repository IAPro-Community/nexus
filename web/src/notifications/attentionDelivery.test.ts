import { describe, expect, it } from 'vitest';
import type { RuntimeSession } from '../types';
import {
  ATTENTION_ATTACH_WS_ROLE,
  ATTENTION_PUSH_AUTHORITY,
  attentionPushKey,
  isPtyAttentionFocused,
} from './attentionDelivery';

describe('attention delivery contract', () => {
  it('does not suppress identical questions from separate agent runtimes', () => {
    const first = {
      runtime_id: 'agent-1',
      state: 'WAITING',
      attention_reason: 'QUESTION',
      attention_kind: 'needs_user',
      attention_fingerprint: 'same-provider-text',
      attention_revision: 4,
      attention_generation: '9007199254740992:42',
    } as RuntimeSession;
    const second = { ...first, runtime_id: 'agent-2' };
    expect(attentionPushKey(first)).not.toBe(attentionPushKey(second));
  });

  it('re-notifies a new question episode even when the exact text repeats', () => {
    const original = {
      runtime_id: 'agent-a',
      attention_reason: 'QUESTION',
      attention_kind: 'needs_user',
      attention_fingerprint: 'repeat',
      attention_revision: 1,
      attention_generation: '9007199254740992:42',
    } as RuntimeSession;
    const samePoll = { ...original };
    const newEpisode = { ...original, attention_revision: 2 };
    const newGeneration = {
      ...original,
      attention_generation: '9007199254740993:42',
    };
    expect(attentionPushKey(original)).toBe(attentionPushKey(samePoll));
    expect(attentionPushKey(original)).not.toBe(attentionPushKey(newEpisode));
    expect(attentionPushKey(original)).not.toBe(attentionPushKey(newGeneration));
    expect(attentionPushKey(original)).not.toContain('\0');
  });

  it('distinguishes a question from an error with the same provider context', () => {
    const question = {
      runtime_id: 'agent-a',
      attention_reason: 'QUESTION',
      attention_fingerprint: 'shared-text',
    } as RuntimeSession;
    expect(attentionPushKey(question)).not.toBe(
      attentionPushKey({ ...question, attention_reason: 'ERROR' }),
    );
  });

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
