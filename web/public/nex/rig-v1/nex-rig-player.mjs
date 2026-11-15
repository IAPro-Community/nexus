import {
  CANVAS_SIZE, ORBIT_CX, ORBIT_CY, clamp01, lerp, smoothstep,
  idlePose, blinkScale, orbitPose, idleModulePose, EXPRESSIONS, expressionPose,
} from './nex-motion-core.mjs';

/**
 * Layer-compositor prototype for the original PNG candidate.
 * No invented Rive (.riv) binary: the output below is real, inspectable Canvas2D
 * choreography of original transparent asset layers. Later rig exporters can
 * use the same semantic pivots/tracks.
 *
 * Asset lifecycle: load once, RAF only while visible + motion allowed;
 * reduced motion and other frozen states redraw the rig at one fixed
 * representative pose (staticTimeSec) so repeated renders stay byte-identical.
 *
 * Expressions: setExpression(name) — see nex-motion-core EXPRESSIONS;
 * DOM hook canvas.dataset.nexExpression.
 */
const ROOT = new URL('../hybrid-v1/', import.meta.url);
/** Expression crossfade duration; envelopes inside expressions restart instantly. */
const EXPRESSION_BLEND_SEC = 0.25;
const loadImage = async (filename) => {
  const img = new Image();
  img.decoding = 'async';
  img.src = new URL(filename, ROOT).href;
  await img.decode();
  return img;
};

export class NexRigPlayer {
  constructor(canvas, { onReady = () => {}, onError = () => {}, onFrame = () => {} } = {}) {
    if (!(canvas instanceof HTMLCanvasElement)) throw new TypeError('NEX requires a canvas');
    this.canvas = canvas;
    this.ctx = canvas.getContext('2d', { alpha: true, desynchronized: true });
    if (!this.ctx) throw new Error('Canvas 2D is unavailable');
    this.onReady = onReady;
    this.onError = onError;
    this.onFrame = onFrame;
    this.mode = 'idle';
    this.debugRig = false;
    this.wantsMotion = true;
    this.media = window.matchMedia('(prefers-reduced-motion: reduce)');
    this.smallScreen = window.matchMedia('(max-width: 680px)');
    this.visible = document.visibilityState === 'visible';
    this.inViewport = true;
    this.pointer = { x: 0, y: 0 };
    this.time = 0;
    this.orbitTime = 0;
    this.blend = 0;
    this.expression = 'ambient';
    this.expressionPrev = 'ambient';
    this.renderedExpression = 'ambient';
    this.expressionBlendT = 1;
    this.expressionTimeSec = 0;
    this.staticTimeSec = null;
    this.frame = 0;
    this.lastAt = 0;
    this.raf = null;
    this.loaded = false;
    this.destroyed = false;
    this.moduleInfo = [];
    this.images = {};
    this._onVis = () => { this.visible = document.visibilityState === 'visible'; this.sync(); };
    this._onReduce = () => this.sync();
    this._onResize = () => { this.resize(); this.sync(); };
    this._onPointer = (e) => {
      const bounds = this.canvas.getBoundingClientRect();
      const x = clamp01((e.clientX - bounds.left) / bounds.width);
      const y = clamp01((e.clientY - bounds.top) / bounds.height);
      this.pointer.x = (x - 0.5) * 2;
      this.pointer.y = (y - 0.5) * 2;
    };
    this._onLeave = () => { this.pointer.x = 0; this.pointer.y = 0; };
    document.addEventListener('visibilitychange', this._onVis);
    this.media.addEventListener?.('change', this._onReduce);
    this.canvas.addEventListener('pointermove', this._onPointer, { passive: true });
    this.canvas.addEventListener('pointerleave', this._onLeave);
    this.resizer = typeof ResizeObserver !== 'undefined'
      ? new ResizeObserver(this._onResize) : null;
    this.resizer?.observe(this.canvas);
    this.observer = typeof IntersectionObserver !== 'undefined'
      ? new IntersectionObserver((entries) => {
          this.inViewport = entries[0]?.isIntersecting ?? true;
          this.sync();
        }, { threshold: 0.01 }) : null;
    this.observer?.observe(this.canvas);
  }

