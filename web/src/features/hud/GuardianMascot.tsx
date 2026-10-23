import React from 'react';
import type { GuardianMood, HudAppearance } from './guardianModel';
import styles from './NexusHud.module.scss';
import { NexPixelSprite } from './NexPixelSprite';

interface GuardianMascotProps {
  mood: GuardianMood;
  attention: boolean;
}

/**
 * Canonical NEX 16-bit sprite, preserving the existing HUD projection contract:
 * crisp at every display scale, no network asset, WebGL or video decoder.
 * The animation is presentation-only and never fabricates telemetry.
 */
export const GuardianMascot: React.FC<GuardianMascotProps> = ({ mood, attention }) => (
  <span className={styles.guardian} data-mood={mood} aria-hidden="true">
    <NexPixelSprite mood={mood} />
    {attention && <span className={styles.guardianBubble}>Atenção!</span>}
  </span>
);

interface GuardianModeSelectorProps {
  appearance: HudAppearance;
  pending: boolean;
  onSelect: (next: HudAppearance) => void;
}

/** Tiny, keyboard-accessible controls that also fit a 334 DIP native popup. */
export const GuardianModeSelector: React.FC<GuardianModeSelectorProps> = ({
  appearance,
  pending,
  onSelect,
}) => (
  <div
    className={styles.guardianModeSelector}
    role="group"
    aria-label="Apresentação nativa do Guardian"
  >
    {(['BAR', 'MASCOT', 'TRAY'] as const).map((mode) => (
      <button
        key={mode}
        type="button"
        className={styles.guardianModeButton}
        data-active={appearance === mode ? 'true' : 'false'}
        aria-pressed={appearance === mode}
        title={
          mode === 'BAR'
            ? 'Exibir barra'
            : mode === 'MASCOT'
              ? 'Exibir mascote flutuante'
              : 'Ocultar na bandeja (recuperável no ícone do Nexus)'
        }
        disabled={pending}
        onClick={() => onSelect(mode)}
      >
        {mode === 'BAR' ? 'Barra' : mode === 'MASCOT' ? 'Mascote' : 'Bandeja'}
      </button>
    ))}
  </div>
);
