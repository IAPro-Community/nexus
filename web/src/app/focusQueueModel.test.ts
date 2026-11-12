import { describe, expect, it } from 'vitest';
import type { RuntimeSession } from '../types';
import { sanitizeAttentionText } from '../components/attentionText';
import { buildFocusQueue, focusQueueKey } from './focusQueueModel';

function rt(partial: Partial<RuntimeSession>): RuntimeSession {
  return {
    runtime_id: 'rt-1',
    workspace: '/tmp',
    pid: 1,
    host_pid: 1,
    state: 'RUNNING',
    control_level: 'TERMINAL',
    control_endpoint: '',
    started_at: '',
    ...partial,
  };
}

const NOW = Date.parse('2026-11-09T12:00:00Z');

describe('buildFocusQueue', () => {
  it('orders pending work by severity first, then by the oldest episode', () => {
    const queue = buildFocusQueue(
      [
        rt({
          runtime_id: 'err-old',
          project_id: 'p1',
          attention_kind: 'error',
          attention_reason: 'ERROR',
          updated_at: '2026-11-09T10:00:00Z',
        }),
        rt({
          runtime_id: 'ask-new',
          project_id: 'p2',
          state: 'WAITING',
          attention_kind: 'needs_user',
          attention_reason: 'QUESTION',
          prompt_kind: 'yn',
          attention_context: 'Continue?',
          attention_fingerprint: 'fp-new',
          updated_at: '2026-11-09T11:58:00Z',
        }),
        rt({
          runtime_id: 'ask-old',
          project_id: 'p3',
          state: 'APPROVAL',
          attention_kind: 'needs_user',
          attention_reason: 'QUESTION',
          prompt_kind: 'choice',
          attention_context: 'Pick one',
          attention_fingerprint: 'fp-old',
          updated_at: '2026-11-09T09:00:00Z',
        }),
      ],
      NOW,
    );
    expect(queue.map((item) => item.runtimeId)).toEqual(['ask-old', 'ask-new', 'err-old']);
    expect(queue[0].severity).toBe('needs_user');
    expect(queue[0].waitedMs).toBe(NOW - Date.parse('2026-11-09T09:00:00Z'));
  });

  it('deduplicates repeated entries of the same attention episode', () => {
    const episode = {
      runtime_id: 'dup',
      project_id: 'p1',
      state: 'WAITING',
      attention_kind: 'needs_user',
      attention_reason: 'QUESTION',
      prompt_kind: 'yn',
      attention_context: 'Continue?',
      attention_fingerprint: 'fp-dup',
    } as const;
    const queue = buildFocusQueue([rt(episode), rt(episode)], NOW);
    expect(queue).toHaveLength(1);
    expect(queue[0].key).toBe(focusQueueKey(rt(episode), 'needs_user', 'fp-dup'));
    expect(queue[0].key).toBe('fp:fp-dup');
  });

  it('never reopens an episode already answered for its current revision', () => {
    const queue = buildFocusQueue(
      [
        rt({
          runtime_id: 'answered',
          project_id: 'p1',
          state: 'WAITING',
          attention_kind: 'needs_user',
          attention_reason: 'QUESTION',
          prompt_kind: 'yn',
          attention_context: 'Continue?',
          attention_fingerprint: 'fp-answered',
          attention_revision: 3,
          attention_submitted_revision: 3,
        }),
        rt({
          runtime_id: 'stale-submission',
          project_id: 'p2',
          state: 'WAITING',
          attention_kind: 'needs_user',
          attention_reason: 'QUESTION',
          prompt_kind: 'yn',
          attention_context: 'New question',
          attention_fingerprint: 'fp-newer',
          attention_revision: 4,
          attention_submitted_revision: 3,
        }),
      ],
      NOW,
    );
    expect(queue.map((item) => item.runtimeId)).toEqual(['stale-submission']);
  });

  it('keeps only honest needs_user and observed errors in the queue', () => {
    const queue = buildFocusQueue(
      [
        rt({
          runtime_id: 'chrome-wait',
          project_id: 'p1',
          state: 'WAITING',
          attention_reason: 'QUESTION',
          attention_kind: 'needs_user',
          prompt_kind: 'none',
          attention_context: '',
        }),
        rt({
          runtime_id: 'working',
          project_id: 'p1',
          attention_kind: 'working',
          attention_reason: 'WORKING',
        }),
        rt({
          runtime_id: 'failed',
          project_id: 'p1',
          state: 'FAILED',
          attention_kind: 'error',
          attention_reason: 'ERROR',
        }),
        rt({
          runtime_id: 'observed-error',
          project_id: 'p1',
          attention_kind: 'error',
          attention_reason: 'ERROR',
          attention_context: 'Provider exited',
        }),
      ],
      NOW,
    );
    expect(queue.map((item) => item.runtimeId)).toEqual(['observed-error']);
    expect(queue[0].severity).toBe('error');
    // Error identities are runtime-scoped, not fingerprint-scoped.
    expect(queue[0].key).toBe('rt:observed-error|error');
    expect(queue[0].fingerprint).toBeUndefined();
  });

  it('sanitizes context and keeps rebuilds deterministic for identical input', () => {
    const runtimes = [
      rt({
        runtime_id: 'ask',
        project_id: 'p1',
        project_name: 'Alpha',
        state: 'WAITING',
        attention_kind: 'needs_user',
        attention_reason: 'QUESTION',
        prompt_kind: 'free_text',
        attention_context: 'password=hunter2 Continue?',
        attention_fingerprint: 'fp-sanitize',
        updated_at: '2026-11-09T11:00:00Z',
      }),
    ];
    const first = buildFocusQueue(runtimes, NOW);
    const second = buildFocusQueue(runtimes, NOW);
    expect(first).toEqual(second);
    // Same sanitizer contract as the radar: transport noise is collapsed;
    // textual secrets are not invented away, matching attentionRadarModel.
    expect(first[0].context).toBe(sanitizeAttentionText('password=hunter2 Continue?', ''));
    expect(first[0].projectName).toBe('Alpha');
  });
});
