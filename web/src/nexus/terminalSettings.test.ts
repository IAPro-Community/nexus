import { beforeEach, describe, expect, it } from 'vitest';
import {
  clampFontSize,
  DEFAULT_TERMINAL_FONT_SIZE,
  MAX_TERMINAL_FONT_SIZE,
  MIN_TERMINAL_FONT_SIZE,
  getStoredTerminalFontSize,
  getTerminalTheme,
  storeTerminalFontSize,
  adjustTerminalFontSize,
  resetTerminalFontSize,
  shouldAutoScrollToBottom,
  DEFAULT_TERMINAL_SCROLLBACK,
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

describe('terminalSettings', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('clamps font sizes within safe bounds', () => {
    expect(clampFontSize(5)).toBe(MIN_TERMINAL_FONT_SIZE);
    expect(clampFontSize(50)).toBe(MAX_TERMINAL_FONT_SIZE);
    expect(clampFontSize(14)).toBe(14);
    expect(clampFontSize(NaN)).toBe(DEFAULT_TERMINAL_FONT_SIZE);
  });

  it('reads and writes persistent font size', () => {
    expect(getStoredTerminalFontSize()).toBe(DEFAULT_TERMINAL_FONT_SIZE);
    storeTerminalFontSize(16);
    expect(getStoredTerminalFontSize()).toBe(16);
  });

  it('adjusts font size incrementally', () => {
    expect(adjustTerminalFontSize(1, 13)).toBe(14);
    expect(adjustTerminalFontSize(-2, 13)).toBe(11);
    expect(adjustTerminalFontSize(100, 13)).toBe(MAX_TERMINAL_FONT_SIZE);
    expect(adjustTerminalFontSize(-100, 13)).toBe(MIN_TERMINAL_FONT_SIZE);
  });

  it('resets font size to default', () => {
    storeTerminalFontSize(20);
    expect(resetTerminalFontSize()).toBe(DEFAULT_TERMINAL_FONT_SIZE);
    expect(getStoredTerminalFontSize()).toBe(DEFAULT_TERMINAL_FONT_SIZE);
  });

  it('determines if terminal should auto-scroll to bottom based on buffer position', () => {
    // When viewport is at base or within 1 line, user is tracking the live end
    expect(shouldAutoScrollToBottom(100, 100)).toBe(true);
    expect(shouldAutoScrollToBottom(99, 100)).toBe(true);

    // When viewport is scrolled up significantly, user is inspecting history
    expect(shouldAutoScrollToBottom(50, 100)).toBe(false);
    expect(shouldAutoScrollToBottom(0, 100)).toBe(false);
  });

  it('defines balanced scrollback bounds to avoid memory exhaustion', () => {
    expect(DEFAULT_TERMINAL_SCROLLBACK).toBeGreaterThanOrEqual(2000);
    expect(DEFAULT_TERMINAL_SCROLLBACK).toBeLessThanOrEqual(10000);
  });

  it('resolves xterm colors from the semantic terminal CSS tokens', () => {
    const originalDocument = Object.getOwnPropertyDescriptor(globalThis, 'document');
    const originalGetComputedStyle = Object.getOwnPropertyDescriptor(globalThis, 'getComputedStyle');
    const tokens: Record<string, string> = {
      '--nx-terminal-bg': '#010203',
      '--nx-terminal-text': '#f1f2f3',
      '--nx-terminal-cursor': '#abcdef',
      '--nx-terminal-selection': 'rgba(1, 2, 3, 0.4)',
    };

    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: { documentElement: {} },
    });
    Object.defineProperty(globalThis, 'getComputedStyle', {
      configurable: true,
      value: () => ({
        getPropertyValue: (name: string) => tokens[name] || '',
      }),
    });

    try {
      expect(getTerminalTheme()).toEqual({
        background: '#010203',
        foreground: '#f1f2f3',
        cursor: '#abcdef',
        selectionBackground: 'rgba(1, 2, 3, 0.4)',
      });
    } finally {
      if (originalDocument) {
        Object.defineProperty(globalThis, 'document', originalDocument);
      } else {
        Reflect.deleteProperty(globalThis, 'document');
      }
      if (originalGetComputedStyle) {
        Object.defineProperty(globalThis, 'getComputedStyle', originalGetComputedStyle);
      } else {
        Reflect.deleteProperty(globalThis, 'getComputedStyle');
      }
    }
  });
});
