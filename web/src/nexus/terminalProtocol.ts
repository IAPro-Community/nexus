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
  'agent_state',
  'continuity_state',
  'attention',
  'title',
  'status',
  'error',
]);

function withLegacyView(frame: TerminalFrame): TerminalFrame & LegacyTerminalView {
  Object.defineProperties(frame, {
    kind: { value: frame.type === 'output' ? 'output' : frame.type === 'control' ? 'control' : 'protocol-error' },
    data: { value: frame.type === 'output' ? frame.data : frame.type === 'control' ? frame.payload?.data : '' },
    payload: { value: frame.type === 'control' ? frame.payload ?? {} : {} },
    message: { value: frame.type === 'unknown' ? 'unknown terminal frame' : '' },
  });
  return frame as TerminalFrame & LegacyTerminalView;
}

/** Parse typed terminal envelopes without treating provider JSON as control. */
export function parseTerminalFrame(raw: unknown): TerminalFrame & LegacyTerminalView {
  if (typeof raw !== 'string') return withLegacyView({ type: 'unknown', raw: String(raw) });
  let value: unknown;
  try { value = JSON.parse(raw); } catch { return withLegacyView({ type: 'output', data: raw }); }
  if (!value || typeof value !== 'object' || Array.isArray(value)) return withLegacyView({ type: 'output', data: raw });
  const payload = value as Record<string, unknown>;
  if (!('type' in payload)) return withLegacyView({ type: 'output', data: raw });
  if (payload.type === 'output' && typeof payload.data === 'string') return withLegacyView({ type: 'output', data: payload.data });
  if (payload.type === 'agent_state' && typeof payload.agent_id === 'string' && typeof payload.state === 'string') {
    return withLegacyView({ type: 'agent_state', agent_id: payload.agent_id, state: payload.state });
  }
  if (payload.type === 'continuity_state' && typeof payload.agent_id === 'string' && typeof payload.continuity_state === 'string') {
    return withLegacyView({ type: 'continuity_state', agent_id: payload.agent_id, continuity_state: payload.continuity_state });
  }
  if (typeof payload.type === 'string' && CONTROL_TYPES.has(payload.type)) {
    const { type: event, ...controlPayload } = payload;
    return withLegacyView({ type: 'control', event, payload: controlPayload });
  }
  return withLegacyView({ type: 'unknown', raw });
}

export function frameString(payload: Record<string, unknown>, key: string): string | undefined {
  return typeof payload[key] === 'string' ? payload[key] as string : undefined;
}
