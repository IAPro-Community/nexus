#!/usr/bin/env node
// Browser half of TestACPAttentionRadarBrowserIntegration. It never alters
// runtime API responses; the Go harness supplies real SessionHost attention.
import assert from 'node:assert/strict';
import { mkdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { launchFixtureBrowser, meaningfulFixtureRequestFailures } from './e2e-fixture.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const apiOrigin = process.env.NEXUS_E2E_BROWSER_ORIGIN;
const bootstrapToken = process.env.NEXUS_E2E_BROWSER_BOOTSTRAP;
const alphaProjectId = process.env.NEXUS_E2E_BROWSER_ALPHA_PROJECT;
const betaProjectId = process.env.NEXUS_E2E_BROWSER_BETA_PROJECT;
const alphaRuntimeId = process.env.NEXUS_E2E_BROWSER_ALPHA_RUNTIME;
const betaRuntimeId = process.env.NEXUS_E2E_BROWSER_BETA_RUNTIME;
const artifacts =
  process.env.NEXUS_E2E_ARTIFACTS_DIR || '/tmp/nexus-attention-radar-process-api-20261108';
for (const [name, value] of Object.entries({
  apiOrigin,
  bootstrapToken,
  alphaProjectId,
  betaProjectId,
  alphaRuntimeId,
  betaRuntimeId,
}))
  assert.ok(value, `missing ${name} from Go integration harness`);

async function main() {
  mkdirSync(artifacts, { recursive: true });
  const bootstrapUrl = `${apiOrigin}/#nexus_bootstrap=${bootstrapToken}`;
  const { browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl);
  try {
    const auth = await context.request.post(`${apiOrigin}/api/v1/auth/bootstrap`, {
      headers: { 'Content-Type': 'application/json' },
      data: { token: bootstrapToken },
    });
    assert.equal(auth.status(), 200, 'browser session must authenticate against the real server');
    const runtimesResponse = await context.request.get(`${apiOrigin}/api/v1/runtimes`);
    assert.equal(runtimesResponse.status(), 200);
    const runtimes = await runtimesResponse.json();
    const byId = new Map(runtimes.map((runtime) => [runtime.runtime_id, runtime]));
    const alpha = byId.get(alphaRuntimeId);
    const beta = byId.get(betaRuntimeId);
    assert.equal(alpha?.attention_kind, 'needs_user');
    assert.equal(beta?.attention_kind, 'needs_user');
    assert.equal(alpha?.attention_context, beta?.attention_context);
    assert.notEqual(alpha?.attention_fingerprint, beta?.attention_fingerprint);
    assert.equal(alpha?.attention_source, 'AUTHORITATIVE');
    assert.equal(beta?.attention_source, 'AUTHORITATIVE');
    assert.deepEqual(
      alpha?.attention_options?.map(({ option_id }) => option_id),
      ['deny', 'allow_once'],
      'the exact provider-defined options must reach the authenticated runtime API',
    );
    assert.equal(alpha?.project_id, alphaProjectId);
    assert.equal(beta?.project_id, betaProjectId);

    const websocketPaths = [];
    page.on('websocket', (socket) => websocketPaths.push(new URL(socket.url()).pathname));
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto(`${apiOrigin}/p/${encodeURIComponent(alphaProjectId)}`, {
      waitUntil: 'domcontentloaded',
      timeout: 15000,
    });
    const notifications = page.getByRole('button', {
      name: /Central de Notificações e Radar|Notifications and Radar/i,
    });
    await notifications.waitFor({ state: 'visible', timeout: 15000 });
    const alphaAttention = page.getByRole('alertdialog', {
      name: /Attention Radar Alpha/,
    });
    await page.screenshot({
      path: path.join(artifacts, 'process-attention-structured-options.png'),
      fullPage: true,
    });
    const allowOnce = alphaAttention.getByRole('button', { name: 'Allow once', exact: true });
    await allowOnce.waitFor({ state: 'visible', timeout: 10000 });
    await allowOnce.click();
    await page.waitForFunction(
      async (alphaId) => {
        const response = await fetch('/api/v1/runtimes');
        const runtimes = await response.json();
        const alphaRuntime = runtimes.find((runtime) => runtime.runtime_id === alphaId);
        const betaRuntime = runtimes.find((runtime) => runtime.runtime_id === 'radar-browser-beta');
        return alphaRuntime?.attention_kind === 'completed' && betaRuntime?.attention_kind === 'needs_user';
      },
      alphaRuntimeId,
      { timeout: 10000 },
    );
    console.log('PASS UI option is delivered by interaction ID; only Alpha settles and Beta remains pending');
    const toastDismiss = page.locator('.nx-uui-notification__close');
    const dismissToastCards = async () => {
      const initialCount = await toastDismiss.count();
      for (let index = 0; index < initialCount; index += 1) {
        await toastDismiss.first().click();
        await page.waitForFunction(
          (remaining) =>
            document.querySelectorAll('.nx-uui-notification__close').length < remaining,
          initialCount - index,
        );
      }
      const response = await context.request.get(`${apiOrigin}/api/v1/runtimes`);
      const stillPending = await response.json();
      const pendingById = new Map(stillPending.map((runtime) => [runtime.runtime_id, runtime]));
      assert.equal(pendingById.get(alphaRuntimeId)?.attention_kind, 'completed');
      assert.equal(pendingById.get(betaRuntimeId)?.attention_kind, 'needs_user');
    };
    const openRadar = async () => {
      // Persistent toast cards can overlap the drawer tab after a project
      // switch; dismissing them leaves Beta's attention active in the Radar.
      await dismissToastCards();
      await notifications.click();
      const radarTab = page.getByRole('tab', { name: /radar/i });
      await radarTab.waitFor({ state: 'visible', timeout: 5000 });
      await radarTab.click();
    };
    await openRadar();
    const betaRow = page.locator(`button[data-runtime-id="${betaRuntimeId}"]`);
    await betaRow.waitFor({ state: 'visible', timeout: 10000 });
    assert.match(await betaRow.innerText(), /precisa de você/i);
    await page.screenshot({
      path: path.join(artifacts, 'process-attention-radar-beta-pending.png'),
    });

    await betaRow.click();
    const betaSurface = page.locator(
      `.nx-project-shell-surface[data-runtime-id="${betaRuntimeId}"]`,
    );
    await betaSurface.waitFor({ state: 'attached', timeout: 15000 });
    assert.ok(page.url().includes(encodeURIComponent(betaProjectId)));
    await betaSurface.locator('.xterm').waitFor({ state: 'attached', timeout: 10000 });
    assert.equal(
      await page.getByText('Runtime compatibility validation failed.', { exact: true }).count(),
      0,
    );
    assert.ok(
      websocketPaths.some((value) => value.endsWith(`/runtimes/${betaRuntimeId}/terminal`)),
    );

    assert.equal(collectors.pageErrors.length, 0, collectors.pageErrors.join(' | '));
    assert.deepEqual(meaningfulFixtureRequestFailures(collectors.requestFailures), []);
    console.log(
      'PASS real ACP subprocess attention is visible through Registry and /api/v1/runtimes without API overlay',
    );
    console.log(
      'PASS identical questions stay distinct and cross-project Radar opens the exact runtime',
    );
    console.log(
      'PASS browser attaches the remaining real SessionHost terminal WebSocket; external provider is not called',
    );
  } finally {
    await context.close().catch(() => {});
    await browser.close().catch(() => {});
  }
}

main().catch((error) => {
  console.error(`Process/API attention Radar browser E2E: FAIL: ${error?.stack || error}`);
  process.exit(1);
});
