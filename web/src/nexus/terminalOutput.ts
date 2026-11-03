import { shouldAutoScrollToBottom } from './terminalSettings';

/** Minimal xterm contract, also usable with deterministic output tests. */
export interface TerminalOutputSink {
  readonly buffer: { readonly active: { readonly viewportY: number; readonly baseY: number } };
  readonly rows: number;
  write(data: string, callback?: () => void): void;
  scrollToBottom(): void;
  refresh(start: number, end: number): void;
}

/**
 * Write PTY bytes immediately, including while a workspace panel is hidden.
 *
 * Deferring output in a JavaScript string produced an unbounded memory queue.
 * xterm parses the stream in-order into its bounded scrollback even when a
 * panel is hidden; resizing and repainting wait until the panel is visible.
 * Never truncate a stream of incremental ANSI operations.
 */
export function streamTerminalOutput(
  terminal: TerminalOutputSink,
  data: string,
  isActive: () => boolean,
  isDisposed: () => boolean,
): void {
  if (!data || isDisposed()) return;
  const buffer = terminal.buffer.active;
  const follow = isActive() && shouldAutoScrollToBottom(buffer.viewportY, buffer.baseY);
  terminal.write(data, () => {
    if (!isDisposed() && isActive() && follow) terminal.scrollToBottom();
  });
}

/** Repaint after a hidden panel becomes visible; never replay its PTY bytes. */
export function refreshTerminalAfterActivation(
  terminal: TerminalOutputSink,
  isActive: () => boolean,
  isDisposed: () => boolean,
): void {
  if (isDisposed() || !isActive()) return;
  terminal.refresh(0, Math.max(0, terminal.rows - 1));
}

/** Reset stale incremental TUI state before an attach replays its ring buffer. */
export function resetTerminalForReplay(terminal: Pick<TerminalOutputSink, 'write'>): void {
  // Send RIS through the same ordered xterm write queue as PTY data.
  terminal.write('\x1bc');
}
