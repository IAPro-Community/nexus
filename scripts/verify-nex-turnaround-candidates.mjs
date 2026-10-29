#!/usr/bin/env node
/** Verify six *candidate* NEX camera poses; never approve them as a 360° atlas. */
import { createHash } from 'node:crypto';
import { readFileSync, existsSync } from 'node:fs';
import { dirname, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const dir = resolve(root, 'docs/design/nex/reference-pack/turnaround-v1');
const fail = [];
const test = (ok, why) => { if (!ok) fail.push(why); };
const digest = (file) => createHash('sha256').update(readFileSync(file)).digest('hex');
const manifest = JSON.parse(readFileSync(resolve(dir, 'manifest.json'), 'utf8'));
const expected = new Set(['front', 'right_shallow', 'right_three_quarter', 'rear', 'left_three_quarter', 'left_shallow']);

test(manifest.status === 'REFERENCE_CANDIDATE_NOT_360', 'Candidate pack must not be falsely approved');
test(manifest.turntable_activation === false, 'Turntable must be disabled for this reference pack');
test(manifest.alpha === false, 'Opaque navy background must be reported truthfully');
test(Array.isArray(manifest.poses) && manifest.poses.length === 6, 'Expected exactly six artwork candidates');
test(new Set(manifest.poses.map(p => p.id)).size === 6, 'Duplicate pose IDs');
for (const pose of manifest.poses) {
  test(expected.delete(pose.id), 'Unexpected candidate pose: ' + pose.id);
  test(pose.id === 'front' || pose.id === 'rear' ? [0, 180].includes(pose.angle_degrees_verified) : pose.angle_degrees_verified == null,
    'Unmeasured angles must remain unverified: ' + pose.id);
  test(/^[0-9a-f]{64}$/.test(pose.source_sha256 || ''), 'Invalid source hash: ' + pose.id);
  const asset = pose.webp;
  const file = resolve(dir, asset.path || '');
  test(file.startsWith(dir + sep), 'Path traversal in pose ' + pose.id);
  if (!file.startsWith(dir + sep) || !existsSync(file)) {
    fail.push('Missing pose: ' + pose.id);
    continue;
  }
  const data = readFileSync(file);
  test(data.length === asset.bytes, 'Size mismatch: ' + pose.id);
  test(digest(file) === asset.sha256, 'SHA-256 mismatch: ' + pose.id);
  test(data.subarray(0, 4).toString() === 'RIFF' && data.subarray(8, 16).toString() === 'WEBPVP8L',
    'Not lossless WebP: ' + pose.id);
  if (data.length >= 25 && data.subarray(8, 16).toString() === 'WEBPVP8L') {
    const [b0, b1, b2, b3] = data.subarray(21, 25);
    const width = 1 + (((b1 & 0x3f) << 8) | b0);
    const height = 1 + (((b3 & 0x0f) << 10) | (b2 << 2) | ((b1 & 0xc0) >> 6));
    test(width === 256 && height === 256, 'Pose native dimensions must be exactly 256px: ' + pose.id);
  }
  test(asset.lossless === true && asset.alpha === false, 'Incorrect WebP fidelity metadata: ' + pose.id);
}
test(expected.size === 0, 'Missing expected view names: ' + [...expected].join(', '));
test(existsSync(resolve(dir, 'contact-sheet.webp')), 'Missing contact sheet');
test(existsSync(resolve(dir, 'index.html')), 'Missing local QA page');
if (process.argv.includes('--require-360')) fail.push('Six independently generated poses do not certify a true 16-view 360° atlas');
if (fail.length) {
  fail.forEach(error => console.error('FAIL:', error));
  process.exitCode = 1;
} else console.log('NEX poses: PASS — 6/6 lossless images at 256px with SHA; NOT approved for 360°/HUD');
