import { describe, expect, it } from 'vitest';
import {
  TERMINAL_MAX_RECONNECT_ATTEMPTS,
  agentTerminalWebSocketURL,
  isFatalTerminalAttachError,
  normalizeInitialPrompt,
  normalizeTerminalRole,
  isRecoverAlreadyAlive,
  isRequiredResourceSelection,
  runtimeIdFromRecoverResult,
  shouldAutoRecoverAgentTerminal,
  shouldShowTerminalRecoverOverlay,
  shouldFallbackRecoverToStart,
  nextBoundRuntimeId,
  terminalAttachFailure,
  terminalReconnectDelay,
} from './agentTerminalModel';

describe('Agent terminal model', () => {
  it('builds agent-scoped websocket URLs', () =>
    expect(agentTerminalWebSocketURL('https:', 'nexus.local', 'agt_1')).toBe(
      'wss://nexus.local/api/v1/agents/agt_1/terminal',
    ));
  it('appends runtime_id when focusing a specific radar runtime', () =>
    expect(agentTerminalWebSocketURL('http:', 'localhost:3000', 'agt_1', 'rt_9')).toBe(
      'ws://localhost:3000/api/v1/agents/agt_1/terminal?runtime_id=rt_9',
    ));
  it('appends auth token and runtime_id for desktop sessions', () =>
    expect(agentTerminalWebSocketURL('http:', 'localhost:3000', 'agt_1', 'rt_9', 'sess_abc')).toBe(
      'ws://localhost:3000/api/v1/agents/agt_1/terminal?runtime_id=rt_9&token=sess_abc',
    ));
  it('omits auth token on Cloudflare Quick Tunnel hosts (cookie-only)', () =>
    expect(
      agentTerminalWebSocketURL('https:', 'abc.trycloudflare.com', 'agt_1', 'rt_9', 'sess_abc'),
    ).toBe('wss://abc.trycloudflare.com/api/v1/agents/agt_1/terminal?runtime_id=rt_9'));
  it('omits auth token on private --remote hosts (cookie-only)', () =>
    expect(
      agentTerminalWebSocketURL('http:', '192.168.1.10:13000', 'agt_1', 'rt_9', 'sess_abc'),
    ).toBe('ws://192.168.1.10:13000/api/v1/agents/agt_1/terminal?runtime_id=rt_9'));
  it('supports agent-scoped reattachment after a runtime generation changes', () =>
    expect(agentTerminalWebSocketURL('http:', 'localhost:3000', 'agt_1')).toBe(
      'ws://localhost:3000/api/v1/agents/agt_1/terminal',
    ));
  it('uses ws on http and encodes AgentID', () =>
    expect(agentTerminalWebSocketURL('http:', 'localhost:8080', 'a b')).toBe(
      'ws://localhost:8080/api/v1/agents/a%20b/terminal',
    ));
  it('only grants control for explicit CONTROL lease', () => {
    expect(normalizeTerminalRole('CONTROL')).toBe('CONTROL');
    expect(normalizeTerminalRole('owner')).toBe('VIEW_ONLY');
  });
  it('backs off reconnect attempts with a bounded delay', () => {
    expect(terminalReconnectDelay(0)).toBe(250);
    expect(terminalReconnectDelay(1)).toBe(500);
    expect(terminalReconnectDelay(8)).toBe(3000);
    expect(TERMINAL_MAX_RECONNECT_ATTEMPTS).toBeGreaterThan(0);
  });
  it('normalizes a direct-session kickoff prompt exactly once-ready for terminal input', () => {
    expect(normalizeInitialPrompt('  fix the failing tests  ')).toBe('fix the failing tests\n');
    expect(normalizeInitialPrompt('   ')).toBe('');
  });
  it('keeps the last live runtime id when the poll briefly omits the agent', () => {
    expect(nextBoundRuntimeId('rt_live', '')).toBe('rt_live');
    expect(nextBoundRuntimeId('rt_live', 'rt_next')).toBe('rt_next');
    expect(nextBoundRuntimeId('', 'rt_1')).toBe('rt_1');
  });

  it('never auto-recovers or creates a second runtime on transport loss', () => {
    expect(shouldAutoRecoverAgentTerminal(false, '')).toBe(false);
    expect(shouldAutoRecoverAgentTerminal(false, 'agent has no active runtime: not live')).toBe(
      false,
    );
    expect(shouldAutoRecoverAgentTerminal(true, 'Runtime host is not running (eof)')).toBe(false);
    expect(shouldAutoRecoverAgentTerminal(false, 'authentication required')).toBe(false);
  });
  it('detects fatal attach errors that must stop reconnect loops', () => {
    expect(isFatalTerminalAttachError('agent has no active runtime: not found')).toBe(true);
    expect(isFatalTerminalAttachError('Runtime host is not running (dial timeout)')).toBe(true);
    expect(isFatalTerminalAttachError('temporary blip')).toBe(false);
    expect(terminalAttachFailure('runtime not found: rt_x')).toEqual({
      kind: 'fatal_with_detail',
      detail: 'runtime not found: rt_x',
    });
    expect(terminalAttachFailure('temporary blip')).toEqual({
      kind: 'with_detail',
      detail: 'temporary blip',
    });
    expect(terminalAttachFailure('  ')).toEqual({ kind: 'generic' });
  });
  it('shows recover overlay while reconnecting after reboot, not only on ERROR', () => {
    expect(shouldShowTerminalRecoverOverlay({ connection: 'CONNECTED', recovering: false })).toBe(
      false,
    );
    expect(
      shouldShowTerminalRecoverOverlay({
        connection: 'CONNECTING',
        recovering: false,
        connectingForMs: 0,
      }),
    ).toBe(false);
    expect(
      shouldShowTerminalRecoverOverlay({
        connection: 'CONNECTING',
        recovering: false,
        connectingForMs: 400,
      }),
    ).toBe(false);
    expect(
      shouldShowTerminalRecoverOverlay({
        connection: 'CONNECTING',
        recovering: false,
        connectingForMs: 800,
      }),
    ).toBe(true);
    expect(shouldShowTerminalRecoverOverlay({ connection: 'CONNECTING', recovering: true })).toBe(
      true,
    );
    expect(shouldShowTerminalRecoverOverlay({ connection: 'ERROR', recovering: false })).toBe(true);
  });
  it('falls back recover→start for recoverable host failures', () => {
    expect(shouldFallbackRecoverToStart('agent is STOPPED (use StartAgent to restart)')).toBe(true);
    expect(shouldFallbackRecoverToStart('REQUIRED_RESOURCE_SELECTION')).toBe(false);
  });

  it('treats already-alive recover as soft success, not stop+start', () => {
    expect(isRecoverAlreadyAlive('agent runtime is already alive (no recovery needed)')).toBe(true);
    expect(
      shouldFallbackRecoverToStart('agent runtime is already alive (no recovery needed)'),
    ).toBe(false);
  });

  it('detects required resource selection and extracts runtime ids', () => {
    expect(isRequiredResourceSelection('REQUIRED_RESOURCE_SELECTION: agent x')).toBe(true);
    expect(shouldFallbackRecoverToStart('REQUIRED_RESOURCE_SELECTION: agent x')).toBe(false);
    expect(runtimeIdFromRecoverResult({ runtime_id: 'rt_1' })).toBe('rt_1');
    expect(runtimeIdFromRecoverResult({ runtime: { runtime_id: 'rt_2' } })).toBe('rt_2');
    expect(runtimeIdFromRecoverResult({})).toBe('');
  });
});
