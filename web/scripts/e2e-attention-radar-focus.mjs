#!/usr/bin/env node
// Browser E2E: the global attention radar must switch to the selected project
// and open the exact runtime represented by the clicked row.
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import {
  launchFixtureBrowser,
  meaningfulFixtureRequestFailures,
  seedFixtureProject,
  startFixtureServer,
  terminateProcess,
} from './e2e-fixture.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '..', '..');
const artifacts =
  process.env.NEXUS_E2E_ARTIFACTS_DIR || path.join(repoRoot, '.tempmediaStorage/attention-radar');
if (!existsSync(artifacts)) mkdirSync(artifacts, { recursive: true });

async function createProject(context, apiOrigin, csrfToken, name, projectPath) {
  mkdirSync(projectPath, { recursive: true });
  const init = spawnSync('git', ['init', '-q', projectPath], { encoding: 'utf8' });
  if (init.status !== 0) throw new Error(`fixture git init failed: ${init.stderr || ''}`);
  const response = await context.request.post(`${apiOrigin}/api/v1/projects`, {
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    data: { name, path: projectPath },
  });
  assert.equal(response.status(), 201, `project ${name} must be created through the real API`);
  return response.json();
}

async function startShell(context, apiOrigin, csrfToken, project) {
  const response = await context.request.post(`${apiOrigin}/api/v1/projects/${project.id}/shell`, {
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    data: {},
  });
  assert.equal(response.status(), 201, `Project Shell for ${project.name} must start`);
  const payload = await response.json();
  assert.ok(payload.runtime?.runtime_id, `Project Shell ${project.name} needs a runtime ID`);
  return payload.runtime.runtime_id;
}

