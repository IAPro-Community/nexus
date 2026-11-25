#!/usr/bin/env node
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  launchFixtureBrowser,
  meaningfulFixtureRequestFailures,
  seedFixtureProject,
  startFixtureServer,
  terminateProcess,
} from './e2e-fixture.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..', '..');
const outDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR ||
  path.resolve(repoRoot, 'DEV', 'validation', 'nex-2', '8d1f1592', 'p2-presence');
if (!existsSync(outDir)) mkdirSync(outDir, { recursive: true });

async function main() {
  console.log('=== Nex 2.0 P2 Presence 100-Cycle E2E ===');
  const { server, bootstrapUrl, isolatedProjectPath, getOutput } = await startFixtureServer();

  let browser;
  let context;
  let page;
  let collectors;
  const requestedUrls = [];
  try {
    ({ browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl));
    const { apiOrigin } = await seedFixtureProject(context, bootstrapUrl, isolatedProjectPath);
    page.on('request', (req) => requestedUrls.push(req.url()));

    await page.setViewportSize({ width: 520, height: 640 });
    await page.goto(`${apiOrigin}/nex-v2.html`, {
      waitUntil: 'networkidle',
      timeout: 15000,
    });

    await page.waitForFunction(
      () => Boolean(window.__nexV2PresenceMetrics) && window.__nexV2Metrics?.rigReady === true,
      null,
      { timeout: 10000 },
    );

    const initialPanelChunks = requestedUrls.filter((u) =>
      /nex-v2-chunks\/NexInteractionPanel-[A-Za-z0-9]+\.js/.test(u),
    );
    assert.equal(
      initialPanelChunks.length,
      0,
      `Lazy panel chunk must NOT load initially; got: ${initialPanelChunks.join(', ')}`,
    );
    assert.equal(await page.locator('[data-nex-panel]').count(), 0);
    await page.screenshot({ path: path.join(outDir, 'p2-presence-initial.png') });

    const avatarTrigger = page.locator('[data-nex-ready] button:not([data-wails-draggable])');
    const avatarBoxInitial = await page.locator('[data-nex-ready]').boundingBox();
    assert.ok(avatarBoxInitial, 'Avatar container missing');
    assert.equal(Math.round(avatarBoxInitial.width), 160);
    assert.equal(Math.round(avatarBoxInitial.height), 160);

    await avatarTrigger.focus();
    await avatarTrigger.click();
    await page.waitForSelector('[data-nex-panel]', { timeout: 5000 });
    await page.waitForFunction(() => window.__nexV2PresenceMetrics?.chunkLoaded === true, null, {
      timeout: 5000,
    });

    const postOpenPanelChunks = requestedUrls.filter((u) =>
      /nex-v2-chunks\/NexInteractionPanel-[A-Za-z0-9]+\.js/.test(u),
    );
    assert.ok(postOpenPanelChunks.length >= 1, 'Expected lazy chunk load upon panel open');

    const tabs = page.locator('[role="tab"]');
    assert.equal(await tabs.count(), 3, 'Panel must contain strictly 3 main tabs');
    assert.equal(await page.locator('[data-nex-panel-settings]').count(), 1);
    await page.screenshot({ path: path.join(outDir, 'p2-presence-panel-open.png') });

    // Keyboard tab navigation & Escape focus restoration
    await page.locator('[data-nex-tab="chat"]').focus();
    await page.keyboard.press('ArrowRight');
    assert.equal(
      await page.locator('[data-nex-panel]').getAttribute('data-active-tab'),
      'resources',
    );
    await page.keyboard.press('ArrowRight');
    assert.equal(
      await page.locator('[data-nex-panel]').getAttribute('data-active-tab'),
      'attention',
    );
    await page.keyboard.press('ArrowLeft');
    assert.equal(
      await page.locator('[data-nex-panel]').getAttribute('data-active-tab'),
      'resources',
    );

    await page.keyboard.press('Escape');
    await page.waitForSelector('[data-nex-panel]', { state: 'detached', timeout: 5000 });
    await page.waitForTimeout(30);
    const restoredFocus = await page.evaluate(
      () => document.activeElement?.getAttribute('aria-label') || '',
    );
    assert.ok(
      restoredFocus.includes('Nex'),
      `Expected focus restored to trigger, got: ${restoredFocus}`,
    );

    // Warm reopen
    await avatarTrigger.click();
    await page.waitForSelector('[data-nex-panel]', { timeout: 5000 });
    const afterWarmMetrics = await page.evaluate(() => window.__nexV2PresenceMetrics);
    assert.ok(
      typeof afterWarmMetrics.panelOpenColdMs === 'number' &&
        afterWarmMetrics.panelOpenColdMs <= 1500,
      `Cold panel open exceeded 1500ms: ${afterWarmMetrics.panelOpenColdMs}`,
    );
    assert.ok(
      typeof afterWarmMetrics.panelOpenWarmMs === 'number' &&
        afterWarmMetrics.panelOpenWarmMs <= 500,
      `Warm panel open exceeded 500ms: ${afterWarmMetrics.panelOpenWarmMs}`,
    );

    // Avatar must remain fixed 160x160 even with panel open
    const avatarBoxWithPanel = await page.locator('[data-nex-ready]').boundingBox();
    assert.equal(Math.round(avatarBoxWithPanel.width), 160);
    assert.equal(Math.round(avatarBoxWithPanel.height), 160);
    await page.keyboard.press('Escape');
    await page.waitForSelector('[data-nex-panel]', { state: 'detached', timeout: 5000 });

    // Execute 100 stability cycles covering panel open/close, tabs, settings, hide avatar, compact bar, DND
    for (let i = 0; i < 100; i += 1) {
      // Alternate opening via compact bar when hidden or avatar when visible
      const openBtn =
        (await page.locator('[data-nex-compact-open]').count()) > 0
          ? page.locator('[data-nex-compact-open]')
          : avatarTrigger;
      await openBtn.click();
      await page.waitForSelector('[data-nex-panel]', { timeout: 3000 });

      const tabSequence = ['chat', 'resources', 'attention'];
      const targetTab = tabSequence[i % tabSequence.length];
      await page.locator(`[data-nex-tab="${targetTab}"]`).click();

      if (i % 10 === 0) {
        await page.locator('[data-nex-panel-settings]').click();
        await page.waitForSelector('[data-nex-settings]', { timeout: 3000 });
        await page.locator('[data-nex-toggle="dnd"]').click();
        assert.equal(await page.locator('aside[role="status"]').count(), 0);
        await page.locator('[data-nex-toggle="compact-bar"]').click();
        await page.locator('[data-nex-toggle="show-avatar"]').click();
        await page.locator('[data-nex-settings-close]').click();
      } else if (i % 10 === 5) {
        // Restore avatar before next half of block
        await page.locator('[data-nex-panel-settings]').click();
        await page.locator('[data-nex-toggle="show-avatar"]').click();
        await page.locator('[data-nex-toggle="dnd"]').click();
        await page.locator('[data-nex-settings-close]').click();
      }

      if (await page.locator('[data-nex-ready]').count()) {
        const box = await page.locator('[data-nex-ready]').boundingBox();
        assert.equal(Math.round(box.width), 160, `Avatar width drifted on cycle ${i}`);
        assert.equal(Math.round(box.height), 160, `Avatar height drifted on cycle ${i}`);
      }

      await page.locator('[data-nex-panel-close]').click();
      await page.waitForSelector('[data-nex-panel]', { state: 'detached', timeout: 3000 });
    }

    const finalMetrics = await page.evaluate(() => window.__nexV2PresenceMetrics);
    assert.ok(
      finalMetrics.panelToggleCount >= 204,
      `Unexpected toggle count: ${finalMetrics.panelToggleCount}`,
    );
    await page.screenshot({ path: path.join(outDir, 'p2-presence-100-cycles.png') });

    const requestFailures = meaningfulFixtureRequestFailures(collectors.requestFailures);
    const cyclePageErrors = collectors.pageErrors.length;
    const cycleConsoleErrors = collectors.consoleErrors.length;
    assert.equal(cyclePageErrors, 0, `Page errors: ${collectors.pageErrors.join('; ')}`);
    assert.equal(cycleConsoleErrors, 0, `Console errors: ${collectors.consoleErrors.join('; ')}`);
    assert.equal(requestFailures.length, 0, `Request failures: ${requestFailures.join('; ')}`);

    // Verify honest Core-unreachable offline notice when HUD returns 503
    await page.route('**/api/v1/hud', (route) =>
      route.fulfill({ status: 503, contentType: 'application/json', body: '{"error":"offline"}' }),
    );
    await page.reload({ waitUntil: 'networkidle', timeout: 15000 });
    await page.waitForSelector('[data-nex-offline-banner]', { timeout: 5000 });
    const offlineMetrics = await page.evaluate(() => window.__nexV2PresenceMetrics);
    assert.equal(offlineMetrics.offline, true);
    assert.equal(offlineMetrics.mood, 'offline');
    await page.screenshot({ path: path.join(outDir, 'p2-presence-offline.png') });

    const summary = {
      initialPanelChunks: initialPanelChunks.length,
      postOpenPanelChunks: postOpenPanelChunks.length,
      panelOpenColdMs: finalMetrics.panelOpenColdMs,
      panelOpenWarmMs: finalMetrics.panelOpenWarmMs,
      cyclesCompleted: 100,
      panelToggleCount: finalMetrics.panelToggleCount,
      pageErrors: cyclePageErrors,
      consoleErrors: cycleConsoleErrors,
      requestFailures: requestFailures.length,
      offlineBannerVerified: true,
    };
    writeFileSync(path.join(outDir, 'summary.json'), `${JSON.stringify(summary, null, 2)}\n`);
    console.log('Nex 2.0 P2 Presence 100-Cycle E2E: PASS', JSON.stringify(summary));
  } catch (error) {
    console.error('Nex 2.0 P2 Presence 100-Cycle E2E: FAIL:', error);
    console.error('Server output:\n' + getOutput());
    process.exitCode = 1;
  } finally {
    if (context) await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
    await terminateProcess(server);
  }
}

await main();
