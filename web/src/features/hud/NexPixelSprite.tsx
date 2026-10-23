import React from 'react';
import type { GuardianMood } from './guardianModel';
import styles from './NexusHud.module.scss';

type NexPixelSpriteProps = { mood: GuardianMood };

/**
 * NEX 16-bit: canonical vector pixel artwork for the native HUD.
 * Integer-coordinate 64x64 geometry stays sharp without image downloads.
 * Faces are projections of GuardianMood; no additional telemetry or state.
 */
export const NexPixelSprite: React.FC<NexPixelSpriteProps> = ({ mood }) => (
  <svg
    className={styles.guardianSprite}
    viewBox="0 0 64 64"
    shapeRendering="crispEdges"
    data-nex-art="16bit-v1"
    focusable="false"
  >
    {/* Behind-the-body orbital track */}
    <g opacity="0.85">
      <path d="M5 45V42H9V40H15V38H24V37H43V38H51V40H57V42H60V45" fill="none" stroke="#1455A7" strokeWidth="2" />
      <path d="M7 43H11V41H16M50 40H54V42H59" fill="none" stroke="#40D8FF" strokeWidth="1" />
    </g>
    <path d="M22 61H42V62H22Z" fill="#0A3269" />
    <path d="M27 60H37V61H27Z" fill="#38B8FD" />

    {/* Twin fins are part of NEX's canonical silhouette */}
    <g className={styles.guardianAntenna} data-nex-part="fins">
      <path d="M12 15V10H10V3H13V5H15V8H17V16Z" fill="#122C5A" />
      <path d="M12 5H14V9H16V14H12Z" fill="#4C92FF" />
      <path d="M13 6H14V11H13Z" fill="#E1F7FF" />
      <path d="M47 16V8H49V5H51V3H54V10H52V15Z" fill="#122C5A" />
      <path d="M49 14V9H51V5H53V10H51V14Z" fill="#4C92FF" />
      <path d="M51 6H52V11H51Z" fill="#E1F7FF" />
    </g>

    {/* Pixelated magnetic ear couplings */}
    <g data-nex-part="ear-modules">
      <path d="M6 22H12V37H6V34H4V25H6Z" fill="#0A1630" />
      <path d="M6 25H8V23H12V36H8V34H6Z" fill="#A0BADE" />
      <path d="M7 26H11V33H7Z" fill="#173B79" />
      <path d="M8 27H10V32H8Z" fill="#45D8FF" />
      <path d="M52 22H58V25H60V34H58V37H52Z" fill="#0A1630" />
      <path d="M52 23H56V25H58V34H56V36H52Z" fill="#A0BADE" />
      <path d="M53 26H57V33H53Z" fill="#173B79" />
      <path d="M54 27H56V32H54Z" fill="#45D8FF" />
    </g>

    {/* Luminous core */}
    <g data-nex-part="reactor">
      <path d="M25 42H39V45H43V54H41V58H37V60H27V58H23V54H21V45H25Z" fill="#112444" />
      <path d="M26 43H38V46H41V54H39V57H36V59H28V57H25V54H23V46H26Z" fill="#C0CFE8" />
      <path d="M27 45H37V47H39V54H37V57H27V54H25V47H27Z" fill="#637EAA" />
      <path d="M28 46H36V48H38V54H36V56H28V54H26V48H28Z" fill="#122D58" />
      <path d="M30 47H34V49H36V54H34V56H30V54H28V49H30Z" fill="#217EEB" />
      <path d="M30 49H34V50H35V53H34V54H30V53H29V50H30Z" fill="#3DE3FF" />
      <path d="M31 49H33V52H31Z" fill="#F3FFFF" />
      <path d="M24 47H26V52H24ZM38 47H40V52H38Z" fill="#E4EFFA" />
    </g>

    {/* Pearl-white metal helmet, oversized navy faceplate */}
    <g data-nex-part="visor">
      <path d="M19 10H44V12H49V15H52V19H54V36H52V40H48V43H16V41H12V37H10V19H12V15H15V12H19Z" fill="#09172F" />
      <path d="M19 11H44V13H48V16H51V20H52V36H50V40H46V42H18V40H14V36H12V20H14V16H17V13H19Z" fill="#9FB5D8" />
      <path d="M19 12H43V14H48V17H50V21H51V34H49V38H45V40H19V38H15V34H13V21H15V17H18V14H19Z" fill="#EDF5FF" />
      <path d="M19 13H43V15H48V18H49V20H15V18H17V15H19Z" fill="#F8FBFF" />
      <path d="M16 19H48V21H50V34H48V37H45V39H19V37H16V34H14V22H16Z" fill="#2A4778" />
      <path d="M18 20H46V22H48V34H46V36H43V37H21V36H18V34H16V23H18Z" fill="#071427" />
      <path d="M20 20H27V21H20ZM28 20H42V21H28Z" fill="#31538B" />
      <path d="M18 22H21V23H19V26H18ZM42 21H45V22H46V24H44V23H42Z" fill="#759ED4" />
      <path d="M17 37H20V39H44V38H47V40H44V41H20V40H17Z" fill="#7190B6" />
    </g>

    {/* Eyes change only for observed operational mood */}
    <g className={styles.guardianEyes} data-nex-part="expression">
      {mood === 'working' ? (
        <g fill="#3DD6FF">
          <path d="M21 27H25V28H28V29H31V31H23V30H21Z" />
          <path d="M43 27H39V28H36V29H33V31H41V30H43Z" />
          <path d="M24 29H28V30H24ZM36 29H40V30H36Z" fill="#D5FFFF" />
        </g>
      ) : mood === 'warning' ? (
        <g>
          <path d="M23 26H28V33H23ZM36 26H41V33H36Z" fill="#FF8A45" />
          <path d="M24 27H27V30H24ZM37 27H40V30H37Z" fill="#FFD09E" />
        </g>
      ) : mood === 'offline' ? (
        <path d="M21 30H29V32H21ZM35 30H43V32H35Z" fill="#8094B5" />
      ) : (
        <g>
          <path d="M21 29H23V27H25V26H27V27H29V29H31V31H28V29H26V28H25V29H23V31H21Z" fill="#3DD6FF" />
          <path d="M33 29H35V27H37V26H39V27H41V29H43V31H40V29H38V28H37V29H35V31H33Z" fill="#3DD6FF" />
          <path d="M23 29H24V27H25V28H23ZM36 28H37V27H38V29H36Z" fill="#D5FFFF" />
        </g>
      )}
    </g>

    {/* Detached modules, fully within existing 108x112 DIP native hit region */}
    <g data-nex-part="satellites">
      <path d="M3 45H10V47H12V53H10V55H4V53H2V48H3Z" fill="#0A1833" />
      <path d="M4 46H9V48H11V52H9V54H4V52H3V48H4Z" fill="#CDDDF6" />
      <path d="M5 48H9V52H5Z" fill="#0F3D80" />
      <path d="M6 49H8V51H6Z" fill="#38D4FF" />
      <path d="M54 45H61V48H62V53H60V55H54V53H52V47H54Z" fill="#0A1833" />
      <path d="M55 46H60V48H61V52H59V54H55V52H53V48H55Z" fill="#CDDDF6" />
      <path d="M55 48H59V52H55Z" fill="#0F3D80" />
      <path d="M56 49H58V51H56Z" fill="#38D4FF" />
    </g>

    {/* Orbit/connection motif replaces old generic blinking beacon */}
    <g className={styles.guardianSignal} data-nex-part="orbit">
      <path d="M3 47H7V50H13V52H20V54H45V52H52V50H57V47H60" fill="none" stroke="#215CAD" strokeWidth="2" />
      <path d="M7 49H13V51H20V53H44V52H50V50H55" fill="none" stroke="#45D6FF" strokeWidth="1" />
      <path d="M11 50H14V51H11ZM45 52H48V53H45Z" fill="#D1FFFF" />
    </g>
  </svg>
);
