import React from 'react';
import { useTranslation } from 'react-i18next';
import type { HudAppearance } from './guardianModel';
import styles from './NexusHud.module.scss';

interface GuardianModeSelectorProps {
  appearance: HudAppearance;
  pending: boolean;
  onSelect: (next: HudAppearance) => void;
}

/** Accessible controls for the HUD's three presentations. */
export const GuardianModeSelector: React.FC<GuardianModeSelectorProps> = ({
  appearance,
  pending,
  onSelect,
}) => {
  const { t } = useTranslation();
  const modes = {
    BAR: { label: 'hud.companion.modeBar', title: 'hud.companion.modeBarTitle' },
    MASCOT: {
      label: 'hud.companion.modeMascot',
      title: 'hud.companion.modeMascotTitle',
    },
    TRAY: { label: 'hud.companion.modeTray', title: 'hud.companion.modeTrayTitle' },
  } as const;

  return (
    <div
      className={styles.guardianModeSelector}
      role="group"
      aria-label={t('hud.companion.modeGroupLabel')}
    >
      {(['BAR', 'MASCOT', 'TRAY'] as const).map((mode) => (
        <button
          key={mode}
          type="button"
          className={styles.guardianModeButton}
          data-active={appearance === mode ? 'true' : 'false'}
          aria-pressed={appearance === mode}
          title={t(modes[mode].title)}
          disabled={pending}
          onClick={() => onSelect(mode)}
        >
          {t(modes[mode].label)}
        </button>
      ))}
    </div>
  );
};
