/**
 * NEX hud-32bit idle atlas — locked asset NEX-ID-v1.
 *
 * Source of truth: docs/design/nex/identity-lock.json (role: hud_32bit_idle_atlas).
 * Asset copied byte-a-byte to web/public/nex/hud-32bit/ and hash-checked by
 * nexIdleAtlas.test.ts against the locked SHA-256. The manifest
 * docs/design/nex/experimental-atlas/nex-idle-atlas.json (schema 1.0) defines
 * 12 front-view idle animations of 8 frames at 8 fps (125 ms per frame,
 * 1000 ms per animation) in a 1024x1536 grid of 128x128 frames.
 *
 * Frame shifts derive from the approved raster (PNG layer shifts), not
 * hand-drawn pixel animation; moods are expressed through color treatment
 * overlays, not dedicated frames.
 */
import type { NexIdleAction } from './nexIdleDirector';

export const NEX_ATLAS_URL = '/nex/hud-32bit/nex-idle-atlas-128.png';

/** Locked atlas SHA-256 from docs/design/nex/identity-lock.json. */
export const NEX_ATLAS_SHA256 = '8256a302fff77996e3563ef42d1e62685b59084c8a17446bb2acb2219e85dfda';

export const NEX_ATLAS_FRAME_PX = 128;
export const NEX_ATLAS_FRAMES = 8;
export const NEX_ATLAS_FRAME_MS = 125;
export const NEX_ATLAS_ANIMATION_MS = NEX_ATLAS_FRAMES * NEX_ATLAS_FRAME_MS;

/** Atlas row (background-position-y) for each idle action, in manifest order. */
export const NEX_ATLAS_ROW_BY_ACTION: Readonly<Record<NexIdleAction, number>> = {
  idle_hover: 0,
  idle_blink: 128,
  idle_glance: 256,
  idle_orbit_breathe: 384,
  idle_head_tilt: 512,
  idle_fin_twitch: 640,
  idle_module_inspect: 768,
  idle_orbit_shuffle: 896,
  idle_curiosity: 1024,
  idle_micro_stretch: 1152,
  idle_drowse: 1280,
  idle_wake: 1408,
};

/**
 * Actions whose last frame is a state to hold until the director switches
 * away: drowse stays sleepy, wake stays awake. One-shot actions without hold
 * return to their first frame when the CSS animation ends.
 */
export const NEX_ATLAS_HOLD_LAST_ACTIONS: ReadonlySet<NexIdleAction> = new Set([
  'idle_drowse',
  'idle_wake',
]);
