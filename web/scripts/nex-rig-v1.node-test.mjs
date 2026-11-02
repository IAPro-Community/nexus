import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import {
  CANVAS_SIZE, clamp01, smoothstep, idlePose, blinkScale, orbitPose,
  orbitFrontWeight, idleModulePose,
  ORBIT_ANGULAR_SPEED,
} from '../public/nex/rig-v1/nex-motion-core.mjs';

test('NEX first-phase rig: stable original coordinates at idle', () => {
  assert.equal(CANVAS_SIZE, 1254);
  const expected = [[164, 780], [1096, 784], [311, 992], [929, 1001]];
  expected.forEach(([x, y], index) => {
    const p = orbitPose(index, 2.1, 0);
    assert.deepEqual([p.x, p.y, p.scale, p.alpha, p.behind], [x, y, 1, 1, false]);
  });
});

test('NEX orbit: all four modules travel distinct, finite ellipse poses', () => {
  const starts = [0, 1, 2, 3].map((index) => orbitPose(index, 0, 1));
  const ends = [0, 1, 2, 3].map((index) => orbitPose(index, 1.45, 1));
  for (let i = 0; i < 4; i++) {
    const a = starts[i];
    const b = ends[i];
    for (const v of [a.x, a.y, a.scale, a.alpha, b.x, b.y]) assert.ok(Number.isFinite(v));
    assert.ok(Math.hypot(a.x - b.x, a.y - b.y) > 60, 'Module must actually orbit');
    assert.ok(a.scale > 0.7 && a.scale < 1.2, 'Perspective must remain subtle');
  }
  assert.equal(starts[2].behind, true, 'Rear arc must render behind the body');
  assert.ok(starts.some((pose) => !pose.behind), 'Foreground arc must remain in front');
  assert.notEqual(ORBIT_ANGULAR_SPEED, 0);
});

test('NEX movement is deterministic and seamlessly periodic', () => {
  const p = orbitPose(2, 1.3, 1);
  const period = (Math.PI * 2) / ORBIT_ANGULAR_SPEED;
  const q = orbitPose(2, period + 1.3, 1);
  assert.ok(Math.abs(p.x - q.x) < 1e-9);
  assert.ok(Math.abs(p.y - q.y) < 1e-9);
  assert.deepEqual(idlePose(1.75), idlePose(1.75));
});

test('Blink opens and closes without altering original artwork', () => {
  assert.equal(blinkScale(0), 1);
  assert.ok(blinkScale(2.12) < 1, 'First blink should visibly close');
  assert.ok(blinkScale(7.6) < 1, 'Second blink should visibly close');
  assert.ok(blinkScale(8.12) < 1, 'Double blink should visibly close');
});

test('Invalid rig parameters are bounded or rejected', () => {
  assert.equal(clamp01(Infinity), 0);
  assert.equal(smoothstep(-1), 0);
  assert.equal(smoothstep(5), 1);
  assert.throws(() => orbitPose(4, 1, 1), RangeError);
  assert.throws(() => orbitPose(-1, 1, 1), RangeError);
});

test('NEX idle: all modules have small, distinct non-repetitive secondary drift', () => {
  const poses = [0, 1, 2, 3].map((i) => idleModulePose(i, 4.25));
  const displacements = new Set(poses.map((p) => p.dx.toFixed(3) + ':' + p.dy.toFixed(3)));
  assert.equal(displacements.size, 4, 'modules should not all float in sync');
  for (const p of poses) {
    assert.ok(Math.abs(p.dx) <= 3.2 && Math.abs(p.dy) <= 4.4);
    assert.ok(Math.abs(p.rotation) <= 0.018);
    assert.ok(p.scale > 0.99 && p.scale < 1.01);
  }
  assert.deepEqual(poses, [0, 1, 2, 3].map((i) => idleModulePose(i, 4.25)));
});

test('NEX orbital crossing: no abrupt full-sprite jump between front and rear', () => {
  assert.equal(orbitFrontWeight(-1, 0), 1, 'idle preserves canonical front layer');
  assert.equal(orbitFrontWeight(-1, 1), 0, 'fully rear goes behind body');
  assert.equal(orbitFrontWeight(1, 1), 1, 'fully front stays in front');
  const middle = orbitFrontWeight(0, 1);
  assert.ok(middle > 0.45 && middle < 0.55, 'at crossing both passes blend');
  let prev = orbitFrontWeight(-1, 1);
  for (let step = 1; step <= 1000; step++) {
    const now = orbitFrontWeight(-1 + step * 0.002, 1);
    assert.ok(now >= prev, 'front weight cannot reverse its depth priority');
    assert.ok(Math.abs(now - prev) < 0.02, 'opacity crossing must be continuous');
    prev = now;
  }
});

test('NEX orbital painting weights change continuously while returning to idle', () => {
  let before = orbitFrontWeight(-0.9, 0);
  for (let step = 1; step <= 100; step++) {
    const after = orbitFrontWeight(-0.9, step / 100);
    assert.ok(Math.abs(after - before) < 0.04);
    before = after;
  }
});

test('NEX mobile page keeps canvas unobscured and controls available without scrolling', () => {
  const html = readFileSync(new URL('../public/nex/rig-v1/index.html', import.meta.url), 'utf8');
  assert.match(html, /id="quick-orbit"/);
  assert.match(html, /id="quick-idle"/);
  assert.match(html, /\.stage\{min-height:500px\}/);
  assert.match(html, /\.stage-shortcuts\{display:flex\}/);
  assert.match(html, /document\.getElementById\('quick-orbit'\)\.addEventListener\('click'/);
  assert.match(html, /document\.getElementById\('quick-idle'\)\.addEventListener\('click'/);
});
