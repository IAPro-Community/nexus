import {
  ART_CANVAS_SIZE,
  VIRTUAL_ARTBOARD_SIZE,
  blinkScale,
  clamp01,
  computeContainTransform,
  expressionPose,
  idleModulePose,
  idlePose,
  lerp,
  orbitPose,
  smoothstep,
  type NexExpression,
} from './nexEnvelope';
import { NexMotionClock, type NexClockMetrics, type NexPacingMode } from './nexMotionClock';

interface ManifestLayer {
  id: string;
  file: string;
  x: number;
  y: number;
  width: number;
  height: number;
  role: 'floating-module' | 'visor-expression' | 'fin-glint';
  anchor: [number, number];
}

interface HybridManifest {
  canvas: [number, number];
  base: string;
  layers: ManifestLayer[];
}

export interface NexRigCanvasOptions {
  onReady?: () => void;
  onError?: (error: Error) => void;
  onMetrics?: (metrics: NexClockMetrics) => void;
}

const STATIC_POSE_SEC = 1.37;
const EXPRESSION_BLEND_SEC = 0.22;
/** Safety margin over the logical-artboard scale for breathing/sway/zoom. */
const PRESCALE_HEADROOM = 1.25;
/** Highest source-pixel bake the HUD ever needs (256 CSS px at DPR 2). */
const PRESCALE_MAX_PX = 512;

const loadImage = async (src: string): Promise<HTMLImageElement> => {
  const img = new Image();
  img.decoding = 'async';
  img.src = src;
  await img.decode();
  return img;
};

/**
 * Scale ratio for baked layer surfaces. Keeps logical geometry unchanged:
 * `drawImage(surface, box)` maps the baked bitmap onto the artboard box.
 */
export const prescaleRatio = (backingSizePx: number): number => {
  const safeBacking = Math.max(1, Number.isFinite(backingSizePx) ? backingSizePx : 1);
  const ratio = (safeBacking / ART_CANVAS_SIZE) * PRESCALE_HEADROOM;
  return Math.min(1, Math.max(1 / 32, ratio));
};

export const prescaleTargetPx = (sourcePx: number, ratio: number): number =>
  Math.max(1, Math.min(PRESCALE_MAX_PX, Math.ceil(sourcePx * ratio)));

/**
 * Bakes a decoded bitmap down to the active viewport scale so every frame
 * blits a small surface instead of resampling the 1254x1254 original.
 * Falls back to the source bitmap when no 2D context is available (tests/SSR).
 */
export const prescaleLayer = (
  source: CanvasImageSource,
  sourceWidth: number,
  sourceHeight: number,
  ratio: number,
): HTMLCanvasElement | CanvasImageSource => {
  const targetWidth = prescaleTargetPx(sourceWidth, ratio);
  const targetHeight = prescaleTargetPx(sourceHeight, ratio);
  if (targetWidth >= Math.ceil(sourceWidth) && targetHeight >= Math.ceil(sourceHeight)) {
    return source;
  }
  if (typeof document === 'undefined') return source;
  const surface = document.createElement('canvas');
  surface.width = targetWidth;
  surface.height = targetHeight;
  const ctx = surface.getContext('2d');
  if (!ctx) return source;
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = 'high';
  ctx.drawImage(source, 0, 0, targetWidth, targetHeight);
  return surface;
};

const releaseSurface = (surface: CanvasImageSource | null | undefined): void => {
  if (surface && typeof HTMLCanvasElement !== 'undefined' && surface instanceof HTMLCanvasElement) {
    surface.width = 0;
    surface.height = 0;
  }
};

/**
 * Isolated Canvas 2D rig renderer using the locked NEX-ID-v1 hybrid-v1 layers.
 * - Pre-scales decoded textures to the active viewport scale so idle ticks do
 *   not resample 1254x1254 bitmaps every frame.
 * - Single NexMotionClock with 0 RAF when paused/hidden/offline/reduced-motion.
 */
export class NexRigCanvas {
  private readonly canvas: HTMLCanvasElement;
  private readonly ctx: CanvasRenderingContext2D;
  private readonly clock: NexMotionClock;
  private readonly options: NexRigCanvasOptions;

  private loaded = false;
  private destroyed = false;
  private baseImage: HTMLImageElement | null = null;
  private layerImages: Record<string, HTMLImageElement> = {};
  private baseSurface: CanvasImageSource | null = null;
  private layerSurfaces: Record<string, CanvasImageSource> = {};
  private surfaceScale = 0;
  private modules: ManifestLayer[] = [];
  private eyes: ManifestLayer[] = [];
  private fins: ManifestLayer[] = [];