  async load() {
    try {
      const res = await fetch(new URL('manifest.json', ROOT));
      if (!res.ok) throw new Error('NEX layer manifest not available');
      const manifest = await res.json();
      if (manifest.canvas?.[0] !== CANVAS_SIZE ||
        !Array.isArray(manifest.layers) ||
        manifest.layers.filter((layer) => layer.role === 'floating-module').length !== 4) {
        throw new Error('NEX layer contract mismatch');
      }
      const moduleInfo = manifest.layers.filter((x) => x.role === 'floating-module');
      const names = ['body_base.png', 'nex_source_original.png',
        ...manifest.layers.map((x) => x.file)];
      const imagePairs = await Promise.all([...new Set(names)].map(async (name) =>
        [name, await loadImage(name)]));
      if (this.destroyed) return;
      this.images = Object.fromEntries(imagePairs);
      this.moduleInfo = moduleInfo;
      this.eyeInfo = manifest.layers.filter((x) => x.role === 'visor-expression');
      this.finInfo = manifest.layers.filter((x) => x.role === 'fin-glint');
      this.loaded = true;
      this.resize();
      this.sync();
      this.onReady({ layerCount: manifest.layers.length, baseSize: CANVAS_SIZE });
    } catch (err) {
      if (!this.destroyed) this.onError(err);
    }
  }

  setMode(mode) {
    if (mode !== 'idle' && mode !== 'orbit') throw new Error('Unknown NEX mode');
    this.mode = mode;
    this.sync();
  }

  setMotion(allowed) {
    this.wantsMotion = Boolean(allowed);
    this.sync();
  }

  /**
   * Switch the expression layer (presentation-only choreography; never a
   * status claim). Blends over EXPRESSION_BLEND_SEC from the expression
   * actually on screen (renderedExpression), and the frozen path shows the
   * new pose immediately.
   */
  setExpression(name) {
    if (!EXPRESSIONS.includes(name)) throw new Error('Unknown NEX expression');
    if (name !== this.expression) {
      this.expressionPrev = this.renderedExpression;
      this.expression = name;
      this.expressionBlendT = 0;
      this.expressionTimeSec = 0;
    }
    // Mirror the expression for tests/e2e even when unchanged (idempotent):
    // the boot-time ambient call must still publish the attribute.
    this.canvas.dataset.nexExpression = this.expression;
    if (!this.motionActive) this.render(true);
  }

  getExpression() {
    return this.expression;
  }

  setRigOverlay(enabled) {
    this.debugRig = Boolean(enabled);
    if (this.loaded) this.render(!this.motionActive);
  }

  get motionActive() {
    return this.loaded && this.wantsMotion && !this.media.matches &&
      this.visible && this.inViewport && !this.destroyed;
  }

  resize() {
    if (this.destroyed) return;
    const bounds = this.canvas.getBoundingClientRect();
    // Cap rasterization resolution on mobile to keep the continuous orbit economical.
    const dimension = Math.max(1, Math.min(1254, Math.round(
      Math.max(1, bounds.width) * Math.min(devicePixelRatio || 1, 1.7))));
    if (this.canvas.width !== dimension || this.canvas.height !== dimension) {
      this.canvas.width = dimension;
      this.canvas.height = dimension;
    }
    if (this.loaded) this.render(!this.motionActive);
  }

  sync() {
    if (!this.loaded || this.destroyed) return;
    if (!this.motionActive) {
      if (this.raf !== null) cancelAnimationFrame(this.raf);
      this.raf = null;
      this.lastAt = 0;
      this.render(true);
      return;
    }
    if (this.raf === null) {
      this.lastAt = 0;
      this.raf = requestAnimationFrame((t) => this.tick(t));
    }
  }

