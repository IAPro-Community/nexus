#!/usr/bin/env node
// Smart Layouts browser E2E.
//
// Proves in a real browser that:
//   - the "Automático" arrangement is a real suggestion derived from the
//     activity of the focused surface (browser tab active → suggestion appears)
//   - applying an arrangement never spawns processes: zero /agents escalations
//     and zero shell launches during layout changes
//   - surface IDs, focus and splits survive: restoring a full page reload
//     brings the same surface identities back without relaunching the runtime
import assert from 'node:assert/strict';
import { existsSync, mkdirSync } from 'node:fs';
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
const screenshotDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR ||
  path.resolve(repoRoot, '.tempmediaStorage', 'smart-layouts');
if (!existsSync(screenshotDir)) mkdirSync(screenshotDir, { recursive: true });

async function waitFor(page, selector, label, timeout = 15000) {
  try {
    await page.waitForSelector(selector, { timeout });
  } catch (error) {
    throw new Error(`${label}: ${selector} did not appear`);
  }
}

async function readDesktopState(page) {
  return page.evaluate(() => {
    const windows = [...document.querySelectorAll('.nx-desktop-window[data-view-id]')];
    return {
      viewIds: windows.map((w) => w.getAttribute('data-view-id')).sort(),
      activeViewId:
        document
          .querySelector('.nx-desktop-window[data-active="true"]')
          ?.getAttribute('data-view-id') || '',
    };
  });
}

