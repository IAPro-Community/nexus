import React from 'react';
import type { GuardianMood } from './guardianModel';
import styles from './NexusHud.module.scss';

type NexPixelSpriteProps = {
  mood: GuardianMood;
};

/**
 * NEX — official, front-facing 32-bit-inspired pixel art mascot.
 *
 * A single authored 128x128 SVG master for native HUD usage.
 * Retains the existing four truthful HUD moods: idle/working/warning/offline.
 * All anatomy remains stable across states. No network, GIF, WebGL, provider
 * probing or second state machine. Hero/marketing derivatives must preserve
 * this silhouette; the reference illustration is NOT a sprite sheet.
 */
export const NexPixelSprite: React.FC<NexPixelSpriteProps> = ({ mood }) => {
  const offline = mood === 'offline';
  const warning = mood === 'warning';
  const accent = offline ? '#71839D' : warning ? '#FF8A45' : '#31D5FF';
  const glint = offline ? '#A5B4C8' : warning ? '#FFE2A3' : '#DFFFFF';

  return (
    <svg
      className={styles.guardianSprite}
      viewBox="0 0 128 128"
      shapeRendering="crispEdges"
      data-nex-art="32bit-official"
      data-nex-facing="front"
      role="presentation"
      focusable="false"
    >
      <defs>
        <linearGradient id="nex-shell-32" x1="0" x2="0.95" y1="0" y2="1">
          <stop offset="0" stopColor="#FFFFFF" />
          <stop offset="0.34" stopColor="#E5F2FF" />
          <stop offset="0.7" stopColor="#AAC4E2" />
          <stop offset="1" stopColor="#5C83BB" />
        </linearGradient>
        <linearGradient id="nex-glass-32" x1="0" x2="0.8" y1="0" y2="1">
          <stop offset="0" stopColor="#15365C" />
          <stop offset="0.3" stopColor="#081B38" />
          <stop offset="1" stopColor="#030B21" />
        </linearGradient>
      </defs>

      {/* Recessed light / hover shadow */}
      <g className={styles.nexShadow} data-nex-part="shadow">
        <ellipse cx="64" cy="123" rx="26" ry="3" fill="#092B56" opacity=".65" />
        <path d="M42 122h44v2H42z" fill="#2775C3" opacity=".65" />
        <path d="M53 122h22v1H53z" fill={accent} opacity=".9" />
      </g>

      {/* Energy orbit is behind the body; foreground arc appears at the bottom. */}
      <g className={styles.guardianSignal} data-nex-part="orbit">
        <path d="M7 91v-3h5v-3h11v-3h16v-2h51v2h16v3h11v3h5v3" fill="none" stroke="#124F9A" strokeWidth="2" />
        <path d="M11 88h8v-3h14v-2h18m26 0h17v2h14v3h9" stroke="#37BFFF" fill="none" strokeWidth="2" opacity=".9" />
        <path d="M5 92h5v3h12v3h18v2h48v-2h18v-3h12v-3h5" stroke="#276CC4" strokeWidth="2" fill="none" />
        <path d="M12 94h10v3h21v2h42v-2h21v-3h9" stroke={accent} strokeWidth="1.5" fill="none" />
        <path d="M13 91h3v3h-3zM110 94h3v3h-3z" fill={glint} />
      </g>

      {/* Twin upright fins: official front-facing silhouette. */}
      <g className={styles.guardianAntenna} data-nex-part="fins">
        <path d="M26 34V11h3V7h4v4h3v23zM92 34V11h3V7h4v4h3v23z" fill="#102B60" />
        <path d="M28 32V12h2V9h2v4h2v19zM94 32V12h2V9h2v4h2v19z" fill="#7FA5E0" />
        <path d="M30 29V13h2v17h-2zM96 29V13h2v17h-2z" fill="#46CFFF" />
        <path d="M30 17h1v10h-1zM96 17h1v10h-1z" fill="#E3FDFF" />
      </g>

      {/* A pair of circular ear modules with metallic rings. */}
      <g data-nex-part="ear-couplings">
        {[21, 107].map((x) => (
          <g key={x} transform={`translate(${x} 55)`}>
            <path d="M-11-15h22v4h4v22h-4v4h-22v-4h-4v-22h4z" fill="#071834" />
            <path d="M-10-13h20v4h4v18h-4v4h-20v-4h-4V-9h4z" fill="url(#nex-shell-32)" />
            <path d="M-8-9h16v4h3V5H8v4H-8V5h-3V-5h3z" fill="#4A76AD" />
            <path d="M-6-8H6v4h3v8H6v4H-6V4h-3v-8h3z" fill="#062C6F" />
            <path d="M-4-6H4v3h3v6H4v3h-8V3h-3v-6h3z" fill={offline ? '#607A9B' : '#137BE6'} />
            <path d="M-2-4h4v8h-4z" fill={accent} />
            <path d="M-6-10h7v2h-7z" fill="#F7FBFF" />
          </g>
        ))}
      </g>

      {/* Three-dimensional, symmetrical compact body, not a stick figure. */}
      <g className={styles.guardianBody} data-nex-part="torso">
        <path d="M49 82h30v3h7v11h-3v11h-6v7H51v-7h-6V96h-3V85h7z" fill="#06182F" />
        <path d="M50 84h28v3h6v9h-3v11h-6v5H53v-5h-6V96h-3v-9h6z" fill="url(#nex-shell-32)" />
        <path d="M51 85h26v2H51z" fill="#FAFDFF" />
        <path d="M47 90h5v14h-5zM76 90h5v14h-5z" fill="#6C9AC9" />
        <path d="M55 109h18v2H55z" fill="#779EC9" />
        <path d="M61 81h6v6h-6z" fill="#1D3B70" />
        <path d="M48 95h4v4h-4zM76 95h4v4h-4z" fill="#193D79" />
      </g>

      {/* Reactor and rotating luminous core. */}
      <g className={styles.nexReactor} data-nex-part="reactor">
        <circle cx="64" cy="99" r="15" fill="#092349" />
        <circle cx="64" cy="99" r="13" fill="#5C82B9" />
        <circle cx="64" cy="99" r="11" fill="#DFEFFB" />
        <circle cx="64" cy="99" r="9" fill="#104B9E" />
        <circle cx="64" cy="99" r="7" fill="#1B86ED" />
        <circle cx="64" cy="99" r="5" fill={accent} />
        <path d="M62 94h4v2h-4zM59 99h2v3h-2z" fill={glint} />
      </g>

      {/* Heroic large round helmet: fixed frontal view and distinct dark visor. */}
      <g data-nex-part="helmet">
        <path d="M45 24h38v3h9v4h6v6h4v9h2v20h-2v8h-4v7h-7v5H37v-5h-7v-7h-4v-8h-2V46h2v-9h4v-6h6v-4h9z" fill="#091B39" />
        <path d="M45 26h38v3h9v4h5v6h4v9h2v16h-2v8h-4v7h-7v5H38v-5h-7v-7h-4v-8h-2V48h2v-9h4v-6h5v-4h9z" fill="url(#nex-shell-32)" />
        <path d="M45 27h38v3h8v4h5v6h3v6H29v-6h3v-6h5v-4h8z" fill="#F2F8FF" />
        <path d="M47 29h34v2h8v2H39v-2h8z" fill="#FFFFFF" />
        <path d="M34 42h3v-5h6v-3h42v3h6v5h3v4H34z" fill="#CDDEF3" />
        <path d="M43 35h42v3h6v5h3v24h-3v6h-7v3H44v-3h-7v-6h-3V43h3v-5h6z" fill="#37517A" />
        <path d="M44 38h40v3h6v4h2v21h-3v5h-6v3H45v-3h-6v-5h-3V45h2v-4h6z" fill="url(#nex-glass-32)" />
        <path d="M43 41h7v2h-7zM51 39h20v2H51zM75 40h8v2h-8z" fill="#5A80B4" opacity=".75" />
        <path d="M38 48h2v12h-2zM88 50h2v9h-2z" fill="#22436F" />
        <path d="M40 73h7v3h34v-3h7v3h-6v3H46v-3h-6z" fill="#9EBBDE" />
        <path d="M49 80h30v2H49z" fill="#F7FCFF" />
        {/* Pixel highlights instead of a photorealistic glass reflection */}
        <path d="M46 34h8v2h-8zM70 33h8v2h-8z" fill="#C3D8EF" />
        <path d="M52 32h4v2h-4z" fill="#4A8CCF" />
      </g>

      <g className={styles.guardianEyes} data-nex-part="face">
        {warning ? (
          <>
            <path d="M47 53h4v-2h6v2h3v10h-3v3h-7v-3h-3zM68 53h3v-2h6v2h4v10h-3v3h-7v-3h-3z" fill="#A53A27" />
            <path d="M49 54h8v9h-8zM71 54h8v9h-8z" fill={accent} />
            <path d="M51 55h3v5h-3zM73 55h3v5h-3z" fill={glint} />
          </>
        ) : offline ? (
          <>
            <path d="M45 59h17v3H45zM66 59h17v3H66z" fill="#8195AF" />
            <path d="M48 58h10v1H48zM69 58h10v1H69z" fill="#BCC8D8" />
          </>
        ) : mood === 'working' ? (
          <>
            <path d="M45 58h4v-3h5v-2h4v2h3v5h-3v2h-4v-3h-5v3h-4zM67 58h4v-3h5v-2h4v2h3v5h-3v2h-4v-3h-5v3h-4z" fill="#0C5AAB" />
            <path d="M47 59h3v-3h5v-2h2v3h2v3h-3v-2h-6v3h-3zM69 59h3v-3h5v-2h2v3h2v3h-3v-2h-6v3h-3z" fill={accent} />
            <path d="M52 55h3v2h-3zM74 55h3v2h-3z" fill={glint} />
          </>
        ) : (
          <>
            <path d="M44 61h4v-6h3v-3h6v3h3v6h-3v-3h-2v-2h-3v2h-2v3h-2v3h-4zM68 61h3v-6h3v-3h6v3h3v6h-3v-3h-2v-2h-3v2h-2v3h-2v3h-3z" fill="#0F5DAD" />
            <path d="M46 60h3v-5h3v-2h4v2h3v5h-3v-4h-4v2h-2v4h-4zM69 60h3v-5h3v-2h4v2h3v5h-3v-4h-4v2h-2v4h-4z" fill={accent} />
            <path d="M51 55h3v2h-3zM75 55h3v2h-3z" fill={glint} />
          </>
        )}
      </g>

      {/* Exactly four magnetic modules. Always visible, same anatomy. */}
      <g data-nex-part="orbitals">
        {([
          [14, 96, 0],
          [114, 96, 1],
          [30, 115, 2],
          [98, 115, 3],
        ] as const).map(([x, y, index]) => (
          <g
            key={index}
            className={index % 2 === 0 ? styles.nexModuleNear : styles.nexModuleFar}
            transform={`translate(${x} ${y})`}
            data-nex-module={index + 1}
          >
            <path d="M-7-6h14v3h3v7H7v3H-7V4h-3v-7h3z" fill="#091D3E" />
            <path d="M-6-5H6v3h2v5H6v3H-6V3h-2v-5h2z" fill="#B7D0EC" />
            <path d="M-5-4H5v2H7v4H5v3H-5V2h-2v-4h2z" fill="#EDF7FF" />
            <path d="M-3-2h6v2h2v3H3v2h-6V3h-2V0h2z" fill="#194A89" />
            <path d="M-2-1h4v2h1v2H2v1h-4V3h-1V1h1z" fill={accent} />
            <path d="M-3-4h5v1h-5z" fill="#FFFFFF" />
          </g>
        ))}
      </g>

      {/* Essential specular pixels and a lightweight hover spark */}
      <g className={styles.nexSparkle} data-nex-part="sparkle" fill={accent}>
        <path d="M6 52h2v-4h2v4h2v2h-2v4H8v-4H6zM117 42h2v-3h2v3h2v2h-2v3h-2v-3h-2z" />
        <path d="M29 93h2v2h-2zM99 90h2v2h-2z" fill={glint} />
      </g>
    </svg>
  );
};
