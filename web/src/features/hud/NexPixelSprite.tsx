import React from 'react';
import type { GuardianMood } from './guardianModel';
import type { NexIdleAction } from './nexIdleDirector';
import { NEX_ATLAS_HOLD_LAST_ACTIONS } from './nexIdleAtlas';
import styles from './NexPixelSprite.module.scss';

type NexPixelSpriteProps = {
  mood: GuardianMood;
  idleAction?: NexIdleAction | null;
};

const ROW_CLASS_BY_ACTION: Record<NexIdleAction, string> = {
  idle_hover: styles.rowHover,
  idle_blink: styles.rowBlink,
  idle_glance: styles.rowGlance,
  idle_orbit_breathe: styles.rowOrbitBreathe,
  idle_head_tilt: styles.rowHeadTilt,
  idle_fin_twitch: styles.rowFinTwitch,
  idle_module_inspect: styles.rowModuleInspect,
  idle_orbit_shuffle: styles.rowOrbitShuffle,
  idle_curiosity: styles.rowCuriosity,
  idle_micro_stretch: styles.rowMicroStretch,
  idle_drowse: styles.rowDrowse,
  idle_wake: styles.rowWake,
};

/**
 * NEX — official hud-32bit sprite rendered from the locked NEX-ID-v1 atlas.
 *
 * The 1024x1536 raster atlas (12 idle actions x 8 frames, 125 ms per frame)
 * is the approved HUD art; frame playback walks one row with CSS steps().
 * Moods are a color treatment overlay over the authentic raster, and offline
 * renders still and desaturated. No success or progress is ever inferred.
 */
export const NexPixelSprite: React.FC<NexPixelSpriteProps> = ({ mood, idleAction = null }) => {
  const holdLast = idleAction != null && NEX_ATLAS_HOLD_LAST_ACTIONS.has(idleAction);
  const frameClassName = [
    styles.frame,
    idleAction ? ROW_CLASS_BY_ACTION[idleAction] : styles.rowHover,
    idleAction ? styles.playOnce : styles.playLoop,
    holdLast ? styles.holdLast : '',
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <div
      className={styles.sprite}
      data-nex-art="hud-32bit-atlas"
      data-nex-mood={mood}
      role="presentation"
    >
      <div className={frameClassName} data-idle-action={idleAction ?? undefined} />
      <div className={styles.veil} aria-hidden="true" />
    </div>
  );
};
