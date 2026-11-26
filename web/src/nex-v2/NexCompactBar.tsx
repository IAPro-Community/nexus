import React from 'react';
import { useTranslation } from 'react-i18next';
import { BellOff, Settings } from 'lucide-react';
import type { NexOperationalMood } from './motion/nexBehaviorState';
import { countAttentionItems, hasCriticalAttention } from './nexV2HudHelpers';
import type { NexV2HudState } from './nexV2Types';
import styles from './NexCompactBar.module.scss';

export interface NexCompactBarProps {
  mood: NexOperationalMood;
  hudState: NexV2HudState | null;
  offline: boolean;
  doNotDisturb: boolean;
  panelOpen: boolean;
  onTogglePanel: () => void;
  onOpenAttention?: () => void;
  onOpenSettings: () => void;
}

export function resolveCompactQuotaLabel(hudState: NexV2HudState | null, offline: boolean): string {
  if (offline || !hudState) {
    return 'UNKNOWN';
  }
  const primaryProvider = hudState.providers.find(
    (p) => p.limiting_window && typeof p.limiting_window.remaining_percent === 'number',
  );
  if (primaryProvider?.limiting_window?.remaining_percent !== undefined) {
    return `${Math.round(primaryProvider.limiting_window.remaining_percent)}%`;
  }
  for (const provider of hudState.providers) {
    const win = provider.windows.find((w) => typeof w.remaining_percent === 'number');
    if (win && win.remaining_percent !== undefined) {
      return `${Math.round(win.remaining_percent)}%`;
    }
  }
  return 'UNKNOWN';
}

export const NexCompactBar: React.FC<NexCompactBarProps> = ({
  mood,
  hudState,
  offline,
  doNotDisturb,
  panelOpen,
  onTogglePanel,
  onOpenAttention,
  onOpenSettings,
}) => {
  const { t } = useTranslation();
  const quotaCode = resolveCompactQuotaLabel(hudState, offline);
  const quotaKnown = quotaCode !== 'UNKNOWN';
  const quotaDisplay = quotaKnown ? quotaCode : t('nexV2.resourcesStatusUnknown');
  const attentionCount = countAttentionItems(hudState);
  const critical = hasCriticalAttention(hudState);

  return (
    <nav className={styles.bar} data-mood={mood} aria-label={t('nexV2.compactBarNavLabel')}>
      <span className={styles.statusDot} aria-hidden="true" />
      <span className={styles.quota} data-quota-state={quotaKnown ? 'KNOWN' : 'UNKNOWN'}>
        {t('nexV2.compactQuotaLabel')}: {quotaDisplay}
      </span>

      {attentionCount > 0 && (
        <button
          type="button"
          className={styles.alertPill}
          data-severity={critical ? 'CRITICAL' : 'WARNING'}
          onClick={onOpenAttention ?? onTogglePanel}
        >
          {t('nexV2.compactAttentionBadge', { count: attentionCount })}
        </button>
      )}

      {doNotDisturb && (
        <span className={styles.badge} title={t('nexV2.doNotDisturbLabel')}>
          <BellOff size={12} aria-hidden="true" />
          {t('nexV2.dndActiveBadge')}
        </span>
      )}

      <div className={styles.actions}>
        <button
          type="button"
          className={styles.actionBtn}
          aria-expanded={panelOpen}
          data-nex-compact-open
          onClick={onTogglePanel}
        >
          {panelOpen ? t('nexV2.closePanelLabel') : t('nexV2.bubbleOpen')}
        </button>
        <button
          type="button"
          className={styles.iconBtn}
          aria-label={t('nexV2.openSettingsLabel')}
          data-nex-compact-settings
          onClick={onOpenSettings}
        >
          <Settings size={14} aria-hidden="true" />
        </button>
      </div>
    </nav>
  );
};
