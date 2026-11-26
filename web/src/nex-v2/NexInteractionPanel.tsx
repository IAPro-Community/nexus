import React, { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Settings, X } from 'lucide-react';
import type { NexOperationalMood } from './motion/nexBehaviorState';
import {
  countAttentionItems,
  hasCriticalAttention,
  resolveLowestQuotaPercent,
  resolveMoodTranslationKey,
} from './nexV2HudHelpers';
import type {
  NexV2ActionProposal,
  NexV2EventEnvelope,
  NexV2HudState,
  NexV2ModelCatalogWatchSnapshot,
  NexV2TabId,
} from './nexV2Types';
import { NexAttentionTab } from './NexAttentionTab';
import { NexChatTab } from './NexChatTab';
import { NexResourcesTab } from './NexResourcesTab';
import styles from './NexInteractionPanel.module.scss';

export interface NexInteractionPanelProps {
  activeTab: NexV2TabId;
  mood?: NexOperationalMood;
  hudState: NexV2HudState | null;
  modelWatch?: NexV2ModelCatalogWatchSnapshot | null;
  events?: NexV2EventEnvelope[];
  privacyMode?: boolean;
  offline: boolean;
  onSelectTab: (tab: NexV2TabId) => void;
  onClose: () => void;
  onOpenSettings: () => void;
  onAckAlert: (fingerprint: string) => void;
  onSnoozeAlert: (fingerprint: string) => void;
  onSendingChange?: (sending: boolean) => void;
  onReplyResolved?: () => void;
}

const TAB_ORDER: readonly NexV2TabId[] = ['resources', 'attention', 'chat'];

export const NexInteractionPanel: React.FC<NexInteractionPanelProps> = ({
  activeTab,
  mood = 'idle',
  hudState,
  modelWatch = null,
  events = [],
  privacyMode = false,
  offline,
  onSelectTab,
  onClose,
  onOpenSettings,
  onAckAlert,
  onSnoozeAlert,
  onSendingChange,
  onReplyResolved,
}) => {
  const handleConfirmProposal = (proposal: NexV2ActionProposal) => {
    switch (proposal.target) {
      case 'usage':
      case 'alerts':
      case 'resources':
        onSelectTab('resources');
        break;
      case 'attention':
      case 'missions':
        onSelectTab('attention');
        break;
      case 'settings':
      case 'intelligence':
        onOpenSettings();
        break;
      default:
        break;
    }
  };
  const { t } = useTranslation();
  const tabRefs = useRef<Record<NexV2TabId, HTMLButtonElement | null>>({
    resources: null,
    attention: null,
    chat: null,
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

  const attentionCount = countAttentionItems(hudState);
  const criticalAttention = hasCriticalAttention(hudState);
  const lowestQuota = resolveLowestQuotaPercent(hudState);
  const agentCount = hudState?.agents.length ?? 0;

  let summaryText = t('nexV2.statusSummaryOffline');
  if (!offline) {
    if (attentionCount > 0) {
      summaryText = t('nexV2.statusSummaryUrgent', { count: attentionCount });
    } else if (lowestQuota !== null) {
      summaryText = t('nexV2.statusSummaryHealthy', {
        agents: agentCount,
        quota: `${lowestQuota}%`,
      });
    } else {
      summaryText = t('nexV2.statusSummaryAgentsOnly', { agents: agentCount });
    }
  }

  return (
    <section
      className={styles.panel}
      aria-label={t('nexV2.presenceShellLabel')}
      data-nex-panel
      data-active-tab={activeTab}
    >
      <header className={styles.header}>
        <div className={styles.statusStrip} data-mood={mood}>
          <div className={styles.statusLead} data-wails-draggable="true">
            <span className={styles.statusDot} aria-hidden="true" />
            <span className={styles.statusMood}>{t(resolveMoodTranslationKey(mood))}</span>
            <span className={styles.statusDivider} aria-hidden="true">
              ·
            </span>
            <span className={styles.statusSummary}>{summaryText}</span>
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
        </div>

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
                <span>{t(labelKey)}</span>
                {tab === 'attention' && attentionCount > 0 && (
                  <span
                    className={styles.tabBadge}
                    data-severity={criticalAttention ? 'CRITICAL' : 'WARNING'}
                    data-nex-tab-badge={attentionCount}
                  >
                    {attentionCount}
                  </span>
                )}
              </button>
            );
          })}
        </div>
      </header>

      <div
        className={styles.body}
        role="tabpanel"
        id={`nex-v2-tabpanel-${activeTab}`}
        aria-labelledby={`nex-v2-tab-${activeTab}`}
      >
        {activeTab === 'chat' && (
          <NexChatTab
            projectId={hudState?.project_id ?? ''}
            privacyMode={privacyMode}
            onConfirmProposal={handleConfirmProposal}
            onSendingChange={onSendingChange}
            onReplyResolved={onReplyResolved}
            onOpenResources={() => onSelectTab('resources')}
            onOpenSettings={onOpenSettings}
          />
        )}

        {activeTab === 'resources' && (
          <NexResourcesTab
            hudState={hudState}
            modelWatch={modelWatch}
            offline={offline}
            privacyMode={privacyMode}
          />
        )}

        {activeTab === 'attention' && (
          <NexAttentionTab
            hudState={hudState}
            events={events}
            privacyMode={privacyMode}
            onAckAlert={onAckAlert}
            onSnoozeAlert={onSnoozeAlert}
          />
        )}
      </div>
    </section>
  );
};

export default NexInteractionPanel;
