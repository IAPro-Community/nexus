import React, { useEffect, useRef, useState } from 'react';
import {
  MessageSquare,
  Play,
  RefreshCw,
  Send,
  ShieldAlert,
  Sparkles,
  Trash2,
  Unplug,
  X,
} from 'lucide-react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { nexus } from './api';
import {
  TERMINAL_MAX_RECONNECT_ATTEMPTS,
  agentTerminalWebSocketURL,
  isFatalTerminalAttachError,
  normalizeInitialPrompt,
  normalizeTerminalRole,
  nextBoundRuntimeId,
  runtimeIdFromRecoverResult,
  shouldAutoRecoverAgentTerminal,
  shouldShowTerminalRecoverOverlay,
  terminalAttachFailureMessage,
  terminalReconnectDelay,
  type TerminalRole,
} from './agentTerminalModel';
import { isRequiredResourceError, recoverOrStartAgent } from './agentRecover';
import { ResourcePicker } from './ResourcePicker';
import { TerminalActionDialog } from './TerminalActionDialog';
import { scrubProtocolOutput } from './terminalProtocol';
import { ConfirmDialog, Tooltip } from '../design-system';
import type { RuntimeSession } from '../types';
import { consumePtyOutputForChrome, extractOscTitle } from '../workspace/ptyLiveChrome';
import { usePtyLiveChromeOptional } from '../workspace/PtyLiveChromeContext';
import { canFitTerminal } from './terminalFitModel';
import styles from './AgentTerminal.module.scss';

