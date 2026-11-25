#!/usr/bin/env node
// Validate planning scopes; NEVER grant publication authority. The existing
// full exact-SHA signed verifier remains authoritative until a profile-bound
// evidence schema and producer are approved.
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('..', import.meta.url));
const read = (file) => JSON.parse(readFileSync(resolve(root, file), 'utf8'));
const fail = (message) => { throw new Error(message); };
const contract = read('docs/reference/agent-workspace-v1.json');
const profiles = read('docs/reference/agent-workspace-v1-profiles.json');
const ids = contract.features.map((feature) => feature.id);
const mandatory = [
  'terminal-studio', 'agent-command-center', 'remote-companion', 'remote-terminal',
  'workspace-sharing', 'mobile-inbox', 'device-handoff', 'runtime-health',
  'workspace-snapshots', 'attention-radar', 'browser-automation', 'live-preview',
  'remote-generated-outputs', 'single-execution-owner', 'maestro-canonical-memory',
  'two-client-real-e2e', 'cross-platform-exact-sha',
];
const expectedOptional = {
  'comparison-lab': ['COMPARISON_LAB', 'FLOW', 'MISSIONS'],
  'mission-observatory': ['MISSIONS'],
  'smart-layouts': ['SMART_LAYOUTS'],
  'design-inspect': ['DESIGN_INSPECT'],
  'visual-diff': ['VISUAL_DIFF'],
  'device-preview': ['DEVICE_PREVIEW'],
  'voice-actions': ['VOICE_ACTIONS'],
  'parallel-challenge': ['PARALLEL_CHALLENGE', 'FLOW', 'MISSIONS'],
  'composer-flow-onboarding': ['COMPOSER', 'FLOW', 'MISSIONS'],
};
const sameSet = (a, b) => Array.isArray(a) && Array.isArray(b) &&
  a.length === b.length && new Set(a).size === a.length &&
  a.every((id) => b.includes(id));
if (profiles.schemaVersion !== 1 || profiles.status !== 'planning-only' ||
    profiles.releaseGate !== 'full-until-signed-profile-evidence') fail('Invalid planning contract');
if (ids.length !== 48 || new Set(ids).size !== 48) fail('Historic full scope changed');
if (!sameSet(profiles.full.features, ids)) fail('Full inventory must remain intact');
if (!sameSet(Object.keys(profiles.optional.features), Object.keys(expectedOptional))) {
  fail('Optional classification drift');
}
for (const [id, keys] of Object.entries(expectedOptional)) {
  if (!sameSet(profiles.optional.features[id], keys)) fail('Invalid flag dependencies: ' + id);
}
if (!sameSet([...profiles.core.features, ...Object.keys(expectedOptional)], ids)) {
  fail('Core and Optional must partition the historic contract without omissions');
}
if (mandatory.some((id) => !profiles.core.features.includes(id))) fail('Mandatory safety gate removed');
if (profiles.core.global !== false || profiles.optional.global !== true || profiles.full.global !== true) {
  fail('Invalid runtime profile');
}
if (process.argv.includes('--release')) fail('Planning profiles are not publication certification');
console.log('Core: ' + profiles.core.features.length + '; Optional: ' +
  Object.keys(expectedOptional).length + '; Full: ' + ids.length +
  '. Planning only; signed full release gate unchanged.');
