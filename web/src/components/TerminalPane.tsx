import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { Shield, ShieldAlert, XSquare, Pencil, Check } from 'lucide-react';
import { pushNotifications } from '../notifications/PushNotificationManager';
import { frameString, parseTerminalFrame } from '../nexus/terminalProtocol';

interface TerminalPaneProps {
  runtimeId: string;
  title?: string;
  provider: string;
  profile: string;
  onUpdateTitle?: (id: string, newTitle: string) => void;
  onClose?: () => void;
}

export const TerminalPane: React.FC<TerminalPaneProps> = ({
  runtimeId,
  title,
  provider,
  profile,
  onUpdateTitle,
  onClose,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  const [role, setRole] = useState<'CONTROL' | 'VIEW_ONLY'>('VIEW_ONLY');
  const [errorMsg, setErrorMsg] = useState<string>('');
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [customTitle, setCustomTitle] = useState(title || '');

  useEffect(() => {
    setCustomTitle(title || '');
  }, [title]);

  const handleSaveTitle = () => {
    setIsEditingTitle(false);
    if (customTitle.trim() && onUpdateTitle) {
      onUpdateTitle(runtimeId, customTitle.trim());
    }
  };

  useEffect(() => {
    if (!containerRef.current) return;

    // Initialize xterm
    const term = new Terminal({
      theme: {
        background: '#090d16',
        foreground: '#e2e8f0',
        cursor: '#38bdf8',
        selectionBackground: 'rgba(56, 189, 248, 0.3)',
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
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      fontSize: 13,
      lineHeight: 1.2,
      cursorBlink: true,
      cursorStyle: 'block',
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(containerRef.current);
    fitAddon.fit();

    termRef.current = term;
    fitAddonRef.current = fitAddon;

    // Connect WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/runtimes/${runtimeId}/terminal`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setErrorMsg('');
      fitAddon.fit();
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', rows: term.rows, cols: term.cols }));
      }
    };

    ws.onmessage = (event) => {
      const frame = parseTerminalFrame(event.data);
      if (frame.kind === 'output') {
        term.write(frame.data);
      } else if (frame.kind === 'protocol-error') {
        console.error(`Terminal protocol error: ${frame.message}`);
        setErrorMsg('Terminal protocol error');
      } else if (frame.type === 'control') {
        const msg = frame.payload || {};
        if (frame.event === 'lease') {
          setRole(frameString(msg, 'role') === 'CONTROL' ? 'CONTROL' : 'VIEW_ONLY');
        } else if (frame.event === 'title' && frameString(msg, 'data')) {
          const data = frameString(msg, 'data')!;
          setCustomTitle(data);
          if (onUpdateTitle) onUpdateTitle(runtimeId, data);
        } else if (frame.event === 'attention') {
          const dynamicTitle = frameString(msg, 'dynamic_title');
          if (dynamicTitle) {
            setCustomTitle(dynamicTitle);
            if (onUpdateTitle) onUpdateTitle(runtimeId, dynamicTitle);
          }
          pushNotifications.sendPush({
            runtimeId,
            projectName: frameString(msg, 'project_name'),
            reason: frameString(msg, 'attention_reason') || 'QUESTION',
            context: frameString(msg, 'context') || frameString(msg, 'summary') || 'Atenção necessária no terminal',
            dynamicTitle,
          });
        } else if (frame.event === 'error') {
          setErrorMsg(frameString(msg, 'data') || 'Terminal error');
        }
      }
    };

    ws.onclose = () => {
      setErrorMsg('Disconnected from runtime');
    };

    ws.onerror = () => {
      setErrorMsg('WebSocket connection error');
    };

    // Forward terminal input to WebSocket with focus/CPR sequence filtering
    const dataListener = term.onData((data) => {
      // Discard browser focus in/out and cursor position report sequences that leak during tab switching
      if (
        data === '\x1b[I' ||
        data === '\x1b[O' ||
        data === '[I' ||
        data === '[O' ||
        // eslint-disable-next-line no-control-regex
        /^\x1b\[\d+;\d+R$/.test(data) ||
        /^\[\d+;\d+R$/.test(data)
      ) {
        return;
      }
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }));
      }
    });

    // Resize handling
    const resizeListener = term.onResize(({ rows, cols }) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', rows, cols }));
      }
    });

    const handleWindowResize = () => fitAddon.fit();
    const resizeObserver = new ResizeObserver(() => {
      window.requestAnimationFrame(handleWindowResize);
    });
    resizeObserver.observe(containerRef.current);
    window.addEventListener('resize', handleWindowResize);

    return () => {
      dataListener.dispose();
      resizeListener.dispose();
      resizeObserver.disconnect();
      window.removeEventListener('resize', handleWindowResize);
      ws.close();
      term.dispose();
    };
  }, [runtimeId]);

  const requestControl = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'lease_acquire' }));
    }
  };

  const releaseControl = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'lease_release' }));
    }
  };

  return (
    <div className="nx-agent-terminal">
      {/* Terminal Toolbar */}
      <div className="flex items-center justify-between px-3 py-2 bg-slate-900 border-b border-slate-800 text-xs font-mono select-none">
        <div className="flex items-center space-x-2">
          <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
          {isEditingTitle ? (
            <div className="flex items-center space-x-1">
              <input
                type="text"
                value={customTitle}
                onChange={(e) => setCustomTitle(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleSaveTitle()}
                className="bg-slate-950 border border-sky-500 rounded px-1.5 py-0.5 text-xs text-white focus:outline-none"
                autoFocus
              />
              <button
                onClick={handleSaveTitle}
                className="p-0.5 text-emerald-400 hover:text-emerald-300"
              >
                <Check className="w-3.5 h-3.5" />
              </button>
            </div>
          ) : (
            <div
              className="flex items-center space-x-1.5 cursor-pointer group"
              onClick={() => setIsEditingTitle(true)}
              title="Click to rename session title"
            >
              <span className="text-slate-100 font-bold font-sans">
                {customTitle || `${provider.toUpperCase()} (${profile})`}
              </span>
              <Pencil className="w-3 h-3 text-slate-500 opacity-0 group-hover:opacity-100 transition" />
            </div>
          )}
          <span className="text-slate-600">|</span>
          <span className="text-slate-400 uppercase text-[11px]">{provider}</span>
          <span className="text-slate-500 text-[11px]">({profile})</span>
          <span className="text-slate-600">|</span>
          <span className="text-slate-500 text-[10px]">ID: {runtimeId}</span>
        </div>

        <div className="flex items-center space-x-3">
          {errorMsg && <span className="text-rose-400 font-sans text-xs">{errorMsg}</span>}

          {/* Lease Status Badge */}
          {role === 'CONTROL' ? (
            <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold bg-emerald-950 text-emerald-300 border border-emerald-800">
              <Shield className="w-3 h-3 mr-1" />
              CONTROL
            </span>
          ) : (
            <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold bg-amber-950 text-amber-300 border border-amber-800">
              <ShieldAlert className="w-3 h-3 mr-1" />
              VIEW ONLY
            </span>
          )}

          {/* Lease Actions */}
          {role === 'VIEW_ONLY' ? (
            <button
              onClick={requestControl}
              className="px-2 py-0.5 rounded bg-sky-600 hover:bg-sky-500 text-white font-sans text-xs transition"
            >
              Take Control
            </button>
          ) : (
            <button
              onClick={releaseControl}
              className="px-2 py-0.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 font-sans text-xs transition"
            >
              Release
            </button>
          )}

          {onClose && (
            <button
              onClick={onClose}
              className="p-1 rounded text-slate-400 hover:text-white hover:bg-slate-800 transition"
              title="Close Pane"
            >
              <XSquare className="w-4 h-4" />
            </button>
          )}
        </div>
      </div>

      {/* xterm container */}
      <div className="nx-agent-terminal__xterm" ref={containerRef} />
    </div>
  );
};
