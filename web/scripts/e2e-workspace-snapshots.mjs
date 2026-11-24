#!/usr/bin/env node
// Named workspace snapshot UI E2E.
// Proves project scoping and that logical restoration never launches a runtime.
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
const artifactDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR ||
  path.resolve(repoRoot, '.tempmediaStorage', 'workspace-snapshots');
if (!existsSync(artifactDir)) mkdirSync(artifactDir, { recursive: true });

async function waitForProjectShell(page, runtimeId) {
  await page.waitForFunction(
    (expected) =>
      [...document.querySelectorAll('.nx-project-shell-surface[data-runtime-id]')].some(
        (surface) => surface.getAttribute('data-runtime-id') === expected,
      ),
    runtimeId,
    { timeout: 20000 },
  );
}

async function openSnapshots(page) {
  await page.locator('[data-tour="workspace-snapshots"]').click();
  const dialog = page.getByRole('dialog');
  await dialog.waitFor({ state: 'visible', timeout: 10000 });
  return dialog;
}

async function main() {
  console.log('=== Workspace named snapshots E2E ===');
  const { server, bootstrapUrl, isolatedProjectPath, getOutput } = await startFixtureServer();
  let browser;
  let context;
  let page;
  let collectors;
  let launches = 0;
  let agentCreates = 0;
  try {
    ({ browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl));
    const { project, apiOrigin } = await seedFixtureProject(
      context,
      bootstrapUrl,
      isolatedProjectPath,
    );

    page.on('request', (request) => {
      if (request.method() !== 'POST') return;
      const pathname = new URL(request.url()).pathname;
      if (
        pathname === `/api/v1/projects/${project.id}/shell` ||
        pathname === '/api/v1/runtimes' ||
        /^\/api\/v1\/agents\/[^/]+\/(start|ask|recover)$/.test(pathname)
      ) {
        launches += 1;
      }
      if (pathname === `/api/v1/projects/${project.id}/agents`) agentCreates += 1;
    });

    await page.goto(
      new URL(`/p/${encodeURIComponent(project.id)}/overview`, apiOrigin).toString(),
      {
        waitUntil: 'domcontentloaded',
        timeout: 15000,
      },
    );
    await page.waitForSelector('.nx-os-shell', { timeout: 15000 });
    await page.locator('.nx-workspace-tab[data-kind="terminals"]').first().click();
    const openTerminal = page
      .getByRole('button', { name: /Abrir Terminal|Open Terminal|Terminal do Projeto/i })
      .first();
    await openTerminal.waitFor({ state: 'visible', timeout: 15000 });
    await openTerminal.click();
    const shell = page.locator('.nx-project-shell-surface[data-runtime-id]').first();
    await shell.waitFor({ state: 'attached', timeout: 20000 });
    const runtimeId = await shell.getAttribute('data-runtime-id');
    assert.ok(runtimeId, 'the explicit terminal action must expose a runtime id');
    assert.equal(launches, 1, 'one explicit shell action must produce one launch');

    const dialog = await openSnapshots(page);
    await dialog
      .getByLabel(/Snapshot name|Nome do snapshot|Nombre del snapshot/i)
      .fill('Logical layout checkpoint');
    await dialog
      .getByRole('button', { name: /Save snapshot|Salvar snapshot|Guardar snapshot/i })
      .click();
    const snapshotRow = dialog.getByRole('list').getByText('Logical layout checkpoint');
    await snapshotRow.waitFor({ state: 'visible', timeout: 10000 });
    console.log('  ✓ save and list named snapshot from the project taskbar');
    await page.keyboard.press('Escape');
    await dialog.waitFor({ state: 'detached', timeout: 10000 });

    // Restore while the captured runtime is still active.
    await page.locator('.nx-workspace-tab[data-kind="overview"]').first().click();
    await page.waitForFunction(
      () =>
        document
          .querySelector('.nx-workspace-tab[aria-selected="true"]')
          ?.getAttribute('data-kind') === 'overview',
    );
    const activeRestoreDialog = await openSnapshots(page);
    const activeRow = activeRestoreDialog.getByRole('listitem').filter({
      hasText: 'Logical layout checkpoint',
    });
    await activeRow.getByRole('button', { name: /Restore|Restaurar/i }).click();
    await activeRestoreDialog.waitFor({ state: 'detached', timeout: 10000 });
    await page.waitForFunction(
      () =>
        document
          .querySelector('.nx-workspace-tab[aria-selected="true"]')
          ?.getAttribute('data-kind') === 'terminals',
      undefined,
      { timeout: 10000 },
    );
    await waitForProjectShell(page, runtimeId);
    assert.equal(launches, 1, 'restoring an active runtime must not relaunch it');
    assert.equal(agentCreates, 0, 'restoring a workspace must not create agents');
    console.log('  ✓ active runtime restored with the same identity and no extra launch');

    const sessionResponse = await context.request.get(`${apiOrigin}/api/v1/session`);
    assert.equal(sessionResponse.status(), 200, 'fixture session must be available');
    const { csrf_token: csrf } = await sessionResponse.json();
    const stopResponse = await context.request.post(
      `${apiOrigin}/api/v1/runtimes/${encodeURIComponent(runtimeId)}/stop`,
      { headers: { 'X-CSRF-Token': csrf } },
    );
    assert.equal(stopResponse.status(), 200, 'the captured runtime must be stopped');
    const stoppedResponse = await context.request.get(
      `${apiOrigin}/api/v1/runtimes/${encodeURIComponent(runtimeId)}`,
    );
    assert.equal(stoppedResponse.status(), 200, 'the stopped runtime record must remain readable');
    const stoppedDetail = await stoppedResponse.json();
    assert.equal(stoppedDetail.session?.state, 'STOPPED', 'fixture must confirm ended state');

    // Restore the same logical snapshot after the runtime has ended.
    await page.locator('.nx-workspace-tab[data-kind="overview"]').first().click();
    await page.waitForFunction(
      () =>
        document
          .querySelector('.nx-workspace-tab[aria-selected="true"]')
          ?.getAttribute('data-kind') === 'overview',
    );
    const endedRestoreDialog = await openSnapshots(page);
    const endedRow = endedRestoreDialog.getByRole('listitem').filter({
      hasText: 'Logical layout checkpoint',
    });
    await endedRow.getByRole('button', { name: /Restore|Restaurar/i }).click();
    await endedRestoreDialog.waitFor({ state: 'detached', timeout: 10000 });
    await page.waitForFunction(
      () =>
        document
          .querySelector('.nx-workspace-tab[aria-selected="true"]')
          ?.getAttribute('data-kind') === 'terminals',
      undefined,
      { timeout: 10000 },
    );
    await waitForProjectShell(page, runtimeId);
    assert.equal(launches, 1, 'restoring an ended runtime must not relaunch it');
    assert.equal(agentCreates, 0, 'restoring an ended runtime must not create agents');
    const finalDetail = await context.request.get(
      `${apiOrigin}/api/v1/runtimes/${encodeURIComponent(runtimeId)}`,
    );
    assert.equal((await finalDetail.json()).session?.state, 'STOPPED');
    console.log('  ✓ ended runtime remains stopped after restoring its logical surface');

    const deleteDialog = await openSnapshots(page);
    const deleteRow = deleteDialog.getByRole('listitem').filter({
      hasText: 'Logical layout checkpoint',
    });
    await deleteRow
      .getByRole('button', { name: /Delete snapshot|Excluir snapshot|Eliminar snapshot/i })
      .click();
    await deleteRow.waitFor({ state: 'detached', timeout: 10000 });
    console.log('  ✓ delete removes the named snapshot');

    // A second project must not see the first project’s local snapshot history.
    const secondPath = `${isolatedProjectPath}-second`;
    mkdirSync(secondPath, { recursive: true });
    const secondProjectResponse = await context.request.post(`${apiOrigin}/api/v1/projects`, {
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf },
      data: { name: 'Workspace snapshots second project', path: secondPath },
    });
    assert.equal(secondProjectResponse.status(), 201, 'second fixture project must be created');
    const secondProject = await secondProjectResponse.json();
    await page.goto(
      new URL(`/p/${encodeURIComponent(secondProject.id)}/overview`, apiOrigin).toString(),
      { waitUntil: 'domcontentloaded', timeout: 15000 },
    );
    await page.waitForSelector('.nx-os-shell', { timeout: 15000 });
    const exploreWorkspace = page.getByRole('button', {
      name: /Explore this area|Conhecer esta área|Conocer esta área/i,
    });
    await exploreWorkspace.waitFor({ state: 'visible', timeout: 10000 });
    await exploreWorkspace.click();
    await page.waitForSelector('.nx-tour-layer', { timeout: 10000 });
    const tourTitle = await page.locator('.nx-tour-card strong').innerText();
    assert.match(tourTitle, /workspace|layout|organização|diseño/i);
    await page.getByRole('button', { name: /Close tour|Fechar tour|Cerrar recorrido/i }).click();
    await page.waitForSelector('.nx-tour-layer', { state: 'detached', timeout: 10000 });
    console.log('  ✓ taskbar opens the scoped shared ProductTour for workspace snapshots');

    await page.setViewportSize({ width: 390, height: 844 });
    const mobileBounds = await page.evaluate(() => ({
      viewport: window.innerWidth,
      document: document.documentElement.scrollWidth,
      taskbar: document.querySelector('[data-feature-tour-scope="workspace"]')?.scrollWidth || 0,
      taskbarClient:
        document.querySelector('[data-feature-tour-scope="workspace"]')?.clientWidth || 0,
    }));
    assert.ok(
      mobileBounds.document <= mobileBounds.viewport + 2,
      `snapshot taskbar must not cause horizontal page overflow at 390px: ${JSON.stringify(mobileBounds)}`,
    );
    const secondDialog = await openSnapshots(page);
    await secondDialog
      .getByText(/No named snapshots|Ainda não há snapshots|Todavía no hay snapshots/i)
      .waitFor({ state: 'visible', timeout: 10000 });
    const dialogBounds = await secondDialog.boundingBox();
    assert.ok(dialogBounds, 'snapshot dialog must be visible at 390px');
    assert.ok(
      dialogBounds.x >= 0 && dialogBounds.x + dialogBounds.width <= 390,
      `snapshot dialog must fit 390px viewport: ${JSON.stringify(dialogBounds)}`,
    );
    await page.keyboard.press('Escape');
    await secondDialog.waitFor({ state: 'detached', timeout: 10000 });
    assert.equal(launches, 1, 'project isolation must not launch a process');
    assert.deepEqual(meaningfulFixtureRequestFailures(collectors.requestFailures), []);
    console.log('  ✓ named snapshots remain isolated to their project');
    console.log('✓ All workspace named snapshots assertions PASSED');
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
    writeFileSync(
      path.join(artifactDir, 'e2e-workspace-snapshots-failure-diagnostics.json'),
      `${JSON.stringify(diagnostics, null, 2)}\n`,
    );
    console.error('Workspace named snapshots E2E FAIL');
    console.error(diagnostics.error);
    process.exitCode = 1;
  } finally {
    await browser?.close();
    await terminateProcess(server);
  }
}

await main();