  tick(timestamp) {
    this.raf = null;
    if (!this.motionActive) { this.sync(); return; }
    const dt = this.lastAt ? Math.max(0, Math.min(0.05, (timestamp - this.lastAt) / 1000)) : 0;
    this.lastAt = timestamp;
    const fpsInterval = this.smallScreen.matches ? 1 / 30 : 1 / 60;
    this.frame += dt;
    // Accumulate elapsed time continuously, but only rasterize at target FPS.
    this.time += dt;
    this.expressionTimeSec += dt;
    this.expressionBlendT = clamp01(this.expressionBlendT + dt / EXPRESSION_BLEND_SEC);
    const amount = this.mode === 'orbit' ? 1 : 0;
    // Exponential damping is frame-rate independent (30/60Hz match on mobile).
    this.blend = lerp(this.blend, amount, 1 - Math.exp(-dt * 3.0));
    if (Math.abs(this.blend - amount) < 0.001) this.blend = amount;
    if (this.blend > 0.002) this.orbitTime += dt;
    if (this.frame >= fpsInterval) {
      this.frame = 0;
      this.render(false);
      this.onFrame({ mode: this.mode, orbitBlend: this.blend });
    }
    this.raf = requestAnimationFrame((t) => this.tick(t));
  }

  paintLayer(layer, pose = null, alpha = 1) {
    const image = this.images[layer.file];
    if (!image) return;
    const anchor = layer.anchor;
    const centerX = pose?.x ?? anchor[0];
    const centerY = pose?.y ?? anchor[1];
    this.ctx.save();
    this.ctx.globalAlpha = alpha * (pose?.alpha ?? 1);
    this.ctx.translate(centerX, centerY);
    this.ctx.rotate(pose?.rotation ?? 0);
    const scale = pose?.scale ?? 1;
    this.ctx.scale(scale, scale);
    this.ctx.drawImage(image, layer.x - anchor[0], layer.y - anchor[1],
      layer.width, layer.height);
    this.ctx.restore();
  }

