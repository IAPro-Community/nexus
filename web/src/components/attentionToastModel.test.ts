import { describe, expect, it } from 'vitest';
import type { RuntimeSession } from '../types';
import { attentionToastKey } from './attentionToastModel';

function waiting(runtime_id: string, attention_fingerprint: string): RuntimeSession {
  return {
    runtime_id,
    workspace: '/tmp',
    pid: 1,
    host_pid: 1,
    state: 'WAITING',
    control_level: 'EVENTS',
    control_endpoint: '',
    started_at: '',
    attention_kind: 'needs_user',
    attention_reason: 'QUESTION',
    prompt_kind: 'native_status',
    attention_context: 'Approve the change?',
    attention_fingerprint,
  };
}

describe('attention toast identity', () => {
  it('separates identical questions from two independent runtimes', () => {
    expect(attentionToastKey(waiting('rt-a', 'question-x'))).not.toBe(
      attentionToastKey(waiting('rt-b', 'question-x')),
    );
  });
  it('replaces a stale card when the same runtime asks a new question', () => {
    expect(attentionToastKey(waiting('rt-a', 'question-1'))).not.toBe(
      attentionToastKey(waiting('rt-a', 'question-2')),
    );
  });
  it('keeps the same identity through repeated polling of the same question', () => {
    expect(attentionToastKey(waiting('rt-a', 'question-1'))).toBe(
      attentionToastKey(waiting('rt-a', 'question-1')),
    );
  });
});
