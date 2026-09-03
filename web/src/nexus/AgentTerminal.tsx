import React, { useEffect, useRef, useState } from 'react';
import { Eye, Keyboard, Shield, ShieldAlert, Unplug } from 'lucide-react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import {
  agentTerminalWebSocketURL,
  normalizeInitialPrompt,
  normalizeTerminalRole,
  terminalReconnectDelay,
  type TerminalRole,
} from './agentTerminalModel';
import { pushNotifications } from '../notifications/PushNotificationManager';

export const AgentTerminal: React.FC<{
  agentId: string;
  initialPrompt?: string;
  onClose?: () => void;
}> = ({ agentId, initialPrompt, onClose }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
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
    const kickoffRef = { current: normalizeInitialPrompt(initialPrompt), sent: false };

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      lineHeight: 1.25,
      fontFamily: 'var(--nx-font-mono)',
      theme: { background: '#090b10' },
      scrollback: 5000,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(container);

    const fitAndResize = () => {
      try {
        if (container.clientWidth > 0 && container.clientHeight > 0) {
          fit.fit();
        }
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
        connect();
      }, delay);
    };

    const connect = async () => {
      if (disposed) return;
      setConnection('CONNECTING');
      setMessage('');

      if (disposed) return;

      const ws = new WebSocket(agentTerminalWebSocketURL(window.location.protocol, window.location.host, agentId));
      wsRef.current = ws;
      roleRef.current = 'VIEW_ONLY';
      setRole('VIEW_ONLY');

      ws.onopen = () => {
        reconnectAttempt = 0;
        setConnection('CONNECTED');
        setMessage('');
        window.requestAnimationFrame(fitAndResize);
      };

      ws.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data);
          if (payload.type === 'output' && payload.data) {
            term.write(payload.data);
          } else if (payload.type === 'lease') {
            const next = normalizeTerminalRole(payload.role);
            roleRef.current = next;
            setRole(next);
            maybeSendKickoff();
          } else if (payload.type === 'runtime_changed') {
            setMessage('Runtime generation changed — rebinding terminal…');
            // The server currently closes the generation-bound socket after this
            // event. Closing proactively also keeps this client compatible with a
            // future server that leaves the old transport open for a grace period.
            ws.close(1012, 'runtime generation changed');
          } else if (payload.type === 'title' && payload.data) {
            setCustomTitle(payload.data);
          } else if (payload.type === 'attention') {
            if (payload.dynamic_title) setCustomTitle(payload.dynamic_title);
            pushNotifications.sendPush({
              runtimeId: payload.runtime_id || agentId,
              projectId: payload.project_id,
              projectName: payload.project_name,
              reason: payload.attention_reason || 'QUESTION',
              context: payload.context || payload.summary || 'Atenção necessária no terminal',
              dynamicTitle: payload.dynamic_title,
              attentionEventId: payload.attention_event_id,
              view: payload.view,
            });
          } else if (payload.type === 'error') {
            setMessage(String(payload.data ?? 'Terminal error'));
          }
        } catch {
          term.write(event.data);
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

    connect();

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

  const requestControl = () => {
    const ws = wsRef.current;
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'lease_acquire' }));
    }
  };

  const releaseControl = () => {
    const ws = wsRef.current;
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'lease_release' }));
    }
  };

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
          {role === 'CONTROL' ? (
            <button type="button" onClick={releaseControl} title="Release control">
              <Shield size={13} />
              Release
            </button>
          ) : (
            <button type="button" onClick={requestControl} title="Take control">
              <ShieldAlert size={13} />
              Take Control
            </button>
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
