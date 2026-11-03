import { describe, expect, it, vi } from 'vitest';
import {
  refreshTerminalAfterActivation,
  resetTerminalForReplay,
  streamTerminalOutput,
  type TerminalOutputSink,
} from './terminalOutput';

function createSink() {
  const writes: Array<{ data: string; callback?: () => void }> = [];
  const scrollToBottom = vi.fn();
  const refresh = vi.fn();
  const terminal: TerminalOutputSink = {
    buffer: { active: { viewportY: 5, baseY: 5 } },
    rows: 24,
    write: (data, callback) => { writes.push({ data, callback }); },
    scrollToBottom,
    refresh,
  };
  return { terminal, writes, scrollToBottom, refresh };
}

describe('terminal output', () => {
  it('streams all output in order even while a panel is hidden', () => {
    const { terminal, writes, scrollToBottom } = createSink();
    let active = false;
    const isActive = () => active;
    for (const chunk of ['\x1b[2J', '\x1b[10;2H', 'Olá', '\rX']) {
      streamTerminalOutput(terminal, chunk, isActive, () => false);
    }
    expect(writes.map((item) => item.data).join('')).toBe('\x1b[2J\x1b[10;2HOlá\rX');
    writes.forEach((item) => item.callback?.());
    expect(scrollToBottom).not.toHaveBeenCalled();
    active = true;
    refreshTerminalAfterActivation(terminal, isActive, () => false);
  });

  it('does not steal scroll position when user scrolled away', () => {
    const { terminal, writes, scrollToBottom } = createSink();
    const scrolled = { ...terminal, buffer: { active: { viewportY: 1, baseY: 50 } } };
    streamTerminalOutput(scrolled, 'output', () => true, () => false);
    writes[0]?.callback?.();
    expect(scrollToBottom).not.toHaveBeenCalled();
  });

  it('scrolls when the user was following live output', () => {
    const { terminal, writes, scrollToBottom } = createSink();
    streamTerminalOutput(terminal, 'output', () => true, () => false);
    writes[0]?.callback?.();
    expect(scrollToBottom).toHaveBeenCalledOnce();
  });

  it('keeps reconnect reset ordered before replay frames', () => {
    const { terminal, writes } = createSink();
    resetTerminalForReplay(terminal);
    streamTerminalOutput(terminal, '\x1b[Hnew screen', () => true, () => false);
    expect(writes.map((item) => item.data)).toEqual(['\x1bc', '\x1b[Hnew screen']);
  });

  it('does not render after disposal and repaints on activation', () => {
    const { terminal, writes, refresh } = createSink();
    streamTerminalOutput(terminal, 'ignored', () => true, () => true);
    refreshTerminalAfterActivation(terminal, () => false, () => false);
    expect(writes).toHaveLength(0);
    expect(refresh).not.toHaveBeenCalled();
    refreshTerminalAfterActivation(terminal, () => true, () => false);
    expect(refresh).toHaveBeenCalledWith(0, 23);
  });
});
