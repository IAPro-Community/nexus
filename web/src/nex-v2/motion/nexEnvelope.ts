export const ART_CANVAS_SIZE = 1254;
export const SAFE_ENVELOPE_PADDING = 112;
export const VIRTUAL_ARTBOARD_SIZE = ART_CANVAS_SIZE + SAFE_ENVELOPE_PADDING * 2; // 1478

export const ORBIT_CX = 627;
export const ORBIT_CY = 800;
export const ORBIT_RX = 446;
export const ORBIT_RY = 186;
export const ORBIT_ANGULAR_SPEED = (Math.PI * 2) / 6.4;

export const NEX_EXPRESSIONS = [
  'ambient',
  'greet',
  'look',
  'thinking',
  'attention',
  'error',
  'resolved',
  'rest',
  'still',
] as const;

export type NexExpression = (typeof NEX_EXPRESSIONS)[number];

export interface LayerBoundsSpec {
  id: string;
  x: number;
  y: number;
  width: number;
  height: number;
  anchor: readonly [number, number];
}

export const MODULE_HOME_ANCHORS: readonly (readonly [number, number])[] = [
  [164, 780],
  [1096, 784],
  [311, 992],
  [929, 1001],
];

export const LOCKED_MODULE_BOUNDS: readonly LayerBoundsSpec[] = [
  { id: 'module_1', x: 41, y: 656, width: 253, height: 249, anchor: [164, 780] },
  { id: 'module_2', x: 977, y: 658, width: 240, height: 246, anchor: [1096, 784] },
  { id: 'module_3', x: 204, y: 888, width: 215, height: 207, anchor: [311, 992] },
  { id: 'module_4', x: 830, y: 894, width: 199, height: 215, anchor: [929, 1001] },
];

const MODULE_PHASES = [Math.PI, 0, Math.PI * 1.5, Math.PI * 0.5];

export function clamp01(value: number): number {
  return Math.max(0, Math.min(1, Number.isFinite(value) ? value : 0));
}

export function lerp(a: number, b: number, amount: number): number {
  return a + (b - a) * clamp01(amount);
}

export function smoothstep(value: number): number {
  const x = clamp01(value);
  return x * x * (3 - 2 * x);
}

export interface IdlePose {
  bob: number;
  swayDeg: number;
  breathing: number;
  coreLight: number;
}

export function idlePose(timeSec: number): IdlePose {
  return {
    bob: Math.sin(timeSec * 1.45) * 18 + Math.sin(timeSec * 0.62 + 1) * 5,
    swayDeg: Math.sin(timeSec * 0.72) * 1.6,
    breathing: 1 + Math.sin(timeSec * 1.35) * 0.012,
    coreLight: 0.65 + Math.sin(timeSec * 1.6 - 0.5) * 0.3,
  };
}

export function blinkScale(timeSec: number): number {
  const segment = (((timeSec + 0.9) % 5.2) + 5.2) % 5.2;
  const windows: readonly (readonly [number, number])[] = [
    [1.6, 1.84],
    [3.8, 4.04],
    [4.22, 4.42],
  ];
  for (const [start, end] of windows) {
    if (segment >= start && segment <= end) {
      const progress = (segment - start) / (end - start);
      return 1 - Math.pow(Math.sin(Math.PI * progress), 1.15) * 0.92;
    }
  }
  return 1;
}

export function idleModulePose(index: number, timeSec: number) {
  const phase = (index * Math.PI) / 2 + 0.4;
  return {
    dx: Math.sin(timeSec * 1.15 + phase) * 14,
    dy: Math.sin(timeSec * 1.42 + phase) * 18,
    rotation: Math.sin(timeSec * 0.85 + phase) * 0.08,
    scale: 1 + Math.sin(timeSec * 1.25 + phase) * 0.03,
  };
}

export function orbitFrontWeight(depth: number, blend: number): number {
  const engaged = smoothstep(clamp01(blend * 1.5));
  const crossing = smoothstep(clamp01((depth + 0.22) / 0.44));
  return lerp(1, crossing, engaged);
}

