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
    BAR: { label: 'hud.mascot.modeBar', title: 'hud.mascot.modeBarTitle' },
    MASCOT: {
      label: 'hud.mascot.modeMascot',
      title: 'hud.mascot.modeMascotTitle',
    },
    TRAY: { label: 'hud.mascot.modeTray', title: 'hud.mascot.modeTrayTitle' },
  } as const;

  return (
    <div
      className={styles.guardianModeSelector}
      role="group"
      aria-label={t('hud.mascot.modeGroupLabel')}
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
