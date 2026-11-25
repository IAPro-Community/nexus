#!/usr/bin/env node
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  launchFixtureBrowser,
  meaningfulFixtureRequestFailures,
  startFixtureServer,
  terminateProcess,
} from './e2e-fixture.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..', '..');
const outDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR ||
  path.resolve(repoRoot, 'DEV', 'validation', 'nex-2', '8d1f1592', 'p1-lab');
if (!existsSync(outDir)) mkdirSync(outDir, { recursive: true });

const MOODS = ['idle', 'working', 'attention', 'critical', 'offline'];
const BEHAVIORS = [
  'ambient',
  'greet',
  'look',
  'thinking',
  'attention',
  'error',
  'resolved',
  'rest',
];

async function main() {
  console.log('=== Nex 2.0 P1 Desktop Lab E2E ===');
  const { server, bootstrapUrl, getOutput } = await startFixtureServer();

  let browser;
  let context;
  let page;
  let collectors;
  try {
    ({ browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl));
    const apiOrigin = new URL(bootstrapUrl).origin;

    await page.setViewportSize({ width: 420, height: 420 });
    const navStart = Date.now();
    await page.goto(`${apiOrigin}/nex-v2.html?lab=1`, {
      waitUntil: 'domcontentloaded',
      timeout: 15000,
    });

    await page.waitForFunction(() => window.__nexV2Metrics?.rigReady === true, null, {
      timeout: 10000,
    });
    const totalColdPaintMs = Date.now() - navStart;
    const initialMetrics = await page.evaluate(() => window.__nexV2Metrics);
    assert.ok(
      (initialMetrics.firstPaintMs ?? totalColdPaintMs) <= 1000,
      `First paint exceeded 1000ms budget: ${JSON.stringify(initialMetrics)}`,
    );

    const resizeBefore = initialMetrics.resizeCount;

    // Inspect all 5 operational moods and verify non-empty canvas without border clipping.
    const moodResults = {};
    for (const mood of MOODS) {
      await page.click(`[data-lab-mood="${mood}"]`);
      await page.waitForFunction(
        (expected) => document.querySelector('[data-mood]')?.getAttribute('data-mood') === expected,
        mood,
      );
      await page.waitForTimeout(180);
      const check = await page.evaluate(() => {
        const canvas = document.querySelector('canvas');
        const ctx = canvas?.getContext('2d');
        const rect = canvas?.getBoundingClientRect();
        if (!canvas || !ctx || !rect) return { ok: false, reason: 'missing canvas' };
        const w = canvas.width;
        const h = canvas.height;
        const data = ctx.getImageData(0, 0, w, h).data;
        let nonZero = 0;
        let edgeClip = 0;
        for (let y = 0; y < h; y++) {
          for (let x = 0; x < w; x++) {
            const alpha = data[(y * w + x) * 4 + 3];
            if (alpha > 16) {
              nonZero += 1;
              if (x === 0 || y === 0 || x === w - 1 || y === h - 1) edgeClip += 1;
            }
          }
        }
        return {
          ok: nonZero > 50 && edgeClip === 0,
          nonZero,
          edgeClip,
          width: Math.round(rect.width),
          height: Math.round(rect.height),
          metrics: window.__nexV2Metrics,
        };
      });
      assert.equal(
        check.ok,
        true,
        `Mood ${mood} failed paint/clipping check: ${JSON.stringify(check)}`,
      );
      moodResults[mood] = check;
      await page.screenshot({ path: path.join(outDir, `mood-${mood}.png`) });
    }

    // Exercise all 8 companion behaviors and collect interactive frame pacing.
    const behaviorResults = {};
    for (const behavior of BEHAVIORS) {
      await page.click(`[data-lab-behavior="${behavior}"]`);
      await page.waitForFunction(
        (expected) =>
          document.querySelector('[data-nex-behavior]')?.getAttribute('data-nex-behavior') ===
          expected,
        behavior,
      );
      await page.waitForTimeout(220);
      const metrics = await page.evaluate(() => window.__nexV2Metrics);
      behaviorResults[behavior] = {
        behavior: metrics.behavior,
        expression: metrics.expression,
        clock: metrics.clock,
      };
    }

    // Verify idle 20 FPS pacing vs paused reduced-motion (0 incremental frames).
    await page.click('[data-lab-mood="idle"]');
    await page.waitForTimeout(350);
    const idleMetrics = await page.evaluate(() => window.__nexV2Metrics);
    assert.equal(
      idleMetrics.resizeCount,
      resizeBefore,
      'Animations/behaviors must never trigger window resize',
    );

    await page.click('[data-lab-reduced-motion]');
    await page.waitForTimeout(100);
    const beforeFrozen = await page.evaluate(() => window.__nexV2Metrics.clock?.frameCount ?? 0);
    await page.waitForTimeout(450);
    const afterFrozen = await page.evaluate(() => window.__nexV2Metrics.clock?.frameCount ?? 0);
    assert.equal(
      afterFrozen,
      beforeFrozen,
      `Reduced-motion must emit 0 animation frames (before=${beforeFrozen}, after=${afterFrozen})`,
    );

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

    const report = {
      coldNavigationToPaintMs: totalColdPaintMs,
      rigFirstPaintMs: initialMetrics.firstPaintMs,
      resizeCountDelta: idleMetrics.resizeCount - resizeBefore,
      idleMetrics: idleMetrics.clock,
      reducedMotionFramesDelta: afterFrozen - beforeFrozen,
      moodResults,
      behaviorResults,
    };
    writeFileSync(path.join(outDir, 'p1-lab-metrics.json'), `${JSON.stringify(report, null, 2)}\n`);
    console.log(
      'Nex 2.0 P1 Desktop Lab E2E: PASS',
      JSON.stringify({
        coldNavigationToPaintMs: totalColdPaintMs,
        rigFirstPaintMs: initialMetrics.firstPaintMs,
        idleTargetFps: idleMetrics.clock?.targetFps,
        p95FrameMs: idleMetrics.clock?.p95FrameMs,
        reducedMotionFramesDelta: afterFrozen - beforeFrozen,
      }),
    );
  } catch (error) {
    console.error('Nex 2.0 P1 Desktop Lab E2E: FAIL:', error);
    console.error('Server output:\n' + getOutput());
    process.exitCode = 1;
  } finally {
    if (context) await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
    await terminateProcess(server);
  }
}

await main();
