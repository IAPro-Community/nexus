import React from 'react';
import { useTranslation } from 'react-i18next';
import { BellOff, Settings } from 'lucide-react';
import type { NexOperationalMood } from './motion/nexBehaviorState';
import type { NexV2HudState } from './nexV2Types';
import styles from './NexCompactBar.module.scss';

export interface NexCompactBarProps {
  mood: NexOperationalMood;
  hudState: NexV2HudState | null;
  offline: boolean;
  doNotDisturb: boolean;
  panelOpen: boolean;
  onTogglePanel: () => void;
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
  onOpenSettings,
}) => {
  const { t } = useTranslation();
  const quotaText = resolveCompactQuotaLabel(hudState, offline);
  const activeAlerts = hudState ? hudState.alerts.filter((a) => !a.acked).length : 0;

  return (
    <nav className={styles.bar} data-mood={mood} aria-label={t('nexV2.compactBarNavLabel')}>
      <span className={styles.statusDot} aria-hidden="true" />
      <span className={styles.quota}>
        {t('nexV2.compactQuotaLabel')}: {quotaText}
      </span>
      {activeAlerts > 0 && (
        <span className={styles.badge}>
          {t('nexV2.compactAlertsCount', { count: activeAlerts })}
        </span>
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
          {panelOpen ? t('nexV2.closePanelLabel') : t('nexV2.tabChat')}
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
