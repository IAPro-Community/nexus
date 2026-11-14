#!/usr/bin/env node
import { createHash } from 'node:crypto';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const local = (path) => resolve(root, path);
const read = (path) => readFileSync(local(path), 'utf8');
const failures = [];
const check = (pass, message) => { if (!pass) failures.push(message); };
const manifest = JSON.parse(read('docs/design/NEX_VISUAL_MANIFEST.json'));
const source = read(manifest.source_of_truth.compact_runtime_component);
const atlasSource = read('web/src/features/hud/nexIdleAtlas.ts');
const guardianModel = read(manifest.source_of_truth.runtime_mood_type);
const companionModel = read(manifest.source_of_truth.companion_mood_type);
const integration = read('web/src/features/hud/GuardianMascot.tsx');
const contract = read(manifest.source_of_truth.normative_contract);
const spriteStyles = read(manifest.animation.runtimeStyles);
const companionStyles = read(manifest.animation.interactionStyles);
const values = (union, typeName) => {
  const match = union.match(new RegExp(`export type ${typeName}\\s*=\\s*([^;]+);`));
  if (!match) return [];
  return [...match[1].matchAll(/'([^']+)'/g)].map((m) => m[1]);
};
const equalSets = (a, b) =>
  [...new Set(a)].sort().join('|') === [...new Set(b)].sort().join('|');

check(manifest.identity.status === 'OFFICIAL_VISUAL_DIRECTION', 'visual identity is not approved');
check(manifest.identity.default_view === 'FRONT', 'NEX standard pose must be FRONT');
for (const [field, count] of [
  ['vertical_fin_count', 2],
  ['ear_coupling_count', 2],
  ['orbital_module_count', 4],
  ['reactor_core_count', 1],
]) check(manifest.anatomy[field] === count, `invalid anatomical count: ${field}`);

const sourceMoods = values(guardianModel, 'GuardianMood');
const companionMoods = values(companionModel, 'CompanionMood');
const manifestMoods = manifest.source_of_truth.runtime_states || [];
check(equalSets(sourceMoods, manifestMoods),
  `runtime moods out of sync: manifest=${manifestMoods} source=${sourceMoods}`);
check(equalSets(companionMoods, manifest.source_of_truth.companion_ui_moods || []),
  `Companion moods out of sync: manifest=${manifest.source_of_truth.companion_ui_moods} source=${companionMoods}`);
check(equalSets(manifestMoods, manifest.operational_policy.only_bind_implemented_moods || []),
  'runtime and binding-policy mood lists differ');
check(integration.includes("mood === 'attention' ? 'warning' : mood"),
  'Companion attention -> warning projection changed: review artwork mapping');
check(manifest.source_of_truth.mood_projection?.attention === 'warning',
  'manifest attention mapping is not warning');

check(source.includes('data-nex-art="hud-32bit-atlas"'), 'runtime lacks the locked atlas art marker');
check(source.includes('data-nex-mood={mood}'), 'runtime does not project the mood onto the sprite');
check(source.includes('data-idle-action={idleAction ?? undefined}'),
  'runtime does not expose the active idle action for tests');

// The locked atlas model must bind to the identity lock exactly.
const atlasSha = manifest.animation.spriteSheet.sha256;
check(source.includes("from './nexIdleAtlas'") && source.includes('NEX_ATLAS_HOLD_LAST_ACTIONS.has(idleAction)'),
  'sprite component no longer binds to the locked atlas model');
check(atlasSource.includes(`'${atlasSha}'`), 'atlas model SHA differs from the visual manifest');
check(atlasSource.includes("'/nex/hud-32bit/nex-idle-atlas-128.png'"),
  'atlas model URL drifted from the published sprite sheet');
check(atlasSource.includes('NEX_ATLAS_FRAME_MS = 125') && atlasSource.includes('NEX_ATLAS_FRAMES = 8'),
  'atlas timing must stay at 8 frames of 125 ms');
check((atlasSource.match(/idle_\w+:\s*\d+/g) || []).length === 12,
  'atlas model must declare exactly the twelve approved action rows');
check(atlasSource.includes("'idle_drowse'") && atlasSource.includes("'idle_wake'"),
  'drowse/wake must be declared as hold-last-frame actions');

const atlasBytes = existsSync(local(manifest.animation.spriteSheet.file))
  ? readFileSync(local(manifest.animation.spriteSheet.file))
  : null;
check(Boolean(atlasBytes), 'published sprite sheet bytes are missing');
if (atlasBytes) {
  check(createHash('sha256').update(atlasBytes).digest('hex') === atlasSha,
    'published sprite sheet SHA-256 mismatch');
}

// Mood treatment lives in the runtime styles now, not in vector accents.
check(spriteStyles.includes('$mood-warning'), 'runtime styles lack the warning treatment');
check(spriteStyles.includes('$mood-critical'), 'runtime styles lack the critical treatment');
check(spriteStyles.includes(manifest.palette.warning.replace('#', '').toLowerCase()) ||
  spriteStyles.toLowerCase().includes(manifest.palette.warning.toLowerCase()),
  'warning treatment color differs from the manifest palette');
check(spriteStyles.toLowerCase().includes(manifest.palette.critical.toLowerCase()),
  'critical treatment color differs from the manifest palette');
check(spriteStyles.includes('grayscale(1)'), 'offline must render still and desaturated');

