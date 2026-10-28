import { test } from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, readFileSync, writeFileSync, rmSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';

const root = resolve(import.meta.dirname, '..');
const script = join(root, 'scripts/verify-agent-workspace-v1.mjs');
const manifest = join(root, 'docs/reference/agent-workspace-v1.json');
function run(args) {
  return spawnSync(process.execPath, [script, ...args], {cwd: root, encoding: 'utf8'});
}

test('source contract is structurally complete without pretending to certify it', () => {
  const p = run([]);
  assert.equal(p.status, 0, p.stderr);
  assert.match(p.stdout, /certification still pending/);
});

test('release refuses unverified contract without externally produced evidence', () => {
  const p = run(['--release', '--sha', '0'.repeat(40), '--ci-run-id', '123']);
  assert.notEqual(p.status, 0);
  assert.match(p.stderr, /external exact-SHA evidence|Checkout SHA differs/);
});

test('contract cannot silently drop, duplicate or waive required features', () => {
  const temp = mkdtempSync(join(tmpdir(), 'nexus-v1-scope-'));
  try {
    const doc = JSON.parse(readFileSync(manifest, 'utf8'));
    doc.features[0].releaseRequired = false;
    doc.features[1].id = doc.features[0].id;
    const mutated = join(temp, 'altered.json');
    writeFileSync(mutated, JSON.stringify(doc));
    const p = run(['--manifest', mutated]);
    assert.notEqual(p.status, 0);
    assert.match(p.stderr, /cannot drop mandatory scope/);
    assert.match(p.stderr, /Duplicated feature/);
  } finally { rmSync(temp, {recursive: true, force: true}); }
});
