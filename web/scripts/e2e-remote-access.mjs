#!/usr/bin/env node
/**
 * Two-device remote access E2E.
 *
 * The host (loopback) starts a real Cloudflare Quick Tunnel and issues two
 * independent invitations. Two separate mobile browsers then pair over the
 * public HTTPS URL with the human-typed access code:
 *
 *   - controller phone: acts on the workspace and can be revoked by the host
 *   - viewer phone: read-only, refused device management and mutations
 *
 * Covered: the pairing screen on a 390px viewport (no horizontal overflow),
 * role enforcement, the device registry, host-side revocation and the denial
 * that follows it.
 *
 * The Quick Tunnel is used because it needs no account. A named tunnel only
 * changes the hostname, not these flows.
 *
 * Set NEXUS_TUNNEL_TOKEN and NEXUS_TUNNEL_HOSTNAME to run the same scenario
 * against a Named Tunnel. In that mode the script additionally restarts the
 * Nexus process and asserts the public origin is byte-identical, which is the
 * property the installed PWA depends on.
 */
import { spawn, spawnSync } from 'node:child_process';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import {
  chmodSync,
  copyFileSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  rmSync,
  writeFileSync,
} from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';
import { ensureNexusBinary } from './e2e-platform.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const webDir = path.resolve(__dirname, '..');
const repoRoot = path.resolve(webDir, '..');

// Same pinned digests as internal/control/web/tunnel.go: an unverified
// cloudflared would own the tunnel endpoint of the control plane.
const CLOUDFLARED_SHA256 = {
  'linux-amd64': 'fcfb02b575a52ca1af2e3267af4e1517bcdeb30ac48c834c69abaed3c0576ad2',
  'linux-arm64': '7747d94570fb390cf47dcb4f9555c193c6355cda9793f0d878d9049e5d6a7790',
};
const CLOUDFLARED_RELEASE = 'https://github.com/cloudflare/cloudflared/releases/download/2026.8.2';

const IPHONE_UA =
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1';
const ANDROID_UA =
  'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36';

async function waitFor(label, fn, { timeout = 30000, interval = 250 } = {}) {
  const deadline = Date.now() + timeout;
  let lastError = null;
  while (Date.now() < deadline) {
    try {
      const result = await fn();
      if (result) return result;
    } catch (error) {
      lastError = error;
    }
    await new Promise((resolve) => setTimeout(resolve, interval));
  }
  throw new Error(`timeout waiting for ${label}${lastError ? `: ${lastError.message}` : ''}`);
}

function sha256(buffer) {
  return createHash('sha256').update(buffer).digest('hex');
}

/**
 * Places a checksum-verified cloudflared in the Nexus-managed data dir so the
 * tunnel starts without a 40MB download on every run.
 */
async function prepareCloudflared(dataDir) {
  const arch = { x64: 'amd64', arm64: 'arm64' }[process.arch] || process.arch;
  const key = `${process.platform === 'win32' ? 'windows' : process.platform}-${arch}`;
  const expected = CLOUDFLARED_SHA256[key];
  if (!expected) {
    throw new Error(`no pinned cloudflared digest for ${key}`);
  }
  const binDir = path.join(dataDir, 'bin');
  mkdirSync(binDir, { recursive: true });
  const dest = path.join(binDir, process.platform === 'win32' ? 'cloudflared.exe' : 'cloudflared');

  const candidates = [
    process.env.NEXUS_E2E_CLOUDFLARED,
    path.join(os.homedir(), '.local', 'bin', 'cloudflared'),
    '/usr/local/bin/cloudflared',
    '/usr/bin/cloudflared',
  ].filter(Boolean);

  for (const candidate of candidates) {
    if (!existsSync(candidate)) continue;
    const { readFileSync } = await import('node:fs');
    const buffer = readFileSync(candidate);
    if (sha256(buffer) !== expected) continue;
    writeFileSync(dest, buffer);
    chmodSync(dest, 0o755);
    return dest;
  }

  console.log('   cloudflared not cached locally, downloading the pinned release...');
  const suffix = process.platform === 'win32' ? '.exe' : '';
  const url = `${CLOUDFLARED_RELEASE}/cloudflared-${key}${suffix}`;
  const response = await fetch(url, { redirect: 'follow' });
  if (!response.ok) {
    throw new Error(`cloudflared download failed: HTTP ${response.status}`);
  }
  const buffer = Buffer.from(await response.arrayBuffer());
  const digest = sha256(buffer);
  if (digest !== expected) {
    throw new Error(`cloudflared checksum mismatch: expected ${expected}, got ${digest}`);
  }
  writeFileSync(dest, buffer);
  chmodSync(dest, 0o755);
  return dest;
}

