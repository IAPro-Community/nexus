#!/usr/bin/env node
// Structural traceability only. A referenced file is not acceptance or release evidence.
import { existsSync, readFileSync } from 'node:fs';
import { relative, resolve, sep } from 'node:path';
import process from 'node:process';

const root = resolve(import.meta.dirname, '..');
function option(name, fallback) {
  const index = process.argv.indexOf(name);
  if (index < 0) return fallback;
  const value = process.argv[index + 1];
  if (!value || value.startsWith('--')) throw new Error(`missing value for ${name}`);
  return resolve(value);
}
const contractPath = option('--contract', resolve(root, 'docs/reference/agent-workspace-v1.json'));
const mapPath = option('--map', resolve(root, 'docs/reference/agent-workspace-v1-evidence-map.json'));
const errors = [];
const check = (ok, message) => { if (!ok) errors.push(message); };
const readJson = (file) => JSON.parse(readFileSync(file, 'utf8'));

function safeTrackedReference(ref, kind, featureId, criterionIndex) {
  check(typeof ref === 'string' && ref.length > 0, `${featureId}[${criterionIndex}]: empty ${kind} reference`);
  if (typeof ref !== 'string' || !ref) return;
  const absolute = resolve(root, ref);
  const inside = absolute === root || absolute.startsWith(root + sep);
  check(inside && !ref.split(/[\\/]/).includes('..'), `${featureId}[${criterionIndex}]: unsafe ${kind} path ${ref}`);
  if (!inside) return;
  check(existsSync(absolute), `${featureId}[${criterionIndex}]: missing ${kind} path ${ref}`);
  if (kind === 'test') check(/(_test\.go|\.test\.[jt]sx?)$/.test(ref), `${featureId}[${criterionIndex}]: not a test path ${ref}`);
}

try {
  const contract = readJson(contractPath);
  const map = readJson(mapPath);
  check(map.schemaVersion === 1, 'unsupported map schema');
  check(map.contract === 'docs/reference/agent-workspace-v1.json', 'map points to a different contract');
  check(map.policy === 'TRACEABILITY_ONLY_NO_ACCEPTANCE_CLAIMS', 'map must remain explicitly non-certifying');

  const requirements = Array.isArray(contract.features) ? contract.features : [];
  const entries = Array.isArray(map.features) ? map.features : [];
  check(entries.length === requirements.length, 'map inventory size differs from canonical contract');
  const byId = new Map();
  for (const entry of entries) {
    if (!entry || typeof entry.id !== 'string') { errors.push('malformed map entry'); continue; }
    check(!byId.has(entry.id), `duplicate map entry ${entry.id}`);
    byId.set(entry.id, entry);
  }

  for (const feature of requirements) {
    const entry = byId.get(feature.id);
    check(Boolean(entry), `${feature.id}: mapping missing`);
    if (!entry) continue;
    check(entry.verification === 'pending', `${feature.id}: map cannot certify acceptance`);
    check(Array.isArray(entry.criteria), `${feature.id}: criteria mapping missing`);
    const criteria = Array.isArray(entry.criteria) ? entry.criteria : [];
    check(criteria.length === feature.acceptance.length, `${feature.id}: acceptance clause count differs`);
    const seen = new Set();
    for (const criterion of criteria) {
      const index = criterion?.acceptanceIndex;
      check(Number.isInteger(index) && index >= 0 && index < feature.acceptance.length,
        `${feature.id}: invalid acceptance index ${index}`);
      if (!Number.isInteger(index)) continue;
      check(!seen.has(index), `${feature.id}: duplicate acceptance index ${index}`);
      seen.add(index);
      check(criterion.acceptance === feature.acceptance[index], `${feature.id}[${index}]: acceptance text drift`);
      check(criterion.coverage === 'unverified', `${feature.id}[${index}]: acceptance cannot be certified by a traceability map`);
      check(['source-and-test-referenced', 'source-only', 'unmapped'].includes(criterion.mappingStatus),
        `${feature.id}[${index}]: invalid mapping status`);
      const sources = Array.isArray(criterion.sourceRefs) ? criterion.sourceRefs : [];
      const tests = Array.isArray(criterion.testRefs) ? criterion.testRefs : [];
      check(sources.length > 0 || criterion.mappingStatus === 'unmapped',
        `${feature.id}[${index}]: missing source reference requires unmapped status`);
      check(tests.length > 0 || criterion.mappingStatus !== 'source-and-test-referenced',
        `${feature.id}[${index}]: test reference required by status`);
      check(sources.length === 0 || criterion.mappingStatus !== 'unmapped',
        `${feature.id}[${index}]: source exists but status is unmapped`);
      check(tests.length === 0 || criterion.mappingStatus === 'source-and-test-referenced',
        `${feature.id}[${index}]: test references conflict with mapping status`);
      for (const ref of sources) safeTrackedReference(ref, 'source', feature.id, index);
      for (const ref of tests) safeTrackedReference(ref, 'test', feature.id, index);
      if (criterion.mappingStatus !== 'source-and-test-referenced')
        check(typeof criterion.gap === 'string' && criterion.gap.trim().length >= 12,
          `${feature.id}[${index}]: explain source/test mapping gap`);
    }
    for (let index = 0; index < feature.acceptance.length; index++)
      check(seen.has(index), `${feature.id}: acceptance index ${index} is unmapped`);
  }
  for (const id of byId.keys()) check(requirements.some((feature) => feature.id === id), `unknown map entry ${id}`);

  if (errors.length) {
    for (const error of errors) console.error('evidence-map: ' + error);
    process.exitCode = 1;
  } else {
    const criteria = entries.flatMap((feature) => feature.criteria);
    const totals = Object.fromEntries(['source-and-test-referenced', 'source-only', 'unmapped']
      .map((status) => [status, criteria.filter((item) => item.mappingStatus === status).length]));
    console.log(`workspace evidence map structurally valid (${entries.length} capabilities, ${criteria.length} acceptance clauses; ${JSON.stringify(totals)}; no acceptance certified)`);
  }
} catch (error) {
  console.error('evidence-map: ' + (error?.message || error));
  process.exitCode = 1;
}
