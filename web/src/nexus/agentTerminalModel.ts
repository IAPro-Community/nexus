import { shouldAttachWebSocketQueryToken } from '../lib/networkHost';

export type TerminalRole = 'CONTROL' | 'VIEW_ONLY';

/** Hard stop after this many failed reconnects so the UI is not stuck on "connecting". */
export const TERMINAL_MAX_RECONNECT_ATTEMPTS = 5;
/** A blocked WebSocket handshake must never leave CONNECTING indefinitely. */
export const TERMINAL_CONNECT_TIMEOUT_MS = 30000;

export function agentTerminalWebSocketURL(
  protocol: string,
  host: string,
  agentId: string,
  runtimeId?: string,
  token?: string,
): string {
  const wsProto = protocol === 'https:' || protocol === 'wss:' ? 'wss:' : 'ws:';
  const base = `${wsProto}//${host}/api/v1/agents/${encodeURIComponent(agentId)}/terminal`;
  const params = new URLSearchParams();
  const trimmedRuntime = (runtimeId || '').trim();
  if (trimmedRuntime) params.set('runtime_id', trimmedRuntime);
  const trimmedToken = (token || '').trim();
  // Loopback only; tunnel and private --remote authenticate via HttpOnly cookie.
  if (trimmedToken && shouldAttachWebSocketQueryToken(host)) {
    params.set('token', trimmedToken);
  }
  const qs = params.toString();
  return qs ? `${base}?${qs}` : base;
}

export function normalizeTerminalRole(role: unknown): TerminalRole {
  return role === 'CONTROL' ? 'CONTROL' : 'VIEW_ONLY';
}

export function terminalReconnectDelay(attempt: number): number {
  const safeAttempt = Number.isFinite(attempt) ? Math.max(0, Math.floor(attempt)) : 0;
  return Math.min(3000, 250 * 2 ** safeAttempt);
}

export function normalizeInitialPrompt(prompt?: string): string {
  const value = (prompt || '').trim();
  return value ? `${value}\n` : '';
}

/** Fatal attach failures should stop the reconnect loop and surface a clear message. */
export function isFatalTerminalAttachError(message: string): boolean {
  const lower = (message || '').toLowerCase();
  return (
    lower.includes('no active runtime') ||
    lower.includes('runtime not found') ||
    lower.includes('runtime host is not running') ||
    lower.includes('no longer responding') ||
    lower.includes('is not live') ||
    lower.includes('authentication required') ||
    lower.includes('invalid origin')
  );
}

export function nextBoundRuntimeId(current: string, incoming?: string, preferred?: string): string {
  const pref = (preferred || '').trim();
  const cur = (current || '').trim();
  const next = (incoming || '').trim();
  // Preferred only wins when the host already selected it as the live incoming runtime.
  if (pref && pref === next) return pref;
  if (!next) return cur;
  if (!cur) return next;
  // Sticky: keep the bound generation. Polls may briefly list a sibling/stale runtime.
  return cur;
}

export function isLiveRuntimeState(state?: string): boolean {
  switch ((state || '').toUpperCase()) {
    case 'RUNNING':
    case 'STARTING':
    case 'HANDOFF':
    case 'WAITING':
    case 'APPROVAL':
      return true;
    default:
      return false;
  }
}

/** Transport loss never relaunches a runtime; recovery is an explicit action. */
export function shouldAutoRecoverAgentTerminal(openedOnce: boolean, lastError?: string): boolean {
  void openedOnce;
  void lastError;
  return false;
}

export type TerminalAttachFailure =
  | { kind: 'fatal_with_detail'; detail: string }
  | { kind: 'with_detail'; detail: string }
  | { kind: 'generic' };

/** Classifies an attach failure so the UI translates the hint at render time. */
export function terminalAttachFailure(detail?: string): TerminalAttachFailure {
  const trimmed = (detail || '').trim();
  if (trimmed && isFatalTerminalAttachError(trimmed)) {
    return { kind: 'fatal_with_detail', detail: trimmed };
  }
  if (trimmed) return { kind: 'with_detail', detail: trimmed };
  return { kind: 'generic' };
}

export function shouldShowTerminalRecoverOverlay(input: {
  connection: 'CONNECTING' | 'CONNECTED' | 'DISCONNECTED' | 'ERROR';
  recovering: boolean;
  connectingForMs?: number;
}): boolean {
  void input.connectingForMs;
  if (input.recovering) return true;
  if (input.connection === 'ERROR' || input.connection === 'DISCONNECTED') return true;
  // CONNECTING alone is not a recover failure — use a light status, not the fullscreen overlay.
  return false;
}

/** Parse Gate-4 runtime_changed payload for the new generation id. */
export function parseRuntimeChangedId(data: unknown): string {
  if (data == null) return '';
  let record: unknown = data;
  if (typeof data === 'string') {
    const trimmed = data.trim();
    if (!trimmed) return '';
    try {
      record = JSON.parse(trimmed) as unknown;
    } catch {
      return '';
    }
  }
  if (!record || typeof record !== 'object') return '';
  const payload = record as { new_runtime_id?: unknown; newRuntimeId?: unknown };
  const fromSnake = typeof payload.new_runtime_id === 'string' ? payload.new_runtime_id.trim() : '';
  if (fromSnake) return fromSnake;
  return typeof payload.newRuntimeId === 'string' ? payload.newRuntimeId.trim() : '';
}

/** Recover refused because the runtime is healthy — reconnect, do not stop+start. */
export function isRecoverAlreadyAlive(message: string): boolean {
  return (message || '').toLowerCase().includes('already alive');
}

/** Start/Recover needs an explicit provider allocation first. */
export function isRequiredResourceSelection(message: string): boolean {
  return (message || '').toLowerCase().includes('required_resource_selection');
}

/** Whether a recover failure should fall through to StartAgent. */
export function shouldFallbackRecoverToStart(message: string): boolean {
  const lower = (message || '').toLowerCase();
  if (isRecoverAlreadyAlive(lower)) return false;
  if (isRequiredResourceSelection(lower)) return false;
  return (
    lower.includes('no recoverable runtime') ||
    lower.includes('use startagent') ||
    lower.includes('agent is stopped') ||
    lower.includes('host did not accept') ||
    lower.includes('no longer responding') ||
    lower.includes('runtime not found') ||
    lower.includes('not live')
  );
}

/** Extract runtime_id from Recover/Start API shapes used by the terminal overlay. */
export function runtimeIdFromRecoverResult(result: unknown): string {
  if (!result || typeof result !== 'object') return '';
  const record = result as { runtime_id?: unknown; runtime?: { runtime_id?: unknown } };
  const direct = typeof record.runtime_id === 'string' ? record.runtime_id.trim() : '';
  if (direct) return direct;
  const nested =
    record.runtime && typeof record.runtime.runtime_id === 'string'
      ? record.runtime.runtime_id.trim()
      : '';
  return nested;
}
