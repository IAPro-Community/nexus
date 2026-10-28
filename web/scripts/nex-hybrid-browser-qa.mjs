/**
 * Real browser QA of the NEX faithful raster layering preview.
 *
 * The files are not yet in the release tree. Run this AFTER importing
 * nex_hybrid_github_assets_v1_1.zip, using:
 *
 *   cd web && npm run art:nex:hybrid:qa
 *
 * Tests run against the REAL index.html and PNGs, rendered in Chromium
 * from an in-memory HTML document. This also works where file:// and
 * localhost navigation are blocked by sandbox policy.
 *
 * Outputs:
 *   web/dist/nex-hybrid-qa/{idle,eyes-closed,offline}.png
 *   web/dist/nex-hybrid-qa/result.json
 *
 * This does not certify native HUD/windows or approve artistic fidelity.
 */
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const webDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const root = resolve(webDir, '..');
const assetDirectory = resolve(webDir, 'public/nex/hybrid-v1');
const contract = JSON.parse(readFileSync(
  resolve(root, 'docs/design/nex/hybrid-v1/asset-manifest.json'), 'utf8'));
const output = resolve(webDir, 'dist/nex-hybrid-qa');
const required = contract.assets.filter((asset) =>
  asset.path.startsWith('web/public/nex/hybrid-v1/'));

const missing = required.filter((asset) => !existsSync(resolve(root, asset.path)));
if (missing.length) {
  const message = 'NEX hybrid browser QA NOT_RUN: missing original verified binary assets: '
    + missing.map((asset) => asset.filename).join(', ');
  if (process.argv.includes('--require-assets')) {
    console.error(message);
    process.exit(1);
  }
  console.warn(message);
  process.exit(0);
}

const asDataUrl = {};
for (const asset of required) {
  const bytes = readFileSync(resolve(root, asset.path));
  const sha = createHash('sha256').update(bytes).digest('hex');
  if (sha !== asset.sha256 || bytes.length !== asset.bytes) {
    throw new Error(`NEX hybrid asset SHA-256/size mismatch: ${asset.filename}`);
  }
  if (asset.filename.endsWith('.png')) {
    asDataUrl[asset.filename] = `data:image/png;base64,${bytes.toString('base64')}`;
  }
}

let html = readFileSync(resolve(assetDirectory, 'index.html'), 'utf8');
for (const [name, dataUrl] of Object.entries(asDataUrl)) {
  html = html.split(`"${name}"`).join(`"${dataUrl}"`);
}
// Dynamic layers are added in the original preview JS. Redirect their
// img.src to the exact verified in-memory PNG instead of a file URL.
if (!html.includes('img.src=item.file;')
  || !html.includes("const figure=document.getElementById('nex')")) {
  throw new Error('Original NEX HTML asset-loading contract changed');
}
html = html.replace('img.src=item.file;', 'img.src=__nexAssetData[item.file]||item.file;');
html = html.replace(
  "const figure=document.getElementById('nex')",
  `const __nexAssetData=${JSON.stringify(asDataUrl)};const figure=document.getElementById('nex')`
);

mkdirSync(output, { recursive: true });
const result = {
  status: 'NOT_RUN',
  sourceSha: contract.source_sha256,
  checks: [],
  nativeHudCertified: false,
  humanVisualApproval: 'PENDING',
};
const errors = [];
const check = (ok, label) => {
  result.checks.push({ label, ok: Boolean(ok) });
  if (!ok) errors.push(label);
};

const browser = await chromium.launch({
  headless: true,
  args: ['--no-sandbox', '--disable-dev-shm-usage'],
});
try {
  const page = await browser.newPage({ viewport: { width: 1280, height: 820 } });
  const pageErrors = [];
  page.on('pageerror', (error) => pageErrors.push(String(error)));
  await page.setContent(html, { waitUntil: 'load', timeout: 30_000 });
  await page.locator('.module').first.waitFor();
  await page.getByLabel('Microações aleatórias discretas').uncheck();
  check(await page.locator('.module').count() === 4, '4 independent modules');
  check(await page.locator('.eyes').count() === 2, '2 independently animatable visor eyes');
  check(await page.locator('#idles button').count() === 6, '6 P0 idle actions');

  await page.screenshot({ path: resolve(output, 'idle.png') });

  await page.getByRole('button', { name: 'Piscar' }).click();
  check(await page.locator('#nex').getAttribute('data-action') === 'idle_blink',
    'blink triggered');
  // Capture a deterministic closed-eyes frame for human review.
  await page.locator('.eyes').evaluateAll((nodes) => {
    for (const node of nodes) {
      node.style.setProperty('animation', 'none', 'important');
      node.style.transform = 'scaleY(0.05)';
    }
  });
  await page.screenshot({ path: resolve(output, 'eyes-closed.png') });
  await page.locator('.eyes').evaluateAll((nodes) => {
    for (const node of nodes) {
      node.style.removeProperty('animation');
      node.style.removeProperty('transform');
    }
  });

  await page.getByRole('button', { name: 'Olhar' }).click();
  check(await page.locator('#nex').getAttribute('data-action') === 'idle_glance',
    'glance triggered');
  await page.getByRole('button', { name: 'Módulos' }).click();
  check(await page.locator('#nex').getAttribute('data-action') === 'idle_orbit_breathe',
    'module breathing triggered');

  for (const mood of ['working', 'warning', 'critical', 'offline']) {
    await page.getByRole('button', { name: mood, exact: true }).click();
    check(await page.locator('#nex').getAttribute('data-state') === mood,
      `${mood} state projected`);
    check(!await page.locator('#nex').getAttribute('data-action'),
      `${mood} preempts idle action`);
  }
  const offlineAnimation = await page.locator('.module').first.evaluate(
    (node) => getComputedStyle(node).animationName);
  check(offlineAnimation === 'none', 'offline disables module animations');
  await page.screenshot({ path: resolve(output, 'offline.png') });

  await page.getByLabel('Movimento ativo').uncheck();
  const frozenAnimation = await page.locator('#nex').evaluate(
    (node) => getComputedStyle(node).animationName);
  check(frozenAnimation === 'none', 'explicit motion off disables figure animation');

  await page.getByRole('button', { name: 'idle', exact: true }).click();
  await page.getByLabel('Movimento ativo').check();
  await page.emulateMedia({ reducedMotion: 'reduce' });
  const reduced = await page.locator('.module').first.evaluate(
    (node) => getComputedStyle(node).animationName);
  check(reduced === 'none', 'prefers-reduced-motion disables animations');

  check(pageErrors.length === 0, 'zero browser JS errors');
  result.browserErrors = pageErrors;
  result.status = errors.length ? 'FAIL' : 'PASS_BROWSER_ONLY';
  writeFileSync(resolve(output, 'result.json'), JSON.stringify(result, null, 2) + '\n');
  if (errors.length) throw new Error('NEX browser QA failed: ' + errors.join('; '));
  console.log('NEX raster browser QA PASS: ' + result.checks.length
    + ' checks, see dist/nex-hybrid-qa/. Native/visual certification still pending.');
} finally {
  await browser.close();
}
