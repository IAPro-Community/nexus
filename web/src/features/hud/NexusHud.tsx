import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Bot,
  ChevronDown,
  ChevronUp,
  CircleAlert,
  Gauge,
  GripHorizontal,
  Settings2,
  Users,
  Workflow,
  X,
  Zap,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { nexus } from '../../nexus/api';
import {
  activeMission,
  formatHudCost,
  isWorkingAgent,
  loadHudPreferences,
  providerCapacityLabel,
  saveHudPreferences,
  type HudAlert,
  type HudPresentationPreferences,
  type HudResponse,
  type HudState,
} from './hudModel';
import styles from './NexusHud.module.scss';

const MIN_REFRESH_MS = 15_000;
const FLOATING_MIN_X = 8;
const FLOATING_MIN_Y = 8;
const FLOATING_KEYBOARD_STEP = 16;

export interface NexusHudProps {
  projectId?: string;
  onOpenUsage?: () => void;
  onOpenAgents?: () => void;
  onOpenMissions?: () => void;
  onOpenSettings?: () => void;
}

const clampFloatingPosition = (x: number, y: number) => {
  if (typeof window === 'undefined') return { x, y };
  return {
    x: Math.max(FLOATING_MIN_X, Math.min(x, Math.max(FLOATING_MIN_X, window.innerWidth - 340))),
    y: Math.max(FLOATING_MIN_Y, Math.min(y, Math.max(FLOATING_MIN_Y, window.innerHeight - 90))),
  };
};

const alertCopy = (alert?: HudAlert) => {
  if (!alert) return null;
  const remaining =
    alert.remaining_percent === undefined ? 'UNKNOWN' : `${Math.round(alert.remaining_percent)}%`;
  return {
    title: `${alert.provider || 'Provider'} · ${remaining}`,
    detail: alert.reason || alert.severity,
  };
};

const primaryCopy = (state: HudState, requestedMode: HudPresentationPreferences['mode']) => {
  const effectiveKind = requestedMode === 'AUTO' || requestedMode === 'MINIMAL' ? state.primary.kind : requestedMode;
  const alert = state.alerts.find((item) => item.severity === 'CRITICAL') || state.alerts[0];
  if (effectiveKind === 'ALERT' && alert) {
    return { kind: 'ALERT', icon: CircleAlert, ...alertCopy(alert)! };
  }
  const mission = activeMission(state.missions);
  if (effectiveKind === 'MISSION' && mission) {
    return {
      kind: 'MISSION',
      icon: Workflow,
      title: mission.needs_user ? 'Mission aguardando você' : 'Mission em andamento',
      detail:
        mission.current_task_title ||
        mission.intervention_reason ||
        `${mission.completed_packages}/${mission.total_packages} packages`,
    };
  }
  const agent = state.agents.find((item) => item.id === state.primary.id) || state.agents.find(isWorkingAgent);
  if (effectiveKind === 'AGENT' && agent) {
    return {
      kind: 'AGENT',
      icon: Bot,
      title: agent.name,
      detail: [agent.provider, agent.model, agent.status].filter(Boolean).join(' · '),
    };
  }
  if (effectiveKind === 'RESOURCE' && state.recommendation) {
    return {
      kind: 'RESOURCE',
      icon: Zap,
      title: `Recurso recomendado: ${state.recommendation.display_name || state.recommendation.provider}`,
      detail: state.recommendation.explanation || 'Recomendação geral do ResourceScheduler',
    };
  }
  return {
    kind: 'CAPACITY',
    icon: Gauge,
    title: 'Capacity',
    detail: state.providers.length ? `${state.providers.length} recursos monitorados` : 'Sem dados de provider',
  };
};

