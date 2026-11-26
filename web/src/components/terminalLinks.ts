import type { IBufferLine, ILink, ILinkProvider, Terminal } from 'xterm';
import { handleTerminalZoomKey } from '../nexus/terminalZoomKey';
import { safeExternalUrl } from '../platform/externalUrl';

const URL_CANDIDATE_RE = /https?:\/\/[^\s<>"'`]+/g;
const SIMPLE_TRAILING_PUNCTUATION = new Set(['.', ',', ';', ':', '!', '?', '"', "'"]);

interface CellPosition {
  x: number;
  y: number;
}

function defaultOpenUrl(url: string): void {
  if (typeof window === 'undefined') return;
  window.open(url, '_blank', 'noopener,noreferrer');
}

function defaultWriteClipboard(text: string): void {
  if (typeof navigator === 'undefined' || !navigator.clipboard?.writeText) return;
  void navigator.clipboard.writeText(text).catch(() => {
    // Clipboard writes can be rejected in restricted browser sandboxes.
  });
}

function hasUnbalancedClosing(url: string, openChar: string, closeChar: string): boolean {
  let balance = 0;
  for (const ch of url) {
    if (ch === openChar) balance += 1;
    else if (ch === closeChar) balance -= 1;
  }
  return balance < 0;
}

/** Trim trailing sentence punctuation while keeping balanced paired delimiters. */
export function trimTrailingUrlPunctuation(raw: string): string {
  let end = raw.length;
  while (end > 0) {
    const ch = raw[end - 1]!;
    if (SIMPLE_TRAILING_PUNCTUATION.has(ch)) {
      end -= 1;
      continue;
    }
    const candidate = raw.slice(0, end);
    if (
      (ch === ')' && hasUnbalancedClosing(candidate, '(', ')')) ||
      (ch === ']' && hasUnbalancedClosing(candidate, '[', ']')) ||
      (ch === '}' && hasUnbalancedClosing(candidate, '{', '}'))
    ) {
      end -= 1;
      continue;
    }
    break;
  }
  return raw.slice(0, end);
}

function appendLineCells(
  line: IBufferLine,
  y1: number,
  trimTrailingSpaces: boolean,
  positions: CellPosition[],
  chars: string[],
): void {
  const rowChars: string[] = [];
  const rowPositions: CellPosition[] = [];
  for (let col = 0; col < line.length; col += 1) {
    const cell = line.getCell(col);
    if (!cell || cell.getWidth() === 0) continue;
    const text = cell.getChars() || ' ';
    for (let i = 0; i < text.length; i += 1) {
      rowChars.push(text[i]!);
      rowPositions.push({ x: col + 1, y: y1 });
    }
  }
  if (trimTrailingSpaces) {
    while (rowChars.length > 0 && rowChars[rowChars.length - 1] === ' ') {
      rowChars.pop();
      rowPositions.pop();
    }
  }
  for (let i = 0; i < rowChars.length; i += 1) {
    chars.push(rowChars[i]!);
    positions.push(rowPositions[i]!);
  }
}

/**
 * Provide clickable http(s) links across soft-wrapped xterm buffer lines.
 * xterm's built-in buffer indices are 0-based for getLine() and 1-based for
 * provideLinks()/IBufferRange.
 */
export function createWrappedUrlLinkProvider(
  term: Pick<Terminal, 'buffer'>,
  openUrl: (url: string) => void = defaultOpenUrl,
): ILinkProvider {
  return {
    provideLinks(bufferLineNumber: number, callback: (links: ILink[] | undefined) => void): void {
      const buffer = term.buffer?.active;
      if (!buffer || bufferLineNumber < 1 || bufferLineNumber > buffer.length) {
        callback(undefined);
        return;
      }

      let startY0 = bufferLineNumber - 1;
      while (startY0 > 0 && buffer.getLine(startY0)?.isWrapped) {
        startY0 -= 1;
      }
      let endY0 = bufferLineNumber - 1;
      while (endY0 + 1 < buffer.length && buffer.getLine(endY0 + 1)?.isWrapped) {
        endY0 += 1;
      }

      const chars: string[] = [];
      const positions: CellPosition[] = [];
      for (let y0 = startY0; y0 <= endY0; y0 += 1) {
        const line = buffer.getLine(y0);
        if (!line) continue;
        appendLineCells(line, y0 + 1, y0 === endY0, positions, chars);
      }

      const logicalText = chars.join('');
      if (!logicalText.includes('http://') && !logicalText.includes('https://')) {
        callback(undefined);
        return;
      }

      const links: ILink[] = [];
      URL_CANDIDATE_RE.lastIndex = 0;
      let match: RegExpExecArray | null;
      while ((match = URL_CANDIDATE_RE.exec(logicalText)) !== null) {
        const cleaned = trimTrailingUrlPunctuation(match[0]);
        const safe = safeExternalUrl(cleaned);
        if (!safe || cleaned.length === 0) continue;

        const startIdx = match.index;
        const endIdx = startIdx + cleaned.length - 1;
        const startPos = positions[startIdx];
        const endPos = positions[endIdx];
        if (!startPos || !endPos) continue;
        if (bufferLineNumber < startPos.y || bufferLineNumber > endPos.y) continue;

        links.push({
          range: { start: startPos, end: endPos },
          text: cleaned,
          activate: () => {
            openUrl(safe);
          },
        });
      }

      callback(links.length > 0 ? links : undefined);
    },
  };
}

export type TerminalCopyTarget = Pick<Terminal, 'hasSelection' | 'getSelection'>;

/**
 * Combines terminal font zoom shortcuts with Ctrl+Shift+C / Cmd+C clipboard copy.
 * Leaves plain Ctrl+C untouched so SIGINT (\x03) continues reaching the PTY.
 */
export function createTerminalKeyEventHandler(
  term: TerminalCopyTarget,
  writeClipboard: (text: string) => void = defaultWriteClipboard,
): (event: KeyboardEvent) => boolean {
  return (event: KeyboardEvent): boolean => {
    if (!handleTerminalZoomKey(event)) return false;

    const keyLower = event.key.toLowerCase();
    const isCtrlShiftCopy =
      event.ctrlKey && event.shiftKey && !event.altKey && !event.metaKey && keyLower === 'c';
    const isMetaCopy =
      event.metaKey && !event.ctrlKey && !event.altKey && !event.shiftKey && keyLower === 'c';

    if (isCtrlShiftCopy || (isMetaCopy && term.hasSelection())) {
      if (event.type === 'keydown') {
        const selected = term.getSelection();
        if (selected) writeClipboard(selected);
      }
      return false;
    }

    return true;
  };
}
