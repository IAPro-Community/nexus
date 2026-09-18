export const DEFAULT_TERMINAL_FONT_SIZE = 13;
export const MIN_TERMINAL_FONT_SIZE = 9;
export const MAX_TERMINAL_FONT_SIZE = 24;
export const DEFAULT_TERMINAL_SCROLLBACK = 5000;

const STORAGE_KEY = 'nx_terminal_font_size';
const EVENT_NAME = 'nexus:terminal-font-size';

export function clampFontSize(size: number): number {
  if (Number.isNaN(size) || typeof size !== 'number') {
    return DEFAULT_TERMINAL_FONT_SIZE;
  }
  return Math.min(MAX_TERMINAL_FONT_SIZE, Math.max(MIN_TERMINAL_FONT_SIZE, Math.round(size)));
}

export function getStoredTerminalFontSize(): number {
  if (typeof window === 'undefined') return DEFAULT_TERMINAL_FONT_SIZE;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT_TERMINAL_FONT_SIZE;
    const parsed = parseInt(raw, 10);
    return clampFontSize(parsed);
  } catch {
    return DEFAULT_TERMINAL_FONT_SIZE;
  }
}

export function storeTerminalFontSize(size: number): number {
  const clamped = clampFontSize(size);
  if (typeof window !== 'undefined') {
    try {
      localStorage.setItem(STORAGE_KEY, String(clamped));
      window.dispatchEvent(new CustomEvent(EVENT_NAME, { detail: clamped }));
    } catch {
      // ignore storage errors
    }
  }
  return clamped;
}

export function adjustTerminalFontSize(delta: number, current?: number): number {
  const base = current !== undefined ? current : getStoredTerminalFontSize();
  return storeTerminalFontSize(base + delta);
}

export function resetTerminalFontSize(): number {
  return storeTerminalFontSize(DEFAULT_TERMINAL_FONT_SIZE);
}

export function shouldAutoScrollToBottom(viewportY: number, baseY: number): boolean {
  // If viewportY is at baseY or within 1 line of the bottom, the user was tracking live output
  return viewportY >= baseY - 1;
}

export function subscribeTerminalFontSize(callback: (size: number) => void): () => void {
  if (typeof window === 'undefined') return () => {};
  const handler = (event: Event) => {
    const custom = event as CustomEvent<number>;
    if (typeof custom.detail === 'number') {
      callback(custom.detail);
    }
  };
  window.addEventListener(EVENT_NAME, handler);
  return () => window.removeEventListener(EVENT_NAME, handler);
}
