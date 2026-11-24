#!/usr/bin/env node
// Device preview E2E: cycles the viewport presets (phone -> tablet -> desktop)
// inside a single browser session via real POST /action calls, asserts the
// scale change is visible on the same surface, and runs Axe (WCAG 2 AA) on
// each preset with keyboard focus retained on the toggle button.
import assert from 'node:assert/strict';
import { existsSync, mkdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import AxeBuilder from '@axe-core/playwright';
import {
  launchFixtureBrowser,
  seedFixtureProject,
  startFixtureServer,
  terminateProcess,
} from './e2e-fixture.mjs';

const filename = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(filename), '..', '..');
const artifactDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR || path.join(repoRoot, '.tempmediaStorage', 'device-preview');
if (!existsSync(artifactDir)) mkdirSync(artifactDir, { recursive: true });

const SESSION_ID = '0123456789abcdef0123456789abcdef';
const PRESETS = [
  { id: 'phone', width: 390, height: 844, label: 'Celular (390x844)' },
  { id: 'tablet', width: 820, height: 1180, label: 'Tablet (820x1180)' },
  { id: 'desktop', width: 1280, height: 800, label: 'Desktop (1280x800)' },
];
const jpeg =
  '/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAP//////////////////////////////////////////////////////////////////////////////////////2wBDAf//////////////////////////////////////////////////////////////////////////////////////wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAb/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIQAxAAAAH/AP/EABQRAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQEAAT8Af//EABQRAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQMBAT8Af//EABQRAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQIBAT8Af//Z';

async function main() {
  console.log('=== Device preview E2E (single session, Axe per preset) ===');
  const { server, bootstrapUrl, isolatedProjectPath, getOutput } = await startFixtureServer();
  let browser;
  let context;
  let page;
  const viewportActions = [];
  const runtimeErrors = [];
  let sessionWidth = 1280;
  let sessionHeight = 800;

  try {
    ({ browser, context, page } = await launchFixtureBrowser(bootstrapUrl));
    page.on('console', (message) => {
      if (message.type() === 'error') runtimeErrors.push(message.text());
    });
    page.on('pageerror', (error) => runtimeErrors.push(error.message));
    const { project, apiOrigin } = await seedFixtureProject(
      context,
      bootstrapUrl,
      isolatedProjectPath,
    );

    // Fake session store: POST /action with type=viewport mutates the state,
    // mirroring browsercontrol's server-side behavior.
    await context.route('**/api/v1/browser/sessions**', async (route) => {
      const url = new URL(route.request().url());
      const method = route.request().method();
      const parts = url.pathname.split('/').filter(Boolean);
      if (method === 'GET' && parts.length === 4) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: SESSION_ID,
              project_id: project.id,
              url: 'http://localhost:3000',
              title: 'Fixture App',
              width: sessionWidth,
              height: sessionHeight,
            },
          ]),
        });
      }
      if (method === 'POST' && parts.length === 4) {
        sessionWidth = 1280;
        sessionHeight = 800;
        return route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: SESSION_ID,
            project_id: project.id,
            url: 'http://localhost:3000',
            title: 'Fixture App',
            width: sessionWidth,
            height: sessionHeight,
          }),
        });
      }
      if (parts[5] === 'action' && method === 'POST') {
        const body = route.request().postDataJSON();
        if (body.type === 'viewport') {
          viewportActions.push({ width: body.width, height: body.height });
          assert.ok(body.width >= 320 && body.width <= 1920, 'width in server range');
          assert.ok(body.height >= 240 && body.height <= 1200, 'height in server range');
          sessionWidth = body.width;
          sessionHeight = body.height;
        }
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: SESSION_ID,
            project_id: project.id,
            url: 'http://localhost:3000',
            title: 'Fixture App',
            width: sessionWidth,
            height: sessionHeight,
          }),
        });
      }
      if (parts[5] === 'frame' && method === 'GET') {
        return route.fulfill({
          status: 200,
          contentType: 'image/jpeg',
          body: Buffer.from(jpeg, 'base64'),
        });
      }
      return route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });
    await page.routeWebSocket(
      (url) => url.pathname === `/api/v1/browser/sessions/${SESSION_ID}/stream`,
      () => {},
    );

    await page.goto(
      new URL(`/p/${encodeURIComponent(project.id)}/overview`, apiOrigin).toString(),
      { waitUntil: 'domcontentloaded' },
    );
    await page.locator('.nx-os-shell').waitFor({ state: 'visible', timeout: 10000 });
    await page.locator('.nx-workspace-tab[data-kind="terminals"]').click();
    await page
      .getByRole('button', { name: /novo navegador|new browser|nuevo navegador/i })
      .first()
      .click();
    const surface = page.locator('[data-feature-tour-scope^="browser-"]');
    await surface.waitFor({ state: 'visible', timeout: 10000 });

    // Session starts at desktop (1280x800): cycle must advance
    // desktop -> phone -> tablet -> desktop on one session.
    for (let step = 0; step < 3; step += 1) {
      const currentIndex = (2 + step) % PRESETS.length;
      const expected = PRESETS[(currentIndex + 1) % PRESETS.length];
      const toggle = surface.locator('button[aria-label^="Alternar para"]').first();
      assert.match(
        await toggle.getAttribute('aria-label'),
        new RegExp(expected.label.replace(/[()]/g, '\\$&'), 'i'),
        `toggle announces ${expected.label} before the cycle step ${step}`,
      );
      await toggle.click();
      await page
        .locator('[data-feature-tour-scope^="browser-"]')
        .getByTitle(expected.label)
        .waitFor({ state: 'visible', timeout: 5000 });
      await page.waitForTimeout(150);
      assert.equal(viewportActions.length, step + 1, `viewport action ${step + 1} sent`);
      assert.deepEqual(
        viewportActions[step],
        { width: expected.width, height: expected.height },
        `cycle step ${step} sends ${expected.id} dimensions`,
      );
      // Focus stays on the toggle for keyboard users across the cycle.
      await toggle.focus();
      const focused = await page.evaluate(() => {
        const active = document.activeElement;
        return active ? (active.getAttribute('aria-label') ?? '') : '';
      });
      assert.match(
        focused,
        /alternar para/i,
        `focus must remain on the viewport toggle at step ${step}`,
      );
      const axe = await new AxeBuilder({ page }).analyze();
      const violations = axe.violations.filter((violation) => violation.impact !== 'minor');
      assert.deepEqual(
        violations.map((violation) => `${violation.id}:${violation.impact}`),
        [],
        `axe violations at ${expected.id}: ${JSON.stringify(
          violations.map((violation) => ({
            id: violation.id,
            impact: violation.impact,
            targets: violation.nodes.map((node) => node.target),
          })),
        )}`,
      );
      await page.screenshot({
        path: path.join(artifactDir, `device-preview-${expected.id}.png`),
        fullPage: false,
      });
    }

    assert.equal(viewportActions.length, 3, 'exactly one action per preset step');
    assert.deepEqual(runtimeErrors, [], 'no console/page errors during the cycle');
    console.log('Device preview E2E: PASS (3 presets, same session, Axe clean)');
  } catch (error) {
    console.error('Device preview E2E: FAIL');
    console.error(getOutput ? getOutput() : '');
    throw error;
  } finally {
    if (browser) await browser.close().catch(() => undefined);
    if (server) await terminateProcess(server);
  }
}

await main();