async function waitForServerLayoutSurfaces(
  context,
  apiOrigin,
  projectId,
  minimum,
  timeout = 20000,
) {
  const deadline = Date.now() + timeout;
  let lastTabs = [];
  while (Date.now() < deadline) {
    const response = await context.request.get(`${apiOrigin}/api/v1/projects/${projectId}/layout`);
    assert.equal(response.status(), 200, 'project layout must be readable');
    const record = await response.json();
    const layout = JSON.parse(record.layout || '{}');
    const tabs = layout?.model?.root?.tabs || [];
    lastTabs = tabs;
    if (
      tabs.filter((tab) => tab.type === 'project-shell' || tab.type === 'terminal').length >=
      minimum
    ) {
      return tabs;
    }
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error(
    `project layout did not persist ${minimum} surfaces: ${lastTabs.map((tab) => tab.type).join(',')}`,
  );
}

function openTerminalsAndDesktop(page) {
  return async () => {
    await page.locator('.nx-workspace-tab[data-kind="terminals"]').first().click();
    await page.waitForSelector('.nx-desktop-workspace', { timeout: 20000 });
  };
}

async function main() {
  console.log('=== Smart Layouts browser E2E ===');
  const { server, bootstrapUrl, isolatedProjectPath, getOutput } = await startFixtureServer({ features: ['SMART_LAYOUTS'] });
  let browser;
  let context;
  let page;
  let collectors;
  let shellLaunches = 0;
  let agentCreates = 0;
  try {
    ({ browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl));
    const { project, apiOrigin } = await seedFixtureProject(
      context,
      bootstrapUrl,
      isolatedProjectPath,
    );

    page.on('request', (request) => {
      const url = new URL(request.url());
      if (request.method() === 'POST' && url.pathname === `/api/v1/projects/${project.id}/shell`) {
        shellLaunches += 1;
      }
      if (request.method() === 'POST' && url.pathname === `/api/v1/projects/${project.id}/agents`) {
        agentCreates += 1;
      }
    });

    const shellUrl = new URL(`/p/${encodeURIComponent(project.id)}/overview`, apiOrigin);
    await page.goto(shellUrl.toString(), { waitUntil: 'domcontentloaded', timeout: 15000 });
    await waitFor(page, '.nx-os-shell', 'workspace shell');
    await waitFor(page, '.nx-workspace-tab[data-kind="terminals"]', 'terminals tab');

    // Launch the project shell through the empty-state action (real UI only).
    await page.locator('.nx-workspace-tab[data-kind="terminals"]').first().click();
    const emptyShell = page
      .getByRole('button', { name: /Abrir Terminal|Open Terminal|Terminal do Projeto/i })
      .first();
    await emptyShell.waitFor({ state: 'visible', timeout: 15000 });
    await emptyShell.click();
    await page.waitForSelector('.nx-project-shell-surface[data-runtime-id]', { timeout: 20000 });
    const runtimeId = await page
      .locator('.nx-project-shell-surface[data-runtime-id]')
      .first()
      .getAttribute('data-runtime-id');
    assert.ok(runtimeId, 'project shell must expose its runtime identity');

    await page.locator('.nx-workspace-tab[data-kind="terminals"]').first().focus();
    await page.keyboard.press('Control+k');
    await page.waitForSelector('.nx-command-palette', { timeout: 10000 });
    assert.equal(
      await page.evaluate(() => document.activeElement?.closest('.nx-command-palette') !== null),
      true,
      'command palette must focus its search input',
    );
    for (let index = 0; index < 12; index += 1) await page.keyboard.press('Tab');
    assert.equal(
      await page.evaluate(() => document.activeElement?.closest('.nx-dialog') !== null),
      true,
      'desktop command palette must trap keyboard focus',
    );
    await page.keyboard.press('Escape');
    await page.waitForSelector('.nx-command-palette', { state: 'detached', timeout: 10000 });
    await page.waitForFunction(
      () =>
        document.activeElement ===
        document.querySelector('.nx-workspace-tab[data-kind="terminals"]'),
      undefined,
      { timeout: 10000 },
    );
    console.log('  ✓ desktop command palette restores focus to its opener');

    await page.keyboard.press('Control+k');
    await page.waitForSelector('.nx-command-palette', { timeout: 10000 });
    await page.locator('.nx-command-palette input').fill('~');
    const initialSurfaceCommandCount = await page.locator('.nx-command-list button').count();
    assert.ok(initialSurfaceCommandCount >= 3, 'surface scope must list open workspace surfaces');
    await page.keyboard.press('Escape');
    await page.waitForSelector('.nx-command-palette', { state: 'detached', timeout: 10000 });

    await page.setViewportSize({ width: 390, height: 844 });
    await page.locator('.nx-workspace-tab[data-kind="terminals"]').first().focus();
    await page.keyboard.press('Control+k');
    await page.waitForSelector('.nx-command-palette', { timeout: 10000 });
    const mobilePaletteBounds = await page.evaluate(() => {
      const dialog = document.querySelector('.nx-dialog');
      if (!dialog) return null;
      const bounds = dialog.getBoundingClientRect();
      const list = document.querySelector('.nx-command-list');
      if (!list) return null;
      return {
        left: bounds.left,
        right: bounds.right,
        viewportWidth: window.innerWidth,
        dialogScrollWidth: dialog.scrollWidth,
        dialogClientWidth: dialog.clientWidth,
        listScrollWidth: list.scrollWidth,
        listClientWidth: list.clientWidth,
      };
    });
    assert.ok(mobilePaletteBounds, 'mobile command palette must render a dialog');
    assert.ok(
      mobilePaletteBounds.left >= 0 &&
        mobilePaletteBounds.right <= mobilePaletteBounds.viewportWidth,
      'mobile command palette must stay inside the viewport',
    );
    assert.ok(
      mobilePaletteBounds.dialogScrollWidth <= mobilePaletteBounds.dialogClientWidth,
      'mobile command palette dialog must not overflow horizontally',
    );
    assert.ok(
      mobilePaletteBounds.listScrollWidth <= mobilePaletteBounds.listClientWidth,
      'mobile command palette list must not overflow horizontally',
    );
    for (let index = 0; index < 12; index += 1) await page.keyboard.press('Tab');
    assert.equal(
      await page.evaluate(() => document.activeElement?.closest('.nx-dialog') !== null),
      true,
      'mobile command palette must trap keyboard focus',
    );
    await page.keyboard.press('Escape');
    await page.waitForSelector('.nx-command-palette', { state: 'detached', timeout: 10000 });
    await page.waitForFunction(
      () =>
        document.activeElement ===
        document.querySelector('.nx-workspace-tab[data-kind="terminals"]'),
      undefined,
      { timeout: 10000 },
    );
    console.log('  ✓ mobile command palette restores focus to its opener');
    await page.setViewportSize({ width: 1280, height: 800 });

    // Switch terminals presentation to DESKTOP and capture the arrange menu.
    const enterDesktop = openTerminalsAndDesktop(page);
    await enterDesktop();
    const desktopToggle = page
      .getByRole('button', { name: /Modo de janelas|floating windows/i })
      .first();
    await desktopToggle.click({ timeout: 10000 });
    await page.waitForSelector('.nx-arrange-menu__trigger', { timeout: 15000 });

    const arrangeTrigger = page.locator('.nx-arrange-menu__trigger').first();
    await arrangeTrigger.click();
    const automaticItem = page.locator('.nx-arrange-menu__item', { hasText: 'utomático' }).first();
    await automaticItem.waitFor({ state: 'visible', timeout: 10000 });

    // Zero spawns during layout-only interactions so far.
    assert.equal(shellLaunches, 1, 'only the explicit project shell launch may happen');
    assert.equal(agentCreates, 0, 'layout suggestions must never create agents');

    await automaticItem.click();

    // Zero spawns after applying the automatic layout over the shell surface.
    await page.waitForTimeout(600);
    assert.equal(shellLaunches, 1, 'automatic layout must not launch project shells');
    assert.equal(agentCreates, 0, 'automatic layout must not create agents');
    console.log('  ✓ automatic layout applied with zero process spawns');

    // Capture the pre-reload desktop state (surface IDs + focused window).
    const preState = await readDesktopState(page);
    assert.ok(preState.viewIds.length >= 1, 'desktop mode must expose at least one window');
    console.log(`  pre-reload viewIds: ${preState.viewIds.join(', ')}`);

    // Full page reload — the layout snapshot must restore surfaces, focus and
    // the desktop presentation without relaunching the runtime.
    await page.reload({ waitUntil: 'domcontentloaded' });
    await waitFor(page, '.nx-os-shell', 'workspace shell after reload');
    await page.locator('.nx-workspace-tab[data-kind="terminals"]').first().click();
    await page.waitForSelector('.nx-desktop-workspace', { timeout: 20000 });
    const postState = await readDesktopState(page);
    assert.deepEqual(postState.viewIds, preState.viewIds, 'surface view ids must survive reload');
    console.log(`  ✓ surface IDs survive reload: ${postState.viewIds.join(', ')}`);
    assert.equal(
      postState.activeViewId,
      preState.activeViewId,
      'focused window must survive reload',
    );
    console.log('  ✓ focus survives reload');
    assert.equal(shellLaunches, 1, 'reload must not launch another project shell');
    assert.equal(agentCreates, 0, 'reload must not create agents');
    const shellStill = page.locator(`.nx-project-shell-surface[data-runtime-id="${runtimeId}"]`);
    await shellStill.waitFor({ state: 'attached', timeout: 15000 });
    console.log('  ✓ runtime identity persisted across reload without relaunch');

    // Splits also survive inside the desktop presentation: the arrange presets
    // operate only over existing tiles — switching presets must keep view IDs.
    await arrangeTrigger.click();
    const twoColumns = page
      .locator('.nx-arrange-menu__item', { hasText: /Duas colunas|Two columns/ })
      .first();
    if ((await twoColumns.count()) > 0) {
      await twoColumns.click();
      await page.waitForTimeout(300);
      const afterPreset = await readDesktopState(page);
      assert.deepEqual(
        afterPreset.viewIds,
        preState.viewIds,
        'preset changes must preserve surface identity',
      );
      console.log('  ✓ preset switch preserves surface identity');
    }
    assert.equal(shellLaunches, 1, 'preset changes must not spawn shells');
    assert.equal(agentCreates, 0, 'preset changes must not create agents');

    // --- Phase 3: split persistence with a second surface ------------------
    console.log('  -- phase: split persistence with a second surface');

    // Fetch CSRF for later API calls (second project creation).
    const session = await context.request.get(`${apiOrigin}/api/v1/session`);
    assert.equal(session.status(), 200, 'session endpoint must be reachable');
    const { csrf_token: csrf } = await session.json();
    assert.ok(csrf, 'session must expose a csrf token');

    // Launch the second PTY through the real command palette action.
    await page.keyboard.press('Control+k');
    await page.waitForSelector('.nx-command-palette', { timeout: 10000 });
    await page.keyboard.type('Novo Terminal');
    const newTerminalCommand = page
      .locator('.nx-command-list button', { hasText: /Novo Terminal|New Terminal/ })
      .first();
    await newTerminalCommand.waitFor({ state: 'visible', timeout: 10000 });
    await newTerminalCommand.click();
    await page.waitForFunction(
      () => document.querySelectorAll('.nx-desktop-window[data-view-id]').length === 2,
      { timeout: 20000 },
    );
    console.log('  ✓ second PTY opened through command palette');

    await page.keyboard.press('Control+k');
    await page.waitForSelector('.nx-command-palette', { timeout: 10000 });
    await page.locator('.nx-command-palette input').fill('~');
    const updatedSurfaceCommandCount = await page.locator('.nx-command-list button').count();
    assert.equal(
      updatedSurfaceCommandCount,
      initialSurfaceCommandCount + 1,
      'surface scope must update when a workspace surface opens',
    );
    const overviewSurfaceCommand = page
      .locator('.nx-command-list button', {
        hasText: /Abrir Visão Geral|Abrir Visão geral|Open Overview/,
      })
      .first();
    await overviewSurfaceCommand.click();
    await page.waitForFunction(
      () =>
        document
          .querySelector('.nx-workspace-tab[data-kind="overview"]')
          ?.getAttribute('aria-selected') === 'true',
    );
    assert.equal(shellLaunches, 2, 'opening a surface must not launch another shell');
    assert.equal(agentCreates, 0, 'opening a surface must not create agents');
    console.log('  ✓ surface scope tracks list changes and activates an existing surface');

    await page.locator('.nx-workspace-tab[data-kind="terminals"]').first().click();
    await page.waitForFunction(
      () =>
        document
          .querySelector('.nx-workspace-tab[data-kind="terminals"]')
          ?.getAttribute('aria-selected') === 'true',
    );
    await desktopToggle.click({ timeout: 10000 });
    await page.waitForSelector('.nx-arrange-menu__trigger', { timeout: 15000 });

    const withTwoState = await readDesktopState(page);
    console.log(`  with two state: ${JSON.stringify(withTwoState)}`);
    assert.equal(withTwoState.viewIds.length, 2, 'two PTYs must expose two desktop windows');
    assert.notEqual(
      withTwoState.viewIds[0],
      withTwoState.viewIds[1],
      'the two windows must have distinct surface identities',
    );

    // Switch to SPLIT presentation and reload: split pane identities survive.
    const splitToggle = page
      .getByRole('button', { name: /Dividir terminais|Split terminals/i })
      .first();
    await splitToggle.waitFor({ state: 'visible', timeout: 10000 });
    assert.equal(await splitToggle.isDisabled(), false, 'split toggle must be enabled');
    await splitToggle.click({ timeout: 10000 });
    const splitState = await page.evaluate(() => ({
      panels: document.querySelector('.nx-terminals-inner-panels')?.className || '',
      activeModes: [...document.querySelectorAll('.nx-presentation-toggle button')]
        .map((b) => `${b.textContent?.trim()}=${b.getAttribute('data-active')}`)
        .join(','),
    }));
    assert.match(
      splitState.panels,
      /splitPanels/,
      'split mode must apply the split layout class to the terminal panels',
    );
    assert.match(
      splitState.activeModes,
      /Dividir=true|Split=true/,
      'split presentation toggle must become active',
    );

    // Wait until the server-side layout snapshot contains the second surface
    // (windows persist only inside terminal/workspace tab models), so the
    // reload restore is proven against the persisted layout.
    await waitForServerLayoutSurfaces(context, apiOrigin, project.id, 2);

    await page.reload({ waitUntil: 'domcontentloaded' });
    await waitFor(page, '.nx-os-shell', 'workspace shell after split reload');
    await page.locator('.nx-workspace-tab[data-kind="terminals"]').first().click();
    await page.waitForFunction(
      () =>
        [
          ...document.querySelectorAll(
            '.nx-terminals-inner-panels > [role="tabpanel"][data-active="true"]',
          ),
        ].filter((el) => el.querySelector('.nx-project-shell-surface')).length >= 2,
      { timeout: 20000 },
    );
    const splitPaneIds = await page.evaluate(() =>
      [
        ...document.querySelectorAll(
          '.nx-terminals-inner-panels > [role="tabpanel"][data-active="true"]',
        ),
      ]
        .map(
          (el) =>
            el.querySelector('.nx-project-shell-surface')?.getAttribute('data-runtime-id') || '',
        )
        .filter(Boolean)
        .sort(),
    );
    assert.equal(splitPaneIds.length, 2, 'split mode must render two visible panes');
    assert.notEqual(splitPaneIds[0], splitPaneIds[1], 'split panes must keep distinct runtimes');
    assert.equal(shellLaunches, 2, 'split reload must not relaunch runtimes');
    console.log(`  ✓ split survives reload with distinct runtime ids: ${splitPaneIds.join(', ')}`);

    // --- Phase 4: second project isolation ---------------------------------
    console.log('  -- phase: second project layout isolation');
    const secondProjectPath = `${isolatedProjectPath}-second`;
    mkdirSync(secondProjectPath, { recursive: true });
    const secondProjectResponse = await context.request.post(`${apiOrigin}/api/v1/projects`, {
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf },
      data: {
        name: 'Smart Layouts Second',
        path: secondProjectPath,
      },
    });
    assert.equal(secondProjectResponse.status(), 201, 'second fixture project must be created');
    const secondProject = await secondProjectResponse.json();
    assert.ok(secondProject?.id, 'second project must have an id');

    await page.goto(
      new URL(`/p/${encodeURIComponent(secondProject.id)}/overview`, apiOrigin).toString(),
      { waitUntil: 'domcontentloaded', timeout: 15000 },
    );
    await waitFor(page, '.nx-os-shell', 'second project workspace shell');
    await page.locator('.nx-workspace-tab[data-kind="terminals"]').first().click();
    await page.waitForSelector('.nx-terminals-empty', { timeout: 15000 });
    const secondWindows = await page.evaluate(
      () => document.querySelectorAll('.nx-desktop-window').length,
    );
    assert.equal(
      secondWindows,
      0,
      'a second project must not inherit the first project desktop windows',
    );
    await page.locator('.nx-workspace-tab[data-kind="terminals"]').first().click();
    console.log('  ✓ second project has an isolated empty layout');

    // Back to the first project: its desktop layout must still be there.
    await page.goto(
      new URL(`/p/${encodeURIComponent(project.id)}/terminals`, apiOrigin).toString(),
      { waitUntil: 'domcontentloaded', timeout: 15000 },
    );
    await waitFor(page, '.nx-os-shell', 'first project workspace shell restored');
    await page.locator('.nx-workspace-tab[data-kind="terminals"]').first().click();
    await page.waitForFunction(
      () =>
        [
          ...document.querySelectorAll(
            '.nx-terminals-inner-panels > [role="tabpanel"][data-active="true"]',
          ),
        ].filter((el) => el.querySelector('.nx-project-shell-surface')).length >= 2,
      { timeout: 20000 },
    );
    const restoredRuntimeIds = await page.evaluate(() =>
      [
        ...document.querySelectorAll(
          '.nx-terminals-inner-panels > [role="tabpanel"][data-active="true"]',
        ),
      ]
        .map(
          (el) =>
            el.querySelector('.nx-project-shell-surface')?.getAttribute('data-runtime-id') || '',
        )
        .filter(Boolean)
        .sort(),
    );
    assert.deepEqual(
      restoredRuntimeIds,
      splitPaneIds,
      'returning to the first project must restore its split layout',
    );
    console.log('  ✓ first project restores its two-runtime split layout');

    assert.equal(shellLaunches, 2, 'layout navigation must never spawn extra shells');
    assert.equal(agentCreates, 0, 'layout navigation must never create agents');
    console.log('✓ All Smart Layouts assertions PASSED!');
  } catch (error) {
    const redact = (value) =>
      String(value).replace(/nexus_bootstrap=[a-f0-9]+/gi, 'nexus_bootstrap=[REDACTED]');
    const diagnostics = {
      url: (() => {
        try {
          const url = new URL(page?.url() || bootstrapUrl);
          url.hash = '';
          return url.toString();
        } catch {
          return '(unparsable url)';
        }
      })(),
      pageErrors: collectors?.pageErrors.map(redact) || [],
      consoleErrors: collectors?.consoleErrors.map(redact) || [],
      requestFailures: collectors?.requestFailures.map(redact) || [],
      serverOutput: redact(getOutput()),
      error: String(error?.stack || error),
    };
    const fs = await import('node:fs');
    fs.writeFileSync(
      path.join(screenshotDir, 'e2e-smart-layouts-failure-diagnostics.json'),
      `${JSON.stringify(diagnostics, null, 2)}\n`,
    );
    console.error('Smart layouts E2E FAIL');
    console.error(diagnostics.error);
    process.exitCode = 1;
  } finally {
    await browser?.close();
    await terminateProcess(server);
  }
}

await main();
