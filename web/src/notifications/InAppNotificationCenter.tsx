import React, { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { useTranslation } from 'react-i18next';
import {
  AlertTriangle,
  CheckCircle2,
  ClipboardList,
  Radio,
  Terminal,
  X,
  XCircle,
} from 'lucide-react';
import type { AttentionItem, EventRecord, RuntimeSession } from '../types';
import { api } from '../api';
import { nexus } from '../nexus/api';
import { sanitizeAttentionText } from '../components/attentionText';
import { Button, IconButton } from '../design-system';
import { ContextDrawer } from '../design-system/primitives/ContextDrawer';
import { buildAttentionRadar, type RadarRuntimeItem } from '../app/attentionRadarModel';
import {
  capacityAlertFromEvent,
  notificationFromQuotaEvent,
  notificationFromRuntime,
  type InAppNotification,
  type InAppNotificationAction,
} from './inAppNotificationModel';
import { loadNotificationPrefs } from './notificationPrefs';
import { MissionAttentionPanel } from './MissionAttentionPanel';

const DISMISS_AFTER_MS = 7_000;

export const InAppNotificationCenter: React.FC<{
  runtimes: RuntimeSession[];
  events?: EventRecord[];
  focusedProjectId?: string;
  drawerOpen?: boolean;
  onCloseDrawer?: () => void;
  onFocusRuntime: (runtimeId: string) => void;
  onFocusAttention?: (item: RadarRuntimeItem) => void;
  onOpenMission?: (item: AttentionItem) => void;
  onAttentionCountChange?: (count: number) => void;
}> = ({
  runtimes,
  events,
  focusedProjectId,
  drawerOpen = false,
  onCloseDrawer,
  onFocusRuntime,
  onFocusAttention,
  onOpenMission,
  onAttentionCountChange,
}) => {
  const { t } = useTranslation();
  const [notifications, setNotifications] = useState<InAppNotification[]>([]);
  const [history, setHistory] = useState<InAppNotification[]>([]);
  const [activeTab, setActiveTab] = useState<'missions' | 'radar' | 'notifications'>('missions');
  const [actionLoadingId, setActionLoadingId] = useState<string | null>(null);
  const [attentionNeedsCount, setAttentionNeedsCount] = useState(0);
  const observed = useRef(new Set<string>());
  const initialized = useRef(false);

  useEffect(() => {
    let active = true;
    const loadNeedsCount = () => {
      void nexus
        .getAttention()
        .then((group) => {
          if (!active) return;
          const count = Number.isFinite(group.total_needs) ? group.total_needs : 0;
          setAttentionNeedsCount(count);
          onAttentionCountChange?.(count);
        })
        .catch(() => undefined);
    };
    loadNeedsCount();
    window.addEventListener('focus', loadNeedsCount);
    const interval = window.setInterval(loadNeedsCount, 30_000);
    return () => {
      active = false;
      window.removeEventListener('focus', loadNeedsCount);
      window.clearInterval(interval);
    };
  }, [onAttentionCountChange]);

  const handleExecuteAction = async (action: InAppNotificationAction, notifId: string) => {
    if (!action.sourceRuntimeId) {
      dismiss(notifId);
      return;
    }
    setActionLoadingId(notifId);
    try {
      let newSession: RuntimeSession;
      if (action.type === 'account_handoff') {
        newSession = await api.accountHandoff(
          action.sourceRuntimeId,
          `${action.targetProvider}:${action.targetProfile}`,
        );
      } else {
        newSession = await api.contextContinue(
          action.sourceRuntimeId,
          action.targetProvider,
          action.targetProfile,
        );
      }
      dismiss(notifId);
      if (newSession?.runtime_id) {
        onFocusRuntime(newSession.runtime_id);
      }
    } catch (err) {
      console.error('Failed to execute failover handoff:', err);
    } finally {
      setActionLoadingId(null);
    }
  };

  useEffect(() => {
    if (!loadNotificationPrefs().notificationsEnabled) return;
    const scoped = focusedProjectId
      ? runtimes.filter((runtime) => runtime.project_id === focusedProjectId)
      : [];
    const runtimeNotifs = scoped
      .filter(
        (runtime) => (runtime.provider_id || runtime.provider || '').toLowerCase() !== 'shell',
      )
      .map(notificationFromRuntime)
      .filter((notification): notification is InAppNotification => notification !== null);

    const quotaNotifs = (events || [])
      .map(
        (event) =>
          capacityAlertFromEvent(event, {
            lowTitle: (provider, remaining) =>
              t('attention.capacityLow', { provider, remaining, defaultValue: '' }) ||
              `Consumo de quota: ${provider} (${remaining})`,
            exhaustedTitle: (provider) =>
              t('attention.capacityExhausted', { provider, defaultValue: '' }) ||
              `Quota esgotada: ${provider}`,
            recoveredTitle: (provider) =>
              t('attention.capacityRecovered', { provider, defaultValue: '' }) ||
              `Quota recuperada: ${provider}`,
            stateLabel: (state) => t(`attention.capacityState_${state}`, { defaultValue: state }),
            resetLabel: (reset) =>
              t('attention.capacityReset', { reset, defaultValue: '' }) || `reset ${reset}`,
            freshnessLabel: (age) =>
              t('attention.capacityFreshness', { age, defaultValue: '' }) ||
              `Última leitura: ${age}`,
            recommendationLabel: (name, reason) =>
              t('attention.capacityRecommendation', { name, reason, defaultValue: '' }) ||
              (reason ? `Recomendado: ${name} — ${reason}` : `Recomendado: ${name}`),
          }) ||
          notificationFromQuotaEvent(event, {
            degradedTitle: t('attention.quotaMonitorDegraded'),
            degradedMessage: (reason) => t('attention.quotaMonitorDegradedMessage', { reason }),
          }),
      )
      .filter((notification): notification is InAppNotification => notification !== null);

    const next = [...quotaNotifs, ...runtimeNotifs];

    if (!initialized.current) {
      next.forEach((notification) => observed.current.add(notification.id));
      initialized.current = true;
      setHistory(next);
      return;
    }

    const fresh = next.filter((notification) => !observed.current.has(notification.id));
    if (fresh.length === 0) return;
    fresh.forEach((notification) => observed.current.add(notification.id));
    setNotifications((current) => [...fresh, ...current].slice(0, 3));
    setHistory((current) => [...fresh, ...current].slice(0, 30));
  }, [runtimes, events, focusedProjectId, t]);

  useEffect(() => {
    if (notifications.length === 0) return;
    const timer = window.setTimeout(
      () => setNotifications((current) => current.slice(0, -1)),
      DISMISS_AFTER_MS,
    );
    return () => window.clearTimeout(timer);
  }, [notifications]);

  const dismiss = (id: string) =>
    setNotifications((current) => current.filter((notification) => notification.id !== id));

  const radarGroups = buildAttentionRadar(runtimes);
  const totalNeeds = radarGroups.reduce((sum, group) => sum + group.needsUserCount, 0);

  return (
    <>
      {/* Transient Toasts */}
      {notifications.length > 0 &&
        createPortal(
          <section
            className="nx-notification-center"
            aria-label={t('notifications.recent', 'Notificações recentes')}
            aria-live="polite"
          >
            {notifications.map((notification) => (
              <article
                className="nx-notification-toast"
                data-tone={notification.tone}
                key={notification.id}
                role="status"
              >
                {notification.tone === 'success' ? (
                  <CheckCircle2 size={18} />
                ) : notification.tone === 'warning' ? (
                  <AlertTriangle size={18} />
                ) : (
                  <XCircle size={18} />
                )}
                <div className="nx-notification-toast__body">
                  <strong>{notification.title}</strong>
                  <span>
                    {sanitizeAttentionText(notification.projectName, 'Projeto')} ·{' '}
                    {sanitizeAttentionText(notification.message, 'Sem detalhes adicionais.')}
                  </span>
                </div>
                {notification.action && notification.action.sourceRuntimeId && (
                  <Button
                    size="sm"
                    tone="brand"
                    disabled={actionLoadingId === notification.id}
                    onClick={() => void handleExecuteAction(notification.action!, notification.id)}
                  >
                    {actionLoadingId === notification.id
                      ? t('legacy.executingHandoff', 'Realizando Handoff...')
                      : notification.action.label}
                  </Button>
                )}
                {notification.runtimeId && (
                  <IconButton
                    label={t('notifications.openTerminal', 'Abrir terminal')}
                    onClick={() => onFocusRuntime(notification.runtimeId!)}
                  >
                    <Terminal size={15} />
                  </IconButton>
                )}
                <IconButton
                  label={t('notifications.dismiss', 'Fechar notificação')}
                  onClick={() => dismiss(notification.id)}
                >
                  <X size={15} />
                </IconButton>
              </article>
            ))}
          </section>,
          document.body,
        )}

      {/* Drawer: Central de Notificações & Radar de Atenção */}
      {drawerOpen && onCloseDrawer && (
        <ContextDrawer
          open={drawerOpen}
          onClose={onCloseDrawer}
          title={t('notifications.title', { defaultValue: 'Attention & Notifications Center' })}
          description={t('notifications.description', {
            defaultValue: 'Real-time radar of terminals and system alert history',
          })}
          width={440}
        >
          <div className="nx-notification-drawer">
            {/* Tabs */}
            <div className="nx-notification-drawer__tabs" role="tablist">
              <button
                type="button"
                role="tab"
                aria-selected={activeTab === 'missions'}
                className={`nx-notification-drawer__tab ${activeTab === 'missions' ? 'nx-notification-drawer__tab--active' : ''}`}
                onClick={() => setActiveTab('missions')}
              >
                <ClipboardList size={13} />
                <span>{t('notifications.missionAttentionTab')}</span>
                {attentionNeedsCount > 0 && (
                  <span className="nx-notification-drawer__badge">{attentionNeedsCount}</span>
                )}
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={activeTab === 'radar'}
                className={`nx-notification-drawer__tab ${activeTab === 'radar' ? 'nx-notification-drawer__tab--active' : ''}`}
                onClick={() => setActiveTab('radar')}
              >
                <Radio size={13} />
                <span>{t('notifications.radarTab', { defaultValue: 'Attention Radar' })}</span>
                {totalNeeds > 0 && (
                  <span className="nx-notification-drawer__badge">{totalNeeds}</span>
                )}
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={activeTab === 'notifications'}
                className={`nx-notification-drawer__tab ${activeTab === 'notifications' ? 'nx-notification-drawer__tab--active' : ''}`}
                onClick={() => setActiveTab('notifications')}
              >
                <span>
                  {t('notifications.notificationsTab', { defaultValue: 'Notifications' })} (
                  {history.length})
                </span>
              </button>
            </div>

            {activeTab === 'missions' && (
              <div className="nx-notification-drawer__panel">
                <MissionAttentionPanel
                  onOpenMission={(item) => {
                    onCloseDrawer();
                    onOpenMission?.(item);
                  }}
                />
              </div>
            )}

            {/* Radar Panel */}
            {activeTab === 'radar' && (
              <div className="nx-notification-drawer__panel">
                {radarGroups.length === 0 ? (
                  <div className="nx-notification-drawer__empty">
                    <Radio size={24} />
                    <p>
                      {t('notifications.noRuntimesRunning', {
                        defaultValue: 'No terminals or agents running at the moment.',
                      })}
                    </p>
                  </div>
                ) : (
                  radarGroups.map((group) => (
                    <section key={group.projectId} className="nx-attention-radar__group">
                      <header>
                        <strong>{group.projectName}</strong>
                        {group.projectId === focusedProjectId && (
                          <span>{t('common.current', { defaultValue: 'current' })}</span>
                        )}
                      </header>
                      <ul>
                        {group.items.map((item) => {
                          const label = sanitizeAttentionText(
                            item.context || item.title,
                            item.title,
                          );
                          return (
                            <li key={item.runtimeId}>
                              <button
                                type="button"
                                className="nx-attention-radar__item"
                                data-badge={item.badge}
                                onClick={() => {
                                  onCloseDrawer();
                                  if (onFocusAttention) onFocusAttention(item);
                                  else onFocusRuntime(item.runtimeId);
                                }}
                                title={label}
                              >
                                <span className="nx-attention-radar__badge" data-badge={item.badge}>
                                  {item.badge === 'needs_user' ? (
                                    <AlertTriangle size={12} />
                                  ) : item.badge === 'error' ? (
                                    <AlertTriangle size={12} />
                                  ) : item.badge === 'completed' ? (
                                    <CheckCircle2 size={12} />
                                  ) : (
                                    <Radio size={12} />
                                  )}
                                  {item.badge}
                                </span>
                                <span className="nx-attention-radar__title">{label}</span>
                              </button>
                            </li>
                          );
                        })}
                      </ul>
                    </section>
                  ))
                )}
              </div>
            )}

            {/* History Panel */}
            {activeTab === 'notifications' && (
              <div className="nx-notification-drawer__panel">
                {history.length === 0 ? (
                  <div className="nx-notification-drawer__empty">
                    <p>
                      {t('notifications.noNotifications', {
                        defaultValue: 'No notifications recorded in this session.',
                      })}
                    </p>
                  </div>
                ) : (
                  history.map((notif) => (
                    <article
                      key={notif.id}
                      className="nx-notification-drawer__item"
                      data-tone={notif.tone}
                    >
                      <div className="nx-notification-drawer__item-main">
                        <div className="nx-notification-drawer__item-header">
                          <strong>{notif.title}</strong>
                          <small>{notif.projectName}</small>
                        </div>
                        <p>{notif.message}</p>
                      </div>
                      {notif.action && notif.action.sourceRuntimeId && (
                        <Button
                          size="sm"
                          tone="brand"
                          disabled={actionLoadingId === notif.id}
                          onClick={() => {
                            onCloseDrawer?.();
                            void handleExecuteAction(notif.action!, notif.id);
                          }}
                        >
                          {actionLoadingId === notif.id
                            ? t('legacy.executingHandoff', 'Realizando Handoff...')
                            : notif.action.label}
                        </Button>
                      )}
                      {notif.runtimeId && (
                        <IconButton
                          label={t('notifications.openTerminal', 'Abrir terminal')}
                          onClick={() => {
                            onCloseDrawer();
                            onFocusRuntime(notif.runtimeId!);
                          }}
                        >
                          <Terminal size={14} />
                        </IconButton>
                      )}
                    </article>
                  ))
                )}
              </div>
            )}
          </div>
        </ContextDrawer>
      )}
    </>
  );
};
