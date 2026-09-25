#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import path from 'node:path';

export function nexusBinaryPath(repoRoot) {
  return path.resolve(repoRoot, process.platform === 'win32' ? 'nexus.exe' : 'nexus');
}

function run(command, args, options, label) {
  const result = spawnSync(command, args, {
    ...options,
    stdio: 'inherit',
  });
  if (result.error) {
    throw new Error(`${label} failed to start: ${result.error.message}`);
  }
  if (result.status !== 0) {
    throw new Error(`${label} failed with exit code ${result.status ?? 'unknown'}`);
  }
}

export function buildNexusForE2E(repoRoot) {
  const webDir = path.resolve(repoRoot, 'web');
  const binPath = nexusBinaryPath(repoRoot);

  run(process.execPath, ['scripts/build.mjs'], { cwd: webDir }, 'frontend build');
  run(
    process.platform === 'win32' ? 'go.exe' : 'go',
    ['build', '-trimpath', '-buildvcs=false', '-o', binPath, './cmd/nexus'],
    { cwd: repoRoot },
    'nexus CLI build',
  );

  if (!existsSync(binPath)) {
    throw new Error(`nexus binary missing after build: ${binPath}`);
  }
  return binPath;
}

export function ensureNexusBinary(repoRoot) {
  const binPath = nexusBinaryPath(repoRoot);
  if (process.env.SKIP_BUILD !== '1') return buildNexusForE2E(repoRoot);
  if (!existsSync(binPath)) {
    throw new Error(
      `nexus binary missing at ${binPath}; unset SKIP_BUILD or build the current checkout first`,
    );
  }
  return binPath;
}
