import React from 'react';
import { useTranslation } from 'react-i18next';
import { ArrowLeft, X } from 'lucide-react';
import type {
  NexV2NotificationCooldownSec,
  NexV2NotificationLevel,
  NexV2PlatformCapabilities,
  NexV2PresenceState,
  NexV2PresetId,
} from './nexV2Types';
import styles from './NexSettingsDrawer.module.scss';

export interface NexSettingsDrawerProps {
  open: boolean;
  canGoBack?: boolean;
  state: NexV2PresenceState;
  capabilities: NexV2PlatformCapabilities;
  onClose: () => void;
  onPresetChange: (presetId: NexV2PresetId) => void;
  onDisplayModeChange?: (showAvatar: boolean, compactBar: boolean) => void;
  onToggleShowAvatar?: (visible: boolean) => void;
  onToggleCompactBar?: (enabled: boolean) => void;
  onToggleAlwaysOnTop: (enabled: boolean) => void;
  onToggleDoNotDisturb: (enabled: boolean) => void;
  onTogglePrivacyMode?: (enabled: boolean) => void;
  onTogglePaused: (paused: boolean) => void;
  onToggleReducedMotion: (enabled: boolean) => void;
  onNotificationLevelChange?: (level: NexV2NotificationLevel) => void;
  onNotificationCooldownChange?: (seconds: NexV2NotificationCooldownSec) => void;
  onToggleHudMemoryMode?: (onDemand: boolean) => void;
}

type DisplayMode = 'avatar' | 'bar' | 'both';

const NOTIFICATION_LEVELS: readonly { id: NexV2NotificationLevel; labelKey: string }[] = [
  { id: 'all', labelKey: 'nexV2.notifLevelAll' },
  { id: 'important', labelKey: 'nexV2.notifLevelImportant' },
  { id: 'critical', labelKey: 'nexV2.notifLevelCritical' },
  { id: 'quiet', labelKey: 'nexV2.notifLevelQuiet' },
];

const NOTIFICATION_COOLDOWNS: readonly {
  seconds: NexV2NotificationCooldownSec;
  labelKey: string;
}[] = [
  { seconds: 0, labelKey: 'nexV2.cooldownImmediate' },
  { seconds: 30, labelKey: 'nexV2.cooldown30s' },
  { seconds: 120, labelKey: 'nexV2.cooldown2m' },
  { seconds: 300, labelKey: 'nexV2.cooldown5m' },
];

const PRESETS: readonly { id: NexV2PresetId; labelKey: string }[] = [
  { id: 'compact', labelKey: 'nexV2.presetCompactShort' },
  { id: 'standard', labelKey: 'nexV2.presetStandardShort' },
  { id: 'large', labelKey: 'nexV2.presetLargeShort' },
];

const DISPLAY_MODES: readonly {
  id: DisplayMode;
  labelKey: string;
  showAvatar: boolean;
  compactBar: boolean;
}[] = [
  { id: 'avatar', labelKey: 'nexV2.displayModeAvatar', showAvatar: true, compactBar: false },
  { id: 'bar', labelKey: 'nexV2.displayModeBar', showAvatar: false, compactBar: true },
  { id: 'both', labelKey: 'nexV2.displayModeBoth', showAvatar: true, compactBar: true },
];

function resolveCurrentDisplayMode(state: NexV2PresenceState): DisplayMode {
  if (state.showAvatar && state.compactBar) return 'both';
  if (!state.showAvatar) return 'bar';
  return 'avatar';
}

