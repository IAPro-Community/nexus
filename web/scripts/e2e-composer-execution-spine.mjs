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

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '../..');
const outDir = path.join(repoRoot, 'DEV/validation/composer-execution-spine');
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

  const goal = page.getByPlaceholder(/main goal|objetivo principal/i);
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

  await page.screenshot({ path: path.join(outDir, 'north-star.png'), fullPage: true });
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
    if (fixture?.isolatedDataDir && !process.env.NEXUS_E2E_DATA_DIR) {
      try {
        rmSync(fixture.isolatedDataDir, { recursive: true, force: true });
      } catch (_) {}
    }
  });
