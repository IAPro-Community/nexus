#!/usr/bin/env node
// Shared fixture helper for Nexus browser E2E scripts: builds nothing, starts
// an isolated `nexus web` server and seeds one fixture project, then hands the
// caller a ready Playwright page with collectors attached.
import { spawn, spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
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

// Fake Maestro Bridge used only by browser fixtures. Nexus talks to the
// external `orquestrador-maestro bridge --stdio` process; without a Bridge the
// read-only submissions projection answers 503 and Chromium logs it as a
// console error, which the surface isolation gate treats as an unexpected
// failure. The fake speaks the same JSON-RPC contract and never dispatches
// work: it only answers initialize and the read-only submissions poll.
const FAKE_MAESTRO_BRIDGE = String.raw`#!/usr/bin/env node
'use strict';
function send(payload) {
  process.stdout.write(JSON.stringify(payload) + '\n');
}
function reply(id, result) {
  send({ jsonrpc: '2.0', id: id, result: result });
}
function fail(id, message) {
  send({ jsonrpc: '2.0', id: id, error: { code: -32601, message: message } });
}
function handle(request) {
  const id = request.id;
  const method = request.method;
  const params = request.params || {};
  if (method === 'initialize') {
    reply(id, {
      protocolVersion: 1,
      capabilities: {
        projectContext: true,
        contextBrief: true,
        plansRoutePreview: true,
        runsSubmissionsPoll: true,
        runsSubmissionsCancel: true,
        runsSubmit: true,
        projectCompose: true,
        memory: { read: true },
      },
    });
    return;
  }
  if (method === 'runs.submissions.poll') {
    const limit = Number(params.limit || 20);
    const offset = Number(params.offset || 0);
    const previous = String(params.cursor || '');
    const cursor = 's1:' + '0'.repeat(64);
    reply(id, {
      schemaVersion: 1,
      authority: 'MAESTRO',
      owner: 'MAESTRO',
      delivery: 'SNAPSHOT_POLL',
      replayable: false,
      projectId: 'prj_e2e_maestro_fixture',
      cursor: cursor,
      changed: cursor !== previous,
      total: 0,
      offset: offset,
      limit: limit,
      nextOffset: null,
      submissions: [],
    });
    return;
  }
  fail(id, 'method not found: ' + method);
}
let buffer = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', function (chunk) {
  buffer += chunk;
  let index = buffer.indexOf('\n');
  while (index >= 0) {
    const line = buffer.slice(0, index).trim();
    buffer = buffer.slice(index + 1);
    if (line) {
      let request = null;
      try {
        request = JSON.parse(line);
      } catch (error) {
        request = null;
      }
      if (request) handle(request);
    }
    index = buffer.indexOf('\n');
  }
});
`;

export function fixtureFeatureEnvironment(source, { composerFlow = false, features = [] } = {}) {
  // Test processes never inherit accidental host opt-ins.
  const env = Object.fromEntries(
    Object.entries(source).filter(([key]) => !key.startsWith('NEXUS_FEATURE_') && key !== 'NEXUS_FEATURES_ENABLED'),
  );
  const enabled = new Set(features);
  if (composerFlow) {
    for (const key of ['COMPOSER', 'FLOW', 'MISSIONS']) enabled.add(key);
  }
  env.NEXUS_FEATURES_ENABLED = enabled.size > 0 ? 'true' : 'false';
  for (const key of enabled) env[`NEXUS_FEATURE_${key}`] = 'true';
  return env;
}

export async function startFixtureServer({ composerFlow = false, features = [] } = {}) {
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
  const fakeBridgeDir = path.join(isolatedDataDir, 'fake-maestro-bin');
  mkdirSync(fakeBridgeDir, { recursive: true });
  writeFileSync(path.join(fakeBridgeDir, 'orquestrador-maestro'), FAKE_MAESTRO_BRIDGE, {
    mode: 0o755,
  });
  const serverEnv = {
    ...fixtureFeatureEnvironment(process.env, { composerFlow, features }),
    NEXUS_DATA_DIR: isolatedDataDir,
    NEXUS_CONFIG_DIR: path.join(isolatedDataDir, 'config'),
    AI_MANAGER_DATA_DIR: isolatedDataDir,
    NEXUS_E2E_PROJECT_PATH: isolatedProjectPath,
    PATH: `${fakeBridgeDir}${path.delimiter}${process.env.PATH || ''}`,
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

export function playwrightExecutablePath() {
  const configuredPath = process.env.PLAYWRIGHT_EXECUTABLE_PATH;
  if (configuredPath) return existsSync(configuredPath) ? configuredPath : undefined;

  const candidates =
    process.platform === 'win32'
      ? [
          'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
          'C:/Program Files/Microsoft/Edge/Application/msedge.exe',
        ]
      : ['/usr/bin/google-chrome'];
  return candidates.find((candidate) => existsSync(candidate));
}

export async function launchFixtureBrowser(bootstrapUrl) {
  const browser = await chromium.launch({
    executablePath: playwrightExecutablePath(),
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
  if (projectsResponse.status() !== 200)
    throw new Error('project list unavailable after bootstrap');
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