export const NexusHud: React.FC<NexusHudProps> = ({
  projectId,
  onOpenUsage,
  onOpenAgents,
  onOpenMissions,
  onOpenSettings,
}) => {
  const { t } = useTranslation();
  const [prefs, setPrefs] = useState<HudPresentationPreferences>(() => loadHudPreferences());
  const [response, setResponse] = useState<HudResponse | null>(null);
  const [expanded, setExpanded] = useState(false);
  const [offline, setOffline] = useState(false);
  const [lastSuccessAt, setLastSuccessAt] = useState<number | null>(null);
  const [alertActionPending, setAlertActionPending] = useState(false);
  const requestInFlight = useRef<Promise<void> | null>(null);
  const dragState = useRef<{ pointerId: number; offsetX: number; offsetY: number } | null>(null);

  const refresh = useCallback(() => {
    if (requestInFlight.current) return requestInFlight.current;
    const task = nexus
      .getHUDState(projectId)
      .then((next) => {
        setResponse(next);
        setOffline(false);
        setLastSuccessAt(Date.now());
      })
      .catch(() => {
        setOffline(true);
      })
      .finally(() => {
        requestInFlight.current = null;
      });
    requestInFlight.current = task;
    return task;
  }, [projectId]);

  useEffect(() => {
    void refresh();
    const onVisibility = () => {
      if (document.visibilityState === 'visible') void refresh();
    };
    const onFocus = () => void refresh();
    document.addEventListener('visibilitychange', onVisibility);
    window.addEventListener('focus', onFocus);

    const ttl = response?.state?.system.cache_ttl_millis || MIN_REFRESH_MS;
    const timer = window.setInterval(() => {
      if (document.visibilityState === 'visible') void refresh();
    }, Math.max(MIN_REFRESH_MS, ttl));

    return () => {
      document.removeEventListener('visibilitychange', onVisibility);
      window.removeEventListener('focus', onFocus);
      window.clearInterval(timer);
    };
  }, [refresh, response?.state?.system.cache_ttl_millis]);

  useEffect(() => {
    const onPreferences = () => setPrefs(loadHudPreferences());
    window.addEventListener('nexus:hud-preferences', onPreferences);
    return () => window.removeEventListener('nexus:hud-preferences', onPreferences);
  }, []);

  useEffect(() => {
    if (prefs.surfaceMode !== 'floating') return;
    const onResize = () => {
      setPrefs((current) => {
        const nextPosition = clampFloatingPosition(
          current.floatingPosition.x,
          current.floatingPosition.y,
        );
        if (
          nextPosition.x === current.floatingPosition.x &&
          nextPosition.y === current.floatingPosition.y
        ) {
          return current;
        }
        const next = { ...current, floatingPosition: nextPosition };
        saveHudPreferences(next);
        return next;
      });
    };
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, [prefs.surfaceMode]);

  const state = response?.state;
  const cost = useMemo(() => formatHudCost(state?.cost), [state?.cost]);
  const primary = useMemo(() => (state ? primaryCopy(state, prefs.mode) : null), [state, prefs.mode]);
  const topAlert = state?.alerts[0];
  const activeAgents = state?.agents.filter(isWorkingAgent) || [];
  const mission = state ? activeMission(state.missions) : undefined;
  const modeShows = (kind: 'CAPACITY' | 'RESOURCE' | 'AGENT' | 'MISSION' | 'ALERT' | 'COST') =>
    prefs.mode === 'AUTO' ||
    (prefs.mode === 'MINIMAL' && kind === 'CAPACITY') ||
    prefs.mode === kind;

  const updatePrefs = (patch: Partial<HudPresentationPreferences>) => {
    const next = { ...prefs, ...patch };
    setPrefs(next);
    saveHudPreferences(next);
  };

  const hideHUD = () => updatePrefs({ enabled: false });

  const handleAck = async () => {
    if (!topAlert?.fingerprint) return;
    setAlertActionPending(true);
    try {
      await nexus.ackCapacityAlert(topAlert.fingerprint);
      toast.success(t('settings.capacityAcked', 'Alerta reconhecido'));
      await refresh();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error));
    } finally {
      setAlertActionPending(false);
    }
  };

  const handleSnooze = async () => {
    if (!topAlert?.fingerprint) return;
    setAlertActionPending(true);
    try {
      await nexus.snoozeCapacityAlert(topAlert.fingerprint);
      toast.success(t('settings.capacitySnoozed', 'Alerta adiado'));
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

  const onDragPointerDown = (event: React.PointerEvent<HTMLButtonElement>) => {
    if (prefs.surfaceMode !== 'floating') return;
    const panel = event.currentTarget.closest('[data-hud-panel]')?.getBoundingClientRect();
    if (!panel) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    dragState.current = {
      pointerId: event.pointerId,
      offsetX: event.clientX - panel.left,
      offsetY: event.clientY - panel.top,
    };
  };

  const onDragPointerMove = (event: React.PointerEvent<HTMLButtonElement>) => {
    const drag = dragState.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    updateFloatingPosition(event.clientX - drag.offsetX, event.clientY - drag.offsetY);
  };

  const onDragPointerUp = (event: React.PointerEvent<HTMLButtonElement>) => {
    const drag = dragState.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    dragState.current = null;
    updateFloatingPosition(event.clientX - drag.offsetX, event.clientY - drag.offsetY, true);
  };

  const onDragKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>) => {
    if (prefs.surfaceMode !== 'floating') return;
    let dx = 0;
    let dy = 0;
    if (event.key === 'ArrowLeft') dx = -FLOATING_KEYBOARD_STEP;
    else if (event.key === 'ArrowRight') dx = FLOATING_KEYBOARD_STEP;
    else if (event.key === 'ArrowUp') dy = -FLOATING_KEYBOARD_STEP;
    else if (event.key === 'ArrowDown') dy = FLOATING_KEYBOARD_STEP;
    else return;
    event.preventDefault();
    updateFloatingPosition(
      prefs.floatingPosition.x + dx,
      prefs.floatingPosition.y + dy,
      true,
    );
  };

  if (!prefs.enabled || !response?.enabled || !state) return null;

  const floatingStyle =
    prefs.surfaceMode === 'floating'
      ? ({
          '--hud-floating-x': `${prefs.floatingPosition.x}px`,
          '--hud-floating-y': `${prefs.floatingPosition.y}px`,
        } as React.CSSProperties)
      : undefined;

  return (
    <aside
      className={styles.root}
      data-surface={prefs.surfaceMode}
      data-dock={prefs.dock}
      style={floatingStyle}
      aria-label="Nexus HUD"
      aria-live="polite"
    >
      <section className={styles.panel} data-expanded={expanded ? 'true' : 'false'} data-hud-panel>
        <div className={styles.compactBar}>
          <div className={styles.brand}>
            <Zap size={15} aria-hidden="true" />
            <span>Nexus HUD</span>
            <span className={styles.rollout}>{response.rollout}</span>
          </div>

          <div className={styles.metrics}>
            {prefs.showCapacity && modeShows('CAPACITY') &&
              state.providers.slice(0, 4).map((provider) => (
                <button
                  type="button"
                  className={styles.metric}
                  data-state={provider.capacity_state}
                  key={`${provider.provider}:${provider.profile}`}
                  onClick={onOpenUsage}
                  aria-label={`${provider.provider} ${provider.profile}: ${providerCapacityLabel(provider)}, ${provider.capacity_state}`}
                >
                  <span>{provider.provider}</span>
                  <strong>{providerCapacityLabel(provider)}</strong>
                </button>
              ))}
            {prefs.showRecommendation && modeShows('RESOURCE') && state.recommendation ? (
              <button
                type="button"
                className={styles.metric}
                onClick={onOpenUsage}
                aria-label={`Recurso recomendado: ${state.recommendation.display_name || state.recommendation.provider}`}
              >
                <Zap size={12} aria-hidden="true" />
                <strong>{state.recommendation.provider}</strong>
              </button>
            ) : null}
            {prefs.showAgents && modeShows('AGENT') && activeAgents.length > 0 ? (
              <button
                type="button"
                className={styles.metric}
                onClick={onOpenAgents}
                aria-label={`${activeAgents.length} agentes trabalhando`}
              >
                <Users size={12} aria-hidden="true" />
                <strong>{activeAgents.length}</strong>
              </button>
            ) : null}
            {prefs.showMissions && modeShows('MISSION') && mission ? (
              <button
                type="button"
                className={styles.metric}
                data-state={mission.needs_user ? 'WARNING' : mission.state}
                onClick={onOpenMissions}
                aria-label={mission.needs_user ? 'Mission aguardando intervenção' : `Mission ${mission.state}`}
              >
                <Workflow size={12} aria-hidden="true" />
                <strong>
                  {mission.total_packages > 0
                    ? `${mission.completed_packages}/${mission.total_packages}`
                    : mission.needs_user
                      ? 'ação'
                      : 'ativa'}
                </strong>
              </button>
            ) : null}
            {prefs.showCost && modeShows('COST') && !prefs.privacyMode ? (
              <span
                className={styles.metric}
                aria-label={`Custo ${cost.semantics}: ${cost.label}`}
              >
                <strong>{cost.label}</strong>
              </span>
            ) : null}
          </div>

          <div className={styles.actions}>
            {offline ? (
              <span className={styles.offline}>
                OFFLINE
                {lastSuccessAt
                  ? ` · ${Math.max(1, Math.round((Date.now() - lastSuccessAt) / 60_000))}m`
                  : ''}
              </span>
            ) : null}
            <button
              type="button"
              className={styles.dragHandle}
              aria-label="Mover Nexus HUD. Use as setas do teclado ou arraste."
              onPointerDown={onDragPointerDown}
              onPointerMove={onDragPointerMove}
              onPointerUp={onDragPointerUp}
              onKeyDown={onDragKeyDown}
            >
              <GripHorizontal size={15} aria-hidden="true" />
            </button>
            <button
              type="button"
              className={styles.iconButton}
              aria-expanded={expanded}
              aria-label={expanded ? 'Recolher Nexus HUD' : 'Expandir Nexus HUD'}
              onClick={() => setExpanded((current) => !current)}
            >
              {expanded ? <ChevronDown size={16} /> : <ChevronUp size={16} />}
            </button>
            <button
              type="button"
              className={styles.iconButton}
              aria-label="Configurações do Nexus HUD"
              onClick={onOpenSettings}
            >
              <Settings2 size={15} />
            </button>
            <button
              type="button"
              className={styles.iconButton}
              aria-label="Ocultar Nexus HUD"
              onClick={hideHUD}
            >
              <X size={15} />
            </button>
          </div>
        </div>

        {expanded && (
          <div className={styles.expanded}>
            {primary ? (() => {
              const PrimaryIcon = primary.icon;
              return (
              <section className={styles.primary} data-kind={primary.kind}>
                <PrimaryIcon size={18} aria-hidden="true" />
                <div className={styles.primaryText}>
                  <strong>{primary.title}</strong>
                  <span>{primary.detail}</span>
                </div>
              </section>
              );
            })() : null}

            {topAlert && modeShows('ALERT') ? (
              <div className={styles.actionRow}>
                <button
                  type="button"
                  className={styles.textButton}
                  disabled={alertActionPending}
                  onClick={() => void handleAck()}
                >
                  Reconhecer alerta
                </button>
                <button
                  type="button"
                  className={styles.textButton}
                  disabled={alertActionPending}
                  onClick={() => void handleSnooze()}
                >
                  Adiar
                </button>
              </div>
            ) : null}

            {prefs.showCapacity && modeShows('CAPACITY') ? (
              <section className={styles.section}>
                <div className={styles.sectionHeader}>
                  <h3>Capacity</h3>
                  {onOpenUsage ? (
                    <button type="button" className={styles.textButton} onClick={onOpenUsage}>
                      Abrir Usage
                    </button>
                  ) : null}
                </div>
                <div className={styles.providerGrid}>
                  {state.providers.map((provider) => {
                    const pct = provider.limiting_window?.remaining_percent;
                    const width =
                      pct === undefined ? 0 : Math.max(0, Math.min(100, Math.round(pct)));
                    return (
                      <article
                        className={styles.providerCard}
                        key={`${provider.provider}:${provider.profile}`}
                        data-state={provider.capacity_state}
                      >
                        <div className={styles.providerTitle}>
                          <strong>{provider.display_name || provider.provider}</strong>
                          <span className={styles.capacityValue}>
                            {providerCapacityLabel(provider)}
                          </span>
                        </div>
                        <div
                          className={styles.windowTrack}
                          role="progressbar"
                          aria-label={`Capacidade limitante de ${provider.provider}`}
                          aria-valuenow={pct === undefined ? undefined : width}
                          aria-valuemin={0}
                          aria-valuemax={100}
                        >
                          <span
                            className={styles.windowFill}
                            style={{ '--hud-capacity-width': `${width}%` } as React.CSSProperties}
                          />
                        </div>
                        <div className={styles.providerMeta}>
                          <span>{provider.limiting_window?.label || provider.limiting_window?.kind || 'janela desconhecida'}</span>
                          <span>{provider.limiting_window?.reset_description || 'reset desconhecido'}</span>
                          <span>{provider.freshness}</span>
                          <span>{provider.source || 'fonte desconhecida'}</span>
                        </div>
                      </article>
                    );
                  })}
                </div>
              </section>
            ) : null}

            {prefs.showRecommendation && modeShows('RESOURCE') && state.recommendation ? (
              <section className={styles.section}>
                <div className={styles.sectionHeader}>
                  <h3>ResourceScheduler</h3>
                </div>
                <div className={styles.listRow}>
                  <div>
                    <strong>
                      {state.recommendation.display_name || state.recommendation.provider}
                    </strong>
                    <div className={styles.meta}>
                      <span>recomendação {state.recommendation.context.toLowerCase()}</span>
                      <span>{state.recommendation.confidence}</span>
                    </div>
                  </div>
                  {onOpenUsage ? (
                    <button type="button" className={styles.textButton} onClick={onOpenUsage}>
                      Por quê?
                    </button>
                  ) : null}
                </div>
              </section>
            ) : null}

            {prefs.showAgents && modeShows('AGENT') && state.agents.length > 0 ? (
              <section className={styles.section}>
                <div className={styles.sectionHeader}>
                  <h3>Agents</h3>
                  {onOpenAgents ? (
                    <button type="button" className={styles.textButton} onClick={onOpenAgents}>
                      Abrir Agents
                    </button>
                  ) : null}
                </div>
                <div className={styles.list}>
                  {state.agents.slice(0, 5).map((agent) => (
                    <div className={styles.listRow} key={agent.id}>
                      <div>
                        <strong>{prefs.privacyMode ? agent.role || 'Agent' : agent.name}</strong>
                        <div className={styles.meta}>
                          <span>{agent.status}</span>
                          {agent.provider ? <span>{agent.provider}</span> : null}
                          {agent.model ? <span>{agent.model}</span> : null}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </section>
            ) : null}

            {prefs.showMissions && modeShows('MISSION') && mission ? (
              <section className={styles.section}>
                <div className={styles.sectionHeader}>
                  <h3>Mission</h3>
                  {onOpenMissions ? (
                    <button type="button" className={styles.textButton} onClick={onOpenMissions}>
                      Abrir Mission
                    </button>
                  ) : null}
                </div>
                <div className={styles.listRow}>
                  <div>
                    <strong>
                      {mission.needs_user
                        ? 'Aguardando intervenção'
                        : mission.current_task_title || mission.current_stage}
                    </strong>
                    <div className={styles.meta}>
                      <span>{mission.state}</span>
                      {mission.total_packages > 0 ? (
                        <span>
                          {mission.completed_packages}/{mission.total_packages} packages
                        </span>
                      ) : null}
                    </div>
                  </div>
                </div>
              </section>
            ) : null}

            {prefs.showCost && modeShows('COST') && !prefs.privacyMode && state.cost ? (
              <section className={styles.section}>
                <div className={styles.sectionHeader}>
                  <h3>Cost Intelligence</h3>
                  <strong>{cost.label}</strong>
                </div>
                <div className={styles.list}>
                  {state.cost.by_kind.map((entry) => (
                    <div className={styles.listRow} key={entry.kind}>
                      <span>{entry.kind}</span>
                      <strong>
                        {new Intl.NumberFormat(undefined, {
                          style: 'currency',
                          currency: 'USD',
                        }).format(entry.microusd / 1_000_000)}
                      </strong>
                    </div>
                  ))}
                  {state.cost.unknown_observation_count > 0 ? (
                    <div className={styles.listRow}>
                      <span>UNKNOWN</span>
                      <strong>{state.cost.unknown_observation_count} observações</strong>
                    </div>
                  ) : null}
                </div>
              </section>
            ) : null}
          </div>
        )}
      </section>
    </aside>
  );
};