export const AgentTerminal: React.FC<{
  agentId: string;
  runtimeId?: string;
  initialPrompt?: string;
  provider?: string;
  profile?: string;
  mode?: 'Safe' | 'YOLO';
  agentName?: string;
  /** When "window", identity lives on the Desktop titlebar; actions stay in a compact toolbar. */
  chrome?: 'full' | 'window';
  /** Workspace view id so Janelas/Mosaico can mirror OSC settitle. */
  liveTitleKey?: string;
  onRecover?: () => Promise<RuntimeSession | void> | RuntimeSession | void;
  onRestartWithMode?: (
    mode: 'Safe' | 'YOLO',
  ) => Promise<RuntimeSession | void> | RuntimeSession | void;
  onClose?: (stopRuntime: boolean) => void | Promise<void>;
  onDelete?: () => void | Promise<void>;
}> = ({
  agentId,
  runtimeId,
  initialPrompt,
  provider,
  profile,
  mode = 'Safe',
  agentName,
  chrome = 'full',
  liveTitleKey,
  onRecover,
  onRestartWithMode,
  onClose,
  onDelete,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [connection, setConnection] = useState<
    'CONNECTING' | 'CONNECTED' | 'DISCONNECTED' | 'ERROR'
  >('CONNECTING');
  const [message, setMessage] = useState('');
  const [role, setRole] = useState<TerminalRole>('VIEW_ONLY');
  const [ptyTitle, setPtyTitle] = useState('');
  const [recovering, setRecovering] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [boundRuntimeId, setBoundRuntimeId] = useState(runtimeId || '');
  const [connectNonce, setConnectNonce] = useState(0);
  const [askOpen, setAskOpen] = useState(false);
  const [askPrompt, setAskPrompt] = useState('');
  const [availableSkills, setAvailableSkills] = useState<
    Array<{ id: string; name?: string; description?: string }>
  >([]);
  const [selectedSkills, setSelectedSkills] = useState<string[]>([]);
  const [asking, setAsking] = useState(false);
  const [askFeedback, setAskFeedback] = useState('');
  const [modeAction, setModeAction] = useState<'Applying' | 'Restarting' | 'Ready' | 'Error' | ''>(
    '',
  );
  const [selectedMode, setSelectedMode] = useState<'Safe' | 'YOLO'>(
    mode === 'YOLO' ? 'YOLO' : 'Safe',
  );
  const [pendingMode, setPendingMode] = useState<'Safe' | 'YOLO' | null>(null);
  const [closeConfirmOpen, setCloseConfirmOpen] = useState(false);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [closing, setClosing] = useState(false);
  const [needsResourceSelection, setNeedsResourceSelection] = useState(false);
  const [connectingForMs, setConnectingForMs] = useState(0);
  const autoRecoveredRef = useRef(false);
  const onRecoverRef = useRef(onRecover);
  onRecoverRef.current = onRecover;
  const liveChrome = usePtyLiveChromeOptional();
  const liveTitleKeyRef = useRef(liveTitleKey);
  liveTitleKeyRef.current = liveTitleKey;
  const liveChromeRef = useRef(liveChrome);
  liveChromeRef.current = liveChrome;
  const ptyChromeRef = useRef({ title: '', questionnaire: false });

  useEffect(() => {
    setBoundRuntimeId((current) => nextBoundRuntimeId(current, runtimeId));
  }, [runtimeId]);

  useEffect(() => {
    const key = liveTitleKey;
    return () => {
      if (key) liveChromeRef.current?.clearLive(key);
    };
  }, [liveTitleKey]);

  useEffect(() => {
    if (connection !== 'CONNECTING' || recovering) {
      setConnectingForMs(0);
      return;
    }
    const started = Date.now();
    const timer = window.setInterval(() => setConnectingForMs(Date.now() - started), 200);
    return () => window.clearInterval(timer);
  }, [connection, recovering]);

  const rebindTerminal = (nextRuntimeId?: string) => {
    const trimmed = (nextRuntimeId || '').trim();
    if (trimmed) setBoundRuntimeId(trimmed);
    autoRecoveredRef.current = false;
    setNeedsResourceSelection(false);
    setMessage('');
    setConnection('CONNECTING');
    // Always bump nonce — parent may already have synced the same runtime_id,
    // so boundRuntimeId alone would not remount the WebSocket effect.
    setConnectNonce((n) => n + 1);
  };

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    let disposed = false;
    let reconnectAttempt = 0;
    let reconnectTimer: number | undefined;
    let openedOnce = false;
    let stopReconnect = false;
    const roleRef: { current: TerminalRole } = { current: 'VIEW_ONLY' };
    const kickoffRef = { current: normalizeInitialPrompt(initialPrompt), sent: false };
    let lastError = '';
    let leased = false;
    let recoverInFlight = false;
    let termInstance: Terminal | null = null;
    const redrawTimers: number[] = [];

    (container as any).__triggerReconnect = () => {
      stopReconnect = false;
      reconnectAttempt = 0;
      lastError = '';
      if (reconnectTimer !== undefined) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = undefined;
      }
      connect();
    };

    (container as any).__takeControl = () => {
      const ws = wsRef.current;
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'lease_acquire' }));
      }
      termInstance?.focus();
    };

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      lineHeight: 1.25,
      fontFamily: 'var(--nx-font-mono)',
      theme: { background: '#090b10' },
      scrollback: 5000,
    });
    termInstance = term;
    termRef.current = term;
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(container);

    let lastSentSize = { rows: 0, cols: 0 };

    const publishChrome = (nextTitle?: string, questionnaire?: boolean) => {
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
      const trimmed = String(nextTitle || '').trim();
      if (trimmed) publishChrome(trimmed);
    });

    const fitAndResize = (force = false) => {
      if (
        !canFitTerminal({
          disposed,
          sessionReady: openedOnce,
          terminalConnected: term.element?.isConnected === true,
          containerConnected: container.isConnected,
          width: container.clientWidth,
          height: container.clientHeight,
        })
      ) {
        return;
      }
      try {
        fit.fit();
      } catch {
        return;
      }
      const ws = wsRef.current;
      if (ws?.readyState !== WebSocket.OPEN) return;
      if (!force && term.rows === lastSentSize.rows && term.cols === lastSentSize.cols) return;
      lastSentSize = { rows: term.rows, cols: term.cols };
      ws.send(JSON.stringify({ type: 'resize', rows: term.rows, cols: term.cols }));
    };

    const scheduleRedrawPulse = () => {
      // After history replay, TUI agents need a second SIGWINCH-sized pulse
      // or the screen stays on a stale "thinking" frame from the ring buffer.
      redrawTimers.push(window.setTimeout(() => fitAndResize(true), 120));
      redrawTimers.push(window.setTimeout(() => fitAndResize(true), 480));
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

    const failPermanently = (detail?: string) => {
      stopReconnect = true;
      if (reconnectTimer !== undefined) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = undefined;
      }
      setRecovering(false);
      setConnection('ERROR');
      setMessage(terminalAttachFailureMessage(detail));
    };

    const scheduleReconnect = (detail?: string) => {
      if (disposed || stopReconnect || reconnectTimer !== undefined) return;
      if (detail && isFatalTerminalAttachError(detail)) {
        failPermanently(detail);
        return;
      }
      if (!openedOnce && reconnectAttempt >= TERMINAL_MAX_RECONNECT_ATTEMPTS) {
        failPermanently(detail);
        return;
      }
      if (openedOnce && reconnectAttempt >= TERMINAL_MAX_RECONNECT_ATTEMPTS) {
        failPermanently(detail || 'Conexão com o terminal perdida.');
        return;
      }
      const delay = terminalReconnectDelay(reconnectAttempt++);
      setConnection('CONNECTING');
      setMessage(`Reconnecting Agent terminal in ${delay}ms…`);
      reconnectTimer = window.setTimeout(() => {
        reconnectTimer = undefined;
        connect();
      }, delay);
    };

    const recoverMissingRuntime = async (detail?: string) => {
      stopReconnect = true;
      if (reconnectTimer !== undefined) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = undefined;
      }
      if (recoverInFlight) return;
      if (autoRecoveredRef.current) {
        failPermanently(detail);
        return;
      }
      recoverInFlight = true;
      autoRecoveredRef.current = true;
      setRecovering(true);
      setConnection('CONNECTING');
      setMessage('Runtime do agente ausente — recuperando…');
      try {
        const result = onRecoverRef.current
          ? await onRecoverRef.current()
          : await recoverOrStartAgent(agentId);
        if (disposed) return;
        const nextId = runtimeIdFromRecoverResult(result);
        setBoundRuntimeId(nextId);
        stopReconnect = false;
        reconnectAttempt = 0;
        lastError = '';
        leased = false;
        setConnectNonce((n) => n + 1);
      } catch (error) {
        if (isRequiredResourceError(error)) {
          setRecovering(false);
          setNeedsResourceSelection(true);
          setMessage('Selecione um provedor/conta para iniciar o runtime deste agente.');
          setConnection('ERROR');
          return;
        }
        failPermanently(error instanceof Error ? error.message : detail);
      } finally {
        recoverInFlight = false;
      }
    };

    const connect = async () => {
      if (disposed || stopReconnect) return;
      setConnection('CONNECTING');
      if (!openedOnce) setMessage('');
      lastError = '';
      leased = false;

      const previous = wsRef.current;
      if (previous) {
        previous.onclose = null;
        previous.onerror = null;
        previous.onmessage = null;
        if (
          previous.readyState === WebSocket.OPEN ||
          previous.readyState === WebSocket.CONNECTING
        ) {
          previous.close();
        }
        if (wsRef.current === previous) wsRef.current = null;
      }

      const ws = new WebSocket(
        agentTerminalWebSocketURL(
          window.location.protocol,
          window.location.host,
          agentId,
          boundRuntimeId,
        ),
      );
      wsRef.current = ws;
      roleRef.current = 'VIEW_ONLY';
      setRole('VIEW_ONLY');

      ws.onopen = () => {
        openedOnce = true;
        setRecovering(false);
        setConnection('CONNECTED');
        setModeAction((current) => (current === 'Restarting' ? 'Ready' : current));
        setMessage((current) => (current === 'Reiniciando runtime…' ? 'Pronto' : ''));
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'lease_acquire' }));
        }
        window.requestAnimationFrame(() => {
          fitAndResize(true);
          term.focus();
          scheduleRedrawPulse();
        });
      };

      ws.onmessage = (event) => {
        if (disposed) return;
        try {
          const payload = JSON.parse(event.data);
          if (payload.type === 'output' && payload.data) {
            const cleaned = scrubProtocolOutput(String(payload.data));
            if (cleaned) {
              ingestOutput(cleaned);
              term.write(cleaned);
            }
          } else if (payload.type === 'lease') {
            const next = normalizeTerminalRole(payload.role);
            leased = true;
            reconnectAttempt = 0;
            roleRef.current = next;
            setRole(next);
            setMessage(
              next === 'CONTROL'
                ? ''
                : 'Somente leitura — outro acesso está digitando neste runtime.',
            );
            if (next === 'CONTROL') {
              window.requestAnimationFrame(() => term.focus());
            }
            maybeSendKickoff();
          } else if (payload.type === 'runtime_changed') {
            setMessage('Runtime generation changed — rebinding terminal…');
            ws.close(1012, 'runtime generation changed');
          } else if (payload.type === 'title' && payload.data) {
            const next = String(payload.data).trim();
            publishChrome(extractOscTitle(next) || next);
          } else if (payload.type === 'attention') {
            if (payload.dynamic_title) publishChrome(String(payload.dynamic_title));
            const kind = String(payload.attention_kind || payload.prompt_kind || '');
            if (
              kind === 'needs_user' ||
              kind === 'choice' ||
              kind === 'yn' ||
              kind === 'free_text'
            ) {
              publishChrome(undefined, true);
            }
          } else if (payload.type === 'error') {
            const detail = String(payload.data ?? 'Terminal error');
            lastError = detail;
            setMessage(detail);
            if (isFatalTerminalAttachError(detail)) {
              void recoverMissingRuntime(detail);
              ws.close(1011, 'fatal attach error');
            }
          }
        } catch {
          const raw = String(event.data);
          ingestOutput(raw);
          term.write(event.data);
        }
      };

      ws.onerror = () => {
        if (!disposed && !stopReconnect && !recoverInFlight) {
          setMessage('Terminal transport error — reconnecting…');
        }
      };

      ws.onclose = () => {
        if (wsRef.current === ws) wsRef.current = null;
        if (disposed || stopReconnect) return;
        if (!leased) {
          if (shouldAutoRecoverAgentTerminal(openedOnce, lastError)) {
            void recoverMissingRuntime(lastError || 'agent has no active runtime');
            return;
          }
          scheduleReconnect(lastError);
          return;
        }
        scheduleReconnect(lastError);
      };
    };

    const dataDisposable = term.onData((data) => {
      const ws = wsRef.current;
      if (!ws || ws.readyState !== WebSocket.OPEN) return;
      if (roleRef.current !== 'CONTROL') {
        // Opportunistically reclaim the writer seat, then still drop this
        // keystroke (broker would ignore it). Next keystrokes work after lease.
        ws.send(JSON.stringify({ type: 'lease_acquire' }));
        return;
      }
      ws.send(JSON.stringify({ type: 'input', data }));
    });

    const observer = new ResizeObserver(() => window.requestAnimationFrame(() => fitAndResize()));
    observer.observe(container);
    const visibility = () => {
      if (!document.hidden) window.requestAnimationFrame(() => fitAndResize());
    };
    document.addEventListener('visibilitychange', visibility);

    connect();

    return () => {
      disposed = true;
      stopReconnect = true;
      if (reconnectTimer !== undefined) window.clearTimeout(reconnectTimer);
      redrawTimers.forEach((timer) => window.clearTimeout(timer));
      observer.disconnect();
      document.removeEventListener('visibilitychange', visibility);
      try {
        titleListener.dispose();
      } catch {
        // ignore
      }
      try {
        dataDisposable.dispose();
      } catch {
        // ignore
      }
      try {
        if (wsRef.current) {
          wsRef.current.onopen = null;
          wsRef.current.onmessage = null;
          wsRef.current.onerror = null;
          wsRef.current.onclose = null;
          wsRef.current.close();
          wsRef.current = null;
        }
      } catch {
        // ignore
      }
      termRef.current = null;
      try {
        fit.dispose?.();
      } catch {
        // ignore
      }
      try {
        term.dispose();
      } catch {
        // ignore
      }
    };
  }, [agentId, boundRuntimeId, initialPrompt, connectNonce]);

  const handleManualStartOrRecover = async () => {
    setRecovering(true);
    setNeedsResourceSelection(false);
    setMessage('Iniciando runtime do agente…');
    try {
      let nextRuntimeId = '';
      if (onRecover) {
        const result = await onRecover();
        nextRuntimeId = runtimeIdFromRecoverResult(result);
      } else {
        const result = await recoverOrStartAgent(agentId);
        nextRuntimeId = runtimeIdFromRecoverResult(result);
      }
      rebindTerminal(nextRuntimeId);
    } catch (e) {
      if (isRequiredResourceError(e)) {
        setNeedsResourceSelection(true);
        setMessage('Selecione um provedor/conta para iniciar o runtime deste agente.');
        setConnection('ERROR');
      } else {
        setMessage(e instanceof Error ? e.message : String(e));
        setConnection('ERROR');
      }
    } finally {
      setRecovering(false);
    }
  };

  const handleResourceSelected = async () => {
    setRecovering(true);
    setMessage('Iniciando runtime com o recurso selecionado…');
    try {
      const result = await nexus.startAgent(agentId);
      rebindTerminal(runtimeIdFromRecoverResult(result));
    } catch (e) {
      setMessage(e instanceof Error ? e.message : String(e));
      setConnection('ERROR');
    } finally {
      setRecovering(false);
    }
  };

  const handleDeleteAgent = () => {
    if (!onDelete || deleting) return;
    setDeleteConfirmOpen(true);
  };

  const executeDeleteAgent = async () => {
    setDeleteConfirmOpen(false);
    if (!onDelete || deleting) return;
    setDeleting(true);
    setMessage('Removendo agente…');
    try {
      await onDelete();
    } catch (e) {
      setMessage(e instanceof Error ? e.message : String(e));
      setConnection('ERROR');
    } finally {
      setDeleting(false);
    }
  };

  const loadSkills = async () => {
    try {
      const status = await nexus.getMaestroStatus();
      if (status?.capabilities?.skills) {
        setAvailableSkills(status.capabilities.skills);
      }
    } catch {
      setAvailableSkills([]);
    }
  };

  const handleToggleSkill = (id: string) => {
    setSelectedSkills((prev) =>
      prev.includes(id) ? prev.filter((s) => s !== id) : prev.length < 3 ? [...prev, id] : prev,
    );
  };

  const handleSendPrompt = async () => {
    if (!askPrompt.trim() || asking) return;
    setAsking(true);
    setAskFeedback('');
    try {
      await nexus.askAgent(
        agentId,
        askPrompt.trim(),
        true,
        selectedSkills.length > 0 ? selectedSkills : undefined,
      );
      setAskPrompt('');
      setSelectedSkills([]);
      setAskOpen(false);
      setAskFeedback('Prompt enviado');
    } catch (err) {
      setAskFeedback(err instanceof Error ? err.message : String(err));
    } finally {
      setAsking(false);
    }
  };

  const requestModeChange = (nextMode: 'Safe' | 'YOLO') => {
    if (
      !onRestartWithMode ||
      nextMode === selectedMode ||
      modeAction === 'Applying' ||
      modeAction === 'Restarting'
    )
      return;
    setPendingMode(nextMode);
  };

  const handleModeChange = async () => {
    if (
      !onRestartWithMode ||
      !pendingMode ||
      modeAction === 'Applying' ||
      modeAction === 'Restarting'
    )
      return;
    const nextMode = pendingMode;
    setPendingMode(null);
    setModeAction('Applying');
    setMessage('Salvando configuração…');
    try {
      const nextRuntime = await onRestartWithMode(nextMode);
      setModeAction('Restarting');
      setMessage('Reiniciando runtime…');
      rebindTerminal(nextRuntime?.runtime_id);
      setSelectedMode(nextMode);
    } catch (error) {
      setModeAction('Error');
      setMessage(error instanceof Error ? error.message : String(error));
    }
  };

  const confirmClose = async (stopRuntime: boolean) => {
    if (!onClose || closing) return;
    setClosing(true);
    try {
      await onClose(stopRuntime);
    } finally {
      setClosing(false);
      setCloseConfirmOpen(false);
    }
  };

  const displayName = agentName || agentId;
  const showRecoverOverlay = shouldShowTerminalRecoverOverlay({
    connection,
    recovering,
    connectingForMs,
  });
  const windowChrome = chrome === 'window';

  const takeControl = () => {
    const host = containerRef.current as any;
    if (typeof host?.__takeControl === 'function') {
      host.__takeControl();
      return;
    }
    const ws = wsRef.current;
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'lease_acquire' }));
    }
  };

  const modeButtons = (
    <div className="nx-agent-terminal__modes" role="group" aria-label="Modo de execução">
      {(['Safe', 'YOLO'] as const).map((m) => (
        <button
          key={m}
          type="button"
          className="nx-agent-terminal__mode"
          data-active={selectedMode === m ? 'true' : 'false'}
          onClick={() => requestModeChange(m)}
          disabled={!onRestartWithMode || modeAction === 'Applying' || modeAction === 'Restarting'}
          title={`Alternar modo para ${m}`}
        >
          {m}
        </button>
      ))}
    </div>
  );

  const terminalActions = (
    <div className="nx-agent-terminal__controls">
      {role === 'CONTROL' ? (
        <span className="nx-agent-terminal__lease" data-role="CONTROL">
          CONTROL
        </span>
      ) : (
        <>
          <span className="nx-agent-terminal__lease" data-role="VIEW_ONLY">
            VIEW ONLY
          </span>
          <Tooltip content="Assumir controle do teclado">
            <button type="button" className="nx-agent-terminal__ask-btn" onClick={takeControl}>
              Assumir controle
            </button>
          </Tooltip>
        </>
      )}
      {ptyTitle && !windowChrome && (
        <span className="nx-agent-terminal__pty-title" title={ptyTitle}>
          {ptyTitle}
        </span>
      )}
      {message && !showRecoverOverlay && (
        <span className="nx-terminal-message">
          {modeAction === 'Error' ? <ShieldAlert size={12} /> : <Unplug size={12} />}
          {message}
        </span>
      )}
      <Tooltip content="Perguntar ao Agente / Sugerir skills">
        <button
          type="button"
          className="nx-agent-terminal__ask-btn"
          onClick={() => {
            const next = !askOpen;
            setAskOpen(next);
            if (next && availableSkills.length === 0) {
              void loadSkills();
            }
          }}
        >
          <Sparkles size={13} />
          <span>Perguntar</span>
        </button>
      </Tooltip>
      {onClose && !windowChrome && (
        <Tooltip content="Escolher como fechar este terminal">
          <button type="button" onClick={() => setCloseConfirmOpen(true)}>
            Fechar terminal
          </button>
        </Tooltip>
      )}
    </div>
  );

  return (
    <section
      className={`nx-agent-terminal ${styles.terminalRoot}`}
      data-chrome={chrome}
      aria-label={`Terminal for Agent ${agentId}`}
    >
      {!windowChrome && (
        <header className="nx-agent-terminal__header">
          <div className="nx-agent-terminal__identity">
            <span className={styles.agentBadge}>AGENT</span>
            <strong className={styles.agentTitle} title={agentId}>
              {displayName}
            </strong>
            {provider && (
              <span className={styles.providerBadge}>
                nexus {provider}
                {profile && profile !== 'default' ? `:${profile}` : ''}
              </span>
            )}
            <span data-state={connection}>{connection.toLowerCase()}</span>
          </div>
          <div className="nx-agent-terminal__status">
            {modeButtons}
            {terminalActions}
          </div>
        </header>
      )}
      {windowChrome && (
        <div
          className="nx-agent-terminal__toolbar nx-agent-terminal__toolbar--actions"
          aria-label="Terminal actions"
        >
          {modeButtons}
          {terminalActions}
        </div>
      )}
      {askOpen && (
        <div className={`nx-agent-terminal__composer ${styles.composer}`}>
          <div className={styles.composerHeader}>
            <span className={styles.composerTitle}>
              <MessageSquare size={13} /> Enviar instrução ao agente (One-shot)
            </span>
            <button type="button" onClick={() => setAskOpen(false)} className={styles.closeButton}>
              <X size={13} />
            </button>
          </div>
          <div className={styles.composerRow}>
            <input
              type="text"
              className={`nx-input ${styles.composerInput}`}
              placeholder="Digite o objetivo ou comando para o agente..."
              value={askPrompt}
              onChange={(e) => setAskPrompt(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  void handleSendPrompt();
                }
              }}
            />
            <button
              type="button"
              className="nx-button"
              data-tone="brand"
              data-size="sm"
              disabled={asking || !askPrompt.trim()}
              onClick={() => void handleSendPrompt()}
            >
              <Send size={12} />
              <span>{asking ? 'Enviando...' : 'Enviar'}</span>
            </button>
          </div>
          {availableSkills.length > 0 && (
            <div className={styles.skillsRow}>
              <span className={styles.skillsLabel}>Skills Maestro (até 3 no próximo prompt):</span>
              {availableSkills.slice(0, 8).map((s) => {
                const active = selectedSkills.includes(s.id);
                return (
                  <button
                    key={s.id}
                    type="button"
                    onClick={() => handleToggleSkill(s.id)}
                    className={styles.skillButton}
                    data-active={active ? 'true' : 'false'}
                    title={s.description || s.name || s.id}
                  >
                    {active ? '✓ ' : '+ '}
                    {s.name || s.id}
                  </button>
                );
              })}
            </div>
          )}
          {askFeedback && (
            <span
              className={
                askFeedback.includes('erro') || askFeedback.includes('failed')
                  ? styles.feedbackAlert
                  : styles.feedbackSuccess
              }
            >
              {askFeedback}
            </span>
          )}
        </div>
      )}
      {closeConfirmOpen && onClose && (
        <TerminalActionDialog
          close
          busy={closing}
          onCancel={() => setCloseConfirmOpen(false)}
          onCloseTab={() => void confirmClose(false)}
          onStopRuntime={() => void confirmClose(true)}
        />
      )}
      <TerminalActionDialog
        mode={pendingMode || undefined}
        busy={modeAction === 'Applying' || modeAction === 'Restarting'}
        onCancel={() => setPendingMode(null)}
        onConfirmMode={() => void handleModeChange()}
      />
      <div
        ref={containerRef}
        className="nx-agent-terminal__xterm"
        onPointerDown={() => termRef.current?.focus()}
      />

      {showRecoverOverlay && (
        <div className={styles.recoverOverlay} data-window-chrome={windowChrome ? 'true' : 'false'}>
          <div className={styles.recoverHeader}>
            {recovering ? <RefreshCw size={18} className="nx-spin-slow" /> : <Unplug size={18} />}
            <strong className={styles.recoverTitle}>
              {recovering || connection === 'CONNECTING'
                ? 'Recuperando o terminal do agente'
                : 'Runtime do Agente desconectado'}
            </strong>
          </div>
          <p className={styles.recoverText}>
            {recovering || connection === 'CONNECTING'
              ? 'O processo anterior não sobreviveu ao reinício da máquina ou do serviço. O Nexus está relançando o runtime para anexar de novo.'
              : 'O processo do agente não está rodando no momento ou foi finalizado. Inicie o runtime para anexar o terminal e executar comandos.'}
          </p>
          {message && connection === 'ERROR' && <p className={styles.errorMessage}>{message}</p>}
          {needsResourceSelection && (
            <div className={styles.resourcePickerWrapper}>
              <ResourcePicker
                agentId={agentId}
                preferProvider={provider}
                onSelected={() => void handleResourceSelected()}
              />
            </div>
          )}
          <div className={styles.recoverActions}>
            <button
              type="button"
              className={`nx-button ${styles.actionButtonContent}`}
              data-tone="brand"
              disabled={recovering}
              onClick={() => void handleManualStartOrRecover()}
            >
              {recovering ? <RefreshCw size={13} className="nx-spin-slow" /> : <Play size={13} />}
              <span>{recovering ? 'Iniciando…' : 'Iniciar / Recuperar Agente'}</span>
            </button>
            <button
              type="button"
              className={`nx-button ${styles.actionButtonContent}`}
              title="Reabre só o WebSocket; não relança o processo do agente"
              onClick={() => {
                setMessage('Reconectando transporte…');
                setConnection('CONNECTING');
                if (containerRef.current && (containerRef.current as any).__triggerReconnect) {
                  (containerRef.current as any).__triggerReconnect();
                } else {
                  setConnectNonce((n) => n + 1);
                }
              }}
            >
              <RefreshCw size={13} />
              <span>Reconectar WS</span>
            </button>
            {onDelete && (
              <button
                type="button"
                className={`nx-button ${styles.actionButtonContent}`}
                data-tone="danger"
                disabled={recovering || deleting}
                onClick={() => void handleDeleteAgent()}
              >
                <Trash2 size={13} />
                <span>{deleting ? 'Removendo…' : 'Remover Agente'}</span>
              </button>
            )}
          </div>
        </div>
      )}
      <ConfirmDialog
        open={deleteConfirmOpen}
        title="Remover Agente"
        description={`Remover o agente "${agentName || agentId}"? A identidade e o terminal serão excluídos.`}
        onConfirm={() => void executeDeleteAgent()}
        onCancel={() => setDeleteConfirmOpen(false)}
      />
    </section>
  );
};
