#!/usr/bin/env node
import { spawn, spawnSync } from 'node:child_process';
import assert from 'node:assert/strict';
import { mkdirSync, rmSync, existsSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';

const webDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const root = path.resolve(webDir, '..');
const output = path.join(root, '.tempmediaStorage', 'task-preparation');
const fixture = path.join(os.tmpdir(), `nexus-task-preparation-${process.pid}`);
mkdirSync(output, { recursive: true });
mkdirSync(fixture, { recursive: true });
writeFileSync(path.join(fixture, 'AGENTS.md'), '# Visual test fixture\n\nRead DEV before acting.\n', { mode: 0o600 });

const build = spawnSync('make', ['build'], { cwd: root, stdio: 'inherit' });
assert.equal(build.status, 0, 'make build must pass before visual verification');

const server = spawn(path.join(root, 'nexus'), ['web', '--port', '0', '--listen', '127.0.0.1', '--no-open'], {
  cwd: root,
  detached: true,
  stdio: ['ignore', 'pipe', 'pipe'],
});

const stop = () => {
  try {
    process.kill(-server.pid, 'SIGTERM');
  } catch {}
};
process.on('exit', () => {
  stop();
  rmSync(fixture, { recursive: true, force: true });
});

const bootstrap = await new Promise((resolve, reject) => {
  let buffer = '';
  const timer = setTimeout(() => reject(new Error(`Timed out waiting for bootstrap: ${buffer}`)), 15000);
  const onData = (chunk) => {
    buffer += chunk.toString();
    const match = buffer.match(/Bootstrap:\s*(http:\/\/127\.0\.0\.1:\d+\/\S+)/i);
    if (match) {
      clearTimeout(timer);
      resolve(match[1]);
      return;
    }
    const base = buffer.match(/URL:\s*(http:\/\/127\.0\.0\.1:\d+)/i);
    if (base) {
      const saved = spawnSync(path.join(root, 'nexus'), ['web', 'url'], { cwd: root, encoding: 'utf8' });
      const savedUrl = saved.stdout?.trim();
      if (saved.status === 0 && savedUrl?.includes('#nexus_bootstrap=')) {
        clearTimeout(timer);
        resolve(savedUrl);
      }
    }
  };
  server.stdout.on('data', onData);
  server.stderr.on('data', onData);
  server.on('exit', (code) => reject(new Error(`Nexus exited with ${code}: ${buffer}`)));
});

const browser = await chromium.launch({
  executablePath: existsSync('/usr/bin/google-chrome') ? '/usr/bin/google-chrome' : undefined,
  headless: true,
  args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
});
const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
const page = await context.newPage();
let csrfToken = '';
const api = async (pathName, options = {}) => {
  const response = await page.evaluate(async ({ pathName, options }) => {
    const result = await fetch(pathName, { ...options, headers: { 'Content-Type': 'application/json', ...(options.headers || {}) } });
    return { status: result.status, body: await result.json().catch(() => ({})) };
  }, { pathName, options });
  assert.ok(response.status >= 200 && response.status < 300, `${pathName}: HTTP ${response.status}`);
  return response.body;
};

try {
  await page.goto(bootstrap, { waitUntil: 'domcontentloaded', timeout: 15000 });
  await page.waitForSelector('.nx-os-shell', { timeout: 10000 });
  const session = await page.evaluate(async () => (await fetch('/api/v1/session')).json());
  csrfToken = session.csrf_token || '';
  assert.ok(csrfToken, 'authenticated browser session must expose CSRF token');
  const project = await api('/api/v1/projects', {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ path: fixture, name: 'Task Preparation Visual Fixture' }),
  });
  const agent = await api(`/api/v1/projects/${project.id}/agents`, {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ name: 'Visual Preparation Agent', role: 'tester' }),
  });
  await api(`/api/v1/projects/${project.id}/context/prepare`, {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ create_context: false }),
  });

  await page.goto(new URL(`/p/${project.id}/agents`, bootstrap).toString(), { waitUntil: 'domcontentloaded' });
  await page.getByRole('button', { name: /Preparar tarefa/i }).click();
  await page.getByRole('heading', { name: /Preparar tarefa/i }).waitFor({ state: 'visible' });
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('textbox').first().fill('Revisar o fluxo de preparação de tarefa.');

  for (const [width, height] of [[320, 568], [390, 844], [768, 1024], [1024, 768], [1440, 900]]) {
    await page.setViewportSize({ width, height });
    await page.waitForTimeout(100);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1), false, `horizontal overflow at ${width}x${height}`);
    await page.screenshot({ path: path.join(output, `task-preparation-${width}x${height}.png`), fullPage: true });
  }

  const axe = await new AxeBuilder({ page }).exclude('.xterm').analyze();
  const serious = axe.violations.filter((item) => ['serious', 'critical'].includes(item.impact));
  if (serious.length > 0) console.error(JSON.stringify(serious, null, 2));
  assert.equal(serious.length, 0, `Axe serious/critical violations: ${serious.map((item) => item.id).join(', ')}`);
  await dialog.getByRole('textbox').first().focus();
  await page.keyboard.press('Tab');
  assert.ok(await page.evaluate(() => document.activeElement), 'keyboard focus must remain in the dialog flow');
  console.log(`Task preparation visual/Axe verification PASS; screenshots: ${output}`);
} finally {
  await context.close();
  await browser.close();
  stop();
}
