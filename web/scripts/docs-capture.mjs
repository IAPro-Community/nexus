#!/usr/bin/env node
import { spawn, spawnSync } from 'node:child_process';
import {
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
try {
  const url = await bootstrap;
  const browser = await chromium.launch({
    executablePath: existsSync('/usr/bin/google-chrome') ? '/usr/bin/google-chrome' : undefined,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
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
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 20000 });
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
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 20000 });
  await page.waitForFunction(
    () => document.body.innerText.includes('Projetos') || document.body.innerText.includes('Projects') || document.querySelector('.nx-os-shell'),
    { timeout: 15000 },
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

  const shell = await page.evaluate(async ({ projectId, csrf }) => {
    const response = await fetch(`/api/v1/projects/${encodeURIComponent(projectId)}/shell`, {
      method: 'POST',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json', 'X-CSRF-Token': csrf },
      body: '{}',
    });
    if (!response.ok) throw new Error(`fixture shell failed: HTTP ${response.status}`);
    return response.json();
  }, { projectId: createProject.id, csrf: session.csrf_token });

  // Refresh app state so the newly created synthetic project is present in the workspace route resolver.
  const refreshResponse = await page.reload({ waitUntil: 'domcontentloaded', timeout: 20000 });
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
    { waitUntil: 'domcontentloaded', timeout: 20000 },
  );
  await page.waitForSelector('.nx-os-shell', { timeout: 15000 });
  const projectRoute = '/p/' + encodeURIComponent(createProject.id) +
    '/terminals/' + encodeURIComponent(shell.runtime.runtime_id);

  let resolveTerminalControl;
  const terminalControlReady = new Promise((resolve) => { resolveTerminalControl = resolve; });
  const terminalSocketDiagnostics = [];
  page.on('websocket', (socket) => {
    terminalSocketDiagnostics.push('opened');
    socket.on('close', (code) => terminalSocketDiagnostics.push('closed:' + code));
    socket.on('socketerror', () => terminalSocketDiagnostics.push('socket-error'));
    socket.on('framereceived', ({ payload }) => {
      try {
        const frame = JSON.parse(String(payload));
        terminalSocketDiagnostics.push('frame:' + String(frame.type || 'unknown') + ':' + String(frame.role || ''));
        if (frame.type === 'lease' && frame.role === 'CONTROL') resolveTerminalControl();
      } catch {
        terminalSocketDiagnostics.push('frame:non-json');
      }
    });
  });
  await page.goto(new URL(projectRoute, url).toString(), { waitUntil: 'domcontentloaded', timeout: 20000 });
  await page.waitForSelector('.nx-os-shell', { timeout: 15000 });
  await page.waitForSelector('.nx-project-shell-surface', { timeout: 15000 });
  await page.waitForFunction(() => {
    const body = document.body.textContent || '';
    return !/desconectado|disconnected|erro|error|recuperando|recovering/i.test(body);
  }, { timeout: 15000 });
  const terminalUiState = await page.evaluate(() => {
    const surface = document.querySelector('.nx-project-shell-surface');
    return {
      projectShellSurface: Boolean(surface),
      terminal: Boolean(surface?.querySelector('.xterm')),
      rows: Boolean(surface?.querySelector('.xterm-rows')),
    };
  });
  let leaseTimer;
  await Promise.race([
    terminalControlReady,
    new Promise((_, reject) => {
      leaseTimer = setTimeout(() => reject(new Error(
        'project shell writer lease was not ready; websocket=' +
          (terminalSocketDiagnostics.join(',') || 'none') +
          '; terminalUi=' + JSON.stringify(terminalUiState) +
          '; pageErrors=' + pageErrors.length,
      )), 5000);
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
    return page.evaluate((sensitivePaths) => {
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
      const replacement = 'C:\\NexusDemo\\workspace';
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
    }, paths);
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
      if (!bodyText.includes('c:\\nexusdemo\\workspace')) {
        throw new Error('workspace overview does not show the synthetic fixture path');
      }
    }
    await assertSafeFixture();
    await page.screenshot({
      path: path.join(outputDir, file),
      fullPage: false,
      style: '.nx-statusbar-project [class*="canonicalPath"] { font-size: 0 !important; color: transparent !important; } .nx-statusbar-project [class*="canonicalPath"]::after { content: "C:\\\\NexusDemo\\\\workspace"; font-size: 11px; color: var(--nx-text-soft); }',
    });
    captures.push({ id, file, source_sha: sourceSha, surface, scenario: 'isolated-synthetic-fixture', viewport: '1440x900', automated: true, capture_command: 'bun run docs:capture', data_classification: 'SYNTHETIC', status: 'PASS', ...evidence });
  };
  const openProjectSurface = async (surface, subId = '') => {
    const route = `/p/${encodeURIComponent(createProject.id)}/${surface}${subId ? `/${encodeURIComponent(subId)}` : ''}`;
    await page.goto(new URL(route, url).toString(), { waitUntil: 'domcontentloaded', timeout: 20000 });
    await page.waitForSelector('.nx-os-shell', { timeout: 15000 });
    await page.waitForTimeout(500);
  };
  await openProjectSurface('overview');
  await capture('VIS-001', 'workspace-overview.png', 'web/workspace');
  const directButton = page.getByRole('button', { name: /nova sessão ia|new ai session/i }).first();
  if (await directButton.count() && await directButton.isVisible().catch(() => false)) {
    await directButton.click();
    await page.waitForTimeout(300);
    await capture('VIS-003', 'direct-session.png', 'web/direct-session');
    await page.waitForTimeout(3000);
    await page.keyboard.press('Escape');
  }
  await openProjectSurface('terminals', shell.runtime.runtime_id);
  await page.waitForSelector('.nx-project-shell-surface .xterm-rows', { timeout: 15000 });
  await capture('VIS-005', 'terminal.png', 'web/terminal', {
    terminal_evidence: { platform: process.platform, transport: 'project-shell', marker, verified: true },
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
    await page.goto(providersUrl.toString(), { waitUntil: 'domcontentloaded', timeout: 20000 });
    await page.waitForSelector('.nx-os-shell', { timeout: 15000 });
    await capture('VIS-004', 'provider.png', 'web/provider');
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
} finally {
  await stop();
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