/** Calls a JSON API from inside the page, so cookies and Origin are real. */
function apiCall(page) {
  return (apiPath, { method = 'GET', body, csrf, timeoutMs = 0 } = {}) =>
    page.evaluate(
      async ({ apiPath, method, body, csrf, timeoutMs }) => {
        const headers = { Accept: 'application/json' };
        if (body !== undefined) headers['Content-Type'] = 'application/json';
        if (csrf) headers['X-CSRF-Token'] = csrf;
        const controller = timeoutMs > 0 ? new AbortController() : null;
        const timer = timeoutMs > 0 ? setTimeout(() => controller.abort(), timeoutMs) : null;
        try {
          const res = await fetch(apiPath, {
            method,
            headers,
            credentials: 'same-origin',
            body: body === undefined ? undefined : JSON.stringify(body),
            signal: controller ? controller.signal : undefined,
          });
          const text = await res.text();
          let parsed = null;
          try {
            parsed = text ? JSON.parse(text) : null;
          } catch {
            parsed = text;
          }
          return { status: res.status, body: parsed };
        } finally {
          if (timer) clearTimeout(timer);
        }
      },
      { apiPath, method, body, csrf, timeoutMs },
    );
}

async function pairPhone(context, tunnelUrl, code, expectedRole, label) {
  const page = await context.newPage();
  const call = apiCall(page);
  await page.goto(tunnelUrl, { waitUntil: 'domcontentloaded', timeout: 90000 });

  const input = page.locator('#access-code-input');
  await input.waitFor({ state: 'visible', timeout: 90000 });

  // The pairing screen must fit a 390px handset without horizontal scroll.
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - window.innerWidth,
  );
  assert.ok(
    overflow <= 1,
    `${label}: horizontal overflow of ${overflow}px at 390px on the pairing screen`,
  );

  await input.fill(code);
  await page.locator('form:has(#access-code-input) button[type="submit"]').click();

  const session = await waitFor(
    `${label} paired session`,
    async () => {
      const res = await call('/api/v1/session');
      return res.body && res.body.authenticated ? res.body : null;
    },
    { timeout: 90000 },
  );
  assert.equal(session.role, expectedRole, `${label}: unexpected session role`);
  return { page, session, call };
}