export function orbitPose(index: number, orbitTime: number, blend: number) {
  const w = smoothstep(blend);
  const angle = MODULE_PHASES[index] + orbitTime * ORBIT_ANGULAR_SPEED;
  const depth = Math.sin(angle);
  const orbitalX = ORBIT_CX + Math.cos(angle) * ORBIT_RX;
  const orbitalY = ORBIT_CY + depth * ORBIT_RY;
  const targetScale = lerp(0.76, 1.1, (depth + 1) * 0.5);
  return {
    x: lerp(MODULE_HOME_ANCHORS[index][0], orbitalX, w),
    y: lerp(MODULE_HOME_ANCHORS[index][1], orbitalY, w),
    scale: lerp(1, targetScale, w),
    rotation: w * (Math.sin(orbitTime * 1.05 + index * 1.73) * 0.16),
    depth: w * depth,
    alpha: lerp(1, lerp(0.75, 1, (depth + 1) / 2), w),
    frontWeight: orbitFrontWeight(depth, w),
  };
}

export interface ExpressionPose {
  bobOffset: number;
  swayDegOffset: number;
  coreLightOffset: number;
  eyeLookX: number;
  eyeLookY: number;
  eyeScaleY: number;
  moduleSpread: number;
  moduleLift: number;
  jitterX: number;
  sinkOffset: number;
}

const bounceOffset = (timeSec: number, amplitude: number, windowSec: number) => {
  const decay = Math.max(0, 1 - timeSec / windowSec);
  return Math.sin((timeSec * Math.PI * 2) / 0.45) * amplitude * decay;
};

const fanEnvelope = (timeSec: number, riseSec: number, settleSec: number) =>
  smoothstep(Math.min(1, timeSec / riseSec)) *
  (1 - smoothstep(Math.max(0, (timeSec - riseSec) / settleSec)));

export function expressionPose(timeSec: number, expression: NexExpression): ExpressionPose {
  const t = Number.isFinite(timeSec) ? timeSec : 0;
  const pose: ExpressionPose = {
    bobOffset: 0,
    swayDegOffset: 0,
    coreLightOffset: 0,
    eyeLookX: 0,
    eyeLookY: 0,
    eyeScaleY: 1,
    moduleSpread: 1,
    moduleLift: 0,
    jitterX: 0,
    sinkOffset: 0,
  };
  switch (expression) {
    case 'ambient':
    case 'still':
      break;
    case 'greet':
      pose.bobOffset = bounceOffset(t, 14, 0.9);
      pose.moduleSpread = 1 + 0.18 * fanEnvelope(t, 0.45, 0.45);
      pose.eyeScaleY = 0.72;
      pose.swayDegOffset = Math.sin((t * Math.PI * 2) / 0.9) * 1.2 * Math.max(0, 1 - t / 0.9);
      break;
    case 'look':
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
    case 'resolved':
      pose.bobOffset = bounceOffset(t, 20, 1.2);
      pose.moduleSpread = 1.04 + 0.18 * fanEnvelope(t, 0.3, 0.9);
      pose.eyeScaleY = 0.68;
      pose.swayDegOffset = Math.sin((t * Math.PI * 2) / 1.2) * 2 * Math.max(0, 1 - t / 1.2);
      break;
    case 'rest':
      pose.sinkOffset = 10 * smoothstep(Math.min(1, t * 0.2));
      pose.coreLightOffset = -0.1;
      pose.eyeScaleY = 0.45;
      pose.bobOffset = Math.sin(t * 0.5) * 2;
      break;
  }
  return pose;
}

export interface ArtExtents {
  minX: number;
  minY: number;
  maxX: number;
  maxY: number;
}

function transformPoint(
  px: number,
  py: number,
  cx: number,
  cy: number,
  angleRad: number,
  scale: number,
  dx: number,
  dy: number,
): [number, number] {
  const ox = px - cx;
  const oy = py - cy;
  const cos = Math.cos(angleRad);
  const sin = Math.sin(angleRad);
  const rx = (ox * cos - oy * sin) * scale + cx + dx;
  const ry = (ox * sin + oy * cos) * scale + cy + dy;
  return [rx, ry];
}

/**
 * Computes the worst-case bounding box of all body and module corners in
 * virtual artboard coordinates [0..VIRTUAL_ARTBOARD_SIZE] for a given pose.
 */