  private expression: NexExpression = 'ambient';
  private expressionPrev: NexExpression = 'ambient';
  private renderedExpression: NexExpression = 'ambient';
  private expressionBlendT = 1;
  private expressionTimeSec = 0;

  private orbitMode: 'idle' | 'orbit' = 'idle';
  private orbitTime = 0;
  private orbitBlend = 0;
  private pointer = { x: 0, y: 0 };
  private animated = true;
  private lastSimTimeSec = 0;

  constructor(canvas: HTMLCanvasElement, options: NexRigCanvasOptions = {}) {
    const ctx = canvas.getContext('2d', { alpha: true, desynchronized: true });
    if (!ctx) throw new Error('Canvas 2D context unavailable');
    this.canvas = canvas;
    this.ctx = ctx;
    this.options = options;
    this.clock = new NexMotionClock((simTimeSec, dtSec) => this.onClockTick(simTimeSec, dtSec));
  }

  async load(): Promise<void> {
    try {
      const root = new URL('/nex/hybrid-v1/', window.location.origin);
      const res = await fetch(new URL('manifest.json', root));
      if (!res.ok) throw new Error(`NEX manifest HTTP ${res.status}`);
      const manifest = (await res.json()) as HybridManifest;
      if (
        manifest.canvas?.[0] !== ART_CANVAS_SIZE ||
        !Array.isArray(manifest.layers) ||
        manifest.layers.filter((l) => l.role === 'floating-module').length !== 4
      ) {
        throw new Error('NEX layer contract mismatch');
      }

      const uniqueFiles = Array.from(
        new Set([manifest.base, ...manifest.layers.map((l) => l.file)]),
      );
      const entries = await Promise.all(
        uniqueFiles.map(async (file) => [file, await loadImage(new URL(file, root).href)] as const),
      );
      if (this.destroyed) return;

      const imageMap = Object.fromEntries(entries);
      this.baseImage = imageMap[manifest.base] ?? null;
      this.layerImages = imageMap;
      this.modules = manifest.layers.filter((l) => l.role === 'floating-module');
      this.eyes = manifest.layers.filter((l) => l.role === 'visor-expression');
      this.fins = manifest.layers.filter((l) => l.role === 'fin-glint');
      this.loaded = true;
      this.resize();
      this.bakeSurfaces();
      this.syncClock();
      this.render(!this.animated);
      this.options.onReady?.();
    } catch (err) {
      if (!this.destroyed) {
        this.options.onError?.(err instanceof Error ? err : new Error(String(err)));
      }
    }
  }

  resize(): void {
    if (this.destroyed) return;
    const rect = this.canvas.getBoundingClientRect();
    const { backingSizePx } = computeContainTransform(
      rect.width || 160,
      rect.height || 160,
      window.devicePixelRatio || 1,
    );
    if (this.canvas.width !== backingSizePx || this.canvas.height !== backingSizePx) {
      this.canvas.width = backingSizePx;
      this.canvas.height = backingSizePx;
    }
    if (this.loaded) {
      this.bakeSurfaces();
      this.render(!this.animated);
    }
  }

  /**
   * Bakes every decoded layer at the current viewport scale. Re-bakes only when
   * the viewport outgrows the existing bake; the original bitmaps stay reachable
   * solely as the bake source (they are never drawn per frame).
   */
  private bakeSurfaces(): void {
    const ratio = prescaleRatio(this.canvas.width);
    if (this.surfaceScale > 0 && ratio <= this.surfaceScale * 1.01) return;
    this.releaseSurfaces();
    this.baseSurface = this.baseImage
      ? prescaleLayer(this.baseImage, this.baseImage.width, this.baseImage.height, ratio)
      : null;
    const surfaced: Record<string, CanvasImageSource> = {};
    for (const [file, image] of Object.entries(this.layerImages)) {
      surfaced[file] = prescaleLayer(image, image.width, image.height, ratio);
    }
    this.layerSurfaces = surfaced;
    this.surfaceScale = ratio;
  }

  private releaseSurfaces(): void {
    releaseSurface(this.baseSurface);
    for (const surface of Object.values(this.layerSurfaces)) releaseSurface(surface);
    this.baseSurface = null;
    this.layerSurfaces = {};
    this.surfaceScale = 0;
  }

