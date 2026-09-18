import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const terminalSource = readFileSync(
  new URL('../../nexus/AgentTerminal.tsx', import.meta.url),
  'utf8',
);

describe('Terminal Skills and Alias Mediation', () => {
  it('exposes one-shot ask composer and skills recommendation in terminal', () => {
    expect(terminalSource).toContain("t('terminal.sendInstructionTitle')");
    expect(terminalSource).toContain('loadSkills');
    expect(terminalSource).toContain('getMaestroStatus');
    expect(terminalSource).toContain('handleSendPrompt');
    expect(terminalSource).toContain('askAgent');
  });

  it('restricts selected skills to maximum 3 for next prompt', () => {
    expect(terminalSource).toContain('prev.length < 3');
  });

  it('displays AGENT badge, supervised nexus alias and safe autonomy toggles', () => {
    expect(terminalSource).toContain('AGENT');
    expect(terminalSource).toContain('nexus {provider}');
    expect(terminalSource).toContain('Safe');
    expect(terminalSource).not.toContain("'Plan'");
    expect(terminalSource).toContain('YOLO');
    expect(terminalSource).not.toContain('Assumir digitação');
    expect(terminalSource).not.toContain('Liberar digitação');
  });
});
