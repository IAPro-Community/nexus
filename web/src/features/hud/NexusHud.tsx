import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Activity,
  AlertTriangle,
  ChevronDown,
  ChevronUp,
  CircleHelp,
  Clock3,
  ExternalLink,
  GripHorizontal,
  RefreshCw,
  Settings2,
  ShieldAlert,
  X,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { nexus } from '../../nexus/api';
import { hudV3 } from '../../platform';
import type { HudSizePreset } from '../../platform';
import type { AttentionGroup, CapacityResourceStatus } from '../../types';
import {
  formatHudCost,
  isWorkingAgent,
  loadHudPreferences,
  resolveNativeHudStatus,
  saveHudPreferences,
  type HudPresentationPreferences,
  type HudResponse,
} from './hudModel';
import {
  hudOtherAccounts,
  hudPercentageFor,
  hudQuotaEvidence,
  hudRemaining,
  hudVisibleMission,
  selectHudFocus,
} from './hudMiniModel';
import { HudSizeControls } from './HudSizeControls';
import { NexusGuardian } from './NexusGuardian';
import {
  companionMood,
  nextCompanionBubble,
  projectCompanionAttention,
  readCompanionSnoozes,
  snoozeCompanionAttention,
  type CompanionAttention,
} from './companionModel';
import styles from './NexusHud.module.scss';

const MIN_REFRESH_MS = 15_000;
const FLOATING_KEYBOARD_STEP = 16;

export interface NexusHudProps {
  projectId?: string;
  native?: boolean;
  onOpenNexus?: () => void;
  onOpenUsage?: () => void;
  onOpenAgents?: () => void;
  onOpenMissions?: () => void;
  onOpenSettings?: () => void;
  onOpenTarget?: (item: CompanionAttention) => void;
  nativeAutoStart?: {
    supported: boolean;
    enabled: boolean;
    pending?: boolean;
    onToggle: () => void;
  };
}

const clampFloatingPosition = (x: number, y: number) => ({
  x: Math.max(8, Math.min(x, Math.max(8, window.innerWidth - 240))),
  y: Math.max(8, Math.min(y, Math.max(8, window.innerHeight - 64))),
});

