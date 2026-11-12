#!/usr/bin/env node
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  launchFixtureBrowser,
  seedFixtureProject,
  startFixtureServer,
  terminateProcess,
} from './e2e-fixture.mjs';

const filename = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(filename), '..', '..');
const artifactDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR ||
  path.join(repoRoot, '.tempmediaStorage', 'browser-design-inspect');
if (!existsSync(artifactDir)) mkdirSync(artifactDir, { recursive: true });
const jpeg =
  '/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAP//////////////////////////////////////////////////////////////////////////////////////2wBDAf//////////////////////////////////////////////////////////////////////////////////////wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAb/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIQAxAAAAH/AP/EABQQAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQEAAT8Af//EABQRAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQMBAT8Af//EABQRAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQIBAT8Af//Z';

async function main() {
  console.log('=== Browser design-inspect UI handoff E2E (API fixtures) ===');
  const { server, bootstrapUrl, isolatedDataDir, isolatedProjectPath, getOutput } =
    await startFixtureServer();
  let browser;
  let context;
  let page;
  const askCalls = [];
  const clickCalls = [];
  const runtimeErrors = [];
  let activeViewport = 'startup';
  let startCalls = 0;

  try {
    ({ browser, context, page } = await launchFixtureBrowser(bootstrapUrl));
    page.on('console', (message) => {
      if (message.type() === 'error') {
        runtimeErrors.push({ viewport: activeViewport, source: 'console', text: message.text() });
      }
    });
    page.on('pageerror', (error) => {
      runtimeErrors.push({ viewport: activeViewport, source: 'pageerror', text: error.message });
    });
    const { project, apiOrigin } = await seedFixtureProject(
      context,
      bootstrapUrl,
      isolatedProjectPath,
    );

    await context.route(`**/api/v1/projects/${project.id}/agents`, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'agent-alpha',
            project_id: project.id,
            name: 'Alpha',
            role: 'Lead',
            status: 'RUNNING',
          },
          {
            id: 'agent-beta',
            project_id: project.id,
            name: 'Beta',
            role: 'Builder',
            status: 'RUNNING',
          },
        ]),
      }),
    );
    await context.route('**/api/v1/agents/*/ask', async (route) => {
      const id = new URL(route.request().url()).pathname.split('/').at(-2);
      askCalls.push({ id, body: route.request().postDataJSON() });
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          agent_id: id,
          runtime_id: `runtime-${id}`,
          started: false,
          accepted: true,
        }),
      });
    });
    await context.route('**/api/v1/agents/*/start', async (route) => {
      startCalls += 1;
      return route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });
    await context.route('**/api/v1/browser/sessions**', async (route) => {
      const url = new URL(route.request().url());
      const method = route.request().method();
      const parts = url.pathname.split('/').filter(Boolean);
      if (method === 'GET' && parts.length === 4) {
        return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      }
      if (method === 'POST' && parts.length === 4) {
        return route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: '0123456789abcdef0123456789abcdef',
            project_id: project.id,
            url: 'http://localhost:3000',
            title: 'Fixture App',
            width: 1280,
            height: 800,
          }),
        });
      }
      if (parts[5] === 'snapshot' && method === 'GET') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            url: 'http://localhost:3000',
            title: 'Fixture App',
            untrusted: true,
            text: 'Fixture page text',
            selection: '',
            elements: [
              {
                selector: '#hero-title',
                tag: 'h1',
                role: 'heading',
                label: 'Hero title',
                text: 'Welcome',
              },
            ],
          }),
        });
      }
      if (parts[5] === 'inspect' && method === 'POST') {
        const body = route.request().postDataJSON();
        assert.equal(body.selector, '#hero-title');
        const crop = await page.evaluate(() => {
          const canvas = document.createElement('canvas');
          canvas.width = 280;
          canvas.height = 64;
          const context = canvas.getContext('2d');
          if (!context) return '';
          context.fillStyle = '#d7e0ec';
          context.fillRect(0, 0, canvas.width, canvas.height);
          context.fillStyle = '#24364b';
          context.font = 'bold 18px sans-serif';
          context.fillText('Selected hero title', 12, 36);
          return canvas.toDataURL('image/jpeg', 0.8).split(',')[1] || '';
        });
        assert.ok(crop, 'fixture crop JPEG must be rendered');
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            selector: '#hero-title',
            tag: 'h1',
            role: 'heading',
            name: 'Hero title',
            text: 'Welcome',
            attributes: { id: 'hero-title' },
            computed_styles: { color: 'rgb(10, 20, 30)', width: '200px' },
            masking_applied: true,
            crop_jpeg: crop,
          }),
        });
      }
      if (parts[5] === 'action' && method === 'POST') {
        clickCalls.push(route.request().postDataJSON());
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: parts[3],
            project_id: project.id,
            url: 'http://localhost:3000',
            width: 1280,
            height: 800,
          }),
        });
      }
      if (parts[5] === 'control' && method === 'POST') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ controller: true, lease_seconds: 25 }),
        });
      }
      if (parts[5] === 'handoff' && method === 'GET') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: '' }),
        });
      }
      if (parts[5] === 'grants' && method === 'GET') {
        return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
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
      (url) =>
        url.pathname ===
        '/api/v1/browser/sessions/0123456789abcdef0123456789abcdef/stream',
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
    await page
      .locator('[data-feature-tour-scope^="browser-"]')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.getByRole('button', { name: /alternar console de inspeção/i }).click();
    await page.getByRole('button', { name: /capturar dom/i }).click();
    await page
      .getByRole('button', { name: /hero title/i })
      .first()
      .click();
    await page
      .getByText('#hero-title', { exact: false })
      .waitFor({ state: 'visible', timeout: 5000 });
    const card = page.locator('[data-feature-tour-scope^="browser-"] [aria-live="polite"]');
    const selectedHeading = card.locator('[data-design-inspection-part="heading"]');
    const selectedSelector = card.locator('[data-design-inspection-part="selector"]');
    const crop = card.locator('[data-design-inspection-part="crop"]');
    const activateButton = card.locator('[data-design-inspection-part="action"]');
    const maskingNotice = card.locator('[data-design-inspection-part="notice"]');
    const agentPicker = card.locator('[data-design-inspection-part="agent"]');
    const quotaNotice = card.locator('[data-design-inspection-part="consent"]');
    const sendButton = card.locator('[data-design-inspection-part="send"]');
    await crop.waitFor({ state: 'visible', timeout: 5000 });
    assert.equal(askCalls.length, 0, 'inspection must not send anything to an Agent');
    assert.equal(startCalls, 0, 'inspection must never start an Agent');

    await page.locator('#design-inspect-agent').selectOption('agent-beta');
    assert.equal(askCalls.length, 0, 'selecting a destination must not send anything');

    for (const viewport of [
      { width: 1280, height: 800 },
      { width: 390, height: 844 },
      { width: 768, height: 900 },
      { width: 1440, height: 900 },
    ]) {
      activeViewport = `${viewport.width}x${viewport.height}`;
      const errorCheckpoint = runtimeErrors.length;
      await page.setViewportSize(viewport);
      await page.waitForTimeout(150);
      for (const [label, locator] of [
        ['selected element tag', selectedHeading],
        ['selected element selector', selectedSelector],
        ['selected element crop', crop],
        ['selected element action', activateButton],
        ['masking limitation notice', maskingNotice],
        ['explicit Agent picker', agentPicker],
        ['quota and no-auto-start notice', quotaNotice],
        ['Send control', sendButton],
      ]) {
        assert.equal(await locator.isVisible(), true, `${label} hidden at ${viewport.width}px`);
      }
      for (const [label, locator] of [
        ['masking limitation notice', maskingNotice],
        ['quota and no-auto-start notice', quotaNotice],
      ]) {
        const hasClippedText = await locator.evaluate(
          (element) => element.scrollHeight > element.clientHeight + 1,
        );
        assert.equal(
          hasClippedText,
          false,
          `${label} text is clipped at ${viewport.width}px`,
        );
      }
      const visibleInDrawer = await card.evaluate((element) => {
        let drawer = element.parentElement;
        while (drawer && drawer.scrollHeight <= drawer.clientHeight + 1) {
          drawer = drawer.parentElement;
        }
        if (!drawer) return false;
        const drawerRect = drawer.getBoundingClientRect();
        const visibleTop = drawerRect.top + drawer.clientTop;
        const visibleBottom = visibleTop + drawer.clientHeight;
        const children = [
          ...element.querySelectorAll('[data-design-inspection-part]'),
        ];
        return {
          visible: children.every((item) => {
          const rect = item.getBoundingClientRect();
          return (
            rect.height === 0 || (rect.top >= visibleTop - 1 && rect.bottom <= visibleBottom + 1)
          );
          }),
          drawer: { top: visibleTop, bottom: visibleBottom, height: drawer.clientHeight },
          card: element.getBoundingClientRect().toJSON(),
          clipped: children
            .filter((item) => {
              const rect = item.getBoundingClientRect();
              return rect.height > 0 && (rect.top < visibleTop - 1 || rect.bottom > visibleBottom + 1);
            })
            .map((item) => ({ tag: item.tagName, text: item.textContent, rect: item.getBoundingClientRect().toJSON() })),
        };
      });
      assert.equal(
        visibleInDrawer.visible,
        true,
        `inspection result clipped inside drawer at ${viewport.width}px: ${JSON.stringify(visibleInDrawer)}`,
      );
      const imageBounds = await crop.boundingBox();
      assert.ok(
        imageBounds && imageBounds.width > 20 && imageBounds.height > 20,
        `crop is not visibly rendered at ${viewport.width}px`,
      );
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth);
      assert.equal(overflow, false, `page overflow at ${viewport.width}px`);
      await page.screenshot({
        path: path.join(artifactDir, `browser-design-inspect-${viewport.width}.png`),
        fullPage: true,
      });
      await page.waitForTimeout(100);
      assert.deepEqual(
        runtimeErrors.slice(errorCheckpoint),
        [],
        `console/page errors at ${activeViewport}: ${JSON.stringify(runtimeErrors.slice(errorCheckpoint))}`,
      );
    }

    assert.match(
      await activateButton.getAttribute('title'),
      /#hero-title/,
      'the activation control must identify its selected DOM target',
    );
    await activateButton.click();
    assert.equal(
      clickCalls.length,
      1,
      'Activate should call only the selected DOM selector action',
    );
    assert.equal(askCalls.length, 0, 'activate action must not hand off context');

    await page
      .getByRole('button', { name: /enviar inspeção|send inspection|enviar inspección/i })
      .click();
    await page.waitForFunction(() =>
      Boolean(document.querySelector('[role="status"]')?.textContent),
    );
    assert.equal(askCalls.length, 1, `expected one explicit ask, saw ${askCalls.length}`);
    assert.equal(askCalls[0].id, 'agent-beta', 'only the selected Agent may receive the context');
    assert.equal(askCalls[0].body.start_if_needed, false, 'handoff must not auto-start an Agent');
    assert.match(askCalls[0].body.prompt, /Selector: #hero-title/);
    assert.equal(startCalls, 0);
    const unexpected = (await page.locator('body').innerText()).includes('undefined');
    assert.equal(unexpected, false, 'UI must not render undefined data');
    console.log(
      'PASS: selected-element inspection is read-only; explicit send targets Beta only; no Agent auto-start.',
    );
    console.log(`Visual QA artifacts: ${artifactDir}`);
  } catch (error) {
    if (page) {
      await page
        .screenshot({
          path: path.join(artifactDir, 'browser-design-inspect-failure.png'),
          fullPage: true,
        })
        .catch(() => {});
    }
    writeFileSync(
      path.join(artifactDir, 'browser-design-inspect-diagnostics.json'),
      JSON.stringify(
        {
          error: String(error?.stack || error),
          serverOutput: getOutput(),
          askCalls,
          clickCalls,
          startCalls,
        },
        null,
        2,
      ),
    );
    throw error;
  } finally {
    if (context) await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
    await terminateProcess(server);
    if (!process.env.NEXUS_E2E_DATA_DIR && isolatedDataDir)
      rmSync(isolatedDataDir, { recursive: true, force: true });
  }
}

main().catch((error) => {
  console.error('Browser design-inspect E2E FAILED:', error);
  process.exit(1);
});
