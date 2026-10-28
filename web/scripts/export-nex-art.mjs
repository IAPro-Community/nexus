/**
 * Export five canonical static SVG renders from the same React source used by the HUD.
 * This prevents a second, drifting mascot design for docs/marketing assets.
 *
 * Usage: node scripts/export-nex-art.mjs [--check]
 */
import { build } from 'esbuild';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

const scriptsDir = dirname(fileURLToPath(import.meta.url));
const webDir = resolve(scriptsDir, '..');
const check = process.argv.includes('--check');
const outArg = process.argv.find((arg) => arg.startsWith('--output-dir='));
const outputDir = outArg
  ? resolve(outArg.slice('--output-dir='.length))
  : resolve(webDir, 'public/nex');
// --check validates the committed exports; an explicit alternate destination
// is for generated build output, not a way to change the provenance target.
if (check && outArg) {
  throw new Error('--check must validate the canonical web/public/nex exports');
}
const tempPath = resolve(scriptsDir, `.nex-art-export-${process.pid}.cjs`);

const result = await build({
  stdin: {
    contents: "export { NexPixelSprite } from '../src/features/hud/NexPixelSprite';",
    resolveDir: scriptsDir,
    sourcefile: 'nex-svg-export.tsx',
    loader: 'tsx',
  },
  bundle: true,
  platform: 'node',
  format: 'cjs',
  target: 'node22',
  jsx: 'automatic',
  packages: 'external',
  write: false,
  logLevel: 'silent',
  plugins: [{
    name: 'ignore-presentation-only-scss',
    setup(builder) {
      builder.onLoad({ filter: /\.module\.scss$/ }, () => ({
        contents: 'const styles = new Proxy({}, { get: (_target, name) => String(name) }); export default styles;',
        loader: 'js',
      }));
    },
  }],
});

try {
  await writeFile(tempPath, result.outputFiles[0].text);
  const { NexPixelSprite } = createRequire(import.meta.url)(tempPath);
  if (typeof NexPixelSprite !== 'function') {
    throw new Error('Unable to load canonical NEX sprite component');
  }
  if (!check) await mkdir(outputDir, { recursive: true });

  for (const mood of ['idle', 'working', 'warning', 'critical', 'offline']) {
    const rendered = renderToStaticMarkup(React.createElement(NexPixelSprite, { mood }));
    if (!rendered.includes('data-nex-art="32bit-official"') ||
        !rendered.includes('data-nex-facing="front"')) {
      throw new Error(`Invalid NEX identity for ${mood}`);
    }
    const svg = rendered.replace('<svg ', '<svg xmlns="http://www.w3.org/2000/svg" ') + '\n';
    const file = resolve(outputDir, `nex-${mood}.svg`);
    if (check) {
      const saved = await readFile(file, 'utf8');
      if (saved !== svg) throw new Error(`NEX SVG is stale: ${file}. Run npm run art:nex`);
    } else {
      await writeFile(file, svg, 'utf8');
    }
  }

  console.log(`NEX 32-bit official SVG ${check ? 'verified' : 'exported'}: ${outputDir}`);
} finally {
  await rm(tempPath, { force: true });
}