/** Boots the Nexus web server and resolves its one-time bootstrap URL. */
async function startServer(binPath, repoRoot, serverEnv, port) {
  const proc = spawn(binPath, ['web', '--port', String(port), '--no-open'], {
    cwd: repoRoot,
    env: serverEnv,
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  proc.stdout.on('data', () => undefined);
  proc.stderr.on('data', () => undefined);
  const bootstrapUrl = await waitFor(
    'server bootstrap URL',
    () => {
      const result = spawnSync(binPath, ['web', 'url'], {
        cwd: repoRoot,
        encoding: 'utf8',
        timeout: 2000,
        env: serverEnv,
      });
      const candidate = String(result.stdout || '').trim();
      return /#nexus_bootstrap=[a-f0-9]+$/i.test(candidate) ? candidate : null;
    },
    { timeout: 30000, interval: 300 },
  );
  return { proc, bootstrapUrl };
}

async function stopServer(proc) {
  if (!proc) return;
  proc.kill('SIGTERM');
  await new Promise((resolve) => setTimeout(resolve, 800));
  if (!proc.killed) proc.kill('SIGKILL');
}

async function main() {
  console.log('=== Nexus Two-Device Remote Access E2E ===');
  console.log('1. Building nexus binary from the current checkout...');
  const binPath = ensureNexusBinary(repoRoot);
  assert.ok(existsSync(binPath), 'nexus binary must exist after build');

  const dataDir = mkdtempSync(path.join(os.tmpdir(), 'nexus-remote-e2e-'));
  const serverEnv = { ...process.env, NEXUS_DATA_DIR: dataDir };
  // A named tunnel is the Gate 0 configuration: the hostname is fixed, so
  // the installed PWA and its session cookie survive a Nexus restart.
  const namedToken = String(process.env.NEXUS_TUNNEL_TOKEN || '').trim();
  const namedHostname = String(process.env.NEXUS_TUNNEL_HOSTNAME || '').trim();
  const namedMode = Boolean(namedToken && namedHostname);
  // A named tunnel applies the ingress rule from the Cloudflare dashboard,
  // which points at a fixed http://localhost:<port>. The Nexus must therefore
  // run on that fixed port, and the restart must reuse it.
  const serverPort = namedMode ? Number(process.env.NEXUS_E2E_PORT || 13400) : 0;
  let browser;
  let server;

  try {
    console.log('2. Installing a checksum-verified cloudflared...');
    await prepareCloudflared(dataDir);

    console.log(`3. Starting Nexus Web Server on port ${serverPort || 'ephemeral'}...`);
    const startedServer = await startServer(binPath, repoRoot, serverEnv, serverPort);
    server = startedServer.proc;
    const bootstrapUrl = startedServer.bootstrapUrl;
    console.log('   Server bound, bootstrap URL resolved.');

    browser = await chromium.launch({
      headless: true,
      args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
    });

    console.log('4. Bootstrapping the host session...');
    const hostContext = await browser.newContext({ viewport: { width: 1440, height: 900 } });
    const hostPage = await hostContext.newPage();
    const hostCall = apiCall(hostPage);
    await hostPage.goto(bootstrapUrl, { waitUntil: 'domcontentloaded', timeout: 60000 });
    const hostSession = await waitFor(
      'host session',
      async () => {
        const res = await hostCall('/api/v1/session');
        return res.body && res.body.authenticated ? res.body : null;
      },
      { timeout: 60000 },
    );
    const hostCsrf = hostSession.csrf_token;
    assert.ok(hostCsrf, 'host session must carry a CSRF token');

    console.log('5. Starting the public tunnel...');
    const started = await hostCall('/api/v1/tunnel/start', {
      method: 'POST',
      csrf: hostCsrf,
      timeoutMs: 180000,
    });
    assert.equal(started.status, 200, `tunnel start failed: ${JSON.stringify(started.body)}`);
    const tunnelUrl = started.body && started.body.url;
    assert.ok(tunnelUrl && tunnelUrl.startsWith('https://'), 'tunnel must expose an HTTPS URL');
    // The operator-facing contract: the status must declare whether the
    // public origin survives a restart. A Quick Tunnel must never claim a
    // stable origin, because its hostname is minted on every start.
    assert.equal(
      started.body.stable_url,
      namedMode,
      `tunnel status must declare stable_url=${namedMode}, got ${JSON.stringify(started.body)}`,
    );
    if (namedMode) {
      assert.equal(
        new URL(tunnelUrl).hostname,
        namedHostname.toLowerCase(),
        'a named tunnel must serve the configured hostname',
      );
      assert.equal(
        started.body.named_hostname,
        namedHostname.toLowerCase(),
        'a named tunnel must report the configured hostname',
      );
      console.log(`   Named tunnel active on the configured hostname: ${namedHostname}`);
      console.log('   The status declares a stable origin: the installed app survives a restart.');
    } else {
      assert.equal(
        started.body.named_hostname,
        undefined,
        'a quick tunnel must not report a named hostname',
      );
      console.log(`   Tunnel active: ${tunnelUrl}`);
      console.log(
        '   The status declares an ephemeral origin: the installed app breaks on restart.',
      );
    }

    console.log('6. Issuing a controller and a viewer invitation...');
    const controllerInvite = await hostCall('/api/v1/tunnel/invite', {
      method: 'POST',
      body: { role: 'controller' },
      csrf: hostCsrf,
    });
    assert.equal(controllerInvite.status, 200, 'controller invitation failed');
    const viewerInvite = await hostCall('/api/v1/tunnel/invite', {
      method: 'POST',
      body: { role: 'viewer' },
      csrf: hostCsrf,
    });
    assert.equal(viewerInvite.status, 200, 'viewer invitation failed');
    assert.ok(controllerInvite.body.access_code, 'controller invitation must carry an access code');
    assert.ok(viewerInvite.body.access_code, 'viewer invitation must carry an access code');
    assert.notEqual(
      controllerInvite.body.access_code,
      viewerInvite.body.access_code,
      'each invitation must mint its own code',
    );

    console.log('7. Pairing the controller phone (390px)...');
    const iphoneContext = await browser.newContext({
      viewport: { width: 390, height: 844 },
      deviceScaleFactor: 3,
      isMobile: true,
      hasTouch: true,
      userAgent: IPHONE_UA,
    });
    const iphone = await pairPhone(
      iphoneContext,
      tunnelUrl,
      controllerInvite.body.access_code,
      'controller',
      'controller phone',
    );
    console.log('   Controller phone paired.');

    console.log('8. Pairing the viewer phone (390px)...');
    const androidContext = await browser.newContext({
      viewport: { width: 390, height: 844 },
      deviceScaleFactor: 3,
      isMobile: true,
      hasTouch: true,
      userAgent: ANDROID_UA,
    });
    const android = await pairPhone(
      androidContext,
      tunnelUrl,
      viewerInvite.body.access_code,
      'viewer',
      'viewer phone',
    );
    console.log('   Viewer phone paired.');

    console.log('9. Asserting the PWA is installable from the public origin...');
    // The installable app lives at the origin the phone opens. If the
    // manifest or the service worker were only reachable on loopback, the
    // phone could never install or cache the app.
    const publicManifest = await iphone.page.request.get(
      new URL('/manifest.webmanifest', tunnelUrl).toString(),
    );
    assert.equal(publicManifest.status(), 200, 'the manifest must be served by the public origin');
    const manifestBody = await publicManifest.json();
    assert.equal(manifestBody.start_url, '/', 'the manifest must start at the origin root');
    assert.ok(
      manifestBody.display && manifestBody.display !== 'browser',
      'the manifest must request an installable display mode',
    );
    const publicServiceWorker = await iphone.page.request.get(
      new URL('/sw.js', tunnelUrl).toString(),
    );
    assert.equal(
      publicServiceWorker.status(),
      200,
      'the service worker must be served by the public origin',
    );
    const serviceWorkerBody = await publicServiceWorker.text();
    assert.ok(
      serviceWorkerBody.includes("addEventListener('push'"),
      'the public service worker must handle push events',
    );
    // The registration must succeed on the public origin, otherwise the
    // installed app never caches its shell offline.
    const registration = await iphone.page.evaluate(async () => {
      if (!('serviceWorker' in navigator)) return 'unsupported';
      const reg = await navigator.serviceWorker.getRegistration();
      return reg ? 'registered' : 'absent';
    });
    assert.notEqual(
      registration,
      'absent',
      'the service worker must be registered on the public origin',
    );
    console.log(`   Public origin serves the installable app (${registration}).`);

    console.log('10. Asserting the viewer session is read-only...');
    const viewerDeviceList = await android.call('/api/v1/remote/devices');
    assert.equal(
      viewerDeviceList.status,
      403,
      'a paired phone must not enumerate the owner devices',
    );

    const workspacesBefore = await hostCall('/api/v1/workspaces');
    const viewerMutation = await android.call('/api/v1/workspaces', {
      method: 'POST',
      body: { path: '/tmp/nexus-viewer-should-not-exist', name: 'viewer-should-not-exist' },
      csrf: android.session.csrf_token,
    });
    assert.equal(viewerMutation.status, 403, 'a viewer session must be refused on mutations');
    assert.match(
      String(viewerMutation.body?.error || ''),
      /read-only/i,
      'the refusal must name the read-only rule',
    );
    const workspacesAfter = await hostCall('/api/v1/workspaces');
    assert.equal(
      JSON.stringify(workspacesAfter.body),
      JSON.stringify(workspacesBefore.body),
      'a refused viewer mutation must not change the workspace',
    );
    console.log('   Viewer refused on device listing and on mutation, with no side effect.');

    console.log('11. Asserting the host device registry...');
    const devices = await hostCall('/api/v1/remote/devices');
    assert.equal(devices.status, 200);
    const list = (devices.body?.devices || []).map((device) => ({
      id: device.device_id,
      label: device.label,
      kind: device.kind,
    }));
    assert.equal(list.length, 2, `expected 2 paired devices, got ${JSON.stringify(list)}`);
    for (const device of list) {
      assert.ok(device.id, 'each device must expose an opaque handle');
      assert.equal(device.kind, 'mobile', 'a paired handset must be classified as mobile');
    }
    const labels = list.map((device) => device.label);
    assert.notEqual(labels[0], labels[1], 'the two handsets must be distinguishable by label');
    console.log(`   Registry: ${labels.join(' | ')}`);

    console.log('12. Revoking the controller phone from the host...');
    // Map each registry entry back to its handset by label: the iOS phone
    // reports "Safari · iOS" and the Android one "Chrome · Android".
    const iosDevice = list.find((device) => /iOS/i.test(device.label));
    assert.ok(iosDevice, `no iOS device in the registry: ${JSON.stringify(list)}`);
    const revoked = await hostCall(
      `/api/v1/remote/devices?device_id=${encodeURIComponent(iosDevice.id)}`,
      { method: 'DELETE', csrf: hostCsrf },
    );
    assert.equal(revoked.status, 200, 'revocation must succeed');
    assert.equal(revoked.body?.revoked, iosDevice.id);

    const revokedCall = apiCall(iphone.page);
    const survivorCall = apiCall(android.page);

    const afterRevoke = await revokedCall('/api/v1/attention');
    assert.equal(afterRevoke.status, 401, 'a revoked device must lose the session');

    const survivor = await survivorCall('/api/v1/attention');
    assert.equal(survivor.status, 200, 'revoking one device must not touch the other');
    console.log('   Revoked device denied, sibling device still authorized.');

    console.log('13. Restarting Nexus and asserting the public origin is unchanged...');
    // This is the Gate 0 property: the installed PWA, its service worker and
    // its host-bound cookie all point at the public origin. If the origin
    // changes across a restart, the icon on the phone opens a dead host.
    const originBefore = new URL(tunnelUrl).origin;
    await stopServer(server);
    const restarted = await startServer(binPath, repoRoot, serverEnv, serverPort);
    server = restarted.proc;

    const restartContext = await browser.newContext({ viewport: { width: 1440, height: 900 } });
    const restartPage = await restartContext.newPage();
    const restartCall = apiCall(restartPage);
    await restartPage.goto(restarted.bootstrapUrl, {
      waitUntil: 'domcontentloaded',
      timeout: 60000,
    });
    const restartSession = await waitFor(
      'host session after restart',
      async () => {
        const res = await restartCall('/api/v1/session');
        return res.body && res.body.authenticated ? res.body : null;
      },
      { timeout: 60000 },
    );
    const restartedTunnel = await restartCall('/api/v1/tunnel/start', {
      method: 'POST',
      csrf: restartSession.csrf_token,
      timeoutMs: 180000,
    });
    assert.equal(
      restartedTunnel.status,
      200,
      `tunnel start after restart failed: ${JSON.stringify(restartedTunnel.body)}`,
    );
    const originAfter = new URL(restartedTunnel.body.url).origin;
    assert.equal(
      restartedTunnel.body.stable_url,
      namedMode,
      `the status after a restart must still declare stable_url=${namedMode}`,
    );
    if (namedMode) {
      assert.equal(
        originAfter,
        originBefore,
        'a named tunnel must serve the same origin after a restart, otherwise the installed PWA breaks',
      );
      console.log(`   Origin stable across restart: ${originAfter}`);
    } else {
      // The Quick Tunnel mints a random hostname per start. That is the exact
      // defect the Named Tunnel removes, so the change is reported instead of
      // asserted away.
      console.log(`   Origin before restart: ${originBefore}`);
      console.log(`   Origin after restart:  ${originAfter}`);
      console.log('   Quick Tunnel changed the origin, which is why the Named Tunnel is required');
      console.log('   for an installed PWA. Set NEXUS_TUNNEL_TOKEN/NEXUS_TUNNEL_HOSTNAME.');
    }

    console.log('\n=== Two-device remote access E2E PASSED ===');
    console.log(`   Paired over: ${tunnelUrl}`);
    console.log(`   Devices: ${labels.join(' | ')}`);
    if (!namedMode) {
      console.log('   NOTE: the Quick Tunnel mints a random hostname per start, so the origin');
      console.log('         above differs across restarts. Set NEXUS_TUNNEL_TOKEN and');
      console.log('         NEXUS_TUNNEL_HOSTNAME to prove the stable hostname.');
    }
  } finally {
    if (browser) await browser.close();
    await stopServer(server);
    rmSync(dataDir, { recursive: true, force: true });
  }
}

main().catch((error) => {
  console.error('\n=== Two-device remote access E2E FAILED ===');
  console.error(error?.stack || String(error));
  process.exitCode = 1;
});
