import { describe, expect, it, vi } from 'vitest';
import type { IBufferCell, IBufferLine, ILink, Terminal } from 'xterm';
import {
  createTerminalKeyEventHandler,
  createWrappedUrlLinkProvider,
  trimTrailingUrlPunctuation,
} from './terminalLinks';

function makeLine(text: string, isWrapped = false): IBufferLine {
  return {
    isWrapped,
    length: text.length,
    getCell(index: number): IBufferCell | undefined {
      const ch = text[index];
      if (ch === undefined) return undefined;
      return {
        getWidth: () => 1,
        getChars: () => ch,
      } as IBufferCell;
    },
  } as IBufferLine;
}

function makeTerminal(lines: IBufferLine[]): Pick<Terminal, 'buffer'> {
  return {
    buffer: {
      active: {
        length: lines.length,
        getLine: (y0: number) => lines[y0],
      },
    } as Terminal['buffer'],
  };
}

function collectLinks(
  term: Pick<Terminal, 'buffer'>,
  line1Based: number,
  openUrl = vi.fn(),
): ILink[] {
  const provider = createWrappedUrlLinkProvider(term, openUrl);
  let result: ILink[] | undefined;
  provider.provideLinks(line1Based, (links) => {
    result = links;
  });
  return result ?? [];
}

describe('trimTrailingUrlPunctuation', () => {
  it('strips trailing sentence punctuation while preserving balanced parens', () => {
    expect(trimTrailingUrlPunctuation('https://example.com/auth.')).toBe(
      'https://example.com/auth',
    );
    expect(trimTrailingUrlPunctuation('https://example.com/a_(b)).')).toBe(
      'https://example.com/a_(b)',
    );
    expect(trimTrailingUrlPunctuation('https://example.com/q?a=1&b=2,')).toBe(
      'https://example.com/q?a=1&b=2',
    );
  });
});

describe('createWrappedUrlLinkProvider', () => {
  it('detects single-line http(s) URLs and opens them safely on activation', () => {
    const openUrl = vi.fn();
    const term = makeTerminal([makeLine('Open https://auth.openai.com/oauth/authorize?code=123.')]);
    const links = collectLinks(term, 1, openUrl);
    expect(links).toHaveLength(1);
    expect(links[0]!.text).toBe('https://auth.openai.com/oauth/authorize?code=123');
    expect(links[0]!.range).toEqual({
      start: { x: 6, y: 1 },
      end: { x: 53, y: 1 },
    });
    links[0]!.activate({} as MouseEvent, links[0]!.text);
    expect(openUrl).toHaveBeenCalledWith('https://auth.openai.com/oauth/authorize?code=123');
  });

  it('reconstructs URLs wrapped across multiple terminal rows', () => {
    const openUrl = vi.fn();
    const part1 = 'Visit https://auth.openai.com/oauth/';
    const part2 = 'authorize?client_id=abc&redirect_uri=';
    const part3 = 'http://localhost:1455/auth/callback';
    const term = makeTerminal([
      makeLine(part1, false),
      makeLine(part2, true),
      makeLine(part3, true),
    ]);

    for (const row of [1, 2, 3]) {
      const links = collectLinks(term, row, openUrl);
      expect(links).toHaveLength(1);
      expect(links[0]!.text).toBe(
        'https://auth.openai.com/oauth/authorize?client_id=abc&redirect_uri=http://localhost:1455/auth/callback',
      );
      expect(links[0]!.range).toEqual({
        start: { x: 7, y: 1 },
        end: { x: part3.length, y: 3 },
      });
    }
  });

  it('ignores non-http(s) schemes', () => {
    const term = makeTerminal([
      makeLine('javascript:alert(1) file:///etc/passwd data:text/html,x'),
    ]);
    expect(collectLinks(term, 1)).toEqual([]);
  });
});

describe('createTerminalKeyEventHandler', () => {
  it('copies selection on Ctrl+Shift+C without forwarding to PTY', () => {
    const writeClipboard = vi.fn();
    const handler = createTerminalKeyEventHandler(
      { hasSelection: () => true, getSelection: () => 'https://example.com' },
      writeClipboard,
    );
    const event = {
      type: 'keydown',
      ctrlKey: true,
      shiftKey: true,
      altKey: false,
      metaKey: false,
      key: 'C',
      code: 'KeyC',
    } as KeyboardEvent;

    expect(handler(event)).toBe(false);
    expect(writeClipboard).toHaveBeenCalledWith('https://example.com');
  });

  it('copies selection on Cmd+C when text is selected, otherwise lets event pass', () => {
    const writeClipboard = vi.fn();
    const withSel = createTerminalKeyEventHandler(
      { hasSelection: () => true, getSelection: () => 'copied text' },
      writeClipboard,
    );
    const withoutSel = createTerminalKeyEventHandler(
      { hasSelection: () => false, getSelection: () => '' },
      writeClipboard,
    );
    const cmdC = {
      type: 'keydown',
      ctrlKey: false,
      shiftKey: false,
      altKey: false,
      metaKey: true,
      key: 'c',
      code: 'KeyC',
    } as KeyboardEvent;

    expect(withSel(cmdC)).toBe(false);
    expect(writeClipboard).toHaveBeenCalledWith('copied text');
    expect(withoutSel(cmdC)).toBe(true);
  });

  it('passes plain Ctrl+C through so SIGINT still works', () => {
    const writeClipboard = vi.fn();
    const handler = createTerminalKeyEventHandler(
      { hasSelection: () => true, getSelection: () => 'selected' },
      writeClipboard,
    );
    const ctrlC = {
      type: 'keydown',
      ctrlKey: true,
      shiftKey: false,
      altKey: false,
      metaKey: false,
      key: 'c',
      code: 'KeyC',
    } as KeyboardEvent;

    expect(handler(ctrlC)).toBe(true);
    expect(writeClipboard).not.toHaveBeenCalled();
  });
});
