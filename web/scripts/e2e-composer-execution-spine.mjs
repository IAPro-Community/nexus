#!/usr/bin/env node
import assert from 'node:assert/strict';
import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  launchFixtureBrowser,
  seedFixtureProject,
  startFixtureServer,
  terminateProcess,
} from './e2e-fixture.mjs';
import { startComposerIntelligenceMock, configureComposerIntelligence } from './e2e-composer-intelligence-mock.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '../..');
const outDir = path.resolve(
  process.env.NEXUS_E2E_ARTIFACTS_DIR || path.join(repoRoot, 'DEV/validation/composer-execution-spine'),
);
mkdirSync(outDir, { recursive: true });

const completePrompt = `Implement OAuth login with Google for the existing React dashboard and Go API.
Scope: implement the backend OAuth callback and the frontend sign-in/session handling.
Security requirements: OAuth state validation, PKCE, HttpOnly SameSite cookies, no secrets in logs.
Out of scope: additional identity providers and deployment changes.
Success means a user can sign in with Google, refresh the dashboard, and keep the authenticated session.
Testing: add Go unit/integration tests, React tests, and browser regression coverage.
Verification: run go test ./..., frontend tests, and the browser authentication scenario.
Independent review must check authentication, authorization, secret handling, and regressions.
Definition of Done: implementation, tests, review findings resolved, and verification evidence recorded.
Deliverables: source changes, tests, review evidence, and verification evidence.
Do not push, merge, deploy, or weaken existing authentication controls.`;

let fixture;
let browser;
let intelligenceMock;

async function getJSON(context, url) {
  const response = await context.request.get(url, { headers: { Accept: 'application/json' } });
  assert.equal(response.status(), 200, `GET ${url} returned HTTP ${response.status()}`);
  return response.json();
}

