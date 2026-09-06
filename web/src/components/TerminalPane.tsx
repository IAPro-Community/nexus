import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { Shield, ShieldAlert, XSquare, Pencil, Check, Play, RefreshCw, Unplug } from 'lucide-react';
import { scrubProtocolOutput } from '../nexus/terminalProtocol';
import { canFitTerminal } from '../nexus/terminalFitModel';
import { consumePtyOutputForChrome, extractOscTitle } from '../workspace/ptyLiveChrome';
import { usePtyLiveChromeOptional } from '../workspace/PtyLiveChromeContext';

interface TerminalPaneProps {
  runtimeId: string;
  title?: string;
  provider: string;
  profile: string;
  /** Desktop mosaic titlebar already shows identity — skip the inner header. */
  hideHeader?: boolean;
  /** Workspace view id so Janelas/Mosaico can mirror OSC settitle. */
  liveTitleKey?: string;
  onUpdateTitle?: (id: string, newTitle: string) => void;
  onClose?: () => void;
  onRestart?: () => void | Promise<void>;
}

export const TerminalPane: React.FC<TerminalPaneProps> = ({
  runtimeId,
  title,
  provider,
  profile,
  hideHeader = false,
  liveTitleKey,
  onUpdateTitle,
  onClose,
  onRestart,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const liveChrome = usePtyLiveChromeOptional();
  const liveTitleKeyRef = useRef(liveTitleKey);
  liveTitleKeyRef.current = liveTitleKey;
  const liveChromeRef = useRef(liveChrome);
  liveChromeRef.current = liveChrome;
  const ptyChromeRef = useRef({ title: '', questionnaire: false });

  const [role, setRole] = useState<'CONTROL' | 'VIEW_ONLY'>('VIEW_ONLY');
  const [errorMsg, setErrorMsg] = useState<string>('');
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [customTitle, setCustomTitle] = useState(title || '');
  const [ptyTitle, setPtyTitle] = useState('');
  const [disconnected, setDisconnected] = useState(false);
  const [restarting, setRestarting] = useState(false);
  const [connectNonce, setConnectNonce] = useState(0);

  useEffect(() => {
    setCustomTitle(title || '');
  }, [title]);

  useEffect(() => {
    const key = liveTitleKey;
    return () => {
      if (key) liveChromeRef.current?.clearLive(key);
    };
  }, [liveTitleKey]);

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

    let disposed = false;
    let sessionReady = false;
    const redrawTimers: number[] = [];
    let lastSentSize = { rows: 0, cols: 0 };

    const safeFit = (force = false) => {
      if (
        !canFitTerminal({
          disposed,
          sessionReady,
          terminalConnected: term.element?.isConnected === true,
          containerConnected: containerRef.current?.isConnected === true,
          width: containerRef.current?.clientWidth || 0,
          height: containerRef.current?.clientHeight || 0,
        })
      ) {
        return;
      }
      try {
        fitAddon.fit();
      } catch {
        // Ignore fit measurement errors when dimensions are transiently undefined
      }
      if (!force) return;
      const ws = wsRef.current;
      if (ws?.readyState !== WebSocket.OPEN) return;
      lastSentSize = { rows: term.rows, cols: term.cols };
      ws.send(JSON.stringify({ type: 'resize', rows: term.rows, cols: term.cols }));
    };

    const scheduleRedrawPulse = () => {
      redrawTimers.push(window.setTimeout(() => safeFit(true), 120));
      redrawTimers.push(window.setTimeout(() => safeFit(true), 480));
    };

    const initialFit = window.requestAnimationFrame(() => safeFit(true));

    termRef.current = term;
    fitAddonRef.current = fitAddon;

    const publishChrome = (nextTitle?: string, questionnaire?: boolean) => {
      if (disposed) return;
      if (nextTitle) {
        ptyChromeRef.current.title = nextTitle;
        setPtyTitle(nextTitle);
      }
      if (questionnaire !== undefined) ptyChromeRef.current.questionnaire = questionnaire;
      const key = liveTitleKeyRef.current;
      if (key) {
        liveChromeRef.current?.setLive(key, {
          ...(nextTitle ? { title: nextTitle } : {}),
          ...(questionnaire !== undefined ? { questionnaire } : {}),
        });
      }
    };

    const ingestOutput = (data: string) => {
      if (disposed) return;
      const next = consumePtyOutputForChrome(data, ptyChromeRef.current);
      if (
        next.title !== ptyChromeRef.current.title ||
        next.questionnaire !== ptyChromeRef.current.questionnaire
      ) {
        publishChrome(next.title, next.questionnaire);
      } else {
        ptyChromeRef.current = next;
      }
    };

    const titleListener = term.onTitleChange((nextTitle) => {
      if (disposed) return;
      const trimmed = String(nextTitle || '').trim();
      if (trimmed) publishChrome(trimmed);
    });

    // Connect WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/runtimes/${runtimeId}/terminal`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      if (disposed) return;
      sessionReady = true;
      setErrorMsg('');
      setDisconnected(false);
      safeFit(true);
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'lease_acquire' }));
      }
      scheduleRedrawPulse();
      try {
        term.focus();
      } catch {
        // ignore
      }
    };

    ws.onmessage = (event) => {
      if (disposed) return;
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'output' && msg.data) {
          const cleaned = scrubProtocolOutput(String(msg.data));
          if (cleaned) {
            ingestOutput(cleaned);
            term.write(cleaned);
          }
        } else if (msg.type === 'lease') {
          setRole(msg.role === 'CONTROL' ? 'CONTROL' : 'VIEW_ONLY');
        } else if (msg.type === 'title' && msg.data) {
          const next = String(msg.data).trim();
          publishChrome(extractOscTitle(next) || next);
        } else if (msg.type === 'attention') {
          // Push is owned by NexusWorkspaceApp poll (focused project only).
          if (msg.dynamic_title) publishChrome(String(msg.dynamic_title));
          const kind = String(msg.attention_kind || msg.prompt_kind || '');
          if (kind === 'needs_user' || kind === 'choice' || kind === 'yn' || kind === 'free_text') {
            publishChrome(undefined, true);
          }
        } else if (msg.type === 'error') {
          setErrorMsg(msg.data);
          setDisconnected(true);
        }
      } catch {
        if (disposed) return;
        // Raw bytes fallback
        const raw = String(event.data);
        ingestOutput(raw);
        term.write(event.data);
      }
    };

    ws.onclose = () => {
      if (disposed) return;
      setErrorMsg((current) => current || 'O host do terminal não está em execução.');
      setDisconnected(true);
    };

    ws.onerror = () => {
      if (disposed) return;
      setErrorMsg((current) => current || 'O host do terminal não está em execução.');
      setDisconnected(true);
    };

    // Forward terminal input to WebSocket with focus/CPR sequence filtering
    const dataListener = term.onData((data) => {
      if (disposed) return;
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

    // Resize handling — FitAddon.fit triggers onResize for real size changes.
    const resizeListener = term.onResize(({ rows, cols }) => {
      if (disposed) return;
      if (rows === lastSentSize.rows && cols === lastSentSize.cols) return;
      lastSentSize = { rows, cols };
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', rows, cols }));
      }
    });

    const handleWindowResize = () => {
      safeFit();
    };
    window.addEventListener('resize', handleWindowResize);

    return () => {
      disposed = true;
      redrawTimers.forEach((timer) => window.clearTimeout(timer));
      try {
        titleListener.dispose();
      } catch {
        // ignore
      }
      try {
        dataListener.dispose();
      } catch {
        // ignore
      }
      try {
        resizeListener.dispose();
      } catch {
        // ignore
      }
      window.removeEventListener('resize', handleWindowResize);
      window.cancelAnimationFrame(initialFit);
      try {
        ws.onopen = null;
        ws.onmessage = null;
        ws.onerror = null;
        ws.onclose = null;
        ws.close();
      } catch {
        // ignore
      }
      try {
        fitAddon.dispose?.();
      } catch {
        // ignore
      }
      try {
        term.dispose();
      } catch {
        // ignore
      }
    };
  }, [runtimeId, connectNonce]);

  const handleRestart = async () => {
    if (restarting) return;
    setRestarting(true);
    try {
      if (onRestart) {
        await onRestart();
        return;
      }
      setDisconnected(false);
      setErrorMsg('');
      setConnectNonce((n) => n + 1);
    } finally {
      setRestarting(false);
    }
  };

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
    <div
      className="flex flex-col h-full bg-[#090d16] border border-slate-800 rounded-lg overflow-hidden shadow-xl relative"
      data-chrome={hideHeader ? 'window' : 'full'}
    >
      {/* Terminal Toolbar */}
      {!hideHeader && (
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
            {ptyTitle && (
              <span
                className="px-2 py-0.5 rounded-full border border-amber-700/60 bg-amber-950/40 text-amber-200 text-[10px] max-w-[220px] truncate"
                title={ptyTitle}
              >
                {ptyTitle}
              </span>
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
      )}

      {/* xterm container */}
      <div
        className="flex-1 w-full p-2 overflow-hidden"
        ref={containerRef}
        onPointerDown={() => termRef.current?.focus()}
      />
      {disconnected && (
        <div
          style={{
            position: 'absolute',
            inset: hideHeader ? 0 : '40px 0 0 0',
            background: 'rgba(9, 11, 16, 0.88)',
            backdropFilter: 'blur(3px)',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '12px',
            padding: '24px',
            zIndex: 10,
            textAlign: 'center',
          }}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              color: 'var(--nx-warning)',
            }}
          >
            <Unplug size={18} />
            <strong style={{ fontSize: '14px' }}>Terminal desconectado</strong>
          </div>
          <p
            style={{
              maxWidth: '420px',
              fontSize: '12px',
              color: 'var(--nx-muted)',
              margin: 0,
              lineHeight: 1.5,
            }}
          >
            O processo deste terminal não sobreviveu ao reinício da máquina ou do serviço. Inicie um
            novo runtime para continuar.
          </p>
          {errorMsg && (
            <p
              style={{
                maxWidth: '440px',
                margin: 0,
                padding: '8px 10px',
                borderRadius: 6,
                border: '1px solid color-mix(in srgb, var(--nx-danger) 40%, var(--nx-border))',
                background: 'var(--nx-danger-soft)',
                color: 'var(--nx-danger)',
                fontSize: '12px',
                lineHeight: 1.45,
              }}
            >
              {errorMsg}
            </p>
          )}
          <button
            type="button"
            className="nx-button"
            data-tone="brand"
            disabled={restarting}
            onClick={() => void handleRestart()}
            style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
          >
            {restarting ? <RefreshCw size={13} className="nx-spin-slow" /> : <Play size={13} />}
            <span>
              {restarting ? 'Iniciando…' : onRestart ? 'Iniciar novo terminal' : 'Reconectar'}
            </span>
          </button>
        </div>
      )}
    </div>
  );
};
