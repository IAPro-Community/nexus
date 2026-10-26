import React, { useEffect, useMemo, useRef, useState } from 'react';
import { handleTerminalZoomKey } from './terminalZoomKey';
import { useTranslation } from 'react-i18next';
import {
  ArrowDown,
  Check,
  Copy,
  CornerDownLeft,
  MessageSquare,
  Minus,
  Play,
  Plus,
  RefreshCw,
  Search,
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
import { getDesktopAuthToken, getDesktopWebSocketBaseUrl } from '../api';
import {
  TERMINAL_MAX_RECONNECT_ATTEMPTS,
  TERMINAL_CONNECT_TIMEOUT_MS,
  agentTerminalWebSocketURL,
  isFatalTerminalAttachError,
  normalizeInitialPrompt,
  normalizeTerminalRole,
  nextBoundRuntimeId,
  parseRuntimeChangedId,
  runtimeIdFromRecoverResult,
  shouldShowTerminalRecoverOverlay,
  terminalAttachFailure,
  terminalReconnectDelay,
  type TerminalRole,
} from './agentTerminalModel';
import {
  DEFAULT_TERMINAL_SCROLLBACK,
  adjustTerminalFontSize,
  getStoredTerminalFontSize,
  getTerminalTheme,
  resetTerminalFontSize,
  shouldAutoScrollToBottom,
  storeTerminalFontSize,
  subscribeTerminalFontSize,
} from './terminalSettings';
import { isRequiredResourceError, recoverOrStartAgent } from './agentRecover';
import { ResourcePicker } from './ResourcePicker';
import { TerminalActionDialog } from './TerminalActionDialog';
import { ProtocolScrubber } from './terminalProtocol';
import {
  shouldForwardTerminalInput,
  stripEmbeddedTerminalMetadata,
  terminalTraceShape,
} from './terminalInput';
import { ConfirmDialog, ContextDrawer, Tooltip } from '../design-system';
import { TaskPreparationDialog } from '../components/TaskPreparationDialog';
import type { RuntimeSession } from '../types';
import { consumePtyOutputForChrome, extractOscTitle } from '../workspace/ptyLiveChrome';
import { usePtyLiveChromeOptional } from '../workspace/PtyLiveChromeContext';
import { canFitTerminal, canRunTerminalFrame } from './terminalFitModel';
import { VoiceCaptureButton } from '../features/voice/VoiceCaptureButton';
import { sendVoiceTranscriptToTerminal } from '../features/voice/terminalVoice';
import { ExecutionRoleSelector } from './ExecutionRoleSelector';
import styles from './AgentTerminal.module.scss';

interface AgentTerminalHostElement extends HTMLDivElement {
  __triggerReconnect?: () => void;
  __takeControl?: () => void;
}

export interface AgentTerminalSkill {
  id: string;
  name?: string;
  description?: string;
  category?: string;
  risk?: string;
  triggers?: string[];
  aliases?: string[];
}

export const AgentTerminal: React.FC<{
  agentId: string;
  projectId?: string;
  runtimeId?: string;
  /** Authoritative generation from the workspace surface (wins over poll). */
  preferredRuntimeId?: string;
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
  onBoundRuntimeChange?: (runtimeId: string) => void;
}> = ({
  agentId,
  projectId,
  runtimeId,
  preferredRuntimeId,
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
  onBoundRuntimeChange,
}) => {
  const { t } = useTranslation();
  const tRef = useRef(t);
  tRef.current = t;
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [connection, setConnection] = useState<
    'CONNECTING' | 'CONNECTED' | 'DISCONNECTED' | 'ERROR'
  >('CONNECTING');
  const [message, setMessage] = useState('');
  const [diagnosticCode, setDiagnosticCode] = useState('');
  const [role, setRole] = useState<TerminalRole>('VIEW_ONLY');
  const [ptyTitle, setPtyTitle] = useState('');
  const [recovering, setRecovering] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [boundRuntimeId, setBoundRuntimeId] = useState(runtimeId || '');
  const [connectNonce, setConnectNonce] = useState(0);
  const [askOpen, setAskOpen] = useState(false);
  const [preparationOpen, setPreparationOpen] = useState(false);
  const [skillsSearch, setSkillsSearch] = useState('');
  const [selectedSkillCategory, setSelectedSkillCategory] = useState('all');
  const [copiedSkillId, setCopiedSkillId] = useState<string | null>(null);
  const [askPrompt, setAskPrompt] = useState('');
  const [availableSkills, setAvailableSkills] = useState<AgentTerminalSkill[]>([]);
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
  const [promotionPrompt, setPromotionPrompt] = useState('');
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [closing, setClosing] = useState(false);
  const [needsResourceSelection, setNeedsResourceSelection] = useState(false);
  const [connectingForMs, setConnectingForMs] = useState(0);
  const liveChrome = usePtyLiveChromeOptional();
  const liveTitleKeyRef = useRef(liveTitleKey);
  liveTitleKeyRef.current = liveTitleKey;
  const liveChromeRef = useRef(liveChrome);
  liveChromeRef.current = liveChrome;
  const onBoundRuntimeChangeRef = useRef(onBoundRuntimeChange);
  onBoundRuntimeChangeRef.current = onBoundRuntimeChange;
  const ptyChromeRef = useRef({ title: '', questionnaire: false });
  const [fontSize, setFontSize] = useState<number>(() => getStoredTerminalFontSize());
  const initialFontSizeRef = useRef(fontSize);
  const [isScrolledUp, setIsScrolledUp] = useState(false);
  const fitAddonRef = useRef<FitAddon | null>(null);

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
    setBoundRuntimeId((current) => nextBoundRuntimeId(current, runtimeId, preferredRuntimeId));
  }, [runtimeId, preferredRuntimeId]);

  useEffect(() => {
    const key = liveTitleKey;
    const chrome = liveChromeRef.current;
    return () => {
      if (key) chrome?.clearLive(key);
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
    if (trimmed) {
      setBoundRuntimeId(trimmed);
      onBoundRuntimeChangeRef.current?.(trimmed);
    }
    setNeedsResourceSelection(false);
    setMessage('');
    setDiagnosticCode('');
    setConnection('CONNECTING');
    // Always bump nonce — parent may already have synced the same runtime_id,
    // so boundRuntimeId alone would not remount the WebSocket effect.
    setConnectNonce((n) => n + 1);
  };

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    let disposed = false;
    let terminalTraceEnabled = false;
    let reconnectAttempt = 0;
    let reconnectTimer: number | undefined;
    let connectTimeout: number | undefined;
    let openedOnce = false;
    let stopReconnect = false;
    const roleRef: { current: TerminalRole } = { current: 'VIEW_ONLY' };
    const kickoffRef = { current: normalizeInitialPrompt(initialPrompt), sent: false };
    // Buffers split control-RPC lines across output messages (see ProtocolScrubber).
    const protocolScrubber = new ProtocolScrubber();
    let lastError = '';
    let lastFailureCode = '';
    let termInstance: Terminal | null = null;
    const redrawTimers: number[] = [];
    let openFrame: number | undefined;

    (container as AgentTerminalHostElement).__triggerReconnect = () => {
      stopReconnect = false;
      reconnectAttempt = 0;
      lastError = '';
      if (reconnectTimer !== undefined) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = undefined;
      }
      connect();
    };

    (container as AgentTerminalHostElement).__takeControl = () => {
      const ws = wsRef.current;
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'lease_acquire' }));
      }
      termInstance?.focus();
    };

    const term = new Terminal({
      cursorBlink: true,
      fontSize: initialFontSizeRef.current,
      lineHeight: 1.25,
      fontFamily: 'var(--nx-font-mono)',
      theme: getTerminalTheme(),
      scrollback: DEFAULT_TERMINAL_SCROLLBACK,
    });
    termInstance = term;
    termRef.current = term;

    term.attachCustomKeyEventHandler(handleTerminalZoomKey);

    const scrollListener = term.onScroll(() => {
      if (disposed) return;
      const buf = term.buffer.active;
      const atBottom = shouldAutoScrollToBottom(buf.viewportY, buf.baseY);
      setIsScrolledUp(!atBottom);
    });

    const fit = new FitAddon();
    fitAddonRef.current = fit;
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

    const panel = container.closest<HTMLElement>('.nx-workspace-panel');
    const fitAndResize = (force = false) => {
      if (panel && panel.dataset.active !== 'true') return;
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

    // Do not let xterm render while its workspace panel is hidden. Its
    // viewport dimensions are not guaranteed during that transition.
    let pendingOutput = '';
    let outputFrame: number | undefined;
    const flushTerminalOutput = () => {
      outputFrame = undefined;
      if (disposed || (panel && panel.dataset.active !== 'true') || !pendingOutput) return;

      const output = pendingOutput;
      pendingOutput = '';
      const buf = term.buffer.active;
      const wasTracking = shouldAutoScrollToBottom(buf.viewportY, buf.baseY);
      try {
        term.write(output, () => {
          if (wasTracking) {
            term.scrollToBottom();
          }
        });
      } catch {
        pendingOutput = output + pendingOutput;
        if (outputFrame === undefined) {
          outputFrame = window.requestAnimationFrame(flushTerminalOutput);
        }
      }
    };
    const writeTerminalOutput = (data: string) => {
      if (disposed || !data) return;
      pendingOutput += data;
      if (panel && panel.dataset.active !== 'true') return;
      if (outputFrame === undefined) {
        outputFrame = window.requestAnimationFrame(flushTerminalOutput);
      }
    };
    const flushPendingOutput = () => {
      if (disposed || !pendingOutput || (panel && panel.dataset.active !== 'true')) return;
      if (outputFrame === undefined) {
        outputFrame = window.requestAnimationFrame(flushTerminalOutput);
      }
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

    const failPermanently = (detail?: string, attachCode?: string) => {
      stopReconnect = true;
      const diagnosticCode = attachCode || lastFailureCode || 'TERMINAL_ATTACH_FAILED';
      setDiagnosticCode(diagnosticCode);
      console.warn(
        '[nexus:terminal]',
        JSON.stringify({
          event: 'terminal.attach.failed',
          phase: 'TRANSPORT',
          code: diagnosticCode,
          agentId,
          runtimeId: boundRuntimeId,
          reconnectAttempts: reconnectAttempt,
        }),
      );
      if (reconnectTimer !== undefined) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = undefined;
      }
      setRecovering(false);
      setConnection('ERROR');
      const failure = terminalAttachFailure(detail, attachCode);
      if (failure.kind === 'generic') {
        setMessage(tRef.current('terminal.attachFailedGeneric'));
      } else if (failure.kind === 'runtime_incompatible') {
        setMessage(tRef.current('terminal.runtimeAttachIncompatible'));
      } else if (failure.kind === 'fatal_with_detail') {
        setMessage(`${failure.detail} ${tRef.current('terminal.attachFailedRecoverHint')}`);
      } else {
        setMessage(failure.detail);
      }
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
        failPermanently(detail || tRef.current('terminal.connectionLost'));
        return;
      }
      const delay = terminalReconnectDelay(reconnectAttempt++);
      setConnection('CONNECTING');
      setMessage(tRef.current('terminal.reconnecting', { delay }));
      reconnectTimer = window.setTimeout(() => {
        reconnectTimer = undefined;
        connect();
      }, delay);
    };

    const connect = async () => {
      if (disposed || stopReconnect) return;
      setConnection('CONNECTING');
      if (!openedOnce) setMessage('');
      lastError = '';
      lastFailureCode = '';
      term.options.disableStdin = true;

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

      let wsProto = window.location.protocol;
      let wsHost = window.location.host;
      const desktopBase = getDesktopWebSocketBaseUrl();
      if (desktopBase) {
        try {
          const parsed = new URL(desktopBase);
          wsProto = parsed.protocol;
          wsHost = parsed.host;
        } catch {
          // ignore
        }
      }
      const token = getDesktopAuthToken();

      const ws = new WebSocket(
        agentTerminalWebSocketURL(wsProto, wsHost, agentId, boundRuntimeId, token),
      );
      wsRef.current = ws;
      connectTimeout = window.setTimeout(() => {
        if (disposed || stopReconnect || ws.readyState !== WebSocket.CONNECTING) return;
        lastFailureCode = 'TERMINAL_CONNECT_TIMEOUT';
        lastError = tRef.current('terminal.connectTimeout', {
          defaultValue: 'Tempo limite excedido ao conectar o terminal.',
        });
        setDiagnosticCode(lastFailureCode);
        console.warn(
          '[nexus:terminal]',
          JSON.stringify({
            event: 'terminal.connect.timeout',
            phase: 'TRANSPORT',
            code: lastFailureCode,
            agentId,
            runtimeId: boundRuntimeId,
            reconnectAttempt,
            timeoutMs: TERMINAL_CONNECT_TIMEOUT_MS,
          }),
        );
        ws.close();
      }, TERMINAL_CONNECT_TIMEOUT_MS);
      roleRef.current = 'VIEW_ONLY';
      setRole('VIEW_ONLY');

      ws.onopen = () => {
        if (connectTimeout !== undefined) window.clearTimeout(connectTimeout);
        connectTimeout = undefined;
        term.options.disableStdin = false;
        lastFailureCode = '';
        setDiagnosticCode('');
        openedOnce = true;
        setRecovering(false);
        setConnection('CONNECTED');
        setModeAction((current) => (current === 'Restarting' ? 'Ready' : current));
        setMessage((current) =>
          current === tRef.current('terminal.restartingRuntime')
            ? tRef.current('common.ready', { defaultValue: 'Ready' })
            : '',
        );
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'lease_acquire' }));
        }
        openFrame = window.requestAnimationFrame(() => {
          openFrame = undefined;
          if (!canRunTerminalFrame({ disposed, socketOpen: ws.readyState === WebSocket.OPEN })) {
            return;
          }
          fitAndResize(true);
          flushPendingOutput();
          if (disposed) return;
          if (!panel || panel.dataset.active === 'true') term.focus();
          scheduleRedrawPulse();
        });
      };

      ws.onmessage = (event) => {
        if (disposed) return;
        try {
          const payload = JSON.parse(event.data);
          if (payload.type === 'trace_config') {
            terminalTraceEnabled = payload.data === 'enabled';
          } else if (payload.type === 'output' && payload.data) {
            const cleaned = protocolScrubber.scrub(String(payload.data));
            if (cleaned) {
              ingestOutput(cleaned);
              writeTerminalOutput(cleaned);
            }
          } else if (payload.type === 'lease') {
            const next = normalizeTerminalRole(payload.role);
            reconnectAttempt = 0;
            roleRef.current = next;
            setRole(next);
            setMessage(next === 'CONTROL' ? '' : tRef.current('terminal.readOnlyNotice'));
            if (next === 'CONTROL' && (!panel || panel.dataset.active === 'true')) {
              window.requestAnimationFrame(() => term.focus());
            }
            maybeSendKickoff();
          } else if (payload.type === 'runtime_changed') {
            const nextId = parseRuntimeChangedId(payload.data);
            setMessage(tRef.current('terminal.rebinding'));
            // Prevent reconnect on the old generation; remount against the new id.
            stopReconnect = true;
            if (reconnectTimer !== undefined) {
              window.clearTimeout(reconnectTimer);
              reconnectTimer = undefined;
            }
            try {
              ws.close(1012, 'runtime generation changed');
            } catch {
              // ignore
            }
            if (nextId) {
              setBoundRuntimeId(nextId);
              onBoundRuntimeChangeRef.current?.(nextId);
              setNeedsResourceSelection(false);
              setConnection('CONNECTING');
              setConnectNonce((n) => n + 1);
            } else {
              stopReconnect = false;
              scheduleReconnect(tRef.current('terminal.rebinding'));
            }
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
          } else if (payload.type === 'promotion') {
            const promotionRole = String(payload.role || '').toUpperCase();
            if (promotionRole === 'ASK') {
              setPromotionPrompt(String(payload.data || ''));
            } else if (promotionRole === 'PROMOTED') {
              setMessage(String(payload.data || tRef.current('terminal.promotion.started')));
            } else if (promotionRole === 'FALLBACK' || promotionRole === 'BLOCKED') {
              setMessage(String(payload.data || 'Mission admission did not start.'));
            }
          } else if (payload.type === 'error') {
            const detail = String(payload.data ?? 'Terminal error');
            const code = typeof payload.code === 'string' ? payload.code : '';
            if (code) lastFailureCode = code;
            lastError = detail;
            if (code === 'RUNTIME_ATTACH_INCOMPATIBLE') {
              failPermanently(detail, code);
              ws.close(1011, 'runtime attach incompatible');
            } else {
              setMessage(detail);
            }
            if (code !== 'RUNTIME_ATTACH_INCOMPATIBLE' && isFatalTerminalAttachError(detail)) {
              // A dead transport is not proof that the runtime should be
              // relaunched. Recovery is only performed by the explicit
              // Recover/Start action in the overlay.
              failPermanently(detail);
              ws.close(1011, 'fatal attach error');
            }
          }
        } catch {
          const raw = String(event.data);
          ingestOutput(raw);
          writeTerminalOutput(raw);
        }
      };

      ws.onerror = () => {
        if (!disposed && !stopReconnect) {
          if (!lastFailureCode) lastFailureCode = 'TERMINAL_TRANSPORT_ERROR';
          setDiagnosticCode(lastFailureCode);
          setMessage(tRef.current('terminal.transportError'));
        }
      };

      ws.onclose = () => {
        if (connectTimeout !== undefined) window.clearTimeout(connectTimeout);
        connectTimeout = undefined;
        term.options.disableStdin = true;
        if (wsRef.current === ws) wsRef.current = null;
        if (disposed || stopReconnect) return;
        // Release any buffered split-line tail: the stream ended here and a
        // reconnect starts a fresh scrubber for replayed history.
        const rest = protocolScrubber.flush();
        if (rest) {
          ingestOutput(rest);
          writeTerminalOutput(rest);
        }
        scheduleReconnect(lastError);
      };
    };

    const dataDisposable = term.onData((data) => {
      term.scrollToBottom();
      setIsScrolledUp(false);
      const ws = wsRef.current;
      if (!ws || ws.readyState !== WebSocket.OPEN) return;
      if (roleRef.current !== 'CONTROL') {
        // Opportunistically reclaim the writer seat, then still drop this
        // keystroke (broker would ignore it). Next keystrokes work after lease.
        ws.send(JSON.stringify({ type: 'lease_acquire' }));
        return;
      }
      if (terminalTraceEnabled) {
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
      if (!shouldForwardTerminalInput(data, true)) return;
      const clean = stripEmbeddedTerminalMetadata(data, true);
      if (!clean) return;
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
    });

    let flushFrame: number | undefined;
    const panelObserver = panel
      ? new MutationObserver(() => {
          if (panel.dataset.active !== 'true' || flushFrame !== undefined) return;
          flushFrame = window.requestAnimationFrame(() => {
            flushFrame = undefined;
            fitAndResize(true);
            flushPendingOutput();
            term.scrollToBottom();
          });
        })
      : null;
    if (panel)
      panelObserver?.observe(panel, { attributes: true, attributeFilter: ['data-active'] });
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
      if (connectTimeout !== undefined) window.clearTimeout(connectTimeout);
      if (openFrame !== undefined) window.cancelAnimationFrame(openFrame);
      if (flushFrame !== undefined) window.cancelAnimationFrame(flushFrame);
      if (outputFrame !== undefined) window.cancelAnimationFrame(outputFrame);
      redrawTimers.forEach((timer) => window.clearTimeout(timer));
      observer.disconnect();
      panelObserver?.disconnect();
      document.removeEventListener('visibilitychange', visibility);
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
    setMessage(t('terminal.startingRuntime'));
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
        setMessage(t('terminal.selectProvider'));
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
    setMessage(t('terminal.startingWithResource'));
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
    setMessage(t('terminal.removingAgent'));
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

  const handleVoiceTranscript = (text: string) => {
    sendVoiceTranscriptToTerminal(wsRef.current, role, text);
    termRef.current?.focus();
  };

  const handleInsertSkillToTerminal = (skill: AgentTerminalSkill) => {
    const trigger = skill.triggers?.[0] ? `${skill.triggers[0]} ` : `/${skill.id} `;
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'input', data: trigger }));
      setAskOpen(false);
      termRef.current?.focus();
    }
  };

  const handleCopySkill = (skill: AgentTerminalSkill) => {
    const text = skill.triggers?.[0] || `/${skill.id}`;
    try {
      const pending = navigator.clipboard.writeText(text);
      pending.catch(() => setAskFeedback(t('terminal.copyFailed')));
    } catch {
      setAskFeedback(t('terminal.copyFailed'));
    }
    setCopiedSkillId(skill.id);
    setTimeout(() => setCopiedSkillId(null), 1500);
  };

  const skillCategories = useMemo(() => {
    const cats = new Set<string>();
    availableSkills.forEach((s) => {
      if (s.category) cats.add(s.category);
    });
    return ['all', ...Array.from(cats)];
  }, [availableSkills]);

  const filteredSkills = useMemo(() => {
    const q = skillsSearch.trim().toLowerCase();
    return availableSkills.filter((s) => {
      const matchCat =
        selectedSkillCategory === 'all' ||
        (s.category && s.category.toLowerCase() === selectedSkillCategory.toLowerCase());
      if (!matchCat) return false;
      if (!q) return true;
      const name = (s.name || s.id).toLowerCase();
      const desc = (s.description || '').toLowerCase();
      const trigs = (s.triggers || []).join(' ').toLowerCase();
      return name.includes(q) || desc.includes(q) || trigs.includes(q);
    });
  }, [availableSkills, skillsSearch, selectedSkillCategory]);

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
      setAskFeedback(t('terminal.promptSent'));
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
    setMessage(t('terminal.savingConfig'));
    try {
      const nextRuntime = await onRestartWithMode(nextMode);
      setModeAction('Restarting');
      setMessage(t('terminal.restartingRuntime'));
      rebindTerminal(nextRuntime?.runtime_id);
      setSelectedMode(nextMode);
    } catch (error) {
      setModeAction('Error');
      setMessage(error instanceof Error ? error.message : String(error));
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
  const showConnectingStatus =
    connection === 'CONNECTING' && !showRecoverOverlay && connectingForMs >= 800;
  const windowChrome = chrome === 'window';

  const takeControl = () => {
    const host = containerRef.current as AgentTerminalHostElement | null;
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
    <div className="nx-agent-terminal__modes" role="group" aria-label={t('terminal.modeAriaLabel')}>
      {(['Safe', 'YOLO'] as const).map((m) => (
        <button
          key={m}
          type="button"
          className="nx-agent-terminal__mode"
          data-active={selectedMode === m ? 'true' : 'false'}
          onClick={() => requestModeChange(m)}
          disabled={
            !onRestartWithMode ||
            connection === 'CONNECTING' ||
            recovering ||
            modeAction === 'Applying' ||
            modeAction === 'Restarting'
          }
          title={t('terminal.switchMode', { mode: m })}
        >
          {m}
        </button>
      ))}
    </div>
  );

  const terminalActions = (
    <div className="nx-agent-terminal__controls">
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

      {role === 'CONTROL' ? null : (
        <>
          <span className="nx-agent-terminal__lease" data-role="VIEW_ONLY">
            {t('agents.viewOnlyLabel')}
          </span>
          <Tooltip content={t('terminal.takeControlTooltip')}>
            <button
              type="button"
              className="nx-agent-terminal__ask-btn"
              disabled={connection !== 'CONNECTED'}
              onClick={takeControl}
            >
              {t('terminal.takeControl')}
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
          {connection === 'CONNECTING' ? (
            <RefreshCw size={12} className="nx-spin-slow" />
          ) : modeAction === 'Error' ? (
            <ShieldAlert size={12} />
          ) : (
            <Unplug size={12} />
          )}
          {message}
        </span>
      )}
      {showConnectingStatus && !message && (
        <span className="nx-terminal-message" role="status">
          <RefreshCw size={12} className="nx-spin-slow" />
          {t('terminal.connectingTransport')}
        </span>
      )}
      <VoiceCaptureButton
        context="terminal"
        disabled={role !== 'CONTROL' || connection !== 'CONNECTED'}
        onTranscript={handleVoiceTranscript}
      />
      <Tooltip content={t('terminal.skillsTooltip')}>
        <button
          type="button"
          className="nx-agent-terminal__ask-btn"
          onClick={() => {
            const next = !askOpen;
            setPreparationOpen(next);
            if (next && availableSkills.length === 0) {
              void loadSkills();
            }
          }}
        >
          <Sparkles size={13} />
          <span>{t('terminal.prepareInstruction', 'Preparar instrução para o Agente')}</span>
          {availableSkills.length > 0 && (
            <span className={styles.skillsCountBadge}>{availableSkills.length}</span>
          )}
        </button>
      </Tooltip>
      {onClose && !windowChrome && (
        <Tooltip content={t('terminal.closeTerminalTooltip')}>
          <button type="button" onClick={() => setCloseConfirmOpen(true)}>
            {t('terminal.closeTerminal')}
          </button>
        </Tooltip>
      )}
    </div>
  );

  return (
    <section
      className={`nx-agent-terminal ${styles.terminalRoot}`}
      data-chrome={chrome}
      aria-label={t('terminal.terminalForAgent', { agentId })}
      onPointerDown={(e) => {
        const target = e.target as HTMLElement | null;
        if (
          !target?.closest('button, input, select, textarea, [role="button"], [role="menuitem"]')
        ) {
          termRef.current?.focus();
        }
      }}
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
            <ExecutionRoleSelector
              runtimeId={boundRuntimeId}
              provider={provider}
              disabled={!boundRuntimeId || connection === 'CONNECTING'}
            />
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
          <ExecutionRoleSelector
            runtimeId={boundRuntimeId}
            provider={provider}
            disabled={!boundRuntimeId || connection === 'CONNECTING'}
          />
          {modeButtons}
          {terminalActions}
        </div>
      )}
      <ContextDrawer
        open={askOpen}
        onClose={() => setAskOpen(false)}
        title={t('terminal.skillsDrawerTitle', 'Skills do Maestro')}
        description={t(
          'terminal.skillsDrawerDesc',
          'Catálogo de capacidades e automações integradas',
        )}
        width={420}
      >
        <div className={styles.skillsDrawerBody}>
          <div className={styles.skillsSearchBar}>
            <Search size={14} className="nx-text-muted" />
            <input
              type="text"
              placeholder={t(
                'terminal.skillsSearchPlaceholder',
                'Buscar por nome, comando ou descrição...',
              )}
              value={skillsSearch}
              onChange={(e) => setSkillsSearch(e.target.value)}
            />
            {skillsSearch && (
              <button
                type="button"
                className={styles.closeButton}
                onClick={() => setSkillsSearch('')}
                title={t('common.clear', 'Limpar')}
              >
                <X size={12} />
              </button>
            )}
          </div>

          {skillCategories.length > 1 && (
            <div className={styles.skillsCategoryBar}>
              {skillCategories.map((cat) => (
                <button
                  key={cat}
                  type="button"
                  className={styles.categoryChip}
                  data-active={selectedSkillCategory === cat ? 'true' : 'false'}
                  onClick={() => setSelectedSkillCategory(cat)}
                >
                  {cat === 'all' ? t('common.all', 'Todas') : cat}
                </button>
              ))}
            </div>
          )}

          <div className={styles.skillsList}>
            {filteredSkills.length === 0 ? (
              <div className="nx-text-muted nx-p-4 nx-text-center nx-text-sm">
                {availableSkills.length === 0
                  ? t('terminal.skillsLoading', 'Carregando catálogo dinâmico de skills...')
                  : t('terminal.skillsEmpty', 'Nenhuma skill encontrada para o filtro atual.')}
              </div>
            ) : (
              filteredSkills.map((skill) => {
                const active = selectedSkills.includes(skill.id);
                const isCopied = copiedSkillId === skill.id;
                return (
                  <div key={skill.id} className={styles.skillCard}>
                    <div className={styles.skillCardHeader}>
                      <span className={styles.skillCardName}>
                        <Sparkles size={12} className="nx-text-accent" />
                        {skill.name || skill.id}
                      </span>
                      <div className={styles.skillBadges}>
                        {skill.category && (
                          <span className={styles.skillCategoryBadge}>{skill.category}</span>
                        )}
                        <span className={styles.skillRiskBadge} data-risk={skill.risk || 'safe'}>
                          {skill.risk || 'safe'}
                        </span>
                      </div>
                    </div>
                    {skill.description && (
                      <p className={styles.skillCardDesc}>{skill.description}</p>
                    )}
                    {skill.triggers && skill.triggers.length > 0 && (
                      <div className={styles.skillTriggers}>
                        {skill.triggers.map((trig) => (
                          <span key={trig} className={styles.skillTriggerTag}>
                            {trig}
                          </span>
                        ))}
                      </div>
                    )}
                    <div className={styles.skillCardActions}>
                      <button
                        type="button"
                        className={styles.skillActionBtn}
                        onClick={() => handleCopySkill(skill)}
                        title={t('terminal.copySkillTitle', 'Copiar comando')}
                      >
                        {isCopied ? (
                          <Check size={11} className="text-[var(--nx-success)]" />
                        ) : (
                          <Copy size={11} />
                        )}
                        <span>
                          {isCopied
                            ? t('terminal.copied', 'Copiado')
                            : t('terminal.copy', 'Copiar')}
                        </span>
                      </button>
                      <button
                        type="button"
                        className={styles.skillActionBtn}
                        data-primary="true"
                        onClick={() => handleInsertSkillToTerminal(skill)}
                        title={t('terminal.insertSkillTitle', 'Inserir comando no terminal')}
                      >
                        <CornerDownLeft size={11} />
                        <span>{t('terminal.insert', 'Inserir')}</span>
                      </button>
                      <button
                        type="button"
                        className={styles.skillActionBtn}
                        onClick={() => handleToggleSkill(skill.id)}
                        data-active={active ? 'true' : 'false'}
                        title={t('terminal.selectForPrompt', 'Vincular para prompt (máx 3)')}
                      >
                        {active ? '✓ ' + t('terminal.selected', 'Prompt') : '+ Prompt'}
                      </button>
                    </div>
                  </div>
                );
              })
            )}
          </div>

          <div className={styles.skillsDrawerFooter}>
            <div className={styles.composerHeader}>
              <span className={styles.composerTitle}>
                <MessageSquare size={13} /> {t('terminal.sendInstructionTitle')}
              </span>
              {selectedSkills.length > 0 && (
                <span className={styles.skillsLabel}>
                  {selectedSkills.length}/3 {t('terminal.skillsSelected', 'selecionadas')}
                </span>
              )}
            </div>
            <div className={styles.composerRow}>
              <input
                type="text"
                className={`nx-input ${styles.composerInput}`}
                placeholder={t('terminal.sendInstructionPlaceholder')}
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
                <span>{asking ? t('terminal.sending') : t('terminal.send')}</span>
              </button>
            </div>
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
        </div>
      </ContextDrawer>
      <TaskPreparationDialog
        open={preparationOpen}
        agentId={agentId}
        agentName={agentName || displayName}
        projectId={projectId}
        initialPrompt={initialPrompt}
        startIfNeeded={false}
        onClose={() => setPreparationOpen(false)}
        onSubmit={async (prompt, skills, _startIfNeeded, contextFingerprintId) => {
          await nexus.askAgent(
            agentId,
            prompt,
            false,
            skills,
            projectId && contextFingerprintId ? { projectId, contextFingerprintId } : undefined,
          );
        }}
      />
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
      <TerminalActionDialog
        promotion={promotionPrompt || undefined}
        onCancel={() => setPromotionPrompt('')}
        onPromotionAnswer={(answer) => {
          const socket = wsRef.current;
          if (socket?.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({ type: 'promotion_answer', data: answer }));
          }
          setPromotionPrompt('');
        }}
      />
      <div
        ref={containerRef}
        className="nx-agent-terminal__xterm"
        onPointerDown={() => termRef.current?.focus()}
      />

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

      {showRecoverOverlay && (
        <div className={styles.recoverOverlay} data-window-chrome={windowChrome ? 'true' : 'false'}>
          <div className={styles.recoverHeader}>
            {recovering ? <RefreshCw size={18} className="nx-spin-slow" /> : <Unplug size={18} />}
            <strong className={styles.recoverTitle}>
              {recovering
                ? t('terminal.recoverTitleReconnecting')
                : t('terminal.recoverTitleStopped')}
            </strong>
          </div>
          <p className={styles.recoverText}>
            {recovering
              ? t('terminal.recoverNoticeReconnecting')
              : t('terminal.recoverNoticeStopped')}
          </p>
          {message && connection === 'ERROR' && <p className={styles.errorMessage}>{message}</p>}
          {diagnosticCode && connection === 'ERROR' && (
            <details className={styles.errorMessage}>
              <summary>
                {t('terminal.diagnosticDetails', { defaultValue: 'Detalhes técnicos' })}
              </summary>
              <code>{diagnosticCode}</code>
              <p>
                {t('terminal.runtimeId', { defaultValue: 'Runtime' })}: {boundRuntimeId || '—'}
              </p>
            </details>
          )}
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
              <span>{recovering ? t('terminal.starting') : t('terminal.startOrRecoverAgent')}</span>
            </button>
            <button
              type="button"
              className={`nx-button ${styles.actionButtonContent}`}
              title={t('terminal.reconnectWsTooltip')}
              disabled={recovering || connection === 'CONNECTING'}
              onClick={() => {
                setDiagnosticCode('');
                setMessage(t('terminal.reconnectingTransport'));
                setConnection('CONNECTING');
                if (
                  containerRef.current &&
                  (containerRef.current as AgentTerminalHostElement).__triggerReconnect
                ) {
                  (containerRef.current as AgentTerminalHostElement).__triggerReconnect?.();
                } else {
                  setConnectNonce((n) => n + 1);
                }
              }}
            >
              <RefreshCw size={13} />
              <span>{t('terminal.reconnectWs')}</span>
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
                <span>{deleting ? t('terminal.removing') : t('terminal.removeAgent')}</span>
              </button>
            )}
          </div>
        </div>
      )}
      <ConfirmDialog
        open={deleteConfirmOpen}
        title={t('agents.confirmRemoveTitle')}
        description={t('agents.confirmRemove', { name: displayName })}
        onConfirm={() => void executeDeleteAgent()}
        onCancel={() => setDeleteConfirmOpen(false)}
      />
    </section>
  );
};
