#!/usr/bin/env node
// Run the Go ACP/Registry/API harness and its Playwright browser continuation.
import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const webDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(webDir, '..', '..');
const build = spawnSync('npm', ['--prefix', 'web', 'run', 'build'], {
  cwd: repoRoot,
  encoding: 'utf8',
  stdio: 'inherit',
});
if (build.status !== 0) process.exit(build.status ?? 1);

const test = spawnSync(
  'go',
  [
    'test',
    './internal/control/web',
    '-run',
    '^TestACPAttentionRadarBrowserIntegration$',
    '-count=1',
    '-v',
  ],
  {
    cwd: repoRoot,
    env: { ...process.env, NEXUS_E2E_BROWSER: '1' },
    stdio: 'inherit',
  },
);
process.exit(test.status ?? 1);
