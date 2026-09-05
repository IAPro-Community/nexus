import React, { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { AlertTriangle, CheckCircle2, Radio, Terminal, X, XCircle } from 'lucide-react';
import type { RuntimeSession } from '../types';
import { sanitizeAttentionText } from '../components/attentionText';
import { IconButton } from '../design-system';
import { ContextDrawer } from '../design-system/primitives/ContextDrawer';
import { buildAttentionRadar, type RadarRuntimeItem } from '../app/attentionRadarModel';
import { notificationFromRuntime, type InAppNotification } from './inAppNotificationModel';
import { loadNotificationPrefs } from './notificationPrefs';
import { nexus } from '../nexus/api';
import { CapacityAlertCard } from './CapacityAlertCard';
import { capacityAlertFromEvent } from './notificationModel';
import type { CapacityAlert } from '../types';

const DISMISS_AFTER_MS = 7_000;

export const InAppNotificationCenter: React.FC<{
  runtimes: RuntimeSession[];
  focusedProjectId?: string;
  drawerOpen?: boolean;
  onCloseDrawer?: () => void;
  onFocusRuntime: (runtimeId: string) => void;
  onFocusAttention?: (item: RadarRuntimeItem) => void;
}> = ({
  runtimes,
  focusedProjectId,
  drawerOpen = false,
  onCloseDrawer,
  onFocusRuntime,
  onFocusAttention,
}) => {
  const [notifications, setNotifications] = useState<InAppNotification[]>([]);
  const [history, setHistory] = useState<InAppNotification[]>([]);
  const [activeTab, setActiveTab] = useState<'radar' | 'notifications'>('radar');
  const [capacityAlerts, setCapacityAlerts] = useState<CapacityAlert[]>([]);
  const observed = useRef(new Set<string>());
  const initialized = useRef(false);

  useEffect(() => {
    if (!loadNotificationPrefs().notificationsEnabled) return;
    const scoped = focusedProjectId
      ? runtimes.filter((runtime) => runtime.project_id === focusedProjectId)
      : [];
    const next = scoped
      .filter(
        (runtime) => (runtime.provider_id || runtime.provider || '').toLowerCase() !== 'shell',
      )
      .map(notificationFromRuntime)
      .filter((notification): notification is InAppNotification => notification !== null);

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
  }, [runtimes, focusedProjectId]);

  useEffect(() => {
    let active = true;
    const loadCapacityAlerts = async () => {
      try {
        const page = await nexus.getCapacityMonitorHistory(30);
        const alerts = page.events
          .map((event) => capacityAlertFromEvent(event))
          .filter((alert): alert is CapacityAlert => alert !== null)
          .filter((alert) => alert.severity !== 'RECOVERED');
        if (active) setCapacityAlerts(alerts);
      } catch {
        // Capacity monitoring is optional; runtime attention remains available.
      }
    };
    void loadCapacityAlerts();
    const timer = window.setInterval(() => void loadCapacityAlerts(), 30_000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, []);

  const acknowledgeCapacityAlert = async (alertId: string) => {
    await nexus.acknowledgeCapacityAlert(alertId);
    setCapacityAlerts((current) => current.filter((alert) => alert.id !== alertId));
  };

  const snoozeCapacityAlert = async (alertId: string, durationMinutes: number) => {
    await nexus.snoozeCapacityAlert(alertId, durationMinutes * 60);
    setCapacityAlerts((current) => current.filter((alert) => alert.id !== alertId));
  };

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
            aria-label="Notificações recentes"
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
                <IconButton
                  label="Abrir terminal"
                  onClick={() => onFocusRuntime(notification.runtimeId)}
                >
                  <Terminal size={15} />
                </IconButton>
                <IconButton label="Fechar notificação" onClick={() => dismiss(notification.id)}>
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
          title="Central de Atenção & Notificações"
          description="Radar em tempo real dos terminais e histórico de alertas do sistema"
          width={440}
        >
          <div className="nx-notification-drawer">
            {/* Tabs */}
            <div className="nx-notification-drawer__tabs" role="tablist">
              <button
                type="button"
                role="tab"
                aria-selected={activeTab === 'radar'}
                className={`nx-notification-drawer__tab ${activeTab === 'radar' ? 'nx-notification-drawer__tab--active' : ''}`}
                onClick={() => setActiveTab('radar')}
              >
                <Radio size={13} />
                <span>Radar de Atenção</span>
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
                <span>Notificações ({history.length + capacityAlerts.length})</span>
              </button>
            </div>

            {/* Radar Panel */}
            {activeTab === 'radar' && (
              <div className="nx-notification-drawer__panel">
                {radarGroups.length === 0 ? (
                  <div className="nx-notification-drawer__empty">
                    <Radio size={24} />
                    <p>Nenhum terminal ou agente em execução no momento.</p>
                  </div>
                ) : (
                  radarGroups.map((group) => (
                    <section key={group.projectId} className="nx-attention-radar__group">
                      <header>
                        <strong>{group.projectName}</strong>
                        {group.projectId === focusedProjectId && <span>atual</span>}
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
                {capacityAlerts.map((alert) => (
                  <CapacityAlertCard
                    key={alert.id}
                    alert={alert}
                    onAcknowledge={(id) => void acknowledgeCapacityAlert(id)}
                    onSnooze={(id, minutes) => void snoozeCapacityAlert(id, minutes)}
                  />
                ))}
                {history.length === 0 && capacityAlerts.length === 0 ? (
                  <div className="nx-notification-drawer__empty">
                    <p>Nenhuma notificação registrada nesta sessão.</p>
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
                      <IconButton
                        label="Abrir terminal"
                        onClick={() => {
                          onCloseDrawer();
                          onFocusRuntime(notif.runtimeId);
                        }}
                      >
                        <Terminal size={14} />
                      </IconButton>
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
