import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const scriptsDir = dirname(fileURLToPath(import.meta.url));
const webDir = resolve(scriptsDir, '..');
const states = ['idle', 'working', 'warning', 'critical', 'offline'];
const fingerprint = (filename) => createHash('sha256').update(readFileSync(filename)).digest('hex');

test('NEX build export renders all moods without modifying committed public SVGs', () => {
  const output = mkdtempSync(resolve(tmpdir(), 'nexus-nex-art-'));
  try {
    const publicAssets = states.map((state) => resolve(webDir, 'public', 'nex', `nex-${state}.svg`));
    const previous = publicAssets.map(fingerprint);
    const run = spawnSync(process.execPath, [resolve(scriptsDir, 'export-nex-art.mjs'), `--output-dir=${output}`], {
      cwd: webDir,
      encoding: 'utf8',
    });
    assert.equal(run.status, 0, run.stderr || run.stdout);
    for (const state of states) {
      const file = resolve(output, `nex-${state}.svg`);
      const body = readFileSync(file, 'utf8');
      assert.match(body, /<svg[\s\S]*data-nex-art="32bit-official"/);
      assert.match(body, /data-nex-facing="front"/);
      assert.ok(body.endsWith('</svg>\n'), `missing SVG end: ${state}`);
    }
    assert.deepEqual(publicAssets.map(fingerprint), previous, 'runtime generation modified tracked NEX reference assets');
  } finally {
    rmSync(output, { force: true, recursive: true });
  }
});

test('NEX checked-in-art verification never accepts an alternate output directory', () => {
  const output = mkdtempSync(resolve(tmpdir(), 'nexus-nex-check-'));
  try {
    const run = spawnSync(process.execPath, [resolve(scriptsDir, 'export-nex-art.mjs'), '--check', `--output-dir=${output}`], {
      cwd: webDir,
      encoding: 'utf8',
    });
    assert.notEqual(run.status, 0, 'check mode must refuse a noncanonical destination');
  } finally {
    rmSync(output, { force: true, recursive: true });
  }
});
