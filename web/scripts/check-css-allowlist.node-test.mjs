import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const webDirectory = path.resolve(scriptDirectory, '..');
const checkerPath = path.join(scriptDirectory, 'check-css-allowlist.js');
const fixturePath = path.join(
  webDirectory,
  'src',
  'styles',
  `__check-css-allowlist-${process.pid}-${randomUUID()}.scss`,
);

test('Sass partials are scoped to CSS Modules, never exempted for global styles', () => {
  const baseline = spawnSync(process.execPath, [checkerPath], {
    cwd: webDirectory,
    encoding: 'utf8',
  });
  assert.equal(baseline.status, 0, baseline.stderr || baseline.stdout);

  try {
    fs.writeFileSync(fixturePath, "@use '../features/hud/hudCapsule';\n", {
      flag: 'wx',
    });
    const withGlobalImporter = spawnSync(process.execPath, [checkerPath], {
      cwd: webDirectory,
      encoding: 'utf8',
    });

    assert.equal(withGlobalImporter.status, 1, withGlobalImporter.stdout);
    assert.match(withGlobalImporter.stderr, /src\/features\/hud\/_hudCapsule\.scss/);
  } finally {
    if (fs.existsSync(fixturePath)) {
      fs.unlinkSync(fixturePath);
    }
  }
});
