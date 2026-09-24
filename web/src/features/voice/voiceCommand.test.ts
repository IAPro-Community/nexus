import { describe, expect, it } from 'vitest';
import { interpretNexusVoiceCommand } from './voiceCommand';

describe('Nexus deterministic voice commands', () => {
  it('maps Portuguese navigation without an LLM', () => {
    expect(interpretNexusVoiceCommand('Abrir agentes')).toEqual({
      kind: 'command',
      commandId: 'agents',
    });
    expect(interpretNexusVoiceCommand('Configurações de voz')).toEqual({ kind: 'voice-settings' });
  });

  it('maps command palette and leaves arbitrary speech unknown', () => {
    expect(interpretNexusVoiceCommand('mostrar comandos')).toEqual({ kind: 'palette' });
    expect(interpretNexusVoiceCommand('apague todo o projeto')).toEqual({ kind: 'unknown' });
  });
});
