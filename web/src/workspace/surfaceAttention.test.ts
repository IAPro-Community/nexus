import { describe, expect, it } from 'vitest';
import {
  shouldMarkUnread,
  statusSuffixFromRuntime,
  surfaceTitleFromAgent,
} from './surfaceAttention';
import type { RuntimeSession } from '../types';

const runtime = (patch: Partial<RuntimeSession>): RuntimeSession =>
  ({
    runtime_id: 'rt-1',
    state: 'RUNNING',
    ...patch,
  }) as RuntimeSession;

describe('surfaceAttention', () => {
  it('keeps the agent name and appends a short status suffix', () => {
    const titled = surfaceTitleFromAgent(
      'Backend',
      runtime({
        attention_reason: 'QUESTION',
        attention_context: 'Continue?',
        prompt_kind: 'yn',
        attention_kind: 'needs_user',
      }),
    );
    expect(titled.title).toBe('Backend · pergunta');
    expect(titled.statusSuffix).toBe('pergunta');
    expect(titled.hasAttention).toBe(true);
  });

  it('does not mark an uncorrelated or stale approval as actionable', () => {
    const noPrompt = runtime({
      state: 'WAITING',
      attention_reason: 'QUESTION',
      attention_kind: 'needs_user',
      prompt_kind: 'none',
      attention_context: '',
    });
    const explicitIdle = runtime({
      state: 'RUNNING',
      attention_reason: 'APPROVAL',
      attention_kind: 'idle',
      prompt_kind: 'yn',
      attention_context: 'Previous approval',
    });
    const stopped = runtime({
      state: 'STOPPED',
      attention_reason: 'QUESTION',
      attention_kind: 'needs_user',
      prompt_kind: 'yn',
      attention_context: 'Previous question',
    });
    for (const item of [noPrompt, explicitIdle, stopped]) {
      const title = surfaceTitleFromAgent('Backend', item);
      expect(title.attentionKind).not.toBe('needs_user');
      expect(title.statusSuffix).not.toBe('pergunta');
      expect(title.hasAttention).toBe(false);
    }
  });

  it('preserves a legacy question with context but without attention_kind', () => {
    const result = surfaceTitleFromAgent(
      'Backend',
      runtime({
        state: 'WAITING',
        attention_reason: 'QUESTION',
        attention_kind: undefined,
        attention_context: 'Continue?',
      }),
    );
    expect(result.attentionKind).toBe('needs_user');
    expect(result.hasAttention).toBe(true);
  });

  it('does not replace the name with dynamic_title', () => {
    const titled = surfaceTitleFromAgent(
      'Frontend',
      runtime({
        attention_reason: 'QUESTION',
        attention_context: 'Apply changes?',
        prompt_kind: 'yn',
        attention_kind: 'needs_user',
        dynamic_title: 'ai-chat · Apply changes?',
      }),
    );
    expect(titled.title.startsWith('Frontend')).toBe(true);
    expect(titled.title).not.toContain('ai-chat');
    expect(titled.dynamicTitle).toContain('ai-chat');
  });

  it('prefers customTitle as stable identity root', () => {
    const titled = surfaceTitleFromAgent('Agent', runtime({ state: 'STOPPED' }), 'Shell A');
    expect(titled.title.startsWith('Shell A')).toBe(true);
    expect(titled.title).not.toContain('Agent');
  });

  it('maps completion and error suffixes', () => {
    expect(statusSuffixFromRuntime(runtime({ attention_reason: 'TASK_COMPLETED' }))).toBe(
      'concluído',
    );
    expect(statusSuffixFromRuntime(runtime({ attention_reason: 'ERROR' }))).toBe('erro');
  });

  it('does not treat bare RUNNING as trabalhando', () => {
    expect(statusSuffixFromRuntime(runtime({ state: 'RUNNING', attention_kind: 'idle' }))).toBe('');
    expect(
      statusSuffixFromRuntime(
        runtime({ state: 'RUNNING', attention_kind: 'working', attention_reason: 'WORKING' }),
      ),
    ).toBe('trabalhando');
  });

  it('marks unread only on fingerprint transition when unfocused', () => {
    expect(
      shouldMarkUnread({
        previousFingerprint: 'old',
        nextFingerprint: 'new',
        hasAttention: true,
        surfaceFocused: false,
        attentionKind: 'needs_user',
      }),
    ).toBe(true);
    expect(
      shouldMarkUnread({
        previousFingerprint: 'same',
        nextFingerprint: 'same',
        hasAttention: true,
        surfaceFocused: false,
        attentionKind: 'needs_user',
      }),
    ).toBe(false);
    expect(
      shouldMarkUnread({
        previousFingerprint: 'old',
        nextFingerprint: 'new',
        hasAttention: true,
        surfaceFocused: true,
        attentionKind: 'needs_user',
      }),
    ).toBe(false);
    expect(
      shouldMarkUnread({
        previousFingerprint: '',
        nextFingerprint: 'work',
        hasAttention: true,
        surfaceFocused: false,
        attentionKind: 'working',
      }),
    ).toBe(false);
  });
});
