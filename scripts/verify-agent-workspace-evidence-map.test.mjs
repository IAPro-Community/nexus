import { test } from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';

const root = resolve(import.meta.dirname, '..');
const script = join(root, 'scripts/verify-agent-workspace-evidence-map.mjs');
const mapFile = join(root, 'docs/reference/agent-workspace-v1-evidence-map.json');
const run = (map) => spawnSync(process.execPath, [script, '--map', map], { cwd: root, encoding: 'utf8' });

function withMap(mutator, callback) {
  const folder = mkdtempSync(join(tmpdir(), 'nexus-evidence-map-'));
  try {
    const map = JSON.parse(readFileSync(mapFile, 'utf8'));
    mutator(map);
    const file = join(folder, 'map.json');
    writeFileSync(file, JSON.stringify(map));
    callback(file);
  } finally { rmSync(folder, { recursive: true, force: true }); }
}

test('maps every canonical acceptance clause while keeping release verification pending', () => {
  const result = run(mapFile);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /48 capabilities, 96 acceptance clauses/);
  assert.match(result.stdout, /no acceptance certified/);
});

test('rejects missing acceptance mappings and attempted certification by the map', () => {
  withMap((map) => { map.features[0].criteria.pop(); }, (file) => {
    const result = run(file);
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /acceptance clause count differs/);
  });
  withMap((map) => { map.features[0].criteria[0].coverage = 'verified'; }, (file) => {
    const result = run(file);
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /cannot be certified/);
  });
});

test('rejects stale, missing, unsafe, and misclassified test references', () => {
  withMap((map) => { map.features[0].criteria[0].sourceRefs = ['../outside.go']; }, (file) => {
    const result = run(file);
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /unsafe source path/);
  });
  withMap((map) => { map.features[0].criteria[0].testRefs = ['internal/missing_test.go']; }, (file) => {
    const result = run(file);
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /missing test path/);
  });
  withMap((map) => { map.features[0].criteria[0].mappingStatus = 'source-only'; }, (file) => {
    const result = run(file);
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /test references conflict with mapping status/);
  });
});
