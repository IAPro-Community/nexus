export type TerminalFrame =
  | { type: 'output'; data: string }
  | { type: 'agent_state'; agent_id: string; state: string }
  | { type: 'continuity_state'; agent_id: string; continuity_state: string }
  | { type: 'control'; event: string; payload?: Record<string, unknown> }
  | { type: 'unknown'; raw: string };

type LegacyTerminalView = {
  kind: 'output' | 'control' | 'protocol-error';
  data: string;
  payload: Record<string, unknown>;
  message: string;
};

const CONTROL_TYPES = new Set([
  'lease',
  'runtime_changed',
  'attention',
  'title',
  'status',
  'error',
]);

function withLegacyView(frame: TerminalFrame): TerminalFrame & LegacyTerminalView {
  Object.defineProperties(frame, {
    kind: {
      value:
        frame.type === 'output'
          ? 'output'
          : frame.type === 'control'
            ? 'control'
            : 'protocol-error',
    },
    data: {
      value:
        frame.type === 'output' ? frame.data : frame.type === 'control' ? frame.payload?.data : '',
    },
    payload: { value: frame.type === 'control' ? (frame.payload ?? {}) : {} },
    message: { value: frame.type === 'unknown' ? 'unknown terminal frame' : '' },
  });
  return frame as TerminalFrame & LegacyTerminalView;
}

/** Parse typed terminal envelopes without treating provider JSON as control. */
export function parseTerminalFrame(raw: unknown): TerminalFrame & LegacyTerminalView {
  if (typeof raw !== 'string') return withLegacyView({ type: 'unknown', raw: String(raw) });
  let value: unknown;
  try {
    value = JSON.parse(raw);
  } catch {
    return withLegacyView({ type: 'output', data: raw });
  }
  if (!value || typeof value !== 'object' || Array.isArray(value))
    return withLegacyView({ type: 'output', data: raw });
  const payload = value as Record<string, unknown>;
  if (!('type' in payload)) return withLegacyView({ type: 'output', data: raw });
  if (payload.type === 'output' && typeof payload.data === 'string') {
    return withLegacyView({ type: 'output', data: payload.data });
  }
  if (
    payload.type === 'agent_state' &&
    typeof payload.agent_id === 'string' &&
    typeof payload.state === 'string'
  ) {
    return withLegacyView({
      type: 'agent_state',
      agent_id: payload.agent_id,
      state: payload.state,
    });
  }
  if (
    payload.type === 'continuity_state' &&
    typeof payload.agent_id === 'string' &&
    typeof payload.continuity_state === 'string'
  ) {
    return withLegacyView({
      type: 'continuity_state',
      agent_id: payload.agent_id,
      continuity_state: payload.continuity_state,
    });
  }
  if (typeof payload.type === 'string' && CONTROL_TYPES.has(payload.type)) {
    const { type: event, ...controlPayload } = payload;
    return withLegacyView({ type: 'control', event, payload: controlPayload });
  }
  return withLegacyView({ type: 'unknown', raw });
}

export function frameString(payload: Record<string, unknown>, key: string): string | undefined {
  return typeof payload[key] === 'string' ? (payload[key] as string) : undefined;
}

/** True when a PTY line is a Nexus Control RPC request/response that must not render. */
export function isProtocolControlLine(line: string): boolean {
  const trimmed = line.trim();
  if (!trimmed.startsWith('{')) return false;
  try {
    const value = JSON.parse(trimmed) as Record<string, unknown>;
    if (!value || typeof value !== 'object' || Array.isArray(value)) return false;
    return 'command' in value || 'ok' in value;
  } catch {
    return false;
  }
}

/**
 * Strip Nexus Control RPC frames that leaked into PTY output (ring replay or
 * echo from older SessionHosts). Preserves ordinary provider JSON and text.
 */
export function scrubProtocolOutput(data: string): string {
  if (!data) return '';
  let out = '';
  let start = 0;
  for (let i = 0; i < data.length; i += 1) {
    if (data[i] !== '\n') continue;
    const line = data.slice(start, i + 1);
    if (!isProtocolControlLine(line)) out += line;
    start = i + 1;
  }
  if (start < data.length) {
    const rest = data.slice(start);
    if (!isProtocolControlLine(rest)) out += rest;
  }
  return out;
}

/**
 * Stateful scrubber for streamed PTY output. A control RPC line split across
 * two WebSocket messages parses as JSON in neither half, so the stateless
 * scrub above would render it. This holds back only a trailing segment that
 * could complete into a control line (starts with `{`); ordinary trailing
 * text still renders immediately.
 */
export class ProtocolScrubber {
  private tail = '';

  scrub(chunk: string): string {
    if (!chunk && !this.tail) return '';
    const data = this.tail + chunk;
    this.tail = '';
    const lastNewline = data.lastIndexOf('\n');
    const head = lastNewline < 0 ? '' : data.slice(0, lastNewline + 1);
    const tail = lastNewline < 0 ? data : data.slice(lastNewline + 1);
    let out = scrubProtocolOutput(head);
    if (tail.trimStart().startsWith('{')) {
      this.tail = tail;
    } else {
      out += tail;
    }
    return out;
  }

  flush(): string {
    const rest = this.tail;
    this.tail = '';
    if (!rest || isProtocolControlLine(rest)) return '';
    return rest;
  }
}

export interface AttentionNotificationPayload {
  runtimeId: string;
  projectId?: string;
  projectName?: string;
  reason: string;
  attentionEventId?: string;
  view?: string;
  context?: string;
  summary?: string;
  dynamicTitle?: string;
}

export function attentionNotificationFromFrame(
  frame: TerminalFrame & LegacyTerminalView,
  fallbackRuntimeId: string,
): AttentionNotificationPayload | undefined {
  if (frame.type !== 'control' || frame.event !== 'attention') return undefined;
  const payload = frame.payload ?? {};
  return {
    runtimeId: frameString(payload, 'runtime_id') ?? fallbackRuntimeId,
    projectId: frameString(payload, 'project_id'),
    projectName: frameString(payload, 'project_name'),
    reason: frameString(payload, 'attention_reason') ?? 'QUESTION',
    attentionEventId: frameString(payload, 'attention_event_id'),
    view: frameString(payload, 'view') ?? 'work',
    context: frameString(payload, 'context'),
    summary: frameString(payload, 'summary'),
    dynamicTitle: frameString(payload, 'dynamic_title'),
  };
}