  updatePresentation(next: {
    expression: NexExpression;
    orbitMode: 'idle' | 'orbit';
    animated: boolean;
  }): void {
    if (next.expression !== this.expression) {
      this.expressionPrev = this.renderedExpression;
      this.expression = next.expression;
      this.expressionBlendT = 0;
      this.expressionTimeSec = 0;
    }
    this.orbitMode = next.orbitMode;
    this.animated = next.animated;
    this.canvas.dataset.nexExpression = this.expression;
    this.syncClock();
    if (!this.animated && this.loaded) {
      this.render(true);
    }
  }

  setPointerNormalized(nx: number, ny: number): void {
    this.pointer.x = Math.max(-1, Math.min(1, Number.isFinite(nx) ? nx : 0));
    this.pointer.y = Math.max(-1, Math.min(1, Number.isFinite(ny) ? ny : 0));
  }

  getMetrics(): NexClockMetrics {
    return this.clock.getMetrics();
  }

  private syncClock(): void {
    if (!this.loaded || !this.animated || this.destroyed) {
      this.clock.setMode('paused');
      return;
    }
    const interactive =
      this.orbitMode === 'orbit' ||
      this.expressionBlendT < 1 ||
      (this.expression !== 'ambient' && this.expression !== 'rest');
    const pacing: NexPacingMode = interactive ? 'interactive' : 'idle';
    this.clock.setMode(pacing);
  }

  private onClockTick(simTimeSec: number, dtSec: number): void {
    const delta = Math.max(0, simTimeSec - this.lastSimTimeSec || dtSec);
    this.lastSimTimeSec = simTimeSec;
    this.expressionTimeSec += delta;
    this.expressionBlendT = clamp01(this.expressionBlendT + delta / EXPRESSION_BLEND_SEC);
    const targetOrbit = this.orbitMode === 'orbit' ? 1 : 0;
    this.orbitBlend = lerp(this.orbitBlend, targetOrbit, 1 - Math.exp(-delta * 3.0));
    if (Math.abs(this.orbitBlend - targetOrbit) < 0.001) this.orbitBlend = targetOrbit;
    if (this.orbitBlend > 0.002) this.orbitTime += delta;

    this.render(false);
    this.syncClock();
    this.options.onMetrics?.(this.clock.getMetrics());
  }

  private paintLayer(
    layer: ManifestLayer,
    pose: { x?: number; y?: number; rotation?: number; scale?: number } | null,
    alpha = 1,
  ): void {
    const image = this.layerSurfaces[layer.file];
    if (!image) return;
    const cx = pose?.x ?? layer.anchor[0];
    const cy = pose?.y ?? layer.anchor[1];
    this.ctx.save();
    this.ctx.globalAlpha = alpha;
    this.ctx.translate(cx, cy);
    this.ctx.rotate(pose?.rotation ?? 0);
    const scale = pose?.scale ?? 1;
    this.ctx.scale(scale, scale);
    this.ctx.drawImage(
      image,
      layer.x - layer.anchor[0],
      layer.y - layer.anchor[1],
      layer.width,
      layer.height,
    );
    this.ctx.restore();
  }

