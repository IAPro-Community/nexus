import { describe, expect, it } from 'vitest';
import type { RuntimeSession } from '../types';
import { buildDocumentTitle, distinctAttentionFingerprints } from './documentTitle';

function rt(partial: Partial<RuntimeSession>): RuntimeSession {
  return {
    runtime_id: 'rt-1',
    workspace: '/tmp',
    pid: 1,
    host_pid: 1,
    state: 'WAITING',
    control_level: 'TERMINAL',
    control_endpoint: '',
    started_at: '',
    ...partial,
  };
}

describe('buildDocumentTitle', () => {
  it('uses the plain project title without emoji when idle', () => {
    expect(buildDocumentTitle('proxy-nginx', [])).toBe('Nexus · proxy-nginx');
  });

  it('shows a short question for a single honest wait', () => {
    const runtimes = [
      rt({
        attention_reason: 'QUESTION',
        attention_kind: 'needs_user',
        prompt_kind: 'yn',
        attention_context: 'Deseja executar a migração? [y/N]',
        attention_fingerprint: 'fp-a',
      }),
    ];
    expect(buildDocumentTitle('proxy-nginx', runtimes)).toBe(
      'Nexus · proxy-nginx · Deseja executar a migração? [y/N]',
    );
  });

  it('counts identical questions from separate runtimes as separate interventions', () => {
    const runtimes = [
      rt({
        runtime_id: 'rt-1',
        attention_reason: 'QUESTION',
        attention_kind: 'needs_user',
        prompt_kind: 'yn',
        attention_context: 'Continue?',
        attention_fingerprint: 'fp-rt1',
      }),
      rt({
        runtime_id: 'rt-2',
        attention_reason: 'QUESTION',
        attention_kind: 'needs_user',
        prompt_kind: 'yn',
        attention_context: 'Continue?',
        attention_fingerprint: 'fp-rt2',
      }),
      rt({
        runtime_id: 'rt-3',
        attention_reason: 'QUESTION',
        attention_kind: 'needs_user',
        prompt_kind: 'yn',
        attention_context: 'Continue?',
        attention_fingerprint: 'fp-rt3',
      }),
    ];
    expect(distinctAttentionFingerprints(runtimes)).toHaveLength(3);
    expect(buildDocumentTitle('proxy-nginx', runtimes)).toBe('Nexus · proxy-nginx · 3 esperando input');
    expect(buildDocumentTitle('proxy-nginx', runtimes)).not.toContain('❓');
  });

  it('counts distinct fingerprints when several waits differ', () => {
    const runtimes = [
      rt({
        runtime_id: 'rt-1',
        attention_kind: 'needs_user',
        prompt_kind: 'yn',
        attention_context: 'A?',
        attention_fingerprint: 'a',
        attention_reason: 'QUESTION',
      }),
      rt({
        runtime_id: 'rt-2',
        attention_kind: 'needs_user',
        prompt_kind: 'free_text',
        attention_context: 'B?',
        attention_fingerprint: 'b',
        attention_reason: 'QUESTION',
      }),
    ];
    expect(buildDocumentTitle('demo', runtimes)).toBe('Nexus · demo · 2 esperando input');
  });

  it('ignores waits without usable context', () => {
    const runtimes = [
      rt({
        attention_reason: 'QUESTION',
        attention_kind: 'needs_user',
        prompt_kind: 'none',
        attention_context: '',
      }),
    ];
    expect(buildDocumentTitle('demo', runtimes)).toBe('Nexus · demo');
  });

  it('ignores waits whose context is only UTF-8 replacement noise', () => {
    const runtimes = [
      rt({
        attention_reason: 'QUESTION',
        attention_kind: 'needs_user',
        prompt_kind: 'free_text',
        attention_context: '\uFFFD\uFFFD\uFFFD\uFFFD',
      }),
    ];
    expect(buildDocumentTitle('demo', runtimes)).toBe('Nexus · demo');
  });
});
