import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  NEXUS_TERMINAL_VOICE_EVENT,
  dispatchTerminalVoiceDictation,
  interpretNexusVoiceCommand,
  resolveNexusVoiceCommand,
} from './voiceCommand';

describe('Nexus deterministic voice commands', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });
  it('maps Portuguese navigation without an LLM', () => {
    expect(interpretNexusVoiceCommand('Abrir agentes')).toEqual({
      kind: 'command',
      commandId: 'agents',
    });
    expect(interpretNexusVoiceCommand('Configurações de voz')).toEqual({ kind: 'voice-settings' });
    expect(interpretNexusVoiceCommand('Criar agente')).toEqual({
      kind: 'command',
      commandId: 'new-agent',
    });
  });

  it('maps command palette and leaves destructive arbitrary speech unknown', () => {
    expect(interpretNexusVoiceCommand('mostrar comandos')).toEqual({ kind: 'palette' });
    expect(interpretNexusVoiceCommand('apague todo o projeto')).toEqual({ kind: 'unknown' });
  });

  it('extracts explicit terminal dictation without adding Enter', () => {
    expect(interpretNexusVoiceCommand('Escreva no terminal git status')).toEqual({
      kind: 'terminal-dictation',
      text: 'git status',
    });
    expect(interpretNexusVoiceCommand('terminal: digite npm test')).toEqual({
      kind: 'terminal-dictation',
      text: 'npm test',
    });
  });

  it('reuses only the voice-safe Nexus command registry', () => {
    const commands = [
      { id: 'preset-two-columns', label: 'Layout: Two Columns', keywords: ['duas colunas'] },
      { id: 'terminal-agent-1', label: 'Abrir Atlas terminal', keywords: ['atlas'] },
      { id: 'delete-project', label: 'Excluir projeto', keywords: ['excluir'] },
    ];
    expect(resolveNexusVoiceCommand('duas colunas', commands)).toEqual({
      kind: 'command',
      commandId: 'preset-two-columns',
    });
    expect(resolveNexusVoiceCommand('abrir Atlas terminal', commands)).toEqual({
      kind: 'command',
      commandId: 'terminal-agent-1',
    });
    expect(resolveNexusVoiceCommand('excluir projeto', commands)).toEqual({ kind: 'unknown' });
  });

  it('dispatches terminal dictation to the active terminal bridge', () => {
    class TestCustomEvent<T> extends Event {
      detail: T;

      constructor(type: string, init: { detail: T }) {
        super(type);
        this.detail = init.detail;
      }
    }

    vi.stubGlobal('window', new EventTarget());
    vi.stubGlobal('CustomEvent', TestCustomEvent);

    const listener = vi.fn((event: Event) => {
      const custom = event as CustomEvent<{ handled: boolean }>;
      custom.detail.handled = true;
    });
    window.addEventListener(NEXUS_TERMINAL_VOICE_EVENT, listener);
    const result = dispatchTerminalVoiceDictation('git status');
    window.removeEventListener(NEXUS_TERMINAL_VOICE_EVENT, listener);

    expect(listener).toHaveBeenCalledOnce();
    expect(result).toMatchObject({ text: 'git status', handled: true });
  });
});