async function main() {
  fixture = await startFixtureServer({ composerFlow: true });
  const launched = await launchFixtureBrowser(fixture.bootstrapUrl);
  browser = launched.browser;
  const { context, page, collectors } = launched;
  page.setDefaultTimeout(25000);

  const { project, apiOrigin } = await seedFixtureProject(
    context,
    fixture.bootstrapUrl,
    fixture.isolatedProjectPath,
  );

  const session = await getJSON(context, `${apiOrigin}/api/v1/session`);
  assert.ok(session.csrf_token, 'authenticated fixture must expose a CSRF token');
  intelligenceMock = await startComposerIntelligenceMock();
  const intelligence = await configureComposerIntelligence(
    context, apiOrigin, session.csrf_token, project.id, intelligenceMock,
  );
  assert.equal(intelligence.available, true, 'mock Intelligence must be available before Composer');

  const prepare = await context.request.post(
    `${apiOrigin}/api/v1/projects/${project.id}/context/prepare`,
    {
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'X-CSRF-Token': session.csrf_token,
      },
      data: { create_context: true },
    },
  );
  assert.equal(prepare.status(), 200, `context prepare returned HTTP ${prepare.status()}`);
  const readiness = await prepare.json();
  assert.equal(readiness.state, 'READY', `Composer context must be READY: ${JSON.stringify(readiness)}`);

  await page.goto(`${apiOrigin}/p/${encodeURIComponent(project.id)}/work`, {
    waitUntil: 'domcontentloaded',
  });

  await page.getByRole('button', { name: /existing prompt|prompt existente/i }).click();

  const goal = page.locator('[data-tour="composer-goal"] textarea');
  await goal.fill('Implement Google OAuth login safely');

  const source = page.getByPlaceholder(/original prompt|prompt original/i);
  await source.fill(completePrompt);

  await page.getByRole('button', { name: /analyze prompt|analisar prompt/i }).click();

  const reviewFinalize = page.getByRole('button', {
    name: /review and finalize|revisar e finalizar/i,
  });
  await reviewFinalize.waitFor({ state: 'visible', timeout: 45000 });

  const forcedGapButton = page.getByRole('button', {
    name: /finish with confirmed gaps|finalizar.*lacunas|concluir.*lacunas/i,
  });
  assert.equal(
    await forcedGapButton.count(),
    0,
    'complete prompt must not require the confirmed-gaps escape hatch',
  );

  await reviewFinalize.click();

  const turnIntoFlow = page.getByRole('button', {
    name: /turn into flow|transformar.*flow|converter.*flow/i,
  });
  await turnIntoFlow.waitFor({ state: 'visible', timeout: 30000 });

  const composerSessions = await getJSON(
    context,
    `${apiOrigin}/api/v1/projects/${project.id}/composer-sessions`,
  );
  assert.ok(Array.isArray(composerSessions) && composerSessions.length > 0, 'Composer session must persist');

  await turnIntoFlow.click();

  const executionPlan = page.getByRole('button', {
    name: /plano de execução|execution plan/i,
  });
  await executionPlan.waitFor({ state: 'visible', timeout: 30000 });

  const runsBefore = await getJSON(context, `${apiOrigin}/api/v1/runs`);
  const agentsBefore = await getJSON(context, `${apiOrigin}/api/v1/projects/${project.id}/agents`);

  const previewResponsePromise = page.waitForResponse(
    (response) =>
      response.url().includes('/execution-preview') &&
      response.request().method() === 'POST',
    { timeout: 30000 },
  );
  await executionPlan.click();
  const previewResponse = await previewResponsePromise;
  assert.equal(previewResponse.status(), 200, 'Execution Preview API must succeed');
  const preview = await previewResponse.json();

  const untranslatedProviderReason = page
    .locator('small')
    .filter({ hasText: /no provider accounts discovered/i });
  assert.equal(
    await untranslatedProviderReason.count(),
    0,
    'provider discovery blockers must not leak duplicate English detail into the localized preview',
  );
  const untranslatedCostBudget = page
    .locator('small')
    .filter({ hasText: /no local cost budget is configured/i });
  assert.equal(
    await untranslatedCostBudget.count(),
    0,
    'the missing local cost budget message must use the active interface language',
  );

  assert.equal(preview.project_id, project.id, 'preview must belong to the fixture project');
  assert.ok(Number(preview.revision) >= 1, 'preview must be bound to a persisted revision');
  assert.ok(Array.isArray(preview.steps) && preview.steps.length >= 4, 'complex prompt needs a multi-step Flow');
  assert.ok(
    preview.steps.every((step) => step.agent?.reason && step.resource?.reason),
    'every step must explain Agent and resource decisions',
  );

  await page
    .getByText(/sem iniciar Agents ou runtimes|without starting Agents or runtimes/i)
    .waitFor({ state: 'visible', timeout: 10000 });

  const runsAfter = await getJSON(context, `${apiOrigin}/api/v1/runs`);
  const agentsAfter = await getJSON(context, `${apiOrigin}/api/v1/projects/${project.id}/agents`);
  assert.equal(
    runsAfter.length,
    runsBefore.length,
    'Execution Preview must not create a Mission run',
  );
  assert.equal(
    agentsAfter.length,
    agentsBefore.length,
    'Execution Preview must not create persistent Agents',
  );

  // In the isolated browser fixture no provider profiles are installed by the
  // test. The truthful result is blocked/unknown, never fabricated capacity.
  if (!preview.ready) {
    assert.ok(
      Array.isArray(preview.blockers) && preview.blockers.length > 0,
      'not-ready preview must explain blockers',
    );
  }

  const firstStep = page.locator('.nx-composer-flow-body [role="list"] button').first();
  await firstStep.click();
  const goalEditor = page.locator('.nx-flow-inspector').getByLabel(/goal|objetivo/i);
  const originalGoal = await goalEditor.inputValue();
  const semanticEdit = 'NEXUS_E2E_SEMANTIC_REVISION_SENTINEL';
  await goalEditor.fill(`${originalGoal}\n${semanticEdit}`);

  const planUpdatePromise = page.waitForResponse(
    (response) =>
      response.url().includes(`/api/v1/plans/${preview.plan_id}`) &&
      response.request().method() === 'PUT',
    { timeout: 30000 },
  );
  const updatedPreviewPromise = page.waitForResponse(
    (response) =>
      response.url().includes(`/api/v1/plans/${preview.plan_id}/execution-preview`) &&
      response.request().method() === 'POST',
    { timeout: 30000 },
  );
  await page.getByRole('button', { name: /approve.*run|aprovar.*executar/i }).click();

  const planUpdateResponse = await planUpdatePromise;
  assert.equal(planUpdateResponse.status(), 200, 'semantic WorkPlan update must persist');
  const updatedPreviewResponse = await updatedPreviewPromise;
  assert.equal(updatedPreviewResponse.status(), 200, 'updated revision preview must succeed');
  const updatedPreview = await updatedPreviewResponse.json();
  const planDetail = await getJSON(context, `${apiOrigin}/api/v1/plans/${preview.plan_id}`);
  assert.equal(
    planDetail.plan.current_revision,
    Number(preview.revision) + 1,
    'semantic Flow edit must create a new WorkPlan revision before preview',
  );
  assert.equal(
    updatedPreview.revision,
    planDetail.plan.current_revision,
    'updated Execution Preview must bind to the persisted semantic revision',
  );
  assert.notEqual(
    updatedPreview.approval_token,
    preview.approval_token,
    'semantic revision must invalidate the prior approval token',
  );
  const savedRevision = (planDetail.revisions || []).find(
    (revision) => Number(revision.revision) === Number(updatedPreview.revision),
  );
  assert.ok(savedRevision, 'persisted plan history must contain the edited revision');
  const savedPlanSnapshot = JSON.parse(savedRevision.snapshot_json);
  const savedGoals = (savedPlanSnapshot.phases || []).flatMap((phase) =>
    (phase.packages || []).map((item) => item.goal || ''),
  );
  assert.ok(
    savedGoals.some((goal) => goal.includes(semanticEdit)),
    'edited Flow goal must be present in the immutable persisted revision snapshot',
  );
  const runsAfterSemanticEdit = await getJSON(context, `${apiOrigin}/api/v1/runs`);
  const agentsAfterSemanticEdit = await getJSON(
    context,
    `${apiOrigin}/api/v1/projects/${project.id}/agents`,
  );
  assert.equal(
    runsAfterSemanticEdit.length,
    runsBefore.length,
    'editing and refreshing a blocked preview must not create a Mission run',
  );
  assert.equal(
    agentsAfterSemanticEdit.length,
    agentsBefore.length,
    'editing and refreshing a blocked preview must not create persistent Agents',
  );

  await page.screenshot({ path: path.join(outDir, 'north-star.png'), fullPage: true });
  await page.screenshot({ path: path.join(outDir, 'semantic-revision.png'), fullPage: true });
  writeFileSync(
    path.join(outDir, 'north-star.json'),
    JSON.stringify(
      {
        project_id: project.id,
        plan_id: preview.plan_id,
        revision: preview.revision,
        preview_ready: preview.ready,
        step_count: preview.steps.length,
        blockers: preview.blockers || [],
        semantic_edit: {
          persisted: true,
          plan_revision: planDetail.plan.current_revision,
          preview_revision: updatedPreview.revision,
          stale_approval_token_replaced:
            updatedPreview.approval_token !== preview.approval_token,
          sentinel_persisted: savedGoals.some((goal) => goal.includes(semanticEdit)),
          mission_runs_created: runsAfterSemanticEdit.length - runsBefore.length,
          agents_created: agentsAfterSemanticEdit.length - agentsBefore.length,
          preview_ready: updatedPreview.ready,
          blockers: updatedPreview.blockers || [],
        },
        console_errors: collectors.consoleErrors,
        page_errors: collectors.pageErrors,
        request_failures: collectors.requestFailures,
      },
      null,
      2,
    ),
  );

  assert.deepEqual(collectors.pageErrors, [], 'browser page errors must remain empty');
  console.log(
    `composer execution spine e2e PASS · plan=${preview.plan_id} revision=${preview.revision} steps=${preview.steps.length} ready=${preview.ready}`,
  );
}

main()
  .catch(async (error) => {
    console.error('composer execution spine e2e FAIL');
    console.error(error);
    if (fixture) {
      try {
        writeFileSync(
          path.join(outDir, 'failure-server.log'),
          String(fixture.getOutput?.() || ''),
        );
      } catch (_) {}
    }
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
    if (intelligenceMock) {
      try { await intelligenceMock.close(); } catch (_) {}
    }
    if (fixture?.isolatedDataDir && !process.env.NEXUS_E2E_DATA_DIR) {
      try {
        rmSync(fixture.isolatedDataDir, { recursive: true, force: true });
      } catch (_) {}
    }
  });
