#!/usr/bin/env node
// Browser E2E for AgentTerminal follow/lease consent. The API and xterm are
// real Nexus code; only the provider WebSocket is controlled by the harness.
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
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
  process.env.NEXUS_E2E_ARTIFACTS_DIR || path.join(repoRoot, '.tempmediaStorage/follow-agent');
if (!existsSync(artifacts)) mkdirSync(artifacts, { recursive: true });

async function main() {
  console.log('=== Agent terminal follow and lease consent E2E ===');
  const { server, bootstrapUrl, isolatedDataDir, isolatedProjectPath, getOutput } =
    await startFixtureServer();
  let browser;
  let context;
  let page;
  let collectors;
  const sockets = [];
  const clientFrames = [];
  try {
    ({ browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl));
    await page.setViewportSize({ width: 1440, height: 900 });
    const { project, apiOrigin } = await seedFixtureProject(
      context,
      bootstrapUrl,
      isolatedProjectPath,
    );
    const sessionResponse = await context.request.get(`${apiOrigin}/api/v1/session`);
    assert.equal(sessionResponse.status(), 200, 'fixture session must be authenticated');
    const session = await sessionResponse.json();
    const createAgent = await context.request.post(
      `${apiOrigin}/api/v1/projects/${project.id}/agents`,
      {
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token },
        data: { name: 'Follow Agent E2E', role: 'developer' },
      },
    );
    assert.equal(createAgent.status(), 201, 'Agent must be created through the real API');
    const agent = await createAgent.json();
    const terminalPath = `/api/v1/agents/${agent.id}/terminal`;

    await page.routeWebSocket(
      (url) => url.pathname === terminalPath,
      (socket) => {
        sockets.push(socket);
        socket.onMessage((message) => {
          try {
            clientFrames.push(JSON.parse(String(message)));
          } catch {
            clientFrames.push({ type: 'invalid-frame' });
          }
          const latest = clientFrames[clientFrames.length - 1];
          if (latest?.type === 'lease_acquire') {
            socket.send(JSON.stringify({ type: 'lease', role: 'CONTROL' }));
          }
        });
        setTimeout(() => {
          socket.send(JSON.stringify({ type: 'lease', role: 'VIEW_ONLY' }));
          socket.send(
            JSON.stringify({
              type: 'output',
              data: Array.from({ length: 70 }, (_, index) => `FOLLOW_E2E_BASE_${index}\r\n`).join(
                '',
              ),
            }),
          );
        }, 20);
      },
    );

    await page.goto(`${apiOrigin}/p/${encodeURIComponent(project.id)}/agents/${agent.id}`, {
      waitUntil: 'domcontentloaded',
      timeout: 15000,
    });
    await page.locator('.nx-agent-terminal__xterm .xterm').waitFor({
      state: 'visible',
      timeout: 15000,
    });
    await page.getByRole('button', { name: /Take control|Assumir controle/i }).waitFor({
      state: 'visible',
      timeout: 10000,
    });
    assert.equal(sockets.length, 1, 'Agent terminal must attach one WebSocket');
    assert.equal(
      clientFrames.some((frame) => frame.type === 'lease_acquire'),
      false,
      'WebSocket attach must not request input lease',
    );

    const follow = page.locator('[data-tour="agent-terminal-follow"]');
    await follow.waitFor({ state: 'visible', timeout: 5000 });
    await page.waitForFunction(() =>
      document.querySelector('.xterm-rows')?.textContent?.includes('FOLLOW_E2E_BASE_69'),
    );
    await follow.click();
    assert.equal(await follow.getAttribute('aria-pressed'), 'false');
    const viewport = page.locator('.xterm-viewport');
    const before = await viewport.evaluate((element) => ({
      top: element.scrollTop,
      max: element.scrollHeight - element.clientHeight,
    }));
    assert.ok(
      before.max > 0,
      `fixture output must make scrollback measurable: ${JSON.stringify(before)}`,
    );

    sockets[0].send(
      JSON.stringify({
        type: 'output',
        data: Array.from({ length: 25 }, (_, index) => `FOLLOW_E2E_PAUSED_${index}\r\n`).join(''),
      }),
    );
    await page.waitForFunction(
      () => (document.querySelector('.xterm-viewport')?.scrollHeight || 0) > 0,
    );
    await page.waitForTimeout(150);
    const paused = await viewport.evaluate((element) => ({
      top: element.scrollTop,
      max: element.scrollHeight - element.clientHeight,
    }));
    assert.ok(
      Math.abs(paused.top - before.top) <= 2,
      `paused view must stay at its prior scroll position: ${JSON.stringify({ before, paused })}`,
    );

    await follow.click();
    assert.equal(await follow.getAttribute('aria-pressed'), 'true');
    await page.waitForFunction(() => {
      const element = document.querySelector('.xterm-viewport');
      return Boolean(
        element && element.scrollTop >= element.scrollHeight - element.clientHeight - 2,
      );
    });

    await page.locator('.nx-agent-terminal__xterm .xterm-screen').click();
    await page.keyboard.type('z');
    await page.waitForTimeout(100);
    assert.equal(
      clientFrames.some((frame) => frame.type === 'input'),
      false,
      'typing in VIEW_ONLY must not forward PTY input',
    );
    assert.equal(
      clientFrames.some((frame) => frame.type === 'lease_acquire'),
      false,
      'typing in VIEW_ONLY must not acquire an input lease',
    );

    await page.getByRole('button', { name: /Take control|Assumir controle/i }).click();
    await page.waitForFunction(() => !document.querySelector('[data-role="VIEW_ONLY"]'));
    assert.equal(
      clientFrames.filter((frame) => frame.type === 'lease_acquire').length,
      1,
      'the explicit control button must request one lease',
    );
    await page.locator('.nx-agent-terminal__xterm .xterm-screen').click();
    await page.keyboard.type('q');
    await page.waitForTimeout(100);
    assert.equal(
      clientFrames.some((frame) => frame.type === 'input' && frame.data === 'q'),
      true,
      'input must be forwarded only after the explicit lease is granted',
    );

    sockets[0].send(JSON.stringify({ type: 'lease', role: 'VIEW_ONLY' }));
    await page.locator('[data-role="VIEW_ONLY"]').waitFor({ state: 'visible', timeout: 5000 });
    await sockets[0].close({ code: 1001, reason: 'exercise reconnect' });
    await page.waitForTimeout(700);
    await page.waitForFunction(() => document.querySelectorAll('.xterm').length > 0);
    await page.waitForTimeout(400);
    assert.equal(sockets.length, 2, 'disconnected terminal must reconnect once');
    assert.equal(
      clientFrames.filter((frame) => frame.type === 'lease_acquire').length,
      1,
      'reconnect must not silently reacquire the input lease',
    );

    await page.screenshot({ path: path.join(artifacts, 'agent-terminal-follow.png') });
    console.log('PASS attach is VIEW_ONLY; no lease is requested automatically');
    console.log('PASS explicit pause preserves scroll position and resume follows live output');
    console.log('PASS keyboard input requires explicit lease; revoke/reconnect remains VIEW_ONLY');
    assert.equal(collectors.pageErrors.length, 0, collectors.pageErrors.join(' | '));
    assert.deepEqual(meaningfulFixtureRequestFailures(collectors.requestFailures), []);
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
      path.join(artifacts, 'agent-terminal-follow-failure.json'),
      `${JSON.stringify({ url: publicUrl, error: redact(error?.stack || error), server: redact(getOutput()) }, null, 2)}\n`,
    );
    if (page)
      await page
        .screenshot({ path: path.join(artifacts, 'agent-terminal-follow-failure.png') })
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
  console.error(`Agent terminal follow E2E: FAIL: ${error?.stack || error}`);
  process.exit(1);
});