async function main() {
  console.log('=== Global attention radar runtime focus E2E ===');
  const { server, bootstrapUrl, isolatedDataDir, isolatedProjectPath, getOutput } =
    await startFixtureServer();
  let browser;
  let context;
  let page;
  let collectors;
  const sockets = [];
  try {
    ({ browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl));
    await page.setViewportSize({ width: 1440, height: 900 });
    const { project: alphaProject, apiOrigin } = await seedFixtureProject(
      context,
      bootstrapUrl,
      isolatedProjectPath,
    );
    const sessionResponse = await context.request.get(`${apiOrigin}/api/v1/session`);
    assert.equal(sessionResponse.status(), 200, 'fixture session must be authenticated');
    const session = await sessionResponse.json();
    const betaProject = await createProject(
      context,
      apiOrigin,
      session.csrf_token,
      'Attention Radar Beta',
      path.join(isolatedDataDir, 'project-beta'),
    );
    const alphaRuntimeId = await startShell(context, apiOrigin, session.csrf_token, alphaProject);
    const betaRuntimeId = await startShell(context, apiOrigin, session.csrf_token, betaProject);
    assert.notEqual(alphaRuntimeId, betaRuntimeId, 'projects must own independent runtimes');

    // The runtime list, project shells, API auth and shell endpoints are real.
    // Only attention state is controlled so the test never needs an AI provider.
    await context.route('**/api/v1/runtimes', async (route) => {
      const response = await route.fetch();
      const runtimes = await response.json();
      for (const runtime of runtimes) {
        if (![alphaRuntimeId, betaRuntimeId].includes(runtime.runtime_id)) continue;
        Object.assign(runtime, {
          state: 'WAITING',
          attention_reason: 'QUESTION',
          attention_kind: 'needs_user',
          prompt_kind: 'native_status',
          attention_context: 'Approve the same safe operation?',
          attention_fingerprint: 'shared-radar-fingerprint',
          attention_interaction_id: 'same-provider-request-id',
          attention_revision: 1,
          attention_generation: 'fixture-generation',
          attention_source: 'AUTHORITATIVE',
        });
      }
      await route.fulfill({ response, body: JSON.stringify(runtimes) });
    });

    await page.routeWebSocket(
      (url) =>
        url.pathname === `/api/v1/runtimes/${alphaRuntimeId}/terminal` ||
        url.pathname === `/api/v1/runtimes/${betaRuntimeId}/terminal`,
      (socket) => {
        sockets.push(socket);
        setTimeout(() => {
          socket.send(JSON.stringify({ type: 'lease', role: 'VIEW_ONLY' }));
          socket.send(JSON.stringify({ type: 'output', data: 'RADAR_FOCUS_FIXTURE\r\n' }));
        }, 20);
      },
    );

    await page.goto(`${apiOrigin}/p/${encodeURIComponent(alphaProject.id)}`, {
      waitUntil: 'domcontentloaded',
      timeout: 15000,
    });
    const notifications = page.getByRole('button', {
      name: /Central de Notificações e Radar|Notifications and Radar/i,
    });
    await notifications.waitFor({ state: 'visible', timeout: 15000 });

    const openRadar = async () => {
      await notifications.click();
      const radarTab = page.getByRole('tab', { name: /radar/i });
      await radarTab.waitFor({ state: 'visible', timeout: 5000 });
      assert.match(await radarTab.innerText(), /Radar de Atenção/);
      await radarTab.click();
    };
    await openRadar();
    const alphaRow = page.locator(`button[data-runtime-id="${alphaRuntimeId}"]`);
    const betaRow = page.locator(`button[data-runtime-id="${betaRuntimeId}"]`);
    await alphaRow.waitFor({ state: 'visible', timeout: 10000 });
    await betaRow.waitFor({ state: 'visible', timeout: 10000 });
    assert.equal(await alphaRow.innerText(), await betaRow.innerText());
    assert.match(await betaRow.innerText(), /precisa de você/i);
    assert.equal(await page.locator('button[data-runtime-id]').count(), 2);
    await page.screenshot({ path: path.join(artifacts, 'attention-radar-two-runtimes.png') });

    await betaRow.click();
    const betaSurface = page.locator(
      `.nx-project-shell-surface[data-runtime-id="${betaRuntimeId}"]`,
    );
    await betaSurface.waitFor({ state: 'attached', timeout: 15000 });
    assert.ok(
      page.url().includes(encodeURIComponent(betaProject.id)),
      'focus must switch projects',
    );
    await betaSurface.locator('.xterm').waitFor({ state: 'attached', timeout: 10000 });
    assert.equal(sockets.length, 1, 'focusing Beta must attach only Beta terminal');
    await openRadar();
    await page.locator(`button[data-runtime-id="${alphaRuntimeId}"]`).click();
    const alphaSurface = page.locator(
      `.nx-project-shell-surface[data-runtime-id="${alphaRuntimeId}"]`,
    );
    await alphaSurface.waitFor({ state: 'attached', timeout: 15000 });
    assert.ok(
      page.url().includes(encodeURIComponent(alphaProject.id)),
      'focus must return to Alpha',
    );
    await alphaSurface.locator('.xterm').waitFor({ state: 'attached', timeout: 10000 });
    assert.equal(sockets.length, 2, 'focusing Alpha must attach its own terminal');
    await page.screenshot({ path: path.join(artifacts, 'attention-radar-focused-alpha.png') });

    assert.equal(collectors.pageErrors.length, 0, collectors.pageErrors.join(' | '));
    assert.deepEqual(meaningfulFixtureRequestFailures(collectors.requestFailures), []);
    console.log('PASS identical questions remain actionable by distinct runtime IDs');
    console.log('PASS cross-project Radar focus opens exactly the selected Project Shell');
    console.log(
      'PASS no provider calls; API, project shell creation and focus routing use Nexus code',
    );
  } catch (error) {
    const publicUrl = (() => {
      try {
        const url = new URL(page?.url() || bootstrapUrl);
        url.hash = '';
        return url.toString();
      } catch {
        return '(unparsable url)';
      }
    })();
    const redact = (value) =>
      String(value).replace(/nexus_bootstrap=[a-f0-9]+/gi, 'nexus_bootstrap=[REDACTED]');
    writeFileSync(
      path.join(artifacts, 'attention-radar-failure.json'),
      `${JSON.stringify({ url: publicUrl, error: redact(error?.stack || error), server: redact(getOutput()) }, null, 2)}\n`,
    );
    if (page)
      await page
        .screenshot({ path: path.join(artifacts, 'attention-radar-failure.png') })
        .catch(() => {});
    throw error;
  } finally {
    if (context) await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
    await terminateProcess(server);
    if (!process.env.NEXUS_E2E_DATA_DIR && isolatedDataDir) {
      rmSync(isolatedDataDir, { recursive: true, force: true });
    }
  }
}

main().catch((error) => {
  console.error(`Global attention radar E2E: FAIL: ${error?.stack || error}`);
  process.exit(1);
});
