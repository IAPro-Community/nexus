/** Returns whether xterm input is safe to forward to the provider PTY. */
export function shouldForwardTerminalInput(data: string, allowMouse = false): boolean {
  if (!data) return false;

  // Browser focus reporting and CPR responses can leak from xterm while a
  // workspace panel is being switched. They are terminal metadata, not user
  // input. Some browser/PTY paths lose the leading ESC, so accept both forms.
  if (
    data === '\x1b[I' ||
    data === '\x1b[O' ||
    data === '[I' ||
    data === '[O' ||
    data === '\x1b[i' ||
    data === '[i' ||
    // Some terminal bridges encode focus reports with a parameter, e.g.
    // ESC[0[I. The ESC is invisible when this leaks into the prompt, leaving
    // the user-facing residue "0[I".
    // eslint-disable-next-line no-control-regex
    /^\x1b\[[0-9;]*(?:\[[iIO]|[iIO])$/.test(data) ||
    /^\[[0-9;]*(?:\[[iIO]|[iIO])$/.test(data) ||
    // Navigation bytes are never classified as metadata here. A split xterm
    // callback may deliver ESC first and "[A"/""[B]" next; the server-side
    // stateful coalescer must receive both pieces intact.
    // eslint-disable-next-line no-control-regex
    /^\x1b\[\d+;\d+R$/.test(data) ||
    /^\[\d+;\d+R$/.test(data)
  ) {
    return false;
  }
  // Mouse reports are only useful after the child explicitly enables tracking.
  // When forwarded through xterm the SessionHost validates that negotiated
  // state; the frontend must not delete a valid click before it reaches the PTY.
  // Bare mouse reports indicate a broken bridge; without ESC they would be
  // forwarded as printable text by the host, even when tracking is active.
  if (isTerminalMouseReport(data)) return allowMouse && data.startsWith('\x1b');
  return true;
}

// Detect full packets only; partial CSI/X10 input is assembled by the host.
function isTerminalMouseReport(data: string): boolean {
  // eslint-disable-next-line no-control-regex
  const sgr = /^\x1b?\[<\d+;\d+;\d+[Mm]$/.test(data);
  // eslint-disable-next-line no-control-regex
  const x10 = /^\x1b\[M[\x20-\xff]{3}$/.test(data);
  // eslint-disable-next-line no-control-regex
  const urxvt = /^\x1b?\[(?:[3-9]\d|\d{3,});\d+;\d+M$/.test(data);
  return sgr || x10 || urxvt;
}

// xterm may coalesce a leaked focus/CPR report with real keystrokes in one
// onData chunk, bypassing the exact-match filter above. Strip embedded
// ESC-prefixed metadata and forward the remainder. Bare (ESC-less) forms are
// only dropped on exact match: inside a larger chunk they are
// indistinguishable from literal user text such as `/e[31m`.
// eslint-disable-next-line no-control-regex
const EMBEDDED_FOCUS_RE = /\x1b\[[0-9;]*(?:\[[iIO]|[iIO])/g;
// eslint-disable-next-line no-control-regex
const EMBEDDED_CPR_RE = /\x1b\[\d+;\d+R/g;
// eslint-disable-next-line no-control-regex
const EMBEDDED_MOUSE_RE = /\x1b\[<\d+;\d+;\d+[Mm]/g;
// eslint-disable-next-line no-control-regex
const EMBEDDED_X10_MOUSE_RE = /\x1b\[M[\x20-\xff]{3}/g;
// eslint-disable-next-line no-control-regex
const EMBEDDED_URXVT_MOUSE_RE = /\x1b\[(?:[3-9]\d|\d{3,});\d+;\d+M/g;
/* eslint-disable no-control-regex -- ESC bytes identify the terminal mouse protocol. */
const TERMINAL_MOUSE_INPUT_RE =
  /(?:\x1b\[<\d+;\d+;\d+[Mm]|\x1b\[M[\x20-\xff]{3}|\x1b\[(?:[3-9]\d|\d{3,});\d+;\d+M)/;
const TERMINAL_SCROLL_NAVIGATION_RE = /\x1b(?:\[(?:[1-8](?:;\d+)?~|[HF])|O[HF])/;
/* eslint-enable no-control-regex */

/** Keep user-controlled scroll position when xterm reports a mouse or paging event. */
export function shouldAutoScrollToBottomOnInput(data: string): boolean {
  return !TERMINAL_MOUSE_INPUT_RE.test(data) && !TERMINAL_SCROLL_NAVIGATION_RE.test(data);
}

/**
 * Strip only provably terminal-generated metadata.
 * A burst of bare [A/[B segments is NOT proof of corruption: xterm may
 * legitimately deliver user text or fragmented navigation in that shape.
 * Heuristic deletion irreversibly corrupts the Codex input buffer.
 * Leave ambiguous chunks byte-transparent; reconstruction belongs at PTY.
 */
export function stripEmbeddedTerminalMetadata(data: string, allowMouse = false): string {
  if (!data) return data;
  const clean = data.replace(EMBEDDED_FOCUS_RE, '').replace(EMBEDDED_CPR_RE, '');
  return allowMouse
    ? clean
    : clean
        .replace(EMBEDDED_MOUSE_RE, '')
        .replace(EMBEDDED_X10_MOUSE_RE, '')
        .replace(EMBEDDED_URXVT_MOUSE_RE, '');
}

/**
 * Privacy-safe terminal byte signature for NEXUS_TERMINAL_TRACE.
 * Ordinary printable content is masked; only control bytes and terminal
 * syntax remain visible so CSI/SS3 fragmentation can be correlated.
 */
export function terminalTraceShape(data: string): { size: number; shape: string } {
  const bytes = new TextEncoder().encode(data);
  const limit = Math.min(bytes.length, 256);
  let shape = '';
  type TraceState = 'normal' | 'after-esc' | 'csi' | 'ss3' | 'osc' | 'osc-esc';
  let state: TraceState = 'normal';

  const appendMasked = (value: number) => {
    if (value < 0x20 || value === 0x7f) {
      shape += `<${value.toString(16).padStart(2, '0')}>`;
    } else if (value >= 0x20 && value <= 0x7e) {
      shape += '.';
    } else {
      shape += '*';
    }
  };

  const looksLikeBareSequence = (index: number) => {
    if (index + 1 >= limit || bytes[index] !== 0x5b) return false;
    const next = bytes[index + 1]!;
    if ((next >= 0x30 && next <= 0x39) || next === 0x3f || next === 0x3c || next === 0x5b) {
      return true;
    }
    return [0x41, 0x42, 0x43, 0x44, 0x48, 0x46, 0x49, 0x4f, 0x52, 0x4d, 0x6d].includes(next);
  };

  for (let index = 0; index < limit; index += 1) {
    const value = bytes[index]!;

    if (state === 'after-esc') {
      if (value === 0x5b) {
        shape += '[';
        state = 'csi';
      } else if (value === 0x4f) {
        shape += 'O';
        state = 'ss3';
      } else if (value === 0x5d) {
        shape += ']';
        state = 'osc';
      } else {
        appendMasked(value);
        state = 'normal';
      }
      continue;
    }

    if (state === 'csi') {
      if (value >= 0x30 && value <= 0x39) {
        shape += '#';
      } else if (value >= 0x40 && value <= 0x7e) {
        shape += String.fromCharCode(value);
        state = 'normal';
      } else if (value >= 0x20 && value <= 0x3f) {
        shape += String.fromCharCode(value);
      } else {
        appendMasked(value);
      }
      continue;
    }

    if (state === 'ss3') {
      if (value >= 0x20 && value <= 0x7e) {
        shape += String.fromCharCode(value);
      } else {
        appendMasked(value);
      }
      state = 'normal';
      continue;
    }

    if (state === 'osc') {
      if (value === 0x07) {
        shape += '<07>';
        state = 'normal';
      } else if (value === 0x1b) {
        shape += '<1b>';
        state = 'osc-esc';
      } else {
        appendMasked(value);
      }
      continue;
    }

    if (state === 'osc-esc') {
      if (value === 0x5c) {
        shape += '\\';
        state = 'normal';
      } else {
        appendMasked(value);
        state = 'osc';
      }
      continue;
    }

    if (value === 0x1b) {
      shape += '<1b>';
      state = 'after-esc';
    } else if (value === 0x5b && looksLikeBareSequence(index)) {
      shape += '[';
      state = 'csi';
    } else {
      appendMasked(value);
    }
  }

  if (bytes.length > limit) shape += '…';
  return { size: bytes.length, shape };
}
