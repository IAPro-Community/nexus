import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

// Keep the source/production-art verification independent of the temporary
// output directory and of the caller's current working directory.
const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const source = (relative) => resolve(repoRoot, relative);
const hash = (bytes) => createHash('sha256').update(bytes).digest('hex');
const manifest = JSON.parse(readFileSync(source('docs/design/NEX_VISUAL_MANIFEST.json'), 'utf8'));

test('locked NEX first-paint atlas is byte-for-byte identical to the visual contract', () => {
  const atlas = manifest.animation.spriteSheet;
  assert.equal(atlas.sha256.length, 64);
  assert.ok(existsSync(source(atlas.file)), `missing approved atlas: ${atlas.file}`);
  assert.equal(hash(readFileSync(source(atlas.file))), atlas.sha256);
  assert.equal(atlas.framesPerAction, 8);
  assert.equal(atlas.actions, 12);
});

test('the owner-approved procedural rig modules match the immutable manifest', () => {
  const rig = manifest.animation.runtime_render;
  assert.ok(rig);
  assert.equal(rig.kind, 'owner_approved_layered_rig');
  assert.ok(Array.isArray(rig.modules) && rig.modules.length >= 2);
  for (const module of rig.modules) {
    assert.ok(existsSync(source(module.file)), `missing rig module: ${module.file}`);
    assert.equal(hash(readFileSync(source(module.file))), module.sha256, module.file);
  }
  const component = readFileSync(source(rig.component), 'utf8');
  assert.match(component, /<NexPixelSprite/);
  assert.match(component, /onError/);
  assert.match(component, /data-nex-art=\{rigReady \? 'rig-v1' : undefined\}/);
});

test('production builder only verifies copied art, never rewrites NEX source assets', () => {
  const builder = readFileSync(source('web/scripts/build.mjs'), 'utf8');
  // The public -> dist copy filters contract-only reference art; the copy call
  // itself (and the ordering guarantee) stays the same.
  const copyAt = builder.indexOf('await cp(publicDir, distDir, {');
  const verifyAt = builder.indexOf('const nexAtlasPath = resolve(distDir');
  assert.ok(copyAt >= 0, 'build must copy approved public assets to dist');
  assert.ok(verifyAt > copyAt, 'hash validation must happen after copy into dist');
  assert.ok(builder.includes(manifest.animation.spriteSheet.sha256));
  assert.ok(builder.includes('nexAtlasHash !== nexAtlasSha'));
  assert.doesNotMatch(builder, /export-nex-art\.mjs/);
  assert.doesNotMatch(builder, /writeFile\s*\(\s*resolve\(\s*publicDir/);
});
