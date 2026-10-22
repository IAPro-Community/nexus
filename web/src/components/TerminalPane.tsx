import React, { useEffect, useRef, useState } from 'react';
import { handleTerminalZoomKey } from '../nexus/terminalZoomKey';
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
import { ProtocolScrubber } from '../nexus/terminalProtocol';
import { canFitTerminal } from '../nexus/terminalFitModel';
import { TERMINAL_CONNECT_TIMEOUT_MS } from '../nexus/agentTerminalModel';
import {
  DEFAULT_TERMINAL_SCROLLBACK,
  adjustTerminalFontSize,
  getStoredTerminalFontSize,
  getTerminalTheme,
  resetTerminalFontSize,
  shouldAutoScrollToBottom,
  storeTerminalFontSize,
  subscribeTerminalFontSize,
} from '../nexus/terminalSettings';
import { consumePtyOutputForChrome, extractOscTitle } from '../workspace/ptyLiveChrome';
import { usePtyLiveChromeOptional } from '../workspace/PtyLiveChromeContext';
import { getWebSocketEndpoint } from '../api';
import {
  shouldForwardTerminalInput,
  stripEmbeddedTerminalMetadata,
  terminalTraceShape,
} from '../nexus/terminalInput';
import { VoiceCaptureButton } from '../features/voice/VoiceCaptureButton';
import { sendVoiceTranscriptToTerminal } from '../features/voice/terminalVoice';
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
  onUpdateTitle?: (id: string, newTitle: string) => void | Promise<void>;
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
  const tRef = useRef(t);
  tRef.current = t;
  const ptyChromeRef = useRef({ title: '', questionnaire: false });

  const [role, setRole] = useState<'CONTROL' | 'VIEW_ONLY'>('VIEW_ONLY');
  const [errorMsg, setErrorMsg] = useState<string>('');
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [customTitle, setCustomTitle] = useState(title || '');
  const [ptyTitle, setPtyTitle] = useState('');
  const [disconnected, setDisconnected] = useState(false);
  const [connecting, setConnecting] = useState(true);
  const [diagnosticCode, setDiagnosticCode] = useState('');
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

  const handleSaveTitle = async () => {
    const next = customTitle.trim();
    if (!next) return;
    try {
      await onUpdateTitle?.(runtimeId, next);
      setCustomTitle(next);
      setIsEditingTitle(false);
      setErrorMsg('');
    } catch (error) {
      setErrorMsg(error instanceof Error ? error.message : String(error));
    }
  };

  useEffect(() => {
    if (!containerRef.current) return;
    setConnecting(true);
    setDisconnected(false);
    setDiagnosticCode('');
    setErrorMsg('');

    // Initialize xterm
    const term = new Terminal({
      disableStdin: true,
      theme: getTerminalTheme(),
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      fontSize: initialFontSizeRef.current,
      lineHeight: 1.2,
      cursorBlink: true,
      cursorStyle: 'block',
      scrollback: DEFAULT_TERMINAL_SCROLLBACK,
    });

    term.attachCustomKeyEventHandler(handleTerminalZoomKey);

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(containerRef.current);

    let disposed = false;
    let terminalTraceEnabled = false;
    let sessionReady = false;
    // Buffers split control-RPC lines across output messages (see ProtocolScrubber).
    const protocolScrubber = new ProtocolScrubber();
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
    const connectTimeout = window.setTimeout(() => {
      if (disposed || ws.readyState !== WebSocket.CONNECTING) return;
      setErrorMsg(
        tRef.current('terminal.connectTimeout', {
          defaultValue: 'Tempo limite excedido ao conectar o terminal.',
        }),
      );
      setDiagnosticCode('TERMINAL_CONNECT_TIMEOUT');
      setConnecting(false);
      setDisconnected(true);
      console.warn(
        '[nexus:terminal]',
        JSON.stringify({
          event: 'terminal.connect.timeout',
          phase: 'TRANSPORT',
          code: 'TERMINAL_CONNECT_TIMEOUT',
          runtimeId,
          attempt: connectNonce + 1,
          timeoutMs: TERMINAL_CONNECT_TIMEOUT_MS,
        }),
      );
      ws.close();
    }, TERMINAL_CONNECT_TIMEOUT_MS);

    ws.onopen = () => {
      window.clearTimeout(connectTimeout);
      term.options.disableStdin = false;
      setConnecting(false);
      setDiagnosticCode('');
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
        if (msg.type === 'trace_config') {
          terminalTraceEnabled = msg.data === 'enabled';
        } else if (msg.type === 'output' && msg.data) {
          const cleaned = protocolScrubber.scrub(String(msg.data));
          if (cleaned) {
            ingestOutput(cleaned);
            writeTerminalOutput(cleaned);
          }
        } else if (msg.type === 'lease') {
          const nextRole = msg.role === 'CONTROL' ? 'CONTROL' : 'VIEW_ONLY';
          setRole(nextRole);
          if (nextRole === 'CONTROL' && (!panel || panel.dataset.active === 'true')) {
            window.requestAnimationFrame(() => term.focus());
          }
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
          setErrorMsg(String(msg.data || 'Terminal error'));
          setDiagnosticCode('TERMINAL_SERVER_ERROR');
          term.options.disableStdin = true;
          setConnecting(false);
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
      window.clearTimeout(connectTimeout);
      if (disposed) return;
      term.options.disableStdin = true;
      setConnecting(false);
      setDiagnosticCode((current) => current || 'TERMINAL_CONNECTION_CLOSED');
      // Release any buffered split-line tail: this stream ended.
      const rest = protocolScrubber.flush();
      if (rest) {
        ingestOutput(rest);
        writeTerminalOutput(rest);
      }
      setErrorMsg((current) => current || tRef.current('terminal.hostNotRunning'));
      setDisconnected(true);
    };

    ws.onerror = () => {
      if (disposed) return;
      setConnecting(false);
      setDiagnosticCode((current) => current || 'TERMINAL_TRANSPORT_ERROR');
      term.options.disableStdin = true;
      setErrorMsg((current) => current || tRef.current('terminal.hostNotRunning'));
      setDisconnected(true);
    };

    // Forward terminal input to WebSocket with focus/CPR sequence filtering
    const dataListener = term.onData((data) => {
      if (disposed) return;
      term.scrollToBottom();
      setIsScrolledUp(false);
      if (ws.readyState === WebSocket.OPEN && terminalTraceEnabled) {
        const trace = terminalTraceShape(data);
        ws.send(
          JSON.stringify({
            type: 'trace',
            stage: 'xterm.ondata',
            size: trace.size,
            data: trace.shape,
          }),
        );
      }
      if (!shouldForwardTerminalInput(data)) return;
      const clean = stripEmbeddedTerminalMetadata(data);
      if (!clean) return;
      if (ws.readyState === WebSocket.OPEN) {
        if (terminalTraceEnabled) {
          const trace = terminalTraceShape(clean);
          ws.send(
            JSON.stringify({
              type: 'trace',
              stage: 'xterm.filtered',
              size: trace.size,
              data: trace.shape,
            }),
          );
        }
        ws.send(JSON.stringify({ type: 'input', data: clean }));
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
    // Project Shells used to resize only on window events. Split panes change
    // their container size without resizing the browser window.
    let fitFrame: number | undefined;
    const sizeObserver = new ResizeObserver(() => {
      if (fitFrame !== undefined || disposed) return;
      fitFrame = window.requestAnimationFrame(() => {
        fitFrame = undefined;
        safeFit();
      });
    });
    if (containerRef.current) sizeObserver.observe(containerRef.current);

    return () => {
      disposed = true;
      window.clearTimeout(connectTimeout);
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
      sizeObserver.disconnect();
      if (fitFrame !== undefined) window.cancelAnimationFrame(fitFrame);
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

  // Retry always reattaches the existing PTY; transport failure is not proof the process died.
  const handleRestart = () => {
    if (restarting || connecting) return;
    setDiagnosticCode('');
    setDisconnected(false);
    setErrorMsg('');
    setConnecting(true);
    setConnectNonce((n) => n + 1);
  };

  // A new process is a separate, explicit action so retry cannot duplicate runtimes.
  const handleStartNew = async () => {
    if (!onRestart || restarting || connecting) return;
    setRestarting(true);
    try {
      await onRestart();
    } catch (error) {
      setConnecting(false);
      setDisconnected(true);
      setDiagnosticCode('TERMINAL_RESTART_FAILED');
      setErrorMsg(error instanceof Error ? error.message : String(error));
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

  const handleVoiceTranscript = (text: string) => {
    sendVoiceTranscriptToTerminal(wsRef.current, role, text);
    termRef.current?.focus();
  };

  return (
    <div
      className={styles.terminalPaneRoot}
      data-chrome={hideHeader ? 'window' : 'full'}
      onPointerDown={(e) => {
        const target = e.target as HTMLElement | null;
        if (
          !target?.closest('button, input, select, textarea, [role="button"], [role="menuitem"]')
        ) {
          termRef.current?.focus();
        }
      }}
    >
      {/* Terminal Toolbar */}
      {!hideHeader && (
        <div className="flex items-center justify-between px-3 py-2 bg-[var(--nx-terminal-surface)] border-b border-[var(--nx-terminal-border)] text-xs font-mono select-none">
          <div className="flex items-center space-x-2">
            <span className="w-2.5 h-2.5 rounded-full bg-[var(--nx-success)] animate-pulse"></span>
            {isEditingTitle ? (
              <div className="flex items-center space-x-1">
                <input
                  type="text"
                  value={customTitle}
                  onChange={(e) => setCustomTitle(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') void handleSaveTitle();
                    if (e.key === 'Escape') {
                      setCustomTitle(title || '');
                      setIsEditingTitle(false);
                    }
                  }}
                  className="bg-[var(--nx-terminal-bg)] border border-[var(--nx-terminal-cursor)] rounded px-1.5 py-0.5 text-xs text-[var(--nx-terminal-text)] focus:outline-none"
                  autoFocus
                />
                <button
                  type="button"
                  onClick={() => void handleSaveTitle()}
                  className="p-0.5 text-[var(--nx-success)] hover:opacity-80"
                >
                  <Check className="w-3.5 h-3.5" />
                </button>
              </div>
            ) : (
              <button
                type="button"
                className="flex items-center space-x-1.5 cursor-pointer group"
                onClick={() => setIsEditingTitle(true)}
                title={t('terminal.renameSessionTitle')}
                aria-label={t('terminal.renameSessionTitle')}
              >
                <span className="text-[var(--nx-terminal-text)] font-bold font-sans">
                  {customTitle || `${provider.toUpperCase()} (${profile})`}
                </span>
                <Pencil className="w-3 h-3 text-[var(--nx-terminal-muted)] opacity-0 group-hover:opacity-100 transition" />
              </button>
            )}
            {ptyTitle && (
              <span
                className="px-2 py-0.5 rounded-full border border-[var(--nx-warning)] bg-[var(--nx-warning-soft)] text-[var(--nx-warning)] text-[10px] max-w-[220px] truncate"
                title={ptyTitle}
              >
                {ptyTitle}
              </span>
            )}
            <span className="text-[var(--nx-terminal-muted)]">|</span>
            <span className="text-[var(--nx-terminal-text-soft)] uppercase text-[11px]">
              {provider}
            </span>
            <span className="text-[var(--nx-terminal-muted)] text-[11px]">({profile})</span>
            <span className="text-[var(--nx-terminal-muted)]">|</span>
            <span className="text-[var(--nx-terminal-muted)] text-[10px]">ID: {runtimeId}</span>
          </div>

          <div className="flex items-center space-x-2">
            {errorMsg && (
              <span className="text-[var(--nx-danger)] font-sans text-xs">{errorMsg}</span>
            )}

            <VoiceCaptureButton
              context="terminal"
              disabled={role !== 'CONTROL' || disconnected || connecting}
              onTranscript={handleVoiceTranscript}
            />

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
              <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold bg-[var(--nx-success-soft)] text-[var(--nx-success)] border border-[var(--nx-success)]">
                <Shield className="w-3 h-3 mr-1" />
                CONTROL
              </span>
            ) : (
              <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold bg-[var(--nx-warning-soft)] text-[var(--nx-warning)] border border-[var(--nx-warning)]">
                <ShieldAlert className="w-3 h-3 mr-1" />
                VIEW ONLY
              </span>
            )}

            {/* Lease Actions */}
            {role === 'VIEW_ONLY' ? (
              <button
                type="button"
                onClick={requestControl}
                disabled={connecting || disconnected}
                className="px-2 py-0.5 rounded bg-[var(--nx-accent)] hover:bg-[var(--nx-accent-hover)] text-[var(--nx-accent-contrast)] font-sans text-xs transition"
              >
                {t('terminal.takeControl')}
              </button>
            ) : (
              <button
                type="button"
                onClick={releaseControl}
                className="px-2 py-0.5 rounded bg-[var(--nx-surface-2)] hover:bg-[var(--nx-surface-3)] text-[var(--nx-text-soft)] font-sans text-xs transition"
              >
                Release
              </button>
            )}

            {onClose && (
              <button
                type="button"
                onClick={onClose}
                className="p-1 rounded text-[var(--nx-muted)] hover:text-[var(--nx-text)] hover:bg-[var(--nx-surface-2)] transition"
                title={t('terminal.closePane')}
                aria-label={t('terminal.closePane')}
              >
                <XSquare className="w-4 h-4" />
              </button>
            )}
          </div>
        </div>
      )}

      {hideHeader && (
        <div className={styles.voiceFloating}>
          <VoiceCaptureButton
            context="terminal"
            disabled={role !== 'CONTROL' || disconnected || connecting}
            onTranscript={handleVoiceTranscript}
          />
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

      {connecting && !disconnected && (
        <div
          className={styles.disconnectedOverlay}
          data-window-chrome={hideHeader ? 'true' : 'false'}
          role="status"
          aria-live="polite"
        >
          <div className={styles.disconnectedHeader}>
            <RefreshCw size={18} className="nx-spin-slow" />
            <strong className={styles.disconnectedTitle}>
              {t('terminal.connectingTransport', { defaultValue: 'Conectando terminal…' })}
            </strong>
          </div>
          <p className={styles.disconnectedDesc}>
            {t('terminal.connectingNotice', {
              defaultValue:
                'Aguardando a conexão segura com o processo. Você pode continuar navegando no workspace.',
            })}
          </p>
        </div>
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
          {diagnosticCode && (
            <details className={styles.errorMessage}>
              <summary>
                {t('terminal.diagnosticDetails', { defaultValue: 'Detalhes técnicos' })}
              </summary>
              <code>{diagnosticCode}</code>
              <p>
                {t('terminal.runtimeId', { defaultValue: 'Runtime' })}: {runtimeId}
              </p>
            </details>
          )}
          <button
            type="button"
            className="nx-button"
            data-tone="brand"
            disabled={restarting || connecting}
            onClick={handleRestart}
          >
            <RefreshCw size={13} />
            <span>{t('terminal.retry', { defaultValue: 'Tentar novamente' })}</span>
          </button>
          {onRestart && (
            <button
              type="button"
              className="nx-button"
              disabled={restarting || connecting}
              onClick={() => void handleStartNew()}
            >
              {restarting ? <RefreshCw size={13} className="nx-spin-slow" /> : <Play size={13} />}
              <span>{t('terminal.startNewTerminal')}</span>
            </button>
          )}
        </div>
      )}
    </div>
  );
};
