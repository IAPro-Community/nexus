import { describe, it, expect, beforeEach } from 'vitest';
import {
  DEFAULT_TERMINAL_FONT_SIZE,
  DEFAULT_TERMINAL_SCROLLBACK,
  adjustTerminalFontSize,
  getStoredTerminalFontSize,
  resetTerminalFontSize,
  shouldAutoScrollToBottom,
  storeTerminalFontSize,
} from './terminalSettings';

const store = new Map<string, string>();
const mockLocalStorage = {
  getItem: (key: string) => store.get(key) ?? null,
  setItem: (key: string, val: string) => {
    store.set(key, String(val));
  },
  removeItem: (key: string) => {
    store.delete(key);
  },
  clear: () => {
    store.clear();
  },
};

(globalThis as any).window = globalThis;
(globalThis as any).localStorage = mockLocalStorage;

describe('terminalUsability integration logic', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  function simulateTerminalKey(event: {
    ctrlKey?: boolean;
    metaKey?: boolean;
    key?: string;
    code?: string;
    type?: string;
  }): boolean {
    if (event.ctrlKey || event.metaKey) {
      if (
        event.key === '=' ||
        event.key === '+' ||
        event.code === 'Equal' ||
        event.code === 'NumpadAdd'
      ) {
        if (event.type === 'keydown') {
          const next = adjustTerminalFontSize(1);
          storeTerminalFontSize(next);
        }
        return false;
      }
      if (
        event.key === '-' ||
        event.key === '_' ||
        event.code === 'Minus' ||
        event.code === 'NumpadSubtract'
      ) {
        if (event.type === 'keydown') {
          const next = adjustTerminalFontSize(-1);
          storeTerminalFontSize(next);
        }
        return false;
      }
      if (event.key === '0' || event.code === 'Digit0' || event.code === 'Numpad0') {
        if (event.type === 'keydown') {
          const next = resetTerminalFontSize();
          storeTerminalFontSize(next);
        }
        return false;
      }
    }
    return true;
  }

  it('handles Ctrl+= and Ctrl++ to zoom in without leaking to PTY', () => {
    expect(getStoredTerminalFontSize()).toBe(DEFAULT_TERMINAL_FONT_SIZE); // 13

    const handled = simulateTerminalKey({
      ctrlKey: true,
      key: '=',
      code: 'Equal',
      type: 'keydown',
    });

    expect(handled).toBe(false); // intercepted
    expect(getStoredTerminalFontSize()).toBe(14);

    const handledPlus = simulateTerminalKey({
      ctrlKey: true,
      key: '+',
      code: 'Equal',
      type: 'keydown',
    });

    expect(handledPlus).toBe(false);
    expect(getStoredTerminalFontSize()).toBe(15);
  });

  it('handles Ctrl+- to zoom out without leaking to PTY', () => {
    storeTerminalFontSize(13);

    const handled = simulateTerminalKey({
      ctrlKey: true,
      key: '-',
      code: 'Minus',
      type: 'keydown',
    });

    expect(handled).toBe(false);
    expect(getStoredTerminalFontSize()).toBe(12);
  });

  it('handles Ctrl+0 to reset zoom to default 13px', () => {
    storeTerminalFontSize(18);
    expect(getStoredTerminalFontSize()).toBe(18);

    const handled = simulateTerminalKey({
      ctrlKey: true,
      key: '0',
      code: 'Digit0',
      type: 'keydown',
    });

    expect(handled).toBe(false);
    expect(getStoredTerminalFontSize()).toBe(DEFAULT_TERMINAL_FONT_SIZE);
  });

  it('allows essential shell shortcuts (Ctrl+C, Ctrl+Z, Ctrl+L) to pass to PTY', () => {
    const ctrlC = simulateTerminalKey({
      ctrlKey: true,
      key: 'c',
      code: 'KeyC',
      type: 'keydown',
    });
    expect(ctrlC).toBe(true); // not intercepted! Passes to shell for SIGINT

    const ctrlZ = simulateTerminalKey({
      ctrlKey: true,
      key: 'z',
      code: 'KeyZ',
      type: 'keydown',
    });
    expect(ctrlZ).toBe(true); // Passes to shell for SIGTSTP

    const ctrlL = simulateTerminalKey({
      ctrlKey: true,
      key: 'l',
      code: 'KeyL',
      type: 'keydown',
    });
    expect(ctrlL).toBe(true); // Passes to shell for clear screen
  });

  it('verifies smart scrolling behavior for incoming output', () => {
    // Case 1: user is tracking bottom
    const atBottom = shouldAutoScrollToBottom(100, 100);
    expect(atBottom).toBe(true);

    // Case 2: user scrolled up to inspect previous output
    const scrolledUp = shouldAutoScrollToBottom(40, 100);
    expect(scrolledUp).toBe(false);
    // When scrolledUp is false, incoming term.write MUST NOT snap viewport to bottom,
    // preserving user reading position and showing floating button.
  });

  it('verifies memory safety with default 5000 lines scrollback', () => {
    expect(DEFAULT_TERMINAL_SCROLLBACK).toBe(5000);
    expect(DEFAULT_TERMINAL_SCROLLBACK).toBeGreaterThanOrEqual(1000);
    expect(DEFAULT_TERMINAL_SCROLLBACK).toBeLessThanOrEqual(10000);
  });
});
