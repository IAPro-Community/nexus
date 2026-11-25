import { describe, expect, it } from 'vitest';
import { formatAttentionPushBody, shouldSendBrowserAttentionPush } from './attentionPushCopy';

describe('attention push copy', () => {
  it('frames yn waits with agent and action', () => {
    expect(
      formatAttentionPushBody({
        reason: 'QUESTION',
        promptKind: 'yn',
        agentName: 'Codex · revise o projeto',
        context: 'Continue anyway? [y/N]:',
      }),
    ).toBe(
      'Codex · revise o projeto pede confirmação (Sim/Não): Continue anyway? [y/N]: — Abra o Nexus e responda no terminal.',
    );
  });

  it('skips browser push while the Nexus tab is visible', () => {
    expect(
      shouldSendBrowserAttentionPush({
        permission: 'granted',
        documentHidden: false,
        reason: 'QUESTION',
        promptKind: 'yn',
        context: 'Continue anyway? [y/N]:',
      }),
    ).toBe(false);
  });

  it('allows browser push when the tab is hidden', () => {
    expect(
      shouldSendBrowserAttentionPush({
        permission: 'granted',
        documentHidden: true,
        reason: 'QUESTION',
        promptKind: 'yn',
        context: 'Continue anyway? [y/N]:',
      }),
    ).toBe(true);
  });

  it('describes authentication and permission requests as urgent attention', () => {
    expect(
      formatAttentionPushBody({
        reason: 'AUTH',
        context: 'Sign in to continue',
      }),
    ).toContain('aguarda autenticação ou permissão');
    expect(
      shouldSendBrowserAttentionPush({
        permission: 'granted',
        documentHidden: true,
        reason: 'AUTH',
        context: 'Sign in to continue',
      }),
    ).toBe(true);
  });
});
