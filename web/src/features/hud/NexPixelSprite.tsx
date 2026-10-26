import React from 'react';
import type { GuardianMood } from './guardianModel';
import styles from './NexusHud.module.scss';

type NexPixelSpriteProps = {
  mood: GuardianMood;
};

const MOOD_COLORS: Record<GuardianMood, { accent: string; accentDark: string; glint: string }> = {
  idle: { accent: '#31D5FF', accentDark: '#1460BF', glint: '#DFFFFF' },
  working: { accent: '#31D5FF', accentDark: '#1460BF', glint: '#DFFFFF' },
  warning: { accent: '#FF8A45', accentDark: '#A83D12', glint: '#FFE2A3' },
  critical: { accent: '#FF496C', accentDark: '#9E1838', glint: '#FFD0DA' },
  offline: { accent: '#71839D', accentDark: '#3E4D61', glint: '#A5B4C8' },
};

/**
 * NEX — official, front-facing 32-bit-inspired vector mascot.
 *
 * A single authored 128x128 SVG master for native HUD usage.
 * Reflects live HUD states without turning mission completion into a success signal.
 * High-precision vector curves based directly on nex-character-reference.svg.
 * All anatomy remains stable across states. No network, GIF, WebGL, provider
 * probing or second state machine.
 */
export const NexPixelSprite: React.FC<NexPixelSpriteProps> = ({ mood }) => {
  const gradientPrefix = React.useId().replace(/:/g, '_');
  const shell = `url(#${gradientPrefix}-shell)`;
  const glass = `url(#${gradientPrefix}-glass)`;
  const coreGlow = `url(#${gradientPrefix}-core)`;
  const offline = mood === 'offline';
  const warning = mood === 'warning';
  const critical = mood === 'critical';
  const { accent, accentDark, glint } = MOOD_COLORS[mood];

  return (
    <svg
      className={styles.guardianSprite}
      viewBox="0 0 128 128"
      shapeRendering="geometricPrecision"
      data-nex-art="32bit-official"
      data-nex-facing="front"
      role="presentation"
      focusable="false"
    >
      <defs>
        <linearGradient id={`${gradientPrefix}-shell`} x1="0" x2="0.83" y1="0" y2="1">
          <stop offset="0%" stopColor="#FFFFFF" />
          <stop offset="48%" stopColor="#DFECFC" />
          <stop offset="100%" stopColor="#7499CB" />
        </linearGradient>
        <linearGradient id={`${gradientPrefix}-glass`} x1="0" x2="0.8" y1="0" y2="1">
          <stop offset="0%" stopColor="#1E4778" />
          <stop offset="30%" stopColor="#0A1830" />
          <stop offset="100%" stopColor="#030B21" />
        </linearGradient>
        <radialGradient id={`${gradientPrefix}-core`} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor={glint} />
          <stop offset="42%" stopColor={accent} />
          <stop offset="100%" stopColor={accentDark} />
        </radialGradient>
      </defs>

      {/* Recessed ground hover shadow */}
      <g className={styles.nexShadow} data-nex-part="shadow">
        <ellipse cx="64" cy="123" rx="28" ry="4" fill="#041224" opacity=".55" />
        <ellipse cx="64" cy="123" rx="20" ry="2.5" fill="#092B56" opacity=".7" />
        <ellipse cx="64" cy="123" rx="12" ry="1.5" fill={accent} opacity=".8" />
        <ellipse cx="64" cy="123" rx="6" ry="0.8" fill={glint} opacity=".95" />
      </g>

      {/* Energy orbit behind the body */}
      <g className={styles.guardianSignal} data-nex-part="orbit">
        <ellipse
          cx="64"
          cy="100"
          rx="56"
          ry="15"
          fill="none"
          stroke="#1C61BF"
          strokeWidth="3.5"
          opacity=".35"
        />
        <path d="M8 100q56-30 112 0" fill="none" stroke="#1B70E7" strokeWidth="1.8" />
      </g>

      {/* Independent mirrored fins share a fixed frontal anatomy. */}
      <g data-nex-part="fins">
        <g className={styles.nexFinLeft} data-nex-fin="left">
          <path
            d="M31 34L33 8l5 0 9 26z"
            fill="#112C5D"
            stroke="#4E92F3"
            strokeWidth="2"
          />
          <path d="M35 30l1-19 7 19z" fill={accent} opacity=".92" />
          <path d="M37 20l1-8 2 8z" fill={glint} opacity=".95" />
        </g>
        <g className={styles.nexFinRight} data-nex-fin="right">
          <path
            d="M97 34l-9-26h-5l2 26z"
            fill="#112C5D"
            stroke="#4E92F3"
            strokeWidth="2"
          />
          <path d="M93 30l-7-19 1 19z" fill={accent} opacity=".92" />
          <path d="M91 20l-2-8 1 8z" fill={glint} opacity=".95" />
        </g>
      </g>

      {/* Compact floating robot torso */}
      <g className={styles.guardianBody} data-nex-part="torso">
        <path
          d="M49 81h30l10 10-4 19-12 9h-18l-12-9-4-19z"
          fill="#091A39"
          stroke="#0C2E61"
          strokeWidth="2"
        />
        <path
          d="M51 83h26l9 8-3 16-10 8h-16l-10-8-3-16z"
          fill={shell}
          stroke="#A5C4EC"
          strokeWidth="1.5"
        />
        <path d="M57 84h14l4 7h-22z" fill="#F8FCFF" />
        <path d="M59 93h10v1H59z" fill="#8AAED6" opacity=".7" />
      </g>

      {/* Central circular reactor core with glowing emitter */}
      <g className={styles.nexReactor} data-nex-part="reactor">
        <circle cx="64" cy="99" r="13" fill="#071B3D" stroke="#3B72E1" strokeWidth="2.8" />
        <circle cx="64" cy="99" r="9.5" fill={coreGlow} stroke="#B4E9FF" strokeWidth="2" />
        <circle cx="61.5" cy="96.5" r="3" fill="#F0FFFF" opacity=".92" />
        <path d="M63.2 94.5h1.6v9h-1.6z M59.5 98.2h9v1.6h-9z" fill={glint} opacity=".85" />
      </g>

      {/* Concentric circular ear couplings */}
      <g data-nex-part="ear-couplings">
        <g transform="translate(21 54)">
          <circle cx="0" cy="0" r="14" fill="#102F65" stroke="#DCE9FB" strokeWidth="4" />
          <circle cx="0" cy="0" r="9.5" fill="#15498D" stroke="#5284CB" strokeWidth="2.2" />
          <circle cx="0" cy="0" r="5.5" fill={coreGlow} />
          <circle cx="-1.5" cy="-1.5" r="1.8" fill={glint} opacity=".9" />
        </g>
        <g transform="translate(107 54)">
          <circle cx="0" cy="0" r="14" fill="#102F65" stroke="#DCE9FB" strokeWidth="4" />
          <circle cx="0" cy="0" r="9.5" fill="#15498D" stroke="#5284CB" strokeWidth="2.2" />
          <circle cx="0" cy="0" r="5.5" fill={coreGlow} />
          <circle cx="-1.5" cy="-1.5" r="1.8" fill={glint} opacity=".9" />
        </g>
      </g>

      {/* Canonical big rounded helmet and dark glass visor */}
      <g data-nex-part="helmet">
        <path
          d="M40 24q24-11 48 0l11 12 5 20-4 19-14 12q-22 7-44 0l-14-12-4-19 5-20z"
          fill={shell}
          stroke="#0C2955"
          strokeWidth="3"
        />
        <path
          d="M45 32q19-7 38 0l8 8 4 16-4 16-10 7q-18 4-36 0l-10-7-4-16 4-16z"
          fill={glass}
          stroke="#5C85BA"
          strokeWidth="2.5"
        />
        <path d="M46 26q18-8 36 0l5 5h-46z" fill="#FFFFFF" opacity=".7" />
        <path d="M43 36q13-8 26-6l-7 3q-9-1-19 5z" fill="#91B6E8" opacity=".38" />
        <path d="M77 31l5 2 3 4-4-1z" fill="#A6DCFF" opacity=".58" />
      </g>

      {/* Expressive eyes corresponding to canonical moods */}
      <g className={styles.guardianEyes} data-nex-part="face">
        {warning || critical ? (
          <>
            <g fill={accent}>
              <rect x="48" y="49" width="10" height="13" rx="3" />
              <rect x="70" y="49" width="10" height="13" rx="3" />
            </g>
            <circle cx="53" cy="53" r="1.8" fill={glint} />
            <circle cx="75" cy="53" r="1.8" fill={glint} />
          </>
        ) : offline ? (
          <g stroke="#8195AF" strokeWidth="4" strokeLinecap="round">
            <path d="M47 56h14 M67 56h14" />
          </g>
        ) : mood === 'working' ? (
          <>
            <g fill="none" stroke="#31D5FF" strokeWidth="4" strokeLinecap="square">
              <path d="M48 57l9-8 5 8 M80 57l-9-8-5 8" />
            </g>
            <path
              d="M53 48l3-2 M75 48l3-2"
              stroke="#DBFFFF"
              strokeWidth="1.4"
              opacity=".6"
              strokeLinecap="round"
            />
            <circle cx="54" cy="54" r="1" fill="#FFFFFF" />
            <circle cx="74" cy="54" r="1" fill="#FFFFFF" />
          </>
        ) : (
          <>
            <g fill="none" stroke="#31D5FF" strokeWidth="4" strokeLinecap="round">
              <path d="M47 55q8-13 16 0 M65 55q8-13 16 0" />
            </g>
            <path
              d="M53 48l3-2 M75 48l3-2"
              stroke="#DBFFFF"
              strokeWidth="1.4"
              opacity=".6"
              strokeLinecap="round"
            />
            <circle cx="55" cy="53" r="1" fill="#FFFFFF" />
            <circle cx="73" cy="53" r="1" fill="#FFFFFF" />
          </>
        )}
      </g>

      {/* Exactly four magnetic modules. Always visible, identical canonical anatomy. */}
      <g data-nex-part="orbitals">
        {(
          [
            [14, 96, 0],
            [114, 96, 1],
            [30, 115, 2],
            [98, 115, 3],
          ] as const
        ).map(([x, y, index]) => (
          <g key={index} transform={`translate(${x} ${y})`} data-nex-module={index + 1}>
            {/* Do not animate the outer transform: it anchors the module to its canonical position. */}
            <g className={index % 2 === 0 ? styles.nexModuleNear : styles.nexModuleFar}>
              <circle cx="0" cy="0" r="7.5" fill="#0A2252" stroke="#ECF8FF" strokeWidth="2.8" />
              <circle cx="0" cy="0" r="3.5" fill={accent} stroke="#225C9C" strokeWidth="1.6" />
              <circle cx="-1" cy="-1" r="1" fill={glint} />
            </g>
          </g>
        ))}
      </g>

      {/* Lower orbital arc (foreground) */}
      <g className={styles.nexOrbitFront} data-nex-part="orbit-front">
        <path d="M8 100q56 36 112 0" fill="none" stroke={accent} strokeWidth="2.2" />
        <path d="M26 113h6 M96 113h6" stroke={glint} strokeWidth="1.6" strokeLinecap="round" />
      </g>

      {/* Specular glints and orbital sparkles */}
      <g className={styles.nexSparkle} data-nex-part="sparkle" fill={accent}>
        <circle cx="10" cy="46" r="1" fill={glint} />
        <circle cx="118" cy="46" r="1" fill={glint} />
        <path
          d="M6 52h2v-4h2v4h2v2h-2v4H8v-4H6z M117 42h2v-3h2v3h2v2h-2v3h-2v-3h-2z"
          opacity=".8"
        />
      </g>
    </svg>
  );
};
