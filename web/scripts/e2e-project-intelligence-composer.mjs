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
const outDir = path.join(
  repoRoot,
  'DEV/validation/linux-100-2026-10-25/composer-project-intelligence',
);
mkdirSync(outDir, { recursive: true });

let fixture;
let browser;

async function getJSON(context, url) {
  const response = await context.request.get(url, { headers: { Accept: 'application/json' } });
  assert.equal(response.status(), 200, `GET ${url} returned HTTP ${response.status()}`);
  return response.json();
}

function openBlockingUnknowns(view) {
  return (view?.brief?.unknowns || []).filter(
    (item) => item.severity === 'BLOCKING' && item.status === 'OPEN',
  );
}

async function main() {
  fixture = await startFixtureServer({ composerFlow: true });
  writeFileSync(
    path.join(fixture.isolatedProjectPath, 'package.json'),
    JSON.stringify({
      name: 'grounded-frontend-fixture',
      scripts: { test: 'vitest run', build: 'vite build' },
      devDependencies: { react: '19.0.0', vitest: '3.0.0', vite: '7.0.0' },
    }),
  );

  const launched = await launchFixtureBrowser(fixture.bootstrapUrl);
  browser = launched.browser;
  const { context, page, collectors } = launched;
  page.setDefaultTimeout(20000);

  const { project, apiOrigin } = await seedFixtureProject(
    context,
    fixture.bootstrapUrl,
    fixture.isolatedProjectPath,
  );
  const session = await getJSON(context, `${apiOrigin}/api/v1/session`);
  assert.ok(session.csrf_token, 'fixture session must expose a CSRF token');

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
  assert.equal(prepare.status(), 200, 'project context must become READY');

  await page.goto(`${apiOrigin}/p/${encodeURIComponent(project.id)}/work`, {
    waitUntil: 'domcontentloaded',
  });
  await page
    .getByPlaceholder(/which idea or goal|qual ideia|ideia ou objetivo/i)
    .fill('Improve authentication safely');
  const createPromise = page.waitForResponse(
    (response) =>
      response.url().includes(`/api/v1/projects/${project.id}/composer-sessions`) &&
      response.request().method() === 'POST',
    { timeout: 30000 },
  );
  await page.getByRole('button', { name: /converse|conversar/i }).click();
  const createResponse = await createPromise;
  assert.ok(
    createResponse.status() === 200 || createResponse.status() === 201,
    `vague Composer session returned HTTP ${createResponse.status()}`,
  );
  await page
    .getByRole('button', { name: /review and finalize|revisar e finalizar/i })
    .waitFor({ state: 'visible' });

  const sessions = await getJSON(
    context,
    `${apiOrigin}/api/v1/projects/${project.id}/composer-sessions`,
  );
  assert.equal(sessions.length, 1, 'fixture must contain one vague Composer session');
  const vague = await getJSON(context, `${apiOrigin}/api/v1/composer-sessions/${sessions[0].id}`);
  assert.ok(
    vague.brief?.quality?.testing?.includes('vitest run'),
    'Composer must answer the frontend test-stack question from repository metadata',
  );
  assert.ok(
    openBlockingUnknowns(vague).length > 0,
    'Composer must still ask a material question for the vague authentication goal',
  );
  const testEvidence = (vague.context_manifest?.items || []).find(
    (item) =>
      item.kind === 'PROJECT_FACT' &&
      item.source === 'package.json' &&
      item.locator === 'scripts.test' &&
      item.confidence === 'HIGH' &&
      /test=vitest run · observed \d{4}-\d\d-\d\dT/.test(item.summary),
  );
  assert.ok(testEvidence, 'test command must include confidence, source citation, and freshness');

  const completePrompt =
    'Implement a Vitest unit test for project intelligence. Place it beside the existing frontend tests. Preserve React and Vite conventions. Acceptance criteria: the test covers the reported test command and passes. Verification: run vitest run. Out of scope: package dependency changes and runtime behavior changes.';
  const completeResponse = await context.request.post(
    `${apiOrigin}/api/v1/projects/${project.id}/composer-sessions`,
    {
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'X-CSRF-Token': session.csrf_token,
      },
      data: {
        goal: completePrompt,
        source_prompt: completePrompt,
        input_mode: 'EXISTING_PROMPT',
        interview_mode: 'STANDARD',
      },
    },
  );
  assert.equal(completeResponse.status(), 201, 'complete prompt session must be created');
  const complete = await completeResponse.json();
  assert.deepEqual(
    openBlockingUnknowns(complete),
    [],
    'complete prompt must not create ritual blocking questions',
  );

  await page.screenshot({
    path: path.join(outDir, 'composer-project-intelligence-p4.png'),
    fullPage: true,
  });
  writeFileSync(
    path.join(outDir, 'composer-project-intelligence-p4.json'),
    JSON.stringify(
      {
        project_id: project.id,
        vague_session_id: vague.session.id,
        complete_session_id: complete.session.id,
        frontend_test_command: testEvidence.summary,
        source: testEvidence.source,
        locator: testEvidence.locator,
        confidence: testEvidence.confidence,
        vague_goal_blocking_questions: openBlockingUnknowns(vague).length,
        complete_prompt_blocking_questions: openBlockingUnknowns(complete).length,
        manifest_fingerprint: vague.context_manifest?.fingerprint,
        browser_errors: collectors.pageErrors,
      },
      null,
      2,
    ),
  );
  assert.deepEqual(collectors.pageErrors, [], 'browser page errors must remain empty');
  console.log(
    `project intelligence Composer E2E PASS · test=${testEvidence.summary} · vague-blockers=${openBlockingUnknowns(vague).length} · complete-blockers=0`,
  );
}

main()
  .catch((error) => {
    console.error('project intelligence Composer E2E FAIL');
    console.error(error);
    process.exitCode = 1;
  })
  .finally(async () => {
    if (browser) {
      try {
        await browser.close();
      } catch (_) {}
    }
    if (fixture?.server) await terminateProcess(fixture.server);
    if (fixture?.isolatedDataDir && !process.env.NEXUS_E2E_DATA_DIR) {
      try {
        rmSync(fixture.isolatedDataDir, { recursive: true, force: true });
      } catch (_) {}
    }
  });
