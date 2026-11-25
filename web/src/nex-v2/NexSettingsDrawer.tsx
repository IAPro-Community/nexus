import React from 'react';
import { useTranslation } from 'react-i18next';
import { X } from 'lucide-react';
import type { NexV2PlatformCapabilities, NexV2PresenceState, NexV2PresetId } from './nexV2Types';
import styles from './NexSettingsDrawer.module.scss';

export interface NexSettingsDrawerProps {
  open: boolean;
  state: NexV2PresenceState;
  capabilities: NexV2PlatformCapabilities;
  onClose: () => void;
  onPresetChange: (presetId: NexV2PresetId) => void;
  onToggleShowAvatar: (visible: boolean) => void;
  onToggleCompactBar: (enabled: boolean) => void;
  onToggleAlwaysOnTop: (enabled: boolean) => void;
  onToggleDoNotDisturb: (enabled: boolean) => void;
  onTogglePaused: (paused: boolean) => void;
  onToggleReducedMotion: (enabled: boolean) => void;
}

const PRESETS: readonly { id: NexV2PresetId; labelKey: string }[] = [
  { id: 'compact', labelKey: 'nexV2.presetCompact' },
  { id: 'standard', labelKey: 'nexV2.presetStandard' },
  { id: 'large', labelKey: 'nexV2.presetLarge' },
];

export const NexSettingsDrawer: React.FC<NexSettingsDrawerProps> = ({
  open,
  state,
  capabilities,
  onClose,
  onPresetChange,
  onToggleShowAvatar,
  onToggleCompactBar,
  onToggleAlwaysOnTop,
  onToggleDoNotDisturb,
  onTogglePaused,
  onToggleReducedMotion,
}) => {
  const { t } = useTranslation();

  if (!open) {
    return null;
  }

  return (
    <section
      className={styles.drawer}
      role="dialog"
      aria-modal="false"
      aria-label={t('nexV2.settingsTitle')}
      data-nex-settings
    >
      <header className={styles.header}>
        <h2 className={styles.title}>{t('nexV2.settingsTitle')}</h2>
        <button
          type="button"
          className={styles.closeBtn}
          aria-label={t('nexV2.closeSettingsLabel')}
          data-nex-settings-close
          onClick={onClose}
        >
          <X size={14} aria-hidden="true" />
        </button>
      </header>

      <div className={styles.group}>
        <label className={styles.selectLabel} htmlFor="nex-v2-preset-select">
          <span>{t('nexV2.presetLabel')}</span>
          <select
            id="nex-v2-preset-select"
            className={styles.select}
            value={state.placement.presetId}
            onChange={(event) => onPresetChange(event.target.value as NexV2PresetId)}
          >
            {PRESETS.map((preset) => (
              <option key={preset.id} value={preset.id}>
                {t(preset.labelKey)}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className={styles.toggles}>
        <button
          type="button"
          role="switch"
          aria-checked={state.showAvatar}
          data-nex-toggle="show-avatar"
          className={styles.toggleBtn}
          onClick={() => onToggleShowAvatar(!state.showAvatar)}
        >
          <span>{t('nexV2.showAvatarLabel')}</span>
          <span className={styles.toggleState}>{state.showAvatar ? 'ON' : 'OFF'}</span>
        </button>

        <button
          type="button"
          role="switch"
          aria-checked={state.compactBar}
          data-nex-toggle="compact-bar"
          className={styles.toggleBtn}
          onClick={() => onToggleCompactBar(!state.compactBar)}
        >
          <span>{t('nexV2.compactBarLabel')}</span>
          <span className={styles.toggleState}>{state.compactBar ? 'ON' : 'OFF'}</span>
        </button>

        <button
          type="button"
          role="switch"
          aria-checked={state.placement.alwaysOnTop}
          data-nex-toggle="always-on-top"
          className={styles.toggleBtn}
          onClick={() => onToggleAlwaysOnTop(!state.placement.alwaysOnTop)}
        >
          <span>{t('nexV2.alwaysOnTopLabel')}</span>
          <span className={styles.toggleState}>{state.placement.alwaysOnTop ? 'ON' : 'OFF'}</span>
        </button>

        <button
          type="button"
          role="switch"
          aria-checked={state.doNotDisturb}
          data-nex-toggle="dnd"
          className={styles.toggleBtn}
          onClick={() => onToggleDoNotDisturb(!state.doNotDisturb)}
        >
          <span>{t('nexV2.doNotDisturbLabel')}</span>
          <span className={styles.toggleState}>{state.doNotDisturb ? 'ON' : 'OFF'}</span>
        </button>

        <button
          type="button"
          role="switch"
          aria-checked={state.paused}
          data-nex-toggle="paused"
          className={styles.toggleBtn}
          onClick={() => onTogglePaused(!state.paused)}
        >
          <span>{t('nexV2.pauseMotionLabel')}</span>
          <span className={styles.toggleState}>{state.paused ? 'ON' : 'OFF'}</span>
        </button>

        <button
          type="button"
          role="switch"
          aria-checked={state.reducedMotion}
          data-nex-toggle="reduced-motion"
          className={styles.toggleBtn}
          onClick={() => onToggleReducedMotion(!state.reducedMotion)}
        >
          <span>{t('nexV2.reducedMotionLabel')}</span>
          <span className={styles.toggleState}>{state.reducedMotion ? 'ON' : 'OFF'}</span>
        </button>
      </div>

      <div className={styles.capabilities}>
        <strong className={styles.capTitle}>
          {t('nexV2.capabilityNoticeTitle', { platform: capabilities.platform })}
        </strong>
        {(!capabilities.globalPositioning || capabilities.session === 'wayland') && (
          <p className={styles.capText}>{t('nexV2.capabilityWaylandNotice')}</p>
        )}
        <p className={styles.capText}>
          {t('nexV2.capabilityHotkeyNotice', { hotkey: state.hotkey })}
          {capabilities.globalHotkeyReason ? ` · ${capabilities.globalHotkeyReason}` : ''}
        </p>
      </div>
    </section>
  );
};
