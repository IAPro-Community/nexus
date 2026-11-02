import {
  CANVAS_SIZE, clamp01, lerp, smoothstep, idlePose, blinkScale, orbitPose, idleModulePose,
} from './nex-motion-core.mjs';

/**
 * Layer-compositor prototype for the original PNG candidate.
 * No invented Rive (.riv) binary: the output below is real, inspectable Canvas2D
 * choreography of original transparent asset layers. Later rig exporters can
 * use the same semantic pivots/tracks.
 *
 * Asset lifecycle: load once, RAF only while visible + motion allowed,
 * reduce-motion uses the byte-faithful source PNG as a static pose.
 */
const ROOT = new URL('../hybrid-v1/', import.meta.url);
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
    this.wantsMotion = true;
    this.media = window.matchMedia('(prefers-reduced-motion: reduce)');
    this.smallScreen = window.matchMedia('(max-width: 680px)');
    this.visible = document.visibilityState === 'visible';
    this.inViewport = true;
    this.pointer = { x: 0, y: 0 };
    this.time = 0;
    this.orbitTime = 0;
    this.blend = 0;
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

    // The static fallback is the exact source image: no resynthesized pixels.
    if (still) {
      ctx.drawImage(this.images['nex_source_original.png'], 0, 0, CANVAS_SIZE, CANVAS_SIZE);
      return;
    }
    const pose = idlePose(this.time);
    ctx.save();
    ctx.translate(CANVAS_SIZE / 2, CANVAS_SIZE / 2);
    ctx.rotate(pose.swayDeg * Math.PI / 180);
    ctx.scale(pose.breathing, pose.breathing);
    ctx.translate(-CANVAS_SIZE / 2, -CANVAS_SIZE / 2 + pose.bob);

    const orbit = this.moduleInfo.map((layer, index) => {
      const orbital = orbitPose(index, this.orbitTime, this.blend);
      const secondary = idleModulePose(index, this.time);
      const idleInfluence = 1 - smoothstep(this.blend);
      return {
        layer,
        motion: {
          ...orbital,
          x: orbital.x + secondary.dx * idleInfluence,
          y: orbital.y + secondary.dy * idleInfluence,
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

    const blink = blinkScale(this.time);
    for (const eye of this.eyeInfo) {
      const lookX = this.pointer.x * 4.5;
      const lookY = this.pointer.y * 2.1;
      ctx.save();
      ctx.translate(eye.anchor[0] + lookX, eye.anchor[1] + lookY);
      ctx.scale(1, blink);
      ctx.drawImage(this.images[eye.file], eye.x - eye.anchor[0],
        eye.y - eye.anchor[1], eye.width, eye.height);
      ctx.restore();
    }

    // Fin assets are ORIGINAL glints, not fabricated full hinged fins.
    for (const [index, fin] of this.finInfo.entries()) {
      this.paintLayer(fin, null,
        0.055 + 0.075 * (0.5 + 0.5 * Math.sin(this.time * 0.8 + index * 2.2)));
    }
    // Subtle additive halo does not replace body art or paint new anatomy.
    ctx.save();
    const light = ctx.createRadialGradient(627, 869, 0, 627, 869, 82);
    light.addColorStop(0, `rgba(51,208,255,${0.085 * pose.coreLight})`);
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
