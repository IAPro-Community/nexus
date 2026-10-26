#!/usr/bin/env node
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
  path.resolve(repoRoot, '.tempmediaStorage', 'mission-attention');
if (!existsSync(screenshotDir)) mkdirSync(screenshotDir, { recursive: true });

function attentionItem(projectId, missionId, overrides = {}) {
  return {
    mission_id: missionId,
    project_id: projectId,
    state: 'BLOCKED_NEEDS_USER',
    level: 'REQUIRE_USER',
    reason_code: 'INTERVENTION_REQUIRED',
    summary: `${missionId} review`,
    question: 'Choose the safe next step',
    impact: 'Execution is paused',
    recommended_actions: ['Review mission'],
    age_seconds: 30,
    ...overrides,
  };
}

function attentionGroup(items) {
  const needsYou = items.filter((item) => item.level === 'REQUIRE_USER');
  const completed = items.filter((item) => item.state === 'COMPLETED_VERIFIED');
  const failed = items.filter((item) => item.state === 'FAILED_NO_PROGRESS');
  return {
    needs_you: needsYou,
    completed,
    failed,
    all_items: items,
    total_needs: needsYou.length,
  };
}

async function main() {
  console.log('=== Mission Attention browser E2E ===');
  const { server, bootstrapUrl, isolatedProjectPath, getOutput } = await startFixtureServer();
  let browser;
  let context;
  let page;
  let collectors;
  let drawerOpened = false;
  let initialDrawerResponseReleased = false;
  let releaseInitialDrawerResponse = () => {};
  const initialDrawerGate = new Promise((resolve) => {
    releaseInitialDrawerResponse = resolve;
  });
  try {
    ({ browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl));
    const { project, apiOrigin } = await seedFixtureProject(
      context,
      bootstrapUrl,
      isolatedProjectPath,
    );

    const alpha = { id: project.id, name: 'Attention Alpha', path: isolatedProjectPath };
    const beta = {
      id: 'fixture-project-beta',
      name: 'Attention Beta',
      path: `${isolatedProjectPath}-beta`,
    };
    const gamma = {
      id: 'fixture-project-gamma',
      name: 'Attention Gamma',
      path: `${isolatedProjectPath}-gamma`,
    };
    const needsUser = attentionItem(alpha.id, 'mission-alpha-needs-user');
    const completed = attentionItem(beta.id, 'mission-beta-completed', {
      state: 'COMPLETED_VERIFIED',
      level: 'IN_APP',
      question: '',
    });
    const failed = attentionItem(beta.id, 'mission-beta-failed', {
      state: 'FAILED_NO_PROGRESS',
      level: 'IN_APP',
      question: '',
    });
    const missionPlan = {
      id: 'plan-attention-alpha',
      project_id: alpha.id,
      title: 'Attention Alpha mission',
      description: 'Mission detail fixture for the Attention Center navigation flow.',
      status: 'EXECUTING',
      current_revision: 1,
      phases: [],
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    const missionRun = {
      id: needsUser.mission_id,
      plan_id: missionPlan.id,
      plan_revision: 1,
      project_id: alpha.id,
      workspace: isolatedProjectPath,
      state: 'BLOCKED_NEEDS_USER',
      current_pkg_index: 0,
      total_iterations: 0,
      package_runs: [],
      started_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      needs_human: {
        id: 'intervention-alpha',
        reason_code: 'INTERVENTION_REQUIRED',
        summary: needsUser.summary,
        question: needsUser.question,
        context: 'Controlled mission fixture',
        recommended_actions: needsUser.recommended_actions,
        impact: needsUser.impact,
        mission_id: needsUser.mission_id,
        task_id: 'task-alpha',
        source: 'e2e-fixture',
        timestamp: new Date().toISOString(),
        version: 1,
        scope: 'PACKAGE',
        options: [{ id: 'retry-safe', operation: 'RETRY_SAFE_PACKAGE', label: 'Retry safely' }],
        resolved: false,
      },
    };
    const allItems = [needsUser, completed, failed];
    const requests = [];
    let failMissionFilter = false;

    await context.route('**/api/v1/projects', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([alpha, beta, gamma]),
      }),
    );
    await context.route('**/api/v1/attention**', async (route) => {
      const url = new URL(route.request().url());
      const projectId = url.searchParams.get('project_id') || '';
      const missionId = url.searchParams.get('mission_id') || '';
      requests.push({ projectId, missionId });

      if (drawerOpened && !projectId && !missionId && !initialDrawerResponseReleased) {
        await initialDrawerGate;
      }
      if (failMissionFilter && missionId) {
        await route.fulfill({
          status: 503,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'attention fixture temporarily unavailable' }),
        });
        return;
      }

      const visibleItems = allItems.filter(
        (item) =>
          (!projectId || item.project_id === projectId) &&
          (!missionId || item.mission_id === missionId),
      );
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(attentionGroup(visibleItems)),
      });
    });
    await context.route('**/api/v1/runs/**', async (route) => {
      const pathname = new URL(route.request().url()).pathname;
      const runPath = `/api/v1/runs/${encodeURIComponent(needsUser.mission_id)}`;
      if (pathname === runPath) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(missionRun),
        });
      } else if (pathname === `${runPath}/evidence`) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ run_id: needsUser.mission_id, capsules: [], receipts: [] }),
        });
      } else if (pathname === `${runPath}/evidence-bundle`) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: 'null' });
      } else {
        await route.continue();
      }
    });
    await context.route('**/api/v1/plans/plan-attention-alpha', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ plan: missionPlan, revisions: [] }),
      }),
    );

    const shellUrl = new URL(`/p/${encodeURIComponent(project.id)}/overview`, apiOrigin);
    await page.goto(shellUrl.toString(), { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('.nx-os-shell', { timeout: 10000 });

    const openCenter = page.getByRole('button', { name: /central de notificações e radar/i });
    drawerOpened = true;
    await openCenter.click();
    const drawer = page.getByRole('dialog');
    await drawer.waitFor({ state: 'visible', timeout: 5000 });
    const loading = drawer.getByRole('status').filter({ hasText: /carregando/i });
    await loading.waitFor({ state: 'visible', timeout: 3000 });
    initialDrawerResponseReleased = true;
    releaseInitialDrawerResponse();
    await loading.waitFor({ state: 'hidden', timeout: 5000 });

    const panel = drawer.locator('.nx-notification-drawer__panel').first();
    const missionCard = (missionId) => panel.locator('[data-level]').filter({ hasText: missionId });
    await missionCard('mission-alpha-needs-user').waitFor({ state: 'visible' });
    await missionCard('mission-beta-completed').waitFor({ state: 'visible' });
    await missionCard('mission-beta-failed').waitFor({ state: 'visible' });
    await page.screenshot({ path: path.join(screenshotDir, 'mission-attention-desktop.png') });

    const projectFilter = drawer.locator('#mission-attention-project-filter');
    const missionFilter = drawer.locator('#mission-attention-mission-filter');
    await projectFilter.focus();
    await page.keyboard.press('Tab');
    assert.equal(
      await missionFilter.evaluate((element) => document.activeElement === element),
      true,
      'Tab must move from the project filter to the mission filter',
    );
    await projectFilter.selectOption(beta.id);
    await missionCard('mission-beta-completed').waitFor({ state: 'visible' });
    await missionCard('mission-alpha-needs-user').waitFor({ state: 'hidden' });
    assert.ok(
      requests.some((request) => request.projectId === beta.id && request.missionId === ''),
      'project selection must request the server-side project filter',
    );

    await missionFilter.selectOption(failed.mission_id);
    await missionCard('mission-beta-failed').waitFor({ state: 'visible' });
    await missionCard('mission-beta-completed').waitFor({ state: 'hidden' });
    assert.ok(
      requests.some(
        (request) => request.projectId === beta.id && request.missionId === failed.mission_id,
      ),
      'mission selection must intersect with the project filter',
    );

    await projectFilter.selectOption(gamma.id);
    await drawer
      .getByText('Nenhuma missão precisa de atenção agora.')
      .waitFor({ state: 'visible' });
    assert.equal(
      await panel.locator('[data-level]').count(),
      0,
      'empty projection must not retain old cards',
    );

    await projectFilter.selectOption(beta.id);
    await missionCard('mission-beta-completed').waitFor({ state: 'visible' });
    failMissionFilter = true;
    await missionFilter.selectOption(failed.mission_id);
    await drawer
      .getByRole('alert')
      .filter({ hasText: /temporarily unavailable/i })
      .waitFor({ state: 'visible' });
    assert.equal(
      await panel.locator('[data-level]').count(),
      0,
      'an API error must not present stale mission cards as if they were current',
    );

    failMissionFilter = false;
    await projectFilter.selectOption(alpha.id);
    await missionCard('mission-alpha-needs-user').waitFor({ state: 'visible' });
    const alphaMission = missionCard('mission-alpha-needs-user');
    await alphaMission.click();
    await drawer.waitFor({ state: 'hidden', timeout: 5000 });
    await page.waitForFunction(
      (missionId) =>
        document.querySelector('.nx-workspace-tab[data-kind="flow-run"][aria-selected="true"]') &&
        window.location.pathname.endsWith(`/missions/${encodeURIComponent(missionId)}`),
      needsUser.mission_id,
      { timeout: 10000 },
    );
    const openedMission = page.locator(
      `.nx-flow-run-surface[data-run-id="${needsUser.mission_id}"]`,
    );
    await openedMission.waitFor({ state: 'visible', timeout: 10000 });
    await openedMission.getByText(needsUser.question).waitFor({ state: 'visible' });
    await page.screenshot({
      path: path.join(screenshotDir, 'mission-attention-opened-mission.png'),
    });

    await page.setViewportSize({ width: 390, height: 844 });
    await openCenter.click();
    await page.getByRole('dialog').waitFor({ state: 'visible' });
    const mobileDrawer = page.getByRole('dialog');
    await page.waitForFunction(
      () => {
        const drawer = document.querySelector('[role="dialog"]');
        return (
          Boolean(drawer) &&
          drawer.getAnimations().every((animation) => animation.playState !== 'running')
        );
      },
      undefined,
      { timeout: 3000 },
    );
    const mobileBox = await mobileDrawer.boundingBox();
    await page.screenshot({ path: path.join(screenshotDir, 'mission-attention-mobile.png') });
    assert.ok(
      mobileBox && mobileBox.x >= 0 && mobileBox.x + mobileBox.width <= 390.5,
      `drawer must fit the 390px viewport: ${JSON.stringify(mobileBox)}`,
    );
    const mobileDocumentWidth = await page.evaluate(() => document.documentElement.scrollWidth);
    assert.ok(
      mobileDocumentWidth <= 390,
      `page must not scroll horizontally at 390px: ${mobileDocumentWidth}px`,
    );
    const mobileFilterColumns = await mobileDrawer
      .locator('.nx-notification-drawer__panel')
      .first()
      .locator('select')
      .evaluateAll((elements) => elements.map((element) => element.getBoundingClientRect().width));
    assert.equal(
      mobileFilterColumns.length,
      2,
      'both mission filters must remain available on mobile',
    );
    assert.ok(mobileFilterColumns.every((width) => width > 0 && width < 390));

    const requestFailures = meaningfulFixtureRequestFailures(collectors.requestFailures);
    assert.equal(
      requestFailures.length,
      0,
      `unexpected browser request failures: ${requestFailures.join(' | ')}`,
    );
    assert.equal(
      collectors.pageErrors.length,
      0,
      `page errors: ${collectors.pageErrors.join(' | ')}`,
    );
    console.log(`Mission Attention browser E2E: PASS (${requests.length} attention requests)`);
  } catch (error) {
    console.error(`Mission Attention fixture server output:\n${getOutput()}`);
    throw error;
  } finally {
    initialDrawerResponseReleased = true;
    releaseInitialDrawerResponse?.();
    if (context) await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
    await terminateProcess(server);
  }
}

main().catch((error) => {
  console.error(`Mission Attention browser E2E: FAIL: ${error?.stack || error}`);
  process.exit(1);
});
