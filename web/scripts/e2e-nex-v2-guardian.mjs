#!/usr/bin/env node
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { request as pwRequest } from 'playwright';
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
const outDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR ||
  path.resolve(repoRoot, 'DEV', 'validation', 'nex-2', '8d1f1592', 'p3-guardian');
if (!existsSync(outDir)) mkdirSync(outDir, { recursive: true });

async function main() {
  console.log('=== Nex 2.0 P3 Guardian Parity E2E ===');
  const { server, bootstrapUrl, isolatedProjectPath, getOutput } = await startFixtureServer();

  let browser;
  let context;
  let page;
  let collectors;
  let unauthApi;
  try {
    ({ browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl));
    const { project, apiOrigin } = await seedFixtureProject(
      context,
      bootstrapUrl,
      isolatedProjectPath,
    );

    // 1. Verify authentication gate and valid read-only schemas on real server
    unauthApi = await pwRequest.newContext();
    const unauthEvents = await unauthApi.get(`${apiOrigin}/api/v1/nex/events`);
    assert.equal(unauthEvents.status(), 401, 'Unauthenticated /api/v1/nex/events must return 401');
    const unauthWatch = await unauthApi.get(`${apiOrigin}/api/v1/nex/models/watch`);
    assert.equal(
      unauthWatch.status(),
      401,
      'Unauthenticated /api/v1/nex/models/watch must return 401',
    );

    const authEventsRes = await context.request.get(`${apiOrigin}/api/v1/nex/events?since=0`);
    assert.equal(authEventsRes.status(), 200);
    const authEventsBody = await authEventsRes.json();
    assert.equal(typeof authEventsBody.revision, 'number');
    assert.ok(Array.isArray(authEventsBody.events));

    const authWatchRes = await context.request.get(`${apiOrigin}/api/v1/nex/models/watch`);
    assert.equal(authWatchRes.status(), 200);
    const authWatchBody = await authWatchRes.json();
    assert.ok(['UNKNOWN', 'FRESH', 'STALE'].includes(authWatchBody.freshness));

    // 2. Load real fixture HUD page (no LLM configured) and navigate Guardian tabs
    await page.setViewportSize({ width: 520, height: 680 });
    await page.goto(`${apiOrigin}/nex-v2.html`, { waitUntil: 'networkidle', timeout: 15000 });
    await page.waitForFunction(
      () => Boolean(window.__nexV2PresenceMetrics) && window.__nexV2Metrics?.rigReady === true,
      null,
      { timeout: 10000 },
    );
    await page.locator('[data-nex-ready] button:not([data-wails-draggable])').click();
    await page.waitForSelector('[data-nex-panel]', { timeout: 5000 });
    await page.locator('[data-nex-tab="resources"]').click();
    await page.waitForSelector('[data-nex-resources]', { timeout: 5000 });
    await page.locator('[data-nex-tab="attention"]').click();
    await page.waitForSelector('[data-nex-attention]', { timeout: 5000 });

    // 3. Inject canonical Guardian multi-window, UNKNOWN, cost, alert, attention & model-watch snapshot
    const mockHud = {
      enabled: true,
      rollout: 'experimental',
      state: {
        generated_at: '2026-11-25T13:00:00Z',
        project_id: project.id,
        providers: [
          {
            provider: 'claude',
            profile: 'work-primary',
            health: 'WARNING',
            availability: 'AVAILABLE',
            capacity_state: 'LIMITED',
            freshness: 'FRESH',
            rate_limited: true,
            limiting_window: {
              kind: '5h',
              group: 'claude-sonnet-4',
              label: '5h',
              remaining_percent: 38,
              reset_description: 'in 2h',
              status: 'OK',
            },
            windows: [
              {
                kind: '5h',
                group: 'claude-sonnet-4',
                label: '5h',
                remaining_percent: 38,
                reset_description: 'in 2h',
                status: 'OK',
              },
              {
                kind: 'weekly',
                group: 'claude-opus-4',
                label: 'weekly',
                status: 'UNKNOWN',
              },
            ],
          },
        ],
        agents: [{ id: 'ag-1', project_id: project.id, name: 'Coder', status: 'WORKING' }],
        missions: [
          {
            run_id: 'run-1',
            project_id: project.id,
            state: 'BLOCKED_NEEDS_USER',
            current_stage: 'review',
            current_task_title: 'Confirm schema migration',
            completed_packages: 1,
            total_packages: 2,
            needs_user: true,
            intervention_reason: 'User decision required',
          },
        ],
        alerts: [
          {
            fingerprint: 'fp-guardian-1',
            category: 'quota',
            severity: 'WARNING',
            provider: 'claude',
            profile: 'work-primary',
            reason: 'Quota below 40%',
            acked: false,
            last_observed_at: '2026-11-25T13:00:00Z',
          },
        ],
        runtime_attention: [
          {
            runtime_id: 'rt-1',
            project_id: project.id,
            project_name: 'Guardian Demo',
            provider: 'codex',
            reason: 'Approve test command',
            prompt_kind: 'yn',
            fingerprint: 'rt-fp-1',
            updated_at: '2026-11-25T13:00:00Z',
          },
        ],
        cost: {
          evidence: 'PARTIAL',
          token_evidence: 'PARTIAL',
          known_microusd: 4_250_000,
          unknown_observation_count: 1,
          budget_level: 'OK',
        },
        primary: { kind: 'ALERT', id: 'fp-guardian-1' },
        system: {
          capacity: 'OK',
          resources: 'OK',
          agents: 'OK',
          missions: 'OK',
          cost: 'PARTIAL',
          cache_ttl_millis: 15_000,
        },
      },
    };

    const mockWatch = {
      freshness: 'FRESH',
      observed_at: '2026-11-25T13:00:00Z',
      entries: [
        {
          provider: 'claude',
          profile: 'work-primary',
          model_id: 'claude-sonnet-4',
          is_default: true,
          deprecated: false,
          source: 'provider_snapshot',
          observed_at: '2026-11-25T13:00:00Z',
        },
      ],
      changes: [
        {
          fingerprint: 'mc-default-1',
          kind: 'DEFAULT_CHANGED',
          provider: 'claude',
          profile: 'work-primary',
          model_id: 'claude-sonnet-4',
          previous_default: 'claude-opus-4',
          new_default: 'claude-sonnet-4',
          source: 'hud_projection',
          observed_at: '2026-11-25T13:00:00Z',
        },
      ],
    };

    const mockEvent = {
      revision: 1,
      event_id: 'nex-ev-1',
      fingerprint: 'model:mc-default-1',
      kind: 'MODEL_CATALOG_DEFAULT_CHANGED',
      severity: 'INFO',
      source: 'model_catalog_watch',
      title: 'claude · claude-sonnet-4',
      detail: 'DEFAULT_CHANGED',
      provider: 'claude',
      profile: 'work-primary',
      created_at: '2026-11-25T13:00:00Z',
    };

    await page.route('**/api/v1/hud', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockHud),
      }),
    );
    await page.route('**/api/v1/nex/events*', (route) => {
      const url = new URL(route.request().url());
      const since = Number(url.searchParams.get('since') || '0');
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          revision: 1,
          replay_truncated: false,
          events: since >= 1 ? [] : [mockEvent],
          model_watch_freshness: 'FRESH',
        }),
      });
    });
    await page.route('**/api/v1/nex/models/watch', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockWatch),
      }),
    );

    await page.reload({ waitUntil: 'networkidle', timeout: 15000 });
    await page.waitForSelector('aside[role="status"]', { timeout: 5000 });
    const bubbleText = await page.locator('aside[role="status"]').innerText();
    assert.ok(bubbleText.includes('work-primary'));

    // Dismissing bubble prevents duplicate resurfacing.
    await page.locator('aside[role="status"] button[aria-label]').last().click();
    assert.equal(await page.locator('aside[role="status"]').count(), 0);

    await page.locator('[data-nex-ready] button:not([data-wails-draggable])').click();
    await page.waitForSelector('[data-nex-panel]', { timeout: 5000 });
    await page.locator('[data-nex-tab="resources"]').click();
    await page.waitForSelector('[data-nex-resources]', { timeout: 5000 });
    const resText = await page.locator('[data-nex-resources]').innerText();
    assert.ok(resText.includes('38%'));
    assert.ok(resText.includes('UNKNOWN'));
    assert.ok(!resText.includes('0%'));
    assert.ok(resText.includes('PARTIAL'));
    assert.ok(resText.includes('DEFAULT_CHANGED'));
    await page.screenshot({ path: path.join(outDir, 'p3-guardian-resources.png') });

    // Enable Privacy Mode and verify sensitive identifiers are masked.
    await page.locator('[data-nex-panel-settings]').click();
    await page.waitForSelector('[data-nex-settings]', { timeout: 3000 });
    await page.locator('[data-nex-toggle="privacy-mode"]').click();
    const maskedResText = await page.locator('[data-nex-resources]').innerText();
    assert.ok(!maskedResText.includes('work-primary'), 'Profile must be masked under Privacy Mode');
    assert.ok(!maskedResText.includes('4.25'), 'Spend amount must be masked under Privacy Mode');
    assert.ok(maskedResText.includes('•••'));
    await page.screenshot({ path: path.join(outDir, 'p3-guardian-privacy.png') });
    await page.locator('[data-nex-settings-close]').click();

    await page.locator('[data-nex-tab="attention"]').click();
    await page.waitForSelector('[data-nex-attention]', { timeout: 5000 });
    const attText = await page.locator('[data-nex-attention]').innerText();
    assert.ok(attText.includes('Quota below 40%'));
    assert.ok(attText.includes('MODEL_CATALOG_DEFAULT_CHANGED'));
    await page.screenshot({ path: path.join(outDir, 'p3-guardian-attention.png') });

    const metrics = await page.evaluate(() => window.__nexV2PresenceMetrics);
    assert.equal(metrics.privacyMode, true);
    assert.equal(metrics.eventRevision, 1);
    assert.equal(metrics.modelWatchFreshness, 'FRESH');

    const requestFailures = meaningfulFixtureRequestFailures(collectors.requestFailures);
    assert.equal(
      collectors.pageErrors.length,
      0,
      `Page errors: ${collectors.pageErrors.join('; ')}`,
    );
    assert.equal(
      collectors.consoleErrors.length,
      0,
      `Console errors: ${collectors.consoleErrors.join('; ')}`,
    );
    assert.equal(requestFailures.length, 0, `Request failures: ${requestFailures.join('; ')}`);

    const summary = {
      authEventsStatus: authEventsRes.status(),
      authWatchStatus: authWatchRes.status(),
      privacyMode: metrics.privacyMode,
      eventRevision: metrics.eventRevision,
      modelWatchFreshness: metrics.modelWatchFreshness,
      pageErrors: collectors.pageErrors.length,
      consoleErrors: collectors.consoleErrors.length,
      requestFailures: requestFailures.length,
    };
    writeFileSync(path.join(outDir, 'summary.json'), `${JSON.stringify(summary, null, 2)}\n`);
    console.log('Nex 2.0 P3 Guardian Parity E2E: PASS', JSON.stringify(summary));
  } catch (error) {
    console.error('Nex 2.0 P3 Guardian Parity E2E: FAIL:', error);
    console.error('Server output:\n' + getOutput());
    process.exitCode = 1;
  } finally {
    if (unauthApi) await unauthApi.dispose().catch(() => {});
    if (context) await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
    await terminateProcess(server);
  }
}

await main();
