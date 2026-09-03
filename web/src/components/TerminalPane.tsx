import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { Shield, ShieldAlert, XSquare, Pencil, Check } from 'lucide-react';
import { pushNotifications } from '../notifications/PushNotificationManager';

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

    const safeFit = () => {
      try {
        if (containerRef.current && containerRef.current.clientWidth > 0 && containerRef.current.clientHeight > 0) {
          fitAddon.fit();
        }
      } catch {
        // Ignore fit measurement errors when dimensions are transiently undefined
      }
    };
    safeFit();

    termRef.current = term;
    fitAddonRef.current = fitAddon;

    // Connect WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/runtimes/${runtimeId}/terminal`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setErrorMsg('');
      safeFit();
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', rows: term.rows, cols: term.cols }));
      }
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'output' && msg.data) {
          term.write(msg.data);
        } else if (msg.type === 'lease') {
          setRole(msg.role === 'CONTROL' ? 'CONTROL' : 'VIEW_ONLY');
        } else if (msg.type === 'title' && msg.data) {
          setCustomTitle(msg.data);
          if (onUpdateTitle) onUpdateTitle(runtimeId, msg.data);
        } else if (msg.type === 'attention') {
          if (msg.dynamic_title) {
            setCustomTitle(msg.dynamic_title);
            if (onUpdateTitle) onUpdateTitle(runtimeId, msg.dynamic_title);
          }
          pushNotifications.sendPush({
            runtimeId,
            projectId: msg.project_id,
            projectName: msg.project_name,
            reason: msg.attention_reason || 'QUESTION',
            context: msg.context || msg.summary || 'Atenção necessária no terminal',
            dynamicTitle: msg.dynamic_title,
            attentionEventId: msg.attention_event_id,
            view: msg.view,
          });
        } else if (msg.type === 'error') {
          setErrorMsg(msg.data);
        }
      } catch {
        // Raw bytes fallback
        term.write(event.data);
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

    const handleWindowResize = () => {
      safeFit();
    };
    window.addEventListener('resize', handleWindowResize);

    return () => {
      dataListener.dispose();
      resizeListener.dispose();
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
    <div className="flex flex-col h-full bg-[#090d16] border border-slate-800 rounded-lg overflow-hidden shadow-xl">
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
      <div className="flex-1 w-full p-2 overflow-hidden" ref={containerRef} />
    </div>
  );
};
