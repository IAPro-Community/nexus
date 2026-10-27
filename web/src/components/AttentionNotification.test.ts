import { describe, expect, it } from 'vitest';
import { sanitizeAttentionText } from './attentionText';
import { attentionCardActions, shouldRenderAttentionCard } from './AttentionNotificationCard';
import type { RuntimeSession } from '../types';

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

describe('Attention Notification Card sanitization & logic', () => {
  it('correctly handles raw context with UTF-8 noise', () => {
    const raw = ' Deseja executar a migração? [y/N]';
    expect(sanitizeAttentionText(raw, 'fallback')).toBe('Deseja executar a migração? [y/N]');
  });

  it('provides safe fallback when context is empty or corrupted', () => {
    expect(sanitizeAttentionText(null, 'O agente requer atenção.')).toBe(
      'O agente requer atenção.',
    );
    expect(sanitizeAttentionText('', 'O agente requer atenção.')).toBe('O agente requer atenção.');
  });

  it('does not render a question card without usable context', () => {
    expect(
      shouldRenderAttentionCard(
        rt({
          attention_reason: 'QUESTION',
          attention_kind: 'needs_user',
          prompt_kind: 'none',
          attention_context: '',
        }),
      ),
    ).toBe(false);
  });

  it('renders honest needs_user with context', () => {
    expect(
      shouldRenderAttentionCard(
        rt({
          attention_reason: 'QUESTION',
          attention_kind: 'needs_user',
          prompt_kind: 'yn',
          attention_context: 'Deseja continuar? [y/N]',
        }),
      ),
    ).toBe(true);
  });

  it('does not offer remote input for OSC status-only approval or authentication', () => {
    for (const reason of ['APPROVAL', 'AUTH'] as const) {
      const actions = attentionCardActions(
        rt({
          attention_reason: reason,
          attention_kind: 'needs_user',
          prompt_kind: 'native_status',
          attention_context: 'Agent requires attention in the terminal',
        }),
      );
      expect(actions.showOpenTerminal).toBe(true);
      expect(actions.showTextInput).toBe(false);
      expect(actions.showYesNo).toBe(false);
    }
  });

  it('shows Yes/No only for prompt_kind yn', () => {
    const yn = attentionCardActions(
      rt({
        attention_reason: 'QUESTION',
        attention_kind: 'needs_user',
        prompt_kind: 'yn',
        attention_context: 'Deseja continuar? [y/N]',
      }),
    );
    expect(yn.showYesNo).toBe(true);

    const free = attentionCardActions(
      rt({
        attention_reason: 'QUESTION',
        attention_kind: 'needs_user',
        prompt_kind: 'free_text',
        attention_context: 'Por favor, revise o plano.',
      }),
    );
    expect(free.showYesNo).toBe(false);
    expect(free.showTextInput).toBe(true);
    expect(free.showOpenTerminal).toBe(true);

    const choice = attentionCardActions(
      rt({
        attention_reason: 'QUESTION',
        attention_kind: 'needs_user',
        prompt_kind: 'choice',
        attention_context: 'Choice [1-3]:',
      }),
    );
    expect(choice.showYesNo).toBe(false);
    expect(choice.showOpenTerminal).toBe(true);
  });

  it('keeps identical questions in separate runtimes distinct by runtime identity', () => {
    const rtA = rt({
      runtime_id: 'rt-alpha',
      project_id: 'proj-a',
      attention_reason: 'QUESTION',
      attention_kind: 'needs_user',
      prompt_kind: 'yn',
      attention_context: 'Allow action? [y/N]',
    });
    const rtB = rt({
      runtime_id: 'rt-beta',
      project_id: 'proj-b',
      attention_reason: 'QUESTION',
      attention_kind: 'needs_user',
      prompt_kind: 'yn',
      attention_context: 'Allow action? [y/N]',
    });

    expect(shouldRenderAttentionCard(rtA)).toBe(true);
    expect(shouldRenderAttentionCard(rtB)).toBe(true);
    expect(rtA.runtime_id).not.toBe(rtB.runtime_id);
    expect(rtA.project_id).not.toBe(rtB.project_id);
  });
});
