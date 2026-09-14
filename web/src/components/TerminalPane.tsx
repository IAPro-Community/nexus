import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import {
  Shield,
  ShieldAlert,
  XSquare,
  Pencil,
  Check,
  Play,
  RefreshCw,
  Unplug,
  Minus,
  Plus,
  ArrowDown,
} from 'lucide-react';
import { scrubProtocolOutput } from '../nexus/terminalProtocol';
import { canFitTerminal } from '../nexus/terminalFitModel';
import {
  DEFAULT_TERMINAL_SCROLLBACK,
  adjustTerminalFontSize,
  getStoredTerminalFontSize,
  resetTerminalFontSize,
  shouldAutoScrollToBottom,
  storeTerminalFontSize,
  subscribeTerminalFontSize,
} from '../nexus/terminalSettings';
import { consumePtyOutputForChrome, extractOscTitle } from '../workspace/ptyLiveChrome';
import { usePtyLiveChromeOptional } from '../workspace/PtyLiveChromeContext';
import { getWebSocketEndpoint } from '../api';
import { shouldForwardTerminalInput } from '../nexus/terminalInput';
import styles from './TerminalPane.module.scss';

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
  const { t } = useTranslation();
  const ptyChromeRef = useRef({ title: '', questionnaire: false });

  const [role, setRole] = useState<'CONTROL' | 'VIEW_ONLY'>('VIEW_ONLY');
  const [errorMsg, setErrorMsg] = useState<string>('');
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [customTitle, setCustomTitle] = useState(title || '');
  const [ptyTitle, setPtyTitle] = useState('');
  const [disconnected, setDisconnected] = useState(false);
  const [restarting, setRestarting] = useState(false);
  const [connectNonce, setConnectNonce] = useState(0);
  const [fontSize, setFontSize] = useState<number>(() => getStoredTerminalFontSize());
  const initialFontSizeRef = useRef(fontSize);
  const [isScrolledUp, setIsScrolledUp] = useState(false);

  useEffect(() => {
    return subscribeTerminalFontSize((newSize) => {
      setFontSize(newSize);
    });
  }, []);

  useEffect(() => {
    const term = termRef.current;
    if (!term) return;
    term.options.fontSize = fontSize;
    try {
      fitAddonRef.current?.fit();
    } catch {
      // Ignore fit measurement errors during transition
    }
  }, [fontSize]);

  useEffect(() => {
    setCustomTitle(title || '');
  }, [title]);

  useEffect(() => {
    const key = liveTitleKey;
    const chrome = liveChromeRef.current;
    return () => {
      if (key) chrome?.clearLive(key);
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
      fontSize: initialFontSizeRef.current,
      lineHeight: 1.2,
      cursorBlink: true,
      cursorStyle: 'block',
      scrollback: DEFAULT_TERMINAL_SCROLLBACK,
    });

    term.attachCustomKeyEventHandler((event: KeyboardEvent) => {
      if (event.ctrlKey || event.metaKey) {
        if (
          event.key === '=' ||
          event.key === '+' ||
          event.code === 'Equal' ||
          event.code === 'NumpadAdd'
        ) {
          if (event.type === 'keydown') {
            const next = adjustTerminalFontSize(1);
            storeTerminalFontSize(next);
          }
          return false;
        }
        if (
          event.key === '-' ||
          event.key === '_' ||
          event.code === 'Minus' ||
          event.code === 'NumpadSubtract'
        ) {
          if (event.type === 'keydown') {
            const next = adjustTerminalFontSize(-1);
            storeTerminalFontSize(next);
          }
          return false;
        }
        if (event.key === '0' || event.code === 'Digit0' || event.code === 'Numpad0') {
          if (event.type === 'keydown') {
            const next = resetTerminalFontSize();
            storeTerminalFontSize(next);
          }
          return false;
        }
      }
      return true;
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(containerRef.current);

    let disposed = false;
    let sessionReady = false;
    const panel = containerRef.current.closest<HTMLElement>('.nx-workspace-panel');
    const redrawTimers: number[] = [];
    let lastSentSize = { rows: 0, cols: 0 };

    const safeFit = (force = false) => {
      if (panel && panel.dataset.active !== 'true') return;
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

    const scrollListener = term.onScroll(() => {
      if (disposed) return;
      const buf = term.buffer.active;
      const atBottom = shouldAutoScrollToBottom(buf.viewportY, buf.baseY);
      setIsScrolledUp(!atBottom);
    });

    let pendingOutput = '';
    const writeTerminalOutput = (data: string) => {
      if (disposed) return;
      if (panel && panel.dataset.active !== 'true') {
        pendingOutput += data;
        return;
      }
      const buf = term.buffer.active;
      const wasTracking = shouldAutoScrollToBottom(buf.viewportY, buf.baseY);
      try {
        term.write(pendingOutput + data, () => {
          if (wasTracking) {
            term.scrollToBottom();
          }
        });
        pendingOutput = '';
      } catch {
        pendingOutput += data;
      }
    };
    const flushPendingOutput = () => {
      if (!pendingOutput || (panel && panel.dataset.active !== 'true')) return;
      const output = pendingOutput;
      pendingOutput = '';
      writeTerminalOutput(output);
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
    const wsUrl = getWebSocketEndpoint(`/api/v1/runtimes/${runtimeId}/terminal`);
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
      if (!panel || panel.dataset.active === 'true') {
        try {
          term.focus();
        } catch {
          // ignore
        }
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
            writeTerminalOutput(cleaned);
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
        writeTerminalOutput(String(event.data));
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
      term.scrollToBottom();
      setIsScrolledUp(false);
      if (!shouldForwardTerminalInput(data)) return;
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
    let flushFrame: number | undefined;
    const panelObserver = panel
      ? new MutationObserver(() => {
          if (panel.dataset.active !== 'true' || flushFrame !== undefined) return;
          flushFrame = window.requestAnimationFrame(() => {
            flushFrame = undefined;
            safeFit(true);
            flushPendingOutput();
            term.scrollToBottom();
          });
        })
      : null;
    if (panel)
      panelObserver?.observe(panel, { attributes: true, attributeFilter: ['data-active'] });
    window.addEventListener('resize', handleWindowResize);

    return () => {
      disposed = true;
      redrawTimers.forEach((timer) => window.clearTimeout(timer));
      try {
        scrollListener.dispose();
      } catch {
        // ignore
      }
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
      if (flushFrame !== undefined) window.cancelAnimationFrame(flushFrame);
      panelObserver?.disconnect();
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

  const handleZoomIn = () => {
    const next = adjustTerminalFontSize(1);
    storeTerminalFontSize(next);
  };

  const handleZoomOut = () => {
    const next = adjustTerminalFontSize(-1);
    storeTerminalFontSize(next);
  };

  const handleResetZoom = () => {
    const next = resetTerminalFontSize();
    storeTerminalFontSize(next);
  };

  const handleScrollToBottom = () => {
    termRef.current?.scrollToBottom();
    termRef.current?.focus();
    setIsScrolledUp(false);
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
    <div className={styles.terminalPaneRoot} data-chrome={hideHeader ? 'window' : 'full'}>
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
                  type="button"
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

          <div className="flex items-center space-x-2">
            {errorMsg && <span className="text-rose-400 font-sans text-xs">{errorMsg}</span>}

            {/* Terminal Font Size Zoom Controls */}
            <div className={styles.zoomGroup} role="group" aria-label="Terminal font size">
              <button
                type="button"
                className={styles.zoomBtn}
                onClick={handleZoomOut}
                title={t('terminal.zoomOut')}
                aria-label={t('terminal.zoomOut')}
              >
                <Minus size={11} />
              </button>
              <button
                type="button"
                className={styles.zoomLabel}
                onClick={handleResetZoom}
                title={t('terminal.resetZoom')}
                aria-label={t('terminal.resetZoom')}
              >
                {fontSize}px
              </button>
              <button
                type="button"
                className={styles.zoomBtn}
                onClick={handleZoomIn}
                title={t('terminal.zoomIn')}
                aria-label={t('terminal.zoomIn')}
              >
                <Plus size={11} />
              </button>
            </div>

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
                type="button"
                onClick={requestControl}
                className="px-2 py-0.5 rounded bg-sky-600 hover:bg-sky-500 text-white font-sans text-xs transition"
              >
                Take Control
              </button>
            ) : (
              <button
                type="button"
                onClick={releaseControl}
                className="px-2 py-0.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 font-sans text-xs transition"
              >
                Release
              </button>
            )}

            {onClose && (
              <button
                type="button"
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
        className={styles.terminalContainer}
        ref={containerRef}
        onPointerDown={() => termRef.current?.focus()}
      />

      {/* Smart scroll-to-bottom floating button */}
      {isScrolledUp && (
        <button
          type="button"
          className={styles.scrollToBottomBtn}
          onClick={handleScrollToBottom}
          title={t('terminal.scrollToBottom')}
          aria-label={t('terminal.scrollToBottom')}
        >
          <ArrowDown size={12} />
          <span>{t('terminal.scrollToBottom')}</span>
        </button>
      )}

      {disconnected && (
        <div
          className={styles.disconnectedOverlay}
          data-window-chrome={hideHeader ? 'true' : 'false'}
        >
          <div className={styles.disconnectedHeader}>
            <Unplug size={18} />
            <strong className={styles.disconnectedTitle}>{t('terminal.disconnected')}</strong>
          </div>
          <p className={styles.disconnectedDesc}>{t('terminal.disconnectedDesc')}</p>
          {errorMsg && <p className={styles.errorMessage}>{errorMsg}</p>}
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
              {restarting
                ? t('terminal.reconnecting')
                : onRestart
                  ? t('terminal.startNewTerminal')
                  : t('terminal.reconnect')}
            </span>
          </button>
        </div>
      )}
    </div>
  );
};
