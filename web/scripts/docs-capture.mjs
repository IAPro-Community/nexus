#!/usr/bin/env node
import { spawn, spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { chromium } from 'playwright';

const webDir = path.resolve(import.meta.dirname, '..');
const root = path.resolve(webDir, '..');
const outputDir = path.join(root, 'docs/assets/screenshots');
const demoDir = path.join(root, 'docs/assets/demos');
mkdirSync(outputDir, { recursive: true });
mkdirSync(demoDir, { recursive: true });
const build = spawnSync('make', ['build'], { cwd: root, stdio: 'inherit' });
if (build.status !== 0) process.exit(build.status ?? 1);

const server = spawn(path.join(root, 'nexus'), ['web', '--port', '0', '--listen', '127.0.0.1', '--no-open'], {
  cwd: root, detached: true, stdio: ['ignore', 'pipe', 'pipe'],
});
let output = '';
const bootstrap = new Promise((resolve, reject) => {
  const timer = setTimeout(() => reject(new Error(`bootstrap timeout\n${output}`)), 15000);
  const onData = (chunk) => {
    output += chunk.toString();
    const match = output.match(/Bootstrap:\s*(http:\/\/127\.0\.0\.1:\d+\/\?token=[a-f0-9]+)/i);
    if (match) { clearTimeout(timer); resolve(match[1]); }
  };
  server.stdout.on('data', onData);
  server.stderr.on('data', onData);
  server.on('exit', (code) => { clearTimeout(timer); reject(new Error(`server exited ${code}\n${output}`)); });
});
const stop = () => {
  try { process.kill(-server.pid, 'SIGTERM'); } catch {}
};
process.on('exit', stop);

const sourceSha = spawnSync('git', ['rev-parse', 'HEAD'], { cwd: root, encoding: 'utf8' }).stdout.trim();
const captures = [];
try {
  const url = await bootstrap;
  const browser = await chromium.launch({
    executablePath: existsSync('/usr/bin/google-chrome') ? '/usr/bin/google-chrome' : undefined,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
  });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  const pageErrors = [];
  page.on('pageerror', (error) => pageErrors.push(String(error?.message || error)));
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 20000 });
  await page.waitForSelector('.nx-os-shell', { timeout: 15000 });
  await page.evaluate(() => {
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    const nodes = [];
    let node;
    while ((node = walker.nextNode())) nodes.push(node);
    for (const text of nodes) {
      text.nodeValue = text.nodeValue.replace(/\/projetos\/[^\s]+/g, '/workspace/demo');
    }
  });
  const capture = async (id, file, surface) => {
    await page.screenshot({ path: path.join(outputDir, file), fullPage: false });
    captures.push({ id, file, source_sha: sourceSha, surface, scenario: 'real-local-bootstrap', viewport: '1440x900', automated: true, capture_command: 'bun run docs:capture', status: 'PASS' });
  };
  await capture('VIS-001', 'workspace-overview.png', 'web/workspace');
  const candidates = [
    { id: 'VIS-007', file: 'composer.png', surface: 'web/composer', pattern: /composer/i },
    { id: 'VIS-008', file: 'flow.png', surface: 'web/flow', pattern: /flow/i },
    { id: 'VIS-006', file: 'agents.png', surface: 'web/agents', pattern: /agents|agentes/i },
  ];
  for (const candidate of candidates) {
    const button = page.getByRole('button', { name: candidate.pattern }).first();
    if (await button.count() && await button.isVisible().catch(() => false)) {
      await button.click();
      await page.waitForTimeout(300);
      await capture(candidate.id, candidate.file, candidate.surface);
    }
  }
  if (pageErrors.length) console.warn(`page errors during capture: ${pageErrors.join(' | ')}`);
  await context.close();
  await browser.close();
} finally {
  stop();
}
const manifest = { source_sha: sourceSha, generated_by: 'web/scripts/docs-capture.mjs', captures };
writeFileSync(path.join(outputDir, 'manifest.json'), `${JSON.stringify(manifest, null, 2)}\n`);
console.log(`captured ${captures.length} real product surface(s) at ${sourceSha}`);