export const NexSettingsDrawer: React.FC<NexSettingsDrawerProps> = ({
  open,
  canGoBack = false,
  state,
  capabilities,
  onClose,
  onPresetChange,
  onDisplayModeChange,
  onToggleShowAvatar,
  onToggleCompactBar,
  onToggleAlwaysOnTop,
  onToggleDoNotDisturb,
  onTogglePrivacyMode,
  onTogglePaused,
  onToggleReducedMotion,
  onNotificationLevelChange,
  onNotificationCooldownChange,
  onToggleHudMemoryMode,
}) => {
  const { t } = useTranslation();

  if (!open) {
    return null;
  }

  const currentMode = resolveCurrentDisplayMode(state);
  const isWayland = !capabilities.globalPositioning || capabilities.session === 'wayland';

  const handleSelectMode = (showAvatar: boolean, compactBar: boolean) => {
    if (onDisplayModeChange) {
      onDisplayModeChange(showAvatar, compactBar);
      return;
    }
    if (showAvatar !== state.showAvatar) onToggleShowAvatar?.(showAvatar);
    if (compactBar !== state.compactBar) onToggleCompactBar?.(compactBar);
  };

  const behaviorRows: readonly {
    key: string;
    label: string;
    desc: string;
    checked: boolean;
    onToggle: () => void;
  }[] = [
    {
      key: 'always-on-top',
      label: t('nexV2.alwaysOnTopLabel'),
      desc: t('nexV2.alwaysOnTopDesc'),
      checked: state.placement.alwaysOnTop,
      onToggle: () => onToggleAlwaysOnTop(!state.placement.alwaysOnTop),
    },
    {
      key: 'dnd',
      label: t('nexV2.doNotDisturbLabel'),
      desc: t('nexV2.doNotDisturbDesc'),
      checked: state.doNotDisturb,
      onToggle: () => onToggleDoNotDisturb(!state.doNotDisturb),
    },
    {
      key: 'privacy-mode',
      label: t('nexV2.privacyModeLabel'),
      desc: t('nexV2.privacyModeDesc'),
      checked: state.privacyMode,
      onToggle: () => onTogglePrivacyMode?.(!state.privacyMode),
    },
    {
      key: 'paused',
      label: t('nexV2.pauseMotionLabel'),
      desc: t('nexV2.pauseMotionDesc'),
      checked: state.paused,
      onToggle: () => onTogglePaused(!state.paused),
    },
    {
      key: 'reduced-motion',
      label: t('nexV2.reducedMotionLabel'),
      desc: t('nexV2.reducedMotionDesc'),
      checked: state.reducedMotion,
      onToggle: () => onToggleReducedMotion(!state.reducedMotion),
    },
    {
      key: 'hud-ondemand',
      label: t('nexV2.hudOnDemandLabel'),
      desc: t('nexV2.hudOnDemandDesc'),
      checked: state.hudMemoryMode === 'ondemand',
      onToggle: () => onToggleHudMemoryMode?.(state.hudMemoryMode !== 'ondemand'),
    },
  ];

  return (
    <section
      className={styles.drawer}
      role="dialog"
      aria-modal="false"
      aria-label={t('nexV2.settingsTitle')}
      data-nex-settings
    >
      <header className={styles.header}>
        <div className={styles.titleRow}>
          {canGoBack && (
            <button
              type="button"
              className={styles.iconBtn}
              aria-label={t('nexV2.settingsBackLabel')}
              data-nex-settings-back
              onClick={onClose}
            >
              <ArrowLeft size={14} aria-hidden="true" />
            </button>
          )}
          <h2 className={styles.title}>{t('nexV2.settingsTitle')}</h2>
        </div>
        <button
          type="button"
          className={styles.iconBtn}
          aria-label={t('nexV2.closeSettingsLabel')}
          data-nex-settings-close
          onClick={onClose}
        >
          <X size={14} aria-hidden="true" />
        </button>
      </header>

      <div className={styles.body}>
        <section className={styles.section}>
          <h3 className={styles.sectionTitle}>{t('nexV2.settingsSectionDisplay')}</h3>

          <div className={styles.fieldRow}>
            <span className={styles.fieldLabel}>{t('nexV2.displayModeLabel')}</span>
            <div className={styles.segmented} role="group" aria-label={t('nexV2.displayModeLabel')}>
              {DISPLAY_MODES.map((mode) => {
                const active = currentMode === mode.id;
                return (
                  <button
                    key={mode.id}
                    type="button"
                    className={styles.segmentBtn}
                    aria-pressed={active}
                    data-nex-display-mode={mode.id}
                    onClick={() => handleSelectMode(mode.showAvatar, mode.compactBar)}
                  >
                    {t(mode.labelKey)}
                  </button>
                );
              })}
            </div>
          </div>

          <div className={styles.fieldRow}>
            <span className={styles.fieldLabel}>{t('nexV2.presetLabel')}</span>
            <div className={styles.segmented} role="group" aria-label={t('nexV2.presetLabel')}>
              {PRESETS.map((preset) => {
                const active = state.placement.presetId === preset.id;
                return (
                  <button
                    key={preset.id}
                    type="button"
                    className={styles.segmentBtn}
                    disabled={!state.showAvatar}
                    aria-pressed={active}
                    data-nex-preset={preset.id}
                    onClick={() => onPresetChange(preset.id)}
                  >
                    {t(preset.labelKey)}
                  </button>
                );
              })}
            </div>
          </div>
        </section>

        <section className={styles.section}>
          <h3 className={styles.sectionTitle}>{t('nexV2.settingsSectionNotifications')}</h3>
          <div className={styles.fieldRow}>
            <span className={styles.fieldLabel}>{t('nexV2.notifLevelLabel')}</span>
            <div
              className={`${styles.segmented} ${styles.segmented4}`}
              role="group"
              aria-label={t('nexV2.notifLevelLabel')}
            >
              {NOTIFICATION_LEVELS.map((opt) => {
                const active = state.notificationLevel === opt.id;
                return (
                  <button
                    key={opt.id}
                    type="button"
                    className={styles.segmentBtn}
                    aria-pressed={active}
                    data-nex-notif-level={opt.id}
                    onClick={() => onNotificationLevelChange?.(opt.id)}
                  >
                    {t(opt.labelKey)}
                  </button>
                );
              })}
            </div>
          </div>

          <div className={styles.fieldRow}>
            <span className={styles.fieldLabel}>{t('nexV2.cooldownLabel')}</span>
            <div
              className={`${styles.segmented} ${styles.segmented4}`}
              role="group"
              aria-label={t('nexV2.cooldownLabel')}
            >
              {NOTIFICATION_COOLDOWNS.map((opt) => {
                const active = state.notificationCooldownSec === opt.seconds;
                return (
                  <button
                    key={opt.seconds}
                    type="button"
                    className={styles.segmentBtn}
                    aria-pressed={active}
                    data-nex-notif-cooldown={opt.seconds}
                    onClick={() => onNotificationCooldownChange?.(opt.seconds)}
                  >
                    {t(opt.labelKey)}
                  </button>
                );
              })}
            </div>
          </div>
        </section>

        <section className={styles.section}>
          <h3 className={styles.sectionTitle}>{t('nexV2.settingsSectionBehavior')}</h3>
          <div className={styles.switchList}>
            {behaviorRows.map((row) => (
              <div key={row.key} className={styles.switchRow}>
                <div className={styles.switchCopy}>
                  <span className={styles.switchLabel}>{row.label}</span>
                  <span className={styles.switchDesc}>{row.desc}</span>
                </div>
                <button
                  type="button"
                  role="switch"
                  aria-label={row.label}
                  aria-checked={row.checked}
                  data-nex-toggle={row.key}
                  className={styles.switchTrack}
                  onClick={row.onToggle}
                >
                  <span className={styles.switchThumb} aria-hidden="true" />
                </button>
              </div>
            ))}
          </div>
        </section>

        <section className={styles.capabilities}>
          <strong className={styles.capTitle}>
            {t('nexV2.capabilityNoticeTitle', { platform: capabilities.platform })}
          </strong>
          {isWayland && <p className={styles.capText}>{t('nexV2.capabilityWaylandNotice')}</p>}
          <p className={styles.capText}>
            {t('nexV2.capabilityHotkeyNotice', { hotkey: state.hotkey })} ·{' '}
            {isWayland ? t('nexV2.capabilityHotkeyWayland') : t('nexV2.capabilityHotkeyDefault')}
          </p>
        </section>
      </div>
    </section>
  );
};
