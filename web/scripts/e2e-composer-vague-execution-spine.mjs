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
const outDir = path.resolve(
  process.env.NEXUS_E2E_ARTIFACTS_DIR ||
    path.join(repoRoot, 'DEV/validation/linux-100-2026-10-25/composer-project-intelligence'),
);
mkdirSync(outDir, { recursive: true });

const vagueGoal = 'Improve authentication safely';

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

async function activeOpenUnknown(page, view) {
  const visibleQuestions = await page.locator('section[data-severity]').evaluateAll((sections) =>
    sections
      .filter((section) => {
        const bounds = section.getBoundingClientRect();
        return bounds.width > 0 && bounds.height > 0;
      })
      .map((section) => section.getAttribute('aria-label'))
      .filter(Boolean),
  );
  const unknowns = view?.brief?.unknowns || [];
  for (const question of visibleQuestions) {
    const match = unknowns.find((item) => item.status === 'OPEN' && item.question === question);
    if (match) return match;
  }
  return null;
}

async function resolveBlockingUnknown(page, context, apiOrigin, sessionId, unknown) {
  const prompt = page.getByRole('region', { name: unknown.question, exact: true });
  await prompt.waitFor({ state: 'visible', timeout: 15000 });

  const resolveEvents = [];
  const onResolveRequest = (request) => {
    if (request.url().includes('/unknowns/')) {
      resolveEvents.push(`${request.method()} ${request.url()}`);
    }
  };
  const onResolveResponse = (response) => {
    if (response.url().includes('/unknowns/')) {
      resolveEvents.push(`${response.status()} ${response.url()}`);
    }
  };
  const onResolveRequestFailed = (request) => {
    if (request.url().includes('/unknowns/')) {
      resolveEvents.push(`FAILED ${request.url()} ${request.failure()?.errorText || ''}`.trim());
    }
  };
  page.on('request', onResolveRequest);
  page.on('response', onResolveResponse);
  page.on('requestfailed', onResolveRequestFailed);

  const responsePromise = page.waitForResponse(
    (response) =>
      response
        .url()
        .includes(`/api/v1/composer-sessions/${sessionId}/unknowns/${unknown.id}/resolve`) &&
      response.request().method() === 'POST',
    { timeout: 45000 },
  );

  const options = Array.isArray(unknown.suggested_options)
    ? unknown.suggested_options.filter(Boolean)
    : [];
  if (options.length > 0) {
    await prompt.getByRole('option', { name: options[0], exact: true }).click();
  } else {
    const answer =
      'Use the safest backwards-compatible default that follows the existing repository conventions.';
    await prompt.getByPlaceholder(/answer required|resposta obrigatória/i).fill(answer);
    await prompt.getByRole('button', { name: /answer|responder/i }).click();
  }

  let response;
  try {
    response = await responsePromise;
  } catch (error) {
    const buttonState = await prompt
      .getByRole('option')
      .evaluateAll((buttons) =>
        buttons.map((button) => ({
          text: button.textContent,
          disabled: button.disabled,
          selected: button.getAttribute('aria-selected'),
        })),
      )
      .catch(() => []);
    const sessionSnapshot = await getJSON(
      context,
      `${apiOrigin}/api/v1/composer-sessions/${sessionId}`,
    ).catch((snapshotError) => ({ error: String(snapshotError) }));
    const unknownSnapshot = (sessionSnapshot?.brief?.unknowns || []).find(
      (item) => item.id === unknown.id,
    );
    console.error(
      'composer unknown resolve diagnostics',
      JSON.stringify({
        expectedPath: `/api/v1/composer-sessions/${sessionId}/unknowns/${unknown.id}/resolve`,
        events: resolveEvents,
        buttonState,
        unknownStatus: unknownSnapshot?.status || sessionSnapshot?.error || 'not found',
      }),
    );
    throw error;
  } finally {
    page.off('request', onResolveRequest);
    page.off('response', onResolveResponse);
    page.off('requestfailed', onResolveRequestFailed);
  }
  assert.equal(response.status(), 200, `resolve unknown returned HTTP ${response.status()}`);

  const refreshed = await getJSON(context, `${apiOrigin}/api/v1/composer-sessions/${sessionId}`);
  const stillOpen = openBlockingUnknowns(refreshed).some((item) => item.id === unknown.id);
  assert.equal(stillOpen, false, `blocking clarification ${unknown.id} did not converge`);
  return refreshed;
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
  page.setDefaultTimeout(25000);

  const { project, apiOrigin } = await seedFixtureProject(
    context,
    fixture.bootstrapUrl,
    fixture.isolatedProjectPath,
  );

  const session = await getJSON(context, `${apiOrigin}/api/v1/session`);
  assert.ok(session.csrf_token, 'authenticated fixture must expose a CSRF token');

  const skillCatalog = await getJSON(
    context,
    `${apiOrigin}/api/v1/projects/${project.id}/skills?q=security`,
  );
  assert.ok(
    Array.isArray(skillCatalog) && skillCatalog.length > 0,
    'skill catalog must be searchable',
  );
  assert.ok(
    skillCatalog.every(
      (skill) =>
        !Object.prototype.hasOwnProperty.call(skill, 'instructions') &&
        !Object.prototype.hasOwnProperty.call(skill, 'permissions') &&
        !Object.prototype.hasOwnProperty.call(skill, 'provenance'),
    ),
    'browser skill catalog must expose metadata only',
  );

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
  assert.equal(
    readiness.state,
    'READY',
    `Composer context must be READY: ${JSON.stringify(readiness)}`,
  );

  await page.goto(`${apiOrigin}/p/${encodeURIComponent(project.id)}/work`, {
    waitUntil: 'domcontentloaded',
  });

  const idea = page.getByPlaceholder(/which idea or goal|qual ideia|ideia ou objetivo/i);
  await idea.fill(vagueGoal);

  const createResponsePromise = page.waitForResponse(
    (response) =>
      response.url().includes(`/api/v1/projects/${project.id}/composer-sessions`) &&
      response.request().method() === 'POST',
    { timeout: 30000 },
  );
  await page.getByRole('button', { name: /converse|conversar/i }).click();
  const createResponse = await createResponsePromise;
  assert.ok(
    createResponse.status() === 200 || createResponse.status() === 201,
    `Composer create returned HTTP ${createResponse.status()}`,
  );

  const reviewFinalize = page.getByRole('button', {
    name: /review and finalize|revisar e finalizar/i,
  });
  await reviewFinalize.waitFor({ state: 'visible', timeout: 45000 });

  const sessions = await getJSON(
    context,
    `${apiOrigin}/api/v1/projects/${project.id}/composer-sessions`,
  );
  assert.equal(sessions.length, 1, 'isolated vague-goal fixture must create exactly one session');
  const sessionId = sessions[0].id;
  assert.ok(sessionId, 'vague-goal Composer session must persist');

  let view = await getJSON(context, `${apiOrigin}/api/v1/composer-sessions/${sessionId}`);
  assert.ok(
    view.brief?.quality?.testing?.includes('vitest run'),
    'Composer must answer the frontend test-stack question from fixture repository metadata',
  );
  const citedTestEvidence = (view.context_manifest?.items || []).find(
    (item) =>
      item.kind === 'PROJECT_FACT' &&
      item.source === 'package.json' &&
      item.locator === 'scripts.test' &&
      item.confidence === 'HIGH' &&
      /test=vitest run · observed \d{4}-\d\d-\d\dT/.test(item.summary),
  );
  assert.ok(
    citedTestEvidence,
    'Composer must retain the test command, confidence, source citation, and observation freshness',
  );
  assert.ok(
    view.context_manifest?.fingerprint,
    'Composer must persist a context manifest fingerprint',
  );
  assert.ok(
    Number(view.context_manifest?.budget?.used_estimated_tokens || 0) <=
      Number(view.context_manifest?.budget?.max_estimated_tokens || 0),
    'Composer context manifest must stay within its declared budget',
  );
  assert.ok(
    Array.isArray(view.context_manifest?.items) &&
      view.context_manifest.items.some((item) => item.decision === 'KEEP'),
    'Composer context manifest must contain at least one KEEP item',
  );
  assert.ok(
    (view.context_manifest?.items || [])
      .filter((item) => item.kind === 'MISSION_REFERENCE')
      .every((item) => item.decision === 'REFERENCE_ONLY'),
    'Mission history must remain reference-only in Composer context',
  );

  const contextTab = page.getByRole('tab', { name: /context|contexto/i });
  await contextTab.click();
  await page
    .getByRole('heading', {
      name: /context discovery|context used|descoberta.*contexto|contexto utilizado/i,
    })
    .waitFor({ state: 'visible', timeout: 15000 });

  const initialBlocking = openBlockingUnknowns(view).length;
  assert.ok(initialBlocking > 0, 'vague goal must exercise at least one material clarification');

  let resolved = 0;
  while (openBlockingUnknowns(view).length > 0) {
    assert.ok(resolved < 12, 'vague-goal clarification loop did not converge');
    const unknown = await activeOpenUnknown(page, view);
    assert.ok(
      unknown,
      `Composer UI must expose the next unknown from the current view; visible=${JSON.stringify(
        await page
          .locator('section[data-severity]')
          .evaluateAll((sections) =>
            sections
              .filter((section) => section.getBoundingClientRect().height > 0)
              .map((section) => section.getAttribute('aria-label')),
          ),
      )}`,
    );
    view = await resolveBlockingUnknown(page, context, apiOrigin, sessionId, unknown);
    resolved += 1;
  }

  const finalizationResponsePromise = page.waitForResponse(
    (response) =>
      response.url().includes(`/api/v1/composer-sessions/${sessionId}/finalize`) &&
      response.request().method() === 'POST',
    { timeout: 30000 },
  );
  await reviewFinalize.click();

  const turnIntoFlow = page.getByRole('button', {
    name: /turn into flow|transformar.*flow|converter.*flow/i,
  });
  const finalizeWithGaps = page.getByRole('button', {
    name: /finish with confirmed gaps|finalizar com lacunas|concluir com lacunas/i,
  });

  const firstDestination = await Promise.race([
    turnIntoFlow.waitFor({ state: 'visible', timeout: 30000 }).then(() => 'flow'),
    finalizeWithGaps.waitFor({ state: 'visible', timeout: 30000 }).then(() => 'confirm-gaps'),
  ]);

  if (firstDestination === 'confirm-gaps') {
    await finalizeWithGaps.click();
    await turnIntoFlow.waitFor({ state: 'visible', timeout: 30000 });
  }

  const finalizationResponse = await finalizationResponsePromise;
  const finalized = await getJSON(context, `${apiOrigin}/api/v1/composer-sessions/${sessionId}`);
  assert.equal(
    finalized.session.state,
    'FINALIZED',
    `clarified session must finalize normally (POST /finalize HTTP ${finalizationResponse.status()})`,
  );

  await turnIntoFlow.click();

  const executionPlan = page.getByRole('button', {
    name: /plano de execução|execution plan/i,
  });
  await executionPlan.waitFor({ state: 'visible', timeout: 30000 });

  const runsBefore = await getJSON(context, `${apiOrigin}/api/v1/runs`);
  const agentsBefore = await getJSON(context, `${apiOrigin}/api/v1/projects/${project.id}/agents`);

  const previewResponsePromise = page.waitForResponse(
    (response) =>
      response.url().includes('/execution-preview') && response.request().method() === 'POST',
    { timeout: 30000 },
  );
  await executionPlan.click();
  const previewResponse = await previewResponsePromise;
  assert.equal(previewResponse.status(), 200, 'Execution Preview API must succeed');
  const preview = await previewResponse.json();

  assert.equal(preview.project_id, project.id, 'preview must belong to the fixture project');
  assert.ok(Number(preview.revision) >= 1, 'preview must be bound to a persisted revision');
  assert.ok(
    Array.isArray(preview.steps) && preview.steps.length >= 1,
    'clarified vague goal must materialize executable Flow steps',
  );
  assert.ok(
    preview.steps.every((step) => step.agent?.reason && step.resource?.reason),
    'every clarified step must explain Agent and resource decisions',
  );

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

  if (!preview.ready) {
    assert.ok(
      Array.isArray(preview.blockers) && preview.blockers.length > 0,
      'not-ready preview must explain blockers',
    );
  }

  const completePrompt =
    'Add a Vitest unit test for the existing project-intelligence summary. Place it beside the current frontend unit tests, preserve the current React and Vite conventions, verify the test with `vitest run`, and do not change runtime behavior or package dependencies.';
  const completeCreate = await context.request.post(
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
  assert.equal(completeCreate.status(), 201, 'complete prompt Composer create must succeed');
  const completeView = await completeCreate.json();
  assert.deepEqual(
    openBlockingUnknowns(completeView),
    [],
    'complete prompt must bypass ritual blocking questions',
  );

  await page.screenshot({ path: path.join(outDir, 'north-star-vague.png'), fullPage: true });
  writeFileSync(
    path.join(outDir, 'north-star-vague.json'),
    JSON.stringify(
      {
        project_id: project.id,
        composer_session_id: sessionId,
        initial_blocking_questions: initialBlocking,
        resolved_blocking_questions: resolved,
        grounded_frontend_test_command: citedTestEvidence.summary,
        complete_prompt_blocking_questions: openBlockingUnknowns(completeView).length,
        plan_id: preview.plan_id,
        revision: preview.revision,
        preview_ready: preview.ready,
        step_count: preview.steps.length,
        blockers: preview.blockers || [],
        context_manifest: {
          fingerprint: finalized.context_manifest?.fingerprint,
          budget: finalized.context_manifest?.budget,
          item_count: finalized.context_manifest?.items?.length || 0,
        },
        skill_catalog_count: skillCatalog.length,
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
    `composer vague execution spine e2e PASS · session=${sessionId} clarifications=${resolved} plan=${preview.plan_id} revision=${preview.revision} steps=${preview.steps.length} ready=${preview.ready}`,
  );
}

main()
  .catch(async (error) => {
    console.error('composer vague execution spine e2e FAIL');
    console.error(error);
    if (fixture) {
      try {
        const serverOutput = String(fixture.getOutput?.() || '')
          .replace(/(nexus_bootstrap=)[a-f0-9]+/gi, '$1[REDACTED]')
          .replace(/("bootstrap_token"\s*:\s*")[^"]+("?)/gi, '$1[REDACTED]$2');
        console.error(
          'composer vague fixture server diagnostics',
          JSON.stringify({
            exitCode: fixture.server?.exitCode,
            signalCode: fixture.server?.signalCode,
            outputTail: serverOutput.slice(-12000),
          }),
        );
        writeFileSync(path.join(outDir, 'failure-vague-server.log'), serverOutput);
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
