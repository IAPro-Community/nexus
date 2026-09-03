import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const webDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const repoDir = resolve(webDir, '..');
const distDir = resolve(webDir, 'dist');
const embeddedDir = resolve(repoDir, 'internal/control/web/dist');

describe('build script', () => {
  it('publishes nexus-service-worker.js into the real artifact manifest', () => {
    execFileSync(process.execPath, [resolve(webDir, 'scripts/build.mjs')], {
      cwd: webDir,
      stdio: 'pipe',
    });

    const manifest = JSON.parse(readFileSync(resolve(distDir, 'build-manifest.json'), 'utf8')) as {
      files: string[];
    };

    expect(manifest.files).toContain('nexus-service-worker.js');
    expect(existsSync(resolve(distDir, 'nexus-service-worker.js'))).toBe(true);
    expect(existsSync(resolve(embeddedDir, 'nexus-service-worker.js'))).toBe(true);
  });
});