  render(still: boolean): void {
    if (!this.loaded || !this.baseSurface) return;
    const ctx = this.ctx;
    const sizePx = this.canvas.width;
    const { scale, offsetX, offsetY } = computeContainTransform(sizePx, sizePx, 1);

    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.clearRect(0, 0, sizePx, sizePx);
    ctx.setTransform(scale, 0, 0, scale, offsetX * scale, offsetY * scale);
    ctx.imageSmoothingEnabled = true;

    const t = still ? STATIC_POSE_SEC : this.lastSimTimeSec;
    const pose = idlePose(t);
    const rawBlend = still ? 1 : clamp01(this.expressionBlendT);
    const blendT = smoothstep(rawBlend);
    const exprT = still ? STATIC_POSE_SEC : this.expressionTimeSec;
    const prev = expressionPose(exprT, this.expressionPrev);
    const next = expressionPose(exprT, this.expression);
    const fx = {
      bobOffset: lerp(prev.bobOffset, next.bobOffset, blendT),
      swayDegOffset: lerp(prev.swayDegOffset, next.swayDegOffset, blendT),
      coreLightOffset: lerp(prev.coreLightOffset, next.coreLightOffset, blendT),
      eyeLookX: lerp(prev.eyeLookX, next.eyeLookX, blendT),
      eyeLookY: lerp(prev.eyeLookY, next.eyeLookY, blendT),
      eyeScaleY: lerp(prev.eyeScaleY, next.eyeScaleY, blendT),
      moduleSpread: lerp(prev.moduleSpread, next.moduleSpread, blendT),
      moduleLift: lerp(prev.moduleLift, next.moduleLift, blendT),
      jitterX: lerp(prev.jitterX, next.jitterX, blendT),
      sinkOffset: lerp(prev.sinkOffset, next.sinkOffset, blendT),
    };
    if (rawBlend >= 1) this.renderedExpression = this.expression;

    ctx.save();
    ctx.translate(ART_CANVAS_SIZE / 2, ART_CANVAS_SIZE / 2);
    ctx.rotate(((pose.swayDeg + fx.swayDegOffset) * Math.PI) / 180);
    ctx.scale(pose.breathing, pose.breathing);
    ctx.translate(
      -ART_CANVAS_SIZE / 2 + fx.jitterX,
      -ART_CANVAS_SIZE / 2 + pose.bob + fx.bobOffset - fx.sinkOffset,
    );

    const idleInfluence = 1 - smoothstep(this.orbitBlend);
    const orbit = this.modules.map((layer, index) => {
      const orbital = orbitPose(index, this.orbitTime, this.orbitBlend);
      const secondary = idleModulePose(index, t);
      const spreadX = 627 + (orbital.x - 627) * fx.moduleSpread;
      const spreadY = 800 + (orbital.y - 800) * fx.moduleSpread;
      const radiusX = spreadX - 627;
      const radiusY = spreadY - 800;
      const radius = Math.hypot(radiusX, radiusY) || 1;
      return {
        layer,
        motion: {
          ...orbital,
          x: spreadX + (radiusX / radius) * fx.moduleLift + secondary.dx * idleInfluence,
          y:
            spreadY +
            (radiusY / radius) * fx.moduleLift -
            fx.moduleLift +
            secondary.dy * idleInfluence,
          rotation: orbital.rotation + secondary.rotation * idleInfluence,
          scale: orbital.scale * lerp(1, secondary.scale, idleInfluence),
        },
      };
    });

    for (const { layer, motion } of orbit) {
      const behindAlpha = 1 - motion.frontWeight;
      if (behindAlpha > 0.001) this.paintLayer(layer, motion, behindAlpha);
    }

    if (this.baseSurface) {
      ctx.drawImage(this.baseSurface, 0, 0, ART_CANVAS_SIZE, ART_CANVAS_SIZE);
    }

    const blink = still ? 1 : blinkScale(t);
    for (const eye of this.eyes) {
      const lookX = this.pointer.x * 4.5 + fx.eyeLookX;
      const lookY = this.pointer.y * 2.1 + fx.eyeLookY;
      ctx.save();
      ctx.translate(eye.anchor[0] + lookX, eye.anchor[1] + lookY);
      ctx.scale(1, blink * fx.eyeScaleY);
      const eyeSurface = this.layerSurfaces[eye.file];
      if (!eyeSurface) {
        ctx.restore();
        continue;
      }
      ctx.drawImage(
        eyeSurface,
        eye.x - eye.anchor[0],
        eye.y - eye.anchor[1],
        eye.width,
        eye.height,
      );
      ctx.restore();
    }

    for (const [index, fin] of this.fins.entries()) {
      this.paintLayer(fin, null, 0.055 + 0.075 * (0.5 + 0.5 * Math.sin(t * 0.8 + index * 2.2)));
    }

    ctx.save();
    const coreLight = clamp01(pose.coreLight + fx.coreLightOffset);
    const light = ctx.createRadialGradient(627, 869, 0, 627, 869, 82);
    light.addColorStop(0, `rgba(51,208,255,${0.085 * coreLight})`);
    light.addColorStop(1, 'rgba(33,162,255,0)');
    ctx.globalCompositeOperation = 'screen';
    ctx.fillStyle = light;
    ctx.beginPath();
    ctx.arc(627, 869, 82, 0, Math.PI * 2);
    ctx.fill();
    ctx.restore();

    for (const { layer, motion } of [...orbit].sort((a, b) => a.motion.y - b.motion.y)) {
      if (motion.frontWeight > 0.001) {
        this.paintLayer(layer, motion, motion.frontWeight);
      }
    }

    ctx.restore();
  }

  destroy(): void {
    this.destroyed = true;
    this.clock.destroy();
    this.releaseSurfaces();
    this.baseImage = null;
    this.layerImages = {};
  }
}

export { VIRTUAL_ARTBOARD_SIZE };
