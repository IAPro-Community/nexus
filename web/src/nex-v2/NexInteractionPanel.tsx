import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Send, Settings, X } from 'lucide-react';
import type { NexV2ChatTurn, NexV2HudState, NexV2TabId } from './nexV2Types';
import { sendNexV2Chat } from './nexV2Client';
import styles from './NexInteractionPanel.module.scss';

export interface NexInteractionPanelProps {
  activeTab: NexV2TabId;
  hudState: NexV2HudState | null;
  offline: boolean;
  onSelectTab: (tab: NexV2TabId) => void;
  onClose: () => void;
  onOpenSettings: () => void;
  onAckAlert: (fingerprint: string) => void;
  onSnoozeAlert: (fingerprint: string) => void;
}

const TAB_ORDER: readonly NexV2TabId[] = ['chat', 'resources', 'attention'];

function resolveFreshnessLabel(freshness: string | undefined, t: (key: string) => string): string {
  const normalized = (freshness || '').toUpperCase();
  if (normalized === 'STALE') return t('nexV2.resourcesStatusStale');
  if (normalized === 'UNAVAILABLE') return t('nexV2.resourcesStatusUnavailable');
  if (!normalized || normalized === 'UNKNOWN') return t('nexV2.resourcesStatusUnknown');
  return normalized;
}

