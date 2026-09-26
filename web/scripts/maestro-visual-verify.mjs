#!/usr/bin/env node
import { spawn } from 'node:child_process';
import assert from 'node:assert/strict';
import { existsSync, mkdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';
import { ensureNexusBinary, nexusBinaryPath } from './e2e-platform.mjs';
import { terminateProcess, terminateProcessOnExit } from './e2e-fixture.mjs';
import AxeBuilder from '@axe-core/playwright';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const output = path.join(root, '.tempmediaStorage', 'maestro');
mkdirSync(output, { recursive: true });

const binPath = nexusBinaryPath(root);
ensureNexusBinary(root);

const server = spawn(binPath, ['web', '--port', '0', '--listen', '127.0.0.1', '--no-open'], {
  cwd: root,
  detached: true,
  stdio: ['ignore', 'pipe', 'pipe'],
});

const stop = async () => {
  await terminateProcess(server);
};
process.on('exit', () => terminateProcessOnExit(server));

const bootstrap = await new Promise((resolve, reject) => {
  let buffer = '';
  const timer = setTimeout(() => reject(new Error('Timed out waiting for Nexus bootstrap')), 12000);
  server.stdout.on('data', (chunk) => {
    buffer += chunk.toString();
    const match = buffer.match(/Bootstrap:\s*(http:\/\/127\.0\.0\.1:\d+\/\?token=[a-f0-9]+)/i);
    if (match) {
      clearTimeout(timer);
      resolve(match[1]);
    }
  });
  server.stderr.on('data', (chunk) => {
    buffer += chunk.toString();
  });
  server.on('exit', (code) => reject(new Error(`Nexus exited with ${code}: ${buffer}`)));
});

const browser = await chromium.launch({
  executablePath: existsSync('/usr/bin/google-chrome') ? '/usr/bin/google-chrome' : undefined,
  headless: true,
  args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
});
const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
const page = await context.newPage();
const errors = [];
page.on('pageerror', (error) => errors.push(String(error?.message || error)));

try {
  await page.goto(bootstrap, { waitUntil: 'domcontentloaded', timeout: 15000 });
  await page.waitForSelector('.nx-os-shell', { timeout: 10000 });
  await page.setViewportSize({ width: 1440, height: 900 });
  const tools = page.getByRole('button', { name: 'Ferramentas' });
  if ((await tools.count()) > 0 && (await tools.getAttribute('aria-expanded')) !== 'true') {
    await tools.click({ force: true });
  }
  await page.locator('.nx-rail-tool-btn[title="Maestro"]').click({ force: true });
  await page.getByTestId('maestro-surface').waitFor({ state: 'visible', timeout: 10000 });

  for (const viewport of [
    [320, 568],
    [390, 844],
    [768, 1024],
    [1024, 768],
    [1440, 900],
  ]) {
    const [width, height] = viewport;
    await page.setViewportSize({ width, height });
    await page.waitForTimeout(150);
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1);
    assert.equal(overflow, false, `Maestro must not overflow horizontally at ${width}px`);
    await page.screenshot({ path: path.join(output, `maestro-${width}x${height}.png`), fullPage: true });
  }

  const search = page.getByPlaceholder(/Buscar skills|Search skills/i);
  await search.fill('security');
  await page.getByRole('button', { name: /Limpar|Clear/i }).click();
  await search.focus();
  assert.equal(await page.evaluate(() => document.activeElement?.tagName), 'INPUT');

  const axe = await new AxeBuilder({ page }).analyze();
  const serious = axe.violations.filter((item) => ['serious', 'critical'].includes(item.impact));
  if (serious.length > 0) {
    console.error(
      serious
        .map(
          (item) =>
            `${item.id}: ${item.nodes.map((node) => `${node.html} (${node.failureSummary || 'no summary'})`).join(' | ')}`,
        )
        .join('\n'),
    );
  }
  assert.equal(serious.length, 0, `A11y violations: ${serious.map((item) => item.id).join(', ')}`);
  assert.equal(errors.length, 0, `Browser errors: ${errors.join(' | ')}`);
  console.log(`Maestro visual verification PASS; screenshots: ${output}`);
} finally {
  await context.close();
  await browser.close();
  await stop();
}
