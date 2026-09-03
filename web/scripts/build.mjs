import { build } from 'esbuild';
import { cp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const webDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const repoDir = resolve(webDir, '..');
const distDir = resolve(webDir, 'dist');
const embeddedDir = resolve(repoDir, 'internal/control/web/dist');

await rm(distDir, { recursive: true, force: true });
await mkdir(distDir, { recursive: true });

const tailwindCli = resolve(webDir, 'node_modules/@tailwindcss/cli/dist/index.mjs');
const css = spawnSync(process.execPath, [tailwindCli, '-i', 'src/index.css', '-o', 'dist/bundle.css', '--minify'], { cwd: webDir, stdio: 'inherit' });
if (css.status !== 0) process.exit(css.status ?? 1);

await build({
  absWorkingDir: webDir,
  entryPoints: ['src/index.tsx'],
  outfile: 'dist/bundle.js',
  bundle: true,
  minify: true,
  format: 'esm',
  platform: 'browser',
  target: ['es2022'],
  jsx: 'automatic',
  define: { 'process.env.NODE_ENV': '"production"' },
  logLevel: 'info',
});

await cp(resolve(webDir, 'index.html'), resolve(distDir, 'index.html'));
const logo = existsSync(resolve(webDir, 'public/logo.png')) ? resolve(webDir, 'public/logo.png') : resolve(repoDir, 'logo.png');
if (existsSync(logo)) await cp(logo, resolve(distDir, 'logo.png'));
const serviceWorker = resolve(webDir, 'public/nexus-service-worker.js');
if (existsSync(serviceWorker)) await cp(serviceWorker, resolve(distDir, 'nexus-service-worker.js'));

const manifest = {
  files: ['index.html', 'bundle.css', 'bundle.js', ...(existsSync(resolve(distDir, 'logo.png')) ? ['logo.png'] : []), ...(existsSync(resolve(distDir, 'nexus-service-worker.js')) ? ['nexus-service-worker.js'] : [])],
  builder: 'node+esbuild+tailwind',
};
await writeFile(resolve(distDir, 'build-manifest.json'), JSON.stringify(manifest, null, 2) + '\n');

await rm(embeddedDir, { recursive: true, force: true });
await mkdir(embeddedDir, { recursive: true });
await cp(distDir, embeddedDir, { recursive: true });

const index = await readFile(resolve(distDir, 'index.html'), 'utf8');
if (!index.includes('./bundle.js') || !index.includes('./bundle.css')) throw new Error('index.html is missing bundle references');
console.log(`Nexus web build complete: ${distDir}`);
