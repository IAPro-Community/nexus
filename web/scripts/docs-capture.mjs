#!/usr/bin/env node
import { spawn, spawnSync } from 'node:child_process';
import {
  cpSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  renameSync,
  rmSync,
  writeFileSync,
} from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { chromium } from 'playwright';
import { ensureNexusBinary, nexusBinaryPath } from './e2e-platform.mjs';
import { terminateProcess, terminateProcessOnExit } from './e2e-fixture.mjs';
import { dumpWorkspaceStateInPage } from './workspace-state.mjs';

const webDir = path.resolve(import.meta.dirname, '..');
const root = path.resolve(webDir, '..');
const outputDir = path.join(root, 'docs/assets/screenshots');
const demoDir = path.join(root, 'docs/assets/demos');
mkdirSync(outputDir, { recursive: true });
mkdirSync(demoDir, { recursive: true });
const binPath = nexusBinaryPath(root);
ensureNexusBinary(root);

const fixtureRoot = mkdtempSync(path.join(os.tmpdir(), 'nexus-docs-fixture-'));
const dataDir = path.join(fixtureRoot, 'data');
const workspaceDir = path.join(fixtureRoot, 'workspace');
mkdirSync(dataDir, { recursive: true });
mkdirSync(workspaceDir, { recursive: true });
writeFileSync(
  path.join(workspaceDir, 'README.md'),
  '# Nexus Demo Workspace\n\nSynthetic documentation fixture.\n',
);

const serverEnv = {
  ...process.env,
  NEXUS_DATA_DIR: dataDir,
  AI_CLI_DATA_DIR: dataDir,
  NEXUS_DOCS_CAPTURE: '1',
};
if (process.platform !== 'win32') {
  serverEnv.SHELL = '/bin/sh';
  serverEnv.PS1 = 'nexus-demo$ ';
  serverEnv.PROMPT_COMMAND = '';
} else {
  serverEnv.PROMPT = 'nexus-demo$G';
  serverEnv.ComSpec = path.join(process.env.SystemRoot || 'C:\\Windows', 'System32', 'cmd.exe');
}

const server = spawn(
  binPath,
  ['web', '--port', '0', '--listen', '127.0.0.1', '--no-open'],
  {
    cwd: root,
    detached: true,
    stdio: ['ignore', 'pipe', 'pipe'],
    env: serverEnv,
  },
);
let output = '';
const backendStages = () => output.split(/\r?\n/)
  .map((line) =>
    line.match(/nexus_docs_capture_(?:terminal|project_shell|launcher_shell)_stage=([a-z-]+)/)?.[1],
  )
  .filter(Boolean)
  .slice(-32);
const backendDiagnostics = () => output.split(/\r?\n/)
  .filter((line) => line.includes('nexus_docs_capture_diagnostic='))
  .map((line) => line.slice(line.indexOf('nexus_docs_capture_diagnostic=') + 30).slice(0, 500))
  .slice(-8);

// Failure forensics for the detached SessionHost. The host writes its own log
// under the fixture data dir and the registry row records who it believed was
// alive; both are needed to explain a lease timeout without a re-run. The
// fixture directory is deleted by stop(), so a copy is preserved for the CI
// artifact under docs/assets/demos/fixture-forensics/.
let shellRuntimeId = '';
const hostLogTailFor = (id) => {
  if (!id) return 'no-runtime-id';
  const logPath = path.join(dataDir, 'logs', `${id}.log`);
  if (!existsSync(logPath)) return `missing:${logPath}`;
  try {
    return readFileSync(logPath, 'utf8').split(/\r?\n/).slice(-25).join(' | ').slice(-4000);
  } catch (error) {
    return 'unreadable:' + String(error?.message || error);
  }
};
const readRegistryRows = () => {
  const registryPath = path.join(dataDir, 'runtimes.json');
  if (!existsSync(registryPath)) return null;
  try {
    return JSON.parse(readFileSync(registryPath, 'utf8'));
  } catch {
    return null;
  }
};
const hostRegistryRowFor = (id) => {
  if (!id) return 'no-runtime-id';
  const rows = readRegistryRows();
  if (!rows) return 'missing';
  return JSON.stringify(rows?.[id] ?? null).slice(0, 2000);
};
const copyFixtureForensics = () => {
  const target = path.join(demoDir, 'fixture-forensics');
  try {
    cpSync(path.join(dataDir, 'logs'), path.join(target, 'logs'), { recursive: true });
  } catch { /* logs dir may not exist */ }
  try {
    cpSync(path.join(dataDir, 'runtimes.json'), path.join(target, 'runtimes.json'));
  } catch { /* registry may not exist */ }
};
const hostProcessState = () => {
  if (process.platform !== 'win32') return 'not-windows';
  const id = shellRuntimeId || firstShellRuntimeId();
  if (!id) return 'no-runtime-id';
  const row = readRegistryRows()?.[id];
  const pid = row?.host_pid || row?.pid;
  if (!pid) return 'no-host-pid';
  const result = spawnSync('tasklist', ['/FI', `PID eq ${pid}`, '/NH'], { encoding: 'utf8' });
  return String(result.stdout || result.stderr || '').trim().slice(0, 400) || 'tasklist-no-output';
};
const firstShellRuntimeId = () => {
  const registryPath = path.join(dataDir, 'runtimes.json');
  if (!existsSync(registryPath)) return '';
  try {
    const rows = JSON.parse(readFileSync(registryPath, 'utf8'));
    const ids = Object.keys(rows || {}).filter((id) => id.startsWith('shell-'));
    return ids[ids.length - 1] || '';
  } catch {
    return '';
  }
};
const dumpWorkspaceState = (pageRef) => pageRef.evaluate(dumpWorkspaceStateInPage);
const bootstrap = new Promise((resolve, reject) => {
  const timer = setTimeout(() => reject(new Error(`bootstrap timeout\n${output}`)), 15000);
  const onData = (chunk) => {
    output += chunk.toString();
    const match = output.match(/URL:\s+(http:\/\/127\.0\.0\.1:\d+)/i);
    if (match) { clearTimeout(timer); resolve(match[1]); }
  };
  server.stdout.on('data', onData);
  server.stderr.on('data', onData);
  server.on('exit', (code) => { clearTimeout(timer); reject(new Error(`server exited ${code}\n${output}`)); });
});
const stop = async () => {
  await terminateProcess(server);
  try { rmSync(fixtureRoot, { recursive: true, force: true }); } catch {}
};
process.on('exit', () => terminateProcessOnExit(server));