const artwork = manifest.references || [];
check(artwork.filter((ref) => ref.status === 'VERSIONED').length >= 3,
  'not enough committed visual reference drawings');
for (const entry of artwork) {
  if (entry.status !== 'VERSIONED') continue;
  check(Boolean(entry.file) && existsSync(local(entry.file || '')),
    `missing versioned reference: ${entry.file}`);
  if (!entry.file?.endsWith('.svg') || !existsSync(local(entry.file))) continue;
  const svg = read(entry.file);
  check(svg.startsWith('<svg') && svg.trimEnd().endsWith('</svg>'),
    `invalid SVG envelope: ${entry.file}`);
  check(svg.includes('data-nex-figure="front"'), `missing frontal figure: ${entry.file}`);
  check(svg.split('<g').length === svg.split('</g>').length,
    `unbalanced SVG graphic groups: ${entry.file}`);
}
const expressions = artwork.find((ref) => ref.id === 'expressions');
if (expressions?.file && existsSync(local(expressions.file))) {
  const svg = read(expressions.file);
  for (const mood of manifestMoods) {
    check(svg.includes(`data-state="${mood}"`),
      `expression sheet lacks the canonical mood ${mood}`);
  }
}

const highFidelity = artwork.find((ref) => ref.id === 'approved_high_fidelity_concept');
check(Boolean(highFidelity), 'approved original concept missing from manifest');
if (highFidelity) {
  check(/^[a-f0-9]{64}$/.test(highFidelity.sha256 || ''),
    'approved original raster needs a real SHA-256');
  check(highFidelity.width > 0 && highFidelity.height > 0,
    'approved original raster dimensions missing');
  check(highFidelity.format === 'PNG', 'approved original must be PNG');
  const target = highFidelity.file || highFidelity.expected_repository_path;
  if (highFidelity.status === 'VERSIONED') {
    check(Boolean(target) && existsSync(local(target || '')),
      'original concept marked VERSIONED but bytes are missing');
    if (target && existsSync(local(target))) {
      const bytes = readFileSync(local(target));
      const digest = createHash('sha256').update(bytes).digest('hex');
      check(digest === highFidelity.sha256, 'approved original SHA-256 mismatch');
      check(bytes.length === highFidelity.bytes, 'approved original byte size mismatch');
      const signature = bytes.subarray(0, 8).toString('hex');
      check(signature === '89504e470d0a1a0a', 'approved original is not a PNG');
      if (bytes.length >= 24) {
        check(bytes.readUInt32BE(16) === highFidelity.width
          && bytes.readUInt32BE(20) === highFidelity.height,
        'approved original dimensions mismatch');
      }
    }
  } else {
    check(highFidelity.status === 'IDENTIFIED_BINARY_NOT_VERSIONED',
      'original concept status is neither versioned nor identified as pending');
    check(highFidelity.file === null, 'missing binary cannot have a committed file path');
    if (highFidelity.expected_repository_path && existsSync(local(highFidelity.expected_repository_path))) {
      check(false, 'original concept exists but manifest still says NOT_VERSIONED');
    }
  }
}
const animation = manifest.animation;
check(Boolean(animation), 'manifest has no NEX animation contract');
if (animation) {
  check(animation.implementation === 'LOCKED_RASTER_ATLAS_FRAMES',
    'NEX animation implementation changed without visual review');
  check(animation.spriteSheetPublished === true,
    'the locked atlas is published; manifest must not claim otherwise');
  const animationDocument = read(animation.contract);
  check(animationDocument.includes('quatro') && animationDocument.includes('offline'),
    'animation contract omits canonical anatomy or idle safety');
  check(integration.includes("data-motion={motionEnabled ? 'on' : 'off'}"),
    'NEX Guardian no longer projects requested motion mode');
  // Frame playback walks one atlas row with CSS steps(); the wrapper stays an
  // anchor and never transforms.
  check(spriteStyles.includes('@keyframes nex-atlas-play'),
    'missing atlas frame playback keyframes');
  check(spriteStyles.includes('steps(8'), 'atlas playback must step through the 8 frames');
  check(spriteStyles.includes('.l-body') === false && source.includes('.rig') === false,
    'unexpected hybrid-v1 site layer classes leaked into the HUD sprite');
  for (const row of ['.rowHover', '.rowBlink', '.rowDrowse', '.rowWake']) {
    check(spriteStyles.includes(row), `missing atlas row mapping: ${row}`);
  }
  check(spriteStyles.includes('prefers-reduced-motion: reduce'),
    'reduced-motion override missing from atlas runtime styles');
  check(spriteStyles.includes(".sprite[data-nex-mood='offline']"),
    'offline stillness rule missing from atlas runtime styles');
  check(companionStyles.includes(".root[data-motion='off'] .guardianButton [data-nex-art]"),
    'explicitly disabled motion does not cover the atlas sprite');
}

check(contract.includes('assets/nex/nex-character-reference.svg'),
  'contract lacks original frontend anatomy drawing');
check(contract.includes('NEX_VISUAL_MANIFEST.json'), 'contract lacks machine-readable manifest');
check(contract.includes('critical'), 'contract ignores critical attention state');
check(!contract.includes('sandbox:/'), 'contract cannot link ephemeral chat file paths');

if (failures.length) {
  for (const failure of failures) console.error(`FAIL: ${failure}`);
  process.exitCode = 1;
} else {
  console.log('NEX identity: structural parity PASS (visual/native certification remains independent)');
}
