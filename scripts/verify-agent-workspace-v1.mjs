#!/usr/bin/env node
// Nexus V1 contract validator. The default mode checks scope structure only.
// --release consumes external CI evidence for the EXACT candidate SHA.
import { existsSync, readFileSync, realpathSync } from 'node:fs';
import { resolve, sep } from 'node:path';
import { execFileSync } from 'node:child_process';
import process from 'node:process';

const root = resolve(import.meta.dirname, '..');
const expectedGroups = Object.freeze({
  experience: 7, organization: 6, execution: 6, multiagent: 6,
  browser: 6, remote: 6, differentiator: 4, foundation: 7,
});
const requiredPlatforms = ['linux-amd64', 'windows-amd64', 'macos-arm64'];
const allowedOwners = new Set(['NEXUS', 'MAESTRO', 'INTEGRATION']);
const allowedSourceStatuses = new Set(['present', 'partial', 'missing']);
const failures = [];

function option(name) {
  const at = process.argv.indexOf(name);
  if (at < 0) return null;
  const value = process.argv[at + 1];
  if (!value || value.startsWith('--')) throw new Error('Missing value for ' + name);
  return value;
}

function check(ok, message) { if (!ok) failures.push(message); }
function readJson(file) { return JSON.parse(readFileSync(file, 'utf8')); }
function actionUrl(value) {
  return typeof value === 'string' &&
    /^https:\/\/github\.com\/IAPro-Community\/nexus\/actions\/runs\/[0-9]+(?:\/.*)?$/.test(value);
}

function verifyManifest(doc) {
  check(doc.schemaVersion === 1 && doc.target === '1.0.0', 'Invalid v1 contract schema/target');
  check(doc.branch === 'release/v1.0.0', 'Contract must target canonical release branch');
  check(doc.policy === 'ALL_REQUIRED_FEATURES_VERIFIED_EXACT_SHA', 'Release must be fail-closed');
  const items = Array.isArray(doc.features) ? doc.features : [];
  check(items.length === 48, 'V1 contract must enumerate all 48 requirements');
  const ids = new Set();
  const groups = {};
  for (const f of items) {
    if (!f || typeof f !== 'object') { failures.push('Malformed feature'); continue; }
    check(typeof f.id === 'string' && /^[a-z][a-z0-9-]+$/.test(f.id), 'Invalid feature id');
    check(!ids.has(f.id), 'Duplicated feature: ' + f.id);
    ids.add(f.id);
    groups[f.category] = (groups[f.category] || 0) + 1;
    check(typeof f.name === 'string' && f.name.trim().length >= 6, f.id + ': name');
    check(allowedOwners.has(f.owner), f.id + ': invalid owner');
    check(allowedSourceStatuses.has(f.sourceStatus), f.id + ': invalid source status');
    check(f.releaseRequired === true, f.id + ': cannot drop mandatory scope');
    check(Number.isInteger(f.issue) && f.issue > 0, f.id + ': missing tracked issue');
    check(f.verification === 'pending' || f.verification === 'verified', f.id + ': invalid verification');
    check(Array.isArray(f.acceptance) && f.acceptance.length >= 2 &&
      f.acceptance.every(v => typeof v === 'string' && v.trim().length >= 16),
      f.id + ': acceptance must be specific');
  }
  for (const [group, count] of Object.entries(expectedGroups))
    check(groups[group] === count, 'Missing or duplicated requirements in ' + group);
  for (const group of Object.keys(groups))
    check(Object.hasOwn(expectedGroups, group), 'Unexpected group ' + group);
  return items;
}

function checkExternalProof(doc, features, expectedSha, expectedRunId, filename) {
  check(/^[0-9a-f]{40}$/.test(expectedSha || ''), 'Specify exact 40-character SHA');
  check(/^[1-9][0-9]*$/.test(expectedRunId || ''), 'Specify exact CI run id');
  let actualSha = '';
  try { actualSha = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: root, encoding: 'utf8' }).trim(); }
  catch { failures.push('Cannot verify local checkout SHA'); }
  check(actualSha === expectedSha, 'Checkout SHA differs from candidate SHA');
  check(typeof filename === 'string' && filename.length > 0, 'Missing external exact-SHA evidence artifact');
  if (!filename || !existsSync(filename)) return;
  let realFile = '';
  try { realFile = realpathSync(filename); }
  catch { failures.push('Cannot access evidence artifact'); return; }
  check(!realFile.startsWith(realpathSync(root) + sep), 'Evidence must be external CI artifact, not source-controlled fixture');
  let proof;
  try { proof = readJson(realFile); }
  catch { failures.push('Invalid CI evidence JSON'); return; }
  check(proof.schemaVersion === 1 && proof.sourceSha === expectedSha, 'Evidence SHA/schema mismatch');
  check(String(proof.ciRunId) === expectedRunId, 'Evidence is not from selected exact-SHA CI run');
  check(proof.producer === 'nexus-v1-agent-workspace-e2e', 'Evidence producer missing');
  const platformMap = proof.platforms || {};
  for (const platform of requiredPlatforms) {
    const item = platformMap[platform] || {};
    check(item.status === 'passed' && item.sourceSha === expectedSha && actionUrl(item.jobUrl),
      'Platform certification missing/invalid: ' + platform);
  }
  const records = Array.isArray(proof.features) ? proof.features : [];
  check(records.length === features.length, 'Incomplete feature evidence inventory');
  const seen = new Set();
  const byId = new Map();
  for (const item of records) {
    if (!item || typeof item.id !== 'string') { failures.push('Malformed proof feature'); continue; }
    check(!seen.has(item.id), 'Duplicated proof: ' + item.id);
    seen.add(item.id); byId.set(item.id, item);
  }
  for (const f of features) {
    const item = byId.get(f.id) || {};
    check(f.sourceStatus === 'present' && f.verification === 'verified', f.id + ': source/integration not completed');
    check(item.status === 'passed' && item.sourceSha === expectedSha, f.id + ': exact-SHA proof missing');
    check(actionUrl(item.jobUrl) && Array.isArray(item.tests) && item.tests.length > 0, f.id + ': test provenance missing');
    if (f.category === 'multiagent' || f.category === 'remote' || f.id === 'two-client-real-e2e')
      check(Number.isInteger(item.clients) && item.clients >= 2, f.id + ': two-client proof missing');
  }
}

try {
  const file = option('--manifest') || resolve(root, 'docs/reference/agent-workspace-v1.json');
  const contract = readJson(file);
  const features = verifyManifest(contract);
  if (process.argv.includes('--release')) {
    checkExternalProof(contract, features, option('--sha'), option('--ci-run-id'), option('--evidence'));
  }
} catch (error) { failures.push(String(error.message || error)); }
if (failures.length) {
  for (const issue of failures) console.error('v1 workspace: ' + issue);
  process.exitCode = 1;
} else {
  console.log('Nexus V1 Agent Workspace contract verified (scope ' +
    Object.values(expectedGroups).reduce((a, b) => a + b, 0) +
    (process.argv.includes('--release') ? ', exact-SHA evidence accepted)' : ', certification still pending)'));
}
