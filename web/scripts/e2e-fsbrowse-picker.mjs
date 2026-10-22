#!/usr/bin/env node
// E2E: the OS folder picker (DirectoryBrowserModal) must stay usable when the
// `nexus web` process is started from a working directory outside the allowed
// filesystem roots (e.g. a shell opened in C:\Windows\System32). Validates both
// the HTTP contract (/api/v1/fs/browse without a path) and the real UI modal.
//
// Modes:
//   NEXUS_E2E_EXPECT=bug    → asserts the historical failure (403 + empty picker)
//   NEXUS_E2E_EXPECT=fixed  → asserts the fix (home default + allowed bookmarks)
//
// Env:
//   NEXUS_E2E_SERVER_BINARY  override the server binary (default: build checkout)
//   NEXUS_E2E_SERVER_CWD     server working directory (default: C:\Windows\System32)
//   NEXUS_E2E_ARTIFACTS_DIR  screenshot output directory
import assert from 'node:assert/strict';
import { spawn, spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';
import { ensureNexusBinary, nexusBinaryPath } from './e2e-platform.mjs';
import {
  meaningfulFixtureRequestFailures,
  playwrightExecutablePath,
  seedFixtureProject,
  terminateProcess,
  terminateProcessOnExit,
} from './e2e-fixture.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '..', '..');

const mode = process.env.NEXUS_E2E_EXPECT === 'bug' ? 'bug' : 'fixed';
const serverCwd = process.env.NEXUS_E2E_SERVER_CWD || 'C:\\Windows\\System32';
const artifactsDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR || path.resolve(repoRoot, '.tempmediaStorage', 'fsbrowse-picker');
mkdirSync(artifactsDir, { recursive: true });

const expectedHome = process.env.USERPROFILE || os.homedir();
const samePath = (a, b) => String(a).toLowerCase().replace(/[\\/]+$/, '') === String(b).toLowerCase().replace(/[\\/]+$/, '');

function log(step, detail = '') {
  console.log(`[e2e-fsbrowse] ${step}${detail ? ` — ${detail}` : ''}`);
}

async function main() {
  log('mode', mode);
  const binPath = process.env.NEXUS_E2E_SERVER_BINARY || ensureNexusBinary(repoRoot);
  assert.ok(existsSync(binPath), `server binary missing: ${binPath}`);
  assert.ok(existsSync(serverCwd), `server cwd missing: ${serverCwd}`);
  log('server binary', binPath);
  log('server cwd', serverCwd);

  const dataDir = path.join(os.tmpdir(), `nexus-fsbrowse-e2e-${process.pid}`);
  const projectPath = path.join(dataDir, 'project');
  rmSync(dataDir, { recursive: true, force: true });
  mkdirSync(projectPath, { recursive: true });
  const projectGit = spawnSync('git', ['init', '-q', projectPath], { encoding: 'utf8' });
  assert.equal(projectGit.status, 0, `git init failed: ${projectGit.stderr || ''}`);

  const serverEnv = {
    ...process.env,
    NEXUS_DATA_DIR: dataDir,
    AI_MANAGER_DATA_DIR: dataDir,
    NEXUS_E2E_PROJECT_PATH: projectPath,
    // Automated runs must never pop the tray HUD companion.
    NEXUS_HUD_SPAWN: '0',
  };

  let serverOutput = '';
  const appendOutput = (chunk) => {
    serverOutput += chunk.toString();
    if (serverOutput.length > 65536) serverOutput = serverOutput.slice(-65536);
  };

  const server = spawn(binPath, ['web', '--port', '0', '--listen', '127.0.0.1', '--no-open'], {
    cwd: serverCwd,
    detached: true,
    stdio: ['ignore', 'pipe', 'pipe'],
    env: serverEnv,
  });
  // terminateProcessOnExit is a synchronous killer: wire it to process exit,
  // never call it right after spawn.
  process.on('exit', () => terminateProcessOnExit(server));
  const killServer = () => terminateProcess(server);
  server.stdout.on('data', appendOutput);
  server.stderr.on('data', appendOutput);

  const bootstrapUrl = await new Promise((resolve, reject) => {
    const timeoutMs = Number(process.env.NEXUS_E2E_TIMEOUT_MS || 20000);
    const timer = setTimeout(() => {
      clearInterval(poll);
      killServer().finally(() => reject(new Error(`timed out waiting for bootstrap URL. Output:\n${serverOutput}`)));
    }, timeoutMs);
    let resolved = false;
    const poll = setInterval(() => {
      if (resolved) return;
      const result = spawnSync(binPath, ['web', 'url'], {
        cwd: serverCwd,
        encoding: 'utf8',
        timeout: 2000,
        env: serverEnv,
      });
      const candidate = String(result.stdout || '').trim();
      if (!/#nexus_bootstrap=[a-f0-9]+$/i.test(candidate)) return;
      clearTimeout(timer);
      clearInterval(poll);
      resolved = true;
      resolve(candidate);
    }, 150);
    server.on('exit', (code) => {
      if (resolved) return;
      clearTimeout(timer);
      clearInterval(poll);
      reject(new Error(`server exited prematurely with code ${code}. Output:\n${serverOutput}`));
    });
  });
  log('bootstrap', new URL(bootstrapUrl).origin);

  let browser;
  let context;
  let page;
  const collectors = { consoleErrors: [], requestFailures: [], responses: [] };
  try {
    browser = await chromium.launch({
      executablePath: playwrightExecutablePath(),
      headless: true,
      args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
    });
    context = await browser.newContext({ locale: 'pt-BR' });
    page = await context.newPage();
    page.on('console', (message) => {
      if (message.type() === 'error') collectors.consoleErrors.push(message.text());
    });
    page.on('requestfailed', (request) => {
      collectors.requestFailures.push(
        `${request.method()} ${request.url()} → ${request.failure()?.errorText || 'unknown'}`,
      );
    });
    page.on('response', (response) => {
      if (response.url().includes('/api/'))
        collectors.responses.push({ status: response.status(), url: response.url() });
    });

    const { apiOrigin } = await seedFixtureProject(context, bootstrapUrl, projectPath);

    // ── Level 1: HTTP contract — browse without a path ─────────────────────
    const apiResponse = await context.request.get(`${apiOrigin}/api/v1/fs/browse`);
    const apiBody = await apiResponse.json().catch(() => null);
    log('api browse status', String(apiResponse.status()));

    if (mode === 'bug') {
      assert.equal(apiResponse.status(), 403, `expected the bug (403) without a path, got ${apiResponse.status()}`);
      assert.match(String(apiBody?.error || ''), /outside allowed directories/i);
    } else {
      assert.equal(apiResponse.status(), 200, `browse without a path must return 200, got ${apiResponse.status()}: ${JSON.stringify(apiBody)}`);
      assert.ok(samePath(apiBody.current_path, expectedHome), `current_path ${apiBody.current_path} should be the home directory ${expectedHome}`);
      const bookmarks = apiBody.bookmarks || [];
      assert.ok(bookmarks.length > 0, 'bookmarks must not be empty');
      for (const bookmark of bookmarks) {
        assert.ok(!/^\/$/.test(String(bookmark.path)), `bookmark ${bookmark.label} must not point at the OS root`);
      }
      log('api bookmarks', bookmarks.map((b) => b.label).join(', '));
    }

    // ── Level 2: real UI — open the picker modal on /projects ──────────────
    await page.goto(`${apiOrigin}/projects`, { waitUntil: 'domcontentloaded', timeout: 20000 });
    const browseButton = page.getByRole('button', { name: /Procurar Pastas no SO/ }).first();
    await browseButton.waitFor({ state: 'visible', timeout: 20000 });
    await browseButton.click();

    const dialog = page.getByRole('dialog').first();
    await dialog.waitFor({ state: 'visible', timeout: 10000 });
    log('picker dialog open');

    const pathInput = dialog.locator('input[placeholder^="Digite um caminho"]');
    const errorAlert = dialog.locator('[role="alert"]');
    const bookmark = (label) => dialog.getByRole('button', { name: label, exact: true });

    if (mode === 'bug') {
      await errorAlert.waitFor({ state: 'visible', timeout: 10000 });
      const errorText = await errorAlert.innerText();
      assert.match(errorText, /outside allowed directories|Não foi possível ler esta pasta/i, `unexpected picker error: ${errorText}`);
      assert.equal(await bookmark('Home (~)').count(), 0, 'buggy build must expose no usable bookmarks');
      await page.screenshot({ path: path.join(artifactsDir, 'red-picker-error.png'), fullPage: true });
      log('RED reproduced: picker fails with', JSON.stringify(errorText.slice(0, 120)));
    } else {
      await page.waitForFunction(() => {
        const input = document.querySelector('input[placeholder^="Digite um caminho"]');
        return Boolean(input && input.value);
      }, { timeout: 10000 });
      assert.equal(await dialog.locator('[role="alert"]').filter({ hasText: /outside allowed/i }).count(), 0, 'fixed picker must not show the allowed-roots error');
      const shownPath = await pathInput.inputValue();
      assert.ok(samePath(shownPath, expectedHome), `picker should open at ${expectedHome}, got ${shownPath}`);

      await bookmark('Home (~)').waitFor({ state: 'visible', timeout: 5000 });
      await bookmark('Documents').waitFor({ state: 'visible', timeout: 5000 });
      assert.equal(await bookmark('Root (/)').count(), 0, 'fixed build must not expose the dead OS-root bookmark');
      await dialog.getByRole('button', { name: /Abrir pasta src/i }).first().waitFor({ state: 'visible', timeout: 5000 });
      await page.screenshot({ path: path.join(artifactsDir, 'green-picker-home.png'), fullPage: true });
      log('picker opened at', shownPath);

      // Bookmarks must navigate: click "Documents" and confirm the listing follows.
      await bookmark('Documents').click();
      await page.waitForFunction((expected) => {
        const input = document.querySelector('input[placeholder^="Digite um caminho"]');
        return Boolean(input && input.value && input.value.toLowerCase().endsWith(expected));
      }, 'documents', { timeout: 10000 });
      assert.equal(await dialog.locator('[role="alert"]').count(), 0, 'navigating via bookmark must not surface errors');
      await page.screenshot({ path: path.join(artifactsDir, 'green-picker-documents.png'), fullPage: true });
      log('bookmark navigation ok', await pathInput.inputValue());
    }

    const browseStatuses = collectors.responses
      .filter((entry) => entry.url.includes('/api/v1/fs/browse'))
      .map((entry) => entry.status);
    log('browse responses', browseStatuses.join(', ') || 'none');
    if (mode === 'fixed') {
      assert.ok(browseStatuses.length > 0 && browseStatuses.every((status) => status === 200), `every browse request must succeed, got ${browseStatuses.join(', ')}`);
    }

    const requestFailures = meaningfulFixtureRequestFailures(collectors.requestFailures);
    assert.equal(
      requestFailures.length,
      0,
      `unexpected request failures:\n${requestFailures.join('\n')}`,
    );

    console.log(`[e2e-fsbrowse] PASS (${mode}) — artifacts in ${artifactsDir}`);
  } catch (error) {
    try {
      if (page) await page.screenshot({ path: path.join(artifactsDir, `failure-${mode}.png`), fullPage: true }).catch(() => {});
      writeFileSync(
        path.join(artifactsDir, `failure-${mode}.json`),
        JSON.stringify(
          {
            mode,
            serverCwd,
            error: String(error?.message || error),
            serverTail: serverOutput.split(/\r?\n/).slice(-40),
            consoleErrors: collectors.consoleErrors.slice(-20),
            requestFailures: collectors.requestFailures.slice(-20),
            responses: collectors.responses.slice(-30),
          },
          null,
          2,
        ),
      );
    } catch (_) {
      /* forensics are best-effort */
    }
    throw error;
  } finally {
    if (browser) await browser.close().catch(() => {});
    await killServer().catch(() => {});
    rmSync(dataDir, { recursive: true, force: true });
  }
}

main().catch((error) => {
  console.error(`[e2e-fsbrowse] FAIL — ${error?.stack || error}`);
  process.exitCode = 1;
});