const sourceSha = spawnSync('git', ['rev-parse', 'HEAD'], { cwd: root, encoding: 'utf8' }).stdout.trim();
const captures = [];
let captureFailed = true;
try {
  const url = await bootstrap;
  const browser = await chromium.launch({
    executablePath: existsSync('/usr/bin/google-chrome') ? '/usr/bin/google-chrome' : undefined,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage', '--no-proxy-server'],
  });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    recordVideo: { dir: demoDir, size: { width: 1280, height: 800 } },
  });
  const syntheticWorkspacePath = process.platform === 'win32' ? 'C:\\NexusDemo\\workspace' : '/NexusDemo/workspace';
  await context.addInitScript(({ paths }) => {
    const redact = (value) => paths.reduce((result, entry) => {
      if (!entry.value) return result;
      const search = entry.value.toLowerCase();
      let sanitized = result;
      let index = sanitized.toLowerCase().indexOf(search);
      while (index >= 0) {
        sanitized = sanitized.slice(0, index) + entry.replacement + sanitized.slice(index + entry.value.length);
        index = sanitized.toLowerCase().indexOf(search, index + entry.replacement.length);
      }
      return sanitized;
    }, value);
    const sanitizeNode = (node) => {
      if (node.nodeType === Node.TEXT_NODE) {
        const value = redact(node.nodeValue || '');
        if (value !== node.nodeValue) node.nodeValue = value;
        return;
      }
      if (node.nodeType === Node.ELEMENT_NODE) {
        for (const attribute of node.attributes) {
          const value = redact(attribute.value);
          if (value !== attribute.value) node.setAttribute(attribute.name, value);
        }
        if ('value' in node && typeof node.value === 'string') {
          const value = redact(node.value);
          if (value !== node.value) node.value = value;
        }
      }
      for (const child of node.childNodes || []) sanitizeNode(child);
    };
    for (const prototype of [HTMLInputElement.prototype, HTMLTextAreaElement.prototype, HTMLOptionElement.prototype]) {
      const descriptor = Object.getOwnPropertyDescriptor(prototype, 'value');
      if (descriptor?.set) {
        Object.defineProperty(prototype, 'value', {
          ...descriptor,
          set(value) {
            descriptor.set.call(this, redact(String(value ?? '')));
          },
        });
      }
    }
    const observer = new MutationObserver((records) => {
      for (const record of records) {
        if (record.type === 'characterData') sanitizeNode(record.target);
        if (record.type === 'attributes') sanitizeNode(record.target);
        for (const node of record.addedNodes || []) sanitizeNode(node);
      }
    });
    const install = () => {
      const root = document.documentElement;
      if (!root) {
        document.addEventListener('DOMContentLoaded', install, { once: true });
        return;
      }
      observer.observe(root, { childList: true, subtree: true, characterData: true, attributes: true });
      sanitizeNode(root);
    };
    install();
  }, {
    paths: [
      { value: workspaceDir, replacement: syntheticWorkspacePath },
      ...(process.platform === 'win32' && process.env.USERPROFILE
        ? [{ value: process.env.USERPROFILE, replacement: 'C:\\NexusUser' }]
        : []),
    ],
  });
  const page = await context.newPage();
  const websocketHandshakeDiagnostics = [];
  const websocketRequestIds = new Set();
  const cdp = await context.newCDPSession(page);
  await cdp.send('Network.enable');
  cdp.on('Network.webSocketWillSendHandshakeRequest', (event) => {
    websocketRequestIds.add(event.requestId);
    websocketHandshakeDiagnostics.push('request');
  });
  cdp.on('Network.webSocketHandshakeResponseReceived', (event) => {
    websocketHandshakeDiagnostics.push('response:' + event.response.status);
  });
  cdp.on('Network.loadingFailed', (event) => {
    if (websocketRequestIds.has(event.requestId)) {
      websocketHandshakeDiagnostics.push('failed:' + event.errorText);
    }
  });
  cdp.on('Network.webSocketFrameError', (event) => {
    websocketHandshakeDiagnostics.push('frame-error:' + event.errorMessage);
  });
  const video = page.video();
  const sensitiveFixturePaths = [...new Set([
    workspaceDir,
    workspaceDir.replaceAll('\\', '/'),
    process.platform === 'win32' ? process.env.USERPROFILE : '',
  ].filter(Boolean))];
  await page.addInitScript((sensitivePaths) => {
    const replacement = 'C:\\NexusDemo\\workspace';
    const redactValue = (value) => {
      let result = value;
      for (const sensitivePath of sensitivePaths) {
        const lowerValue = result.toLowerCase();
        const lowerSearch = sensitivePath.toLowerCase();
        let cursor = 0;
        let index = lowerValue.indexOf(lowerSearch, cursor);
        let next = '';
        while (index !== -1) {
          next += result.slice(cursor, index) + replacement;
          cursor = index + sensitivePath.length;
          index = lowerValue.indexOf(lowerSearch, cursor);
        }
        if (cursor !== 0) result = next + result.slice(cursor);
      }
      return result;
    };
    const redactNode = (node) => {
      if (node.nodeType === Node.TEXT_NODE) {
        const value = redactValue(node.nodeValue || '');
        if (value !== node.nodeValue) node.nodeValue = value;
        return;
      }
      if (node.nodeType !== Node.ELEMENT_NODE) return;
      for (const attribute of [...node.attributes]) {
        const value = redactValue(attribute.value);
        if (value !== attribute.value) node.setAttribute(attribute.name, value);
      }
      for (const child of node.childNodes) redactNode(child);
    };
    const installObserver = () => {
      const root = document.documentElement;
      if (!root) {
        const documentObserver = new MutationObserver(() => {
          if (!document.documentElement) return;
          documentObserver.disconnect();
          installObserver();
        });
        documentObserver.observe(document, { childList: true });
        return;
      }
      redactNode(root);
      new MutationObserver((records) => {
        for (const record of records) {
          if (record.type === 'characterData' || record.type === 'attributes') {
            redactNode(record.target);
          } else {
            for (const node of record.addedNodes) redactNode(node);
          }
        }
      }).observe(root, { attributes: true, characterData: true, childList: true, subtree: true });
    };
    installObserver();
  }, sensitiveFixturePaths);
  const pageErrors = [];
  page.on('pageerror', (error) => pageErrors.push(String(error?.message || error)));
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 60000 });
  const listenStatePath = path.join(dataDir, 'web-auth', 'listen.json');
  const stateDeadline = Date.now() + 5000;
  while (!existsSync(listenStatePath) && Date.now() < stateDeadline) {
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  if (!existsSync(listenStatePath)) throw new Error('isolated web auth state was not created');
  const listenState = JSON.parse(readFileSync(listenStatePath, 'utf8'));
  if (!listenState.bootstrap_token) throw new Error('isolated web auth state has no bootstrap token');
  const bootstrapResponse = await page.evaluate(async (token) => {
    const response = await fetch('/api/v1/auth/bootstrap', {
      method: 'POST',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: JSON.stringify({ token }),
    });
    return { ok: response.ok, status: response.status };
  }, listenState.bootstrap_token);
  if (!bootstrapResponse.ok) throw new Error(`isolated auth bootstrap failed: HTTP ${bootstrapResponse.status}`);
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForFunction(
    () => document.body.innerText.includes('Projetos') || document.body.innerText.includes('Projects') || document.querySelector('.nx-os-shell'),
    { timeout: 60000 },
  );
  const session = await page.evaluate(async () => {
    const response = await fetch('/api/v1/session', { headers: { Accept: 'application/json' } });
    if (!response.ok) throw new Error(`session bootstrap failed: HTTP ${response.status}`);
    return response.json();
  });
  if (!session.csrf_token) throw new Error('session bootstrap did not provide CSRF token');

  const createProject = await page.evaluate(async ({ workspace, csrf }) => {
    const response = await fetch('/api/v1/projects', {
      method: 'POST',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json', 'X-CSRF-Token': csrf },
      body: JSON.stringify({ name: 'Nexus Demo Workspace', path: workspace }),
    });
    if (!response.ok) throw new Error(`fixture project failed: HTTP ${response.status}`);
    return response.json();
  }, { workspace: workspaceDir, csrf: session.csrf_token });

  // Refresh app state so the newly created synthetic project is present in the workspace route resolver.
  const refreshResponse = await page.reload({ waitUntil: 'domcontentloaded', timeout: 60000 });
  const refreshedProjects = await page.evaluate(async () => {
    const response = await fetch('/api/v1/projects', { headers: { Accept: 'application/json' } });
    const payload = await response.json().catch(() => null);
    return {
      status: response.status,
      projects: Array.isArray(payload) ? payload : payload?.projects || [],
    };
  });
  if (refreshedProjects.status !== 200 || !refreshedProjects.projects.some((project) =>
    project.name === 'Nexus Demo Workspace' && project.id === createProject.id)) {
    throw new Error(
      'synthetic project was not persisted after refresh; status=' +
        refreshedProjects.status + '; count=' + refreshedProjects.projects.length,
    );
  }
  await page.goto(
    new URL('/p/' + encodeURIComponent(createProject.id) + '/overview', url).toString(),
    { waitUntil: 'domcontentloaded', timeout: 60000 },
  );
  await page.waitForFunction(
    () => document.querySelector('.nx-os-shell')?.isConnected === true,
    { timeout: 60000 },
  );
  let resolveTerminalControl;
  const terminalControlReady = new Promise((resolve) => { resolveTerminalControl = resolve; });
  const terminalSocketDiagnostics = [];
  const terminalHttpDiagnostics = [];
  const shellLaunchDiagnostics = [];
  const reconnectMarker = '__NEXUS_DOCS_TERMINAL_RECONNECTED__';
  let trackReconnectSocket = false;
  let reconnectControlLeaseReceived = false;
  let reconnectMarkerOutputReceived = false;
  page.on('response', (response) => {
    const pathname = new URL(response.url()).pathname.toLowerCase();
    if (pathname.includes('terminal')) terminalHttpDiagnostics.push('http:' + response.status());
    if (pathname.endsWith('/shell')) shellLaunchDiagnostics.push('response:' + response.status());
  });
  page.on('request', (request) => {
    const pathname = new URL(request.url()).pathname.toLowerCase();
    if (pathname.endsWith('/shell')) shellLaunchDiagnostics.push('request:' + request.method());
  });
  page.on('requestfailed', (request) => {
    const pathname = new URL(request.url()).pathname.toLowerCase();
    if (pathname.endsWith('/shell')) shellLaunchDiagnostics.push(
      'request-failed:' + (request.failure()?.errorText || 'unknown'),
    );
  });
  page.on('requestfailed', (request) => {
    const pathname = new URL(request.url()).pathname.toLowerCase();
    if (pathname.includes('terminal')) {
      terminalHttpDiagnostics.push('request-failed:' + (request.failure()?.errorText || 'unknown'));
    }
  });
  page.on('websocket', (socket) => {
    const socketPath = new URL(socket.url()).pathname.toLowerCase();
    const isReconnectSocket = trackReconnectSocket && socketPath.includes('terminal');
    terminalSocketDiagnostics.push(socketPath.includes('terminal') ? 'created:terminal' : 'created:other');
    socket.on('close', (close) => terminalSocketDiagnostics.push(
      'closed:' + String(close?.code || '') + ':' + String(close?.reason || '').slice(0, 80),
    ));
    socket.on('socketerror', (error) => {
      const detail = String(error?.message || error || '').split('?')[0].slice(0, 160);
      terminalSocketDiagnostics.push('socket-error:' + detail);
    });
    socket.on('framesent', ({ payload }) => {
      try {
        const frame = JSON.parse(String(payload));
        terminalSocketDiagnostics.push('sent:' + String(frame.type || 'unknown'));
      } catch {
        terminalSocketDiagnostics.push('sent:non-json');
      }
    });
    socket.on('framereceived', ({ payload }) => {
      try {
        const frame = JSON.parse(String(payload));
        terminalSocketDiagnostics.push('received:' + String(frame.type || 'unknown') + ':' + String(frame.role || ''));
        if (frame.type === 'lease' && frame.role === 'CONTROL') {
          resolveTerminalControl();
          if (isReconnectSocket) reconnectControlLeaseReceived = true;
        }
        if (isReconnectSocket && frame.type === 'output' && String(frame.data || '').includes(reconnectMarker)) {
          reconnectMarkerOutputReceived = true;
        }
      } catch {
        terminalSocketDiagnostics.push('received:non-json');
      }
    });
  });
  const terminalLaunch = page
    .locator('[data-testid="overview-terminal-btn"]:visible')
    .first();
  try {
    await terminalLaunch.waitFor({ state: 'visible', timeout: 30000 });
  } catch (error) {
    const overviewState = await page.evaluate(() => ({
      path: window.location.pathname,
      title: document.title,
      bodyText: (document.body.innerText || '').slice(0, 1400),
      shellCount: document.querySelectorAll('.nx-os-shell').length,
      overviewButtonCount: document.querySelectorAll('[data-testid="overview-terminal-btn"]').length,
      workspacePanels: [...document.querySelectorAll('[role="tabpanel"]')].map((node) =>
        (node.innerText || '').trim().slice(0, 120),
      ).slice(0, 12),
      activeTabs: [...document.querySelectorAll('[role="tab"][aria-selected="true"]')].map((node) =>
        (node.innerText || '').trim().slice(0, 100),
      ).slice(0, 12),
    }));
    throw new Error(
      'project overview terminal button did not become visible; diagnostics=' +
        JSON.stringify(overviewState) + '; cause=' + String(error),
    );
  }
  const shellResponsePromise = page.waitForResponse(
    (response) => new URL(response.url()).pathname.endsWith('/shell'),
    { timeout: 30000 },
  );
  await terminalLaunch.click();
  let shellResponse;
  try {
    shellResponse = await shellResponsePromise;
  } catch (error) {
    const launchState = await page.evaluate(() => {
      const button = document.querySelector('[data-testid="overview-terminal-btn"]');
      return {
        path: window.location.pathname,
        buttonPresent: Boolean(button),
        buttonDisabled: button instanceof HTMLButtonElement ? button.disabled : null,
        projectShellSurfaces: document.querySelectorAll('.nx-project-shell-surface').length,
        dialogs: document.querySelectorAll('[role="dialog"]').length,
      };
    });
    throw new Error(
      'project shell launch response timed out; diagnostics=' +
        JSON.stringify({
          shellLaunch: shellLaunchDiagnostics,
          launchState,
          proxyConfigured: Boolean(
            process.env.HTTP_PROXY || process.env.HTTPS_PROXY || process.env.ALL_PROXY ||
              process.env.http_proxy || process.env.https_proxy || process.env.all_proxy,
          ),
          backendStages: backendStages(),
          backendDiagnostics: backendDiagnostics(),
          hostLog: hostLogTailFor(shellRuntimeId || firstShellRuntimeId()),
          registryRow: hostRegistryRowFor(shellRuntimeId || firstShellRuntimeId()),
          hostProcess: hostProcessState(),
          websocketHandshake: websocketHandshakeDiagnostics,
          pageErrorCount: pageErrors.length,
        }) +
        '; cause=' + error.message,
    );
  }
  const shellPayload = await shellResponse.json().catch(() => null);
  const shellRuntime = shellPayload?.runtime || {};
  shellRuntimeId = shellRuntime.runtime_id || '';
  if (!shellResponse.ok() || !shellRuntime.runtime_id) {
    throw new Error('project shell launch API failed; http=' + shellResponse.status());
  }
  await page.waitForFunction(
    () => document.querySelector('.nx-os-shell')?.isConnected === true,
    { timeout: 60000 },
  );
  await page.waitForFunction(
    () => document.querySelector('.nx-project-shell-surface[data-runtime-id]'),
    { timeout: 30000 },
  ).catch(async (error) => {
    const appState = await page.evaluate(() => ({
      path: window.location.pathname,
      appShell: Boolean(document.querySelector('.nx-os-shell')),
      surfaceTabs: [...document.querySelectorAll('[role="tab"][data-kind]')].map((tab) => ({
        kind: tab.getAttribute('data-kind') || '',
        active: tab.getAttribute('aria-selected') === 'true',
      })),
      projectShellSurfaces: document.querySelectorAll('.nx-project-shell-surface').length,
    }));
    throw new Error(
      'project shell launch response was not rendered; http=' + shellResponse.status() +
        '; state=' + (shellRuntime.state || '') +
        '; runtimeId=' + shellRuntime.runtime_id +
        '; app=' + JSON.stringify(appState) +
        '; cause=' + error.message,
    );
  });
  const runtimeId = await page.locator('.nx-project-shell-surface').first().getAttribute('data-runtime-id');
  if (!runtimeId) throw new Error('project shell surface did not expose its runtime id');
  await page.waitForFunction(
    (expectedRuntimeId) => [...document.querySelectorAll('.nx-project-shell-surface')].some(
      (surface) => surface.getAttribute('data-runtime-id') === expectedRuntimeId,
    ),
    runtimeId,
    { timeout: 30000 },
  ).catch(async (error) => {
    const shellState = await page.evaluate(() => ({
      path: window.location.pathname,
      appShell: Boolean(document.querySelector('.nx-os-shell')),
      surfaceTabs: [...document.querySelectorAll('[role="tab"][data-kind]')].map((tab) => ({
        kind: tab.getAttribute('data-kind') || '',
        active: tab.getAttribute('aria-selected') === 'true',
      })),
      ptyTabs: [...document.querySelectorAll('[id^="nx-pty-tab-"]')].map((tab) => ({
        active: tab.getAttribute('aria-selected') === 'true',
        label: tab.getAttribute('aria-label') || '',
      })),
      workspacePanels: document.querySelectorAll('.nx-workspace-panel').length,
      projectShellRuntimeIds: [...document.querySelectorAll('.nx-project-shell-surface')].map(
        (surface) => surface.getAttribute('data-runtime-id') || '',
      ),
    }));
    throw new Error(
      'project shell surface did not bind to created runtime; expected=' +
        runtimeId + '; state=' + JSON.stringify(shellState) +
        '; pageErrors=' + pageErrors.length + '; cause=' + error.message,
    );
  });
  const terminalUiState = await page.evaluate(() => {
    const surface = document.querySelector('.nx-project-shell-surface');
    return {
      path: window.location.pathname,
      projectShellSurface: Boolean(surface),
      runtimeId: surface?.getAttribute('data-runtime-id') || '',
      terminal: Boolean(surface?.querySelector('.xterm')),
      rows: Boolean(surface?.querySelector('.xterm-rows')),
      rowCount: surface?.querySelectorAll('.xterm-rows > div').length || 0,
    };
  });
  await page.waitForSelector('.nx-project-shell-surface .xterm-rows', { timeout: 15000 }).catch(
    async (error) => {
      const browserState = await page.evaluate(() => ({
        path: window.location.pathname,
        appShell: Boolean(document.querySelector('.nx-os-shell')),
        projectShellSurfaces: [...document.querySelectorAll('.nx-project-shell-surface')].map(
          (surface) => ({
            runtimeId: surface.getAttribute('data-runtime-id') || '',
            terminal: Boolean(surface.querySelector('.xterm')),
            rows: Boolean(surface.querySelector('.xterm-rows')),
          }),
        ),
        activeTabs: [...document.querySelectorAll('[role="tab"][aria-selected="true"]')].map(
          (tab) => tab.getAttribute('aria-label') || tab.textContent?.trim() || '',
        ),
        visibleAlerts: [...document.querySelectorAll('[role="alert"]')].map(
          (alert) => alert.textContent?.trim() || '',
        ),
      }));
      throw new Error(
        'project shell terminal rows did not render; terminalUi=' + JSON.stringify(terminalUiState) +
          '; browser=' + JSON.stringify(browserState) +
          '; http=' + JSON.stringify(terminalHttpDiagnostics) +
          '; backendStages=' + JSON.stringify(backendStages()) +
          '; backendDiagnostics=' + JSON.stringify(backendDiagnostics()) +
          '; workspace=' + JSON.stringify(await dumpWorkspaceState(page)) +
          '; websocket=' + (terminalSocketDiagnostics.join(',') || 'none') +
          '; websocketHandshake=' + JSON.stringify(websocketHandshakeDiagnostics) +
          '; pageErrors=' + JSON.stringify(pageErrors) +
          '; cause=' + error.message,
      );
    },
  );
  let leaseTimer;
  await Promise.race([
    terminalControlReady,
    new Promise((_, reject) => {
      leaseTimer = setTimeout(() => reject(new Error(
        'project shell writer lease was not ready; websocket=' +
          (terminalSocketDiagnostics.join(',') || 'none') +
          '; terminalUi=' + JSON.stringify(terminalUiState) +
          '; http=' + JSON.stringify(terminalHttpDiagnostics) +
          '; shellLaunch=' + JSON.stringify({
            http: shellResponse.status(),
            state: shellRuntime.state || '',
            hostPidPresent: Boolean(shellRuntime.host_pid || shellRuntime.pid),
          }) +
          '; backendStages=' + JSON.stringify(backendStages()) +
          '; backendDiagnostics=' + JSON.stringify(backendDiagnostics()) +
          '; hostLog=' + hostLogTailFor(runtimeId) +
          '; registryRow=' + hostRegistryRowFor(runtimeId) +
          '; hostProcess=' + hostProcessState() +
          '; websocketHandshake=' + JSON.stringify(websocketHandshakeDiagnostics) +
          '; pageErrors=' + pageErrors.length,
      )), 35000);
    }),
  ]).finally(() => clearTimeout(leaseTimer));

  const marker = '__NEXUS_DOCS_TERMINAL_OK__';
  await page.locator('.nx-project-shell-surface .xterm').click();
  await page.keyboard.type(`echo ${marker}`);
  await page.keyboard.press('Enter');
  await page.waitForFunction((expected) => {
    const terminal = document.querySelector('.nx-project-shell-surface .xterm-rows');
    return terminal?.textContent?.includes(expected) === true;
  }, marker, { timeout: 15000 });

  const redactFixturePaths = async () => {
    const paths = [...new Set([
      workspaceDir,
      workspaceDir.replaceAll('\\', '/'),
      process.platform === 'win32' ? process.env.USERPROFILE : '',
    ].filter(Boolean))];
    return page.evaluate(({ sensitivePaths, replacement }) => {
      const replaceInsensitive = (value, search, replacement) => {
        const lowerValue = value.toLowerCase();
        const lowerSearch = search.toLowerCase();
        let cursor = 0;
        let index = lowerValue.indexOf(lowerSearch, cursor);
        let result = '';
        while (index !== -1) {
          result += value.slice(cursor, index) + replacement;
          cursor = index + search.length;
          index = lowerValue.indexOf(lowerSearch, cursor);
        }
        return index === -1 && cursor === 0 ? value : result + value.slice(cursor);
      };
      let redacted = 0;
      const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      let node;
      while ((node = walker.nextNode())) {
        const original = node.nodeValue || '';
        let value = original;
        for (const sensitivePath of sensitivePaths) {
          value = replaceInsensitive(value, sensitivePath, replacement);
        }
        if (value !== original) {
          node.nodeValue = value;
          redacted++;
        }
      }
      for (const element of document.body.querySelectorAll('*')) {
        for (const attribute of [...element.attributes]) {
          let value = attribute.value;
          for (const sensitivePath of sensitivePaths) {
            value = replaceInsensitive(value, sensitivePath, replacement);
          }
          if (value !== attribute.value) element.setAttribute(attribute.name, value);
        }
      }
      return redacted;
    }, { sensitivePaths: paths, replacement: syntheticWorkspacePath });
  };
  const assertSafeFixture = async () => {
    const domContent = (await page.evaluate(() => {
      const elements = [...document.querySelectorAll('input, textarea, select, option')];
      return [document.documentElement?.outerHTML || '', ...elements.map((element) => element.value || '')].join('\n');
    })).toLowerCase();
    const visibleText = (await page.locator('body').innerText()).toLowerCase();
    const content = `${domContent}\n${visibleText}`;
    const profilePath = process.platform === 'win32' ? process.env.USERPROFILE || '' : '';
    const forbidden = ['/projetos/', '/home/', 'desenvolvedor@', 'dev-web-kiver', 'escolanet-futura', 'sistemas2', 'proxy-nginx'];
    const leaked = forbidden.filter((value) => content.includes(value));
    if (leaked.length) throw new Error(`documentation capture contains non-synthetic data: ${leaked.join(', ')}`);
    if (profilePath && content.includes(profilePath.toLowerCase())) {
      throw new Error('documentation fixture capture still contains the current user profile');
    }
    if (content.includes(workspaceDir.toLowerCase())) {
      throw new Error('documentation fixture capture still contains its temporary workspace path');
    }
    if (!content.includes('nexus demo workspace')) throw new Error('synthetic project is not visible in capture');
  };
  const capture = async (id, file, surface, evidence = {}) => {
    await redactFixturePaths();
    if (surface === 'web/workspace') {
      const bodyText = (await page.locator('body').innerText()).toLowerCase();
      if (!bodyText.includes(syntheticWorkspacePath.toLowerCase()) && !bodyText.includes('c:\\nexusdemo\\workspace')) {
        throw new Error('workspace overview does not show the synthetic fixture path');
      }
    }
    await assertSafeFixture();
    const displayReplacement = process.platform === 'win32' ? 'C:\\\\NexusDemo\\\\workspace' : '/NexusDemo/workspace';
    const screenshotStyle = [
      `.nx-statusbar-project [class*="canonicalPath"] { font-size: 0 !important; color: transparent !important; } .nx-statusbar-project [class*="canonicalPath"]::after { content: "${displayReplacement}"; font-size: 11px; color: var(--nx-text-soft); }`,
      ...(surface === 'web/terminal' && process.platform === 'win32'
        ? ['.nx-project-shell-surface .xterm-screen { position: relative !important; } .nx-project-shell-surface .xterm-screen::after { content: ""; position: absolute; z-index: 100; inset: 0 0 auto; height: 2.5em; background: #080a0f; pointer-events: none; }']
        : []),
    ].join(' ');
    await page.screenshot({
      path: path.join(outputDir, file),
      fullPage: false,
      style: screenshotStyle,
    });
    captures.push({ id, file, source_sha: sourceSha, surface, scenario: 'isolated-synthetic-fixture', viewport: '1440x900', automated: true, capture_command: 'bun run docs:capture', data_classification: 'SYNTHETIC', status: 'PASS', ...evidence });
  };
  const openProjectSurface = async (surface, subId = '') => {
    const route = `/p/${encodeURIComponent(createProject.id)}/${surface}${subId ? `/${encodeURIComponent(subId)}` : ''}`;
    await page.goto(new URL(route, url).toString(), { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForFunction(
    () => document.querySelector('.nx-os-shell')?.isConnected === true,
    { timeout: 60000 },
  );
    await page.waitForTimeout(500);
  };
  await openProjectSurface('overview');
  const workspaceState = await dumpWorkspaceState(page);
  if (workspaceState.layoutError || workspaceState.projectId !== createProject.id) {
    throw new Error(
      'workspace state diagnostics did not resolve the synthetic project; ' +
        JSON.stringify(workspaceState),
    );
  }
  // Wait for the overview header to be interactive before capturing: on a
  // slow runner the app shell mounts before the project content hydrates, and
  // a single instant check for the direct-session button would silently skip
  // VIS-003 and fail the documentation gate later. The create-actions group
  // renders in both the workspace chrome and the overview header, so the
  // locator must target a *visible* instance instead of the first DOM match.
  try {
    await page
      .locator('[data-testid="overview-terminal-btn"]:visible')
      .first()
      .waitFor({ state: 'visible', timeout: 45000 });
  } catch (error) {
    const headerState = await page.evaluate(() => ({
      path: window.location.pathname,
      headerButtons: [...document.querySelectorAll('.nx-project-create-actions button')].map(
        (button) => button.textContent?.trim() || '',
      ),
      terminalTestIdCount: document.querySelectorAll('[data-testid="overview-terminal-btn"]')
        .length,
      terminalTestIdVisibleCount: [
        ...document.querySelectorAll('[data-testid="overview-terminal-btn"]'),
      ].filter((button) => {
        const rect = button.getBoundingClientRect();
        return rect.width > 0 && rect.height > 0;
      }).length,
      visibleAlerts: [...document.querySelectorAll('[role="alert"]')].map((alert) =>
        (alert.textContent?.trim() || '').slice(0, 200),
      ),
    }));
    throw new Error(
      'overview header did not become interactive; ' +
        JSON.stringify(headerState) + '; cause=' + error.message,
    );
  }
  await capture('VIS-001', 'workspace-overview.png', 'web/workspace');
  const directButton = page
    .locator('button:visible', { hasText: /nova sessão ia|new ai session/i })
    .first();
  if (!(await directButton.count())) {
    const headerButtons = await page.evaluate(() =>
      [...document.querySelectorAll('.nx-project-create-actions button')].map(
        (button) => button.textContent?.trim() || '',
      ),
    );
    throw new Error(
      'direct-session button is not visible in the overview header; ' +
        JSON.stringify({ path: new URL(page.url()).pathname, headerButtons }),
    );
  }
  await directButton.click();
  await page.waitForTimeout(300);
  await capture('VIS-003', 'direct-session.png', 'web/direct-session');
  await page.waitForTimeout(3000);
  await page.keyboard.press('Escape');
  trackReconnectSocket = true;
  const reconnectSocketStart = terminalSocketDiagnostics.length;
  await openProjectSurface('terminals', shellRuntime.runtime_id);
  const reconnectTimeline = [];
  // A lost WebSocket handshake only surfaces after the pane's 30s connect
  // timeout (the server may upgrade a connection the browser never sees), and
  // the product recovers through its own retry button — transport failure is
  // not proof the PTY died. The budget therefore covers several full
  // timeout-plus-retry cycles on a slow runner.
  const reconnectDeadline = Date.now() + 180000;
  let reconnected = false;
  let reconnectRetryClicks = 0;
  while (Date.now() < reconnectDeadline) {
    const probe = await page.evaluate(() => ({
      shellSurfaces: document.querySelectorAll('.nx-project-shell-surface').length,
      xtermRows: document.querySelectorAll('.nx-project-shell-surface .xterm-rows').length,
      ptyTabs: document.querySelectorAll('[id^="nx-pty-tab-"]').length,
      activeTabs: [...document.querySelectorAll('[role="tab"][aria-selected="true"]')].map(
        (tab) => tab.getAttribute('aria-label') || tab.textContent?.trim() || '',
      ),
    }));
    const reconnectEvents = terminalSocketDiagnostics.slice(reconnectSocketStart);
    reconnectTimeline.push({
      at: Date.now(),
      ...probe,
      controlLeaseReceived: reconnectControlLeaseReceived,
      retryClicks: reconnectRetryClicks,
      reconnectEvents: reconnectEvents.slice(-8),
    });
    if (probe.xtermRows > 0 && reconnectControlLeaseReceived) { reconnected = true; break; }
    // count() never waits; isEnabled() without an explicit timeout blocks for
    // the full 30s default whenever the retry button is absent, which starves
    // this polling loop exactly when the socket is still CONNECTING.
    const retryButton = page
      .locator('.nx-project-shell-surface')
      .getByRole('button', { name: /tentar novamente|retry/i })
      .first();
    if (
      reconnectRetryClicks < 5 &&
      (await retryButton.count()) > 0 &&
      (await retryButton.isEnabled({ timeout: 1000 }).catch(() => false))
    ) {
      reconnectRetryClicks += 1;
      reconnectTimeline.push({
        at: Date.now(),
        event: 'retry-clicked',
        attempt: reconnectRetryClicks,
      });
      await retryButton.click({ timeout: 3000 }).catch(() => {});
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  if (!reconnected) {
    const error = new Error('project shell terminal did not reconnect with a CONTROL lease');
    const reconnectState = await page.evaluate(() => ({
      path: window.location.pathname,
      appShell: Boolean(document.querySelector('.nx-os-shell')),
      projectShellSurfaces: [...document.querySelectorAll('.nx-project-shell-surface')].map(
        (surface) => ({
          runtimeId: surface.getAttribute('data-runtime-id') || '',
          terminal: Boolean(surface.querySelector('.xterm')),
          rows: Boolean(surface.querySelector('.xterm-rows')),
        }),
      ),
      activeTabs: [...document.querySelectorAll('[role="tab"][aria-selected="true"]')].map(
        (tab) => tab.getAttribute('aria-label') || tab.textContent?.trim() || '',
      ),
      workspacePanels: [...document.querySelectorAll('.nx-workspace-panel')].map((panel) => ({
        active: panel.dataset.active || '',
        hidden: panel.getAttribute('aria-hidden') || '',
      })),
      visibleAlerts: [...document.querySelectorAll('[role="alert"]')].map(
        (alert) => alert.textContent?.trim() || '',
      ),
    }));
    throw new Error(
      'project shell terminal did not reconnect; browser=' + JSON.stringify(reconnectState) +
        '; workspace=' + JSON.stringify(await dumpWorkspaceState(page)) +
        '; timeline=' + JSON.stringify(reconnectTimeline) +
        '; http=' + JSON.stringify(terminalHttpDiagnostics) +
        '; backendStages=' + JSON.stringify(backendStages()) +
        '; backendDiagnostics=' + JSON.stringify(backendDiagnostics()) +
        '; websocket=' + (terminalSocketDiagnostics.join(',') || 'none') +
        '; websocketHandshake=' + JSON.stringify(websocketHandshakeDiagnostics) +
        '; pageErrors=' + JSON.stringify(pageErrors) +
        '; cause=' + error.message,
    );
  }
  await page.locator('.nx-project-shell-surface .xterm').click();
  await page.keyboard.type(`echo ${reconnectMarker}`);
  await page.keyboard.press('Enter');
  await page.waitForFunction((expected) => {
    const terminal = document.querySelector('.nx-project-shell-surface .xterm-rows');
    return terminal?.textContent?.includes(expected) === true;
  }, reconnectMarker, { timeout: 15000 }).catch(async (error) => {
    throw new Error(
      'project shell terminal did not return output after reconnect; marker=' + reconnectMarker +
        '; terminal=' + JSON.stringify(await page.evaluate(() => ({
          path: window.location.pathname,
          runtimeId: document.querySelector('.nx-project-shell-surface')?.getAttribute('data-runtime-id') || '',
          text: document.querySelector('.nx-project-shell-surface .xterm-rows')?.textContent?.slice(-600) || '',
        }))) +
        '; websocket=' + JSON.stringify(terminalSocketDiagnostics.slice(reconnectSocketStart)) +
        '; backendStages=' + JSON.stringify(backendStages()) +
        '; backendDiagnostics=' + JSON.stringify(backendDiagnostics()) +
        '; cause=' + error.message,
    );
  });
  if (!reconnectMarkerOutputReceived) {
    throw new Error(
      'project shell terminal displayed no server output for the reconnect marker; marker=' +
        reconnectMarker + '; websocket=' + JSON.stringify(terminalSocketDiagnostics.slice(reconnectSocketStart)),
    );
  }
  await capture('VIS-005', 'terminal.png', 'web/terminal', {
    terminal_evidence: {
      platform: process.platform,
      transport: 'project-shell',
      marker,
      reconnect_marker: reconnectMarker,
      localized_shell_banner_hidden: process.platform === 'win32',
      verified: true,
    },
  });
  await page.waitForTimeout(2500);
  const candidates = [
    { id: 'VIS-004', file: 'provider.png', surface: 'web/provider', pattern: /providers|provedores/i },
    { id: 'VIS-007', file: 'composer.png', surface: 'web/composer', pattern: /composer/i },
    { id: 'VIS-008', file: 'flow.png', surface: 'web/flow', pattern: /flow/i },
    { id: 'VIS-006', file: 'agents.png', surface: 'web/agents', pattern: /agents|agentes/i },
    { id: 'VIS-009', file: 'mission.png', surface: 'web/mission', pattern: /flow runs|missions|mission/i },
    { id: 'VIS-010', file: 'usage-quota.png', surface: 'web/usage-quota', pattern: /uso|usage|quota|resources/i },
  ];
  for (const candidate of candidates) {
    const button = page.getByRole('button', { name: candidate.pattern }).first();
    if (await button.count() && await button.isVisible().catch(() => false)) {
      await button.click();
      await page.waitForTimeout(300);
      await capture(candidate.id, candidate.file, candidate.surface);
    }
  }
  const projectMatch = new URL(page.url()).pathname.match(/\/p\/([^/]+)/);
  if (projectMatch) {
    const providersUrl = new URL(`/p/${projectMatch[1]}/legacy-providers`, url);
    await page.goto(providersUrl.toString(), { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForFunction(
    () => document.querySelector('.nx-os-shell')?.isConnected === true,
    { timeout: 60000 },
  );
    await capture('VIS-004', 'provider.png', 'web/provider');
  }
  if (!captures.some((c) => c.id === 'VIS-007')) {
    await openProjectSurface('work');
    await capture('VIS-007', 'composer.png', 'web/composer');
  }
  if (!captures.some((c) => c.id === 'VIS-008')) {
    await openProjectSurface('missions');
    await capture('VIS-008', 'flow.png', 'web/flow');
  }
  if (!captures.some((c) => c.id === 'VIS-009')) {
    await openProjectSurface('missions');
    await capture('VIS-009', 'mission.png', 'web/mission');
  }
  await openProjectSurface('projects');
  await page.waitForTimeout(500);
  await capture('VIS-002', 'projects.png', 'web/projects');
  if (pageErrors.length) throw new Error(`page errors during capture: ${pageErrors.join(' | ')}`);
  await context.close();
  await browser.close();
  if (video) {
    const videoPath = await video.path();
    renameSync(videoPath, path.join(demoDir, 'direct-workflow.webm'));
  }
  captureFailed = false;
} finally {
  if (captureFailed) copyFixtureForensics();
  await stop().catch((error) => {
    console.warn('documentation fixture cleanup failed:', error?.message || error);
    process.exitCode = process.exitCode || 1;
  });
}
const manifestFile = path.join(outputDir, 'manifest.json');
let previousCaptures = [];
if (existsSync(manifestFile)) {
  try {
    const previous = JSON.parse(readFileSync(manifestFile, 'utf8'));
    previousCaptures = (previous.captures || []).filter(
      (item) => item.source_sha === sourceSha ||
        (item.id === 'VIS-011' && item.data_classification === 'SYNTHETIC'),
    );
  } catch {
    previousCaptures = [];
  }
}
const merged = new Map(previousCaptures.map((item) => [item.id, item]));
for (const item of captures) merged.set(item.id, item);
const manifest = {
  source_sha: sourceSha,
  generated_by: 'web/scripts/docs-capture.mjs',
  scenario: 'isolated-synthetic-fixture',
  data_classification: 'SYNTHETIC',
  captures: [...merged.values()],
};
writeFileSync(path.join(outputDir, 'manifest.json'), `${JSON.stringify(manifest, null, 2)}\n`);
console.log(`captured ${captures.length} isolated synthetic surface(s); manifest contains ${manifest.captures.length} at ${sourceSha}`);
