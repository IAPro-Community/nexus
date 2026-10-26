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
    await context.route('**/api/v1/attention**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          needs_you: [
            {
              mission_id: 'fixture-mission',
              project_id: project.id,
              state: 'BLOCKED_NEEDS_USER',
              level: 'REQUIRE_USER',
              reason_code: 'INTERVENTION_REQUIRED',
              summary: 'Fixture mission is waiting for a decision',
              question: 'Choose the next step',
              impact: 'Execution is paused',
              recommended_actions: ['Review mission'],
              age_seconds: 1,
            },
          ],
          completed: [],
          failed: [],
          all_items: [],
          total_needs: 1,
        }),
      }),
    );

    // Web HUD remains opt-in even though native HUD needs the projection API
    // available by default. Opt in before the project shell boots.
    await page.addInitScript(() => {
      window.localStorage.setItem(
        'iapro:nexus:hud:presentation:v1',
        JSON.stringify({ version: 1, enabled: true }),
      );
    });

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

    // Compact capsule must remain informative without becoming a dashboard.
    const expand = page.getByRole('button', { name: /expandir nexus hud/i });
    assert.equal(await expand.getAttribute('aria-expanded'), 'false');
    await expand.hover();
    await page.getByRole('tooltip').waitFor({ state: 'visible', timeout: 3000 });

    // The same capsule opens the contextual, accessible provider popover.
    await expand.click();
    await page.locator('#nexus-hud-popover').waitFor({ state: 'visible' });
    assert.equal(await page.getByRole('button', { name: /recolher nexus hud/i }).count() > 0, true);
    const expanded = page.locator('section[data-hud-panel][data-expanded="true"]').first();
    await expanded.waitFor({ state: 'visible', timeout: 5000 });
    const attentionList = page.locator('[data-mascot-attention-list]');
    await attentionList.waitFor({ state: 'visible', timeout: 5000 });
    assert.match(
      await attentionList.innerText(),
      /Uma missão precisa da sua atenção/,
      'Mascot attention must use the canonical Attention Center fixture',
    );
    assert.equal(await attentionList.getByRole('button', { name: 'Abrir' }).count(), 1);
    assert.equal(await attentionList.getByRole('button', { name: 'Adiar' }).count(), 1);
    await attentionList.screenshot({ path: path.join(screenshotDir, 'web-hud-attention.png') });
    await attentionList.getByRole('button', { name: 'Adiar' }).click();
    assert.equal(
      await page.evaluate(() =>
        window.localStorage
          .getItem('iapro:nexus:companion:snooze:v1')
          ?.includes('fixture-mission'),
      ),
      true,
      'Mascot snooze must be persisted without mutating canonical Attention Center state',
    );
    await page.reload({ waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('.nx-os-shell', { timeout: 10000 });
    await page.locator('aside[aria-label="Nexus HUD"]').waitFor({ state: 'visible' });
    await page.getByRole('button', { name: /expandir nexus hud/i }).click();
    await page.locator('#nexus-hud-popover').waitFor({ state: 'visible' });
    assert.equal(
      await page.locator('[data-mascot-attention-list]').count(),
      1,
      'Snoozing the mascot notification must not hide the canonical Attention Center item',
    );

    // Exercise compiled detached-popup styles without claiming native Wails
    // focus, transparency or mouse hit-testing has been certified.
    await hud.evaluate((element) => {
      element.dataset.detachedView = 'popover';
    });
    const entrance = await page.locator('#nexus-hud-popover').evaluate((element) => {
      const animation = getComputedStyle(element);
      return { name: animation.animationName, duration: animation.animationDuration };
    });
    assert.notEqual(entrance.name, 'none', 'detached popup should animate when motion is allowed');
    assert.equal(entrance.duration, '0.15s', 'popup entrance should be short');

    await page.emulateMedia({ reducedMotion: 'reduce' });
    const reduced = await page.locator('#nexus-hud-popover').evaluate((element) => {
      return getComputedStyle(element).animationName;
    });
    assert.equal(reduced, 'none', 'reduced-motion setting must disable popup animation');
    await page.emulateMedia({ reducedMotion: 'no-preference' });
    await hud.evaluate((element) => {
      delete element.dataset.detachedView;
    });

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
