/**
 * NEX Rig v0.1 — deterministic choreography in art-space coordinates.
 * Art units: 1254px original candidate PNG; 100% presentation-only.
 * No mission/status inference. Designed to be renderer-independent and Rive-portable.
 */
export const CANVAS_SIZE = 1254;
export const ORBIT_CX = 627;
export const ORBIT_CY = 800;
export const ORBIT_RX = 446;
export const ORBIT_RY = 186;
export const ORBIT_ANGULAR_SPEED = (Math.PI * 2) / 6.4;

const HOME = [
  [164, 780],
  [1096, 784],
  [311, 992],
  [929, 1001],
];
// Phases preserve bilateral symmetry while building a continuous orbital formation.
const PHASES = [Math.PI, 0, Math.PI * 1.5, Math.PI * 0.5];

export function clamp01(value) {
  return Math.max(0, Math.min(1, Number.isFinite(value) ? value : 0));
}
export function lerp(a, b, amount) {
  return a + (b - a) * clamp01(amount);
}
export function smoothstep(value) {
  const x = clamp01(value);
  return x * x * (3 - 2 * x);
}
export function idlePose(timeSec) {
  return {
    bob: Math.sin(timeSec * 1.23) * 5.4 + Math.sin(timeSec * 0.53 + 1) * 1.6,
    swayDeg: Math.sin(timeSec * 0.64) * 0.55,
    breathing: 1 + Math.sin(timeSec * 1.18) * 0.0035,
    coreLight: 0.58 + Math.sin(timeSec * 1.3 - 0.5) * 0.17,
    // Not a head-only tilt: body_base still contains the whole helmet.
  };
}

/** 0.22s organic blink each ~5.6s, with occasional quick double blink. */
export function blinkScale(timeSec) {
  const segment = ((timeSec + 1.4) % 13.8 + 13.8) % 13.8;
  const windows = [[3.4, 3.65], [8.9, 9.17], [9.41, 9.61]];
  for (const [start, end] of windows) {
    if (segment >= start && segment <= end) {
      const progress = (segment - start) / (end - start);
      return 1 - Math.pow(Math.sin(Math.PI * progress), 1.15) * 0.9;
    }
  }
  return 1;
}

/**
 * Orbit occupies a tilted elliptical path around NEX, not an in-place jitter.
 * depth sign determines real front/back occlusion. Home anchors are preserved
 * exactly at blend=0 to avoid resting-image silhouette drift.
 */
export function orbitPose(index, orbitTime, blend) {
  if (!Number.isInteger(index) || index < 0 || index >= 4) {
    throw new RangeError('NEX has exactly four orbiting modules');
  }
  const w = smoothstep(blend);
  const angle = PHASES[index] + orbitTime * ORBIT_ANGULAR_SPEED;
  const depth = Math.sin(angle);
  const orbitalX = ORBIT_CX + Math.cos(angle) * ORBIT_RX;
  const orbitalY = ORBIT_CY + depth * ORBIT_RY;
  // Size/perspective only when orbiting; original sprite is unchanged at rest.
  const targetScale = lerp(0.76, 1.10, (depth + 1) * 0.5);
  return {
    x: lerp(HOME[index][0], orbitalX, w),
    y: lerp(HOME[index][1], orbitalY, w),
    scale: lerp(1, targetScale, w),
    rotation: w * (Math.sin(orbitTime * 1.05 + index * 1.73) * 0.16),
    depth: w * depth,
    alpha: lerp(1, lerp(0.75, 1, (depth + 1) / 2), w),
    behind: w > 0.45 && depth < -0.14,
  };
}
