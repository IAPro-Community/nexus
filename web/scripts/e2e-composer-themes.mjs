#!/usr/bin/env node
import assert from 'node:assert/strict';
import { mkdirSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';
import {
  playwrightExecutablePath,
  seedFixtureProject,
  startFixtureServer,
  terminateProcess,
} from './e2e-fixture.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '../..');
const screenshotDir = path.join(repoRoot, '.tempmediaStorage', 'composer-themes');
mkdirSync(screenshotDir, { recursive: true });

const themes = [
  {
    preset: 'orca-dusk',
    scheme: 'dark',
    finish: 'subtle-gradient',
    category: /Degradê Suave/i,
    label: /Orca Dusk/i,
  },
  {
    preset: 'slate-flat',
    scheme: 'dark',
    finish: 'flat',
    category: /Essenciais Flat/i,
    label: /Slate Flat/i,
  },
  {
    preset: 'paper-flat',
    scheme: 'light',
    finish: 'flat',
    category: /Essenciais Flat/i,
    label: /Paper Flat/i,
  },
  {
    preset: 'clay-flat',
    scheme: 'light',
    finish: 'flat',
    category: /Essenciais Flat/i,
    label: /Clay Flat/i,
  },
  {
    preset: 'nexus-dark',
    scheme: 'dark',
    category: /Dark Modern/i,
    label: /Nexus Dark/i,
  },
  {
    preset: 'nexus-light',
    scheme: 'light',
    category: /Light Clean/i,
    label: /Nexus Light/i,
  },
  {
    preset: 'midnight',
    scheme: 'dark',
    category: /Dark Modern/i,
    label: /Midnight/i,
  },
  {
    preset: 'nord',
    scheme: 'dark',
    category: /Clássicos do Código/i,
    label: /^Nord$/i,
  },
  {
    preset: 'dracula',
    scheme: 'dark',
    category: /Dark Modern/i,
    label: /Dracula/i,
  },
  {
    preset: 'solarized-dark',
    scheme: 'dark',
    category: /Clássicos do Código/i,
    label: /Solarized Dark/i,
  },
  {
    preset: 'solarized-light',
    scheme: 'light',
    category: /Light Clean/i,
    label: /Solarized Light/i,
  },
  {
    preset: 'graphite-focus',
    scheme: 'dark',
    category: /Dark Modern/i,
    label: /Graphite Focus/i,
  },
  {
    preset: 'vasco-noir',
    scheme: 'dark',
    category: /Dark Modern/i,
    label: /Vasco Noir/i,
  },
  {
    preset: 'nord-light',
    scheme: 'light',
    category: /Light Clean/i,
    label: /Nord Light/i,
  },
  {
    preset: 'aurora-ops',
    scheme: 'dark',
    category: /Dark Modern/i,
    label: /Aurora Ops/i,
  },
  {
    preset: 'monokai',
    scheme: 'dark',
    category: /Clássicos do Código/i,
    label: /Monokai Dark/i,
  },
  {
    preset: 'obsidian',
    scheme: 'dark',
    category: /Dark Modern/i,
    label: /Obsidian/i,
  },
  {
    preset: 'high-contrast-dark',
    scheme: 'high-contrast',
    category: /Acessibilidade Extrema/i,
    label: /High Contrast Dark/i,
  },
  {
    preset: 'high-contrast-light',
    scheme: 'light',
    category: /Acessibilidade Extrema/i,
    label: /High Contrast Light/i,
  },
];

let fixture;
let browser;

async function ensureThemeCategoryOpen(page, theme) {
  let radio = page.getByRole('radio', { name: theme.label }).first();
  if ((await radio.count()) === 0) {
    const header = page.getByRole('button', { name: theme.category }).first();
    await header.waitFor({ state: 'visible', timeout: 10000 });
    if ((await header.getAttribute('aria-expanded')) !== 'true') {
      await header.click();
    }
    radio = page.getByRole('radio', { name: theme.label }).first();
  }
  await radio.waitFor({ state: 'visible', timeout: 10000 });
  return radio;
}

async function assertThemeApplied(page, theme) {
  await page.waitForFunction(
    ({ preset, scheme, finish }) =>
      document.documentElement.dataset.preset === preset &&
      document.documentElement.dataset.theme === scheme &&
      document.documentElement.dataset.finish === finish,
    { preset: theme.preset, scheme: theme.scheme, finish: theme.finish || 'standard' },
    { timeout: 10000 },
  );
  const applied = await page.evaluate(() => {
    const root = document.documentElement;
    const style = getComputedStyle(root);
    return {
      preset: root.dataset.preset || '',
      theme: root.dataset.theme || '',
      finish: root.dataset.finish || '',
      background: style.getPropertyValue('--nx-bg').trim(),
      text: style.getPropertyValue('--nx-text').trim(),
      accent: style.getPropertyValue('--nx-accent').trim(),
    };
  });
  assert.equal(applied.preset, theme.preset);
  assert.equal(applied.theme, theme.scheme);
  assert.equal(applied.finish, theme.finish || 'standard');
  assert.ok(applied.background, `${theme.preset} must define --nx-bg`);
  assert.ok(applied.text, `${theme.preset} must define --nx-text`);
  assert.ok(applied.accent, `${theme.preset} must define --nx-accent`);
  assert.notEqual(
    applied.background,
    applied.text,
    `${theme.preset} foreground/background must differ`,
  );
}

async function assertComposerLayout(page, viewportName) {
  const result = await page.evaluate(() => {
    const composer = document.querySelector('.nx-composer-surface');
    if (!composer) return { found: false };
    const rect = composer.getBoundingClientRect();
    const ancestors = [];
    let current = composer;
    while (current && ancestors.length < 12) {
      const currentRect = current.getBoundingClientRect();
      const style = getComputedStyle(current);
      ancestors.push({
        tag: current.tagName,
        className: typeof current.className === 'string' ? current.className : '',
        width: currentRect.width,
        left: currentRect.left,
        right: currentRect.right,
        cssWidth: style.width,
        minWidth: style.minWidth,
        maxWidth: style.maxWidth,
        display: style.display,
        overflowX: style.overflowX,
        position: style.position,
      });
      current = current.parentElement;
    }
    return {
      found: true,
      width: rect.width,
      left: rect.left,
      right: rect.right,
      viewportWidth: window.innerWidth,
      documentWidth: document.documentElement.scrollWidth,
      bodyWidth: document.body.scrollWidth,
      ancestors,
    };
  });
  assert.equal(result.found, true, `Composer must exist at ${viewportName}`);
  assert.ok(result.width > 0, `Composer must have measurable width at ${viewportName}`);
  assert.ok(result.left >= -1, `Composer cannot overflow left at ${viewportName}`);
  assert.ok(
    result.right <= result.viewportWidth + 1,
    `Composer cannot overflow right at ${viewportName}: ${JSON.stringify(result)}`,
  );
  assert.ok(
    Math.max(result.documentWidth, result.bodyWidth) <= result.viewportWidth + 1,
    `document cannot horizontally overflow at ${viewportName}: ${JSON.stringify(result)}`,
  );
}

async function assertA11y(page, theme) {
  const axe = await new AxeBuilder({ page }).exclude('.xterm').analyze();
  const blockers = axe.violations.filter((violation) =>
    ['serious', 'critical'].includes(violation.impact),
  );
  if (blockers.length > 0) {
    console.error(JSON.stringify(blockers, null, 2));
  }
  assert.equal(
    blockers.length,
    0,
    `${theme.preset} must not expose serious/critical Axe violations: ${blockers
      .map((item) => item.id)
      .join(', ')}`,
  );
}

async function selectThemeThroughSettings(page, apiOrigin, theme) {
  await page.setViewportSize({ width: 1440, height: 900 });
  // Cold start (first navigation in a fresh browser context) pays the bundle
  // decode and JIT warm-up cost, which can exceed 10s on a loaded machine.
  await page.goto(`${apiOrigin}/settings`, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.locator('.nx-settings-tabs').waitFor({ state: 'visible', timeout: 30000 });

  const radio = await ensureThemeCategoryOpen(page, theme);
  await radio.click();
  await assertThemeApplied(page, theme);
  assert.equal(await radio.getAttribute('aria-checked'), 'true');

  // Reload proves the user's selected preset is persisted by ThemeProvider,
  // rather than surviving only in transient React state. Same 30s budget as
  // the cold path: a loaded machine can stall the reload render too.
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.locator('.nx-settings-tabs').waitFor({ state: 'visible', timeout: 30000 });
  await assertThemeApplied(page, theme);
  const persistedRadio = await ensureThemeCategoryOpen(page, theme);
  assert.equal(
    await persistedRadio.getAttribute('aria-checked'),
    'true',
    `${theme.preset} selection must survive reload`,
  );
}

async function validateComposerTheme(page, apiOrigin, project, theme) {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto(`${apiOrigin}/p/${encodeURIComponent(project.id)}/work`, {
    waitUntil: 'domcontentloaded',
    timeout: 15000,
  });
  await page.locator('.nx-composer-surface').waitFor({ state: 'visible', timeout: 10000 });
  await assertThemeApplied(page, theme);
  await assertComposerLayout(page, `${theme.preset} desktop`);
  await assertA11y(page, theme);
  await page.screenshot({
    path: path.join(screenshotDir, `${theme.preset}-desktop.png`),
    fullPage: true,
  });

  await page.setViewportSize({ width: 768, height: 1024 });
  await page.locator('.nx-composer-surface').waitFor({ state: 'visible', timeout: 5000 });
  await assertThemeApplied(page, theme);
  await assertComposerLayout(page, `${theme.preset} tablet`);
  await assertA11y(page, theme);
  await page.screenshot({
    path: path.join(screenshotDir, `${theme.preset}-tablet.png`),
    fullPage: true,
  });

  await page.setViewportSize({ width: 390, height: 844 });
  await page.locator('.nx-composer-surface').waitFor({ state: 'visible', timeout: 5000 });
  await assertThemeApplied(page, theme);
  await page.screenshot({
    path: path.join(screenshotDir, `${theme.preset}-mobile.png`),
    fullPage: true,
  });
  await assertComposerLayout(page, `${theme.preset} mobile`);
  await assertA11y(page, theme);
}

async function main() {
  fixture = await startFixtureServer();
  browser = await chromium.launch({
    executablePath: playwrightExecutablePath(),
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
  });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const { project, apiOrigin } = await seedFixtureProject(
    context,
    fixture.bootstrapUrl,
    fixture.isolatedProjectPath,
  );
  const page = await context.newPage();
  const pageErrors = [];
  const consoleErrors = [];
  page.on('pageerror', (error) => pageErrors.push(String(error?.message || error)));
  page.on('console', (message) => {
    if (message.type() === 'error') consoleErrors.push(message.text());
  });

  const themeFailures = [];
  for (const theme of themes) {
    console.log(`composer theme validation: ${theme.preset}`);
    try {
      await selectThemeThroughSettings(page, apiOrigin, theme);
      await validateComposerTheme(page, apiOrigin, project, theme);
    } catch (error) {
      const detail = error instanceof Error ? error.stack || error.message : String(error);
      themeFailures.push(`${theme.preset}: ${detail}`);
      console.error(`composer theme validation failed: ${theme.preset}`);
      console.error(detail);
    }
  }

  assert.deepEqual(pageErrors, [], `browser page errors: ${pageErrors.join(' | ')}`);
  assert.deepEqual(consoleErrors, [], `browser console errors: ${consoleErrors.join(' | ')}`);
  assert.equal(
    themeFailures.length,
    0,
    `Composer theme matrix failures:\n${themeFailures.join('\n\n')}`,
  );
  console.log('composer theme visual e2e PASS');
}

main()
  .catch((error) => {
    console.error('composer theme visual e2e FAIL');
    console.error(error);
    process.exitCode = 1;
  })
  .finally(async () => {
    if (browser) {
      try {
        await browser.close();
      } catch (_) {}
    }
    if (fixture?.server) {
      await terminateProcess(fixture.server);
    }
    if (fixture?.isolatedDataDir && !process.env.NEXUS_E2E_DATA_DIR) {
      try {
        rmSync(fixture.isolatedDataDir, { recursive: true, force: true });
      } catch (_) {}
    }
  });
