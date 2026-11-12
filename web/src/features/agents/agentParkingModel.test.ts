import { describe, expect, it } from 'vitest';
import { en, es, ptBR } from '../../i18n/resources';
import {
  agentTerminalSurfaceKey,
  isAgentSurfaceOpen,
  isParkedWorking,
  parkingHintKey,
  parkingOptions,
  parkingStatusKey,
  type ParkingActionKind,
} from './agentParkingModel';

describe('agentTerminalSurfaceKey', () => {
  it('matches the workspace logical key convention', () => {
    expect(agentTerminalSurfaceKey('a1')).toBe('agent:a1:terminal');
  });
});

describe('isAgentSurfaceOpen', () => {
  it('detects open and hidden terminal surfaces', () => {
    const keys = [agentTerminalSurfaceKey('a1'), 'work'];
    expect(isAgentSurfaceOpen(keys, 'a1')).toBe(true);
    expect(isAgentSurfaceOpen(keys, 'a2')).toBe(false);
    expect(isAgentSurfaceOpen([], 'a1')).toBe(false);
  });
});

describe('parkingOptions', () => {
  it('never offers suspend while the backend does not support it', () => {
    expect(parkingOptions(false).map((action) => action.kind)).toEqual(['hide']);
    expect(parkingOptions(false).some((action) => action.kind === 'suspend')).toBe(false);
  });

  it('only includes suspend when explicitly supported', () => {
    expect(parkingOptions(true).map((action) => action.kind)).toEqual(['hide', 'suspend']);
  });
});

describe('parkingStatusKey', () => {
  it('returns nothing while the surface is open', () => {
    expect(parkingStatusKey(true, true)).toBeNull();
    expect(parkingStatusKey(true, false)).toBeNull();
  });

  it('says the runtime keeps working when only the surface is hidden', () => {
    expect(parkingStatusKey(false, true)).toBe('agents.parking.hiddenWorking');
  });

  it('uses the idle copy when nothing is running', () => {
    expect(parkingStatusKey(false, false)).toBe('agents.parking.hiddenIdle');
  });
});

describe('isParkedWorking', () => {
  it('marks exactly the hidden-but-running case', () => {
    expect(isParkedWorking(false, true)).toBe(true);
    expect(isParkedWorking(true, true)).toBe(false);
    expect(isParkedWorking(false, false)).toBe(false);
  });
});

describe('parkingHintKey', () => {
  it('maps each action to its hint', () => {
    expect(parkingHintKey('hide')).toBe('agents.parking.hideHint');
    expect(parkingHintKey('suspend')).toBe('agents.parking.suspendHint');
  });
});

// Regression guard for acceptance criterion 2: the UI must never claim the
// process is paused when the surface was merely hidden. The hide copy must
// affirmatively say the runtime keeps running, in every locale.
describe('parking i18n honesty', () => {
  const locales: Array<[string, { agents: { parking: Record<string, string> } }]> = [
    ['en', en],
    ['ptBR', ptBR],
    ['es', es],
  ];

  it.each(locales)('%s hide copy denies the paused claim', (_name, catalog) => {
    const copy = catalog.agents.parking.hideHint as string;
    expect(/keeps running|continua em execução|sigue en ejecución/i.test(copy)).toBe(true);
    expect(/not paus|não paus|no paus/i.test(copy)).toBe(true);
  });

  it.each(locales)('%s hiddenWorking copy is not a paused claim', (_name, catalog) => {
    const copy = catalog.agents.parking.hiddenWorking as string;
    expect(/RUNNING|em execução|en ejecución/i.test(copy)).toBe(true);
    expect(/not paused|não pausad|no pausad/i.test(copy)).toBe(true);
  });

  it('suspend copy only exists alongside backend support', () => {
    // The suspend action is unreachable in the UI today; the key exists for
    // the future backend capability, but parkingOptions(false) hides it.
    expect(parkingOptions(false).length).toBe(1);
    expect(parkingOptions(true).length).toBe(2);
    expect(parkingHintKey('suspend' as ParkingActionKind)).toContain('suspend');
  });
});
