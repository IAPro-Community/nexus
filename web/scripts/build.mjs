import { build, transform } from 'esbuild';
import { createHash } from 'node:crypto';
import { cp, mkdir, readFile, readdir, rm, stat, writeFile } from 'node:fs/promises';
import { existsSync, readFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { dirname, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import * as sass from 'sass';
import { scopeCssModule } from './css-module-scope.mjs';
import { serviceWorkerBuildId } from './service-worker-build-id.mjs';

const webDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const repoDir = resolve(webDir, '..');
const distDir = resolve(webDir, 'dist');
const embeddedDir = resolve(repoDir, 'internal/control/web/embedded');

await rm(distDir, { recursive: true, force: true });
await mkdir(distDir, { recursive: true });

const tailwindCli = resolve(webDir, 'node_modules/@tailwindcss/cli/dist/index.mjs');
const css = spawnSync(
  process.execPath,
  [tailwindCli, '-i', 'src/index.css', '-o', 'dist/bundle.css', '--minify'],
  { cwd: webDir, stdio: 'inherit' },
);
if (css.status !== 0) process.exit(css.status ?? 1);

const collectedStyles = [];

// Contract/reference artwork that must never ship inside the runtime bundle:
// these bytes are the NEX-ID-v1 reference sources kept under web/public/nex/
// for human and automated comparison, not runtime assets. They are filtered
// only on the public -> dist copy; the source tree stays byte-identical.
const NON_RUNTIME_PUBLIC_PATHS = new Set([
  'nex/hybrid-v1/nex_source_original.png',
  'nex/hybrid-v1/reconstructed_idle.png',
  'nex/nex-site-hero.svg',
]);

// esbuild never rewrites declaration values that carry a custom property
// (`--x: 1px`), which is where the expanded Sass and Tailwind layers keep
// their spacing. Every non-comment stretch outside strings is collapsed to a
// single space; strings and quoted URLs are skipped so their bytes survive.
const compressCss = (source) =>
  source
    .replace(/\/\*(?![!])[\s\S]*?\*\//g, '')
    .split(/("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')/)
    .map((chunk, index) => (index % 2 === 1 ? chunk : chunk.replace(/\s+/g, ' ')))
    .join('')
    .replace(/ ?([{};,>]) ?/g, '$1')
    .replace(/ ?\) ?/g, ')')
    .replace(/\( ?/g, '(');

// Sass and the Tailwind CLI keep a space after the colon of a declaration and
// esbuild leaves values untouched when the rule carries a custom property, so
// the declaration delimiter is normalized after serialization.
const compressDeclarations = (css) => css.replace(/: /g, ':');

const minifyCss = async (source) =>
  compressDeclarations(
    (await transform(compressCss(source), { loader: 'css', minify: true })).code,
  );

// The scoped-module pass re-serializes CSS through PostCSS, which re-expands
// compressed Sass output. Minifying each fragment *before* scoping keeps the
// concatenated bundle free of expanded formatting without touching rule order.
const scopeCssModuleMinified = async (compiledCss, moduleIdentity) =>
  scopeCssModule(await minifyCss(compiledCss), moduleIdentity);

function createSassPlugin(targetStyles) {
  return {
    name: 'sass-modules',
    setup(buildInstance) {
      // Global SCSS
      buildInstance.onLoad({ filter: /\.scss$/ }, (args) => {
        if (args.path.includes('.module.')) return;
        const result = sass.compile(args.path, { style: 'compressed' });
        targetStyles.push({ path: args.path, css: result.css });
        return {
          contents: '',
          loader: 'js',
        };
      });

      // SCSS Modules and CSS Modules
      buildInstance.onLoad({ filter: /\.module\.(scss|css)$/ }, async (args) => {
        const isScss = args.path.endsWith('.scss');
        const compiledCss = isScss
          ? sass.compile(args.path, { style: 'compressed' }).css
          : readFileSync(args.path, 'utf8');
        // Hash the module's repository-relative identity so builds are stable
        // when the same checkout is built from a different absolute directory.
        const moduleIdentity = relative(webDir, args.path).replaceAll('\\', '/');
        const { css: transformedCss, classMap } = await scopeCssModuleMinified(
          compiledCss,
          moduleIdentity,
        );
        targetStyles.push({ path: args.path, css: transformedCss });

        return {
          contents: `export default ${JSON.stringify(classMap)};`,
          loader: 'js',
        };
      });
    },
  };
}

await build({
  absWorkingDir: webDir,
  entryPoints: { bundle: 'src/index.tsx' },
  outdir: 'dist',
  chunkNames: 'chunks/[name]-[hash]',
  bundle: true,
  splitting: true,
  treeShaking: true,
  minify: true,
  format: 'esm',
  platform: 'browser',
  target: ['es2022'],
  jsx: 'automatic',
  plugins: [createSassPlugin(collectedStyles)],
  define: {
    'process.env.NODE_ENV': '"production"',
  },
  logLevel: 'info',
});

// Isolated lightweight Nex 2.0 desktop entrypoint (loads zero Workspace OS
// chunks, terminal/Flow UI, or service worker).
await rm(resolve(distDir, 'nex-v2-chunks'), { recursive: true, force: true });
const nexV2Styles = [];
await build({
  absWorkingDir: webDir,
  entryPoints: { 'nex-v2': 'src/nex-v2/index.tsx' },
  outdir: 'dist',
  chunkNames: 'nex-v2-chunks/[name]-[hash]',
  bundle: true,
  splitting: true,
  treeShaking: true,
  minify: true,
  format: 'esm',
  platform: 'browser',
  target: ['es2022'],
  jsx: 'automatic',
  plugins: [createSassPlugin(nexV2Styles)],
  define: {
    'process.env.NODE_ENV': '"production"',
  },
  logLevel: 'info',
});
const nexV2CssPath = resolve(distDir, 'nex-v2.css');
const existingNexV2Css = existsSync(nexV2CssPath) ? await readFile(nexV2CssPath, 'utf8') : '';
const stableNexV2Css = nexV2Styles
  .toSorted((left, right) => left.path.localeCompare(right.path))
  .map((entry) => entry.css)
  .join('\n');
await writeFile(nexV2CssPath, await minifyCss(`${existingNexV2Css}\n${stableNexV2Css}`));
await cp(resolve(webDir, 'nex-v2.html'), resolve(distDir, 'nex-v2.html'));

// Tailwind's standalone CLI and esbuild do not resolve this package CSS
// import into the final stylesheet. xterm relies on these rules for its
// helper textarea and viewport; without them the textarea becomes visible
// over the terminal and keyboard input is rendered as stray characters. Keep
// the vendor stylesheet in the generated bundle so Web, Desktop and the
// embedded UI share the same terminal contract.
// The vendor file starts with an MIT banner; strip leading comments so the
// bundle stays a single valid stylesheet that can be minified as a whole.
const xtermCss = (
  await readFile(resolve(webDir, 'node_modules/xterm/css/xterm.css'), 'utf8')
).replace(/^\s*(?:\/\*[\s\S]*?\*\/\s*)+/, '');
const generatedCssPath = resolve(distDir, 'bundle.css');
const generatedCss = await readFile(generatedCssPath, 'utf8');
await writeFile(generatedCssPath, await minifyCss(`${generatedCss}\n${xtermCss}`));

if (collectedStyles.length > 0) {
  const currentBundleCss = await readFile(resolve(distDir, 'bundle.css'), 'utf8');
  const stableStyles = collectedStyles
    .toSorted((left, right) => left.path.localeCompare(right.path))
    .map((entry) => entry.css)
    .join('\n');
  await writeFile(
    resolve(distDir, 'bundle.css'),
    await minifyCss(currentBundleCss + '\n' + stableStyles),
  );
}

// Global sections live in the index.css graph while SCSS modules live in the
// esbuild graph; the shipped bundle must carry both halves of the token contract.
const requiredTokens = [
  '--nx-elevation-overlay:',
  '--nx-elevation-modal:',
  '--nx-shadow-overlay:',
  '--nx-shadow-modal:',
  '--nx-shadow-drawer-left:',
  '--nx-shadow-drawer-right:',
  '--nx-drop-shadow-overlay:',
  '--nx-space-7:',
  '--nx-font-size-sm:',
  '--nx-radius-md:',
];
const shippedCss = await readFile(resolve(distDir, 'bundle.css'), 'utf8');
const missingTokens = requiredTokens.filter((token) => !shippedCss.includes(token));
if (missingTokens.length > 0) {
  throw new Error(`bundle.css is missing semantic tokens: ${missingTokens.join(', ')}`);
}

await cp(resolve(webDir, 'index.html'), resolve(distDir, 'index.html'));
// Copy static assets first, then verify the locked NEX hud atlas *in dist only*.
// A build must never mutate checked-in web/public source assets: exact-SHA CI
// and Windows homologation verify a clean source worktree.
const publicDir = resolve(webDir, 'public');
if (existsSync(publicDir)) {
  await cp(publicDir, distDir, {
    recursive: true,
    filter: (src) => {
      if (src === publicDir) return true;
      return !NON_RUNTIME_PUBLIC_PATHS.has(relative(publicDir, src).replaceAll('\\', '/'));
    },
  });
}
const serviceWorkerPath = resolve(distDir, 'sw.js');
if (existsSync(serviceWorkerPath)) {
  const serviceWorker = await readFile(serviceWorkerPath, 'utf8');
  const buildAssets = await Promise.all(
    (await readdir(distDir, { recursive: true }))
      .filter((entry) => typeof entry === 'string' && entry.endsWith('.js'))
      .map(async (entry) => ({ name: entry, content: await readFile(resolve(distDir, entry)) })),
  );
  const buildId = serviceWorkerBuildId(buildAssets);
  if (!serviceWorker.includes('__NEXUS_BUILD_ID__')) {
    throw new Error('Service worker is missing the build ID placeholder');
  }
  await writeFile(serviceWorkerPath, serviceWorker.replaceAll('__NEXUS_BUILD_ID__', buildId));
}
// The HUD renders the locked NEX-ID-v1 hud-32bit idle atlas (see
// web/src/features/hud/nexIdleAtlas.ts). The previous vector SVG export was
// retired with the rejected vector direction; the raster asset is the
// canonical art and must ship byte-a-byte.
const nexAtlasPath = resolve(distDir, 'nex', 'hud-32bit', 'nex-idle-atlas-128.png');
const nexAtlasSha = '8256a302fff77996e3563ef42d1e62685b59084c8a17446bb2acb2219e85dfda';
if (!existsSync(nexAtlasPath)) {
  throw new Error(
    'Locked NEX hud-32bit atlas missing from dist: nex/hud-32bit/nex-idle-atlas-128.png',
  );
}
const nexAtlasHash = createHash('sha256').update(readFileSync(nexAtlasPath)).digest('hex');
if (nexAtlasHash !== nexAtlasSha) {
  throw new Error(`Locked NEX hud atlas SHA mismatch in dist: ${nexAtlasHash}`);
}
const logo = existsSync(resolve(webDir, 'public/logo.png'))
  ? resolve(webDir, 'public/logo.png')
  : resolve(repoDir, 'logo.png');
if (existsSync(logo)) await cp(logo, resolve(distDir, 'logo.png'));

const manifest = {
  files: [
    'index.html',
    'bundle.css',
    'bundle.js',
    ...(existsSync(resolve(distDir, 'logo.png')) ? ['logo.png'] : []),
  ],
  builder: 'node+esbuild+tailwind+code-splitting',
};
await writeFile(resolve(distDir, 'build-manifest.json'), JSON.stringify(manifest, null, 2) + '\n');

await mkdir(embeddedDir, { recursive: true });
for (const entry of await readdir(embeddedDir)) {
  if (entry !== '.keep') await rm(resolve(embeddedDir, entry), { recursive: true, force: true });
}
await cp(distDir, embeddedDir, { recursive: true });

const index = await readFile(resolve(distDir, 'index.html'), 'utf8');
if (!index.includes('./bundle.js') || !index.includes('./bundle.css'))
  throw new Error('index.html is missing bundle references');

// Size budget: the whole dist is embedded in three Go binaries, so a quiet
// regression here multiplies by three. The ceilings sit ~10% above the
// measured release numbers; `--check-budget` makes a breach fail the build
// (CI uses it), while the default build only reports the table.
const SIZE_BUDGET = {
  distTotal: 8_000_000,
  bundleCss: 500_000,
  bundleJs: 220_000,
  nexV2Js: 270_000,
  nexV2Css: 50_000,
};
const dirSize = async (dir) => {
  let total = 0;
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const path = resolve(dir, entry.name);
    total += entry.isDirectory() ? await dirSize(path) : (await stat(path)).size;
  }
  return total;
};
const fileSize = async (name) => {
  const path = resolve(distDir, name);
  return existsSync(path) ? (await stat(path)).size : 0;
};
const kb = (bytes) => `${(bytes / 1000).toFixed(1)} KB`;
const sizeRows = [
  ['dist/', await dirSize(distDir), SIZE_BUDGET.distTotal],
  ['bundle.css', await fileSize('bundle.css'), SIZE_BUDGET.bundleCss],
  ['bundle.js', await fileSize('bundle.js'), SIZE_BUDGET.bundleJs],
  ['nex-v2.js', await fileSize('nex-v2.js'), SIZE_BUDGET.nexV2Js],
  ['nex-v2.css', await fileSize('nex-v2.css'), SIZE_BUDGET.nexV2Css],
];
const breached = sizeRows.filter(([, size, limit]) => size > limit);
console.log('Web build size budget:');
for (const [name, size, limit] of sizeRows) {
  const headroom = Math.round((1 - size / limit) * 100);
  console.log(
    `  ${name.padEnd(12)} ${kb(size).padStart(10)} / ${kb(limit).padStart(10)}  ${size > limit ? 'EXCEEDED' : `${headroom}% headroom`}`,
  );
}
if (breached.length > 0) {
  const detail = breached
    .map(([name, size, limit]) => `${name} ${kb(size)} > ${kb(limit)}`)
    .join(', ');
  if (process.argv.includes('--check-budget')) {
    throw new Error(`Web build exceeds the size budget: ${detail}`);
  }
  console.warn(`WARNING: web build exceeds the size budget: ${detail}`);
}
console.log(`Nexus web build complete: ${distDir}`);
