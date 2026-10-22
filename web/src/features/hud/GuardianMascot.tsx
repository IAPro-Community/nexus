import React from 'react';
import type { GuardianMood, HudAppearance } from './guardianModel';
import styles from './NexusHud.module.scss';

interface GuardianMascotProps {
  mood: GuardianMood;
  attention: boolean;
}

/**
 * Original CSS-animated 2D pixel Guardian. The shapes are vector pixels:
 * crisp at every display scale, no network asset, WebGL or video decoder.
 * The animation is presentation-only and never fabricates telemetry.
 */
export const GuardianMascot: React.FC<GuardianMascotProps> = ({ mood, attention }) => (
  <span className={styles.guardian} data-mood={mood} aria-hidden="true">
    <svg className={styles.guardianSprite} viewBox="0 0 64 64" shapeRendering="crispEdges">
      <g className={styles.guardianAntenna}>
        <rect x="30" y="3" width="4" height="9" fill="#8396d6" />
        <rect x="26" y="1" width="12" height="6" fill="#5a86e3" />
        <rect x="29" y="2" width="6" height="3" fill="#c9e8ff" />
      </g>
      <g className={styles.guardianBody}>
        <rect x="14" y="16" width="36" height="37" fill="#6f9ff0" />
        <rect x="10" y="20" width="44" height="28" fill="#4467ae" />
        <rect x="14" y="15" width="36" height="32" fill="#c1d6ff" />
        <rect x="17" y="18" width="30" height="28" fill="#38578e" />
        <rect x="17" y="21" width="30" height="19" fill="#132c50" />
        <rect x="18" y="43" width="28" height="4" fill="#7ea9fb" />
        <rect x="23" y="48" width="18" height="5" fill="#5274ba" />
        <rect x="14" y="52" width="13" height="6" fill="#3e5b98" />
        <rect x="37" y="52" width="13" height="6" fill="#3e5b98" />
        <rect x="16" y="49" width="10" height="4" fill="#9dbeff" />
        <rect x="38" y="49" width="10" height="4" fill="#9dbeff" />
        <g className={styles.guardianEyes}>
          <rect x="23" y="26" width="6" height="8" fill="#8df8e6" />
          <rect x="35" y="26" width="6" height="8" fill="#8df8e6" />
          <rect x="25" y="27" width="2" height="4" fill="#eefeff" />
          <rect x="37" y="27" width="2" height="4" fill="#eefeff" />
        </g>
        <rect x="29" y="37" width="6" height="2" fill="#7ba2e0" />
        <rect x="11" y="32" width="5" height="13" fill="#6588cc" />
        <rect x="48" y="32" width="5" height="13" fill="#6588cc" />
        <rect x="8" y="39" width="5" height="7" fill="#4767a7" />
        <rect x="51" y="39" width="5" height="7" fill="#4767a7" />
      </g>
      <g className={styles.guardianSignal}>
        <rect x="49" y="11" width="7" height="7" fill="#78ebbe" />
        <rect x="51" y="13" width="3" height="3" fill="#e4fff4" />
      </g>
    </svg>
    {attention && <span className={styles.guardianBubble}>Atenção!</span>}
  </span>
);

interface GuardianModeSelectorProps {
  appearance: HudAppearance;
  detachedAvailable: boolean;
  pending: boolean;
  onSelect: (next: HudAppearance) => void;
}

/** Tiny, keyboard-accessible controls that also fit a 334 DIP native popup. */
export const GuardianModeSelector: React.FC<GuardianModeSelectorProps> = ({
  appearance,
  detachedAvailable,
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
              ? detachedAvailable
                ? 'Exibir mascote flutuante'
                : 'Ative "Popup independente" nas configurações do HUD para usar o mascote'
              : 'Ocultar na bandeja (recuperável no ícone do Nexus)'
        }
        disabled={pending || (mode === 'MASCOT' && !detachedAvailable)}
        onClick={() => onSelect(mode)}
      >
        {mode === 'BAR' ? 'Barra' : mode === 'MASCOT' ? 'Mascote' : 'Bandeja'}
      </button>
    ))}
  </div>
);
