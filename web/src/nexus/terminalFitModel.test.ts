import { describe, expect, it } from 'vitest';
import { canFitTerminal } from './terminalFitModel';

const ready = () => ({
  disposed: false,
  sessionReady: true,
  terminalConnected: true,
  containerConnected: true,
  width: 640,
  height: 320,
});

describe('canFitTerminal', () => {
  it('allows fitting only after the terminal session and both DOM nodes are ready', () => {
    expect(canFitTerminal(ready())).toBe(true);
    expect(canFitTerminal({ ...ready(), sessionReady: false })).toBe(false);
    expect(canFitTerminal({ ...ready(), terminalConnected: false })).toBe(false);
    expect(canFitTerminal({ ...ready(), containerConnected: false })).toBe(false);
  });

  it('rejects disposed or zero-sized terminals', () => {
    expect(canFitTerminal({ ...ready(), disposed: true })).toBe(false);
    expect(canFitTerminal({ ...ready(), width: 0 })).toBe(false);
    expect(canFitTerminal({ ...ready(), height: 0 })).toBe(false);
  });
});
