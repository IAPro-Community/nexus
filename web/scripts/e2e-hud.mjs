#!/usr/bin/env node
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import AxeBuilder from '@axe-core/playwright';
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
const screenshotDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR || path.resolve(repoRoot, '.tempmediaStorage', 'hud');
if (!existsSync(screenshotDir)) mkdirSync(screenshotDir, { recursive: true });

async function main() {
  console.log('=== Nexus HUD Web E2E ===');
  process.env.NEXUS_ENABLE_HUD = '1';
  const { server, bootstrapUrl, isolatedProjectPath, isolatedDataDir, getOutput } =
    await startFixtureServer();

  let browser;
  let context;
  let page;
  let collectors;
  try {
    ({ browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl));
    const { project, apiOrigin } = await seedFixtureProject(
      context,
      bootstrapUrl,
      isolatedProjectPath,
    );

    const shellUrl = new URL(`/p/${encodeURIComponent(project.id)}/overview`, apiOrigin);
    await page.goto(shellUrl.toString(), { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('.nx-os-shell', { timeout: 10000 });

    const hud = page.locator('aside[aria-label="Nexus HUD"]').first();
    await hud.waitFor({ state: 'visible', timeout: 10000 });
    assert.equal(await hud.isVisible(), true, 'HUD must be visible when experimental flag is enabled');

    const apiResponse = await context.request.get(
      `${apiOrigin}/api/v1/hud?project_id=${encodeURIComponent(project.id)}`,
    );
    assert.equal(apiResponse.status(), 200, 'HUD projection endpoint must be available');
    const envelope = await apiResponse.json();
    assert.equal(envelope.enabled, true);
    assert.equal(envelope.rollout, 'experimental');
    assert.ok(envelope.state, 'HUD response must contain projection state');
    assert.ok(Array.isArray(envelope.state.providers), 'providers must be an array, including zero-state');

    const nodeBefore = await page.evaluate(() => {
      const hudNode = document.querySelector('aside[aria-label="Nexus HUD"]');
      window.__hudE2ENode = hudNode;
      return Boolean(hudNode);
    });
    assert.equal(nodeBefore, true, 'HUD DOM node must be captured');

    // The HUD lives above individual Workspace surfaces and must survive navigation.
    const terminals = page.locator('.nx-workspace-tab[data-kind="terminals"]').first();
    await terminals.waitFor({ state: 'visible', timeout: 10000 });
    await terminals.click();
    await page.waitForFunction(
      () =>
        document
          .querySelector('.nx-workspace-tab[data-kind="terminals"]')
          ?.getAttribute('aria-selected') === 'true',
    );
    const sameNode = await page.evaluate(
      () => window.__hudE2ENode === document.querySelector('aside[aria-label="Nexus HUD"]'),
    );
    assert.equal(sameNode, true, 'HUD must not remount when Workspace surface changes');

    // Desktop capture.
    await page.setViewportSize({ width: 1440, height: 900 });
    await hud.waitFor({ state: 'visible' });
    await page.screenshot({
      path: path.join(screenshotDir, 'web-hud-desktop.png'),
      fullPage: true,
    });

    // Expanded state and accessibility.
    const expand = page.getByRole('button', { name: /expandir nexus hud/i });
    await expand.click();
    const expanded = page.locator('section[data-hud-panel][data-expanded="true"]').first();
    await expanded.waitFor({ state: 'visible', timeout: 5000 });

    const axeDesktop = await new AxeBuilder({ page }).exclude('.xterm').analyze();
    const seriousDesktop = axeDesktop.violations.filter((v) =>
      ['serious', 'critical'].includes(v.impact),
    );
    assert.equal(
      seriousDesktop.length,
      0,
      `HUD desktop Axe serious/critical: ${seriousDesktop.map((v) => v.id).join(', ')}`,
    );
    await page.screenshot({
      path: path.join(screenshotDir, 'web-hud-expanded.png'),
      fullPage: true,
    });

    // Mobile is a condensed viewport treatment, not a scaled desktop dialog.
    await page.setViewportSize({ width: 390, height: 844 });
    await hud.waitFor({ state: 'visible' });
    const hudBox = await hud.boundingBox();
    assert.ok(hudBox, 'mobile HUD must have geometry');
    assert.ok(hudBox.width <= 390, `mobile HUD width must fit viewport: ${hudBox.width}`);
    assert.ok(hudBox.height < 844, `mobile HUD must not consume the whole viewport: ${hudBox.height}`);

    const axeMobile = await new AxeBuilder({ page }).exclude('.xterm').analyze();
    const seriousMobile = axeMobile.violations.filter((v) =>
      ['serious', 'critical'].includes(v.impact),
    );
    assert.equal(
      seriousMobile.length,
      0,
      `HUD mobile Axe serious/critical: ${seriousMobile.map((v) => v.id).join(', ')}`,
    );
    await page.screenshot({
      path: path.join(screenshotDir, 'web-hud-mobile.png'),
      fullPage: true,
    });

    const meaningfulFailures = meaningfulFixtureRequestFailures(collectors.requestFailures);
    assert.equal(
      meaningfulFailures.length,
      0,
      `HUD request failures: ${meaningfulFailures.join(' | ')}`,
    );
    assert.equal(
      collectors.pageErrors.length,
      0,
      `HUD page errors: ${collectors.pageErrors.join(' | ')}`,
    );

    console.log('Nexus HUD Web E2E: PASS');
  } catch (error) {
    console.error(`Nexus HUD Web E2E server output:\n${getOutput()}`);
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
  console.error(`Nexus HUD Web E2E: FAIL: ${error?.stack || error}`);
  process.exit(1);
});
