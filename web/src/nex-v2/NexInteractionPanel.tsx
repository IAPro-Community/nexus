import React, { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Settings, X } from 'lucide-react';
import type {
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
}

const TAB_ORDER: readonly NexV2TabId[] = ['chat', 'resources', 'attention'];

export const NexInteractionPanel: React.FC<NexInteractionPanelProps> = ({
  activeTab,
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
}) => {
  const { t } = useTranslation();
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
          <NexChatTab projectId={hudState?.project_id ?? ''} privacyMode={privacyMode} />
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
