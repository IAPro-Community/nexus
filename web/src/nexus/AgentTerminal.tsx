import React, { useEffect, useRef, useState } from 'react';
import { Eye, Keyboard, Unplug } from 'lucide-react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import {
  agentTerminalWebSocketURL,
  canOpenAgentTerminal,
  normalizeInitialPrompt,
  normalizeTerminalRole,
  terminalLeaseCommand,
  terminalReconnectDelay,
  type TerminalRole,
} from './agentTerminalModel';
import { nexus } from './api';
import { pushNotifications } from '../notifications/PushNotificationManager';
import { frameString, parseTerminalFrame } from './terminalProtocol';

export const AgentTerminal: React.FC<{
  agentId: string;
  initialPrompt?: string;
  onClose?: () => void;
}> = ({ agentId, initialPrompt, onClose }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [role, setRole] = useState<TerminalRole>('VIEW_ONLY');
  const [connection, setConnection] = useState<'CONNECTING' | 'CONNECTED' | 'DISCONNECTED' | 'ERROR'>('CONNECTING');
  const [message, setMessage] = useState('');
  const [customTitle, setCustomTitle] = useState('');

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    let disposed = false;
    let reconnectAttempt = 0;
    let reconnectTimer: number | undefined;
    const roleRef: { current: TerminalRole } = { current: 'VIEW_ONLY' };
    const wsRef: { current: WebSocket | null } = { current: null };
    const kickoffRef = { current: normalizeInitialPrompt(initialPrompt), sent: false };

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      lineHeight: 1.25,
      fontFamily: 'var(--nx-font-mono)',
      theme: {
        background: '#090b10',
        foreground: '#e6e9ef',
        cursor: '#62b4ff',
        selectionBackground: 'rgba(98, 180, 255, 0.28)',
        black: '#0b0f14',
        red: '#ff7b7b',
        green: '#51cf9a',
        yellow: '#f3bd67',
        blue: '#62b4ff',
        magenta: '#bba5ff',
        cyan: '#65d6e7',
        white: '#e6e9ef',
        brightBlack: '#626c82',
        brightRed: '#ff9b9b',
        brightGreen: '#8ae6bd',
        brightYellow: '#ffd58f',
        brightBlue: '#9acbff',
        brightMagenta: '#d6c7ff',
        brightCyan: '#a0eef5',
        brightWhite: '#ffffff',
      },
      scrollback: 5000,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(container);

    const fitAndResize = () => {
      try {
        fit.fit();
      } catch {
        return;
      }
      const ws = wsRef.current;
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', rows: term.rows, cols: term.cols }));
      }
    };

    const maybeSendKickoff = () => {
      const ws = wsRef.current;
      if (
        kickoffRef.current &&
        !kickoffRef.sent &&
        roleRef.current === 'CONTROL' &&
        ws?.readyState === WebSocket.OPEN
      ) {
        kickoffRef.sent = true;
        ws.send(JSON.stringify({ type: 'input', data: kickoffRef.current }));
      }
    };

    const scheduleReconnect = () => {
      if (disposed || reconnectTimer !== undefined) return;
      const delay = terminalReconnectDelay(reconnectAttempt++);
      setConnection('CONNECTING');
      setMessage(`Reconnecting Agent terminal in ${delay}ms…`);
      reconnectTimer = window.setTimeout(() => {
        reconnectTimer = undefined;
        void connect();
      }, delay);
    };

    const connect = async () => {
      if (disposed) return;
      try {
        const detail = await nexus.getAgent(agentId);
        if (!canOpenAgentTerminal(detail.agent.status)) {
          setConnection('DISCONNECTED');
          setMessage('Agent runtime is not running — start or recover the Agent before opening Terminal.');
          return;
        }
      } catch {
        // Let the WebSocket transport surface a transient API/server failure and
        // retain its normal reconnect behavior.
      }
      if (disposed) return;
      const ws = new WebSocket(agentTerminalWebSocketURL(window.location.protocol, window.location.host, agentId));
      wsRef.current = ws;
      roleRef.current = 'VIEW_ONLY';
      setRole('VIEW_ONLY');
      setConnection('CONNECTING');

      ws.onopen = () => {
        reconnectAttempt = 0;
        setConnection('CONNECTED');
        setMessage('');
        window.requestAnimationFrame(fitAndResize);
      };

      ws.onmessage = (event) => {
        const frame = parseTerminalFrame(event.data);
        if (frame.type === 'output') {
          term.write(frame.data);
        } else if (frame.type === 'unknown') {
          console.error('Terminal protocol error: unknown terminal frame');
          setMessage('Terminal protocol error');
        } else if (frame.type === 'agent_state') {
          setMessage(`Agent state: ${frame.state}`);
        } else if (frame.type === 'continuity_state') {
          setMessage(`Continuity: ${frame.continuity_state}`);
        } else {
          const payload = frame.payload ?? {};
          if (frame.event === 'lease') {
            const previous = roleRef.current;
            const next = normalizeTerminalRole(frameString(payload, 'role'));
            roleRef.current = next;
            setRole(next);
            const leaseCommand = terminalLeaseCommand(previous, next);
            if (leaseCommand && ws.readyState === WebSocket.OPEN) {
              ws.send(JSON.stringify({ type: leaseCommand }));
            }
            maybeSendKickoff();
          } else if (frame.event === 'runtime_changed') {
            setMessage('Runtime generation changed — rebinding terminal…');
            // The server currently closes the generation-bound socket after this
            // event. Closing proactively also keeps this client compatible with a
            // future server that leaves the old transport open for a grace period.
            ws.close(1012, 'runtime generation changed');
          } else if (frame.event === 'title' && frameString(payload, 'data')) {
            setCustomTitle(frameString(payload, 'data')!);
          } else if (frame.event === 'attention') {
            const dynamicTitle = frameString(payload, 'dynamic_title');
            if (dynamicTitle) setCustomTitle(dynamicTitle);
            pushNotifications.sendPush({
              runtimeId: frameString(payload, 'runtime_id') || agentId,
              projectName: frameString(payload, 'project_name'),
              reason: frameString(payload, 'attention_reason') || 'QUESTION',
              context: frameString(payload, 'context') || frameString(payload, 'summary') || 'Atenção necessária no terminal',
              dynamicTitle,
            });
          } else if (frame.event === 'error') {
            setMessage(frameString(payload, 'data') ?? 'Terminal error');
          }
        }
      };

      ws.onerror = () => {
        if (!disposed) {
          setConnection('ERROR');
          setMessage('Terminal transport error — reconnecting…');
        }
      };

      ws.onclose = () => {
        if (wsRef.current === ws) wsRef.current = null;
        if (disposed) return;
        setConnection('DISCONNECTED');
        scheduleReconnect();
      };
    };

    const dataDisposable = term.onData((data) => {
      const ws = wsRef.current;
      if (ws?.readyState === WebSocket.OPEN && roleRef.current === 'CONTROL') {
        ws.send(JSON.stringify({ type: 'input', data }));
      }
    });

    const observer = new ResizeObserver(() => window.requestAnimationFrame(fitAndResize));
    observer.observe(container);
    const visibility = () => {
      if (!document.hidden) window.requestAnimationFrame(fitAndResize);
    };
    document.addEventListener('visibilitychange', visibility);

    void connect();

    return () => {
      disposed = true;
      if (reconnectTimer !== undefined) window.clearTimeout(reconnectTimer);
      observer.disconnect();
      document.removeEventListener('visibilitychange', visibility);
      dataDisposable.dispose();
      wsRef.current?.close();
      term.dispose();
    };
  }, [agentId, initialPrompt]);

  return (
    <section className="nx-agent-terminal" aria-label={`Terminal for Agent ${agentId}`}>
      <header className="nx-agent-terminal__header">
        <div className="nx-agent-terminal__identity">
          <code>{customTitle || agentId}</code>
          <span data-state={connection}>{connection.toLowerCase()}</span>
        </div>
        <div className="nx-agent-terminal__status">
          <span data-role={role}>
            {role === 'CONTROL' ? <Keyboard size={13} /> : <Eye size={13} />}
            {role}
          </span>
          {message && (
            <span className="nx-terminal-message">
              <Unplug size={12} />
              {message}
            </span>
          )}
          {onClose && (
            <button type="button" onClick={onClose}>
              Detach
            </button>
          )}
        </div>
      </header>
      <div ref={containerRef} className="nx-agent-terminal__xterm" />
    </section>
  );
};
