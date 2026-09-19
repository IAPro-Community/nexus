#!/usr/bin/env node
import assert from 'node:assert/strict';
import { spawn, spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const webDir = path.resolve(__dirname, '..');
const repoRoot = path.resolve(webDir, '..');
const binPath = path.resolve(repoRoot, 'nexus');
const outDir = path.resolve(repoRoot, 'DEV/validation/composer-workspace');

mkdirSync(outDir, { recursive: true });

if (process.env.SKIP_BUILD !== '1') {
  const build = spawnSync('make', ['build'], { cwd: repoRoot, stdio: 'inherit' });
  assert.equal(build.status, 0, 'make build must succeed');
}
assert.ok(existsSync(binPath), `nexus binary missing at ${binPath}`);

const isolatedDataDir = path.join(os.tmpdir(), `nexus-composer-e2e-${process.pid}`);
rmSync(isolatedDataDir, { recursive: true, force: true });
const projectPath = path.join(isolatedDataDir, 'project');
mkdirSync(projectPath, { recursive: true });
writeFileSync(path.join(projectPath, 'README.md'), '# Composer E2E fixture\n');
spawnSync('git', ['init', '-q', projectPath], { stdio: 'ignore' });

const serverEnv = {
  ...process.env,
  NEXUS_DATA_DIR: isolatedDataDir,
  AI_CLI_DATA_DIR: isolatedDataDir,
  NEXUS_DOCS_CAPTURE: '1',
};

const server = spawn(binPath, ['web', '--port', '0', '--listen', '127.0.0.1', '--no-open'], {
  cwd: repoRoot,
  detached: true,
  stdio: ['ignore', 'pipe', 'pipe'],
  env: serverEnv,
});

const waitForUrl = () =>
  new Promise((resolve, reject) => {
    let output = '';
    const timer = setTimeout(() => reject(new Error(`bootstrap timeout\n${output}`)), 20000);
    const onData = (chunk) => {
      output += chunk.toString();
      const match = output.match(/URL:\s+(http:\/\/127\.0\.0\.1:\d+)/i);
      if (match) {
        clearTimeout(timer);
        resolve(match[1]);
      }
    };
    server.stdout.on('data', onData);
    server.stderr.on('data', onData);
    server.on('exit', (code) => {
      clearTimeout(timer);
      reject(new Error(`server exited ${code}\n${output}`));
    });
  });

const stop = () => {
  try {
    if (server.pid) process.kill(-server.pid, 'SIGTERM');
  } catch {
    /* already gone */
  }
};

process.on('exit', stop);

async function main() {
  console.log('starting isolated nexus web…');
  const origin = await waitForUrl();
  console.log('origin', origin);

  const browser = await chromium.launch({
    executablePath: existsSync('/usr/bin/google-chrome') ? '/usr/bin/google-chrome' : undefined,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
  });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  page.setDefaultTimeout(12000);

  const listenPath = path.join(isolatedDataDir, 'web-auth', 'listen.json');
  const deadline = Date.now() + 8000;
  while (!existsSync(listenPath) && Date.now() < deadline) {
    await new Promise((r) => setTimeout(r, 50));
  }
  assert.ok(existsSync(listenPath), 'listen.json was not created');
  const listenState = JSON.parse(readFileSync(listenPath, 'utf8'));
  await page.goto(origin, { waitUntil: 'domcontentloaded', timeout: 20000 });
  const boot = await page.evaluate(async (token) => {
    const response = await fetch('/api/v1/auth/bootstrap', {
      method: 'POST',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: JSON.stringify({ token }),
    });
    return { ok: response.ok, status: response.status };
  }, listenState.bootstrap_token);
  assert.ok(boot.ok, `bootstrap failed HTTP ${boot.status}`);
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 20000 });
  console.log('authenticated');

  const session = await page.evaluate(async () => {
    const response = await fetch('/api/v1/session', { headers: { Accept: 'application/json' } });
    if (!response.ok) throw new Error(`session HTTP ${response.status}`);
    return response.json();
  });
  const created = await page.evaluate(
    async ({ workspacePath, csrfToken }) => {
      const response = await fetch('/api/v1/projects', {
        method: 'POST',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
          'X-CSRF-Token': csrfToken,
        },
        body: JSON.stringify({ name: 'Composer E2E', path: workspacePath }),
      });
      if (!response.ok) throw new Error(`project HTTP ${response.status} ${await response.text()}`);
      return response.json();
    },
    { workspacePath: projectPath, csrfToken: session.csrf_token },
  );
  console.log('project', created.id);

  const createdSession = await page.evaluate(
    async ({ projectId, csrfToken }) => {
      const response = await fetch(`/api/v1/projects/${projectId}/composer-sessions`, {
        method: 'POST',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
          'X-CSRF-Token': csrfToken,
        },
        body: JSON.stringify({ goal: 'READY_FOR_MAIN', input_mode: 'IDEA', source_prompt: '' }),
      });
      const text = await response.text();
      return { ok: response.ok, status: response.status, text: text.slice(0, 400) };
    },
    { projectId: created.id, csrfToken: session.csrf_token },
  );
  assert.ok(createdSession.ok, `composer session HTTP ${createdSession.status}: ${createdSession.text}`);
  console.log('composer session created');

  await page.goto(`${origin}/p/${encodeURIComponent(created.id)}/work`, {
    waitUntil: 'domcontentloaded',
    timeout: 20000,
  });
  await page.getByRole('heading', { name: 'Composer' }).waitFor({ timeout: 15000 });
  await page.getByRole('button', { name: /nova elaboração|new elaboration/i }).waitFor();
  await page.getByRole('tab', { name: /briefing/i }).waitFor();
  await page.getByRole('tab', { name: /contexto|context/i }).waitFor();
  await page.getByRole('tab', { name: /prompt/i }).waitFor();
  await page.getByPlaceholder(/escreva uma mensagem|write a message/i).waitFor();
  const options = page.getByRole('option');
  assert.ok((await options.count()) > 0, 'clarification options should be in the thread');

  await page.screenshot({ path: path.join(outDir, 'e2e-desktop-1440.png') });

  await page.setViewportSize({ width: 900, height: 1024 });
  await page.waitForTimeout(300);
  await page.screenshot({ path: path.join(outDir, 'e2e-tablet-900.png') });

  await page.setViewportSize({ width: 390, height: 844 });
  await page.waitForTimeout(400);
  const toggle = page.getByRole('button', { name: /mostrar detalhes|show details/i });
  if ((await toggle.count()) > 0) {
    await toggle.first().click();
    await page.getByRole('tab', { name: /briefing/i }).waitFor({ timeout: 8000 });
  }
  await page.screenshot({ path: path.join(outDir, 'e2e-mobile-390.png') });

  await browser.close();
  console.log('composer workspace e2e PASS');
}

main()
  .catch(async (error) => {
    console.error('composer workspace e2e FAIL');
    console.error(error);
    process.exitCode = 1;
  })
  .finally(() => {
    stop();
    try {
      rmSync(isolatedDataDir, { recursive: true, force: true });
    } catch {
      /* ignore */
    }
  });