export const NexInteractionPanel: React.FC<NexInteractionPanelProps> = ({
  activeTab,
  hudState,
  offline,
  onSelectTab,
  onClose,
  onOpenSettings,
  onAckAlert,
  onSnoozeAlert,
}) => {
  const { t } = useTranslation();
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [history, setHistory] = useState<NexV2ChatTurn[]>([]);
  const tabRefs = useRef<Record<NexV2TabId, HTMLButtonElement | null>>({
    chat: null,
    resources: null,
    attention: null,
  });

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        onClose();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [onClose]);

  const handleTabKeyDown = (
    event: React.KeyboardEvent<HTMLButtonElement>,
    currentIndex: number,
  ) => {
    let nextIndex = -1;
    if (event.key === 'ArrowRight') nextIndex = (currentIndex + 1) % TAB_ORDER.length;
    else if (event.key === 'ArrowLeft')
      nextIndex = (currentIndex - 1 + TAB_ORDER.length) % TAB_ORDER.length;
    else if (event.key === 'Home') nextIndex = 0;
    else if (event.key === 'End') nextIndex = TAB_ORDER.length - 1;
    if (nextIndex < 0) return;
    event.preventDefault();
    const targetTab = TAB_ORDER[nextIndex];
    onSelectTab(targetTab);
    tabRefs.current[targetTab]?.focus();
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    const trimmed = input.trim();
    if (!trimmed || sending) return;
    const userTurn: NexV2ChatTurn = { role: 'user', content: trimmed };
    const nextHistory = [...history, userTurn];
    setHistory(nextHistory);
    setInput('');
    setSending(true);
    const reply = await sendNexV2Chat(trimmed, history, hudState?.project_id);
    setSending(false);
    setHistory((prev) => [
      ...prev,
      {
        role: 'assistant',
        content: reply.offline || !reply.answer ? t('nexV2.chatOfflineReply') : reply.answer,
        source: reply.source,
      },
    ]);
  };

  const providers = hudState?.providers ?? [];
  const activeAlerts = (hudState?.alerts ?? []).filter((a) => !a.acked);
  const runtimeAttention = hudState?.runtime_attention ?? [];

  return (
    <section
      className={styles.panel}
      aria-label={t('nexV2.presenceShellLabel')}
      data-nex-panel
      data-active-tab={activeTab}
    >
      <header className={styles.header}>
        <div className={styles.tabList} role="tablist" aria-label={t('nexV2.tabsLabel')}>
          {TAB_ORDER.map((tab, idx) => {
            const labelKey =
              tab === 'chat'
                ? 'nexV2.tabChat'
                : tab === 'resources'
                  ? 'nexV2.tabResources'
                  : 'nexV2.tabAttention';
            const selected = activeTab === tab;
            return (
              <button
                key={tab}
                ref={(el) => {
                  tabRefs.current[tab] = el;
                }}
                type="button"
                role="tab"
                id={`nex-v2-tab-${tab}`}
                aria-selected={selected}
                aria-controls={`nex-v2-tabpanel-${tab}`}
                tabIndex={selected ? 0 : -1}
                data-nex-tab={tab}
                className={styles.tabBtn}
                onClick={() => onSelectTab(tab)}
                onKeyDown={(event) => handleTabKeyDown(event, idx)}
              >
                {t(labelKey)}
              </button>
            );
          })}
        </div>

        <div className={styles.headerActions}>
          <button
            type="button"
            className={styles.iconBtn}
            aria-label={t('nexV2.openSettingsLabel')}
            data-nex-panel-settings
            onClick={onOpenSettings}
          >
            <Settings size={14} aria-hidden="true" />
          </button>
          <button
            type="button"
            className={styles.iconBtn}
            aria-label={t('nexV2.closePanelLabel')}
            data-nex-panel-close
            onClick={onClose}
          >
            <X size={14} aria-hidden="true" />
          </button>
        </div>
      </header>

      <div
        className={styles.body}
        role="tabpanel"
        id={`nex-v2-tabpanel-${activeTab}`}
        aria-labelledby={`nex-v2-tab-${activeTab}`}
      >
        {activeTab === 'chat' && (
          <div className={styles.chatContainer}>
            <div className={styles.chatMessages}>
              {history.length === 0 ? (
                <p className={styles.emptyText}>{t('nexV2.chatEmpty')}</p>
              ) : (
                history.map((turn, idx) => (
                  <div
                    key={`${turn.role}-${idx}`}
                    className={styles.chatMessage}
                    data-role={turn.role}
                  >
                    <span className={styles.chatRole}>
                      {turn.role === 'user'
                        ? t('nexV2.chatUserRole')
                        : t('nexV2.chatAssistantRole')}
                    </span>
                    <p className={styles.chatText}>{turn.content}</p>
                  </div>
                ))
              )}
            </div>
            <form className={styles.chatForm} onSubmit={(e) => void handleSubmit(e)}>
              <input
                type="text"
                className={styles.chatInput}
                value={input}
                onChange={(event) => setInput(event.target.value)}
                placeholder={t('nexV2.chatPlaceholder')}
                aria-label={t('nexV2.chatPlaceholder')}
              />
              <button type="submit" className={styles.sendBtn} disabled={sending || !input.trim()}>
                <Send size={14} aria-hidden="true" />
                <span>{sending ? t('nexV2.chatSending') : t('nexV2.chatSend')}</span>
              </button>
            </form>
          </div>
        )}

        {activeTab === 'resources' && (
          <div className={styles.stack} data-nex-resources>
            {offline || providers.length === 0 ? (
              <div className={styles.card}>
                <span className={styles.statusBadge}>{t('nexV2.resourcesStatusUnknown')}</span>
                <p className={styles.emptyText}>{t('nexV2.resourcesEmpty')}</p>
              </div>
            ) : (
              providers.map((provider) => {
                const pct = provider.limiting_window?.remaining_percent;
                const freshness = resolveFreshnessLabel(provider.freshness, t);
                return (
                  <div key={`${provider.provider}-${provider.profile}`} className={styles.card}>
                    <div className={styles.cardHeader}>
                      <strong>{provider.display_name || provider.provider}</strong>
                      <span className={styles.statusBadge}>{freshness}</span>
                    </div>
                    <p className={styles.cardMeta}>
                      {typeof pct === 'number'
                        ? t('nexV2.resourcesRemaining', { percent: Math.round(pct) })
                        : t('nexV2.resourcesQuotaUnknown')}
                    </p>
                  </div>
                );
              })
            )}

            <div className={styles.summaryRow}>
              <span className={styles.summaryChip}>
                {t('nexV2.resourcesAgentsTitle', { count: hudState?.agents.length ?? 0 })}
              </span>
              <span className={styles.summaryChip}>
                {t('nexV2.resourcesMissionsTitle', { count: hudState?.missions.length ?? 0 })}
              </span>
              <span className={styles.summaryChip}>
                {t('nexV2.resourcesCostTitle', {
                  level: hudState?.cost?.budget_level || t('nexV2.resourcesStatusUnknown'),
                })}
              </span>
            </div>
          </div>
        )}

        {activeTab === 'attention' && (
          <div className={styles.stack} data-nex-attention>
            {activeAlerts.length === 0 && runtimeAttention.length === 0 ? (
              <p className={styles.emptyText}>{t('nexV2.attentionEmpty')}</p>
            ) : (
              <>
                {activeAlerts.map((alert) => (
                  <div key={alert.fingerprint} className={styles.card}>
                    <div className={styles.cardHeader}>
                      <strong>{alert.provider || alert.category}</strong>
                      <span className={styles.statusBadge}>{alert.severity || 'INFO'}</span>
                    </div>
                    {alert.reason && <p className={styles.cardMeta}>{alert.reason}</p>}
                    <div className={styles.cardActions}>
                      <button
                        type="button"
                        className={styles.smallBtn}
                        onClick={() => onAckAlert(alert.fingerprint)}
                      >
                        {t('nexV2.attentionAck')}
                      </button>
                      <button
                        type="button"
                        className={styles.smallBtn}
                        onClick={() => onSnoozeAlert(alert.fingerprint)}
                      >
                        {t('nexV2.attentionSnooze')}
                      </button>
                    </div>
                  </div>
                ))}
                {runtimeAttention.map((item) => (
                  <div key={item.fingerprint} className={styles.card}>
                    <div className={styles.cardHeader}>
                      <strong>
                        {t('nexV2.attentionRuntimeLabel', { provider: item.provider })}
                      </strong>
                      <span className={styles.statusBadge}>{item.prompt_kind}</span>
                    </div>
                    <p className={styles.cardMeta}>{item.reason}</p>
                  </div>
                ))}
              </>
            )}
          </div>
        )}
      </div>
    </section>
  );
};

export default NexInteractionPanel;
