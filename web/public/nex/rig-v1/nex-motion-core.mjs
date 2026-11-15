/**
 * NEX Rig v0.2 — deterministic choreography in art-space coordinates.
 * Expressions v1 owner-approved 2026-11-15; presentation-only; no status inference.
 * Art units: 1254px original candidate PNG; 100% presentation-only.
 * Renderer-independent and Rive-portable.
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
 * Idle's independent secondary motion. Each floating companion drifts a little
 * even when the full orbital choreography is inactive; drift is intentionally
 * smaller than 0.5% of the 1254px artboard.
 */
export function idleModulePose(index, timeSec) {
  if (!Number.isInteger(index) || index < 0 || index >= HOME.length) {
    throw new RangeError('NEX has exactly four orbiting modules');
  }
  const phase = index * Math.PI / 2 + 0.4;
  return {
    dx: Math.sin(timeSec * 0.91 + phase) * 3.2,
    dy: Math.sin(timeSec * 1.13 + phase) * 4.4,
    rotation: Math.sin(timeSec * 0.62 + phase) * 0.018,
    scale: 1 + Math.sin(timeSec * 1.02 + phase) * 0.007,
  };
}

/**
 * A pair of compositing weights instead of abrupt z-index switching.
 * Rear/foreground paints overlap across a shallow 3D crossing zone.
 * At idle every module must be fully in front of the base, matching the
 * source's reconstructed silhouette.
 */
export function orbitFrontWeight(depth, blend) {
  const engaged = smoothstep(clamp01(blend * 1.5));
  const crossing = smoothstep(clamp01((depth + 0.22) / 0.44));
  return lerp(1, crossing, engaged);
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
    frontWeight: orbitFrontWeight(depth, w),
    behind: w > 0.45 && depth < -0.14,
  };
}

/** Named expression layer (v1). The player validates names against this list. */
export const EXPRESSIONS = ['ambient', 'greet', 'look', 'thinking',
  'attention', 'error', 'resolved', 'rest', 'still'];

/**
 * Damped double-bounce shared by greet/resolved: ~0.45s period, linear
 * amplitude fade across windowSec so the motion always lands back on zero.
 */
const bounceOffset = (timeSec, amplitude, windowSec) => {
  const decay = Math.max(0, 1 - timeSec / windowSec);
  return Math.sin((timeSec * Math.PI * 2) / 0.45) * amplitude * decay;
};

/** Smoothstep rise over riseSec, then smoothstep settle back to zero. */
const fanEnvelope = (timeSec, riseSec, settleSec) => smoothstep(Math.min(1, timeSec / riseSec))
  * (1 - smoothstep(Math.max(0, (timeSec - riseSec) / settleSec)));

/**
 * Pure, deterministic expression OFFSETS layered on top of idlePose by the
 * player. timeSec is the time spent inside the active expression, so
 * envelopes (bounces, fades, settling) restart on every setExpression call.
 * Unknown names fall back to the neutral ambient pose. No status inference:
 * callers decide when an expression is appropriate.
 */
export function expressionPose(timeSec, expression) {
  const t = Number.isFinite(timeSec) ? timeSec : 0;
  const pose = {
    bobOffset: 0,        // art px, additive to idlePose bob
    swayDegOffset: 0,    // degrees, additive to swayDeg
    coreLightOffset: 0,  // additive to coreLight (caller clamps 0..1)
    eyeLookX: 0,         // art px offsets for both eyes
    eyeLookY: 0,
    eyeScaleY: 1,        // 1 = normal; <1 squint/happy; 0.45 eyelids; 0.25 blink
    moduleSpread: 1,     // multiplier on orbit radii (1 = home anchors)
    moduleLift: 0,       // art px radial lift of modules toward front-top
    jitterX: 0,          // art px small deterministic shake (error only)
    sinkOffset: 0,       // art px downward (rest only)
  };
  switch (expression) {
    case 'ambient':
    case 'still':
      // Neutral / frozen as-is: no offsets at any time.
      break;
    case 'greet': {
      pose.bobOffset = bounceOffset(t, 14, 0.9);
      pose.moduleSpread = 1 + 0.18 * fanEnvelope(t, 0.45, 0.45);
      pose.eyeScaleY = 0.72;
      pose.swayDegOffset = Math.sin((t * Math.PI * 2) / 0.9) * 1.2
        * Math.max(0, 1 - t / 0.9);
      break;
    }
    case 'look':
      // Attentive rise: eyes slightly widened, tiny upward bob.
      pose.eyeScaleY = 1.06;
      pose.bobOffset = -2;
      break;
    case 'thinking':
      pose.eyeLookX = 6;
      pose.eyeLookY = -7;
      pose.coreLightOffset = Math.sin(t * 0.9) * 0.08;
      pose.moduleLift = 8 + Math.sin(t * 0.8) * 4;
      pose.eyeScaleY = 0.9;
      break;
    case 'attention':
      pose.bobOffset = -6;
      pose.moduleSpread = 0.94;
      pose.coreLightOffset = 0.12;
      pose.eyeScaleY = 1.08;
      break;
    case 'error':
      pose.jitterX = Math.sin(t * 27) * 3 * Math.max(0, 1 - t / 0.9);
      pose.coreLightOffset = Math.sin(t * 8) * 0.14;
      pose.moduleSpread = 0.9;
      pose.bobOffset = 2;
      pose.eyeScaleY = 0.85;
      break;
    case 'resolved': {
      pose.bobOffset = bounceOffset(t, 20, 1.2);
      // Fast happy orbit: spike to 1.22, then settle at 1.04 while active.
      pose.moduleSpread = 1.04 + 0.18 * fanEnvelope(t, 0.3, 0.9);
      pose.eyeScaleY = 0.68;
      pose.swayDegOffset = Math.sin((t * Math.PI * 2) / 1.2) * 2
        * Math.max(0, 1 - t / 1.2);
      break;
    }
    case 'rest':
      pose.sinkOffset = 10 * smoothstep(Math.min(1, t * 0.2));
      pose.coreLightOffset = -0.1;
      pose.eyeScaleY = 0.45;
      pose.bobOffset = Math.sin(t * 0.5) * 2;
      break;
    default:
      break;
  }
  return pose;
}
