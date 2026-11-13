#!/usr/bin/env node
// NEX Companion 2D lab: exercises the eight contextual behaviors from
// COMPANION_2D_CLIPPY_LITE.md against the real web HUD with an explicitly
// fixture-isolated Core (attention/HUD routes), capturing per-behavior
// evidence. Never claims native Wails shell or real-provider coverage.
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { execSync } from 'node:child_process';
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
const evidenceDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR ||
  path.resolve(repoRoot, 'DEV', 'validation', 'current', 'companion-lab');
const captureDir = path.join(evidenceDir, 'captures');
if (!existsSync(captureDir)) mkdirSync(captureDir, { recursive: true });

const emptyAttention = {
  needs_you: [],
  completed: [],
  failed: [],
  all_items: [],
  total_needs: 0,
};

async function main() {
  const shortSha = execSync('git rev-parse --short HEAD', { cwd: repoRoot }).toString().trim();
  console.log(`=== NEX Companion 2D Lab (fixture SHA ${shortSha}) ===`);
  process.env.NEXUS_ENABLE_HUD = '1';
  const { server, bootstrapUrl, isolatedProjectPath, isolatedDataDir, getOutput } =
    await startFixtureServer();

  let browser;
  let collectors;
  let context;
  let page;
  try {
    ({ browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl));
    const { project, apiOrigin } = await seedFixtureProject(
      context,
      bootstrapUrl,
      isolatedProjectPath,
    );

    // The companion behaviors live in the native capsule shell. The lab
    // renders the real DesktopHudApp in a plain browser by presenting the
    // Wails user agent (platform detection), while every native call fails
    // gracefully exactly as it does without a Wails runtime.
    const wailsContext = await browser.newContext({
      userAgent:
        'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 Wails',
    });
    context = wailsContext;
    page = await wailsContext.newPage();
    collectors = { consoleErrors: [], requestFailures: [], pageErrors: [], responses: [] };
    const attachCollectors = (target) => {
      target.on('pageerror', (error) => {
        console.log('PAGE ERROR:', String(error?.message || error).slice(0, 160));
        collectors.pageErrors.push(String(error?.message || error));
      });
      target.on('console', (message) => {
        if (message.type() === 'error') collectors.consoleErrors.push(message.text());
      });
      target.on('requestfailed', (request) => {
        collectors.requestFailures.push(
          `${request.method()} ${request.url()} → ${request.failure()?.errorText || 'unknown'}`,
        );
      });
    };
    attachCollectors(page);
    await seedFixtureProject(wailsContext, bootstrapUrl, isolatedProjectPath);
    await page.addInitScript(() => {
      window.localStorage.setItem(
        'iapro:nexus:hud:presentation:v1',
        JSON.stringify({ version: 1, enabled: true }),
      );
    });

    // Explicitly fixture-isolated Core projection. The production runtime is
    // never touched: these routes only exist inside this browser context.
    let attentionFixture = emptyAttention;
    let hudOverride = null;
    await context.route(/\/api\/v1\/attention/, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(attentionFixture),
      }),
    );
    await context.route(/\/api\/v1\/hud(\?|$)/, async (route) => {
      if (!hudOverride) return route.continue();
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(hudOverride),
      });
    });

    const shellUrl = new URL(`/?nativeHud=1`, apiOrigin).toString();
    const behaviorLocator = () => page.locator('[data-hud-mascot="true"]');
    const mascotRoot = () => page.locator('[data-nex-behavior]');

    // Each scenario renders in a fresh page: the standalone HUD app holds no
    // cross-scenario state, and repeated same-URL reloads of one long-lived
    // page proved fragile under navigation interrupts.
    const openShell = async () => {
      let lastError;
      for (let attempt = 0; attempt < 3; attempt += 1) {
        try {
          if (!page.isClosed()) await page.close().catch(() => {});
          page = await context.newPage();
          attachCollectors(page);
          await page.goto(shellUrl, { waitUntil: 'domcontentloaded', timeout: 15000 });
          await page
            .locator('aside[aria-label="Nexus HUD"]')
            .waitFor({ state: 'visible', timeout: 15000 });
          await mascotRoot().waitFor({ state: 'visible', timeout: 15000 });
          return;
        } catch (error) {
          lastError = error;
          if (collectors.consoleErrors.length) {
            console.log('console errors during load:', collectors.consoleErrors.slice(-4));
          }
          await page.waitForTimeout(1200).catch(() => {});
        }
      }
      const rawHtml = await page.content().catch(() => 'content unreadable');
      const diag = await page
        .evaluate(() => ({
          readyState: document.readyState,
          rootChildren: document.getElementById('root')?.children.length ?? -1,
          bodyText: document.body?.innerText?.slice(0, 120) || '',
        }))
        .catch((e) => `eval failed: ${String(e).slice(0, 80)}`);
      throw new Error(
        `HUD shell did not load after 3 attempts (url=${page.url()}): ${String(lastError?.message || lastError).slice(0, 120)} | diag=${JSON.stringify(diag)} | html="${rawHtml.slice(0, 200)}"`,
      );
    };
    const reloadWith = async ({ attention = emptyAttention, hud = null } = {}) => {
      attentionFixture = attention;
      hudOverride = hud;
      await openShell();
    };
    const capture = async (name) => {
      await page
        .locator('aside[aria-label="Nexus HUD"]')
        .screenshot({ path: path.join(captureDir, `${name}.png`) });
    };

    await openShell();

    // Real HUD envelope snapshot keeps every synthetic scenario shape-authentic.
    const hudResponse = await context.request.get(
      `${apiOrigin}/api/v1/hud?project_id=${encodeURIComponent(project.id)}`,
    );
    assert.equal(hudResponse.status(), 200);
    const realEnvelope = await hudResponse.json();
    const envelopeWith = (mutate) => {
      const clone = JSON.parse(JSON.stringify(realEnvelope));
      mutate(clone.state || {});
      return clone;
    };

    // The production HUD refresh cadence is max(15s, cache_ttl); poll-driven
    // scenario switches avoid repeated same-URL reloads entirely.
    const POLL_WAIT_MS = 50000;

    // 1. ambient — idle mood, motion on, visible, no attention.
    assert.equal(await mascotRoot().getAttribute('data-mood'), 'idle');
    await assertBehavior(mascotRoot(), 'ambient', 'ambient idle must compose ambient');
    await capture('01-ambient');
    assert.equal(
      await page.locator('[data-nex-progress]').count(),
      0,
      'companion must never render invented progress',
    );

    // 2. greet — real pointer entry on the mascot button.
    await behaviorLocator().hover();
    await assertBehavior(mascotRoot(), 'greet', 'pointer enter must greet');
    await capture('02-greet');
    await page.mouse.move(10, 10);
    await assertBehavior(mascotRoot(), 'ambient', 'pointer leave must rest back to ambient');
    // Keyboard focus must greet exactly like pointer entry (a11y parity).
    await behaviorLocator().focus();
    await assertBehavior(mascotRoot(), 'greet', 'keyboard focus must greet like pointer entry');
    await page.keyboard.press('Tab');
    await behaviorLocator()
      .blur()
      .catch(() => {});
    await assertBehavior(mascotRoot(), 'ambient', 'blur must rest back to ambient');

    // 3. look — mouse move after the greet window resolves to a side glance.
    await behaviorLocator().hover();
    await page.waitForTimeout(650);
    const box = await behaviorLocator().boundingBox();
    assert.ok(box, 'mascot must have geometry for the look gesture');
    await page.mouse.move(box.x + box.width * 0.15, box.y + box.height / 2);
    await assertBehavior(mascotRoot(), 'look', 'lateral mouse move must look');
    assert.equal(await mascotRoot().getAttribute('data-nex-look'), 'left');
    await capture('03-look');
    await page.mouse.move(10, 10);
    await page.mouse.move(700, 500);
    assert.equal(
      await mascotRoot().getAttribute('data-nex-look'),
      'left',
      'no global cursor listener may mutate the glance outside the hitbox',
    );

    // 4. thinking — canonical working mood from an authoritative agent status.
    hudOverride = envelopeWith((state) => {
      state.agents = [
        {
          id: 'fixture-agent',
          name: 'Fixture Agent',
          status: 'WORKING',
          availability: 'AVAILABLE',
        },
      ];
    });
    await page_waitForAttribute(
      mascotRoot(),
      'data-mood',
      'working',
      POLL_WAIT_MS,
      'fixture working agent must reach the HUD via poll',
    );
    await assertBehavior(mascotRoot(), 'thinking', 'working mood must think');
    await capture('04-thinking');
    assert.equal(
      await page.locator('[data-nex-progress]').count(),
      0,
      'working must not fabricate a progress percentage',
    );

    // 5. attention — canonical warning attention: an agent waiting for input.
    hudOverride = envelopeWith((state) => {
      state.agents = [
        {
          id: 'fixture-agent-await',
          name: 'Awaiting Agent',
          status: 'WAITING_INPUT',
          availability: 'AVAILABLE',
        },
      ];
    });
    await page_waitForAttribute(
      mascotRoot(),
      'data-mood',
      'attention',
      POLL_WAIT_MS,
      'fixture waiting agent must reach the HUD via poll',
    );
    await assertBehavior(mascotRoot(), 'attention', 'agent waiting input must surface attention');
    await capture('05-attention');

    // 6. error — real critical evidence preempts every user gesture.
    const failure = {
      mission_id: 'fixture-mission-failed',
      project_id: project.id,
      state: 'FAILED',
      level: 'REQUIRE_USER',
      reason_code: 'EXECUTION_FAILED',
      summary: 'Fixture mission failed',
      question: '',
      impact: 'Mission aborted',
      recommended_actions: [],
      age_seconds: 3,
    };
    attentionFixture = { ...emptyAttention, failed: [failure] };
    await page_waitForAttribute(
      mascotRoot(),
      'data-mood',
      'critical',
      POLL_WAIT_MS,
      'fixture failure must reach the HUD via poll',
    );
    await assertBehavior(mascotRoot(), 'error', 'critical evidence must error');
    await behaviorLocator().hover();
    await assertBehavior(mascotRoot(), 'error', 'critical status must preempt greet/look gestures');
    await capture('06-error');

    // 7. resolved — only from a newly observed, deduplicated completion ID.
    // The baseline reload must observe zero completions; a fresh page holds no
    // history, so the next poll observation is genuinely new.
    await reloadWith();
    attentionFixture = {
      ...emptyAttention,
      completed: [
        {
          mission_id: 'fixture-mission-done',
          project_id: project.id,
          state: 'COMPLETED_VERIFIED',
          age_seconds: 1,
        },
      ],
    };
    await assertBehavior(
      mascotRoot(),
      'resolved',
      'new verified completion must celebrate once',
      55000,
    );
    await capture('07-resolved');
    await page.waitForTimeout(1600);
    await assertBehavior(mascotRoot(), 'ambient', 'resolution must not loop or repeat');
    await assertBehavior(mascotRoot(), 'ambient', 'same completion id must stay deduplicated');

    // 8. rest — drowse timing (180s) is validated by nexIdleDirector unit
    // tests with an injected clock; the lab documents that split honestly.
    assert.notEqual(
      await mascotRoot().getAttribute('data-nex-behavior'),
      'rest',
      'short sessions must not fall asleep',
    );
    assert.notEqual(
      await mascotRoot().getAttribute('data-nex-behavior'),
      'still',
      'idle rest is not offline stillness',
    );

    // Safety: offline mood renders static. The honest offline path is the
    // app's own failure handling: an unreachable HUD projection flips the
    // canonical offline flag (fixture-isolated 500, no fabricated providers).
    hudOverride = { fixture: 'unreachable' };
    let offlineMode = false;
    await context.route(/\/api\/v1\/hud(\?|$)/, async (route) => {
      if (offlineMode) return route.fulfill({ status: 500, body: 'fixture offline' });
      if (!hudOverride || hudOverride.fixture) return route.continue();
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(hudOverride),
      });
    });
    offlineMode = true;
    await page_waitForAttribute(
      mascotRoot(),
      'data-mood',
      'offline',
      POLL_WAIT_MS,
      'projection failure must surface canonical offline via poll',
    );
    assert.equal(await mascotRoot().getAttribute('data-motion'), 'off');
    await assertBehavior(mascotRoot(), 'still', 'offline must be fully static');
    await capture('08-offline-still');

    await page.emulateMedia({ reducedMotion: 'reduce' });
    await assertBehavior(mascotRoot(), 'still', 'reduced motion must freeze the companion');
    await page.emulateMedia({ reducedMotion: 'no-preference' });

    const meaningfulFailures = meaningfulFixtureRequestFailures(collectors.requestFailures);
    assert.equal(
      meaningfulFailures.length,
      0,
      `lab request failures: ${meaningfulFailures.join(' | ')}`,
    );
    assert.equal(
      collectors.pageErrors.length,
      0,
      `lab page errors: ${collectors.pageErrors.join(' | ')}`,
    );

    writeFileSync(
      path.join(evidenceDir, 'README.md'),
      [
        '# NEX Companion 2D Lab — evidência',
        '',
        `- Data: ${new Date().toISOString()}`,
        `- SHA: ${shortSha}`,
        '- Script: `web/scripts/e2e-companion-lab.mjs` (fixture Core isolado por contexto de browser)',
        '- Comportamentos capturados: ambient, greet, look, thinking, attention, error, resolved, offline-still.',
        '- `rest` (drowse 180s): validado por `nexIdleDirector.test.ts` com clock injetado; sem captura visual em tempo real.',
        '- O mock vive apenas em `context.route`; nenhuma rota de produção é tocada e nenhum estado é inferido.',
        '',
        '## Capturas',
        '',
        ...[
          '01-ambient',
          '02-greet',
          '03-look',
          '04-thinking',
          '05-attention',
          '06-error',
          '07-resolved',
          '08-offline-still',
        ].map((n) => `- \`captures/${n}.png\``),
        '',
      ].join('\n'),
    );

    console.log('NEX Companion 2D Lab: PASS');
  } catch (error) {
    console.error(`Companion lab server output:\n${getOutput()}`);
    throw error;
  } finally {
    if (context) await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
    await terminateProcess(server);
    if (!process.env.NEXUS_E2E_DATA_DIR && isolatedDataDir) {
      const { rmSync } = await import('node:fs');
      rmSync(isolatedDataDir, { recursive: true, force: true });
    }
  }
}

async function assertBehavior(locator, expected, message, timeout = 5000) {
  await locator.waitFor({ state: 'visible' }).catch(() => {});
  await page_waitForAttribute(locator, 'data-nex-behavior', expected, timeout, message);
}

async function page_waitForAttribute(locator, attribute, expected, timeout, message) {
  const start = Date.now();
  for (;;) {
    const value = await locator.getAttribute(attribute);
    if (value === expected) return;
    if (Date.now() - start > timeout) {
      assert.equal(value, expected, `${message} (last=${value})`);
    }
    await new Promise((resolve) => setTimeout(resolve, 120));
  }
}

main().catch((error) => {
  console.error(`NEX Companion 2D Lab: FAIL: ${error?.stack || error}`);
  process.exit(1);
});
