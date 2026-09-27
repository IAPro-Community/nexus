#!/usr/bin/env node
// Shared fixture helper for Nexus browser E2E scripts: builds nothing, starts
// an isolated `nexus web` server and seeds one fixture project, then hands the
// caller a ready Playwright page with collectors attached.
import { spawn, spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, rmSync } from 'node:fs';
import path from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';
import { ensureNexusBinary, nexusBinaryPath } from './e2e-platform.mjs';
import { terminateProcess } from './process-termination.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const webDir = path.resolve(__dirname, '..');
const repoRoot = path.resolve(webDir, '..');
const binPath = nexusBinaryPath(repoRoot);

export function terminateProcessOnExit(proc) {
  if (!proc || !proc.pid) return;
  try {
    if (process.platform === 'win32') {
      spawnSync('taskkill', ['/pid', String(proc.pid), '/T', '/F'], { stdio: 'ignore' });
    } else {
      process.kill(-proc.pid, 'SIGTERM');
    }
  } catch (_) {
    /* best-effort synchronous exit cleanup */
  }
}

export { terminateProcess };

export async function startFixtureServer() {
  ensureNexusBinary(repoRoot);
  const isolatedDataDir =
    process.env.NEXUS_E2E_DATA_DIR || path.join(tmpdir(), `nexus-surface-e2e-${process.pid}`);
  const isolatedProjectPath =
    process.env.NEXUS_E2E_PROJECT_PATH || path.join(isolatedDataDir, 'project');
  if (!process.env.NEXUS_E2E_DATA_DIR) rmSync(isolatedDataDir, { recursive: true, force: true });
  mkdirSync(isolatedProjectPath, { recursive: true });
  if (!existsSync(path.join(isolatedProjectPath, '.git'))) {
    const gitInit = spawnSync('git', ['init', '-q', isolatedProjectPath], { encoding: 'utf8' });
    if (gitInit.status !== 0) throw new Error(`fixture git init failed: ${gitInit.stderr || ''}`);
  }
  const serverEnv = {
    ...process.env,
    NEXUS_DATA_DIR: isolatedDataDir,
    AI_MANAGER_DATA_DIR: isolatedDataDir,
    NEXUS_E2E_PROJECT_PATH: isolatedProjectPath,
  };
  let serverOutput = '';
  const appendOutput = (chunk) => {
    serverOutput += chunk.toString();
    // Cap the retained tail: long soaks must not grow memory unbounded.
    if (serverOutput.length > 65536) serverOutput = serverOutput.slice(-65536);
  };
  const server = spawn(binPath, ['web', '--port', '0', '--listen', '127.0.0.1', '--no-open'], {
    cwd: repoRoot,
    detached: true,
    stdio: ['ignore', 'pipe', 'pipe'],
    env: serverEnv,
  });
  const killServer = () => terminateProcess(server);

  const bootstrapTimeoutMs = Number(process.env.NEXUS_E2E_TIMEOUT_MS || 12000);
  const bootstrapUrl = await new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      clearInterval(poll);
      // The caller never receives the handle on this path, so kill here to
      // avoid leaking the spawned server.
      killServer().finally(() => {
        reject(new Error(`Timed out waiting for server bootstrap URL. Output:\n${serverOutput}`));
      });
    }, bootstrapTimeoutMs);
    let resolved = false;
    const resolveFromListenState = () => {
      const result = spawnSync(binPath, ['web', 'url'], {
        cwd: repoRoot,
        encoding: 'utf8',
        timeout: 1000,
        env: serverEnv,
      });
      const candidate = String(result.stdout || '').trim();
      if (!/#nexus_bootstrap=[a-f0-9]+$/i.test(candidate)) return false;
      clearTimeout(timer);
      clearInterval(poll);
      resolved = true;
      resolve(candidate);
      return true;
    };
    const poll = setInterval(() => {
      if (!resolved) resolveFromListenState();
    }, 100);
    server.stdout.on('data', (chunk) => {
      appendOutput(chunk);
    });
    server.stderr.on('data', (chunk) => {
      appendOutput(chunk);
    });
    server.on('exit', (code) => {
      clearTimeout(timer);
      clearInterval(poll);
      reject(new Error(`Server exited prematurely with code ${code}. Output:\n${serverOutput}`));
    });
  });

  return {
    server,
    serverEnv,
    bootstrapUrl,
    isolatedProjectPath,
    isolatedDataDir,
    getOutput: () => serverOutput,
  };
}

export function isIgnorableFixtureRequestFailure(entry) {
  const value = String(entry || '');
  if (/favicon\.ico|data:/i.test(value)) return true;

  // Wails v3 probes this browser-development hook with HEAD. In a normal web
  // session there is no native Wails host, and Chromium reports the cancelled
  // probe as net::ERR_ABORTED. It is runtime capability discovery, not an app
  // resource/API failure.
  return /\bHEAD\s+https?:\/\/[^\s]+\/wails\/custom\.js\s+→\s+net::ERR_ABORTED\b/i.test(value);
}

export function meaningfulFixtureRequestFailures(entries) {
  return (entries || []).filter((entry) => !isIgnorableFixtureRequestFailure(entry));
}

export async function launchFixtureBrowser(bootstrapUrl) {
  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
  });
  const context = await browser.newContext();
  const page = await context.newPage();
  await context.addInitScript(() => {
    try {
      window.localStorage.clear();
      window.sessionStorage.clear();
    } catch (_) {}
  });
  const collectors = { consoleErrors: [], requestFailures: [], pageErrors: [], responses: [] };
  page.on('pageerror', (error) => collectors.pageErrors.push(String(error?.message || error)));
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
      collectors.responses.push(`${response.status()} ${response.url()}`);
  });
  return { browser, context, page, collectors };
}

export async function seedFixtureProject(context, bootstrapUrl, isolatedProjectPath) {
  const apiOrigin = new URL(bootstrapUrl).origin;
  const tryProjects = () => context.request.get(`${apiOrigin}/api/v1/projects`);
  let projectsResponse = await tryProjects();
  if (projectsResponse.status() === 401) {
    // No session cookie yet (e.g. API-only caller): exchange the one-time
    // bootstrap fragment token exactly like the web client's initSession().
    const fragment = new URL(bootstrapUrl).hash.replace(/^#/, '');
    const token = new URLSearchParams(fragment).get('nexus_bootstrap') || '';
    if (!token) throw new Error('bootstrap URL lacks the nexus_bootstrap fragment token');
    const exchange = await context.request.post(`${apiOrigin}/api/v1/auth/bootstrap`, {
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      data: { token },
    });
    if (!exchange.ok()) throw new Error(`bootstrap exchange failed: ${exchange.status()}`);
    projectsResponse = await tryProjects();
  }
  if (projectsResponse.status() !== 200) throw new Error('project list unavailable after bootstrap');
  let projects = await projectsResponse.json();
  if (!Array.isArray(projects) || projects.length === 0) {
    const sessionResponse = await context.request.get(`${apiOrigin}/api/v1/session`);
    if (sessionResponse.status() !== 200) throw new Error('session unavailable after bootstrap');
    const session = await sessionResponse.json();
    if (!session.csrf_token) throw new Error('fixture needs a CSRF token');
    const createResponse = await context.request.post(`${apiOrigin}/api/v1/projects`, {
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token },
      data: { name: 'Surface Isolation Fixture', path: isolatedProjectPath },
    });
    if (createResponse.status() !== 201) throw new Error('fixture project creation failed');
    projects = [await createResponse.json()];
  }
  const project = projects[0];
  if (!project?.id) throw new Error('fixture project must have an id');
  return { project, apiOrigin };
}