  render(still) {
    if (!this.loaded) return;
    const ctx = this.ctx;
    const pixelSize = this.canvas.width;
    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.clearRect(0, 0, pixelSize, pixelSize);
    ctx.setTransform(pixelSize / CANVAS_SIZE, 0, 0, pixelSize / CANVAS_SIZE, 0, 0);
    ctx.imageSmoothingEnabled = true;

    // Frozen renders (reduced motion, hidden, offline) reuse one fixed
    // representative clock so two captures 600ms apart are byte-identical.
    this.staticTimeSec = this.staticTimeSec ?? 1.37;
    const t = still ? this.staticTimeSec : this.time;
    const pose = idlePose(t);

    // Expression layer: crossfade the previous shown expression into the
    // active one; blends complete instantly on the frozen path. Envelopes
    // (bounces, fades, settling) run on the per-expression clock so they
    // restart on every setExpression, while idlePose keeps the continuous
    // clock for bob/breathing continuity.
    const rawBlend = still ? 1 : clamp01(this.expressionBlendT);
    const blendT = smoothstep(rawBlend);
    const exprT = still ? this.staticTimeSec : this.expressionTimeSec;
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
    ctx.translate(CANVAS_SIZE / 2, CANVAS_SIZE / 2);
    ctx.rotate((pose.swayDeg + fx.swayDegOffset) * Math.PI / 180);
    ctx.scale(pose.breathing, pose.breathing);
    // Expression offsets ride the existing bob translate; the anchor
    // transform of the whole composition stays untouched.
    ctx.translate(-CANVAS_SIZE / 2 + fx.jitterX,
      -CANVAS_SIZE / 2 + pose.bob + fx.bobOffset - fx.sinkOffset);

    const orbit = this.moduleInfo.map((layer, index) => {
      const orbital = orbitPose(index, this.orbitTime, this.blend);
      const secondary = idleModulePose(index, t);
      const idleInfluence = 1 - smoothstep(this.blend);
      // moduleSpread scales the radial distance from the orbit center
      // (1 preserves the home anchors exactly). moduleLift adds an extra
      // radial offset biased toward front-top so lifted modules hover
      // up over the head instead of drifting downward.
      const spreadX = ORBIT_CX + (orbital.x - ORBIT_CX) * fx.moduleSpread;
      const spreadY = ORBIT_CY + (orbital.y - ORBIT_CY) * fx.moduleSpread;
      const radiusX = spreadX - ORBIT_CX;
      const radiusY = spreadY - ORBIT_CY;
      const radius = Math.hypot(radiusX, radiusY) || 1;
      const liftX = (radiusX / radius) * fx.moduleLift;
      const liftY = (radiusY / radius) * fx.moduleLift - fx.moduleLift;
      return {
        layer,
        motion: {
          ...orbital,
          x: spreadX + liftX + secondary.dx * idleInfluence,
          y: spreadY + liftY + secondary.dy * idleInfluence,
          rotation: orbital.rotation + secondary.rotation * idleInfluence,
          scale: orbital.scale * lerp(1, secondary.scale, idleInfluence),
        },
      };
    });
    // A module is cross-composited smoothly when moving from the rear plane
    // to the front plane. The old threshold caused a whole module to pop.
    for (const { layer, motion } of orbit) {
      const behindAlpha = 1 - motion.frontWeight;
      if (behindAlpha > 0.001) this.paintLayer(layer, motion, behindAlpha);
    }
    ctx.drawImage(this.images['body_base.png'], 0, 0, CANVAS_SIZE, CANVAS_SIZE);

    const blink = blinkScale(t);
    for (const eye of this.eyeInfo) {
      const lookX = this.pointer.x * 4.5 + fx.eyeLookX;
      const lookY = this.pointer.y * 2.1 + fx.eyeLookY;
      ctx.save();
      ctx.translate(eye.anchor[0] + lookX, eye.anchor[1] + lookY);
      ctx.scale(1, blink * fx.eyeScaleY);
      ctx.drawImage(this.images[eye.file], eye.x - eye.anchor[0],
        eye.y - eye.anchor[1], eye.width, eye.height);
      ctx.restore();
    }

    // Fin assets are ORIGINAL glints, not fabricated full hinged fins.
    for (const [index, fin] of this.finInfo.entries()) {
      this.paintLayer(fin, null,
        0.055 + 0.075 * (0.5 + 0.5 * Math.sin(t * 0.8 + index * 2.2)));
    }
    // Subtle additive halo does not replace body art or paint new anatomy.
    // Expression core light rides on the idle breathing, clamped to 0..1.
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

    // Foreground modules overlap body and glow ring as they cross the orbit.
    for (const { layer, motion } of [...orbit].sort((a, b) => a.motion.y - b.motion.y)) {
      if (motion.frontWeight > 0.001) {
        this.paintLayer(layer, motion, motion.frontWeight);
      }
    }
    if (this.debugRig) this.drawRigOverlay(orbit);
    ctx.restore();
  }

  /** Optional review aid: show exactly which modules have independent pivots. */
  drawRigOverlay(modules) {
    const ctx = this.ctx;
    ctx.save();
    ctx.lineWidth = 2.6;
    ctx.strokeStyle = 'rgba(67,230,255,.72)';
    ctx.setLineDash([10, 12]);
    ctx.beginPath();
    ctx.ellipse(627, 800, 446, 186, 0, 0, 2 * Math.PI);
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.font = 'bold 29px ui-sans-serif, sans-serif';
    for (const { motion, layer } of modules) {
      ctx.fillStyle = motion.frontWeight < 0.5
        ? 'rgba(255,171,95,.96)' : 'rgba(83,236,255,.96)';
      ctx.beginPath();
      ctx.arc(motion.x, motion.y, 10, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillText(layer.id, motion.x + 16, motion.y - 15);
    }
    ctx.restore();
  }

  destroy() {
    this.destroyed = true;
    if (this.raf !== null) cancelAnimationFrame(this.raf);
    this.raf = null;
    document.removeEventListener('visibilitychange', this._onVis);
    this.media.removeEventListener?.('change', this._onReduce);
    this.canvas.removeEventListener('pointermove', this._onPointer);
    this.canvas.removeEventListener('pointerleave', this._onLeave);
    this.observer?.disconnect();
    this.resizer?.disconnect();
    this.images = {};
  }
}
