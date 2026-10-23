#!/usr/bin/env node
import { readFileSync, existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const read = (path) => readFileSync(resolve(root, path), 'utf8');
const manifestPath = 'docs/design/NEX_VISUAL_MANIFEST.json';
const errors = [];
const assert = (condition, message) => { if (!condition) errors.push(message); };
const manifest = JSON.parse(read(manifestPath));
const source = read(manifest.source_of_truth.compact_runtime_component);
const contract = read(manifest.source_of_truth.normative_contract);

assert(manifest.identity.status === 'OFFICIAL_VISUAL_DIRECTION', 'visual identity is not official');
assert(manifest.identity.default_view === 'FRONT', 'front pose is not the default');
assert(manifest.anatomy.vertical_fin_count === 2, 'invalid fin count');
assert(manifest.anatomy.ear_coupling_count === 2, 'invalid ear coupling count');
assert(manifest.anatomy.orbital_module_count === 4, 'invalid module count');
assert(manifest.anatomy.reactor_core_count === 1, 'invalid reactor count');
assert(Array.isArray(manifest.references) && manifest.references.length >= 3, 'missing references');
assert(source.includes('data-nex-art="32bit-official"'), 'runtime missing official art mark');
assert(source.includes('data-nex-facing="front"'), 'runtime has wrong default facing');
assert(source.includes('data-nex-module='), 'runtime missing orbital module marker');
assert(source.includes('viewBox="0 0 128 128"'), 'Compact geometry changed without manifest update');
assert((source.match(/data-nex-module=/g) || []).length === 1, 'module marker should be generated from shared module mapping');
const coordinates = [...source.matchAll(/\[\s*\d+\s*,\s*\d+\s*,\s*[0-3]\s*\]/g)];
assert(coordinates.length === 4, 'unexpected number of orbital module coordinates; review required');
for (const mood of ['idle', 'working', 'warning', 'offline']) {
  assert(manifest.source_of_truth.runtime_states.includes(mood), `runtime mood missing: ${mood}`);
  assert(source.includes(mood === 'idle' ? "mood" : mood), `source no longer supports mood: ${mood}`);
}
for (const ref of manifest.references) {
  if (ref.status !== 'VERSIONED') continue;
  assert(Boolean(ref.file) && existsSync(resolve(root, ref.file)), `missing art: ${ref.file}`);
  if (!ref.file?.endsWith('.svg')) continue;
  const svg = read(ref.file);
  assert(svg.startsWith('<svg') && svg.trimEnd().endsWith('</svg>'), `invalid SVG envelope: ${ref.file}`);
  assert(svg.includes('data-nex-figure="front"'), `missing shared NEX figure: ${ref.file}`);
  const gOpen = (svg.match(/<g(?:\s|>)/g) || []).length;
  const gClose = (svg.match(/<\/g>/g) || []).length;
  assert(gOpen === gClose, `unbalanced graphic groups: ${ref.file}`);
}
assert(contract.includes('assets/nex/nex-character-reference.svg'), 'normative contract does not link official art');
assert(contract.includes('NEX_VISUAL_MANIFEST.json'), 'normative contract lacks machine-readable manifest');
assert(!contract.includes('sandbox:/'), 'contract cannot reference chat-only sandbox URLs');
if (errors.length) {
  for (const error of errors) console.error('FAIL:', error);
  process.exitCode = 1;
} else {
  console.log('NEX identity structural parity: PASS (human visual certification remains required)');
}