export const NexusHud: React.FC<NexusHudProps> = ({
  projectId,
  native = false,
  onOpenNexus,
  onOpenUsage,
  onOpenAgents,
  onOpenMissions,
  onOpenSettings,
  onOpenTarget,
  nativeAutoStart,
}) => {
  const { t } = useTranslation();
  const [prefs, setPrefs] = useState<HudPresentationPreferences>(() => loadHudPreferences());
  const [response, setResponse] = useState<HudResponse | null>(null);
  const [expanded, setExpanded] = useState(false);
  const [offline, setOffline] = useState(false);
  const [lastSuccessAt, setLastSuccessAt] = useState<number | null>(null);
  const [paceWarning, setPaceWarning] = useState<CapacityResourceStatus | null>(null);
  const [attentionGroup, setAttentionGroup] = useState<AttentionGroup | undefined>();
  const [alertActionPending, setAlertActionPending] = useState(false);
  const requestInFlight = useRef<Promise<void> | null>(null);
  const dragState = useRef<{ pointerId: number; x: number; y: number } | null>(null);
  const panelRef = useRef<HTMLElement | null>(null);
  const nativeExpansionPending = useRef(false);
  const [nativeExpansionError, setNativeExpansionError] = useState<string | null>(null);
  const [sizePresets, setSizePresets] = useState<HudSizePreset[]>([]);
  const [sizePreset, setSizePreset] = useState('');
  const [sizePending, setSizePending] = useState(false);
  const [bubble, setBubble] = useState<CompanionAttention | undefined>();
  const announced = useRef(new Set<string>());
  const bubbleInFlight = useRef(false);
  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false);
  const [nativePresentationReady, setNativePresentationReady] = useState(!native);

  const refresh = useCallback(() => {
    if (requestInFlight.current) return requestInFlight.current;
    const task = nexus
      .getHUDState(projectId)
      .then((next) => {
        setResponse(next);
        setOffline(false);
        setLastSuccessAt(Date.now());
        // Presentation reads existing capacity state; it never probes providers.
        if (prefs.appearance === 'companion') {
          void nexus.getAttention().then(setAttentionGroup).catch(() => setAttentionGroup(undefined));
        }
        void nexus
          .getCapacityStatus()
          .then((capacity) => {
            const rising = (capacity.resources || [])
              .filter((item) => item.pace?.status === 'RISING' && item.pace?.exhausts_at)
              .sort(
                (a, b) =>
                  Date.parse(a.pace?.exhausts_at || '') - Date.parse(b.pace?.exhausts_at || ''),
              );
            setPaceWarning(rising[0] || null);
          })
          .catch(() => setPaceWarning(null));
      })
      .catch(() => setOffline(true))
      .finally(() => {
        requestInFlight.current = null;
      });
    requestInFlight.current = task;
    return task;
  }, [projectId, prefs.appearance]);

  useEffect(() => {
    void refresh();
    const onVisible = () => {
      if (document.visibilityState === 'visible') void refresh();
    };
    const onFocus = () => void refresh();
    document.addEventListener('visibilitychange', onVisible);
    window.addEventListener('focus', onFocus);
    const timer = window.setInterval(
      () => {
        if (document.visibilityState === 'visible') void refresh();
      },
      Math.max(MIN_REFRESH_MS, response?.state?.system.cache_ttl_millis || MIN_REFRESH_MS),
    );
    return () => {
      document.removeEventListener('visibilitychange', onVisible);
      window.removeEventListener('focus', onFocus);
      window.clearInterval(timer);
    };
  }, [refresh, response?.state?.system.cache_ttl_millis]);

  useEffect(() => {
    const sync = () => setPrefs(loadHudPreferences());
    window.addEventListener('nexus:hud-preferences', sync);
    return () => window.removeEventListener('nexus:hud-preferences', sync);
  }, []);

  useEffect(() => {
    if (!native) return;
    let active = true;
    void Promise.all([hudV3.getSizePresets(), hudV3.getWindowState()])
      .then(([presets, windowState]) => {
        if (!active) return;
        const availablePresets = Array.isArray(presets) ? presets : [];
        setSizePresets(availablePresets);
        const compactWidth = windowState.compactWidth ?? windowState.width;
        const compactHeight = windowState.compactHeight ?? windowState.height;
        const match = availablePresets.find(
          (preset) =>
            preset.compactWidth === compactWidth && preset.compactHeight === compactHeight,
        );
        if (match) setSizePreset(match.id);
      })
      .catch((error: unknown) => {
        console.error('Nexus HUD size presets unavailable:', error);
      });
    return () => {
      active = false;
    };
  }, [native]);

  useEffect(() => {
    if (!expanded) return;
    const onPointerDown = (event: PointerEvent) => {
      if (panelRef.current && !panelRef.current.contains(event.target as Node)) {
        setExpanded(false);
        if (native) void hudV3.setExpanded(false).catch(() => undefined);
      }
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && panelRef.current?.contains(document.activeElement)) {
        setExpanded(false);
        if (native) void hudV3.setExpanded(false).catch(() => undefined);
      }
    };
    document.addEventListener('pointerdown', onPointerDown);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('pointerdown', onPointerDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [expanded, native]);

  useEffect(() => {
    if (typeof window.matchMedia !== 'function') return;
    const media = window.matchMedia('(prefers-reduced-motion: reduce)');
    const sync = () => setPrefersReducedMotion(media.matches);
    sync();
    media.addEventListener('change', sync);
    return () => media.removeEventListener('change', sync);
  }, []);

  // Presentation switches are native geometry changes, not new monitoring
  // sessions. The bar preset is retained independently from the guardian.
  useEffect(() => {
    if (!native) return;
    let active = true;
    setNativePresentationReady(false);
    const preset = prefs.appearance === 'companion' ? 'companion' : prefs.barSizePreset;
    void hudV3.setSizePreset(preset, expanded).then(() => {
      if (!active) return;
      setSizePreset(preset);
      setNativePresentationReady(true);
    }).catch((error: unknown) => {
      console.error('Nexus Companion native presentation resize failed:', error);
      toast.error('Não foi possível alterar o modo do HUD nativo.');
    });
    return () => { active = false; };
  // Only switching presentation or a persisted bar size changes geometry.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [native, prefs.appearance, prefs.barSizePreset]);

  const state = response?.state;
  const focus = useMemo(() => (state ? selectHudFocus(state, offline) : null), [state, offline]);
  const otherAccounts = useMemo(
    () => (state ? hudOtherAccounts(state, focus?.provider) : []),
    [state, focus?.provider],
  );
  const cost = useMemo(() => formatHudCost(state?.cost), [state?.cost]);
  const costEvidenceLabel = {
    reported: 'Informado pelo provedor',
    estimated: 'Estimativa',
    'known-partial': 'Parcial',
    unknown: 'Desconhecido',
  } as const;
  const showDetails = (kind: HudPresentationPreferences['mode']) =>
    prefs.mode === 'AUTO' || prefs.mode === kind;
  const alert =
    state?.alerts.find(
      (item) =>
        item.provider?.toLowerCase() === focus?.provider?.provider.toLowerCase() &&
        (!item.profile || item.profile.toLowerCase() === focus?.provider?.profile.toLowerCase()),
    ) || state?.alerts[0];
  const activeAgents = state?.agents.filter(isWorkingAgent) || [];
  const currentMission = state ? hudVisibleMission(state.missions) : undefined;
  const activeMission = currentMission?.needs_user ? currentMission : undefined;
  const companionAttention = useMemo(
    () => projectCompanionAttention(state, Date.now(), attentionGroup),
    [state, attentionGroup],
  );
  const mood = companionMood(state, companionAttention, offline);
  const isAttention = Boolean(alert || activeMission);
  const displayTone = offline
    ? 'unknown'
    : activeMission || alert?.severity.toUpperCase() === 'WARNING'
      ? 'warning'
      : focus?.tone || (alert ? 'warning' : 'unknown');
  const displayPercent = hudPercentageFor(focus?.provider, offline);
  const quotaEvidence = hudQuotaEvidence(focus?.provider, offline);
  const evidenceCopy = {
    live: 'Ao vivo',
    cached: 'Em cache',
    estimated: 'Estimativa',
    'rate-limited': 'Limite temporário',
    unknown: 'Não confirmado',
  }[quotaEvidence];
  const quotaAccessibleText =
    focus?.remaining === null || focus?.remaining === undefined
      ? 'Capacidade não confirmada'
      : `${displayPercent} · ${evidenceCopy}`;
  const focusedName = focus?.provider?.provider || 'Nexus';
  const displayName =
    focusedName === 'Nexus'
      ? focusedName
      : focusedName.charAt(0).toUpperCase() + focusedName.slice(1);

  const clearBubble = useCallback((shrink = true) => {
    setBubble(undefined);
    if (native && shrink) {
      void hudV3.setBubbleVisible(false).catch((error: unknown) => {
        console.error('Nexus Companion bubble resize failed:', error);
      });
    }
  }, [native]);

  // Announce each incoming attention item once per session. Snoozes only
  // silence the floating speech: they never acknowledge canonical alerts.
  useEffect(() => {
    if (prefs.appearance !== 'companion' || !prefs.attentionBubbles || expanded || bubble || bubbleInFlight.current) return;
    if (native && !nativePresentationReady) return;
    if (typeof window === 'undefined') return;
    const pending = companionAttention.filter((item) => !announced.current.has(item.id));
    let snoozes: Record<string, number> = {};
    try { snoozes = readCompanionSnoozes(window.localStorage); } catch { /* Storage may be unavailable */ }
    const next = nextCompanionBubble(pending, snoozes);
    if (!next) return;
    announced.current.add(next.id);
    if (!native) { setBubble(next); return; }
    bubbleInFlight.current = true;
    void hudV3.setBubbleVisible(true).then(() => setBubble(next)).catch((error: unknown) => {
      console.error('Nexus Companion attention bubble unavailable:', error);
    }).finally(() => { bubbleInFlight.current = false; });
  }, [prefs.appearance, prefs.attentionBubbles, expanded, bubble, companionAttention, native, nativePresentationReady]);

  useEffect(() => {
    if (!bubble) return;
    const timer = window.setTimeout(() => clearBubble(), 8500);
    return () => window.clearTimeout(timer);
  }, [bubble, clearBubble]);

  const setExpandedState = (value: boolean) => {
    if (value) clearBubble(false);
    if (!native) {
      setExpanded(value);
      return;
    }
    if (nativeExpansionPending.current) return;
    nativeExpansionPending.current = true;
    // First resize the independent Wails window, THEN show the popover inside
    // its viewport. Never claim an expanded panel when the native bridge fails.
    void hudV3
      .setExpanded(value)
      .then(() => {
        setNativeExpansionError(null);
        setExpanded(value);
      })
      .catch((error: unknown) => {
        setNativeExpansionError('Falha ao expandir a janela nativa');
        console.error('Nexus HUD native window resize failed:', error);
        toast.error('Não foi possível abrir o painel do Nexus HUD.');
      })
      .finally(() => {
        nativeExpansionPending.current = false;
      });
  };

  const changeAppearance = (appearance: 'bar' | 'companion') => {
    if (appearance !== 'companion' && bubble) clearBubble(false);
    const next = { ...prefs, appearance };
    setPrefs(next);
    saveHudPreferences(next);
  };

  const snoozeBubble = (item: CompanionAttention) => {
    try { snoozeCompanionAttention(window.localStorage, item.id); } catch { /* private storage may be denied */ }
    clearBubble();
  };

  const openAttention = (item: CompanionAttention) => {
    if (onOpenTarget) {
      clearBubble();
      onOpenTarget(item);
      return;
    }
    const destination =
      item.target === 'missions' ? onOpenMissions :
      item.target === 'agents' ? onOpenAgents :
      item.target === 'usage' ? onOpenUsage : onOpenNexus;
    if (destination || onOpenNexus) {
      clearBubble();
      (destination || onOpenNexus)?.();
    } else {
      setExpandedState(true);
    }
  };

  const applySize = (preset: string) => {
    if (!native || sizePending || preset === 'companion') return;
    setSizePending(true);
    void hudV3
      .setSizePreset(preset, expanded)
      .then(() => {
        setSizePreset(preset);
        if (preset === 'compact' || preset === 'comfortable' || preset === 'large') {
          const next = { ...prefs, barSizePreset: preset };
          setPrefs(next);
          saveHudPreferences(next);
        }
        toast.success(t('hud.sizeUpdated'));
      })
      .catch((error: unknown) => {
        console.error('Nexus HUD size preset failed:', error);
        toast.error(t('hud.sizeUpdateFailed'));
      })
      .finally(() => setSizePending(false));
  };

  const onNativeDragKeys = (event: React.KeyboardEvent<HTMLButtonElement>) => {
    if (!native) return;
    const step = event.shiftKey ? 64 : 16;
    let dx = 0;
    let dy = 0;
    if (event.key === 'ArrowLeft') dx = -step;
    else if (event.key === 'ArrowRight') dx = step;
    else if (event.key === 'ArrowUp') dy = -step;
    else if (event.key === 'ArrowDown') dy = step;
    if (!dx && !dy) return;
    event.preventDefault();
    void hudV3.move(dx, dy).catch((error: unknown) => {
      console.error('Nexus HUD keyboard move failed:', error);
      toast.error(t('hud.moveFailed'));
    });
  };

  const handleAlert = async (action: 'ack' | 'snooze') => {
    if (!alert?.fingerprint || alertActionPending) return;
    setAlertActionPending(true);
    try {
      if (action === 'ack') await nexus.ackCapacityAlert(alert.fingerprint);
      else await nexus.snoozeCapacityAlert(alert.fingerprint);
      toast.success(action === 'ack' ? 'Alerta reconhecido' : 'Alerta adiado');
      await refresh();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error));
    } finally {
      setAlertActionPending(false);
    }
  };

  const updateFloatingPosition = (x: number, y: number, persist = false) => {
    const floatingPosition = clampFloatingPosition(x, y);
    const next = { ...prefs, floatingPosition };
    setPrefs(next);
    if (persist) saveHudPreferences(next);
  };

  const onDragDown = (event: React.PointerEvent<HTMLButtonElement>) => {
    if (prefs.surfaceMode !== 'floating') return;
    event.currentTarget.setPointerCapture(event.pointerId);
    dragState.current = {
      pointerId: event.pointerId,
      x: event.clientX - prefs.floatingPosition.x,
      y: event.clientY - prefs.floatingPosition.y,
    };
  };
  const onDragMove = (event: React.PointerEvent<HTMLButtonElement>) => {
    if (dragState.current?.pointerId !== event.pointerId) return;
    updateFloatingPosition(
      event.clientX - dragState.current.x,
      event.clientY - dragState.current.y,
    );
  };
  const onDragUp = (event: React.PointerEvent<HTMLButtonElement>) => {
    if (dragState.current?.pointerId !== event.pointerId) return;
    const drag = dragState.current;
    dragState.current = null;
    updateFloatingPosition(event.clientX - drag.x, event.clientY - drag.y, true);
  };
  const onDragKeys = (event: React.KeyboardEvent<HTMLButtonElement>) => {
    if (prefs.surfaceMode !== 'floating') return;
    const vector: Record<string, [number, number]> = {
      ArrowLeft: [-1, 0],
      ArrowRight: [1, 0],
      ArrowUp: [0, -1],
      ArrowDown: [0, 1],
    };
    const delta = vector[event.key];
    if (!delta) return;
    event.preventDefault();
    updateFloatingPosition(
      prefs.floatingPosition.x + delta[0] * FLOATING_KEYBOARD_STEP,
      prefs.floatingPosition.y + delta[1] * FLOATING_KEYBOARD_STEP,
      true,
    );
  };

  const nativeStatus = resolveNativeHudStatus(response?.enabled, Boolean(state), offline);
  if (!native && !prefs.enabled) return null;
  if (!native && (!response?.enabled || !state)) return null;

  const waiting = native && nativeStatus !== 'ready';
  const mainLabel = waiting
    ? nativeStatus === 'loading'
      ? t('common.loading')
      : nativeStatus === 'disabled'
        ? t('common.hudDisabled')
        : t('common.hudUnavailable')
    : displayName;
  const currentPercent = waiting ? '—' : displayPercent;
  const capsuleCapacityCopy =
    currentPercent === '—'
      ? 'capacidade não confirmada'
      : `${currentPercent} restante, ${evidenceCopy}`;
  const capsuleAccessibleLabel = `${expanded ? 'Recolher' : 'Expandir'} Nexus HUD. ${mainLabel}, ${capsuleCapacityCopy}`;
  const nativeTitle = `${mainLabel}: ${currentPercent} · ${focus?.provider?.limiting_window?.reset_description || 'Renovação desconhecida'}`;
  const tone = waiting ? 'unknown' : displayTone;
  const floatingStyle =
    !native && prefs.surfaceMode === 'floating'
      ? ({
          '--hud-floating-x': `${prefs.floatingPosition.x}px`,
          '--hud-floating-y': `${prefs.floatingPosition.y}px`,
        } as React.CSSProperties)
      : undefined;

  return (
    <aside
      className={styles.root}
      data-surface={native ? 'native' : prefs.surfaceMode}
      data-dock={prefs.dock}
      data-native={native ? 'true' : 'false'}
      style={floatingStyle}
      aria-label="Nexus HUD"
      aria-live="polite"
      data-appearance={prefs.appearance}
      data-bubble-visible={bubble ? 'true' : 'false'}
    >
      <section
        ref={panelRef}
        className={styles.panel}
        data-hud-panel
        data-expanded={expanded ? 'true' : 'false'}
        data-tone={tone}
        data-size={native ? sizePreset || 'custom' : undefined}
      >
        {expanded && state && !waiting && (
          <div id="nexus-hud-popover" className={styles.popover} aria-label="Resumo do Nexus HUD">
            <div className={styles.popoverHeader}>
              <div
                className={`${styles.brand} ${native ? styles.nativeBrandDrag : ''}`}
                data-hud-drag-handle={native ? 'true' : undefined}
                title={native ? t('hud.dragTitle') : undefined}
              >
                <img src="/nexus-icon.svg" alt="" width={25} height={25} />
                <strong>Nexus</strong>
                {native && (
                  <GripHorizontal size={14} className={styles.dragGrip} aria-hidden="true" />
                )}
              </div>
              <div className={styles.headerActions}>
                {onOpenSettings && (
                  <button
                    type="button"
                    className={styles.iconButton}
                    aria-label="Configurações do Nexus HUD"
                    onClick={onOpenSettings}
                  >
                    <Settings2 size={16} />
                  </button>
                )}
                <button
                  type="button"
                  className={styles.iconButton}
                  aria-label="Recolher Nexus HUD"
                  onClick={() => setExpandedState(false)}
                >
                  <X size={16} />
                </button>
              </div>
            </div>

            <div className={styles.sectionCaption}>
              {focus?.active ? 'Conta em execução' : 'Recurso em destaque'}
            </div>
            <div className={styles.accountCard} data-tone={tone} data-evidence={quotaEvidence}>
              <div className={styles.accountTop}>
                <span className={styles.accountTitle}>
                  <span className={styles.statusDot} />
                  {displayName}
                  {!prefs.privacyMode && focus?.provider?.profile
                    ? ` · ${focus.provider.profile}`
                    : ''}
                </span>
                <strong>{displayPercent}</strong>
              </div>
              <div
                className={styles.track}
                role="progressbar"
                aria-label="Capacidade restante da janela limitante"
                aria-valuenow={focus?.remaining ?? undefined}
                aria-valuetext={quotaAccessibleText}
                aria-valuemin={0}
                aria-valuemax={100}
              >
                {focus?.remaining !== null && focus?.remaining !== undefined && (
                  <div className={styles.fill} style={{ width: `${focus.remaining}%` }} />
                )}
              </div>
              <div className={styles.accountMeta}>
                <span>
                  Janela{' '}
                  {focus?.provider?.limiting_window?.label ||
                    focus?.provider?.limiting_window?.kind ||
                    'desconhecida'}
                </span>
                <span>
                  {focus?.provider?.limiting_window?.reset_description || 'Renovação não informada'}
                </span>
              </div>
              <div className={styles.evidence}>
                {focus?.provider?.source || 'Fonte não informada'} · {evidenceCopy}
                {focus?.provider?.last_updated
                  ? ` · ${new Intl.DateTimeFormat(undefined, {
                      dateStyle: 'short',
                      timeStyle: 'short',
                    }).format(new Date(focus.provider.last_updated))}`
                  : ''}
              </div>
            </div>

            {prefs.showCapacity && showDetails('CAPACITY') && otherAccounts.length > 0 && (
              <div className={styles.accounts}>
                <div className={styles.sectionCaption}>Outras contas</div>
                {otherAccounts.slice(0, 3).map((provider) => {
                  const remaining = hudRemaining(provider, offline);
                  return (
                    <div
                      className={styles.accountLine}
                      key={`${provider.provider}:${provider.profile}`}
                    >
                      <span className={styles.accountIdentity}>
                        <Activity size={14} aria-hidden="true" />
                        <span>
                          {provider.provider}
                          {prefs.privacyMode ? '' : ` · ${provider.profile}`}
                        </span>
                      </span>
                      <span className={styles.smallTrack}>
                        {remaining !== null && <span style={{ width: `${remaining}%` }} />}
                      </span>
                      <strong>{hudPercentageFor(provider, offline)}</strong>
                    </div>
                  );
                })}
                {otherAccounts.length > 3 && (
                  <span className={styles.evidence}>
                    +{otherAccounts.length - 3} contas em Usage
                  </span>
                )}
              </div>
            )}

            {companionAttention.length > 0 && (
              <div className={styles.companionPendings} aria-label="Pendências do Nexus">
                <strong>Pendências ({companionAttention.length})</strong>
                {companionAttention.slice(0, 5).map((item) => (
                  <button type="button" key={item.id} onClick={() => openAttention(item)}>
                    <span>{item.title}</span>
                    <ExternalLink size={13} aria-hidden="true" />
                  </button>
                ))}
              </div>
            )}
            {activeMission?.needs_user ? (
              <div className={styles.notice} data-tone="warning">
                <ShieldAlert size={18} aria-hidden="true" />
                <div>
                  <strong>Mission precisa de você</strong>
                  <span>{activeMission.intervention_reason || 'Intervenção necessária'}</span>
                </div>
                {onOpenMissions && (
                  <button type="button" className={styles.textLink} onClick={onOpenMissions}>
                    Abrir
                  </button>
                )}
              </div>
            ) : alert ? (
              <div className={styles.notice} data-tone="warning">
                <AlertTriangle size={18} aria-hidden="true" />
                <div>
                  <strong>Capacidade em atenção</strong>
                  <span>
                    {alert.provider ? `${alert.provider}: ` : ''}
                    {alert.reason || alert.severity}
                  </span>
                </div>
              </div>
            ) : tone === 'unknown' ? (
              <div className={styles.notice}>
                <CircleHelp size={18} aria-hidden="true" />
                <div>
                  <strong>Quota não confirmada</strong>
                  <span>Sem medição confiável; não significa quota esgotada.</span>
                </div>
              </div>
            ) : tone === 'critical' || tone === 'warning' ? (
              <div className={styles.notice} data-tone="warning">
                <AlertTriangle size={18} aria-hidden="true" />
                <div>
                  <strong>Capacidade com restrição</strong>
                  <span>
                    {focus?.provider?.rate_limited || quotaEvidence === 'rate-limited'
                      ? 'Provider com limite temporário; consulte Usage antes de continuar.'
                      : 'Verifique a janela e o estado de capacidade em Usage.'}
                  </span>
                </div>
              </div>
            ) : (
              <div className={styles.notice}>
                <Activity size={18} aria-hidden="true" />
                <div>
                  <strong>Operação normal</strong>
                  <span>Nenhuma intervenção de capacidade pendente.</span>
                </div>
              </div>
            )}

            {prefs.showRecommendation && showDetails('RESOURCE') && state.recommendation && (
              <div className={styles.recommendation}>
                <strong>Recomendação do ResourceScheduler</strong>
                <span>{state.recommendation.display_name || state.recommendation.provider}</span>
                <small>{state.recommendation.explanation || 'Recurso alternativo validado.'}</small>
                {(onOpenUsage || onOpenNexus) && (
                  <button type="button" className={styles.cta} onClick={onOpenUsage || onOpenNexus}>
                    Ver recurso <ExternalLink size={14} />
                  </button>
                )}
              </div>
            )}

            {prefs.showMissions &&
              showDetails('MISSION') &&
              currentMission &&
              !currentMission.needs_user && (
                <div className={styles.missionPreview}>
                  <div>
                    <strong>Mission · {currentMission.state}</strong>
                    <span>
                      {prefs.privacyMode
                        ? currentMission.current_stage || 'Execução em andamento'
                        : currentMission.current_task_title ||
                          currentMission.current_stage ||
                          'Execução em andamento'}
                    </span>
                    {currentMission.total_packages > 0 && (
                      <small>
                        {currentMission.completed_packages}/{currentMission.total_packages} packages
                        verificados
                      </small>
                    )}
                  </div>
                  {onOpenMissions && (
                    <button type="button" className={styles.textLink} onClick={onOpenMissions}>
                      Abrir
                    </button>
                  )}
                </div>
              )}

            {prefs.showCapacity && showDetails('CAPACITY') && paceWarning?.pace?.exhausts_at && (
              <p className={styles.evidence}>
                Tendência de esgotamento estimada:{' '}
                {new Intl.DateTimeFormat(undefined, {
                  dateStyle: 'short',
                  timeStyle: 'short',
                }).format(new Date(paceWarning.pace.exhausts_at))}
              </p>
            )}
            {prefs.showAgents && showDetails('AGENT') && activeAgents.length > 0 && (
              <button
                type="button"
                className={styles.quickLink}
                onClick={onOpenAgents}
                disabled={!onOpenAgents}
              >
                {activeAgents.length} agente(s) em execução <ChevronDown size={14} />
              </button>
            )}
            {prefs.showCost &&
              !prefs.privacyMode &&
              !offline &&
              prefs.mode === 'AUTO' &&
              cost.semantics !== 'unknown' && (
                <span className={styles.evidence}>
                  Custo: {cost.label} · {costEvidenceLabel[cost.semantics]}
                </span>
              )}
            {alert && (
              <div className={styles.footerActions}>
                <button
                  type="button"
                  onClick={() => void handleAlert('ack')}
                  disabled={alertActionPending}
                >
                  Reconhecer
                </button>
                <button
                  type="button"
                  onClick={() => void handleAlert('snooze')}
                  disabled={alertActionPending}
                >
                  Adiar
                </button>
              </div>
            )}
            {native && prefs.appearance === 'bar' && (
              <HudSizeControls
                presets={sizePresets.filter((preset) => preset.id !== 'companion')}
                active={sizePreset}
                pending={sizePending}
                onSelect={applySize}
              />
            )}
            <div className={styles.companionAppearance}>
              <span>Apresentação</span>
              <button type="button" aria-pressed={prefs.appearance === 'bar'} onClick={() => changeAppearance('bar')}>Barra</button>
              <button type="button" aria-pressed={prefs.appearance === 'companion'} onClick={() => changeAppearance('companion')}>Mascote 2D</button>
            </div>
            <div className={styles.footerActions}>
              {(onOpenUsage || onOpenNexus) && (
                <button type="button" onClick={onOpenUsage || onOpenNexus}>
                  Abrir Nexus <ExternalLink size={13} />
                </button>
              )}
              {native && nativeAutoStart?.supported && (
                <button
                  type="button"
                  disabled={nativeAutoStart.pending}
                  aria-pressed={nativeAutoStart.enabled}
                  onClick={nativeAutoStart.onToggle}
                >
                  Iniciar com sistema: {nativeAutoStart.enabled ? 'Sim' : 'Não'}
                </button>
              )}
            </div>
          </div>
        )}
        {expanded && (!state || waiting) && (
          <div
            id="nexus-hud-popover"
            className={styles.popover}
            aria-label="Diagnóstico do Nexus HUD"
          >
            <div className={styles.popoverHeader}>
              <div
                className={`${styles.brand} ${native ? styles.nativeBrandDrag : ''}`}
                data-hud-drag-handle={native ? 'true' : undefined}
                title={native ? t('hud.dragTitle') : undefined}
              >
                <img src="/nexus-icon.svg" alt="" width={25} height={25} />
                <strong>Nexus HUD</strong>
                {native && (
                  <GripHorizontal size={14} className={styles.dragGrip} aria-hidden="true" />
                )}
              </div>
              <button
                type="button"
                className={styles.iconButton}
                aria-label="Recolher Nexus HUD"
                onClick={() => setExpandedState(false)}
              >
                <X size={16} />
              </button>
            </div>
            <div className={styles.notice} data-tone="warning" role="status">
              <CircleHelp size={18} aria-hidden="true" />
              <div>
                <strong>
                  {nativeStatus === 'disabled'
                    ? 'Projeção do HUD desativada'
                    : 'Dados do HUD indisponíveis'}
                </strong>
                <span>
                  {nativeStatus === 'disabled'
                    ? 'O Core desativou a projeção. Verifique a configuração NEXUS_DISABLE_HUD.'
                    : 'A conexão ou a projeção ainda não está pronta. O HUD continua acessível.'}
                </span>
              </div>
            </div>
            {native && prefs.appearance === 'bar' && (
              <HudSizeControls
                presets={sizePresets.filter((preset) => preset.id !== 'companion')}
                active={sizePreset}
                pending={sizePending}
                onSelect={applySize}
              />
            )}
            <div className={styles.companionAppearance}>
              <span>Apresentação</span>
              <button type="button" aria-pressed={prefs.appearance === 'bar'} onClick={() => changeAppearance('bar')}>Barra</button>
              <button type="button" aria-pressed={prefs.appearance === 'companion'} onClick={() => changeAppearance('companion')}>Mascote 2D</button>
            </div>
            <div className={styles.footerActions}>
              <button type="button" onClick={() => void refresh()}>
                <RefreshCw size={13} /> Atualizar
              </button>
              {onOpenNexus && (
                <button type="button" onClick={onOpenNexus}>
                  Abrir Nexus <ExternalLink size={13} />
                </button>
              )}
            </div>
          </div>
        )}

        {prefs.appearance === 'companion' ? (
          <NexusGuardian
            mood={mood}
            count={companionAttention.length}
            expanded={expanded}
            motion={prefs.companionMotion && !prefersReducedMotion}
            bubble={bubble}
            native={native}
            onExpand={() => setExpandedState(!expanded)}
            onOpenPending={() => setExpandedState(true)}
            onOpenAttention={openAttention}
            onSnooze={snoozeBubble}
            onMoveKeys={onNativeDragKeys}
          />
        ) : (
          <div className={styles.capsuleRow}>
            {native && (
              <button
                type="button"
                className={styles.nativeDragHandle}
                data-hud-drag-handle="true"
                aria-label={t('hud.dragLabel')}
                title={t('hud.dragTitle')}
                onKeyDown={onNativeDragKeys}
              >
                <GripHorizontal size={14} aria-hidden="true" />
              </button>
            )}
            <button
              type="button"
              className={styles.capsule}
              data-tone={tone}
              data-evidence={quotaEvidence}
              aria-label={capsuleAccessibleLabel}
              aria-expanded={expanded}
              aria-controls={expanded ? 'nexus-hud-popover' : undefined}
              title={native ? nativeExpansionError || nativeTitle : undefined}
              onClick={() => setExpandedState(!expanded)}
            >
              <img className={styles.logo} src="/nexus-icon.svg" alt="" width={24} height={24} />
              <span className={styles.providerName}>{mainLabel}</span>
              <span className={styles.percent}>{currentPercent}</span>
              <span className={styles.capacityRing} aria-hidden="true" />
              {expanded ? (
                <ChevronDown size={12} className={styles.chevron} aria-hidden="true" />
              ) : (
                <ChevronUp size={12} className={styles.chevron} aria-hidden="true" />
              )}
            </button>
            {!native && prefs.surfaceMode === 'floating' && (
              <button
                type="button"
                className={styles.dragHandle}
                aria-label="Mover Nexus HUD com as setas ou arrastando"
                onPointerDown={onDragDown}
                onPointerMove={onDragMove}
                onPointerUp={onDragUp}
                onKeyDown={onDragKeys}
              >
                <GripHorizontal size={14} />
              </button>
            )}
          </div>
  
        )}
        {prefs.appearance === 'bar' && !expanded && !native && (
          <div className={styles.hoverCard} role="tooltip">
            <div>
              <Clock3 size={14} />
              <span>{focus?.active ? 'Conta ativa' : 'Recurso em destaque'}</span>
              <strong>{displayName}</strong>
            </div>
            <div>
              <Activity size={14} />
              <span>Janela</span>
              <strong>
                {focus?.provider?.limiting_window?.label ||
                  focus?.provider?.limiting_window?.kind ||
                  '—'}
              </strong>
            </div>
            <div>
              <RefreshCw size={14} />
              <span>Renovação</span>
              <strong>{focus?.provider?.limiting_window?.reset_description || '—'}</strong>
            </div>
            <div className={styles.hoverStatus}>
              {!offline && <span className={styles.evidenceIndicator}>{evidenceCopy} · </span>}
              {offline
                ? `Sem conexão · última leitura ${lastSuccessAt ? `${Math.max(1, Math.round((Date.now() - lastSuccessAt) / 60_000))} min atrás` : 'indisponível'}`
                : isAttention
                  ? 'Atenção necessária'
                  : tone === 'unknown'
                    ? 'Quota não confirmada'
                    : tone === 'critical' || tone === 'warning'
                      ? 'Capacidade com restrição'
                      : 'Operação normal'}
            </div>
          </div>
        )}
      </section>
    </aside>
  );
};
