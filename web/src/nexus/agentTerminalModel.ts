import { shouldAttachWebSocketQueryToken } from '../lib/networkHost';

export type TerminalRole = 'CONTROL' | 'VIEW_ONLY';

/** Hard stop after this many failed reconnects so the UI is not stuck on "connecting". */
export const TERMINAL_MAX_RECONNECT_ATTEMPTS = 5;

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

export function nextBoundRuntimeId(current: string, incoming?: string): string {
  const next = (incoming || '').trim();
  return next || (current || '').trim();
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
  if (input.recovering) return true;
  if (input.connection === 'ERROR' || input.connection === 'DISCONNECTED') return true;
  if (input.connection === 'CONNECTING' && (input.connectingForMs ?? 0) >= 800) return true;
  return false;
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