export function computeLayerExtents(
  expression: NexExpression,
  timeSec: number,
  orbitTimeSec: number,
  orbitBlend: number,
): ArtExtents {
  const pose = idlePose(timeSec);
  const fx = expressionPose(timeSec, expression);
  const globalAngle = ((pose.swayDeg + fx.swayDegOffset) * Math.PI) / 180;
  const globalScale = pose.breathing;
  const globalDx = fx.jitterX;
  const globalDy = pose.bob + fx.bobOffset - fx.sinkOffset;
  const center = ART_CANVAS_SIZE / 2;

  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;

  const includeVirtualPoint = (ax: number, ay: number) => {
    const [tx, ty] = transformPoint(
      ax,
      ay,
      center,
      center,
      globalAngle,
      globalScale,
      globalDx,
      globalDy,
    );
    const vx = tx + SAFE_ENVELOPE_PADDING;
    const vy = ty + SAFE_ENVELOPE_PADDING;
    if (vx < minX) minX = vx;
    if (vy < minY) minY = vy;
    if (vx > maxX) maxX = vx;
    if (vy > maxY) maxY = vy;
  };

  // Body artboard corners (includes full silhouette + antialias margin).
  for (const [bx, by] of [
    [0, 0],
    [ART_CANVAS_SIZE, 0],
    [0, ART_CANVAS_SIZE],
    [ART_CANVAS_SIZE, ART_CANVAS_SIZE],
  ]) {
    includeVirtualPoint(bx, by);
  }

  // All 4 orbital modules with spread, lift, secondary drift, and perspective scale.
  const idleInfluence = 1 - smoothstep(orbitBlend);
  for (let i = 0; i < LOCKED_MODULE_BOUNDS.length; i++) {
    const layer = LOCKED_MODULE_BOUNDS[i];
    const orbital = orbitPose(i, orbitTimeSec, orbitBlend);
    const secondary = idleModulePose(i, timeSec);
    const spreadX = ORBIT_CX + (orbital.x - ORBIT_CX) * fx.moduleSpread;
    const spreadY = ORBIT_CY + (orbital.y - ORBIT_CY) * fx.moduleSpread;
    const radiusX = spreadX - ORBIT_CX;
    const radiusY = spreadY - ORBIT_CY;
    const radius = Math.hypot(radiusX, radiusY) || 1;
    const mx = spreadX + (radiusX / radius) * fx.moduleLift + secondary.dx * idleInfluence;
    const my =
      spreadY + (radiusY / radius) * fx.moduleLift - fx.moduleLift + secondary.dy * idleInfluence;
    const mAngle = orbital.rotation + secondary.rotation * idleInfluence;
    const mScale = orbital.scale * lerp(1, secondary.scale, idleInfluence);

    const corners: readonly (readonly [number, number])[] = [
      [layer.x, layer.y],
      [layer.x + layer.width, layer.y],
      [layer.x, layer.y + layer.height],
      [layer.x + layer.width, layer.y + layer.height],
    ];
    for (const [cx, cy] of corners) {
      const [localX, localY] = transformPoint(
        cx,
        cy,
        layer.anchor[0],
        layer.anchor[1],
        mAngle,
        mScale,
        mx - layer.anchor[0],
        my - layer.anchor[1],
      );
      includeVirtualPoint(localX, localY);
    }
  }

  return { minX, minY, maxX, maxY };
}

export function isWithinSafeEnvelope(extents: ArtExtents): boolean {
  return (
    extents.minX >= 0 &&
    extents.minY >= 0 &&
    extents.maxX <= VIRTUAL_ARTBOARD_SIZE &&
    extents.maxY <= VIRTUAL_ARTBOARD_SIZE
  );
}

export interface ContainTransform {
  backingSizePx: number;
  cssSize: number;
  scale: number;
  offsetX: number;
  offsetY: number;
}

/**
 * Computes a bounded, centered "contain" transform mapping the virtual safe
 * envelope into any square or rectangular viewport without clipping.
 */
export function computeContainTransform(
  viewportWidth: number,
  viewportHeight: number,
  dpr = 1,
): ContainTransform {
  const safeW = Math.max(1, Number.isFinite(viewportWidth) ? viewportWidth : 160);
  const safeH = Math.max(1, Number.isFinite(viewportHeight) ? viewportHeight : 160);
  const clampedDpr = Math.max(1, Math.min(2, Number.isFinite(dpr) ? dpr : 1));
  const cssSize = Math.min(safeW, safeH);
  const backingSizePx = Math.max(1, Math.min(1024, Math.ceil(cssSize * clampedDpr)));
  const scale = backingSizePx / VIRTUAL_ARTBOARD_SIZE;
  return {
    backingSizePx,
    cssSize,
    scale,
    offsetX: SAFE_ENVELOPE_PADDING,
    offsetY: SAFE_ENVELOPE_PADDING,
  };
}
